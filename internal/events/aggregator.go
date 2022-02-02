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
	"encoding/binary"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/definitions"
	"github.com/hyperledger/firefly/internal/log"
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
}

func newAggregator(ctx context.Context, di database.Plugin, sh definitions.DefinitionHandlers, dm data.Manager, en *eventNotifier) *aggregator {
	batchSize := config.GetInt(config.EventAggregatorBatchSize)
	ag := &aggregator{
		ctx:             log.WithLogField(ctx, "role", "aggregator"),
		database:        di,
		definitions:     sh,
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
		namespace:        fftypes.SystemNamespace,
		offsetType:       fftypes.OffsetTypeAggregator,
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
				fb.Eq("dispatched", false),
				fb.In("batch", batchIDs),
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

// batchActions are synchronous actions to be performed while processing system messages, but which must happen after reading the whole batch
type batchActions struct {
	// PreFinalize callbacks may perform blocking actions (possibly to an external connector)
	// - Will execute after all batch messages have been processed
	// - Will execute outside database RunAsGroup
	// - If any PreFinalize callback errors out, batch will be aborted and retried
	PreFinalize []func(ctx context.Context) error

	// Finalize callbacks may perform final, non-idempotent database operations (such as inserting Events)
	// - Will execute after all batch messages have been processed and any PreFinalize callbacks have succeeded
	// - Will execute inside database RunAsGroup
	// - If any Finalize callback errors out, batch will be aborted and retried (small chance of duplicate execution here)
	Finalize []func(ctx context.Context) error
}

func (ba *batchActions) AddPreFinalize(action func(ctx context.Context) error) {
	if action != nil {
		ba.PreFinalize = append(ba.PreFinalize, action)
	}
}

func (ba *batchActions) AddFinalize(action func(ctx context.Context) error) {
	if action != nil {
		ba.Finalize = append(ba.Finalize, action)
	}
}

func (ba *batchActions) RunPreFinalize(ctx context.Context) error {
	for _, action := range ba.PreFinalize {
		if err := action(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (ba *batchActions) RunFinalize(ctx context.Context) error {
	for _, action := range ba.Finalize {
		if err := action(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (ag *aggregator) processWithBatchActions(callback func(ctx context.Context, actions *batchActions) error) error {
	actions := &batchActions{
		PreFinalize: make([]func(ctx context.Context) error, 0),
		Finalize:    make([]func(ctx context.Context) error, 0),
	}

	err := ag.database.RunAsGroup(ag.ctx, func(ctx context.Context) (err error) {
		if err := callback(ctx, actions); err != nil {
			return err
		}
		if len(actions.PreFinalize) == 0 {
			return actions.RunFinalize(ctx)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(actions.PreFinalize) == 0 {
		return err
	}

	if err := actions.RunPreFinalize(ag.ctx); err != nil {
		return err
	}
	return ag.database.RunAsGroup(ag.ctx, func(ctx context.Context) error {
		return actions.RunFinalize(ctx)
	})
}

func (ag *aggregator) processPinsDBGroup(items []fftypes.LocallySequenced) (repoll bool, err error) {
	pins := make([]*fftypes.Pin, len(items))
	for i, item := range items {
		pins[i] = item.(*fftypes.Pin)
	}

	return false, ag.processWithBatchActions(func(ctx context.Context, actions *batchActions) error {
		return ag.processPins(ctx, pins, actions)
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

func (ag *aggregator) processPins(ctx context.Context, pins []*fftypes.Pin, actions *batchActions) (err error) {
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
		if err = ag.processMessage(ctx, batch, pin.Masked, pin.Sequence, msg, actions); err != nil {
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

func (ag *aggregator) processMessage(ctx context.Context, batch *fftypes.Batch, masked bool, pinnedSequence int64, msg *fftypes.Message, actions *batchActions) (err error) {
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
		earlier, _, err := ag.database.GetPins(ctx, filter)
		if err != nil {
			return err
		}
		if len(earlier) > 0 {
			l.Debugf("Message %s pinned at sequence %d blocked by earlier context %s at sequence %d", msg.Header.ID, pinnedSequence, earlier[0].Hash, earlier[0].Sequence)
			return nil
		}
	}

	dispatched, err := ag.attemptMessageDispatch(ctx, msg, actions)
	if err != nil || !dispatched {
		return err
	}

	actions.AddFinalize(func(ctx context.Context) error {
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
	})

	return nil
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
	nextPins, _, err := ag.database.GetNextPins(ctx, filter)
	if err != nil {
		return nil, err
	}
	l.Debugf("Group=%s Topic='%s' Sequence=%d Pin=%s NextPins=%v", msg.Header.Group, topic, pinnedSequence, pin, nextPins)

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
		l.Warnf("Mismatched nexthash or author group=%s topic=%s context=%s pin=%s nextHash=%+v", msg.Header.Group, topic, contextUnmasked, pin, nextPin)
		return nil, nil
	}
	return nextPin, nil
}

func (ag *aggregator) attemptContextInit(ctx context.Context, msg *fftypes.Message, topic string, pinnedSequence int64, contextUnmasked, pin *fftypes.Bytes32) (*fftypes.NextPin, error) {
	l := log.L(ctx)

	// It might be the system topic/context initializing the group
	group, err := ag.definitions.ResolveInitGroup(ctx, msg)
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
	earlier, _, err := ag.database.GetPins(ctx, filter)
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

func (ag *aggregator) attemptMessageDispatch(ctx context.Context, msg *fftypes.Message, actions *batchActions) (bool, error) {

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
			actions.AddPreFinalize(batchAction.PreFinalize)
			actions.AddFinalize(batchAction.Finalize)
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

	state := fftypes.MessageStateConfirmed
	eventType := fftypes.EventTypeMessageConfirmed
	if !valid {
		state = fftypes.MessageStateRejected
		eventType = fftypes.EventTypeMessageRejected
	}

	actions.AddFinalize(func(ctx context.Context) error {
		// This message is now confirmed
		setConfirmed := database.MessageQueryFactory.NewUpdate(ctx).
			Set("confirmed", fftypes.Now()). // the timestamp of the aggregator provides ordering
			Set("state", state)              // mark if the message was confirmed or rejected
		if err = ag.database.UpdateMessage(ctx, msg.Header.ID, setConfirmed); err != nil {
			return err
		}

		// Generate the appropriate event
		event := fftypes.NewEvent(eventType, msg.Header.Namespace, msg.Header.ID)
		if err = ag.database.InsertEvent(ctx, event); err != nil {
			return err
		}
		log.L(ctx).Infof("Emitting %s for message %s:%s", eventType, msg.Header.Namespace, msg.Header.ID)
		return nil
	})

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
