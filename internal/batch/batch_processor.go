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
	"crypto/sha256"
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
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
	identity           fftypes.Identity
	group              *fftypes.Bytes32
	dispatch           DispatchHandler
	processorQuiescing func()
}

type batchProcessor struct {
	ctx         context.Context
	ni          sysmessaging.LocalNodeInfo
	database    database.Plugin
	name        string
	cancelCtx   func()
	closed      bool
	newWork     chan *batchWork
	persistWork chan *batchWork
	sealBatch   chan bool
	batchSealed chan bool
	retry       *retry.Retry
	conf        *batchProcessorConf
}

const batchSizeEstimateBase = int64(512)

func newBatchProcessor(ctx context.Context, ni sysmessaging.LocalNodeInfo, di database.Plugin, conf *batchProcessorConf, retry *retry.Retry) *batchProcessor {
	pCtx := log.WithLogField(ctx, "role", fmt.Sprintf("batchproc-%s:%s:%s", conf.namespace, conf.identity.Author, conf.identity.Key))
	pCtx, cancelCtx := context.WithCancel(pCtx)
	bp := &batchProcessor{
		ctx:         pCtx,
		cancelCtx:   cancelCtx,
		ni:          ni,
		database:    di,
		name:        fmt.Sprintf("%s:%s:%s", conf.namespace, conf.identity.Author, conf.identity.Key),
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

func (bw *batchWork) estimateSize() int64 {
	sizeEstimate := bw.msg.EstimateSize(false /* we calculate data size separately, as we have the full data objects */)
	for _, d := range bw.data {
		sizeEstimate += d.EstimateSize()
	}
	return sizeEstimate
}

// The assemblyLoop accepts work into the pipe as quickly as possible.
// It dispatches work asynchronously to the persistenceLoop, which is responsible for
// calling back each piece of work once persisted into a batch
// (doesn't wait until that batch is sealed/dispatched).
// The assemblyLoop seals batches when they are full, or timeout.
func (bp *batchProcessor) assemblyLoop() {
	defer bp.close()
	defer close(bp.sealBatch) // close persitenceLoop when we exit
	l := log.L(bp.ctx)
	var batchSize uint
	var batchPayloadEstimate = batchSizeEstimateBase
	var lastBatchSealed = time.Now()
	var quiescing bool
	var overflowedWork *batchWork
	for {
		var timedOut, closed bool
		if overflowedWork != nil {
			// We overflowed the size cap when we took this message out the newWork
			// queue last time round the lop
			bp.persistWork <- overflowedWork
			batchSize++
			batchPayloadEstimate += overflowedWork.estimateSize()
			overflowedWork = nil
		} else {
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
			select {
			case <-timeout.C:
				timedOut = true
			case work, ok := <-bp.newWork:
				if ok && !work.abandoned {
					workSize := work.estimateSize()
					if batchSize > 0 && batchPayloadEstimate+workSize > bp.conf.BatchMaxBytes {
						overflowedWork = work
					} else {
						batchSize++
						batchPayloadEstimate += workSize
						bp.persistWork <- work
					}
				} else {
					closed = true
				}
			}

		}

		// Don't include the sealing time in the duration
		batchFull := overflowedWork != nil || batchSize >= bp.conf.BatchMaxSize
		l.Debugf("Assembly batch loop: Size=%d Full=%t Bytes=%.2fkb (est) Overflow=%t", batchSize, batchFull, float64(batchPayloadEstimate)/1024, overflowedWork != nil)

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
			batchPayloadEstimate = batchSizeEstimateBase
		}

	}
}

func (bp *batchProcessor) createOrAddToBatch(batch *fftypes.Batch, newWork []*batchWork) *fftypes.Batch {
	l := log.L(bp.ctx)
	if batch == nil {
		batchID := fftypes.NewUUID()
		l.Debugf("New batch %s", batchID)
		batch = &fftypes.Batch{
			ID:        batchID,
			Namespace: bp.conf.namespace,
			Identity:  bp.conf.identity,
			Group:     bp.conf.group,
			Payload:   fftypes.BatchPayload{},
			Created:   fftypes.Now(),
			Node:      bp.ni.GetNodeUUID(bp.ctx),
		}
	}
	for _, w := range newWork {
		if w.msg != nil {
			w.msg.BatchID = batch.ID
			w.msg.State = "" // state should always be set by receivers when loading the batch
			batch.Payload.Messages = append(batch.Payload.Messages, w.msg)
		}
		batch.Payload.Data = append(batch.Payload.Data, w.data...)
	}
	return batch
}

func (bp *batchProcessor) maskContext(ctx context.Context, msg *fftypes.Message, topic string) (contextOrPin *fftypes.Bytes32, err error) {

	hashBuilder := sha256.New()
	hashBuilder.Write([]byte(topic))

	// For broadcast we do not need to mask the context, which is just the hash
	// of the topic. There would be no way to unmask it if we did, because we don't have
	// the full list of senders to know what their next hashes should be.
	if msg.Header.Group == nil {
		return fftypes.HashResult(hashBuilder), nil
	}

	// For private groups, we need to make the topic specific to the group (which is
	// a salt for the hash as it is not on chain)
	hashBuilder.Write((*msg.Header.Group)[:])

	// The combination of the topic and group is the context
	contextHash := fftypes.HashResult(hashBuilder)

	// Get the next nonce for this context - we're the authority in the nextwork on this,
	// as we are the sender.
	gc := &fftypes.Nonce{
		Context: contextHash,
		Group:   msg.Header.Group,
		Topic:   topic,
	}
	err = bp.database.UpsertNonceNext(ctx, gc)
	if err != nil {
		return nil, err
	}

	// Now combine our sending identity, and this nonce, to produce the hash that should
	// be expected by all members of the group as the next nonce from us on this topic.
	// Note we use our identity DID (not signing key) for this.
	hashBuilder.Write([]byte(msg.Header.Author))
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, uint64(gc.Nonce))
	hashBuilder.Write(nonceBytes)

	return fftypes.HashResult(hashBuilder), err
}

func (bp *batchProcessor) maskContexts(ctx context.Context, batch *fftypes.Batch) ([]*fftypes.Bytes32, error) {
	// Calculate the sequence hashes
	contextsOrPins := make([]*fftypes.Bytes32, 0, len(batch.Payload.Messages))
	for _, msg := range batch.Payload.Messages {
		for _, topic := range msg.Header.Topics {
			contextOrPin, err := bp.maskContext(ctx, msg, topic)
			if err != nil {
				return nil, err
			}
			contextsOrPins = append(contextsOrPins, contextOrPin)
			if msg.Header.Group != nil {
				msg.Pins = append(msg.Pins, contextOrPin.String())
			}
		}
	}
	return contextsOrPins, nil
}

func (bp *batchProcessor) dispatchBatch(batch *fftypes.Batch, pins []*fftypes.Bytes32) {
	// Call the dispatcher to do the heavy lifting - will only exit if we're closed
	_ = bp.retry.Do(bp.ctx, "batch dispatch", func(attempt int) (retry bool, err error) {
		err = bp.conf.dispatch(bp.ctx, batch, pins)
		if err != nil {
			return !bp.closed, err
		}
		return false, nil
	})
}

func (bp *batchProcessor) persistBatch(batch *fftypes.Batch, newWork []*batchWork, seal bool) (contexts []*fftypes.Bytes32, err error) {
	err = bp.retry.Do(bp.ctx, "batch persist", func(attempt int) (retry bool, err error) {
		err = bp.database.RunAsGroup(bp.ctx, func(ctx context.Context) (err error) {
			// Update all the messages in the batch with the batch ID
			if len(newWork) > 0 {
				msgIDs := make([]driver.Value, 0, len(newWork))
				for _, w := range newWork {
					if w.msg != nil {
						msgIDs = append(msgIDs, w.msg.Header.ID)
					}
				}
				filter := database.MessageQueryFactory.NewFilter(ctx).In("id", msgIDs)
				update := database.MessageQueryFactory.NewUpdate(ctx).
					Set("batch", batch.ID).
					Set("group", batch.Group)
				err = bp.database.UpdateMessages(ctx, filter, update)
			}
			if err == nil && seal {
				// Generate a new Transaction reference, which will be used to record status of the associated transaction as it happens
				batch.Payload.TX = fftypes.TransactionRef{
					Type: fftypes.TransactionTypeBatchPin,
					ID:   fftypes.NewUUID(),
				}
				contexts, err = bp.maskContexts(ctx, batch)
				batch.Hash = batch.Payload.Hash()
				log.L(ctx).Debugf("Batch %s sealed. Hash=%s", batch.ID, batch.Hash)
			}
			if err == nil {
				// Persist the batch itself
				err = bp.database.UpsertBatch(ctx, batch, seal /* we set the hash as it seals */)
			}
			return err
		})
		if err != nil {
			return !bp.closed, err
		}
		return false, nil
	})
	return contexts, err
}

func (bp *batchProcessor) persistenceLoop() {
	defer close(bp.batchSealed)
	l := log.L(bp.ctx)
	var currentBatch *fftypes.Batch
	var batchSize = 0
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

		batchSize += len(newWork)
		currentBatch = bp.createOrAddToBatch(currentBatch, newWork)
		l.Debugf("Adding %d entries to batch %s. Size=%d Seal=%t", len(newWork), currentBatch.ID, batchSize, seal)

		// Persist the batch - indefinite retry (unless we close, or context is cancelled)
		contexts, err := bp.persistBatch(currentBatch, newWork, seal)
		if err != nil {
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
			bp.dispatchBatch(currentBatch, contexts)

			// Move onto the next batch
			currentBatch = nil
			batchSize = 0
		}

	}
}

func (bp *batchProcessor) close() {
	if !bp.closed {
		// We don't cancel the context here, as we use close during quiesce and don't want the
		// persistence loop to have its context cancelled, and fail to perform DB operations
		close(bp.newWork)
		bp.closed = true
	}
}

func (bp *batchProcessor) waitClosed() {
	<-bp.sealBatch
	<-bp.batchSealed
}
