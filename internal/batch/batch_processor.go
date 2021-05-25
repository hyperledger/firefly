// Copyright © 2021 Kaleido, Inc.
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
	"time"

	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/retry"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

type batchWork struct {
	msg        *fftypes.Message
	data       []*fftypes.Data
	dispatched chan *batchDispatch
	abandoned  bool
}

type batchDispatch struct {
	msg     *fftypes.Message
	batchID *fftypes.UUID
}

type batchProcessorConf struct {
	Options
	namespace          string
	author             string
	persitence         database.Plugin
	dispatch           DispatchHandler
	processorQuiescing func()
}

type batchProcessor struct {
	ctx         context.Context
	name        string
	closed      bool
	newWork     chan *batchWork
	persistWork chan *batchWork
	sealBatch   chan bool
	batchSealed chan bool
	retry       *retry.Retry
	conf        *batchProcessorConf
}

func newBatchProcessor(ctx context.Context, conf *batchProcessorConf, retry *retry.Retry) *batchProcessor {
	bp := &batchProcessor{
		ctx:         log.WithLogField(ctx, "role", fmt.Sprintf("batchproc-%s:%s", conf.namespace, conf.author)),
		name:        fmt.Sprintf("%s:%s", conf.namespace, conf.author),
		newWork:     make(chan *batchWork),
		persistWork: make(chan *batchWork, conf.BatchMaxSize),
		sealBatch:   make(chan bool),
		batchSealed: make(chan bool),
		retry:       retry,
		conf:        conf,
	}
	go bp.assemblyLoop()
	go bp.persistenceLoop()
	return bp
}

// The assemblyLoop accepts work into the pipe as quickly as possible.
// It dispatches work asynchronously to the peristenceLoop, which is responsible for
// calling back each piece of work once persisted into a batch
// (doesn't wait until that batch is sealed/dispatched).
// The assemblyLoop seals batches when they are full, or timeout.
func (bp *batchProcessor) assemblyLoop() {
	defer bp.close()
	defer close(bp.sealBatch) // close persitenceLoop when we exit
	l := log.L(bp.ctx)
	var batchSize uint
	var lastBatchSealed = time.Now()
	var quiescing bool
	for {
		// We timeout waiting at the point we think we're ready for disposal,
		// unless we've started a batch in which case we wait for what's left
		// of the batch timeout
		timeToWait := bp.conf.DisposeTimeout
		if quiescing {
			timeToWait = 100 * time.Millisecond
		} else if batchSize > 0 {
			timeToWait = bp.conf.BatchTimeout - time.Since(lastBatchSealed)
		}
		timeout := time.NewTimer(timeToWait)

		// Wait for work, the timeout, or close
		var timedOut, closed bool
		select {
		case <-timeout.C:
			timedOut = true
		case work, ok := <-bp.newWork:
			if ok && !work.abandoned {
				batchSize++
				bp.persistWork <- work
			} else {
				closed = true
			}
		}

		// Don't include the sealing time in the duration
		batchFull := batchSize >= bp.conf.BatchMaxSize
		l.Debugf("Assembly batch loop: Size=%d Full=%t", batchSize, batchFull)

		batchDuration := time.Since(lastBatchSealed)
		if quiescing && batchSize == 0 {
			l.Debugf("Batch assembler disposed after %.2fs of inactivity", float64(batchDuration)/float64(time.Second))
			return
		}

		if closed || batchDuration > bp.conf.DisposeTimeout {
			bp.conf.processorQuiescing()
			quiescing = true
		}

		if (quiescing || timedOut || batchFull) && batchSize > 0 {
			bp.sealBatch <- true
			<-bp.batchSealed
			l.Debugf("Assembly batch sealed")
			lastBatchSealed = time.Now()
			batchSize = 0
		}

	}
}

func (bp *batchProcessor) createOrAddToBatch(batch *fftypes.Batch, newWork []*batchWork, seal bool) *fftypes.Batch {
	l := log.L(bp.ctx)
	if batch == nil {
		batchID := fftypes.NewUUID()
		l.Debugf("New batch %s", batchID)
		batch = &fftypes.Batch{
			ID:        batchID,
			Namespace: bp.conf.namespace,
			Author:    bp.conf.author,
			Payload:   fftypes.BatchPayload{},
			Created:   fftypes.Now(),
		}
	}
	for _, w := range newWork {
		if w.msg != nil {
			batch.Payload.Messages = append(batch.Payload.Messages, w.msg)
		}
		batch.Payload.Data = append(batch.Payload.Data, w.data...)
	}
	if seal {
		// Generate a new Transaction reference, which will be used to record status of the associated transaction as it happens
		batch.Payload.TX = fftypes.TransactionRef{
			Type: fftypes.TransactionTypePin,
			ID:   fftypes.NewUUID(),
		}
		batch.Hash = batch.Payload.Hash()
		l.Debugf("Batch %s sealed. Hash=%s", batch.ID, batch.Hash)
	}
	return batch
}

func (bp *batchProcessor) dispatchBatch(batch *fftypes.Batch) {
	// Call the dispatcher to do the heavy lifting - will only exit if we're closed
	_ = bp.retry.Do(bp.ctx, "batch dispatch", func(attempt int) (retry bool, err error) {
		err = bp.conf.dispatch(bp.ctx, batch)
		if err != nil {
			return !bp.closed, err
		}
		return false, nil
	})
}

func (bp *batchProcessor) persistBatch(batch *fftypes.Batch, seal bool) (err error) {
	return bp.retry.Do(bp.ctx, "batch persist", func(attempt int) (retry bool, err error) {
		err = bp.conf.persitence.UpsertBatch(bp.ctx, batch, true, seal /* we set the hash as it seals */)
		if err != nil {
			return !bp.closed, err
		}
		return false, nil
	})
}

func (bp *batchProcessor) persistenceLoop() {
	defer close(bp.batchSealed)
	l := log.L(bp.ctx)
	var currentBatch *fftypes.Batch
	for !bp.closed {
		var seal bool
		newWork := make([]*batchWork, 0, bp.conf.BatchMaxSize)

		// Block waiting for work, or a batch sealing request
		select {
		case w := <-bp.persistWork:
			newWork = append(newWork, w)
		case <-bp.sealBatch:
			seal = true
		}

		// Drain everything currently in the pipe waiting for dispatch
		// This means we batch the writing to the database, which has to happen before
		// we can callback the work with a persisted batch ID.
		// We drain both the message queue, and the seal, because there's no point
		// going round the loop (persisting twice) if the batch has just filled
		var drained bool
		for !drained {
			select {
			case _, ok := <-bp.sealBatch:
				seal = true
				if !ok {
					return // Closed by termination of assemblyLoop
				}
			case w := <-bp.persistWork:
				newWork = append(newWork, w)
			default:
				drained = true
			}
		}
		currentBatch = bp.createOrAddToBatch(currentBatch, newWork, seal)
		l.Debugf("Adding %d entries to batch %s. Seal=%t", len(newWork), currentBatch.ID, seal)

		// Persist the batch - indefinite retry (unless we close, or context is cancelled)
		if err := bp.persistBatch(currentBatch, seal); err != nil {
			return
		}

		// Inform all the work in this batch of the batch they have been persisted
		// into. At this point they can carry on processing, because we won't lose
		// the work - it's tracked in a batch ready to go
		for _, w := range newWork {
			w.dispatched <- &batchDispatch{
				w.msg,
				currentBatch.ID,
			}
		}

		if seal {
			// At this point the batch is sealed, and the assember can start
			// queing up the next batch. We only let them get one batch ahead
			// (due to the size of the channel being the maxBatchSize) before
			// they start blocking waiting for us to complete database of
			// the current batch.
			bp.batchSealed <- true

			// Synchronously dispatch the batch. Must be last thing we do in the loop, as we
			// will break out of the retry in the case that we close
			bp.dispatchBatch(currentBatch)

			// Move onto the next batch
			currentBatch = nil
		}

	}
}

func (bp *batchProcessor) close() {
	if !bp.closed {
		close(bp.newWork)
		bp.closed = true
	}
}

func (bp *batchProcessor) waitClosed() {
	<-bp.sealBatch
	<-bp.batchSealed
}
