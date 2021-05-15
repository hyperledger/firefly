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
	"time"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/retry"
)

type batchWork struct {
	msg        *fftypes.Message
	data       []*fftypes.Data
	dispatched chan *batchDispatch
	abandoned  bool
}

type batchDispatch struct {
	msg     *fftypes.Message
	batchID *uuid.UUID
}

type batchProcessorConf struct {
	BatchOptions
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
	a := &batchProcessor{
		ctx:         log.WithLogField(ctx, "author", conf.author),
		name:        fmt.Sprintf("%s:%s", conf.namespace, conf.author),
		newWork:     make(chan *batchWork),
		persistWork: make(chan *batchWork, conf.BatchMaxSize),
		sealBatch:   make(chan bool),
		batchSealed: make(chan bool),
		retry:       retry,
		conf:        conf,
	}
	go a.assemblyLoop()
	go a.persistenceLoop()
	return a
}

// The assemblyLoop accepts work into the pipe as quickly as possible.
// It dispatches work asynchronously to the peristenceLoop, which is responsible for
// calling back each piece of work once persisted into a batch
// (doesn't wait until that batch is sealed/dispatched).
// The assemblyLoop seals batches when they are full, or timeout.
func (a *batchProcessor) assemblyLoop() {
	defer a.close()
	defer close(a.sealBatch) // close persitenceLoop when we exit
	l := log.L(a.ctx)
	var batchSize uint = 0
	var lastBatchSealed = time.Now()
	var quiescing bool
	for {
		// We timeout waiting at the point we think we're ready for disposal,
		// unless we've started a batch in which case we wait for what's left
		// of the batch timeout
		timeToWait := a.conf.DisposeTimeout
		if quiescing {
			timeToWait = 100 * time.Millisecond
		} else if batchSize > 0 {
			timeToWait = a.conf.BatchTimeout - time.Since(lastBatchSealed)
		}
		timeout := time.NewTimer(timeToWait)

		// Wait for work, the timeout, or close
		var timedOut, closed bool
		select {
		case <-timeout.C:
			timedOut = true
		case work, ok := <-a.newWork:
			if ok && !work.abandoned {
				batchSize++
				a.persistWork <- work
			} else {
				closed = true
			}
		}

		// Don't include the sealing time in the duration
		batchFull := batchSize >= a.conf.BatchMaxSize
		l.Debugf("Assembly batch loop: Size=%d Full=%t", batchSize, batchFull)

		batchDuration := time.Since(lastBatchSealed)
		if quiescing && batchSize == 0 {
			l.Debugf("Batch assembler disposed after %.2fs of inactivity", float64(batchDuration)/float64(time.Second))
			return
		}

		if closed || batchDuration > a.conf.DisposeTimeout {
			a.conf.processorQuiescing()
			quiescing = true
		}

		if (quiescing || timedOut || batchFull) && batchSize > 0 {
			a.sealBatch <- true
			<-a.batchSealed
			l.Debugf("Assembly batch sealed")
			lastBatchSealed = time.Now()
			batchSize = 0
		}

	}
}

func (a *batchProcessor) createOrAddToBatch(ctx context.Context, batch *fftypes.Batch, newWork []*batchWork, seal bool) *fftypes.Batch {
	l := log.L(ctx)
	if batch == nil {
		batchID := uuid.New()
		l.Debugf("New batch %s", batchID)
		batch = &fftypes.Batch{
			ID:        &batchID,
			Namespace: a.conf.namespace,
			Author:    a.conf.author,
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

func (a *batchProcessor) dispatchBatch(ctx context.Context, batch *fftypes.Batch) {
	l := log.L(ctx)
	// Call the dispatcher to do the heavy lifting - will only exit if we're closed
	_ = a.retry.Do(ctx, func(attempt int) (retry bool, err error) {
		err = a.conf.dispatch(a.ctx, batch)
		if err != nil {
			l.Errorf("Batch dispatch attempt %d failed: %s", attempt, err)
			return !a.closed, err
		}
		return false, nil
	})
}

func (a *batchProcessor) persistBatch(ctx context.Context, batch *fftypes.Batch, seal bool) (err error) {
	l := log.L(ctx)
	return a.retry.Do(ctx, func(attempt int) (retry bool, err error) {
		err = a.conf.persitence.UpsertBatch(ctx, batch, seal /* we set the hash as it seals */)
		if err != nil {
			l.Errorf("Batch persist attempt %d failed: %s", attempt, err)
			return !a.closed, err
		}
		return false, nil
	})
}

func (a *batchProcessor) persistenceLoop() {
	defer close(a.batchSealed)
	l := log.L(a.ctx).WithField("role", fmt.Sprintf("persist-%s", a.name))
	ctx := log.WithLogger(a.ctx, l)
	var currentBatch *fftypes.Batch
	for !a.closed {
		var seal bool
		newWork := make([]*batchWork, 0, a.conf.BatchMaxSize)

		// Block waiting for work, or a batch sealing request
		select {
		case w := <-a.persistWork:
			newWork = append(newWork, w)
		case <-a.sealBatch:
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
			case _, ok := <-a.sealBatch:
				seal = true
				if !ok {
					return // Closed by termination of assemblyLoop
				}
			case w := <-a.persistWork:
				newWork = append(newWork, w)
			default:
				drained = true
			}
		}
		currentBatch = a.createOrAddToBatch(ctx, currentBatch, newWork, seal)
		l.Debugf("Adding %d entries to batch %s. Seal=%t", len(newWork), currentBatch.ID, seal)

		// Persist the batch - indefinite retry (unless we close, or context is cancelled)
		if err := a.persistBatch(ctx, currentBatch, seal); err != nil {
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
			a.batchSealed <- true

			// Synchronously dispatch the batch. Must be last thing we do in the loop, as we
			// will break out of the retry in the case that we close
			a.dispatchBatch(ctx, currentBatch)

			// Move onto the next batch
			currentBatch = nil
		}

	}
}

func (a *batchProcessor) close() {
	if !a.closed {
		close(a.newWork)
		a.closed = true
	}
}
