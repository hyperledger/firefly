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

package events

import (
	"context"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/binary"

	"github.com/hyperledger-labs/firefly/internal/broadcast"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/data"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/privatemessaging"
	"github.com/hyperledger-labs/firefly/internal/retry"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

const (
	aggregatorOffsetName = "ff_aggregator"
)

type aggregator struct {
	ctx             context.Context
	database        database.Plugin
	broadcast       broadcast.Manager
	messaging       privatemessaging.Manager
	data            data.Manager
	eventPoller     *eventPoller
	newPins         chan int64
	offchainBatches chan *fftypes.UUID
	queuedRewinds   chan *fftypes.UUID
	retry           *retry.Retry
}

func newAggregator(ctx context.Context, di database.Plugin, bm broadcast.Manager, pm privatemessaging.Manager, dm data.Manager, en *eventNotifier) *aggregator {
	batchSize := config.GetInt(config.EventAggregatorBatchSize)
	ag := &aggregator{
		ctx:             log.WithLogField(ctx, "role", "aggregator"),
		database:        di,
		broadcast:       bm,
		messaging:       pm,
		data:            dm,
		newPins:         make(chan int64),
		offchainBatches: make(chan *fftypes.UUID, 1), // hops to queuedRewinds with a shouldertab on the event poller
		queuedRewinds:   make(chan *fftypes.UUID, batchSize),
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
		offsetType:       fftypes.OffsetTypeAggregator,
		offsetNamespace:  fftypes.SystemNamespace,
		offsetName:       aggregatorOffsetName,
		newEventsHandler: ag.processPinsDBGroup,
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

func (ag *aggregator) start() error {
	go ag.offchainListener()
	return ag.eventPoller.start()
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
				fb.Eq("dispatched", false),
				fb.In("batch", batchIDs),
			).Sort("sequence").Limit(1) // only need the one oldest sequence
			sequences, err := ag.database.GetPins(ag.ctx, filter)
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

func (ag *aggregator) processPinsDBGroup(items []fftypes.LocallySequenced) (repoll bool, err error) {
	pins := make([]*fftypes.Pin, len(items))
	for i, item := range items {
		pins[i] = item.(*fftypes.Pin)
	}
	err = ag.database.RunAsGroup(ag.ctx, func(ctx context.Context) (err error) {
		err = ag.processPins(ctx, pins)
		return err
	})
	return false, err
}

func (ag *aggregator) getPins(ctx context.Context, filter database.Filter) ([]fftypes.LocallySequenced, error) {
	pins, err := ag.database.GetPins(ctx, filter)
	ls := make([]fftypes.LocallySequenced, len(pins))
	for i, p := range pins {
		ls[i] = p
	}
	return ls, err
}

func (ag *aggregator) processPins(ctx context.Context, pins []*fftypes.Pin) (err error) {
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
		for iM := 0; i < pin.Index && iM < len(batch.Payload.Messages); iM++ {
			msg = batch.Payload.Messages[iM]
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
		if err = ag.processMessage(ctx, batch, pin.Masked, pin.Sequence, msg); err != nil {
			return err
		}
	}

	err = ag.eventPoller.commitOffset(ctx, pins[len(pins)-1].Sequence)
	return err
}

func (ag *aggregator) calcHash(topic string, groupID *fftypes.Bytes32, identity string, nonce int64) *fftypes.Bytes32 {
	h := sha256.New()
	h.Write([]byte(topic))
	h.Write((*groupID)[:])
	h.Write([]byte(identity))
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, uint64(nonce))
	h.Write(nonceBytes)
	return fftypes.HashResult(h)
}

func (ag *aggregator) processMessage(ctx context.Context, batch *fftypes.Batch, masked bool, pinnedSequence int64, msg *fftypes.Message) (err error) {
	l := log.L(ctx)

	// Check if it's ready to be processed
	nextPins := make([]*fftypes.NextPin, len(msg.Pins))
	if masked {
		// Private messages have one or more masked "pin" hashes that allow us to work
		// out if it's the next message in the sequence, given the previous messages
		if msg.Header.Group == nil || len(msg.Pins) == 0 || len(msg.Header.Topics) != len(msg.Pins) {
			log.L(ctx).Errorf("Message '%s' in batch '%s' has invalid pin data pins=%v topics=%v", msg.Header.ID, batch.ID, msg.Pins, msg.Header.Topics)
			return nil
		}
		for i, pinStr := range msg.Pins {
			var pin fftypes.Bytes32
			err := pin.UnmarshalText([]byte(pinStr))
			if err != nil {
				log.L(ctx).Errorf("Message '%s' in batch '%s' has invalid pin at index %d: '%s'", msg.Header.ID, batch.ID, i, pinStr)
				return nil
			}
			nextPin, err := ag.checkMaskedContextReady(ctx, msg, msg.Header.Topics[i], pinnedSequence, &pin)
			if err != nil || nextPin == nil {
				return err
			}
			nextPins[i] = nextPin
		}
	} else {
		// We just need to check there's no earlier sequences with the same unmasked context
		unmaskedContexts := make([]driver.Value, len(msg.Header.Topics))
		for i, topic := range msg.Header.Topics {
			h := sha256.New()
			h.Write([]byte(topic))
			unmaskedContexts[i] = fftypes.HashResult(h)
		}
		fb := database.PinQueryFactory.NewFilter(ctx)
		filter := fb.And(
			fb.Eq("dispatched", false),
			fb.In("hash", unmaskedContexts),
			fb.Lt("sequence", pinnedSequence),
		)
		earlier, err := ag.database.GetPins(ctx, filter)
		if err != nil {
			return err
		}
		if len(earlier) > 0 {
			l.Debugf("Message %s pinned at sequence %d blocked by earlier context %s at sequence %d", msg.Header.ID, pinnedSequence, earlier[0].Hash, earlier[0].Sequence)
			return nil
		}
	}

	dispatched, err := ag.attemptMessageDispatch(ctx, msg)
	if err != nil || !dispatched {
		return err
	}

	// Move the nextPin forwards to the next sequence for this sender, on all
	// topics associated with the message
	if masked {
		for i, nextPin := range nextPins {
			nextPin.Nonce++
			nextPin.Hash = ag.calcHash(msg.Header.Topics[i], msg.Header.Group, nextPin.Identity, nextPin.Nonce)
			if err = ag.database.UpdateNextPin(ctx, nextPin.Sequence, database.NextPinQueryFactory.NewUpdate(ctx).
				Set("nonce", nextPin.Nonce).
				Set("hash", nextPin.Hash),
			); err != nil {
				return err
			}
		}
	}

	// Mark the pin dispatched
	return ag.database.SetPinDispatched(ctx, pinnedSequence)
}

func (ag *aggregator) checkMaskedContextReady(ctx context.Context, msg *fftypes.Message, topic string, pinnedSequence int64, pin *fftypes.Bytes32) (*fftypes.NextPin, error) {
	l := log.L(ctx)

	// For masked pins, we can only process if:
	// - it is the next sequence on this context for one of the members of the group
	// - there are no undispatched messages on this context earlier in the stream
	h := sha256.New()
	h.Write([]byte(topic))
	h.Write((*msg.Header.Group)[:])
	contextUnmasked := fftypes.HashResult(h)
	filter := database.NextPinQueryFactory.NewFilter(ctx).Eq("context", contextUnmasked)
	nextPins, err := ag.database.GetNextPins(ctx, filter)
	if err != nil {
		return nil, err
	}
	l.Debugf("Group=%s Topic='%s' NextPins=%v Sequence=%d Pin=%s NextPins=%v", msg.Header.Group, topic, nextPins, pinnedSequence, pin, nextPins)

	if len(nextPins) == 0 {
		// If this is the first time we've seen the context, then this message is read as long as it is
		// the first (nonce=0) message on the context, for one of the members, and there aren't any earlier
		// messages that are nonce=0.
		return ag.attemptContextInit(ctx, msg, topic, pinnedSequence, contextUnmasked, pin)
	}

	// This message must be the next hash for the author
	var nextPin *fftypes.NextPin
	for _, np := range nextPins {
		if *np.Hash == *pin {
			nextPin = np
			break
		}
	}
	if nextPin == nil || nextPin.Identity != msg.Header.Author {
		l.Debugf("Mismatched nexthash or author group=%s topic=%s context=%s pin=%s nextHash=%+v", msg.Header.Group, topic, contextUnmasked, pin, nextPin)
		return nil, nil
	}
	return nextPin, nil
}

func (ag *aggregator) attemptContextInit(ctx context.Context, msg *fftypes.Message, topic string, pinnedSequence int64, contextUnmasked, pin *fftypes.Bytes32) (*fftypes.NextPin, error) {
	l := log.L(ctx)

	// It might be the system topic/context initializing the group
	group, err := ag.messaging.ResolveInitGroup(ctx, msg)
	if err != nil || group == nil {
		return nil, err
	}

	// Find the list of zerohashes for this context, and match this pin to one of them
	zeroHashes := make([]driver.Value, len(group.Members))
	var nextPin *fftypes.NextPin
	nextPins := make([]*fftypes.NextPin, len(group.Members))
	for i, member := range group.Members {
		zeroHash := ag.calcHash(topic, msg.Header.Group, member.Identity, 0)
		np := &fftypes.NextPin{
			Context:  contextUnmasked,
			Identity: member.Identity,
			Hash:     zeroHash,
			Nonce:    0,
		}
		if *pin == *zeroHash {
			if member.Identity != msg.Header.Author {
				l.Warnf("Author mismatch for zerohash on context: group=%s topic=%s context=%s pin=%s", msg.Header.Group, topic, contextUnmasked, pin)
				return nil, nil
			}
			nextPin = np
		}
		zeroHashes[i] = zeroHash
		nextPins[i] = np
	}
	l.Debugf("Group=%s topic=%s context=%s zeroHashes=%v", msg.Header.Group, topic, contextUnmasked, zeroHashes)
	if nextPin == nil {
		l.Warnf("No match for zerohash on context: group=%s topic=%s context=%s pin=%s", msg.Header.Group, topic, contextUnmasked, pin)
		return nil, nil
	}

	// Check none of the other zerohashes exist before us in the stream
	fb := database.PinQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("dispatched", false),
		fb.In("hash", zeroHashes),
		fb.Lt("sequence", pinnedSequence),
	)
	earlier, err := ag.database.GetPins(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(earlier) > 0 {
		l.Debugf("Group=%s topic=%s context=%s earlier=%v", msg.Header.Group, topic, contextUnmasked, earlier)
		return nil, nil
	}

	// We're good to be the first message on this context.
	// Initialize the nextpins on this context - this is safe to do even if we don't actually dispatch the message
	for _, np := range nextPins {
		if err = ag.database.InsertNextPin(ctx, np); err != nil {
			return nil, err
		}
	}
	return nextPin, err
}

func (ag *aggregator) attemptMessageDispatch(ctx context.Context, msg *fftypes.Message) (bool, error) {

	// If we don't find all the data, then we don't dispatch
	data, foundAll, err := ag.data.GetMessageData(ctx, msg, true)
	if err != nil || !foundAll {
		return false, err
	}

	// We're going to dispatch it at this point, but we need to validate the data first
	valid := true
	eventType := fftypes.EventTypeMessageConfirmed
	switch {
	case msg.Header.Namespace == fftypes.SystemNamespace:
		// We handle system events in-line on the aggregator, as it would be confusing for apps to be
		// dispatched subsequent events before we have processed the system events they depend on.
		if valid, err = ag.broadcast.HandleSystemBroadcast(ctx, msg, data); err != nil {
			// Should only return errors that are retryable
			return false, err
		}
	case msg.Header.Type == fftypes.MessageTypeGroupInit:
		// Already handled as part of resolving the context.
		valid = true
		eventType = fftypes.EventTypeGroupConfirmed
	case len(msg.Data) > 0:
		valid, err = ag.data.ValidateAll(ctx, data)
		if err != nil {
			return false, err
		}
	}
	// This message is now confirmed
	setConfirmed := database.MessageQueryFactory.NewUpdate(ctx).
		Set("pending", false).           // the sequence is locked
		Set("confirmed", fftypes.Now()). // the timestamp of the aggregator provides ordering
		Set("rejected", !valid)          // mark if the message was not accepted
	err = ag.database.UpdateMessage(ctx, msg.Header.ID, setConfirmed)
	if err != nil {
		return false, err
	}
	if !valid {
		// An message with invalid (but complete) data is still considered dispatched.
		// However, we drive a different event to the applications.
		eventType = fftypes.EventTypeMessageRejected
	}

	// Generate the appropriate event
	event := fftypes.NewEvent(eventType, msg.Header.Namespace, msg.Header.ID, msg.Header.Group)
	if err = ag.database.UpsertEvent(ctx, event, false); err != nil {
		return false, err
	}
	log.L(ctx).Infof("Emitting %s for message %s:%s", eventType, msg.Header.Namespace, msg.Header.ID)

	return true, nil
}
