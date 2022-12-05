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
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hyperledger/firefly-common/pkg/ffapi"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/retry"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

type batchWork struct {
	msg  *core.Message
	data core.DataArray
}

type batchProcessorConf struct {
	DispatcherOptions
	name           string
	dispatcherName string
	txType         core.TransactionType
	signer         core.SignerRef
	group          *fftypes.Bytes32
	dispatch       DispatchHandler
}

// FlushStatus is an object that can be returned on REST queries to understand the status
// of the batch processor
type FlushStatus struct {
	LastFlushTime        *fftypes.FFTime `ffstruct:"BatchFlushStatus" json:"lastFlushStartTime"`
	Flushing             *fftypes.UUID   `ffstruct:"BatchFlushStatus" json:"flushing,omitempty"`
	Blocked              bool            `ffstruct:"BatchFlushStatus" json:"blocked"`
	LastFlushError       string          `ffstruct:"BatchFlushStatus" json:"lastFlushError,omitempty"`
	LastFlushErrorTime   *fftypes.FFTime `ffstruct:"BatchFlushStatus" json:"lastFlushErrorTime,omitempty"`
	AverageBatchBytes    int64           `ffstruct:"BatchFlushStatus" json:"averageBatchBytes"`
	AverageBatchMessages float64         `ffstruct:"BatchFlushStatus" json:"averageBatchMessages"`
	AverageBatchData     float64         `ffstruct:"BatchFlushStatus" json:"averageBatchData"`
	AverageFlushTimeMS   int64           `ffstruct:"BatchFlushStatus" json:"averageFlushTimeMS"`
	TotalBatches         int64           `ffstruct:"BatchFlushStatus" json:"totalBatches"`
	TotalErrors          int64           `ffstruct:"BatchFlushStatus" json:"totalErrors"`

	totalBytesFlushed    int64
	totalMessagesFlushed int64
	totalDataFlushed     int64
	totalFlushDuration   time.Duration
}

type batchProcessor struct {
	ctx                context.Context
	bm                 *batchManager
	data               data.Manager
	database           database.Plugin
	txHelper           txcommon.Helper
	cancelCtx          func()
	done               chan struct{}
	quiescing          chan bool
	newWork            chan *batchWork
	assemblyID         *fftypes.UUID
	assemblyQueue      []*batchWork
	assemblyQueueBytes int64
	statusMux          sync.Mutex
	flushStatus        FlushStatus
	retry              *retry.Retry
	conf               *batchProcessorConf
}

type nonceState struct {
	latest int64
	new    bool
}

type DispatchState struct {
	Persisted      core.BatchPersisted
	Messages       []*core.Message
	Data           core.DataArray
	Pins           []*fftypes.Bytes32
	noncesAssigned map[fftypes.Bytes32]*nonceState
	msgPins        map[fftypes.UUID]fftypes.FFStringArray
}

const batchSizeEstimateBase = int64(512)

func newBatchProcessor(bm *batchManager, conf *batchProcessorConf, baseRetryConf *retry.Retry, txHelper txcommon.Helper) *batchProcessor {
	pCtx := log.WithLogField(log.WithLogField(bm.ctx, "d", conf.dispatcherName), "p", conf.name)
	pCtx, cancelCtx := context.WithCancel(pCtx)
	bp := &batchProcessor{
		ctx:       pCtx,
		cancelCtx: cancelCtx,
		bm:        bm,
		database:  bm.database,
		data:      bm.data,
		txHelper:  txHelper,
		newWork:   make(chan *batchWork, conf.BatchMaxSize),
		quiescing: make(chan bool, 1),
		done:      make(chan struct{}),
		retry: &retry.Retry{
			InitialDelay: baseRetryConf.InitialDelay,
			MaximumDelay: baseRetryConf.MaximumDelay,
			Factor:       baseRetryConf.Factor,
		},
		conf: conf,
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
	// Build the new sorted work list
	for _, work := range bp.assemblyQueue {
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
	// Cycle to the next assembly
	id = bp.assemblyID
	byteSize = bp.assemblyQueueBytes
	bp.flushStatus.Flushing = id
	bp.newAssembly(overflowWork...)
	return id, flushAssembly, byteSize
}

func (bp *batchProcessor) notifyFlushComplete(flushWork []*batchWork) {
	sequences := make([]int64, len(flushWork))
	for i, work := range flushWork {
		sequences[i] = work.msg.Sequence
	}
	bp.bm.notifyFlushed(sequences)
}

func (bp *batchProcessor) updateFlushStats(state *DispatchState, byteSize int64) {
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

	fs.totalMessagesFlushed += int64(len(state.Messages))
	fs.AverageBatchMessages = math.Round((float64(fs.totalMessagesFlushed)/float64(fs.TotalBatches))*100) / 100

	fs.totalDataFlushed += int64(len(state.Data))
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
	case bp.quiescing <- true:
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

	log.L(bp.ctx).Debugf("Flushing batch %s", id)
	state := bp.initFlushState(id, flushWork)

	// Sealing phase: assigns persisted pins to messages, and finalizes the manifest
	err := bp.sealBatch(state)
	if err != nil {
		return err
	}
	log.L(bp.ctx).Debugf("Sealed batch %s", id)

	// Dispatch phase: the heavy lifting work - calling plugins to do the hard work of the batch.
	//   The dispatcher can update the state, such as appending to the BlobsPublished array,
	//   to affect DB updates as part of the finalization phase.
	err = bp.dispatchBatch(state)
	if err != nil {
		return err
	}
	log.L(bp.ctx).Debugf("Dispatched batch %s", id)

	// Finalization phase: Writes back the changes to the DB, so that these messages will not be
	//   are all tagged as part of this batch, and won't be included in any future batches.
	err = bp.markPayloadDispatched(state)
	if err != nil {
		return err
	}
	log.L(bp.ctx).Debugf("Finalized batch %s", id)

	// Notify the manager that we've flushed these sequences
	bp.notifyFlushComplete(flushWork)

	// Update our stats
	bp.updateFlushStats(state, byteSize)
	return nil
}

func (bp *batchProcessor) initFlushState(id *fftypes.UUID, flushWork []*batchWork) *DispatchState {
	state := &DispatchState{
		Persisted: core.BatchPersisted{
			BatchHeader: core.BatchHeader{
				ID:        id,
				Type:      bp.conf.DispatcherOptions.BatchType,
				Namespace: bp.bm.namespace,
				SignerRef: bp.conf.signer,
				Group:     bp.conf.group,
				Created:   fftypes.Now(),
			},
		},
	}
	localNode, err := bp.bm.identity.GetLocalNode(bp.ctx)
	if err == nil && localNode != nil {
		state.Persisted.BatchHeader.Node = localNode.ID
	}
	for _, w := range flushWork {
		if w.msg != nil {
			w.msg.BatchID = id
			state.Messages = append(state.Messages, w.msg.BatchMessage())
		}
		for _, d := range w.data {
			log.L(bp.ctx).Debugf("Adding data '%s' to batch '%s' for message '%s'", d.ID, id, w.msg.Header.ID)
			state.Data = append(state.Data, d.BatchData(state.Persisted.Type))
		}
	}
	return state
}

func (bp *batchProcessor) getNextNonce(ctx context.Context, state *DispatchState, nonceKeyHash *fftypes.Bytes32, contextHash *fftypes.Bytes32) (int64, error) {

	// See if the nonceKeyHash is in our cached state already
	if cached, ok := state.noncesAssigned[*nonceKeyHash]; ok {
		cached.latest++
		return cached.latest, nil
	}

	// Query the database for an existing record
	dbNonce, err := bp.database.GetNonce(ctx, nonceKeyHash)
	if err != nil {
		return -1, err
	}
	if dbNonce == nil {
		// For migration we need to query the base contextHash the first time we pass through this for a v0.14.1 or earlier migration
		if dbNonce, err = bp.database.GetNonce(ctx, contextHash); err != nil {
			return -1, err
		}
	}

	// Determine if we're the first - so get nonce zero - or if we need to add one to the DB nonce
	nonceState := &nonceState{}
	if dbNonce == nil {
		nonceState.new = true
	} else {
		nonceState.latest = dbNonce.Nonce + 1
	}

	// Cache it either way for additional messages in this batch to the same nonceKeyHash
	state.noncesAssigned[*nonceKeyHash] = nonceState
	return nonceState.latest, nil
}

func (bp *batchProcessor) maskContext(ctx context.Context, state *DispatchState, msg *core.Message, topic string) (msgPinString string, contextOrPin *fftypes.Bytes32, err error) {

	hashBuilder := sha256.New()
	hashBuilder.Write([]byte(topic))

	// For broadcast we do not need to mask the context, which is just the hash
	// of the topic. There would be no way to unmask it if we did, because we don't have
	// the full list of senders to know what their next hashes should be.
	if msg.Header.Group == nil {
		return "", fftypes.HashResult(hashBuilder), nil
	}

	// For private groups, we need to make the topic specific to the group (which is
	// a salt for the hash as it is not on chain)
	hashBuilder.Write((*msg.Header.Group)[:])

	// The combination of the topic and group is the context
	contextHash := fftypes.HashResult(hashBuilder)

	// Now combine our sending identity, and this nonce, to produce the hash that should
	// be expected by all members of the group as the next nonce from us on this topic.
	// Note we use our identity DID (not signing key) for this.
	hashBuilder.Write([]byte(msg.Header.Author))

	// Our DB of nonces we own, is keyed off of the hash at this point.
	// However, before v0.14.2 we didn't include the Author - so we need to pass the contextHash as a fallback.
	nonceKeyHash := fftypes.HashResult(hashBuilder)
	nonce, err := bp.getNextNonce(ctx, state, nonceKeyHash, contextHash)
	if err != nil {
		return "", nil, err
	}

	// Now we have the nonce, add that at the end of the hash to make it unqiue to this message
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, uint64(nonce))
	hashBuilder.Write(nonceBytes)

	pin := fftypes.HashResult(hashBuilder)
	pinStr := fmt.Sprintf("%s:%.16d", pin, nonce)
	log.L(ctx).Debugf("Assigned pin '%s' to message %s for topic '%s'", pinStr, msg.Header.ID, topic)
	return pinStr, pin, err
}

func (bp *batchProcessor) maskContexts(ctx context.Context, state *DispatchState) (contextsOrPins []*fftypes.Bytes32, err error) {
	// Calculate the sequence hashes
	for _, msg := range state.Messages {
		isPrivate := msg.Header.Group != nil
		if isPrivate && len(msg.Pins) > 0 {
			// We have already allocated pins to this message, we cannot re-allocate.
			log.L(ctx).Debugf("Message %s already has %d pins allocated", msg.Header.ID, len(msg.Pins))
			continue
		}
		var pins fftypes.FFStringArray
		if isPrivate {
			pins = make(fftypes.FFStringArray, len(msg.Header.Topics))
			state.msgPins[*msg.Header.ID] = pins
		}
		for i, topic := range msg.Header.Topics {
			pinString, contextOrPin, err := bp.maskContext(ctx, state, msg, topic)
			if err != nil {
				return nil, err
			}
			contextsOrPins = append(contextsOrPins, contextOrPin)
			if isPrivate {
				pins[i] = pinString
			}
		}
	}
	return contextsOrPins, nil
}

func (bp *batchProcessor) flushNonceState(ctx context.Context, state *DispatchState) error {

	// Flush all the assigned nonces to the DB
	for hash, nonceState := range state.noncesAssigned {
		dbNonce := &core.Nonce{
			Hash:  &hash,
			Nonce: nonceState.latest,
		}
		if nonceState.new {
			if err := bp.database.InsertNonce(ctx, dbNonce); err != nil {
				return err
			}
		} else {
			if err := bp.database.UpdateNonce(ctx, dbNonce); err != nil {
				return err
			}
		}
	}

	// It's important we update the message pins at this phase, as we have "spent" a nonce
	// on this topic from the database. So this message has grabbed a slot in our queue.
	// If we fail the dispatch, and redo the batch sealing process, we must not allocate
	// a second nonce to it (and as such modifiy the batch payload).
	//
	// Note to consider database transactions, we do not update the msg.Pins array, or the
	// message cache, until after the Retry/RunAsGroup has ended in sealBatch.
	for _, msg := range state.Messages {
		if pins, ok := state.msgPins[*msg.Header.ID]; ok {
			if err := bp.database.UpdateMessage(ctx, bp.bm.namespace, msg.Header.ID,
				database.MessageQueryFactory.NewUpdate(ctx).Set("pins", pins),
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func (bp *batchProcessor) sealBatch(state *DispatchState) (err error) {
	err = bp.retry.Do(bp.ctx, "batch persist", func(attempt int) (retry bool, err error) {
		return true, bp.database.RunAsGroup(bp.ctx, func(ctx context.Context) (err error) {

			// Clear state from any previous retry. We need to do fresh queries against the DB for nonces.
			state.noncesAssigned = make(map[fftypes.Bytes32]*nonceState)
			state.msgPins = make(map[fftypes.UUID]fftypes.FFStringArray)

			if bp.conf.txType == core.TransactionTypeBatchPin {
				// Generate a new Transaction, which will be used to record status of the associated transaction as it happens
				if state.Pins, err = bp.maskContexts(ctx, state); err != nil {
					return err
				}
				// Flush the state for all allocated nonces to the database
				if err = bp.flushNonceState(ctx, state); err != nil {
					return err
				}
			}

			state.Persisted.TX.Type = bp.conf.txType
			if state.Persisted.TX.ID, err = bp.txHelper.SubmitNewTransaction(ctx, bp.conf.txType, "" /* no idempotency key for batch TX */); err != nil {
				return err
			}
			manifest := state.Persisted.GenManifest(state.Messages, state.Data)

			// The hash of the batch, is the hash of the manifest to minimize the compute cost.
			// Note in v0.13 and before, it was the hash of the payload - so the inbound route has a fallback to accepting the full payload hash
			manifestString := manifest.String()
			state.Persisted.Manifest = fftypes.JSONAnyPtr(manifestString)
			state.Persisted.Hash = fftypes.HashString(manifestString)

			log.L(ctx).Debugf("Batch %s sealed. Hash=%s", state.Persisted.ID, state.Persisted.Hash)

			// At this point the manifest of the batch is finalized. We write it to the database
			return bp.database.UpsertBatch(ctx, &state.Persisted)
		})
	})
	if err != nil {
		return err
	}

	// Once the DB transaction is done, we need to update the messages with the pins.
	// We do this at this point, so the logic is re-entrant in a way that avoids re-allocating Pins to messages, but
	// means the retry loops above using the batch state function correctly.
	for _, msg := range state.Messages {
		if pins, ok := state.msgPins[*msg.Header.ID]; ok {
			msg.Pins = pins
			bp.data.UpdateMessageIfCached(bp.ctx, msg)
		}
	}
	return nil
}

func (bp *batchProcessor) dispatchBatch(state *DispatchState) error {
	// Call the dispatcher to do the heavy lifting - will only exit if we're closed
	return operations.RunWithOperationContext(bp.ctx, func(ctx context.Context) error {
		return bp.retry.Do(ctx, "batch dispatch", func(attempt int) (retry bool, err error) {
			return true, bp.conf.dispatch(ctx, state)
		})
	})
}

func (bp *batchProcessor) markPayloadDispatched(state *DispatchState) error {
	return bp.retry.Do(bp.ctx, "mark dispatched messages", func(attempt int) (retry bool, err error) {
		return true, bp.database.RunAsGroup(bp.ctx, func(ctx context.Context) (err error) {
			// Update all the messages in the batch with the batch ID
			msgIDs := make([]driver.Value, len(state.Messages))
			confirmTime := fftypes.Now()
			for i, msg := range state.Messages {
				msgIDs[i] = msg.Header.ID
				msg.BatchID = state.Persisted.ID
				if bp.conf.txType == core.TransactionTypeBatchPin {
					msg.State = core.MessageStateSent
				} else {
					msg.State = core.MessageStateConfirmed
					msg.Confirmed = confirmTime
				}
				// We don't want to have to read the DB again if we want to query for the batch ID, or pins,
				// so ensure the copy in our cache gets updated.
				bp.data.UpdateMessageIfCached(ctx, msg)
			}
			fb := database.MessageQueryFactory.NewFilter(ctx)
			filter := fb.And(
				fb.In("id", msgIDs),
				fb.Eq("state", core.MessageStateReady), // In the outside chance the next state transition happens first (which supersedes this)
			)

			var allMsgsUpdate ffapi.Update
			if bp.conf.txType == core.TransactionTypeBatchPin {
				// Sent state waiting for confirm
				allMsgsUpdate = database.MessageQueryFactory.NewUpdate(ctx).
					Set("batch", state.Persisted.ID).   // Mark the batch they are in
					Set("state", core.MessageStateSent) // Set them sent, so they won't be picked up and re-sent after restart/rewind
			} else {
				// Immediate confirmation if no batch pinning
				allMsgsUpdate = database.MessageQueryFactory.NewUpdate(ctx).
					Set("batch", state.Persisted.ID).
					Set("state", core.MessageStateConfirmed).
					Set("confirmed", confirmTime)
			}

			if err = bp.database.UpdateMessages(ctx, bp.bm.namespace, filter, allMsgsUpdate); err != nil {
				return err
			}

			if bp.conf.txType == core.TransactionTypeUnpinned {
				for _, msg := range state.Messages {
					// Emit a confirmation event locally immediately
					for _, topic := range msg.Header.Topics {
						// One event per topic
						event := core.NewEvent(core.EventTypeMessageConfirmed, state.Persisted.Namespace, msg.Header.ID, state.Persisted.TX.ID, topic)
						event.Correlator = msg.Header.CID
						if err := bp.database.InsertEvent(ctx, event); err != nil {
							return err
						}
					}
				}
			}

			return nil
		})
	})
}
