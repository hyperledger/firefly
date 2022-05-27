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

func NewBatchManager(ctx context.Context, ni sysmessaging.LocalNodeInfo, databases map[string]database.Plugin, dm data.Manager, txHelper txcommon.Helper) (Manager, error) {
	if len(databases) == 0 || dm == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "BatchManager")
	}
	pCtx, cancelCtx := context.WithCancel(log.WithLogField(ctx, "role", "batchmgr"))
	readPageSize := config.GetUint(coreconfig.BatchManagerReadPageSize)
	retry := &retry.Retry{
		InitialDelay: config.GetDuration(coreconfig.BatchRetryInitDelay),
		MaximumDelay: config.GetDuration(coreconfig.BatchRetryMaxDelay),
		Factor:       config.GetFloat64(coreconfig.BatchRetryFactor),
	}
	bm := &batchManager{
		ctx:                pCtx,
		cancelCtx:          cancelCtx,
		ni:                 ni,
		done:               make(chan struct{}),
		txHelper:           txHelper,
		minimumPollDelay:   config.GetDuration(coreconfig.BatchManagerMinimumPollDelay),
		messagePollTimeout: config.GetDuration(coreconfig.BatchManagerReadPollTimeout),
		dispatcherMap:      make(map[string]*dispatcher),
		allDispatchers:     make([]*dispatcher, 0),
		shoulderTap:        make(chan bool, 1),
		retry:              retry,
		assemblers:         make(map[database.Plugin]*batchAssembler, len(databases)),
	}
	for _, database := range databases {
		bm.assemblers[database] = &batchAssembler{
			ctx:               pCtx,
			database:          database,
			data:              dm,
			readOffset:        -1, // On restart we trawl for all ready messages
			readPageSize:      uint64(readPageSize),
			newMessages:       make(chan int64, readPageSize),
			inflightSequences: make(map[int64]*batchProcessor),
			rewindOffset:      -1,
			retry:             retry,
		}
	}
	return bm, nil
}

type Manager interface {
	RegisterDispatcher(name string, txType core.TransactionType, msgTypes []core.MessageType, handler DispatchHandler, batchOptions DispatcherOptions)
	NewMessage(database database.Plugin, seq int64)
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
	ctx                context.Context
	cancelCtx          func()
	ni                 sysmessaging.LocalNodeInfo
	txHelper           txcommon.Helper
	dispatcherMux      sync.Mutex
	dispatcherMap      map[string]*dispatcher
	allDispatchers     []*dispatcher
	done               chan struct{}
	retry              *retry.Retry
	shoulderTap        chan bool
	minimumPollDelay   time.Duration
	messagePollTimeout time.Duration
	assemblers         map[database.Plugin]*batchAssembler
}

type batchAssembler struct {
	ctx               context.Context
	database          database.Plugin
	data              data.Manager
	newMessages       chan int64
	retry             *retry.Retry
	readOffset        int64
	rewindOffsetMux   sync.Mutex
	rewindOffset      int64
	inflightMux       sync.Mutex
	inflightSequences map[int64]*batchProcessor
	inflightFlushed   []int64
	readPageSize      uint64
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

func getProcessorKey(namespace string, identity *core.SignerRef, groupID *fftypes.Bytes32) string {
	return fmt.Sprintf("%s|%s|%v", namespace, identity.Author, groupID)
}

func getDispatcherKey(txType core.TransactionType, msgType core.MessageType) string {
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
		bm.dispatcherMap[getDispatcherKey(txType, msgType)] = dispatcher
	}
}

func (bm *batchManager) Start() error {
	go bm.messageSequencer()
	for _, assembler := range bm.assemblers {
		// We must be always ready to process DB events, or we block commits. So we have a dedicated worker for that
		go assembler.newMessageNotifier(bm.shoulderTap)
	}
	return nil
}

func (bm *batchManager) NewMessage(database database.Plugin, seq int64) {
	assembler := bm.assemblers[database]
	assembler.newMessages <- seq
}

func (bm *batchManager) getProcessor(assembler *batchAssembler, txType core.TransactionType, msgType core.MessageType, group *fftypes.Bytes32, namespace string, signer *core.SignerRef) (*batchProcessor, error) {
	bm.dispatcherMux.Lock()
	defer bm.dispatcherMux.Unlock()

	dispatcherKey := getDispatcherKey(txType, msgType)
	dispatcher, ok := bm.dispatcherMap[dispatcherKey]
	if !ok {
		return nil, i18n.NewError(bm.ctx, coremsgs.MsgUnregisteredBatchType, dispatcherKey)
	}
	// The processor key must ensure that a given processor will never be grabbed by two different assemblers
	// This is true because:
	//   - processor key includes the namespace name
	//   - each namespace has exactly one database plugin
	//   - each database plugin has exactly one assembler
	name := getProcessorKey(namespace, signer, group)
	processor, ok := dispatcher.processors[name]
	if !ok {
		processor = newBatchProcessor(
			assembler,
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
			bm.ni,
			bm.txHelper,
		)
		dispatcher.processors[name] = processor
		log.L(bm.ctx).Debugf("Created new processor: %s", name)
	}
	return processor, nil
}

func (ba *batchAssembler) assembleMessageData(id *fftypes.UUID) (msg *core.Message, retData core.DataArray, err error) {
	var foundAll = false
	err = ba.retry.Do(ba.ctx, "retrieve message", func(attempt int) (retry bool, err error) {
		msg, retData, foundAll, err = ba.data.GetMessageWithDataCached(ba.ctx, id)
		// continual retry for persistence error (distinct from not-found)
		return true, err
	})
	if err != nil {
		return nil, nil, err
	}
	if !foundAll {
		return nil, nil, i18n.NewError(ba.ctx, coremsgs.MsgDataNotFound, id)
	}
	return msg, retData, nil
}

// popRewind is called just before reading a page, to pop out a rewind offset if there is one and it's behind the cursor
func (ba *batchAssembler) popRewind() {
	ba.rewindOffsetMux.Lock()
	if ba.rewindOffset >= 0 && ba.rewindOffset < ba.readOffset {
		ba.readOffset = ba.rewindOffset
	}
	ba.rewindOffset = -1
	ba.rewindOffsetMux.Unlock()
}

// filterFlushed is called after we read a page, to remove in-flight IDs, and clean up our flush map
func (ba *batchAssembler) filterFlushed(entries []*core.IDAndSequence) []*core.IDAndSequence {
	ba.inflightMux.Lock()

	// Remove inflight entries
	unflushedEntries := make([]*core.IDAndSequence, 0, len(entries))
	for _, entry := range entries {
		if _, inflight := ba.inflightSequences[entry.Sequence]; !inflight {
			unflushedEntries = append(unflushedEntries, entry)
		}
	}

	// Drain the list of recently flushed entries that processors have notified us about
	for _, seq := range ba.inflightFlushed {
		delete(ba.inflightSequences, seq)
	}
	ba.inflightFlushed = ba.inflightFlushed[:0]

	ba.inflightMux.Unlock()

	return unflushedEntries
}

// notifyFlushed is called by a processor, when it's finished updating the database to record a set
// of messages as sent. So it's safe to remove these sequences from the inflight map on the next
// page read.
func (ba *batchAssembler) notifyFlushed(sequences []int64) {
	ba.inflightMux.Lock()
	ba.inflightFlushed = append(ba.inflightFlushed, sequences...)
	ba.inflightMux.Unlock()
}

func (ba *batchAssembler) readPage(lastPageFull bool) ([]*core.IDAndSequence, bool, error) {

	// Pop out any rewind that has been queued, but each time we read to the front before we rewind
	if !lastPageFull {
		ba.popRewind()
	}

	// Read a page from the DB
	var ids []*core.IDAndSequence
	err := ba.retry.Do(ba.ctx, "retrieve messages", func(attempt int) (retry bool, err error) {
		fb := database.MessageQueryFactory.NewFilterLimit(ba.ctx, ba.readPageSize)
		ids, err = ba.database.GetMessageIDs(ba.ctx, fb.And(
			fb.Gt("sequence", ba.readOffset),
			fb.Eq("state", core.MessageStateReady),
		).Sort("sequence").Limit(ba.readPageSize))
		return true, err
	})

	// Calculate if this was a full page we read (so should immediately re-poll) before we remove flushed IDs
	pageReadLength := len(ids)
	fullPage := (pageReadLength == int(ba.readPageSize))

	// Remove any flushed IDs from the list, and then update our flushed map
	ids = ba.filterFlushed(ids)

	log.L(ba.ctx).Debugf("Read %d records from offset %d. filtered=%d fullPage=%t", pageReadLength, ba.readOffset, len(ids), fullPage)
	return ids, fullPage, err
}

func (bm *batchManager) messageSequencer() {
	l := log.L(bm.ctx)
	l.Debugf("Started batch assembly message sequencer")
	defer close(bm.done)

	lastPageFull := make(map[*batchAssembler]bool)
	for {
		// Each time round the loop we check for quiescing processors
		bm.reapQuiescing()

		anyPageFull := false
		for _, assembler := range bm.assemblers {
			// Read messages from the DB - in an error condition we retry until success, or a closed context
			entries, fullPage, err := assembler.readPage(lastPageFull[assembler])
			if err != nil {
				l.Debugf("Exiting: %s", err)
				return
			}

			if len(entries) > 0 {
				for _, entry := range entries {
					msg, data, err := assembler.assembleMessageData(&entry.ID)
					if err != nil {
						l.Errorf("Failed to retrieve message data for %s (seq=%d): %s", entry.ID, entry.Sequence, err)
						continue
					}

					// We likely retrieved this message from the cache, which is written by the message-writer before
					// the database store. Meaning we cannot rely on the sequence having been set.
					msg.Sequence = entry.Sequence

					processor, err := bm.getProcessor(assembler, msg.Header.TxType, msg.Header.Type, msg.Header.Group, msg.Header.Namespace, &msg.Header.SignerRef)
					if err != nil {
						l.Errorf("Failed to dispatch message %s: %s", msg.Header.ID, err)
						continue
					}

					assembler.dispatchMessage(processor, msg, data)
				}

				// Next time round only read after the messages we just processed (unless we get a tap to rewind)
				assembler.readOffset = entries[len(entries)-1].Sequence
			}
			lastPageFull[assembler] = fullPage
			anyPageFull = anyPageFull || fullPage
		}

		// Wait to be woken again
		if !anyPageFull {
			if done := bm.waitForNewMessages(); done {
				l.Debugf("Exiting")
				return
			}
		}
	}
}

func (ba *batchAssembler) newMessageNotification(shoulderTap chan bool, seq int64) {
	rewindToQueue := int64(-1)

	// Determine if we need to queue a rewind
	ba.rewindOffsetMux.Lock()
	lastSequenceBeforeMsg := seq - 1
	if ba.rewindOffset == -1 || lastSequenceBeforeMsg < ba.rewindOffset {
		rewindToQueue = lastSequenceBeforeMsg
		ba.rewindOffset = lastSequenceBeforeMsg
	}
	ba.rewindOffsetMux.Unlock()

	if rewindToQueue >= 0 {
		log.L(ba.ctx).Debugf("Notifying batch manager of rewind to %d", rewindToQueue)
		select {
		case shoulderTap <- true:
		default:
		}
	}
}

func (ba *batchAssembler) newMessageNotifier(shoulderTap chan bool) {
	l := log.L(ba.ctx)
	for {
		select {
		case seq := <-ba.newMessages:
			ba.newMessageNotification(shoulderTap, seq)
		case <-ba.ctx.Done():
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

func (ba *batchAssembler) dispatchMessage(processor *batchProcessor, msg *core.Message, data core.DataArray) {
	l := log.L(ba.ctx)
	l.Debugf("Dispatching message %s (seq=%d) to %s batch processor %s", msg.Header.ID, msg.Sequence, msg.Header.Type, processor.conf.name)

	ba.inflightMux.Lock()
	ba.inflightSequences[msg.Sequence] = processor
	ba.inflightMux.Unlock()

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
