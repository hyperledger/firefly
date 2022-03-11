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

package data

import (
	"context"
	"time"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type NewMessage struct {
	Message      *fftypes.MessageInOut
	ResolvedData Resolved
}

type Resolved struct {
	AllData       fftypes.DataArray
	NewData       fftypes.DataArray
	DataToPublish []*fftypes.DataAndBlob
}

// writeRequest is a combination of a message and a list of data that is new and needs to be
// inserted into the database.
type writeRequest struct {
	newMessage *fftypes.Message
	newData    fftypes.DataArray
	result     chan error
}

type messageWriterBatch struct {
	messages       []*fftypes.Message
	data           fftypes.DataArray
	listeners      []chan error
	timeoutContext context.Context
	timeoutCancel  func()
}

// messageWriter manages writing messages to the database.
//
// Where supported, it starts background workers to perform batch commits against the database,
// to allow high throughput insertion of messages + data.
//
// Multiple message writers can be started, to combine
// concurrency with batching and tune for maximum throughput.
type messageWriter struct {
	ctx         context.Context
	cancelFunc  func()
	database    database.Plugin
	workQueue   chan *writeRequest
	workersDone []chan struct{}
	conf        *messageWriterConf
	closed      bool
}

type messageWriterConf struct {
	workerCount  int
	batchTimeout time.Duration
	maxInserts   int
}

func newMessageWriter(ctx context.Context, di database.Plugin, conf *messageWriterConf) *messageWriter {
	if !di.Capabilities().Concurrency {
		log.L(ctx).Infof("Database plugin not configured for concurrency. Batched message writing disabled")
		conf.workerCount = 0
	}
	mw := &messageWriter{
		conf:     conf,
		database: di,
	}
	mw.ctx, mw.cancelFunc = context.WithCancel(ctx)
	return mw
}

// WriteNewMessage is the external interface, which depending on whether we have a non-zero
// worker count will dispatch the work to the pool and wait for it to complete on a background
// transaction, or just run it in-line on the context passed ini.
func (mw *messageWriter) WriteNewMessage(ctx context.Context, newMsg *NewMessage) error {
	if mw.conf.workerCount > 0 {
		// Dispatch to background worker
		if newMsg.Message == nil {
			return i18n.NewError(ctx, i18n.MsgNilOrNullObject)
		}
		nmi := &writeRequest{
			newMessage: &newMsg.Message.Message,
			newData:    newMsg.ResolvedData.NewData,
			result:     make(chan error),
		}
		select {
		case mw.workQueue <- nmi:
		case <-mw.ctx.Done():
			return i18n.NewError(ctx, i18n.MsgContextCanceled)
		}
		return <-nmi.result
	}
	// Otherwise do it in-line on this context
	return mw.database.RunAsGroup(ctx, func(ctx context.Context) error {
		return mw.writeMessages(ctx, []*fftypes.Message{&newMsg.Message.Message}, newMsg.ResolvedData.NewData)
	})
}

// WriteData writes a piece of data independently of a message
func (mw *messageWriter) WriteData(ctx context.Context, data *fftypes.Data) error {
	if mw.conf.workerCount > 0 {
		// Dispatch to background worker
		nmi := &writeRequest{
			newData: fftypes.DataArray{data},
			result:  make(chan error),
		}
		select {
		case mw.workQueue <- nmi:
		case <-mw.ctx.Done():
			return i18n.NewError(ctx, i18n.MsgContextCanceled)
		}
		return <-nmi.result
	}
	// Otherwise do it in-line on this context
	return mw.database.UpsertData(ctx, data, database.UpsertOptimizationNew)
}

func (mw *messageWriter) start() {
	if mw.conf.workerCount > 0 {
		mw.workQueue = make(chan *writeRequest)
		mw.workersDone = make([]chan struct{}, mw.conf.workerCount)
		for i := 0; i < mw.conf.workerCount; i++ {
			mw.workersDone[i] = make(chan struct{})
			go mw.writerLoop(i)
		}
	}
}

func (mw *messageWriter) writerLoop(index int) {
	defer close(mw.workersDone[index])

	var batch *messageWriterBatch
	for !mw.closed {
		var timeoutContext context.Context
		var timedOut bool
		if batch != nil {
			timeoutContext = batch.timeoutContext
		} else {
			timeoutContext = mw.ctx
		}
		select {
		case work := <-mw.workQueue:
			if batch == nil {
				batch = &messageWriterBatch{}
				batch.timeoutContext, batch.timeoutCancel = context.WithTimeout(mw.ctx, mw.conf.batchTimeout)
			}
			if work.newMessage != nil {
				batch.messages = append(batch.messages, work.newMessage)
			}
			batch.data = append(batch.data, work.newData...)
			batch.listeners = append(batch.listeners, work.result)
		case <-timeoutContext.Done():
			timedOut = true
		}

		if batch != nil && (timedOut || (len(batch.messages)+len(batch.data) >= mw.conf.maxInserts)) {
			batch.timeoutCancel()
			err := mw.database.RunAsGroup(mw.ctx, func(ctx context.Context) error {
				return mw.writeMessages(ctx, batch.messages, batch.data)
			})
			for _, l := range batch.listeners {
				l <- err
			}
			batch = nil
		}
	}
}

func (mw *messageWriter) writeMessages(ctx context.Context, msgs []*fftypes.Message, data fftypes.DataArray) error {
	if len(data) > 0 {
		if err := mw.database.InsertDataArray(ctx, data); err != nil {
			return err
		}
	}
	if len(msgs) > 0 {
		if err := mw.database.InsertMessages(ctx, msgs); err != nil {
			return err
		}
	}
	return nil
}

func (mw *messageWriter) close() {
	if !mw.closed {
		mw.closed = true
		mw.cancelFunc()
		for _, workerDone := range mw.workersDone {
			<-workerDone
		}
	}
}
