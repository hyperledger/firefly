// Copyright © 2021 Kaleido, Inc.
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

package batching

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/retry"
)

const (
	msgBatchOffsetName = "ff-msgbatch"
)

func NewBatchManager(ctx context.Context, database database.Plugin) (BatchManager, error) {
	if database == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	readPageSize := config.GetUint(config.BatchManagerReadPageSize)
	bm := &batchManager{
		ctx:                        ctx,
		database:                   database,
		readPageSize:               uint64(readPageSize),
		messagePollTimeout:         time.Duration(config.GetUint(config.BatchManagerReadPollTimeoutMS)) * time.Millisecond,
		startupOffsetRetryAttempts: config.GetInt(config.BatchManagerStartupAttempts),
		dispatchers:                make(map[fftypes.MessageType]*dispatcher),
		newMessages:                make(chan *uuid.UUID, readPageSize),
		retry: &retry.Retry{
			InitialDelay: time.Duration(config.GetUint(config.BatchRetryInitDelayMS)) * time.Millisecond,
			MaximumDelay: time.Duration(config.GetUint(config.BatchRetryMaxDelayMS)) * time.Millisecond,
			Factor:       config.GetFloat64(config.BatchRetryFactor),
		},
	}
	return bm, nil
}

type BatchManager interface {
	RegisterDispatcher(batchType fftypes.MessageType, handler DispatchHandler, batchOptions BatchOptions)
	NewMessages() chan<- *uuid.UUID
	Start() error
	Close()
}

type batchManager struct {
	ctx                        context.Context
	database                   database.Plugin
	dispatchers                map[fftypes.MessageType]*dispatcher
	newMessages                chan *uuid.UUID
	retry                      *retry.Retry
	offset                     int64
	closed                     bool
	readPageSize               uint64
	messagePollTimeout         time.Duration
	startupOffsetRetryAttempts int
}

type DispatchHandler func(context.Context, *fftypes.Batch) error

type BatchOptions struct {
	BatchMaxSize   uint
	BatchTimeout   time.Duration
	DisposeTimeout time.Duration
}

type dispatcher struct {
	handler      DispatchHandler
	mux          sync.Mutex
	processors   map[string]*batchProcessor
	batchOptions BatchOptions
}

func (bm *batchManager) RegisterDispatcher(batchType fftypes.MessageType, handler DispatchHandler, batchOptions BatchOptions) {
	bm.dispatchers[batchType] = &dispatcher{
		handler:      handler,
		batchOptions: batchOptions,
		processors:   make(map[string]*batchProcessor),
	}
}

func (bm *batchManager) Start() error {
	if err := bm.restoreOffset(); err != nil {
		return err
	}
	go bm.messageSequencer()
	return nil
}

func (bm *batchManager) NewMessages() chan<- *uuid.UUID {
	return bm.newMessages
}

func (bm *batchManager) restoreOffset() error {
	offset, err := bm.database.GetOffset(bm.ctx, fftypes.OffsetTypeBatch, fftypes.SystemNamespace, msgBatchOffsetName)
	if err != nil {
		return err
	}
	if offset == nil {
		if err = bm.updateOffset(bm.ctx, false, 0); err != nil {
			return err
		}
	} else {
		bm.offset = offset.Current
	}
	log.L(bm.ctx).Infof("Batch manager restored offset %d", bm.offset)
	return nil
}

func (bm *batchManager) removeProcessor(dispatcher *dispatcher, key string) {
	dispatcher.mux.Lock()
	delete(dispatcher.processors, key)
	dispatcher.mux.Unlock()
}

func (bm *batchManager) getProcessor(batchType fftypes.MessageType, namespace, author string) (*batchProcessor, error) {
	dispatcher, ok := bm.dispatchers[batchType]
	if !ok {
		return nil, i18n.NewError(bm.ctx, i18n.MsgUnregisteredBatchType, batchType)
	}
	dispatcher.mux.Lock()
	key := fmt.Sprintf("%s/%s", namespace, author)
	processor, ok := dispatcher.processors[key]
	if !ok {
		processor = newBatchProcessor(
			bm.ctx, // Background context, not the call context
			&batchProcessorConf{
				BatchOptions:       dispatcher.batchOptions,
				namespace:          namespace,
				author:             author,
				persitence:         bm.database,
				dispatch:           dispatcher.handler,
				processorQuiescing: func() { bm.removeProcessor(dispatcher, key) },
			},
			bm.retry,
		)
		dispatcher.processors[key] = processor
	}
	dispatcher.mux.Unlock()
	return processor, nil
}

func (bm *batchManager) Close() {
	if bm != nil && !bm.closed {
		for _, d := range bm.dispatchers {
			d.mux.Lock()
			for _, p := range d.processors {
				p.close()
			}
			d.mux.Unlock()
		}
		bm.closed = true
		close(bm.newMessages)
	}
	bm = nil
}

func (bm *batchManager) assembleMessageData(ctx context.Context, msg *fftypes.Message) (data []*fftypes.Data, err error) {
	// Load all the data - must all be present for us to send
	for _, dataRef := range msg.Data {
		if dataRef.ID == nil {
			continue
		}
		var d *fftypes.Data
		err = bm.retry.Do(ctx, func(attempt int) (retry bool, err error) {
			d, err = bm.database.GetDataById(ctx, msg.Header.Namespace, dataRef.ID)
			if err != nil {
				// continual retry for persistence error (distinct from not-found)
				return !bm.closed, err
			}
			return false, nil
		})
		if err != nil {
			return nil, err
		}
		if d == nil {
			return nil, i18n.NewError(ctx, i18n.MsgDataNotFound, dataRef.ID)
		}
		data = append(data, d)
	}
	log.L(ctx).Infof("Added broadcast message %s", msg.Header.ID)
	return data, nil
}

func (bm *batchManager) readPage(ctx context.Context) ([]*fftypes.Message, error) {
	var msgs []*fftypes.Message
	err := bm.retry.Do(ctx, func(attempt int) (retry bool, err error) {
		fb := database.MessageQueryFactory.NewFilter(bm.ctx, bm.readPageSize)
		msgs, err = bm.database.GetMessages(bm.ctx, fb.Gt("sequence", bm.offset).Sort("sequence").Limit(bm.readPageSize))
		if err != nil {
			log.L(ctx).Errorf("Failed to retrieve messages: %s", err)
			return !bm.closed, err // Retry indefinitely, until closed (or context cancelled)
		}
		return false, nil
	})
	return msgs, err
}

func (bm *batchManager) messageSequencer() {
	l := log.L(bm.ctx).WithField("role", "batch-msg-sequencer")
	ctx := log.WithLogger(bm.ctx, l)
	l.Debugf("Started batch assembly message sequencer")

	dispatched := make(chan *batchDispatch, bm.readPageSize)

	for !bm.closed {
		// Read messages from the DB - in an error condition we retry until success, or a closed context
		msgs, err := bm.readPage(ctx)
		if err != nil {
			l.Debugf("Exiting: %s", err) // errors logged in readPage
			return
		}
		batchWasFull := (uint64(len(msgs)) == bm.readPageSize)

		if len(msgs) > 0 {
			var dispatchCount int
			for _, msg := range msgs {
				data, err := bm.assembleMessageData(ctx, msg)
				if err != nil {
					l.Errorf("Failed to retrieve message data for %s: %s", msg.Header.ID, err)
					continue
				}

				err = bm.dispatchMessage(ctx, dispatched, msg, data...)
				if err != nil {
					l.Errorf("Failed to dispatch message %s: %s", msg.Header.ID, err)
					continue
				}
				dispatchCount++
			}

			for i := 0; i < dispatchCount; i++ {
				dispatched := <-dispatched
				l.Debugf("Dispatched message %s to batch %s", dispatched.msg.Header.ID, dispatched.batchID)

				if err = bm.updateMessage(ctx, dispatched.msg, dispatched.batchID); err != nil {
					l.Errorf("Closed while attempting to update message %s with batch %s", dispatched.msg.Header.ID, dispatched.batchID)
					break
				}
			}

			if !bm.closed {
				_ = bm.updateOffset(ctx, true, msgs[len(msgs)-1].Sequence)
			}
		}

		// Wait to be woken again
		if !bm.closed && !batchWasFull {
			bm.waitForShoulderTapOrPollTimeout()
		}
	}
}

func (bm *batchManager) waitForShoulderTapOrPollTimeout() {
	l := log.L(bm.ctx)
	timeout := time.NewTimer(bm.messagePollTimeout)
	select {
	case <-timeout.C:
		l.Debugf("Woken after poll timeout")
	case m := <-bm.newMessages:
		l.Debugf("Woken for trigger for message %s", m)
	}
	var drained bool
	for !drained {
		select {
		case m := <-bm.newMessages:
			l.Debugf("Absorbing trigger for message %s", m)
		default:
			drained = true
		}
	}
}

func (bm *batchManager) updateMessage(ctx context.Context, msg *fftypes.Message, batchID *uuid.UUID) (err error) {
	l := log.L(ctx)
	return bm.retry.Do(ctx, func(attempt int) (retry bool, err error) {
		u := database.MessageQueryFactory.NewUpdate(ctx).Set("batchid", batchID)
		err = bm.database.UpdateMessage(ctx, msg.Header.ID, u)
		if err != nil {
			l.Errorf("Batch persist attempt %d failed: %s", attempt, err)
			return !bm.closed, err
		}
		return false, nil
	})
}

func (bm *batchManager) updateOffset(ctx context.Context, infiniteRetry bool, newOffset int64) (err error) {
	l := log.L(ctx)
	return bm.retry.Do(ctx, func(attempt int) (retry bool, err error) {
		bm.offset = newOffset
		offset := &fftypes.Offset{
			Type:      fftypes.OffsetTypeBatch,
			Namespace: fftypes.SystemNamespace,
			Name:      msgBatchOffsetName,
			Current:   bm.offset,
		}
		err = bm.database.UpsertOffset(bm.ctx, offset)
		if err != nil {
			l.Errorf("Batch persist attempt %d failed: %s", attempt, err)
			stillRetrying := infiniteRetry || (attempt <= bm.startupOffsetRetryAttempts)
			return !bm.closed && stillRetrying, err
		}
		l.Infof("Batch manager committed offset %d", newOffset)
		return false, nil
	})
}

func (bm *batchManager) dispatchMessage(ctx context.Context, dispatched chan *batchDispatch, msg *fftypes.Message, data ...*fftypes.Data) error {
	l := log.L(ctx)
	processor, err := bm.getProcessor(msg.Header.Type, msg.Header.Namespace, msg.Header.Author)
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
