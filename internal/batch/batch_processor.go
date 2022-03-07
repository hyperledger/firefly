// Copyright © 2022 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package batch

import (
	"context"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type batchWork struct {
	msg  *fftypes.Message
	data []*fftypes.Data
}

type batchProcessorConf struct {
	DispatcherOptions
	name           string
	dispatcherName string
	txType         fftypes.TransactionType
	namespace      string
	identity       fftypes.SignerRef
	group          *fftypes.Bytes32
	dispatch       DispatchHandler
}

// FlushStatus is an object that can be returned on REST queries to understand the status
// of the batch processor
type FlushStatus struct {
	LastFlushTime        *fftypes.FFTime `json:"lastFlushStartTime"`
	Flushing             *fftypes.UUID   `json:"flushing,omitempty"`
	Blocked              bool            `json:"blocked"`
	LastFlushError       string          `json:"lastFlushError,omitempty"`
	LastFlushErrorTime   *fftypes.FFTime `json:"lastFlushErrorTime,omitempty"`
	AverageBatchBytes    int64           `json:"averageBatchBytes"`
	AverageBatchMessages float64         `json:"averageBatchMessages"`
	AverageBatchData     float64         `json:"averageBatchData"`
	AverageFlushTimeMS   int64           `json:"averageFlushTimeMS"`
	TotalBatches         int64           `json:"totalBatches"`
	TotalErrors          int64           `json:"totalErrors"`

	totalBytesFlushed    int64
	totalMessagesFlushed int64
	totalDataFlushed     int64
	totalFlushDuration   time.Duration
}

type batchProcessor struct {
	ctx                context.Context
	ni                 sysmessaging.LocalNodeInfo
	database           database.Plugin
	txHelper           txcommon.Helper
	cancelCtx          func()
	done               chan struct{}
	quescing           chan bool
	newWork            chan *batchWork
	assemblyID         *fftypes.UUID
	assemblyQueue      []*batchWork
	assemblyQueueBytes int64
	flushedSequences   []int64
	statusMux          sync.Mutex
	flushStatus        FlushStatus
	retry              *retry.Retry
	conf               *batchProcessorConf
}

type DispatchState struct {
	Manifest  *fftypes.BatchManifest
	Persisted fftypes.BatchPersisted
	Payload   fftypes.BatchPayload
	Pins      []*fftypes.Bytes32
}

const batchSizeEstimateBase = int64(512)

func newBatchProcessor(ctx context.Context, ni sysmessaging.LocalNodeInfo, di database.Plugin, conf *batchProcessorConf, baseRetryConf *retry.Retry) *batchProcessor {
	pCtx := log.WithLogField(log.WithLogField(ctx, "d", conf.dispatcherName), "p", conf.name)
	pCtx, cancelCtx := context.WithCancel(pCtx)
	bp := &batchProcessor{
		ctx:       pCtx,
		cancelCtx: cancelCtx,
		ni:        ni,
		database:  di,
		txHelper:  txcommon.NewTransactionHelper(di),
		newWork:   make(chan *batchWork, conf.BatchMaxSize),
		quescing:  make(chan bool, 1),
		done:      make(chan struct{}),
		retry: &retry.Retry{
			InitialDelay: baseRetryConf.InitialDelay,
			MaximumDelay: baseRetryConf.MaximumDelay,
			Factor:       baseRetryConf.Factor,
		},
		conf:             conf,
		flushedSequences: []int64{},
		flushStatus: FlushStatus{
			LastFlushTime: fftypes.Now(),
		},
	}
	// Capture flush errors for our status
	bp.retry.ErrCallback = bp.captureFlushError
	bp.newAssembly()
	go bp.assemblyLoop()
	log.L(pCtx).Infof("Batch processor created")
	return bp
}

func (bw *batchWork) estimateSize() int64 {
	sizeEstimate := bw.msg.EstimateSize(false /* we calculate data size separately, as we have the full data objects */)
	for _, d := range bw.data {
		sizeEstimate += d.EstimateSize()
	}
	return sizeEstimate
}

func (bp *batchProcessor) status() *ProcessorStatus {
	bp.statusMux.Lock()
	defer bp.statusMux.Unlock()
	return &ProcessorStatus{
		Dispatcher: bp.conf.dispatcherName,
		Name:       bp.conf.name,
		Status:     bp.flushStatus, // copy
	}
}

func (bp *batchProcessor) newAssembly(initalWork ...*batchWork) {
	bp.assemblyID = fftypes.NewUUID()
	bp.assemblyQueue = append([]*batchWork{}, initalWork...)
	bp.assemblyQueueBytes = batchSizeEstimateBase
}

// addWork adds the work to the assemblyQueue, and calculates if we have overflowed with this work.
// We check for duplicates, and add the work in sequence order.
// This helps in the case for parallel REST APIs all committing to the DB at a similar time.
// With a sufficient batch size and batch timeout, the batch will still dispatch the messages
// in DB sequence order (although this is not guaranteed).
func (bp *batchProcessor) addWork(newWork *batchWork) (full, overflow bool) {
	newQueue := make([]*batchWork, 0, len(bp.assemblyQueue)+1)
	added := false
	// Check it's not in the recently flushed list
	for _, flushedSequence := range bp.flushedSequences {
		if newWork.msg.Sequence == flushedSequence {
			log.L(bp.ctx).Debugf("Ignoring add of recently flushed message %s sequence=%d to in-flight batch assembly %s", newWork.msg.Header.ID, newWork.msg.Sequence, bp.assemblyID)
			return false, false
		}
	}
	// Build the new sorted work list, checking there for duplicates too
	for _, work := range bp.assemblyQueue {
		if newWork.msg.Sequence == work.msg.Sequence {
			log.L(bp.ctx).Debugf("Ignoring duplicate add of message %s sequence=%d to in-flight batch assembly %s", newWork.msg.Header.ID, newWork.msg.Sequence, bp.assemblyID)
			return false, false
		}
		if !added && newWork.msg.Sequence < work.msg.Sequence {
			newQueue = append(newQueue, newWork)
			added = true
		}
		newQueue = append(newQueue, work)
	}
	if !added {
		newQueue = append(newQueue, newWork)
	}
	log.L(bp.ctx).Debugf("Added message %s sequence=%d to in-flight batch assembly %s", newWork.msg.Header.ID, newWork.msg.Sequence, bp.assemblyID)
	bp.assemblyQueueBytes += newWork.estimateSize()
	bp.assemblyQueue = newQueue
	full = len(bp.assemblyQueue) >= int(bp.conf.BatchMaxSize) || (bp.assemblyQueueBytes >= bp.conf.BatchMaxBytes)
	overflow = len(bp.assemblyQueue) > 1 && (bp.assemblyQueueBytes > bp.conf.BatchMaxBytes)
	return full, overflow
}

func (bp *batchProcessor) addFlushedSequences(flushAssembly []*batchWork) {
	// We need to keep track of the sequences we're flushing, because until we finish our flush
	// the batch processor might be re-queuing the same messages to use due to rewinds.

	// We keep twice the batch size, which might be made up of multiple batches
	maxFlushedSeqLen := int(2 * bp.conf.BatchMaxSize)

	// We keep as much of the END of the existing set as we can, and shift it forwards.
	// Then we add the whole of the current flush set after that
	combinedLen := len(flushAssembly) + len(bp.flushedSequences)
	newLength := combinedLen
	if combinedLen > maxFlushedSeqLen {
		newLength = maxFlushedSeqLen
	}
	dropLength := combinedLen - newLength
	retainLength := len(bp.flushedSequences) - dropLength
	newFlushedSequences := make([]int64, newLength)
	for i := 0; i < retainLength; i++ {
		newFlushedSequences[i] = bp.flushedSequences[dropLength+i]
	}
	for i := 0; i < len(flushAssembly); i++ {
		newFlushedSequences[retainLength+i] = flushAssembly[i].msg.Sequence
	}
	bp.flushedSequences = newFlushedSequences
}

func (bp *batchProcessor) startFlush(overflow bool) (id *fftypes.UUID, flushAssembly []*batchWork, byteSize int64) {
	bp.statusMux.Lock()
	defer bp.statusMux.Unlock()
	// Start the clock
	bp.flushStatus.Blocked = false
	bp.flushStatus.LastFlushTime = fftypes.Now()
	// Split the current work if required for overflow
	overflowWork := make([]*batchWork, 0)
	if overflow {
		lastElem := len(bp.assemblyQueue) - 1
		flushAssembly = append(flushAssembly, bp.assemblyQueue[:lastElem]...)
		overflowWork = append(overflowWork, bp.assemblyQueue[lastElem])
	} else {
		flushAssembly = bp.assemblyQueue
	}
	bp.addFlushedSequences(flushAssembly)
	// Cycle to the next assembly
	id = bp.assemblyID
	byteSize = bp.assemblyQueueBytes
	bp.flushStatus.Flushing = id
	bp.newAssembly(overflowWork...)
	return id, flushAssembly, byteSize
}

func (bp *batchProcessor) endFlush(state *DispatchState, byteSize int64) {
	bp.statusMux.Lock()
	defer bp.statusMux.Unlock()
	fs := &bp.flushStatus

	duration := time.Since(*fs.LastFlushTime.Time())
	fs.Flushing = nil

	fs.TotalBatches++

	fs.totalFlushDuration += duration
	fs.AverageFlushTimeMS = (fs.totalFlushDuration / time.Duration(fs.TotalBatches)).Milliseconds()

	fs.totalBytesFlushed += byteSize
	fs.AverageBatchBytes = (fs.totalBytesFlushed / fs.TotalBatches)

	fs.totalMessagesFlushed += int64(len(state.Payload.Messages))
	fs.AverageBatchMessages = math.Round((float64(fs.totalMessagesFlushed)/float64(fs.TotalBatches))*100) / 100

	fs.totalDataFlushed += int64(len(state.Payload.Data))
	fs.AverageBatchData = math.Round((float64(fs.totalDataFlushed)/float64(fs.TotalBatches))*100) / 100
}

func (bp *batchProcessor) captureFlushError(err error) {
	bp.statusMux.Lock()
	defer bp.statusMux.Unlock()
	fs := &bp.flushStatus

	fs.TotalErrors++
	fs.Blocked = true
	fs.LastFlushErrorTime = fftypes.Now()
	fs.LastFlushError = err.Error()
}

func (bp *batchProcessor) startQuiesce() {
	// We are ready to quiesce, but we can't safely close our input channel.
	// We just do a non-blocking pass (queue length is 1) to the manager to
	// ask them to close our channel before their next read.
	// One more item of work might get through the pipe in an edge case here.
	select {
	case bp.quescing <- true:
	default:
	}
}

// The assemblyLoop receives new work, sorts it, and waits for the size/timer to pop before
// flushing the batch. The newWork channel has up to one batch of slots queue length,
// so that we can have one batch of work queuing for assembly, while we have one batch flushing.
func (bp *batchProcessor) assemblyLoop() {
	defer close(bp.done)
	l := log.L(bp.ctx)

	var batchTimeout = time.NewTimer(bp.conf.DisposeTimeout)
	idle := true
	quescing := false
	for !quescing {

		var timedout, full, overflow bool
		select {
		case <-bp.ctx.Done():
			l.Tracef("Batch processor shutting down")
			_ = batchTimeout.Stop()
			return
		case <-batchTimeout.C:
			l.Debugf("Batch timer popped")
			if len(bp.assemblyQueue) == 0 {
				bp.startQuiesce()
			} else {
				// We need to flush
				timedout = true
			}
		case work, ok := <-bp.newWork:
			if !ok {
				quescing = true
			} else {
				full, overflow = bp.addWork(work)
				if idle {
					// We've hit a message while we were idle - we now need to wait for the batch to time out.
					_ = batchTimeout.Stop()
					batchTimeout = time.NewTimer(bp.conf.BatchTimeout)
					idle = false
				}
			}
		}
		if (full || timedout || quescing) && len(bp.assemblyQueue) > 0 {
			// Let Go GC the old timer
			_ = batchTimeout.Stop()

			// If we are in overflow, start the clock for the next batch to start before we do the flush
			// (even though we won't check it until after).
			if overflow {
				batchTimeout = time.NewTimer(bp.conf.BatchTimeout)
			}

			err := bp.flush(overflow)
			if err != nil {
				l.Warnf("Batch processor shutting down: %s", err)
				_ = batchTimeout.Stop()
				return
			}

			// If we didn't overflow, then just go back to idle - we don't know if we have more work to come, so
			// either we'll pop straight away (and move to the batch timeout) or wait for the dispose timeout
			if !overflow && !quescing {
				batchTimeout = time.NewTimer(bp.conf.DisposeTimeout)
				idle = true
			}
		}
	}
}

func (bp *batchProcessor) flush(overflow bool) error {
	id, flushWork, byteSize := bp.startFlush(overflow)
	state := bp.initFlushState(id, flushWork)

	// Sealing phase: assigns persisted pins to messages, and finalizes the manifest
	err := bp.sealBatch(state)
	if err != nil {
		return err
	}

	// Dispatch phase: the heavy lifting work - calling plugins to do the hard work of the batch.
	//   Must manage its own database updates if it performs them, and any that result in updates
	//   to the payload must be reflected back on the payload objects.
	//   For example updates to the Blob.Public of a Data entry must be written do the DB,
	//   and updated in Payload.Data[] array.
	err = bp.dispatchBatch(state)
	if err != nil {
		return err
	}

	// Dispatched phase: Writes back the changes to the DB, so that these messages will not be
	//   are all tagged as part of this batch, and won't be included in any future batches.
	err = bp.markPayloadDispatched(state)
	if err != nil {
		return err
	}

	bp.endFlush(state, byteSize)
	return nil
}

func (bp *batchProcessor) initFlushState(id *fftypes.UUID, flushWork []*batchWork) *DispatchState {
	log.L(bp.ctx).Debugf("Flushing batch %s", id)
	state := &DispatchState{
		Persisted: fftypes.BatchPersisted{
			BatchHeader: fftypes.BatchHeader{
				ID:        id,
				Type:      bp.conf.DispatcherOptions.BatchType,
				Namespace: bp.conf.namespace,
				SignerRef: bp.conf.identity,
				Group:     bp.conf.group,
				Node:      bp.ni.GetNodeUUID(bp.ctx),
			},
			Created: fftypes.Now(),
		},
	}
	for _, w := range flushWork {
		if w.msg != nil {
			state.Payload.Messages = append(state.Payload.Messages, w.msg.BatchMessage())
		}
		for _, d := range w.data {
			state.Payload.Data = append(state.Payload.Data, d.BatchData(state.Persisted.Type))
		}
	}
	state.Manifest = state.Payload.Manifest(id)
	return state
}

func (bp *batchProcessor) maskContext(ctx context.Context, msg *fftypes.Message, topic string) (contextOrPin *fftypes.Bytes32, err error) {

	hashBuilder := sha256.New()
	hashBuilder.Write([]byte(topic))

	// For broadcast we do not need to mask the context, which is just the hash
	// of the topic. There would be no way to unmask it if we did, because we don't have
	// the full list of senders to know what their next hashes should be.
	if msg.Header.Group == nil {
		return fftypes.HashResult(hashBuilder), nil
	}

	// For private groups, we need to make the topic specific to the group (which is
	// a salt for the hash as it is not on chain)
	hashBuilder.Write((*msg.Header.Group)[:])

	// The combination of the topic and group is the context
	contextHash := fftypes.HashResult(hashBuilder)

	// Get the next nonce for this context - we're the authority in the nextwork on this,
	// as we are the sender.
	gc := &fftypes.Nonce{
		Context: contextHash,
		Group:   msg.Header.Group,
		Topic:   topic,
	}
	err = bp.database.UpsertNonceNext(ctx, gc)
	if err != nil {
		return nil, err
	}

	// Now combine our sending identity, and this nonce, to produce the hash that should
	// be expected by all members of the group as the next nonce from us on this topic.
	// Note we use our identity DID (not signing key) for this.
	hashBuilder.Write([]byte(msg.Header.Author))
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, uint64(gc.Nonce))
	hashBuilder.Write(nonceBytes)

	return fftypes.HashResult(hashBuilder), err
}

func (bp *batchProcessor) maskContexts(ctx context.Context, payload *fftypes.BatchPayload) ([]*fftypes.Bytes32, error) {
	// Calculate the sequence hashes
	pinsAssigned := false
	contextsOrPins := make([]*fftypes.Bytes32, 0, len(payload.Messages))
	for _, msg := range payload.Messages {
		if len(msg.Pins) > 0 {
			// We have already allocated pins to this message, we cannot re-allocate.
			log.L(ctx).Debugf("Message %s already has %d pins allocated", msg.Header.ID, len(msg.Pins))
			continue
		}
		for _, topic := range msg.Header.Topics {
			contextOrPin, err := bp.maskContext(ctx, msg, topic)
			if err != nil {
				return nil, err
			}
			contextsOrPins = append(contextsOrPins, contextOrPin)
			if msg.Header.Group != nil {
				msg.Pins = append(msg.Pins, contextOrPin.String())
				pinsAssigned = true
			}
		}
		if pinsAssigned {
			// It's important we update the message pins at this phase, as we have "spent" a nonce
			// on this topic from the database. So this message has grabbed a slot in our queue.
			// If we fail the dispatch, and redo the batch sealing process, we must not allocate
			// a second nonce to it (and as such modifiy the batch payload).
			err := bp.database.UpdateMessage(ctx, msg.Header.ID,
				database.MessageQueryFactory.NewUpdate(ctx).Set("pins", msg.Pins),
			)
			if err != nil {
				return nil, err
			}
		}
	}
	return contextsOrPins, nil
}

func (bp *batchProcessor) sealBatch(state *DispatchState) (err error) {
	err = bp.retry.Do(bp.ctx, "batch persist", func(attempt int) (retry bool, err error) {
		return true, bp.database.RunAsGroup(bp.ctx, func(ctx context.Context) (err error) {

			if bp.conf.txType == fftypes.TransactionTypeBatchPin {
				// Generate a new Transaction, which will be used to record status of the associated transaction as it happens
				if state.Pins, err = bp.maskContexts(ctx, &state.Payload); err != nil {
					return err
				}
			}

			state.Persisted.TX.Type = bp.conf.txType
			if state.Persisted.TX.ID, err = bp.txHelper.SubmitNewTransaction(ctx, state.Persisted.Namespace, bp.conf.txType); err != nil {
				return err
			}

			// The hash of the batch, is the hash of the manifest to minimize the compute cost.
			// Note in v0.13 and before, it was the hash of the payload - so the inbound route has a fallback to accepting the full payload hash
			state.Persisted.Manifest = state.Manifest.String()
			state.Persisted.Hash = fftypes.HashString(state.Persisted.Manifest)

			log.L(ctx).Debugf("Batch %s sealed. Hash=%s", state.Persisted.ID, state.Persisted.Hash)

			// At this point the manifest of the batch is finalized. We write it to the database
			return bp.database.UpsertBatch(ctx, &state.Persisted)
		})
	})
	return err
}

func (bp *batchProcessor) dispatchBatch(state *DispatchState) error {
	// Call the dispatcher to do the heavy lifting - will only exit if we're closed
	return operations.RunWithOperationCache(bp.ctx, func(ctx context.Context) error {
		return bp.retry.Do(ctx, "batch dispatch", func(attempt int) (retry bool, err error) {
			return true, bp.conf.dispatch(ctx, state)
		})
	})
}

func (bp *batchProcessor) markPayloadDispatched(state *DispatchState) error {
	return bp.retry.Do(bp.ctx, "mark dispatched messages", func(attempt int) (retry bool, err error) {
		return true, bp.database.RunAsGroup(bp.ctx, func(ctx context.Context) (err error) {
			// Update all the messages in the batch with the batch ID
			msgIDs := make([]driver.Value, len(state.Payload.Messages))
			for i, msg := range state.Payload.Messages {
				msgIDs[i] = msg.Header.ID
			}
			fb := database.MessageQueryFactory.NewFilter(ctx)
			filter := fb.And(
				fb.In("id", msgIDs),
				fb.Eq("state", fftypes.MessageStateReady), // In the outside chance the next state transition happens first (which supersedes this)
			)

			var allMsgsUpdate database.Update
			if bp.conf.txType == fftypes.TransactionTypeBatchPin {
				// Sent state waiting for confirm
				allMsgsUpdate = database.MessageQueryFactory.NewUpdate(ctx).
					Set("batch", state.Persisted.ID).      // Mark the batch they are in
					Set("state", fftypes.MessageStateSent) // Set them sent, so they won't be picked up and re-sent after restart/rewind
			} else {
				// Immediate confirmation if no batch pinning
				allMsgsUpdate = database.MessageQueryFactory.NewUpdate(ctx).
					Set("batch", state.Persisted.ID).
					Set("state", fftypes.MessageStateConfirmed).
					Set("confirmed", fftypes.Now())
			}

			if err = bp.database.UpdateMessages(ctx, filter, allMsgsUpdate); err != nil {
				return err
			}

			if bp.conf.txType == fftypes.TransactionTypeUnpinned {
				for _, msg := range state.Payload.Messages {
					// Emit a confirmation event locally immediately
					event := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, state.Persisted.Namespace, msg.Header.ID, state.Persisted.TX.ID)
					event.Correlator = msg.Header.CID
					if err := bp.database.InsertEvent(ctx, event); err != nil {
						return err
					}
				}
			}

			return nil
		})
	})
}
