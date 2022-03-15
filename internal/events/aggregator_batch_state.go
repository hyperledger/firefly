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

	"github.com/hyperledger/firefly/internal/definitions"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/sirupsen/logrus"
)

func newBatchState(ag *aggregator) *batchState {
	return &batchState{
		database:           ag.database,
		definitions:        ag.definitions,
		maskedContexts:     make(map[fftypes.Bytes32]*nextPinGroupState),
		unmaskedContexts:   make(map[fftypes.Bytes32]*contextState),
		dispatchedMessages: make([]*dispatchedMessage, 0),
		pendingConfirms:    make(map[fftypes.UUID]*fftypes.Message),

		PreFinalize: make([]func(ctx context.Context) error, 0),
		Finalize:    make([]func(ctx context.Context) error, 0),
	}
}

// nextPinGroupState manages the state during the batch for an individual masked context.
// We read it from the database the first time a context is touched within a batch, and then
// replace the pins in it as they are spent/replaced for each pin processed.
// At the end we flush the changes to the database
type nextPinGroupState struct {
	groupID           *fftypes.Bytes32
	topic             string
	nextPins          []*fftypes.NextPin
	new               bool
	identitiesChanged map[string]bool
}

type nextPinState struct {
	nextPinGroup *nextPinGroupState
	nextPin      *fftypes.NextPin
}

// contextState tracks the unmasked (broadcast) pins related to a particular context
// in the batch, as they might block further messages in the batch
type contextState struct {
	blockedBy int64
}

// dispatchedMessage is a record for a message that was dispatched as part of processing this batch,
// so that all pins associated to the message can be marked dispatched at the end of the batch.
type dispatchedMessage struct {
	batchID       *fftypes.UUID
	msgID         *fftypes.UUID
	firstPinIndex int64
	topicCount    int
	msgPins       fftypes.FFStringArray
}

// batchState is the object that tracks the in-memory state that builds up while processing a batch of pins,
// that needs to be reconciled at the point the batch closes.
// There are three phases:
// 1. Dispatch: Determines if messages are blocked, or can be dispatched. Calls the appropriate dispatch
//              actions for that message type. Reads initial `pin` state for contexts from the DB, and then
//              updates this in-memory throughout the batch, ready for flushing in the Finalize phase.
//              Runs in a database operation group/tranaction.
// 2. Pre-finalize: Runs any PreFinalize callbacks registered by the handlers in (1).
//                  Intended to be used for cross-microservice REST/GRPC etc. calls that have side-effects.
//                  Runs outside any database operation group/tranaction.
// 3. Finalize: Flushes the `pin` state calculated in phase (1), and any Finalize actions registered by handlers
//              during phase (1) or (2).
//              Runs in a database operation group/tranaction, which will be the same as phase (1) if there
//              are no pre-finalize handlers registered.
type batchState struct {
	database           database.Plugin
	definitions        definitions.DefinitionHandlers
	maskedContexts     map[fftypes.Bytes32]*nextPinGroupState
	unmaskedContexts   map[fftypes.Bytes32]*contextState
	dispatchedMessages []*dispatchedMessage
	pendingConfirms    map[fftypes.UUID]*fftypes.Message

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

func (bs *batchState) AddPreFinalize(action func(ctx context.Context) error) {
	if action != nil {
		bs.PreFinalize = append(bs.PreFinalize, action)
	}
}

func (bs *batchState) AddFinalize(action func(ctx context.Context) error) {
	if action != nil {
		bs.Finalize = append(bs.Finalize, action)
	}
}

func (bs *batchState) GetPendingConfirm() map[fftypes.UUID]*fftypes.Message {
	return bs.pendingConfirms
}

func (bs *batchState) RunPreFinalize(ctx context.Context) error {
	for _, action := range bs.PreFinalize {
		if err := action(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (bs *batchState) RunFinalize(ctx context.Context) error {
	for _, action := range bs.Finalize {
		if err := action(ctx); err != nil {
			return err
		}
	}
	return bs.flushPins(ctx)
}

func (bs *batchState) CheckUnmaskedContextReady(ctx context.Context, contextUnmasked *fftypes.Bytes32, msg *fftypes.Message, topic string, firstMsgPinSequence int64) (bool, error) {

	ucs, found := bs.unmaskedContexts[*contextUnmasked]
	if !found {
		ucs = &contextState{blockedBy: -1}
		bs.unmaskedContexts[*contextUnmasked] = ucs

		// We need to check there's no earlier sequences with the same unmasked context
		fb := database.PinQueryFactory.NewFilterLimit(ctx, 1) // only need the first one
		filter := fb.And(
			fb.Eq("hash", contextUnmasked),
			fb.Eq("dispatched", false),
			fb.Lt("sequence", firstMsgPinSequence),
		)
		earlier, _, err := bs.database.GetPins(ctx, filter)
		if err != nil {
			return false, err
		}
		if len(earlier) > 0 {
			ucs.blockedBy = earlier[0].Sequence
		}

	}

	blocked := ucs.blockedBy >= 0
	if blocked {
		log.L(ctx).Debugf("Message %s pinned at sequence %d blocked by earlier context %s at sequence %d", msg.Header.ID, firstMsgPinSequence, contextUnmasked, ucs.blockedBy)
	}
	return !blocked, nil

}

func (bs *batchState) CheckMaskedContextReady(ctx context.Context, msg *fftypes.Message, topic string, firstMsgPinSequence int64, pin *fftypes.Bytes32, nonceStr string) (*nextPinState, error) {
	l := log.L(ctx)

	// For masked pins, we can only process if:
	// - it is the next sequence on this context for one of the members of the group
	// - there are no undispatched messages on this context earlier in the stream
	h := sha256.New()
	h.Write([]byte(topic))
	h.Write((*msg.Header.Group)[:])
	contextUnmasked := fftypes.HashResult(h)
	npg, err := bs.stateForMaskedContext(ctx, msg.Header.Group, topic, *contextUnmasked)
	if err != nil {
		return nil, err
	}
	if npg == nil {
		// If this is the first time we've seen the context, then this message is read as long as it is
		// the first (nonce=0) message on the context, for one of the members, and there aren't any earlier
		// messages that are nonce=0.
		return bs.attemptContextInit(ctx, msg, topic, firstMsgPinSequence, contextUnmasked, pin)
	}

	// This message must be the next hash for the author
	l.Debugf("Group=%s Topic='%s' Sequence=%d Pin=%s", msg.Header.Group, topic, firstMsgPinSequence, pin)
	var nextPin *fftypes.NextPin
	for _, np := range npg.nextPins {
		if *np.Hash == *pin {
			nextPin = np
			break
		}
	}
	if nextPin == nil || nextPin.Identity != msg.Header.Author {
		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			for _, np := range npg.nextPins {
				l.Debugf("NextPin: context=%s author=%s nonce=%d hash=%s", np.Context, np.Identity, np.Nonce, np.Hash)
			}
		}
		l.Warnf("Mismatched nexthash or author group=%s topic=%s context=%s pin=%s nonce=%s nextHash=%+v author=%s", msg.Header.Group, topic, contextUnmasked, pin, nonceStr, nextPin, msg.Header.Author)
		return nil, nil
	}
	return &nextPinState{
		nextPinGroup: npg,
		nextPin:      nextPin,
	}, err
}

func (bs *batchState) MarkMessageDispatched(ctx context.Context, batchID *fftypes.UUID, msg *fftypes.Message, msgBaseIndex int64) {
	bs.dispatchedMessages = append(bs.dispatchedMessages, &dispatchedMessage{
		batchID:       batchID,
		msgID:         msg.Header.ID,
		firstPinIndex: msgBaseIndex,
		topicCount:    len(msg.Header.Topics),
		msgPins:       msg.Pins,
	})
}

func (bs *batchState) SetContextBlockedBy(ctx context.Context, unmaskedContext fftypes.Bytes32, blockedBy int64) {
	ucs, found := bs.unmaskedContexts[unmaskedContext]
	if !found {
		ucs = &contextState{blockedBy: blockedBy}
		bs.unmaskedContexts[unmaskedContext] = ucs
	} else if ucs.blockedBy < 0 {
		// Do not update an existing block, as we want the earliest entry to be the block
		ucs.blockedBy = blockedBy
	}
}

func (bs *batchState) flushPins(ctx context.Context) error {
	l := log.L(ctx)

	// Update all the next pins
	for _, npg := range bs.maskedContexts {
		for _, np := range npg.nextPins {
			if npg.new {
				if err := bs.database.InsertNextPin(ctx, np); err != nil {
					return err
				}
			} else if npg.identitiesChanged[np.Identity] {
				update := database.NextPinQueryFactory.NewUpdate(ctx).
					Set("nonce", np.Nonce).
					Set("hash", np.Hash)
				if err := bs.database.UpdateNextPin(ctx, np.Sequence, update); err != nil {
					return err
				}
			}
		}
	}

	// Update all the pins that have been dispatched
	// It's important we don't re-process the message, so we update all pins for a message to dispatched in one go,
	// using the index range of pins it owns within the batch it is a part of.
	// Note that this might include pins not in the batch we read from the database, as the page size
	// cannot be guaranteed to overlap with the set of indexes of a message within a batch.
	pinsDispatched := make(map[fftypes.UUID][]driver.Value)
	for _, dm := range bs.dispatchedMessages {
		batchDispatched := pinsDispatched[*dm.batchID]
		l.Debugf("Marking message dispatched batch=%s msg=%s firstIndex=%d topics=%d pins=%s", dm.batchID, dm.msgID, dm.firstPinIndex, dm.topicCount, dm.msgPins)
		for i := 0; i < dm.topicCount; i++ {
			batchDispatched = append(batchDispatched, dm.firstPinIndex+int64(i))
		}
		if len(batchDispatched) > 0 {
			pinsDispatched[*dm.batchID] = batchDispatched
		}
	}
	// Build one uber update for DB efficiency
	if len(pinsDispatched) > 0 {
		fb := database.PinQueryFactory.NewFilter(ctx)
		filter := fb.Or()
		for batchID, indexes := range pinsDispatched {
			filter.Condition(fb.And(
				fb.Eq("batch", batchID),
				fb.In("index", indexes),
			))
		}
		update := database.PinQueryFactory.NewUpdate(ctx).Set("dispatched", true)
		if err := bs.database.UpdatePins(ctx, filter, update); err != nil {
			return err
		}
	}

	return nil
}

func (nps *nextPinState) IncrementNextPin(ctx context.Context) {
	npg := nps.nextPinGroup
	np := nps.nextPin
	for i, existingPin := range npg.nextPins {
		if np.Hash.Equals(existingPin.Hash) {
			// We are spending this one, replace it in the list
			newNonce := np.Nonce + 1
			newNextPin := &fftypes.NextPin{
				Context:  np.Context,
				Identity: np.Identity,
				Nonce:    newNonce,
				Hash:     npg.calcPinHash(np.Identity, newNonce),
				Sequence: np.Sequence, // used for update in Flush
			}
			npg.nextPins[i] = newNextPin
			log.L(ctx).Debugf("Incrementing NextPin=%s - Nonce=%d Topic=%s Group=%s NewNextPin=%s", np.Hash, np.Nonce, npg.topic, npg.groupID.String(), newNextPin.Hash)
			// Mark the identity as needing its hash replacing when we come to write back to the DB
			npg.identitiesChanged[np.Identity] = true
		}
	}
}

func (npg *nextPinGroupState) calcPinHash(identity string, nonce int64) *fftypes.Bytes32 {
	h := sha256.New()
	h.Write([]byte(npg.topic))
	h.Write((*npg.groupID)[:])
	h.Write([]byte(identity))
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, uint64(nonce))
	h.Write(nonceBytes)
	return fftypes.HashResult(h)
}

func (bs *batchState) stateForMaskedContext(ctx context.Context, groupID *fftypes.Bytes32, topic string, contextUnmasked fftypes.Bytes32) (*nextPinGroupState, error) {

	if npg, exists := bs.maskedContexts[contextUnmasked]; exists {
		return npg, nil
	}

	filter := database.NextPinQueryFactory.NewFilter(ctx).Eq("context", contextUnmasked)
	nextPins, _, err := bs.database.GetNextPins(ctx, filter)
	if err != nil {
		return nil, err
	}

	if len(nextPins) == 0 {
		return nil, nil
	}

	npg := &nextPinGroupState{
		groupID:           groupID,
		topic:             topic,
		identitiesChanged: make(map[string]bool),
		nextPins:          nextPins,
	}
	bs.maskedContexts[contextUnmasked] = npg
	return npg, nil

}

func (bs *batchState) attemptContextInit(ctx context.Context, msg *fftypes.Message, topic string, pinnedSequence int64, contextUnmasked, pin *fftypes.Bytes32) (*nextPinState, error) {
	l := log.L(ctx)

	// It might be the system topic/context initializing the group
	// - This performs the actual database updates in-line, as it is idempotent
	group, err := bs.definitions.ResolveInitGroup(ctx, msg)
	if err != nil || group == nil {
		return nil, err
	}

	npg := &nextPinGroupState{
		groupID:           msg.Header.Group,
		topic:             topic,
		new:               true,
		identitiesChanged: make(map[string]bool),
		nextPins:          make([]*fftypes.NextPin, len(group.Members)),
	}

	// Find the list of zerohashes for this context, and match this pin to one of them
	zeroHashes := make([]driver.Value, len(group.Members))
	var nextPin *fftypes.NextPin
	for i, member := range group.Members {
		zeroHash := npg.calcPinHash(member.Identity, 0)
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
		npg.nextPins[i] = np
	}
	l.Debugf("Group=%s topic=%s context=%s zeroHashes=%v", msg.Header.Group, topic, contextUnmasked, zeroHashes)
	if nextPin == nil {
		l.Warnf("No match for zerohash on context: group=%s topic=%s context=%s pin=%s", msg.Header.Group, topic, contextUnmasked, pin)
		return nil, nil
	}

	// Check none of the other zerohashes exist before us in the stream
	fb := database.PinQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.In("hash", zeroHashes),
		fb.Eq("dispatched", false),
		fb.Lt("sequence", pinnedSequence),
	)
	earlier, _, err := bs.database.GetPins(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(earlier) > 0 {
		l.Debugf("Group=%s topic=%s context=%s earlier=%v", msg.Header.Group, topic, contextUnmasked, earlier)
		return nil, nil
	}

	// Initialize the nextpins on this context
	bs.maskedContexts[*contextUnmasked] = npg
	return &nextPinState{
		nextPin:      nextPin,
		nextPinGroup: npg,
	}, err
}
