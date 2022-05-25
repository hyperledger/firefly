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
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/retry"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

func NewBatchManager(ctx context.Context, ni sysmessaging.LocalNodeInfo, di database.Plugin, dm data.Manager, txHelper txcommon.Helper) (Manager, error) {
	if di == nil || dm == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "BatchManager")
	}
	pCtx, cancelCtx := context.WithCancel(log.WithLogField(ctx, "role", "batchmgr"))
	readPageSize := config.GetUint(coreconfig.BatchManagerReadPageSize)
	bm := &batchManager{
		ctx:                        pCtx,
		cancelCtx:                  cancelCtx,
		ni:                         ni,
		database:                   di,
		data:                       dm,
		txHelper:                   txHelper,
		readOffset:                 -1, // On restart we trawl for all ready messages
		readPageSize:               uint64(readPageSize),
		minimumPollDelay:           config.GetDuration(coreconfig.BatchManagerMinimumPollDelay),
		messagePollTimeout:         config.GetDuration(coreconfig.BatchManagerReadPollTimeout),
		startupOffsetRetryAttempts: config.GetInt(coreconfig.OrchestratorStartupAttempts),
		dispatcherMap:              make(map[string]*dispatcher),
		allDispatchers:             make([]*dispatcher, 0),
		newMessages:                make(chan int64, readPageSize),
		inflightSequences:          make(map[int64]*batchProcessor),
		shoulderTap:                make(chan bool, 1),
		rewindOffset:               -1,
		done:                       make(chan struct{}),
		retry: &retry.Retry{
			InitialDelay: config.GetDuration(coreconfig.BatchRetryInitDelay),
			MaximumDelay: config.GetDuration(coreconfig.BatchRetryMaxDelay),
			Factor:       config.GetFloat64(coreconfig.BatchRetryFactor),
		},
	}
	return bm, nil
}

type Manager interface {
	RegisterDispatcher(name string, txType core.TransactionType, msgTypes []core.MessageType, handler DispatchHandler, batchOptions DispatcherOptions)
	NewMessages() chan<- int64
	Start() error
	Close()
	WaitStop()
	Status() *ManagerStatus
}

type ManagerStatus struct {
	Processors []*ProcessorStatus `ffstruct:"BatchManagerStatus" json:"processors"`
}

type ProcessorStatus struct {
	Dispatcher string      `ffstruct:"BatchProcessorStatus" json:"dispatcher"`
	Name       string      `ffstruct:"BatchProcessorStatus" json:"name"`
	Status     FlushStatus `ffstruct:"BatchProcessorStatus" json:"status"`
}

type batchManager struct {
	ctx                        context.Context
	cancelCtx                  func()
	ni                         sysmessaging.LocalNodeInfo
	database                   database.Plugin
	data                       data.Manager
	txHelper                   txcommon.Helper
	dispatcherMux              sync.Mutex
	dispatcherMap              map[string]*dispatcher
	allDispatchers             []*dispatcher
	newMessages                chan int64
	done                       chan struct{}
	retry                      *retry.Retry
	readOffset                 int64
	rewindOffsetMux            sync.Mutex
	rewindOffset               int64
	inflightMux                sync.Mutex
	inflightSequences          map[int64]*batchProcessor
	inflightFlushed            []int64
	shoulderTap                chan bool
	readPageSize               uint64
	minimumPollDelay           time.Duration
	messagePollTimeout         time.Duration
	startupOffsetRetryAttempts int
}

type DispatchHandler func(context.Context, *DispatchState) error

type DispatcherOptions struct {
	BatchType      core.BatchType
	BatchMaxSize   uint
	BatchMaxBytes  int64
	BatchTimeout   time.Duration
	DisposeTimeout time.Duration
}

type dispatcher struct {
	name       string
	handler    DispatchHandler
	processors map[string]*batchProcessor
	options    DispatcherOptions
}

func (bm *batchManager) getProcessorKey(namespace string, identity *core.SignerRef, groupID *fftypes.Bytes32) string {
	return fmt.Sprintf("%s|%s|%v", namespace, identity.Author, groupID)
}

func (bm *batchManager) getDispatcherKey(txType core.TransactionType, msgType core.MessageType) string {
	return fmt.Sprintf("tx:%s/%s", txType, msgType)
}

func (bm *batchManager) RegisterDispatcher(name string, txType core.TransactionType, msgTypes []core.MessageType, handler DispatchHandler, options DispatcherOptions) {
	bm.dispatcherMux.Lock()
	defer bm.dispatcherMux.Unlock()

	dispatcher := &dispatcher{
		name:       name,
		handler:    handler,
		options:    options,
		processors: make(map[string]*batchProcessor),
	}
	bm.allDispatchers = append(bm.allDispatchers, dispatcher)
	for _, msgType := range msgTypes {
		bm.dispatcherMap[bm.getDispatcherKey(txType, msgType)] = dispatcher
	}
}

func (bm *batchManager) Start() error {
	go bm.messageSequencer()
	// We must be always ready to process DB events, or we block commits. So we have a dedicated worker for that
	go bm.newMessageNotifier()
	return nil
}

func (bm *batchManager) NewMessages() chan<- int64 {
	return bm.newMessages
}

func (bm *batchManager) getProcessor(txType core.TransactionType, msgType core.MessageType, group *fftypes.Bytes32, namespace string, signer *core.SignerRef) (*batchProcessor, error) {
	bm.dispatcherMux.Lock()
	defer bm.dispatcherMux.Unlock()

	dispatcherKey := bm.getDispatcherKey(txType, msgType)
	dispatcher, ok := bm.dispatcherMap[dispatcherKey]
	if !ok {
		return nil, i18n.NewError(bm.ctx, coremsgs.MsgUnregisteredBatchType, dispatcherKey)
	}
	name := bm.getProcessorKey(namespace, signer, group)
	processor, ok := dispatcher.processors[name]
	if !ok {
		processor = newBatchProcessor(
			bm,
			&batchProcessorConf{
				DispatcherOptions: dispatcher.options,
				name:              name,
				txType:            txType,
				dispatcherName:    dispatcher.name,
				namespace:         namespace,
				signer:            *signer,
				group:             group,
				dispatch:          dispatcher.handler,
			},
			bm.retry,
			bm.txHelper,
		)
		dispatcher.processors[name] = processor
		log.L(bm.ctx).Debugf("Created new processor: %s", name)
	}
	return processor, nil
}

func (bm *batchManager) assembleMessageData(id *fftypes.UUID) (msg *core.Message, retData core.DataArray, err error) {
	var foundAll = false
	err = bm.retry.Do(bm.ctx, "retrieve message", func(attempt int) (retry bool, err error) {
		msg, retData, foundAll, err = bm.data.GetMessageWithDataCached(bm.ctx, id)
		// continual retry for persistence error (distinct from not-found)
		return true, err
	})
	if err != nil {
		return nil, nil, err
	}
	if !foundAll {
		return nil, nil, i18n.NewError(bm.ctx, coremsgs.MsgDataNotFound, id)
	}
	return msg, retData, nil
}

// popRewind is called just before reading a page, to pop out a rewind offset if there is one and it's behind the cursor
func (bm *batchManager) popRewind() {
	bm.rewindOffsetMux.Lock()
	if bm.rewindOffset >= 0 && bm.rewindOffset < bm.readOffset {
		bm.readOffset = bm.rewindOffset
	}
	bm.rewindOffset = -1
	bm.rewindOffsetMux.Unlock()
}

// filterFlushed is called after we read a page, to remove in-flight IDs, and clean up our flush map
func (bm *batchManager) filterFlushed(entries []*core.IDAndSequence) []*core.IDAndSequence {
	bm.inflightMux.Lock()

	// Remove inflight entries
	unflushedEntries := make([]*core.IDAndSequence, 0, len(entries))
	for _, entry := range entries {
		if _, inflight := bm.inflightSequences[entry.Sequence]; !inflight {
			unflushedEntries = append(unflushedEntries, entry)
		}
	}

	// Drain the list of recently flushed entries that processors have notified us about
	for _, seq := range bm.inflightFlushed {
		delete(bm.inflightSequences, seq)
	}
	bm.inflightFlushed = bm.inflightFlushed[:0]

	bm.inflightMux.Unlock()

	return unflushedEntries
}

// notifyFlushed is called by a processor, when it's finished updating the database to record a set
// of messages as sent. So it's safe to remove these sequences from the inflight map on the next
// page read.
func (bm *batchManager) notifyFlushed(sequences []int64) {
	bm.inflightMux.Lock()
	bm.inflightFlushed = append(bm.inflightFlushed, sequences...)
	bm.inflightMux.Unlock()
}

func (bm *batchManager) readPage(lastPageFull bool) ([]*core.IDAndSequence, bool, error) {

	// Pop out any rewind that has been queued, but each time we read to the front before we rewind
	if !lastPageFull {
		bm.popRewind()
	}

	// Read a page from the DB
	var ids []*core.IDAndSequence
	err := bm.retry.Do(bm.ctx, "retrieve messages", func(attempt int) (retry bool, err error) {
		fb := database.MessageQueryFactory.NewFilterLimit(bm.ctx, bm.readPageSize)
		ids, err = bm.database.GetMessageIDs(bm.ctx, fb.And(
			fb.Gt("sequence", bm.readOffset),
			fb.Eq("state", core.MessageStateReady),
		).Sort("sequence").Limit(bm.readPageSize))
		return true, err
	})

	// Calculate if this was a full page we read (so should immediately re-poll) before we remove flushed IDs
	pageReadLength := len(ids)
	fullPage := (pageReadLength == int(bm.readPageSize))

	// Remove any flushed IDs from the list, and then update our flushed map
	ids = bm.filterFlushed(ids)

	log.L(bm.ctx).Debugf("Read %d records from offset %d. filtered=%d fullPage=%t", pageReadLength, bm.readOffset, len(ids), fullPage)
	return ids, fullPage, err
}

func (bm *batchManager) messageSequencer() {
	l := log.L(bm.ctx)
	l.Debugf("Started batch assembly message sequencer")
	defer close(bm.done)

	lastPageFull := false
	for {
		// Each time round the loop we check for quiescing processors
		bm.reapQuiescing()

		// Read messages from the DB - in an error condition we retry until success, or a closed context
		entries, fullPage, err := bm.readPage(lastPageFull)
		if err != nil {
			l.Debugf("Exiting: %s", err)
			return
		}

		if len(entries) > 0 {
			for _, entry := range entries {
				msg, data, err := bm.assembleMessageData(&entry.ID)
				if err != nil {
					l.Errorf("Failed to retrieve message data for %s (seq=%d): %s", entry.ID, entry.Sequence, err)
					continue
				}

				// We likely retrieved this message from the cache, which is written by the message-writer before
				// the database store. Meaning we cannot rely on the sequence having been set.
				msg.Sequence = entry.Sequence

				processor, err := bm.getProcessor(msg.Header.TxType, msg.Header.Type, msg.Header.Group, msg.Header.Namespace, &msg.Header.SignerRef)
				if err != nil {
					l.Errorf("Failed to dispatch message %s: %s", msg.Header.ID, err)
					continue
				}

				bm.dispatchMessage(processor, msg, data)
			}

			// Next time round only read after the messages we just processed (unless we get a tap to rewind)
			bm.readOffset = entries[len(entries)-1].Sequence
		}

		// Wait to be woken again
		if !fullPage {
			if done := bm.waitForNewMessages(); done {
				l.Debugf("Exiting: %s", err)
				return
			}
		}
		lastPageFull = fullPage
	}
}

func (bm *batchManager) newMessageNotification(seq int64) {
	rewindToQueue := int64(-1)

	// Determine if we need to queue a rewind
	bm.rewindOffsetMux.Lock()
	lastSequenceBeforeMsg := seq - 1
	if bm.rewindOffset == -1 || lastSequenceBeforeMsg < bm.rewindOffset {
		rewindToQueue = lastSequenceBeforeMsg
		bm.rewindOffset = lastSequenceBeforeMsg
	}
	bm.rewindOffsetMux.Unlock()

	if rewindToQueue >= 0 {
		log.L(bm.ctx).Debugf("Notifying batch manager of rewind to %d", rewindToQueue)
		select {
		case bm.shoulderTap <- true:
		default:
		}
	}
}

func (bm *batchManager) newMessageNotifier() {
	l := log.L(bm.ctx)
	for {
		select {
		case seq := <-bm.newMessages:
			bm.newMessageNotification(seq)
		case <-bm.ctx.Done():
			l.Debugf("Exiting due to cancelled context")
			return
		}
	}
}

func (bm *batchManager) waitForNewMessages() (done bool) {
	l := log.L(bm.ctx)

	// We have a short minimum timeout, to stop us thrashing the DB
	time.Sleep(bm.minimumPollDelay)

	timeout := time.NewTimer(bm.messagePollTimeout - bm.minimumPollDelay)
	select {
	case <-bm.shoulderTap:
		timeout.Stop()
		return false
	case <-timeout.C:
		l.Debugf("Woken after poll timeout")
		return false
	case <-bm.ctx.Done():
		l.Debugf("Exiting due to cancelled context")
		return true
	}
}

func (bm *batchManager) dispatchMessage(processor *batchProcessor, msg *core.Message, data core.DataArray) {
	l := log.L(bm.ctx)
	l.Debugf("Dispatching message %s (seq=%d) to %s batch processor %s", msg.Header.ID, msg.Sequence, msg.Header.Type, processor.conf.name)

	bm.inflightMux.Lock()
	bm.inflightSequences[msg.Sequence] = processor
	bm.inflightMux.Unlock()

	work := &batchWork{
		msg:  msg,
		data: data,
	}
	processor.newWork <- work
}

func (bm *batchManager) reapQuiescing() {
	bm.dispatcherMux.Lock()
	var reaped []*batchProcessor
	for _, d := range bm.allDispatchers {
		for k, p := range d.processors {
			select {
			case <-p.quiescing:
				// This is called on the goroutine where we dispatch the work, so it's safe to cleanup
				delete(d.processors, k)
				close(p.newWork)
				reaped = append(reaped, p)
			default:
			}
		}
	}
	bm.dispatcherMux.Unlock()

	for _, p := range reaped {
		// We wait for the current process to close, which should be immediate, but there is a tiny
		// chance that we dispatched one last message to it just as it was quiescing.
		// If that's the case, we don't want to spin up a new one, until we've finished the dispatch
		// of that piece of work that snuck in.
		<-p.done
	}
}

func (bm *batchManager) getProcessors() []*batchProcessor {
	bm.dispatcherMux.Lock()
	defer bm.dispatcherMux.Unlock()

	var processors []*batchProcessor
	for _, d := range bm.allDispatchers {
		for _, p := range d.processors {
			processors = append(processors, p)
		}
	}
	return processors
}

func (bm *batchManager) Status() *ManagerStatus {
	processors := bm.getProcessors()
	pStatus := make([]*ProcessorStatus, len(processors))
	for i, p := range processors {
		pStatus[i] = p.status()
	}
	return &ManagerStatus{
		Processors: pStatus,
	}
}

func (bm *batchManager) Close() {
	bm.cancelCtx() // all processor contexts are child contexts
}

func (bm *batchManager) WaitStop() {
	<-bm.done
	processors := bm.getProcessors()
	for _, p := range processors {
		<-p.done
	}
}
