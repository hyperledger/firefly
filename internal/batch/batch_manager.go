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
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func NewBatchManager(ctx context.Context, ni sysmessaging.LocalNodeInfo, di database.Plugin, dm data.Manager) (Manager, error) {
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
		readOffset:                 -1, // On restart we trawl for all ready messages
		readPageSize:               uint64(readPageSize),
		messagePollTimeout:         config.GetDuration(config.BatchManagerReadPollTimeout),
		startupOffsetRetryAttempts: config.GetInt(config.OrchestratorStartupAttempts),
		dispatchers:                make(map[string]*dispatcher),
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
	Status() []*ProcessorStatus
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
	dispatcherMux              sync.Mutex
	dispatchers                map[string]*dispatcher
	newMessages                chan int64
	done                       chan struct{}
	retry                      *retry.Retry
	readOffset                 int64
	readPageSize               uint64
	messagePollTimeout         time.Duration
	startupOffsetRetryAttempts int
}

type DispatchHandler func(context.Context, *fftypes.Batch, []*fftypes.Bytes32) error

type DispatcherOptions struct {
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

func (bm *batchManager) getDispatcherKey(txType fftypes.TransactionType, msgType fftypes.MessageType) string {
	return fmt.Sprintf("tx:%s/%s", txType, msgType)
}

func (bm *batchManager) RegisterDispatcher(name string, txType fftypes.TransactionType, msgTypes []fftypes.MessageType, handler DispatchHandler, options DispatcherOptions) {
	dispatcher := &dispatcher{
		name:       name,
		handler:    handler,
		options:    options,
		processors: make(map[string]*batchProcessor),
	}
	for _, msgType := range msgTypes {
		bm.dispatchers[bm.getDispatcherKey(txType, msgType)] = dispatcher
	}
}

func (bm *batchManager) Start() error {
	go bm.messageSequencer()
	return nil
}

func (bm *batchManager) NewMessages() chan<- int64 {
	return bm.newMessages
}

func (bm *batchManager) getProcessor(txType fftypes.TransactionType, msgType fftypes.MessageType, group *fftypes.Bytes32, namespace string, identity *fftypes.Identity) (*batchProcessor, error) {
	bm.dispatcherMux.Lock()
	defer bm.dispatcherMux.Unlock()

	dispatcherKey := bm.getDispatcherKey(txType, msgType)
	dispatcher, ok := bm.dispatchers[dispatcherKey]
	if !ok {
		return nil, i18n.NewError(bm.ctx, i18n.MsgUnregisteredBatchType, dispatcherKey)
	}
	name := fmt.Sprintf("%s|%s|%v", namespace, identity.Author, group)
	processor, ok := dispatcher.processors[name]
	if !ok {
		processor = newBatchProcessor(
			bm.ctx, // Background context, not the call context
			bm.ni,
			bm.database,
			&batchProcessorConf{
				DispatcherOptions: dispatcher.options,
				name:              name,
				txType:            txType,
				dispatcherName:    dispatcher.name,
				namespace:         namespace,
				identity:          *identity,
				group:             group,
				dispatch:          dispatcher.handler,
			},
			bm.retry,
		)
		dispatcher.processors[name] = processor
	}
	log.L(bm.ctx).Debugf("Created new processor: %s", name)
	return processor, nil
}

func (bm *batchManager) assembleMessageData(msg *fftypes.Message) (data []*fftypes.Data, err error) {
	var foundAll = false
	err = bm.retry.Do(bm.ctx, fmt.Sprintf("assemble message %s data", msg.Header.ID), func(attempt int) (retry bool, err error) {
		data, foundAll, err = bm.data.GetMessageData(bm.ctx, msg, true)
		// continual retry for persistence error (distinct from not-found)
		return true, err
	})
	if err != nil {
		return nil, err
	}
	if !foundAll {
		return nil, i18n.NewError(bm.ctx, i18n.MsgDataNotFound, msg.Header.ID)
	}
	return data, nil
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
				data, err := bm.assembleMessageData(msg)
				if err != nil {
					l.Errorf("Failed to retrieve message data for %s: %s", msg.Header.ID, err)
					continue
				}

				err = bm.dispatchMessage(msg, data...)
				if err != nil {
					l.Errorf("Failed to dispatch message %s: %s", msg.Header.ID, err)
					continue
				}
			}

			// Next time round only read after the messages we just processed (unless we get a tap to rewind)
			bm.readOffset = msgs[len(msgs)-1].Sequence
		}

		// Wait to be woken again
		if !batchWasFull {
			if done := bm.waitForShoulderTapOrPollTimeout(); done {
				l.Debugf("Exiting: %s", err)
				return
			}
		}
	}
}

func (bm *batchManager) waitForShoulderTapOrPollTimeout() (done bool) {
	l := log.L(bm.ctx)

	// Drain any new message notifications, moving back our
	// readOffset as required
	newMessages := false
	checkingMessages := true
	for checkingMessages {
		select {
		case seq := <-bm.newMessages:
			l.Debugf("Notification of message %d", seq)
			if (seq - 1) < bm.readOffset {
				bm.readOffset = seq - 1
			}
			newMessages = true
		default:
			checkingMessages = false
		}
	}
	if newMessages {
		return false
	}

	// Otherwise set a timeout
	timeout := time.NewTimer(bm.messagePollTimeout)
	select {
	case <-timeout.C:
		l.Debugf("Woken after poll timeout")
		return false
	case <-bm.ctx.Done():
		l.Debugf("Exiting due to cancelled context")
		return true
	}
}

func (bm *batchManager) dispatchMessage(msg *fftypes.Message, data ...*fftypes.Data) error {
	l := log.L(bm.ctx)
	processor, err := bm.getProcessor(msg.Header.TxType, msg.Header.Type, msg.Header.Group, msg.Header.Namespace, &msg.Header.Identity)
	if err != nil {
		return err
	}
	l.Debugf("Dispatching message %s to %s batch processor %s", msg.Header.ID, msg.Header.Type, processor.conf.name)
	work := &batchWork{
		msg:  msg,
		data: data,
	}
	processor.newWork <- work
	return nil
}

func (bm *batchManager) reapQuiescing() {
	bm.dispatcherMux.Lock()
	defer bm.dispatcherMux.Unlock()

	for _, d := range bm.dispatchers {
		for k, p := range d.processors {
			select {
			case <-p.quescing:
				// This is called on the goroutine where we dispatch the work, so it's safe to cleanup
				delete(d.processors, k)
				close(p.newWork)
			default:
			}
		}
	}
}

func (bm *batchManager) getProcessors() []*batchProcessor {
	bm.dispatcherMux.Lock()
	defer bm.dispatcherMux.Unlock()

	var processors []*batchProcessor
	for _, d := range bm.dispatchers {
		for _, p := range d.processors {
			processors = append(processors, p)
		}
	}
	return processors
}

func (bm *batchManager) Status() []*ProcessorStatus {
	processors := bm.getProcessors()
	status := make([]*ProcessorStatus, len(processors))
	for i, p := range processors {
		status[i] = p.status()
	}
	return status
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
