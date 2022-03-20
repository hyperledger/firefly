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

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func NewBatchManager(ctx context.Context, ni sysmessaging.LocalNodeInfo, di database.Plugin, dm data.Manager, txHelper txcommon.Helper) (Manager, error) {
	if di == nil || dm == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	pCtx, cancelCtx := context.WithCancel(log.WithLogField(ctx, "role", "batchmgr"))
	readPageSize := config.GetUint(config.BatchManagerReadPageSize)
	bm := &batchManager{
		ctx:                        pCtx,
		cancelCtx:                  cancelCtx,
		ni:                         ni,
		database:                   di,
		data:                       dm,
		txHelper:                   txHelper,
		readOffset:                 -1, // On restart we trawl for all ready messages
		readPageSize:               uint64(readPageSize),
		messagePollTimeout:         config.GetDuration(config.BatchManagerReadPollTimeout),
		startupOffsetRetryAttempts: config.GetInt(config.OrchestratorStartupAttempts),
		dispatcherMap:              make(map[string]*dispatcher),
		allDispatchers:             make([]*dispatcher, 0),
		newMessages:                make(chan int64, readPageSize),
		shoulderTap:                make(chan bool, 1),
		rewindOffset:               -1,
		done:                       make(chan struct{}),
		retry: &retry.Retry{
			InitialDelay: config.GetDuration(config.BatchRetryInitDelay),
			MaximumDelay: config.GetDuration(config.BatchRetryMaxDelay),
			Factor:       config.GetFloat64(config.BatchRetryFactor),
		},
	}
	return bm, nil
}

type Manager interface {
	RegisterDispatcher(name string, txType fftypes.TransactionType, msgTypes []fftypes.MessageType, handler DispatchHandler, batchOptions DispatcherOptions)
	NewMessages() chan<- int64
	Start() error
	Close()
	WaitStop()
	Status() *ManagerStatus
}

type ManagerStatus struct {
	Processors []*ProcessorStatus `json:"processors"`
}

type ProcessorStatus struct {
	Dispatcher string      `json:"dispatcher"`
	Name       string      `json:"name"`
	Status     FlushStatus `json:"status"`
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
	shoulderTap                chan bool
	readPageSize               uint64
	messagePollTimeout         time.Duration
	startupOffsetRetryAttempts int
}

type DispatchHandler func(context.Context, *DispatchState) error

type DispatcherOptions struct {
	BatchType      fftypes.BatchType
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

func (bm *batchManager) getProcessorKey(namespace string, identity *fftypes.SignerRef, groupID *fftypes.Bytes32) string {
	return fmt.Sprintf("%s|%s|%v", namespace, identity.Author, groupID)
}

func (bm *batchManager) getDispatcherKey(txType fftypes.TransactionType, msgType fftypes.MessageType) string {
	return fmt.Sprintf("tx:%s/%s", txType, msgType)
}

func (bm *batchManager) RegisterDispatcher(name string, txType fftypes.TransactionType, msgTypes []fftypes.MessageType, handler DispatchHandler, options DispatcherOptions) {
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

func (bm *batchManager) getProcessor(txType fftypes.TransactionType, msgType fftypes.MessageType, group *fftypes.Bytes32, namespace string, signer *fftypes.SignerRef) (*batchProcessor, error) {
	bm.dispatcherMux.Lock()
	defer bm.dispatcherMux.Unlock()

	dispatcherKey := bm.getDispatcherKey(txType, msgType)
	dispatcher, ok := bm.dispatcherMap[dispatcherKey]
	if !ok {
		return nil, i18n.NewError(bm.ctx, i18n.MsgUnregisteredBatchType, dispatcherKey)
	}
	name := bm.getProcessorKey(namespace, signer, group)
	processor, ok := dispatcher.processors[name]
	if !ok {
		processor = newBatchProcessor(
			bm.ctx, // Background context, not the call context
			bm.ni,
			bm.database,
			bm.data,
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

func (bm *batchManager) assembleMessageData(id *fftypes.UUID) (msg *fftypes.Message, retData fftypes.DataArray, err error) {
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
		return nil, nil, i18n.NewError(bm.ctx, i18n.MsgDataNotFound, id)
	}
	return msg, retData, nil
}

func (bm *batchManager) readPage() ([]*fftypes.IDAndSequence, error) {

	// Pop out a rewind offset if there is one and it's behind the cursor
	bm.rewindOffsetMux.Lock()
	rewindOffset := bm.rewindOffset
	if rewindOffset >= 0 && rewindOffset < bm.readOffset {
		bm.readOffset = rewindOffset
	}
	bm.rewindOffset = -1
	bm.rewindOffsetMux.Unlock()

	var ids []*fftypes.IDAndSequence
	err := bm.retry.Do(bm.ctx, "retrieve messages", func(attempt int) (retry bool, err error) {
		fb := database.MessageQueryFactory.NewFilterLimit(bm.ctx, bm.readPageSize)
		ids, err = bm.database.GetMessageIDs(bm.ctx, fb.And(
			fb.Gt("sequence", bm.readOffset),
			fb.Eq("state", fftypes.MessageStateReady),
		).Sort("sequence").Limit(bm.readPageSize))
		return true, err
	})
	return ids, err
}

func (bm *batchManager) messageSequencer() {
	l := log.L(bm.ctx)
	l.Debugf("Started batch assembly message sequencer")
	defer close(bm.done)

	for {
		// Each time round the loop we check for quiescing processors
		bm.reapQuiescing()

		// Read messages from the DB - in an error condition we retry until success, or a closed context
		entries, err := bm.readPage()
		if err != nil {
			l.Debugf("Exiting: %s", err)
			return
		}
		batchWasFull := (uint64(len(entries)) == bm.readPageSize)

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
		if !batchWasFull {
			if done := bm.waitForNewMessages(); done {
				l.Debugf("Exiting: %s", err)
				return
			}
		}
	}
}

func (bm *batchManager) newMessageNotification(seq int64) {
	// Determine if we need to queue q rewind
	bm.rewindOffsetMux.Lock()
	lastSequenceBeforeMsg := seq - 1
	if bm.rewindOffset == -1 || lastSequenceBeforeMsg < bm.rewindOffset {
		bm.rewindOffset = lastSequenceBeforeMsg
	}
	bm.rewindOffsetMux.Unlock()
	// Shoulder tap that there is a new message, regardless of whether we rewound
	// the cursor. As we need to wake up the poll.
	select {
	case bm.shoulderTap <- true:
	default:
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

	timeout := time.NewTimer(bm.messagePollTimeout)
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

func (bm *batchManager) dispatchMessage(processor *batchProcessor, msg *fftypes.Message, data fftypes.DataArray) {
	l := log.L(bm.ctx)
	l.Debugf("Dispatching message %s (seq=%d) to %s batch processor %s", msg.Header.ID, msg.Sequence, msg.Header.Type, processor.conf.name)
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
			case <-p.quescing:
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
