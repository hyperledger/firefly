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
	"testing"

	"github.com/kaleido-io/firefly/mocks/broadcastmocks"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/mocks/datamocks"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func uuidMatches(id1 *fftypes.UUID) interface{} {
	return mock.MatchedBy(func(id2 *fftypes.UUID) bool { return id1.Equals(id2) })
}

func newTestAggregator() (*aggregator, func()) {
	mdi := &databasemocks.Plugin{}
	mbm := &broadcastmocks.Manager{}
	mdm := &datamocks.Manager{}
	ctx, cancel := context.WithCancel(context.Background())
	ag := newAggregator(ctx, mdi, mbm, mdm, newEventNotifier(ctx))
	return ag, cancel
}

func TestAggregationMaskedZeroNonceMatch(t *testing.T) {

	ag, cancel := newTestAggregator()
	defer cancel()

	// Generate some pin data
	member1 := "0x12345"
	member2 := "0x23456"
	topic := "some-topic"
	batchID := fftypes.NewUUID()
	groupID := fftypes.NewUUID()
	msgID := fftypes.NewUUID()
	h := sha256.New()
	h.Write([]byte(topic))
	h.Write((*groupID)[:])
	contextUnmasked := fftypes.HashResult(h)
	member1NonceZero := ag.calcHash(topic, groupID, member1, 0)
	member2NonceZero := ag.calcHash(topic, groupID, member2, 0)
	member2NonceOne := ag.calcHash(topic, groupID, member2, 1)

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)

	// Get the batch
	mdi.On("GetBatchByID", ag.ctx, uuidMatches(batchID)).Return(&fftypes.Batch{
		ID: batchID,
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{
					Header: fftypes.MessageHeader{
						ID:     msgID,
						Group:  groupID,
						Topics: []string{topic},
						Author: member2,
					},
					Pins: []string{member2NonceZero.String()},
				},
			},
		},
	}, nil)
	// Look for existing nextpins - none found, first on context
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return([]*fftypes.NextPin{}, nil).Once()
	// Get the group members
	mdi.On("GetGroupByID", ag.ctx, uuidMatches(groupID)).Return(&fftypes.Group{
		ID: groupID,
		Members: fftypes.Members{
			{Identity: member1},
			{Identity: member2},
		},
	}, nil)
	// Look for any earlier pins - none found
	mdi.On("GetPins", ag.ctx, mock.Anything).Return([]*fftypes.Pin{}, nil).Once()
	// Insert all the zero pins
	mdi.On("InsertNextPin", ag.ctx, mock.MatchedBy(func(np *fftypes.NextPin) bool {
		assert.Equal(t, *np.Context, *contextUnmasked)
		np.Sequence = 10011
		return *np.Hash == *member1NonceZero && np.Nonce == 0
	})).Return(nil).Once()
	mdi.On("InsertNextPin", ag.ctx, mock.MatchedBy(func(np *fftypes.NextPin) bool {
		assert.Equal(t, *np.Context, *contextUnmasked)
		np.Sequence = 10012
		return *np.Hash == *member2NonceZero && np.Nonce == 0
	})).Return(nil).Once()
	// Validate the message is ok
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	// Insert the confirmed event
	mdi.On("UpsertEvent", ag.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return *e.Reference == *msgID && e.Type == fftypes.EventTypeMessageConfirmed
	}), false).Return(nil)
	// Update member2 to nonce 1
	mdi.On("UpdateNextPin", ag.ctx, mock.MatchedBy(func(seq int64) bool {
		return seq == 10012
	}), mock.MatchedBy(func(update database.Update) bool {
		ui, _ := update.Finalize()
		assert.Equal(t, "nonce", ui.SetOperations[0].Field)
		v, _ := ui.SetOperations[0].Value.Value()
		assert.Equal(t, int64(1), v.(int64))
		assert.Equal(t, "hash", ui.SetOperations[1].Field)
		v, _ = ui.SetOperations[1].Value.Value()
		assert.Equal(t, member2NonceOne.String(), v)
		return true
	})).Return(nil)
	// Set the pin to dispatched
	mdi.On("SetPinDispatched", ag.ctx, int64(10001)).Return(nil)
	// Confirm the offset
	mdi.On("UpdateOffset", ag.ctx, mock.Anything, mock.Anything).Return(nil)

	err := ag.processPins(ag.ctx, []*fftypes.Pin{
		{
			Sequence:   10001,
			Masked:     true,
			Hash:       member2NonceZero,
			Batch:      batchID,
			Index:      0,
			Dispatched: false,
		},
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestAggregationMaskedNextSequenceMatch(t *testing.T) {

	ag, cancel := newTestAggregator()
	defer cancel()

	// Generate some pin data
	member1 := "0x12345"
	member2 := "0x23456"
	topic := "some-topic"
	batchID := fftypes.NewUUID()
	groupID := fftypes.NewUUID()
	msgID := fftypes.NewUUID()
	h := sha256.New()
	h.Write([]byte(topic))
	h.Write((*groupID)[:])
	contextUnmasked := fftypes.HashResult(h)
	member1Nonce100 := ag.calcHash(topic, groupID, member1, 100)
	member2Nonce500 := ag.calcHash(topic, groupID, member2, 500)
	member2Nonce501 := ag.calcHash(topic, groupID, member2, 501)

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)

	// Get the batch
	mdi.On("GetBatchByID", ag.ctx, uuidMatches(batchID)).Return(&fftypes.Batch{
		ID: batchID,
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{
					Header: fftypes.MessageHeader{
						ID:     msgID,
						Group:  groupID,
						Topics: []string{topic},
						Author: member2,
					},
					Pins: []string{member2Nonce500.String()},
				},
			},
		},
	}, nil)
	// Look for existing nextpins - none found, first on context
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return([]*fftypes.NextPin{
		{Context: contextUnmasked, Identity: member1, Hash: member1Nonce100, Nonce: 100, Sequence: 929},
		{Context: contextUnmasked, Identity: member2, Hash: member2Nonce500, Nonce: 500, Sequence: 424},
	}, nil).Once()
	// Validate the message is ok
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	// Insert the confirmed event
	mdi.On("UpsertEvent", ag.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return *e.Reference == *msgID && e.Type == fftypes.EventTypeMessageConfirmed
	}), false).Return(nil)
	// Update member2 to nonce 1
	mdi.On("UpdateNextPin", ag.ctx, mock.MatchedBy(func(seq int64) bool {
		return seq == 424
	}), mock.MatchedBy(func(update database.Update) bool {
		ui, _ := update.Finalize()
		assert.Equal(t, "nonce", ui.SetOperations[0].Field)
		v, _ := ui.SetOperations[0].Value.Value()
		assert.Equal(t, int64(501), v.(int64))
		assert.Equal(t, "hash", ui.SetOperations[1].Field)
		v, _ = ui.SetOperations[1].Value.Value()
		assert.Equal(t, member2Nonce501.String(), v)
		return true
	})).Return(nil)
	// Set the pin to dispatched
	mdi.On("SetPinDispatched", ag.ctx, int64(10001)).Return(nil)
	// Confirm the offset
	mdi.On("UpdateOffset", ag.ctx, mock.Anything, mock.Anything).Return(nil)

	err := ag.processPins(ag.ctx, []*fftypes.Pin{
		{
			Sequence:   10001,
			Masked:     true,
			Hash:       member2Nonce500,
			Batch:      batchID,
			Index:      0,
			Dispatched: false,
		},
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestAggregationBroadcast(t *testing.T) {

	ag, cancel := newTestAggregator()
	defer cancel()

	// Generate some pin data
	topic := "some-topic"
	batchID := fftypes.NewUUID()
	msgID := fftypes.NewUUID()
	h := sha256.New()
	h.Write([]byte(topic))
	contextUnmasked := fftypes.HashResult(h)

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)

	// Get the batch
	member1 := "0x12345"
	mdi.On("GetBatchByID", ag.ctx, uuidMatches(batchID)).Return(&fftypes.Batch{
		ID: batchID,
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{
					Header: fftypes.MessageHeader{
						ID:     msgID,
						Topics: []string{topic},
						Author: member1,
					},
				},
			},
		},
	}, nil)
	// Do not resolve any pins earlier
	mdi.On("GetPins", mock.Anything, mock.Anything).Return([]*fftypes.Pin{}, nil)
	// Validate the message is ok
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	// Insert the confirmed event
	mdi.On("UpsertEvent", ag.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return *e.Reference == *msgID && e.Type == fftypes.EventTypeMessageConfirmed
	}), false).Return(nil)
	// Set the pin to dispatched
	mdi.On("SetPinDispatched", ag.ctx, int64(10001)).Return(nil)
	// Confirm the offset
	mdi.On("UpdateOffset", ag.ctx, mock.Anything, mock.Anything).Return(nil)

	err := ag.processPins(ag.ctx, []*fftypes.Pin{
		{
			Sequence:   10001,
			Hash:       contextUnmasked,
			Batch:      batchID,
			Index:      0,
			Dispatched: false,
		},
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}
func TestShutdownOnCancel(t *testing.T) {
	ag, cancel := newTestAggregator()
	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeAggregator, fftypes.SystemNamespace, aggregatorOffsetName).Return(&fftypes.Offset{
		Type:      fftypes.OffsetTypeAggregator,
		Namespace: fftypes.SystemNamespace,
		Name:      aggregatorOffsetName,
		Current:   12345,
	}, nil)
	mdi.On("GetPins", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Pin{}, nil)
	err := ag.start()
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), ag.eventPoller.pollingOffset)
	ag.eventPoller.eventNotifier.newEvents <- 12345
	cancel()
	<-ag.eventPoller.closed
}
