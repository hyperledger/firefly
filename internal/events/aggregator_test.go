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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/definitions"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/definitionsmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestAggregator() (*aggregator, func()) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	msh := &definitionsmocks.DefinitionHandlers{}
	ctx, cancel := context.WithCancel(context.Background())
	ag := newAggregator(ctx, mdi, msh, mdm, newEventNotifier(ctx, "ut"))
	return ag, cancel
}

func TestAggregationMaskedZeroNonceMatch(t *testing.T) {

	ag, cancel := newTestAggregator()
	defer cancel()
	ba := &batchActions{}

	// Generate some pin data
	member1org := "org1"
	member2org := "org2"
	member2key := "0x23456"
	topic := "some-topic"
	batchID := fftypes.NewUUID()
	groupID := fftypes.NewRandB32()
	msgID := fftypes.NewUUID()
	h := sha256.New()
	h.Write([]byte(topic))
	h.Write((*groupID)[:])
	contextUnmasked := fftypes.HashResult(h)
	member1NonceZero := ag.calcHash(topic, groupID, member1org, 0)
	member2NonceZero := ag.calcHash(topic, groupID, member2org, 0)
	member2NonceOne := ag.calcHash(topic, groupID, member2org, 1)

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)
	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)

	// Get the batch
	mdi.On("GetBatchByID", ag.ctx, batchID).Return(&fftypes.Batch{
		ID: batchID,
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{
					Header: fftypes.MessageHeader{
						ID:     msgID,
						Group:  groupID,
						Topics: []string{topic},
						Identity: fftypes.Identity{
							Author: member2org,
							Key:    member2key,
						},
					},
					Pins: []string{member2NonceZero.String()},
					Data: fftypes.DataRefs{
						{ID: fftypes.NewUUID()},
					},
				},
			},
		},
	}, nil)
	// Look for existing nextpins - none found, first on context
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return([]*fftypes.NextPin{}, nil, nil).Once()
	// Get the group members
	msh.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(&fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{
				{Identity: member1org},
				{Identity: member2org},
			},
		},
	}, nil)
	// Look for any earlier pins - none found
	mdi.On("GetPins", ag.ctx, mock.Anything).Return([]*fftypes.Pin{}, nil, nil).Once()
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
	mdi.On("InsertEvent", ag.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return *e.Reference == *msgID && e.Type == fftypes.EventTypeMessageConfirmed
	})).Return(nil)
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
	// Update the message
	mdi.On("UpdateMessage", ag.ctx, mock.Anything, mock.MatchedBy(func(u database.Update) bool {
		update, err := u.Finalize()
		assert.NoError(t, err)
		assert.Len(t, update.SetOperations, 2)

		assert.Equal(t, "confirmed", update.SetOperations[0].Field)
		v, err := update.SetOperations[0].Value.Value()
		assert.NoError(t, err)
		assert.Greater(t, v, int64(0))

		assert.Equal(t, "state", update.SetOperations[1].Field)
		v, err = update.SetOperations[1].Value.Value()
		assert.NoError(t, err)
		assert.Equal(t, "confirmed", v)

		return true
	})).Return(nil)
	// Confirm the offset
	mdi.On("UpdateOffset", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := ag.processPins(ag.ctx, []*fftypes.Pin{
		{
			Sequence:   10001,
			Masked:     true,
			Hash:       member2NonceZero,
			Batch:      batchID,
			Index:      0,
			Dispatched: false,
		},
	}, ba)
	assert.NoError(t, err)

	err = ba.RunFinalize(ag.ctx)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestAggregationMaskedNextSequenceMatch(t *testing.T) {

	ag, cancel := newTestAggregator()
	defer cancel()
	ba := &batchActions{}

	// Generate some pin data
	member1org := "org1"
	member2org := "org2"
	member2key := "0x12345"
	topic := "some-topic"
	batchID := fftypes.NewUUID()
	groupID := fftypes.NewRandB32()
	msgID := fftypes.NewUUID()
	h := sha256.New()
	h.Write([]byte(topic))
	h.Write((*groupID)[:])
	contextUnmasked := fftypes.HashResult(h)
	member1Nonce100 := ag.calcHash(topic, groupID, member1org, 100)
	member2Nonce500 := ag.calcHash(topic, groupID, member2org, 500)
	member2Nonce501 := ag.calcHash(topic, groupID, member2org, 501)

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)

	// Get the batch
	mdi.On("GetBatchByID", ag.ctx, batchID).Return(&fftypes.Batch{
		ID: batchID,
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{
					Header: fftypes.MessageHeader{
						ID:     msgID,
						Group:  groupID,
						Topics: []string{topic},
						Identity: fftypes.Identity{
							Author: member2org,
							Key:    member2key,
						},
					},
					Pins: []string{member2Nonce500.String()},
					Data: fftypes.DataRefs{
						{ID: fftypes.NewUUID()},
					},
				},
			},
		},
	}, nil)
	// Look for existing nextpins - none found, first on context
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return([]*fftypes.NextPin{
		{Context: contextUnmasked, Identity: member1org, Hash: member1Nonce100, Nonce: 100, Sequence: 929},
		{Context: contextUnmasked, Identity: member2org, Hash: member2Nonce500, Nonce: 500, Sequence: 424},
	}, nil, nil).Once()
	// Validate the message is ok
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	// Insert the confirmed event
	mdi.On("InsertEvent", ag.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return *e.Reference == *msgID && e.Type == fftypes.EventTypeMessageConfirmed
	})).Return(nil)
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
	// Update the message
	mdi.On("UpdateMessage", ag.ctx, mock.Anything, mock.Anything).Return(nil)
	// Confirm the offset
	mdi.On("UpdateOffset", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := ag.processPins(ag.ctx, []*fftypes.Pin{
		{
			Sequence:   10001,
			Masked:     true,
			Hash:       member2Nonce500,
			Batch:      batchID,
			Index:      0,
			Dispatched: false,
		},
	}, ba)
	assert.NoError(t, err)

	err = ba.RunFinalize(ag.ctx)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestAggregationBroadcast(t *testing.T) {

	ag, cancel := newTestAggregator()
	defer cancel()
	ba := &batchActions{}

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
	member1org := "org1"
	member1key := "0x12345"
	mdi.On("GetBatchByID", ag.ctx, batchID).Return(&fftypes.Batch{
		ID: batchID,
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{
					Header: fftypes.MessageHeader{
						ID:     msgID,
						Topics: []string{topic},
						Identity: fftypes.Identity{
							Author: member1org,
							Key:    member1key,
						},
					},
					Data: fftypes.DataRefs{
						{ID: fftypes.NewUUID()},
					},
				},
			},
		},
	}, nil)
	// Do not resolve any pins earlier
	mdi.On("GetPins", mock.Anything, mock.Anything).Return([]*fftypes.Pin{}, nil, nil)
	// Validate the message is ok
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	// Insert the confirmed event
	mdi.On("InsertEvent", ag.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return *e.Reference == *msgID && e.Type == fftypes.EventTypeMessageConfirmed
	})).Return(nil)
	// Set the pin to dispatched
	mdi.On("SetPinDispatched", ag.ctx, int64(10001)).Return(nil)
	// Update the message
	mdi.On("UpdateMessage", ag.ctx, mock.Anything, mock.Anything).Return(nil)
	// Confirm the offset
	mdi.On("UpdateOffset", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := ag.processPins(ag.ctx, []*fftypes.Pin{
		{
			Sequence:   10001,
			Hash:       contextUnmasked,
			Batch:      batchID,
			Index:      0,
			Dispatched: false,
		},
	}, ba)
	assert.NoError(t, err)

	err = ba.RunFinalize(ag.ctx)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestShutdownOnCancel(t *testing.T) {
	ag, cancel := newTestAggregator()
	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeAggregator, aggregatorOffsetName).Return(&fftypes.Offset{
		Type:    fftypes.OffsetTypeAggregator,
		Name:    aggregatorOffsetName,
		Current: 12345,
		RowID:   333333,
	}, nil)
	mdi.On("GetPins", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Pin{}, nil, nil)
	ag.start()
	assert.Equal(t, int64(12345), ag.eventPoller.pollingOffset)
	ag.eventPoller.eventNotifier.newEvents <- 12345
	cancel()
	<-ag.eventPoller.closed
}

func TestProcessPinsDBGroupFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	rag := mdi.On("RunAsGroup", ag.ctx, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}
	mdi.On("GetBatchByID", ag.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, err := ag.processPinsDBGroup([]fftypes.LocallySequenced{
		&fftypes.Pin{
			Batch: fftypes.NewUUID(),
		},
	})
	assert.Regexp(t, "pop", err)
}

func TestGetPins(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetPins", ag.ctx, mock.Anything).Return([]*fftypes.Pin{
		{Sequence: 12345},
	}, nil, nil)

	lc, err := ag.getPins(ag.ctx, database.EventQueryFactory.NewFilter(ag.ctx).Gte("sequence", 12345))
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), lc[0].LocalSequence())
}

func TestProcessPinsMissingBatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	ba := &batchActions{}

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", ag.ctx, mock.Anything).Return(nil, nil)
	mdi.On("UpdateOffset", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := ag.processPins(ag.ctx, []*fftypes.Pin{
		{Sequence: 12345, Batch: fftypes.NewUUID()},
	}, ba)
	assert.NoError(t, err)

}

func TestProcessPinsMissingNoMsg(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	ba := &batchActions{}

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", ag.ctx, mock.Anything).Return(&fftypes.Batch{
		ID: fftypes.NewUUID(),
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}},
			},
		},
	}, nil)
	mdi.On("UpdateOffset", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := ag.processPins(ag.ctx, []*fftypes.Pin{
		{Sequence: 12345, Batch: fftypes.NewUUID(), Index: 25},
	}, ba)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)

}

func TestProcessPinsBadMsgHeader(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	ba := &batchActions{}

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", ag.ctx, mock.Anything).Return(&fftypes.Batch{
		ID: fftypes.NewUUID(),
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{
					ID:     nil, /* missing */
					Topics: fftypes.FFStringArray{"topic1"},
				}},
			},
		},
	}, nil)
	mdi.On("UpdateOffset", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := ag.processPins(ag.ctx, []*fftypes.Pin{
		{Sequence: 12345, Batch: fftypes.NewUUID(), Index: 0},
	}, ba)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)

}

func TestProcessSkipDupMsg(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	ba := &batchActions{}

	batchID := fftypes.NewUUID()
	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", ag.ctx, mock.Anything).Return(&fftypes.Batch{
		ID: batchID,
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{
					ID:     fftypes.NewUUID(),
					Topics: fftypes.FFStringArray{"topic1", "topic2"},
				}},
			},
		},
	}, nil).Once()
	mdi.On("GetPins", mock.Anything, mock.Anything).Return([]*fftypes.Pin{
		{Sequence: 1111}, // blocks the context
	}, nil, nil)
	mdi.On("UpdateOffset", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := ag.processPins(ag.ctx, []*fftypes.Pin{
		{Sequence: 12345, Batch: batchID, Index: 0, Hash: fftypes.NewRandB32()},
		{Sequence: 12345, Batch: batchID, Index: 1, Hash: fftypes.NewRandB32()},
	}, ba)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)

}

func TestProcessMsgFailGetPins(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	ba := &batchActions{}

	batchID := fftypes.NewUUID()
	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", ag.ctx, mock.Anything).Return(&fftypes.Batch{
		ID: batchID,
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{
					ID:     fftypes.NewUUID(),
					Topics: fftypes.FFStringArray{"topic1"},
				}},
			},
		},
	}, nil).Once()
	mdi.On("GetPins", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	err := ag.processPins(ag.ctx, []*fftypes.Pin{
		{Sequence: 12345, Batch: batchID, Index: 0, Hash: fftypes.NewRandB32()},
	}, ba)
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestProcessMsgFailMissingGroup(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	err := ag.processMessage(ag.ctx, &fftypes.Batch{}, true, 12345, &fftypes.Message{}, nil)
	assert.NoError(t, err)

}

func TestProcessMsgFailBadPin(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	err := ag.processMessage(ag.ctx, &fftypes.Batch{}, true, 12345, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:     fftypes.NewUUID(),
			Group:  fftypes.NewRandB32(),
			Topics: fftypes.FFStringArray{"topic1"},
		},
		Pins: fftypes.FFStringArray{"!Wrong"},
	}, nil)
	assert.NoError(t, err)

}

func TestProcessMsgFailGetNextPins(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	err := ag.processMessage(ag.ctx, &fftypes.Batch{}, true, 12345, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:     fftypes.NewUUID(),
			Group:  fftypes.NewRandB32(),
			Topics: fftypes.FFStringArray{"topic1"},
		},
		Pins: fftypes.FFStringArray{fftypes.NewRandB32().String()},
	}, nil)
	assert.EqualError(t, err, "pop")

}

func TestProcessMsgFailDispatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetPins", ag.ctx, mock.Anything).Return([]*fftypes.Pin{}, nil, nil)
	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return(nil, false, fmt.Errorf("pop"))

	err := ag.processMessage(ag.ctx, &fftypes.Batch{}, false, 12345, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:     fftypes.NewUUID(),
			Topics: fftypes.FFStringArray{"topic1"},
		},
		Pins: fftypes.FFStringArray{fftypes.NewRandB32().String()},
	}, nil)
	assert.EqualError(t, err, "pop")

}

func TestProcessMsgFailPinUpdate(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	ba := &batchActions{}
	pin := fftypes.NewRandB32()

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return([]*fftypes.NextPin{
		{Context: fftypes.NewRandB32(), Hash: pin},
	}, nil, nil)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(false, nil)
	mdi.On("InsertEvent", ag.ctx, mock.Anything).Return(nil)
	mdi.On("UpdateMessage", ag.ctx, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpdateNextPin", ag.ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := ag.processMessage(ag.ctx, &fftypes.Batch{}, true, 12345, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:     fftypes.NewUUID(),
			Group:  fftypes.NewRandB32(),
			Topics: fftypes.FFStringArray{"topic1"},
		},
		Pins: fftypes.FFStringArray{pin.String()},
	}, ba)
	assert.NoError(t, err)

	err = ba.RunFinalize(ag.ctx)
	assert.EqualError(t, err, "pop")

}

func TestCheckMaskedContextReadyMismatchedAuthor(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	pin := fftypes.NewRandB32()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return([]*fftypes.NextPin{
		{Context: fftypes.NewRandB32(), Hash: pin},
	}, nil, nil)

	_, err := ag.checkMaskedContextReady(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: fftypes.NewRandB32(),
			Identity: fftypes.Identity{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32())
	assert.NoError(t, err)

}

func TestAttemptContextInitGetGroupByIDFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, err := ag.attemptContextInit(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: fftypes.NewRandB32(),
			Identity: fftypes.Identity{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), fftypes.NewRandB32())
	assert.EqualError(t, err, "pop")

}

func TestAttemptContextInitGroupNotFound(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(nil, nil)

	_, err := ag.attemptContextInit(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: fftypes.NewRandB32(),
			Identity: fftypes.Identity{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), fftypes.NewRandB32())
	assert.NoError(t, err)

}

func TestAttemptContextInitAuthorMismatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	groupID := fftypes.NewRandB32()
	zeroHash := ag.calcHash("topic1", groupID, "author2", 0)
	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(&fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{
				{Identity: "author2"},
			},
		},
	}, nil)

	_, err := ag.attemptContextInit(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: groupID,
			Identity: fftypes.Identity{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), zeroHash)
	assert.NoError(t, err)

}

func TestAttemptContextInitNoMatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	groupID := fftypes.NewRandB32()
	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(&fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{
				{Identity: "author2"},
			},
		},
	}, nil)

	_, err := ag.attemptContextInit(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: groupID,
			Identity: fftypes.Identity{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), fftypes.NewRandB32())
	assert.NoError(t, err)

}

func TestAttemptContextInitGetPinsFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	groupID := fftypes.NewRandB32()
	zeroHash := ag.calcHash("topic1", groupID, "author1", 0)
	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	mdi := ag.database.(*databasemocks.Plugin)
	msh.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(&fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{
				{Identity: "author1"},
			},
		},
	}, nil)
	mdi.On("GetPins", ag.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	_, err := ag.attemptContextInit(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: groupID,
			Identity: fftypes.Identity{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), zeroHash)
	assert.EqualError(t, err, "pop")

}

func TestAttemptContextInitGetPinsBlocked(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	groupID := fftypes.NewRandB32()
	zeroHash := ag.calcHash("topic1", groupID, "author1", 0)
	mdi := ag.database.(*databasemocks.Plugin)
	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(&fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{
				{Identity: "author1"},
			},
		},
	}, nil)
	mdi.On("GetPins", ag.ctx, mock.Anything).Return([]*fftypes.Pin{
		{Sequence: 12345},
	}, nil, nil)

	np, err := ag.attemptContextInit(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: groupID,
			Identity: fftypes.Identity{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), zeroHash)
	assert.NoError(t, err)
	assert.Nil(t, np)

}

func TestAttemptContextInitInsertPinsFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	groupID := fftypes.NewRandB32()
	zeroHash := ag.calcHash("topic1", groupID, "author1", 0)
	mdi := ag.database.(*databasemocks.Plugin)
	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(&fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{
				{Identity: "author1"},
			},
		},
	}, nil)
	mdi.On("GetPins", ag.ctx, mock.Anything).Return([]*fftypes.Pin{}, nil, nil)
	mdi.On("InsertNextPin", ag.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	np, err := ag.attemptContextInit(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: groupID,
			Identity: fftypes.Identity{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), zeroHash)
	assert.Nil(t, np)
	assert.EqualError(t, err, "pop")

}

func TestAttemptMessageDispatchFailGetData(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return(nil, false, fmt.Errorf("pop"))

	_, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
	}, nil)
	assert.EqualError(t, err, "pop")

}

func TestAttemptMessageDispatchFailValidateData(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(false, fmt.Errorf("pop"))

	_, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID()},
		},
	}, nil)
	assert.EqualError(t, err, "pop")

}

func TestAttemptMessageDispatchMissingBlobs(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	blobHash := fftypes.NewRandB32()

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32(), Blob: &fftypes.BlobRef{
			Hash:   blobHash,
			Public: "public-ref",
		}},
	}, true, nil)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", ag.ctx, blobHash).Return(nil, nil)

	mdm.On("CopyBlobPStoDX", ag.ctx, mock.Anything).Return(nil, nil)

	dispatched, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
	}, nil)
	assert.NoError(t, err)
	assert.False(t, dispatched)

}

func TestAttemptMessageDispatchMissingTransfers(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetTokenTransfers", ag.ctx, mock.Anything).Return([]*fftypes.TokenTransfer{}, nil, nil)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:   fftypes.NewUUID(),
			Type: fftypes.MessageTypeTransferBroadcast,
		},
	}
	msg.Hash = msg.Header.Hash()
	dispatched, err := ag.attemptMessageDispatch(ag.ctx, msg, nil)
	assert.NoError(t, err)
	assert.False(t, dispatched)

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAttemptMessageDispatchGetTransfersFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetTokenTransfers", ag.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:   fftypes.NewUUID(),
			Type: fftypes.MessageTypeTransferBroadcast,
		},
	}
	msg.Hash = msg.Header.Hash()
	dispatched, err := ag.attemptMessageDispatch(ag.ctx, msg, nil)
	assert.EqualError(t, err, "pop")
	assert.False(t, dispatched)

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAttemptMessageDispatchTransferMismatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:   fftypes.NewUUID(),
			Type: fftypes.MessageTypeTransferBroadcast,
		},
	}
	msg.Hash = msg.Header.Hash()

	transfers := []*fftypes.TokenTransfer{{
		Message:     msg.Header.ID,
		MessageHash: fftypes.NewRandB32(),
	}}

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetTokenTransfers", ag.ctx, mock.Anything).Return(transfers, nil, nil)

	dispatched, err := ag.attemptMessageDispatch(ag.ctx, msg, nil)
	assert.NoError(t, err)
	assert.False(t, dispatched)

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAttemptMessageDispatchFailValidateBadSystem(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	ba := &batchActions{}

	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("HandleDefinitionBroadcast", mock.Anything, mock.Anything, mock.Anything).Return(definitions.ActionReject, &definitions.DefinitionBatchActions{}, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("UpdateMessage", ag.ctx, mock.Anything, mock.MatchedBy(func(u database.Update) bool {
		update, err := u.Finalize()
		assert.NoError(t, err)
		assert.Len(t, update.SetOperations, 2)

		assert.Equal(t, "confirmed", update.SetOperations[0].Field)
		v, err := update.SetOperations[0].Value.Value()
		assert.NoError(t, err)
		assert.Greater(t, v, int64(0))

		assert.Equal(t, "state", update.SetOperations[1].Field)
		v, err = update.SetOperations[1].Value.Value()
		assert.NoError(t, err)
		assert.Equal(t, "rejected", v)

		return true
	})).Return(nil)
	mdi.On("InsertEvent", ag.ctx, mock.Anything).Return(nil)

	_, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:      fftypes.MessageTypeDefinition,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID()},
		},
	}, ba)
	assert.NoError(t, err)

}

func TestAttemptMessageDispatchFailValidateSystemFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("HandleDefinitionBroadcast", mock.Anything, mock.Anything, mock.Anything).Return(definitions.ActionRetry, &definitions.DefinitionBatchActions{}, fmt.Errorf("pop"))

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)

	_, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:      fftypes.MessageTypeDefinition,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID()},
		},
	}, nil)
	assert.EqualError(t, err, "pop")

}

func TestAttemptMessageDispatchEventFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	ba := &batchActions{}

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	mdi.On("UpdateMessage", ag.ctx, mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", ag.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
	}, ba)
	assert.NoError(t, err)

	err = ba.RunFinalize(ag.ctx)
	assert.EqualError(t, err, "pop")

}

func TestAttemptMessageDispatchGroupInit(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	ba := &batchActions{}

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	mdi.On("UpdateMessage", ag.ctx, mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", ag.ctx, mock.Anything).Return(nil)

	_, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:   fftypes.NewUUID(),
			Type: fftypes.MessageTypeGroupInit,
		},
	}, ba)
	assert.NoError(t, err)

}

func TestAttemptMessageUpdateMessageFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	ba := &batchActions{}

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	mdi.On("UpdateMessage", ag.ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
	}, ba)
	assert.NoError(t, err)

	err = ba.RunFinalize(ag.ctx)
	assert.EqualError(t, err, "pop")

}

func TestRewindOffchainBatchesNoBatches(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("UpdateMessage", ag.ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	rewind, offset := ag.rewindOffchainBatches()
	assert.False(t, rewind)
	assert.Equal(t, int64(0), offset)
}

func TestRewindOffchainBatchesBatchesNoRewind(t *testing.T) {
	config.Set(config.EventAggregatorBatchSize, 10)

	ag, cancel := newTestAggregator()
	defer cancel()
	go ag.offchainListener()

	ag.offchainBatches <- fftypes.NewUUID()
	ag.offchainBatches <- fftypes.NewUUID()
	ag.offchainBatches <- fftypes.NewUUID()
	ag.offchainBatches <- fftypes.NewUUID()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetPins", ag.ctx, mock.Anything, mock.Anything).Return([]*fftypes.Pin{}, nil, nil)

	rewind, offset := ag.rewindOffchainBatches()
	assert.False(t, rewind)
	assert.Equal(t, int64(0), offset)
}

func TestRewindOffchainBatchesBatchesRewind(t *testing.T) {
	config.Set(config.EventAggregatorBatchSize, 10)

	ag, cancel := newTestAggregator()
	defer cancel()
	go ag.offchainListener()

	ag.offchainBatches <- fftypes.NewUUID()
	ag.offchainBatches <- fftypes.NewUUID()
	ag.offchainBatches <- fftypes.NewUUID()
	ag.offchainBatches <- fftypes.NewUUID()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetPins", ag.ctx, mock.Anything, mock.Anything).Return([]*fftypes.Pin{
		{Sequence: 12345},
	}, nil, nil)

	rewind, offset := ag.rewindOffchainBatches()
	assert.True(t, rewind)
	assert.Equal(t, int64(12344) /* one before the batch */, offset)
}

func TestRewindOffchainBatchesBatchesError(t *testing.T) {
	config.Set(config.EventAggregatorBatchSize, 10)

	ag, cancel := newTestAggregator()
	cancel()

	ag.queuedRewinds <- fftypes.NewUUID()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetPins", ag.ctx, mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	rewind, _ := ag.rewindOffchainBatches()
	assert.False(t, rewind)
}

func TestResolveBlobsNoop(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	resolved, err := ag.resolveBlobs(ag.ctx, []*fftypes.Data{
		{ID: fftypes.NewUUID(), Blob: &fftypes.BlobRef{}},
	})

	assert.NoError(t, err)
	assert.True(t, resolved)
}

func TestResolveBlobsErrorGettingHash(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", ag.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	resolved, err := ag.resolveBlobs(ag.ctx, []*fftypes.Data{
		{ID: fftypes.NewUUID(), Blob: &fftypes.BlobRef{
			Hash: fftypes.NewRandB32(),
		}},
	})

	assert.EqualError(t, err, "pop")
	assert.False(t, resolved)
}

func TestResolveBlobsNotFoundPrivate(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", ag.ctx, mock.Anything).Return(nil, nil)

	resolved, err := ag.resolveBlobs(ag.ctx, []*fftypes.Data{
		{ID: fftypes.NewUUID(), Blob: &fftypes.BlobRef{
			Hash: fftypes.NewRandB32(),
		}},
	})

	assert.NoError(t, err)
	assert.False(t, resolved)
}

func TestResolveBlobsFoundPrivate(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", ag.ctx, mock.Anything).Return(&fftypes.Blob{}, nil)

	resolved, err := ag.resolveBlobs(ag.ctx, []*fftypes.Data{
		{ID: fftypes.NewUUID(), Blob: &fftypes.BlobRef{
			Hash: fftypes.NewRandB32(),
		}},
	})

	assert.NoError(t, err)
	assert.True(t, resolved)
}

func TestResolveBlobsCopyNotFound(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", ag.ctx, mock.Anything).Return(nil, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("CopyBlobPStoDX", ag.ctx, mock.Anything).Return(nil, nil)

	resolved, err := ag.resolveBlobs(ag.ctx, []*fftypes.Data{
		{ID: fftypes.NewUUID(), Blob: &fftypes.BlobRef{
			Hash:   fftypes.NewRandB32(),
			Public: "public-ref",
		}},
	})

	assert.NoError(t, err)
	assert.False(t, resolved)
}

func TestResolveBlobsCopyFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", ag.ctx, mock.Anything).Return(nil, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("CopyBlobPStoDX", ag.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	resolved, err := ag.resolveBlobs(ag.ctx, []*fftypes.Data{
		{ID: fftypes.NewUUID(), Blob: &fftypes.BlobRef{
			Hash:   fftypes.NewRandB32(),
			Public: "public-ref",
		}},
	})

	assert.EqualError(t, err, "pop")
	assert.False(t, resolved)
}

func TestResolveBlobsCopyOk(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", ag.ctx, mock.Anything).Return(nil, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("CopyBlobPStoDX", ag.ctx, mock.Anything).Return(&fftypes.Blob{}, nil)

	resolved, err := ag.resolveBlobs(ag.ctx, []*fftypes.Data{
		{ID: fftypes.NewUUID(), Blob: &fftypes.BlobRef{
			Hash:   fftypes.NewRandB32(),
			Public: "public-ref",
		}},
	})

	assert.NoError(t, err)
	assert.True(t, resolved)
}
