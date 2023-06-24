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

package contracts

import (
	"context"
	"fmt"
	"time"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

// txRequest is the dispatched fully validated blockchain transaction, with the previously
// resolved key, and validated signature & inputs ready to generate the operations
// for the blockchain connector.
// Could be a contract deployment, or a transaction invoke.
type txRequest struct {
	deploy           *core.ContractDeployRequest
	invoke           *core.ContractCallRequest
	sentDuplicateErr bool
	result           chan *txResponse
}

type txResponse struct {
	operation *core.Operation
	msgSender syncasync.Sender
	err       error
}

type txWriterBatch struct {
	id             string
	requests       []*txRequest
	timeoutContext context.Context
	timeoutCancel  func()
}

// txWriter manages writing blockchain transactions and operations to the database.
// Does so with optimized multi-insert database operations, on a pool of routines
// that can manage many concurrent API requests efficiently against the DB.
type txWriter struct {
	bgContext   context.Context
	cancelFunc  func()
	database    database.Plugin
	workQueue   chan *txRequest
	workersDone []chan struct{}
	conf        *txWriterConf
	closed      bool
}

type txWriterConf struct {
	workerCount  int
	batchTimeout time.Duration
	maxInserts   int
}

func newTransactionWriter(ctx context.Context, di database.Plugin, conf *txWriterConf) *txWriter {
	if !di.Capabilities().Concurrency {
		log.L(ctx).Infof("Database plugin not configured for concurrency. Batched transaction writing disabled")
		conf.workerCount = 0
	}
	tw := &txWriter{
		conf:     conf,
		database: di,
	}
	tw.bgContext, tw.cancelFunc = context.WithCancel(ctx)
	return tw
}

func (tw *txWriter) dispatch(ctx context.Context, req *txRequest) *txResponse {
	if tw.conf.workerCount == 0 {
		// Workers disabled, execute in-line on the current context
		// (note we provide a slot in the channel to ensure this doesn't block)
		tw.executeBatchOfOne(ctx, req)
	} else {
		// Dispatch to background worker pool
		select {
		case tw.workQueue <- req:
		case <-ctx.Done(): // caller context is cancelled before dispatch
			return &txResponse{
				err: i18n.NewError(ctx, coremsgs.MsgContextCanceled),
			}
		case <-tw.bgContext.Done(): // background context is cancelled before dispatch
			return &txResponse{
				err: i18n.NewError(ctx, coremsgs.MsgContextCanceled),
			}
		}
	}
	return <-req.result
}

// WriteNewMessage is the external interface, which depending on whether we have a non-zero
// worker count will dispatch the work to the pool and wait for it to complete on a background
// transaction, or just run it in-line on the context passed ini.
func (tw *txWriter) writeInvokeTransaction(ctx context.Context, req *core.ContractCallRequest) *txResponse {
	log.L(ctx).Debugf("Writing invoke transaction method=%s idempotencyKey=%s concurrency=%d", req.Method.Name, req.IdempotencyKey, tw.conf.workerCount)
	return tw.dispatch(ctx, &txRequest{
		invoke: req,
		result: make(chan *txResponse, 1), // allocate a slot for the result to avoid blocking
	})
}

// WriteData writes a piece of data independently of a message
func (tw *txWriter) writeDeployTransaction(ctx context.Context, req *core.ContractDeployRequest) *txResponse {
	log.L(ctx).Debugf("Writing deploy contract transaction idempotencyKey=%s concurrency=%d", req.IdempotencyKey, tw.conf.workerCount)
	return tw.dispatch(ctx, &txRequest{
		deploy: req,
		result: make(chan *txResponse, 1), // allocate a slot for the result to avoid blocking
	})
}

func (tw *txWriter) start() {
	if tw.conf.workerCount > 0 {
		tw.workQueue = make(chan *txRequest)
		tw.workersDone = make([]chan struct{}, tw.conf.workerCount)
		for i := 0; i < tw.conf.workerCount; i++ {
			tw.workersDone[i] = make(chan struct{})
			go tw.writerLoop(i)
		}
	}
}

func (tw *txWriter) writerLoop(writerIndex int) {
	defer close(tw.workersDone[writerIndex])

	ctx := log.WithLogField(tw.bgContext, "job", fmt.Sprintf("txwriter_%.3d", writerIndex))
	var batchNumber int
	var batch *txWriterBatch
	for !tw.closed {
		var timeoutContext context.Context
		var timedOut bool
		if batch != nil {
			timeoutContext = batch.timeoutContext
		} else {
			timeoutContext = ctx
		}
		select {
		case work := <-tw.workQueue:
			if batch == nil {
				batchNumber++
				batch = &txWriterBatch{id: fmt.Sprintf("txw_%.3d_%.10d", writerIndex, batchNumber)}
				batch.timeoutContext, batch.timeoutCancel = context.WithTimeout(ctx, tw.conf.batchTimeout)
			}
			batch.requests = append(batch.requests, work)
		case <-timeoutContext.Done():
			timedOut = true
		}

		if batch != nil && (timedOut || (len(batch.requests) >= tw.conf.maxInserts)) {
			batch.timeoutCancel()

			tw.executeBatch(ctx, batch)
			batch = nil
		}
	}
}

func (tw *txWriter) executeBatch(ctx context.Context, batch *txWriterBatch) {
	ctx = log.WithLogField(ctx, "batch", batch.id)
	err := tw.database.RunAsGroup(ctx, func(ctx context.Context) error {
		return tw.processBatch(ctx, batch)
	})
	tw.dispatchBatchErrors(ctx, batch, err)
}

func (tw *txWriter) executeBatchOfOne(ctx context.Context, req *txRequest) {
	batch := &txWriterBatch{requests: []*txRequest{req}}
	err := tw.processBatch(ctx, batch)
	tw.dispatchBatchErrors(ctx, batch, err)
}

func (tw *txWriter) dispatchBatchErrors(ctx context.Context, batch *txWriterBatch, err error) {
	if err != nil {
		// Send back the error as the response
		for _, req := range batch.requests {
			// Unless we already dispatched an idempotency duplicate for that individual TX
			// (that happens before any DB actions commit)
			if !req.sentDuplicateErr {
				req.result <- &txResponse{err: err}
			}
		}
	}
}

func (tw *txWriter) processBatch(ctx context.Context, batch *txWriterBatch) error {

	// Build the array of transactions we need to insert.

}

func (tw *txWriter) close() {
	if !tw.closed {
		tw.closed = true
		tw.cancelFunc()
		for _, workerDone := range tw.workersDone {
			<-workerDone
		}
	}
}
