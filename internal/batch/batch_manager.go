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
		newMessages:                make(chan int64, 1),
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
	}
	log.L(bm.ctx).Debugf("Created new processor: %s", name)
	return processor, nil
}

func (bm *batchManager) assembleMessageData(msg *fftypes.Message) (retData fftypes.DataArray, err error) {
	var foundAll = false
	err = bm.retry.Do(bm.ctx, fmt.Sprintf("assemble message %s data", msg.Header.ID), func(attempt int) (retry bool, err error) {
		retData, foundAll, err = bm.data.GetMessageDataCached(bm.ctx, msg)
		// continual retry for persistence error (distinct from not-found)
		return true, err
	})
	if err != nil {
		return nil, err
	}
	if !foundAll {
		return nil, i18n.NewError(bm.ctx, i18n.MsgDataNotFound, msg.Header.ID)
	}
	return retData, nil
}

func (bm *batchManager) readPage() ([]*fftypes.Message, error) {

	var msgs []*fftypes.Message
	err := bm.retry.Do(bm.ctx, "retrieve messages", func(attempt int) (retry bool, err error) {
		fb := database.MessageQueryFactory.NewFilterLimit(bm.ctx, bm.readPageSize)
		msgs, _, err = bm.database.GetMessages(bm.ctx, fb.And(
			fb.Gt("sequence", bm.readOffset),
			fb.Eq("state", fftypes.MessageStateReady),
		).Sort("sequence").Limit(bm.readPageSize))
		return true, err
	})
	return msgs, err
}

func (bm *batchManager) messageSequencer() {
	l := log.L(bm.ctx)
	l.Debugf("Started batch assembly message sequencer")
	defer close(bm.done)

	for {
		// Each time round the loop we check for quiescing processors
		bm.reapQuiescing()

		// Read messages from the DB - in an error condition we retry until success, or a closed context
		msgs, err := bm.readPage()
		if err != nil {
			l.Debugf("Exiting: %s", err)
			return
		}
		batchWasFull := (uint64(len(msgs)) == bm.readPageSize)

		if len(msgs) > 0 {
			for _, msg := range msgs {
				processor, err := bm.getProcessor(msg.Header.TxType, msg.Header.Type, msg.Header.Group, msg.Header.Namespace, &msg.Header.SignerRef)
				if err != nil {
					l.Errorf("Failed to dispatch message %s: %s", msg.Header.ID, err)
					continue
				}

				data, err := bm.assembleMessageData(msg)
				if err != nil {
					l.Errorf("Failed to retrieve message data for %s: %s", msg.Header.ID, err)
					continue
				}

				bm.dispatchMessage(processor, msg, data)
			}

			// Next time round only read after the messages we just processed (unless we get a tap to rewind)
			bm.readOffset = msgs[len(msgs)-1].Sequence
		}

		// Wait to be woken again
		if !batchWasFull && !bm.drainNewMessages() {
			if done := bm.waitForNewMessages(); done {
				l.Debugf("Exiting: %s", err)
				return
			}
		}
	}
}

func (bm *batchManager) newMessageNotification(seq int64) {
	log.L(bm.ctx).Debugf("Notification of message %d", seq)
	// The readOffset is the last sequence we have already read.
	// So we need to ensure it is at least one earlier, than this message sequence
	lastSequenceBeforeMsg := seq - 1
	if lastSequenceBeforeMsg < bm.readOffset {
		bm.readOffset = lastSequenceBeforeMsg
	}
}

func (bm *batchManager) drainNewMessages() bool {
	// Drain any new message notifications, moving back our readOffset as required
	newMessages := false
	checkingMessages := true
	for checkingMessages {
		select {
		case seq := <-bm.newMessages:
			bm.newMessageNotification(seq)
			newMessages = true
		default:
			checkingMessages = false
		}
	}
	return newMessages
}

func (bm *batchManager) waitForNewMessages() (done bool) {
	l := log.L(bm.ctx)

	// Otherwise set a timeout
	timeout := time.NewTimer(bm.messagePollTimeout)
	select {
	case seq := <-bm.newMessages:
		bm.newMessageNotification(seq)
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
	l.Debugf("Dispatching message %s to %s batch processor %s", msg.Header.ID, msg.Header.Type, processor.conf.name)
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
