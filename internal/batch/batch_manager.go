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

const (
	msgBatchOffsetName = "ff_msgbatch"
)

func NewBatchManager(ctx context.Context, ni sysmessaging.LocalNodeInfo, di database.Plugin, dm data.Manager) (Manager, error) {
	if di == nil || dm == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	readPageSize := config.GetUint(config.BatchManagerReadPageSize)
	bm := &batchManager{
		ctx:                        log.WithLogField(ctx, "role", "batchmgr"),
		ni:                         ni,
		database:                   di,
		data:                       dm,
		readOffset:                 -1, // On restart we trawl for all ready messages
		readPageSize:               uint64(readPageSize),
		messagePollTimeout:         config.GetDuration(config.BatchManagerReadPollTimeout),
		startupOffsetRetryAttempts: config.GetInt(config.OrchestratorStartupAttempts),
		dispatchers:                make(map[fftypes.MessageType]*dispatcher),
		newMessages:                make(chan int64),
		sequencerClosed:            make(chan struct{}),
		retry: &retry.Retry{
			InitialDelay: config.GetDuration(config.BatchRetryInitDelay),
			MaximumDelay: config.GetDuration(config.BatchRetryMaxDelay),
			Factor:       config.GetFloat64(config.BatchRetryFactor),
		},
	}
	return bm, nil
}

type Manager interface {
	RegisterDispatcher(msgTypes []fftypes.MessageType, handler DispatchHandler, batchOptions Options)
	NewMessages() chan<- int64
	Start() error
	Close()
	WaitStop()
}

type batchManager struct {
	ctx                        context.Context
	ni                         sysmessaging.LocalNodeInfo
	database                   database.Plugin
	data                       data.Manager
	dispatchers                map[fftypes.MessageType]*dispatcher
	newMessages                chan int64
	sequencerClosed            chan struct{}
	retry                      *retry.Retry
	offsetID                   int64
	recoveryOffset             int64
	readOffset                 int64
	closed                     bool
	readPageSize               uint64
	messagePollTimeout         time.Duration
	startupOffsetRetryAttempts int
}

type DispatchHandler func(context.Context, *fftypes.Batch, []*fftypes.Bytes32) error

type Options struct {
	BatchMaxSize   uint
	BatchMaxBytes  int64
	BatchTimeout   time.Duration
	DisposeTimeout time.Duration
}

type dispatcher struct {
	handler      DispatchHandler
	mux          sync.Mutex
	processors   map[string]*batchProcessor
	batchOptions Options
}

func (bm *batchManager) RegisterDispatcher(msgTypes []fftypes.MessageType, handler DispatchHandler, batchOptions Options) {
	dispatcher := &dispatcher{
		handler:      handler,
		batchOptions: batchOptions,
		processors:   make(map[string]*batchProcessor),
	}
	for _, msgType := range msgTypes {
		bm.dispatchers[msgType] = dispatcher
	}
}

func (bm *batchManager) Start() error {
	if err := bm.restoreOffset(); err != nil {
		return err
	}
	go bm.messageSequencer()
	return nil
}

func (bm *batchManager) NewMessages() chan<- int64 {
	return bm.newMessages
}

func (bm *batchManager) restoreOffset() (err error) {
	var offset *fftypes.Offset
	for offset == nil {
		offset, err = bm.database.GetOffset(bm.ctx, fftypes.OffsetTypeBatch, msgBatchOffsetName)
		if err != nil {
			return err
		}
		if offset == nil {
			_ = bm.database.UpsertOffset(bm.ctx, &fftypes.Offset{
				Type:    fftypes.OffsetTypeBatch,
				Name:    msgBatchOffsetName,
				Current: 0,
			}, false)
		}
	}
	bm.offsetID = offset.RowID
	bm.readOffset = offset.Current
	bm.recoveryOffset = offset.Current
	log.L(bm.ctx).Infof("Batch manager restored offset %d", offset.Current)
	return nil
}

func (bm *batchManager) getProcessor(batchType fftypes.MessageType, group *fftypes.Bytes32, namespace string, identity *fftypes.Identity) (*batchProcessor, error) {
	dispatcher, ok := bm.dispatchers[batchType]
	if !ok {
		return nil, i18n.NewError(bm.ctx, i18n.MsgUnregisteredBatchType, batchType)
	}
	dispatcher.mux.Lock()
	key := fmt.Sprintf("%s:%s:%s[group=%v]", namespace, identity.Author, identity.Key, group)
	processor, ok := dispatcher.processors[key]
	if !ok {
		processor = newBatchProcessor(
			bm.ctx, // Background context, not the call context
			bm.ni,
			bm.database,
			&batchProcessorConf{
				Options:   dispatcher.batchOptions,
				namespace: namespace,
				identity:  *identity,
				group:     group,
				dispatch:  dispatcher.handler,
			},
			bm.retry,
		)
		dispatcher.processors[key] = processor
	}
	log.L(bm.ctx).Debugf("Created new processor: %s", key)
	dispatcher.mux.Unlock()
	return processor, nil
}

func (bm *batchManager) Close() {
	var processors []*batchProcessor
	if bm != nil && !bm.closed {
		for _, d := range bm.dispatchers {
			d.mux.Lock()
			for _, p := range d.processors {
				processors = append(processors, p)
				p.close()
			}
			d.mux.Unlock()
		}
		bm.closed = true
		close(bm.newMessages)
	}
	bm = nil
	for _, p := range processors {
		<-p.done
	}
}

func (bm *batchManager) assembleMessageData(msg *fftypes.Message) (data []*fftypes.Data, err error) {
	var foundAll = false
	err = bm.retry.Do(bm.ctx, fmt.Sprintf("assemble message %s data", msg.Header.ID), func(attempt int) (retry bool, err error) {
		data, foundAll, err = bm.data.GetMessageData(bm.ctx, msg, true)
		// continual retry for persistence error (distinct from not-found)
		return err != nil && !bm.closed, err
	})
	if err != nil {
		return nil, err
	}
	if !foundAll {
		return nil, i18n.NewError(bm.ctx, i18n.MsgDataNotFound, msg.Header.ID)
	}
	log.L(bm.ctx).Infof("Detected new batch-pinned message %s sequence=%d", msg.Header.ID, msg.Sequence)
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
		if err != nil {
			return !bm.closed, err // Retry indefinitely, until closed (or context cancelled)
		}
		return false, nil
	})
	return msgs, err
}

func (bm *batchManager) messageSequencer() {
	l := log.L(bm.ctx)
	l.Debugf("Started batch assembly message sequencer")
	defer close(bm.sequencerClosed)

	dispatched := make(chan *batchDispatch, bm.readPageSize)

	for !bm.closed {
		// Read messages from the DB - in an error condition we retry until success, or a closed context
		msgs, err := bm.readPage()
		if err != nil {
			l.Debugf("Exiting: %s", err) // errors logged in readPage
			return
		}
		batchWasFull := false

		if len(msgs) > 0 {
			batchWasFull = (uint64(len(msgs)) == bm.readPageSize)
			var dispatchCount int
			for _, msg := range msgs {
				data, err := bm.assembleMessageData(msg)
				if err != nil {
					l.Errorf("Failed to retrieve message data for %s: %s", msg.Header.ID, err)
					continue
				}

				err = bm.dispatchMessage(dispatched, msg, data...)
				if err != nil {
					l.Errorf("Failed to dispatch message %s: %s", msg.Header.ID, err)
					continue
				}
				dispatchCount++
			}

			for i := 0; i < dispatchCount; i++ {
				select {
				case dispatched := <-dispatched:
					l.Debugf("Dispatched message %s to batch %s", dispatched.msg.Header.ID, dispatched.batchID)
				case <-bm.ctx.Done():
					l.Debugf("Message sequencer exiting (context closed)")
					bm.Close()
					return
				}
			}

			// Next time round only read after the messages we just processed (unless we get a tap to rewind)
			bm.readOffset = msgs[len(msgs)-1].Sequence
		}

		// Wait to be woken again
		if !bm.closed && !batchWasFull {
			bm.waitForShoulderTapOrPollTimeout()
		}
	}
}

func (bm *batchManager) waitForShoulderTapOrPollTimeout() {
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
		return
	}

	// Otherwise set a timeout
	timeout := time.NewTimer(bm.messagePollTimeout)
	select {
	case <-timeout.C:
		l.Debugf("Woken after poll timeout")
	case <-bm.ctx.Done():
		l.Debugf("Exiting due to cancelled context")
		bm.Close()
		return
	}
}

func (bm *batchManager) dispatchMessage(dispatched chan *batchDispatch, msg *fftypes.Message, data ...*fftypes.Data) error {
	l := log.L(bm.ctx)
	processor, err := bm.getProcessor(msg.Header.Type, msg.Header.Group, msg.Header.Namespace, &msg.Header.Identity)
	if err != nil {
		return err
	}
	l.Debugf("Dispatching message %s to %s batch", msg.Header.ID, msg.Header.Type)
	work := &batchWork{
		msg:        msg,
		data:       data,
		dispatched: dispatched,
	}
	processor.newWork <- work
	return nil
}

func (bm *batchManager) WaitStop() {
	<-bm.sequencerClosed
	var processors []*batchProcessor
	for _, d := range bm.dispatchers {
		d.mux.Lock()
		for _, p := range d.processors {
			processors = append(processors, p)
		}
		d.mux.Unlock()
	}
	for _, p := range processors {
		p.close()
	}
}
