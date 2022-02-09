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

package events

import (
	"context"
	"crypto/sha256"
	"database/sql/driver"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/definitions"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

const (
	aggregatorOffsetName = "ff_aggregator"
)

type aggregator struct {
	ctx             context.Context
	database        database.Plugin
	definitions     definitions.DefinitionHandlers
	data            data.Manager
	eventPoller     *eventPoller
	newPins         chan int64
	offchainBatches chan *fftypes.UUID
	queuedRewinds   chan *fftypes.UUID
	retry           *retry.Retry
	metricsEnabled  bool
	metrics         metrics.Manager
}

func newAggregator(ctx context.Context, di database.Plugin, sh definitions.DefinitionHandlers, dm data.Manager, en *eventNotifier, mm metrics.Manager) *aggregator {
	batchSize := config.GetInt(config.EventAggregatorBatchSize)
	ag := &aggregator{
		ctx:             log.WithLogField(ctx, "role", "aggregator"),
		database:        di,
		definitions:     sh,
		data:            dm,
		newPins:         make(chan int64),
		offchainBatches: make(chan *fftypes.UUID, 1), // hops to queuedRewinds with a shouldertab on the event poller
		queuedRewinds:   make(chan *fftypes.UUID, batchSize),
		metricsEnabled:  config.GetBool(config.MetricsEnabled),
		metrics:         mm,
	}
	firstEvent := fftypes.SubOptsFirstEvent(config.GetString(config.EventAggregatorFirstEvent))
	ag.eventPoller = newEventPoller(ctx, di, en, &eventPollerConf{
		eventBatchSize:             batchSize,
		eventBatchTimeout:          config.GetDuration(config.EventAggregatorBatchTimeout),
		eventPollTimeout:           config.GetDuration(config.EventAggregatorPollTimeout),
		startupOffsetRetryAttempts: config.GetInt(config.OrchestratorStartupAttempts),
		retry: retry.Retry{
			InitialDelay: config.GetDuration(config.EventAggregatorRetryInitDelay),
			MaximumDelay: config.GetDuration(config.EventAggregatorRetryMaxDelay),
			Factor:       config.GetFloat64(config.EventAggregatorRetryFactor),
		},
		firstEvent:       &firstEvent,
		namespace:        fftypes.SystemNamespace,
		offsetType:       fftypes.OffsetTypeAggregator,
		offsetName:       aggregatorOffsetName,
		newEventsHandler: ag.processPinsEventsHandler,
		getItems:         ag.getPins,
		queryFactory:     database.PinQueryFactory,
		addCriteria: func(af database.AndFilter) database.AndFilter {
			return af.Condition(af.Builder().Eq("dispatched", false))
		},
		maybeRewind: ag.rewindOffchainBatches,
	})
	ag.retry = &ag.eventPoller.conf.retry
	return ag
}

func (ag *aggregator) start() {
	go ag.offchainListener()
	ag.eventPoller.start()
}

func (ag *aggregator) offchainListener() {
	for {
		select {
		case uuid := <-ag.offchainBatches:
			ag.queuedRewinds <- uuid
			ag.eventPoller.shoulderTap()
		case <-ag.ctx.Done():
			return
		}
	}
}

func (ag *aggregator) rewindOffchainBatches() (rewind bool, offset int64) {
	// Retry idefinitely for database errors (until the context closes)
	_ = ag.retry.Do(ag.ctx, "check for off-chain batch deliveries", func(attempt int) (retry bool, err error) {
		var batchIDs []driver.Value
		draining := true
		for draining {
			select {
			case batchID := <-ag.queuedRewinds:
				batchIDs = append(batchIDs, batchID)
			default:
				draining = false
			}
		}
		if len(batchIDs) > 0 {
			fb := database.PinQueryFactory.NewFilter(ag.ctx)
			filter := fb.And(
				fb.In("batch", batchIDs),
				fb.Eq("dispatched", false),
			).Sort("sequence").Limit(1) // only need the one oldest sequence
			sequences, _, err := ag.database.GetPins(ag.ctx, filter)
			if err != nil {
				return true, err
			}
			if len(sequences) > 0 {
				rewind = true
				offset = sequences[0].Sequence - 1
				log.L(ag.ctx).Debugf("Rewinding for off-chain data arrival. New local pin sequence %d", offset)
			}
		}
		return false, nil
	})
	return rewind, offset
}

func (ag *aggregator) processWithBatchState(callback func(ctx context.Context, state *batchState) error) error {
	state := newBatchState(ag)

	err := ag.database.RunAsGroup(ag.ctx, func(ctx context.Context) (err error) {
		if err := callback(ctx, state); err != nil {
			return err
		}
		if len(state.PreFinalize) == 0 {
			return state.RunFinalize(ctx)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(state.PreFinalize) == 0 {
		return err
	}

	if err := state.RunPreFinalize(ag.ctx); err != nil {
		return err
	}
	return ag.database.RunAsGroup(ag.ctx, func(ctx context.Context) error {
		return state.RunFinalize(ctx)
	})
}

func (ag *aggregator) processPinsEventsHandler(items []fftypes.LocallySequenced) (repoll bool, err error) {
	pins := make([]*fftypes.Pin, len(items))
	for i, item := range items {
		pins[i] = item.(*fftypes.Pin)
	}

	return false, ag.processWithBatchState(func(ctx context.Context, state *batchState) error {
		return ag.processPins(ctx, pins, state)
	})
}

func (ag *aggregator) getPins(ctx context.Context, filter database.Filter) ([]fftypes.LocallySequenced, error) {
	pins, _, err := ag.database.GetPins(ctx, filter)
	ls := make([]fftypes.LocallySequenced, len(pins))
	for i, p := range pins {
		ls[i] = p
	}
	return ls, err
}

func (ag *aggregator) processPins(ctx context.Context, pins []*fftypes.Pin, state *batchState) (err error) {
	l := log.L(ctx)

	// Keep a batch cache for this list of pins
	var batch *fftypes.Batch
	// As messages can have multiple topics, we need to avoid processing the message twice in the same poll loop.
	// We must check all the contexts in the message, and mark them dispatched together.
	dupMsgCheck := make(map[fftypes.UUID]bool)
	for _, pin := range pins {
		l.Debugf("Aggregating pin %.10d batch=%s hash=%s masked=%t", pin.Sequence, pin.Batch, pin.Hash, pin.Masked)

		if batch == nil || *batch.ID != *pin.Batch {
			batch, err = ag.database.GetBatchByID(ctx, pin.Batch)
			if err != nil {
				return err
			}
			if batch == nil {
				l.Debugf("Batch %s not available - pin %s is parked", pin.Batch, pin.Hash)
				continue
			}
		}

		// Extract the message from the batch - where the index is of a topic within a message
		var msg *fftypes.Message
		var i int64 = -1
		var msgBaseIndex int64
		for iM := 0; i < pin.Index && iM < len(batch.Payload.Messages); iM++ {
			msg = batch.Payload.Messages[iM]
			msgBaseIndex = i
			for iT := 0; i < pin.Index && iT < len(msg.Header.Topics); iT++ {
				i++
			}
		}

		if i < pin.Index {
			l.Errorf("Batch %s does not have message-topic index %d - pin %s is invalid", pin.Batch, pin.Index, pin.Hash)
			continue
		}
		l.Tracef("Batch %s message %d: %+v", batch.ID, pin.Index, msg)
		if msg == nil || msg.Header.ID == nil {
			l.Errorf("null message entry %d in batch '%s'", pin.Index, batch.ID)
			continue
		}
		if dupMsgCheck[*msg.Header.ID] {
			continue
		}
		dupMsgCheck[*msg.Header.ID] = true

		// Attempt to process the message (only returns errors for database persistence issues)
		err := ag.processMessage(ctx, batch, pin, msgBaseIndex, msg, state)
		if err != nil {
			return err
		}
	}

	err = ag.eventPoller.commitOffset(ctx, pins[len(pins)-1].Sequence)
	return err
}

func (ag *aggregator) processMessage(ctx context.Context, batch *fftypes.Batch, pin *fftypes.Pin, msgBaseIndex int64, msg *fftypes.Message, state *batchState) (err error) {
	// Check if it's ready to be processed
	unmaskedContexts := make([]*fftypes.Bytes32, 0, len(msg.Header.Topics))
	nextPins := make([]*nextPinState, 0, len(msg.Header.Topics))
	if pin.Masked {
		// Private messages have one or more masked "pin" hashes that allow us to work
		// out if it's the next message in the sequence, given the previous messages
		if msg.Header.Group == nil || len(msg.Pins) == 0 || len(msg.Header.Topics) != len(msg.Pins) {
			log.L(ctx).Errorf("Message '%s' in batch '%s' has invalid pin data pins=%v topics=%v", msg.Header.ID, batch.ID, msg.Pins, msg.Header.Topics)
			return nil
		}
		for i, pinStr := range msg.Pins {
			var msgContext fftypes.Bytes32
			err := msgContext.UnmarshalText([]byte(pinStr))
			if err != nil {
				log.L(ctx).Errorf("Message '%s' in batch '%s' has invalid pin at index %d: '%s'", msg.Header.ID, batch.ID, i, pinStr)
				return nil
			}
			nextPin, err := state.CheckMaskedContextReady(ctx, msg, msg.Header.Topics[i], pin.Sequence, &msgContext)
			if err != nil || nextPin == nil {
				return err
			}
			nextPins = append(nextPins, nextPin)
		}
	} else {
		for i, topic := range msg.Header.Topics {
			h := sha256.New()
			h.Write([]byte(topic))
			msgContext := fftypes.HashResult(h)
			unmaskedContexts = append(unmaskedContexts, msgContext)
			ready, err := state.CheckUnmaskedContextReady(ctx, *msgContext, msg, msg.Header.Topics[i], pin.Sequence)
			if err != nil || !ready {
				return err
			}
		}

	}

	dispatched, err := ag.attemptMessageDispatch(ctx, msg, state)
	if err != nil {
		return err
	}

	// Mark all message pins dispatched true/false
	// - dispatched=true: we need to write them dispatched in the DB at the end of the batch, and increment all nextPins
	// - dispatched=false: we need to prevent dispatch of any subsequent messages on the same topic in the batch
	if dispatched {
		for _, np := range nextPins {
			np.IncrementNextPin(ctx)
		}
		state.MarkMessageDispatched(ctx, batch.ID, msg, msgBaseIndex)
	} else {
		for _, unmaskedContext := range unmaskedContexts {
			state.SetContextBlockedBy(ctx, *unmaskedContext, pin.Sequence)
		}
	}

	return nil
}

func (ag *aggregator) attemptMessageDispatch(ctx context.Context, msg *fftypes.Message, state *batchState) (bool, error) {

	// If we don't find all the data, then we don't dispatch
	data, foundAll, err := ag.data.GetMessageData(ctx, msg, true)
	if err != nil || !foundAll {
		return false, err
	}

	// Verify we have all the blobs for the data
	if resolved, err := ag.resolveBlobs(ctx, data); err != nil || !resolved {
		return false, err
	}

	// For transfers, verify the transfer has come through
	if msg.Header.Type == fftypes.MessageTypeTransferBroadcast || msg.Header.Type == fftypes.MessageTypeTransferPrivate {
		fb := database.TokenTransferQueryFactory.NewFilter(ctx)
		filter := fb.And(
			fb.Eq("message", msg.Header.ID),
		)
		if transfers, _, err := ag.database.GetTokenTransfers(ctx, filter); err != nil || len(transfers) == 0 {
			log.L(ctx).Debugf("Transfer for message %s not yet available", msg.Header.ID)
			return false, err
		} else if !msg.Hash.Equals(transfers[0].MessageHash) {
			log.L(ctx).Errorf("Message hash %s does not match hash recorded in transfer: %s", msg.Hash, transfers[0].MessageHash)
			return false, nil
		}
	}

	// Validate the message data
	valid := true
	switch {
	case msg.Header.Type == fftypes.MessageTypeDefinition:
		// We handle definition events in-line on the aggregator, as it would be confusing for apps to be
		// dispatched subsequent events before we have processed the definition events they depend on.
		msgAction, batchAction, err := ag.definitions.HandleDefinitionBroadcast(ctx, msg, data)
		if msgAction == definitions.ActionRetry {
			return false, err
		}
		if batchAction != nil {
			state.AddPreFinalize(batchAction.PreFinalize)
			state.AddFinalize(batchAction.Finalize)
		}
		if msgAction == definitions.ActionWait {
			return false, nil
		}
		valid = msgAction == definitions.ActionConfirm

	case msg.Header.Type == fftypes.MessageTypeGroupInit:
		// Already handled as part of resolving the context - do nothing.

	case len(msg.Data) > 0:
		valid, err = ag.data.ValidateAll(ctx, data)
		if err != nil {
			return false, err
		}
	}

	status := fftypes.MessageStateConfirmed
	eventType := fftypes.EventTypeMessageConfirmed
	if !valid {
		status = fftypes.MessageStateRejected
		eventType = fftypes.EventTypeMessageRejected
	}

	state.AddFinalize(func(ctx context.Context) error {
		// This message is now confirmed
		setConfirmed := database.MessageQueryFactory.NewUpdate(ctx).
			Set("confirmed", fftypes.Now()). // the timestamp of the aggregator provides ordering
			Set("state", status)             // mark if the message was confirmed or rejected
		if err = ag.database.UpdateMessage(ctx, msg.Header.ID, setConfirmed); err != nil {
			return err
		}

		// Generate the appropriate event
		event := fftypes.NewEvent(eventType, msg.Header.Namespace, msg.Header.ID)
		if err = ag.database.InsertEvent(ctx, event); err != nil {
			return err
		}
		log.L(ctx).Infof("Emitting %s %s for message %s:%s", eventType, event.ID, msg.Header.Namespace, msg.Header.ID)
		return nil
	})
	if ag.metricsEnabled {
		ag.metrics.MessageConfirmed(msg, eventType)
	}

	return true, nil
}

// resolveBlobs ensures that the blobs for all the attachments in the data array, have been received into the
// local data exchange blob store. Either because of a private transfer, or by downloading them from the public storage
func (ag *aggregator) resolveBlobs(ctx context.Context, data []*fftypes.Data) (resolved bool, err error) {
	l := log.L(ctx)

	for _, d := range data {
		if d.Blob == nil || d.Blob.Hash == nil {
			continue
		}

		// See if we already have the data
		blob, err := ag.database.GetBlobMatchingHash(ctx, d.Blob.Hash)
		if err != nil {
			return false, err
		}
		if blob != nil {
			l.Debugf("Blob '%s' found in local DX with ref '%s'", blob.Hash, blob.PayloadRef)
			continue
		}

		// If there's a public reference, download it from there and stream it into the blob store
		// We double check the hash on the way, to ensure the streaming from A->B worked ok.
		if d.Blob.Public != "" {
			blob, err = ag.data.CopyBlobPStoDX(ctx, d)
			if err != nil {
				return false, err
			}
			if blob != nil {
				l.Debugf("Blob '%s' downloaded from public storage to local DX with ref '%s'", blob.Hash, blob.PayloadRef)
				continue
			}
		}

		// If we've reached here, the data isn't available yet.
		// This isn't an error, we just need to wait for it to arrive.
		l.Debugf("Blob '%s' not available", d.Blob.Hash)
		return false, nil

	}

	return true, nil

}
