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
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/definitionsmocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestAggregatorCommon(metrics bool) (*aggregator, func()) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	msh := &definitionsmocks.DefinitionHandlers{}
	mim := &identitymanagermocks.Manager{}
	mmi := &metricsmocks.Manager{}
	mbi := &blockchainmocks.Plugin{}
	if metrics {
		mmi.On("MessageConfirmed", mock.Anything, fftypes.EventTypeMessageConfirmed).Return()
	}
	mmi.On("IsMetricsEnabled").Return(metrics)
	mbi.On("VerifierType").Return(fftypes.VerifierTypeEthAddress)
	ctx, cancel := context.WithCancel(context.Background())
	ag := newAggregator(ctx, mdi, mbi, msh, mim, mdm, newEventNotifier(ctx, "ut"), mmi)
	return ag, cancel
}

func newTestAggregatorWithMetrics() (*aggregator, func()) {
	return newTestAggregatorCommon(true)
}

func newTestAggregator() (*aggregator, func()) {
	return newTestAggregatorCommon(false)
}

func TestAggregationMaskedZeroNonceMatch(t *testing.T) {

	ag, cancel := newTestAggregatorWithMetrics()
	defer cancel()
	bs := newBatchState(ag)

	// Generate some pin data
	member1org := newTestOrg("org1")
	member2org := newTestOrg("org2")
	member2key := "0x23456"
	topic := "some-topic"
	batchID := fftypes.NewUUID()
	groupID := fftypes.NewRandB32()
	msgID := fftypes.NewUUID()
	h := sha256.New()
	h.Write([]byte(topic))
	h.Write((*groupID)[:])
	contextUnmasked := fftypes.HashResult(h)
	initNPG := &nextPinGroupState{topic: topic, groupID: groupID}
	member1NonceZero := initNPG.calcPinHash(member1org.DID, 0)
	member2NonceZero := initNPG.calcPinHash(member2org.DID, 0)
	member2NonceOne := initNPG.calcPinHash(member2org.DID, 1)

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)
	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	mim := ag.identity.(*identitymanagermocks.Manager)

	mim.On("FindIdentityForVerifier", ag.ctx, []fftypes.IdentityType{fftypes.IdentityTypeOrg, fftypes.IdentityTypeCustom}, "ns1", &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeEthAddress,
		Value: member2key,
	}).Return(member2org, nil)

	// Get the batch
	mdi.On("GetBatchByID", ag.ctx, batchID).Return(&fftypes.Batch{
		ID: batchID,
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{
					Header: fftypes.MessageHeader{
						ID:        msgID,
						Group:     groupID,
						Namespace: "ns1",
						Topics:    []string{topic},
						SignerRef: fftypes.SignerRef{
							Author: member2org.DID,
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
				{Identity: member1org.DID},
				{Identity: member2org.DID},
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
		return *np.Hash == *member2NonceOne && np.Nonce == 1
	})).Return(nil).Once()
	// Validate the message is ok
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	// Insert the confirmed event
	mdi.On("InsertEvent", ag.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return *e.Reference == *msgID && e.Type == fftypes.EventTypeMessageConfirmed
	})).Return(nil)
	// Set the pin to dispatched
	mdi.On("UpdatePins", ag.ctx, mock.Anything, mock.Anything).Return(nil)
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
			Signer:     member2key,
			Dispatched: false,
		},
	}, bs)
	assert.NoError(t, err)

	err = bs.RunFinalize(ag.ctx)
	assert.NoError(t, err)

	assert.True(t, bs.IsPendingConfirm(msgID))

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestAggregationMaskedNextSequenceMatch(t *testing.T) {
	log.SetLevel("debug")

	ag, cancel := newTestAggregator()
	defer cancel()

	// Generate some pin data
	member1org := newTestOrg("org1")
	member2org := newTestOrg("org2")
	member2key := "0x12345"
	topic := "some-topic"
	batchID := fftypes.NewUUID()
	groupID := fftypes.NewRandB32()
	msgID := fftypes.NewUUID()
	h := sha256.New()
	h.Write([]byte(topic))
	h.Write((*groupID)[:])
	contextUnmasked := fftypes.HashResult(h)
	initNPG := &nextPinGroupState{topic: topic, groupID: groupID}
	member1Nonce100 := initNPG.calcPinHash(member1org.DID, 100)
	member2Nonce500 := initNPG.calcPinHash(member2org.DID, 500)
	member2Nonce501 := initNPG.calcPinHash(member2org.DID, 501)

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)
	mim := ag.identity.(*identitymanagermocks.Manager)

	mim.On("FindIdentityForVerifier", ag.ctx, []fftypes.IdentityType{fftypes.IdentityTypeOrg, fftypes.IdentityTypeCustom}, "ns1", &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeEthAddress,
		Value: member2key,
	}).Return(member2org, nil)

	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{a[1].(func(context.Context) error)(a[0].(context.Context))}
	}

	// Get the batch
	mdi.On("GetBatchByID", ag.ctx, batchID).Return(&fftypes.Batch{
		ID: batchID,
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{
					Header: fftypes.MessageHeader{
						ID:        msgID,
						Group:     groupID,
						Namespace: "ns1",
						Topics:    []string{topic},
						SignerRef: fftypes.SignerRef{
							Author: member2org.DID,
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
		{Context: contextUnmasked, Identity: member1org.DID, Hash: member1Nonce100, Nonce: 100, Sequence: 929},
		{Context: contextUnmasked, Identity: member2org.DID, Hash: member2Nonce500, Nonce: 500, Sequence: 424},
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
	mdi.On("UpdatePins", ag.ctx, mock.Anything, mock.Anything).Return(nil)
	// Update the message
	mdi.On("UpdateMessage", ag.ctx, mock.Anything, mock.Anything).Return(nil)
	// Confirm the offset
	mdi.On("UpdateOffset", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := ag.processPinsEventsHandler([]fftypes.LocallySequenced{
		&fftypes.Pin{
			Sequence:   10001,
			Masked:     true,
			Hash:       member2Nonce500,
			Batch:      batchID,
			Index:      0,
			Signer:     member2key,
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
	bs := newBatchState(ag)

	// Generate some pin data
	member1org := newTestOrg("org1")
	member1key := "0x12345"
	topic := "some-topic"
	batchID := fftypes.NewUUID()
	msgID := fftypes.NewUUID()
	h := sha256.New()
	h.Write([]byte(topic))
	contextUnmasked := fftypes.HashResult(h)

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)
	mim := ag.identity.(*identitymanagermocks.Manager)

	mim.On("FindIdentityForVerifier", ag.ctx, []fftypes.IdentityType{fftypes.IdentityTypeOrg, fftypes.IdentityTypeCustom}, "ns1", &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeEthAddress,
		Value: member1key,
	}).Return(member1org, nil)

	// Get the batch
	mdi.On("GetBatchByID", ag.ctx, batchID).Return(&fftypes.Batch{
		ID: batchID,
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{
					Header: fftypes.MessageHeader{
						ID:        msgID,
						Topics:    []string{topic},
						Namespace: "ns1",
						SignerRef: fftypes.SignerRef{
							Author: member1org.DID,
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
	mdi.On("UpdatePins", ag.ctx, mock.Anything, mock.Anything).Return(nil)
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
			Signer:     member1key,
			Dispatched: false,
		},
	}, bs)
	assert.NoError(t, err)

	err = bs.RunFinalize(ag.ctx)
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

	_, err := ag.processPinsEventsHandler([]fftypes.LocallySequenced{
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
	bs := newBatchState(ag)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", ag.ctx, mock.Anything).Return(nil, nil)
	mdi.On("UpdateOffset", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := ag.processPins(ag.ctx, []*fftypes.Pin{
		{Sequence: 12345, Batch: fftypes.NewUUID()},
	}, bs)
	assert.NoError(t, err)

}

func TestProcessPinsMissingNoMsg(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

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
	}, bs)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)

}

func TestProcessPinsBadMsgHeader(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

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
	}, bs)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)

}

func TestProcessSkipDupMsg(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

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
	}, bs)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)

}

func TestProcessMsgFailGetPins(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

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
	}, bs)
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestProcessMsgFailMissingGroup(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	err := ag.processMessage(ag.ctx, &fftypes.Batch{}, &fftypes.Pin{Masked: true, Sequence: 12345}, 10, &fftypes.Message{}, nil)
	assert.NoError(t, err)

}

func TestProcessMsgFailBadPin(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	err := ag.processMessage(ag.ctx, &fftypes.Batch{}, &fftypes.Pin{Masked: true, Sequence: 12345}, 10, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:     fftypes.NewUUID(),
			Group:  fftypes.NewRandB32(),
			Topics: fftypes.FFStringArray{"topic1"},
		},
		Pins: fftypes.FFStringArray{"!Wrong"},
	}, newBatchState(ag))
	assert.NoError(t, err)

}

func TestProcessMsgFailGetNextPins(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	err := ag.processMessage(ag.ctx, &fftypes.Batch{}, &fftypes.Pin{Masked: true, Sequence: 12345}, 10, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:     fftypes.NewUUID(),
			Group:  fftypes.NewRandB32(),
			Topics: fftypes.FFStringArray{"topic1"},
		},
		Pins: fftypes.FFStringArray{fftypes.NewRandB32().String()},
	}, newBatchState(ag))
	assert.EqualError(t, err, "pop")

}

func TestProcessMsgFailDispatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetPins", ag.ctx, mock.Anything).Return([]*fftypes.Pin{}, nil, nil)
	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return(nil, false, fmt.Errorf("pop"))

	err := ag.processMessage(ag.ctx, &fftypes.Batch{}, &fftypes.Pin{Sequence: 12345}, 10, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:     fftypes.NewUUID(),
			Topics: fftypes.FFStringArray{"topic1"},
		},
		Pins: fftypes.FFStringArray{fftypes.NewRandB32().String()},
	}, newBatchState(ag))
	assert.EqualError(t, err, "pop")

}

func TestProcessMsgFailPinUpdate(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)
	pin := fftypes.NewRandB32()
	org1 := newTestOrg("org1")

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)
	mim := ag.identity.(*identitymanagermocks.Manager)

	mim.On("FindIdentityForVerifier", ag.ctx, []fftypes.IdentityType{fftypes.IdentityTypeOrg, fftypes.IdentityTypeCustom}, "ns1", &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeEthAddress,
		Value: "0x12345",
	}).Return(org1, nil)
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return([]*fftypes.NextPin{
		{Context: fftypes.NewRandB32(), Hash: pin, Identity: org1.DID},
	}, nil, nil)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(false, nil)
	mdi.On("InsertEvent", ag.ctx, mock.Anything).Return(nil)
	mdi.On("UpdateMessage", ag.ctx, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpdateNextPin", ag.ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := ag.processMessage(ag.ctx, &fftypes.Batch{}, &fftypes.Pin{Masked: true, Sequence: 12345, Signer: "0x12345"}, 10, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Group:     fftypes.NewRandB32(),
			Topics:    fftypes.FFStringArray{"topic1"},
			Namespace: "ns1",
			SignerRef: fftypes.SignerRef{
				Author: org1.DID,
				Key:    "0x12345",
			},
		},
		Pins: fftypes.FFStringArray{pin.String()},
	}, bs)
	assert.NoError(t, err)

	err = bs.RunFinalize(ag.ctx)
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

	bs := newBatchState(ag)
	_, err := bs.CheckMaskedContextReady(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: fftypes.NewRandB32(),
			SignerRef: fftypes.SignerRef{
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

	bs := newBatchState(ag)
	_, err := bs.attemptContextInit(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: fftypes.NewRandB32(),
			SignerRef: fftypes.SignerRef{
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

	bs := newBatchState(ag)
	_, err := bs.attemptContextInit(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: fftypes.NewRandB32(),
			SignerRef: fftypes.SignerRef{
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
	initNPG := &nextPinGroupState{topic: "topic1", groupID: groupID}
	zeroHash := initNPG.calcPinHash("author2", 0)
	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(&fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{
				{Identity: "author2"},
			},
		},
	}, nil)

	bs := newBatchState(ag)
	_, err := bs.attemptContextInit(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: groupID,
			SignerRef: fftypes.SignerRef{
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

	bs := newBatchState(ag)
	_, err := bs.attemptContextInit(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: groupID,
			SignerRef: fftypes.SignerRef{
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
	initNPG := &nextPinGroupState{topic: "topic1", groupID: groupID}
	zeroHash := initNPG.calcPinHash("author1", 0)
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

	bs := newBatchState(ag)
	_, err := bs.attemptContextInit(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: groupID,
			SignerRef: fftypes.SignerRef{
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
	initNPG := &nextPinGroupState{topic: "topic1", groupID: groupID}
	zeroHash := initNPG.calcPinHash("author1", 0)
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

	bs := newBatchState(ag)
	np, err := bs.attemptContextInit(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: groupID,
			SignerRef: fftypes.SignerRef{
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
	initNPG := &nextPinGroupState{topic: "topic1", groupID: groupID}
	zeroHash := initNPG.calcPinHash("author1", 0)
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

	bs := newBatchState(ag)
	np, err := bs.attemptContextInit(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: groupID,
			SignerRef: fftypes.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), zeroHash)
	assert.NoError(t, err)
	assert.NotNil(t, np)
	err = bs.RunFinalize(ag.ctx)
	assert.EqualError(t, err, "pop")

}

func TestAttemptMessageDispatchFailGetData(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return(nil, false, fmt.Errorf("pop"))

	_, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
	}, nil, nil, nil)
	assert.EqualError(t, err, "pop")

}

func TestAttemptMessageDispatchFailValidateData(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdm := ag.data.(*datamocks.Manager)
	mim := ag.identity.(*identitymanagermocks.Manager)

	org1 := newTestOrg("org1")
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(false, fmt.Errorf("pop"))

	_, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID(), SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID}},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID()},
		},
	}, nil, nil, &fftypes.Pin{Signer: "0x12345"})
	assert.EqualError(t, err, "pop")

}

func TestAttemptMessageDispatchMissingBlobs(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	blobHash := fftypes.NewRandB32()

	mdm := ag.data.(*datamocks.Manager)
	mim := ag.identity.(*identitymanagermocks.Manager)

	org1 := newTestOrg("org1")
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)
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
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID(), SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID}},
	}, nil, nil, &fftypes.Pin{Signer: "0x12345"})
	assert.NoError(t, err)
	assert.False(t, dispatched)

}

func TestAttemptMessageDispatchMissingTransfers(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mim := ag.identity.(*identitymanagermocks.Manager)

	org1 := newTestOrg("org1")
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetTokenTransfers", ag.ctx, mock.Anything).Return([]*fftypes.TokenTransfer{}, nil, nil)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:   fftypes.NewUUID(),
			Type: fftypes.MessageTypeTransferBroadcast,
			SignerRef: fftypes.SignerRef{
				Author: org1.DID,
				Key:    "0x12345",
			},
		},
	}
	msg.Hash = msg.Header.Hash()
	dispatched, err := ag.attemptMessageDispatch(ag.ctx, msg, nil, nil, &fftypes.Pin{Signer: "0x12345"})
	assert.NoError(t, err)
	assert.False(t, dispatched)

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAttemptMessageDispatchGetTransfersFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mim := ag.identity.(*identitymanagermocks.Manager)

	org1 := newTestOrg("org1")
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetTokenTransfers", ag.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Type:      fftypes.MessageTypeTransferBroadcast,
			SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID},
		},
	}
	msg.Hash = msg.Header.Hash()
	dispatched, err := ag.attemptMessageDispatch(ag.ctx, msg, nil, nil, &fftypes.Pin{Signer: "0x12345"})
	assert.EqualError(t, err, "pop")
	assert.False(t, dispatched)

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAttemptMessageDispatchTransferMismatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	org1 := newTestOrg("org1")

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Type:      fftypes.MessageTypeTransferBroadcast,
			SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID},
		},
	}
	msg.Hash = msg.Header.Hash()

	transfers := []*fftypes.TokenTransfer{{
		Message:     msg.Header.ID,
		MessageHash: fftypes.NewRandB32(),
	}}

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetTokenTransfers", ag.ctx, mock.Anything).Return(transfers, nil, nil)

	dispatched, err := ag.attemptMessageDispatch(ag.ctx, msg, nil, nil, &fftypes.Pin{Signer: "0x12345"})
	assert.NoError(t, err)
	assert.False(t, dispatched)

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestDefinitionBroadcastActionReject(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	org1 := newTestOrg("org1")

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("HandleDefinitionBroadcast", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(definitions.ActionReject, nil)

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
			SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID},
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID()},
		},
	}, nil, bs, &fftypes.Pin{Signer: "0x12345"})
	assert.NoError(t, err)

}

func TestDefinitionBroadcastInvalidSigner(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	org1 := newTestOrg("org1")

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("HandleDefinitionBroadcast", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(definitions.ActionReject, nil)

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
			SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID},
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID()},
		},
	}, nil, bs, &fftypes.Pin{Signer: "0x12345"})
	assert.NoError(t, err)

}

func TestDispatchBroadcastQueuesLaterDispatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	org1 := newTestOrg("org1")

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, false, nil).Once()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetPins", ag.ctx, mock.Anything).Return([]*fftypes.Pin{}, nil, nil)

	msg1 := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:      fftypes.MessageTypeDefinition,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
			Topics:    fftypes.FFStringArray{"topic1"},
			SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID},
		},
	}
	msg2 := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:      fftypes.MessageTypeDefinition,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
			Topics:    fftypes.FFStringArray{"topic1"},
			SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID},
		},
	}

	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{msg1, msg2},
		},
	}

	// First message should dispatch
	err := ag.processMessage(ag.ctx, batch, &fftypes.Pin{Sequence: 12345}, 0, msg1, bs)
	assert.NoError(t, err)

	// Second message should not (mocks have Once limit on GetMessageData to confirm)
	err = ag.processMessage(ag.ctx, batch, &fftypes.Pin{Sequence: 12346}, 0, msg1, bs)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestDispatchPrivateQueuesLaterDispatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	org1 := newTestOrg("org1")

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, false, nil).Once()

	groupID := fftypes.NewRandB32()
	initNPG := &nextPinGroupState{topic: "topic1", groupID: groupID}
	member1NonceOne := initNPG.calcPinHash("org1", 1)
	member1NonceTwo := initNPG.calcPinHash("org1", 2)
	h := sha256.New()
	h.Write([]byte("topic1"))
	context := fftypes.HashResult(h)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return([]*fftypes.NextPin{
		{Context: context, Nonce: 1 /* match member1NonceOne */, Identity: org1.DID, Hash: member1NonceOne},
	}, nil, nil)

	msg1 := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:      fftypes.MessageTypePrivate,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
			Topics:    fftypes.FFStringArray{"topic1"},
			Group:     groupID,
			SignerRef: fftypes.SignerRef{
				Author: org1.DID,
			},
		},
		Pins: fftypes.FFStringArray{member1NonceOne.String()},
	}
	msg2 := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:      fftypes.MessageTypePrivate,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
			Topics:    fftypes.FFStringArray{"topic1"},
			Group:     groupID,
			SignerRef: fftypes.SignerRef{
				Author: org1.DID,
			},
		},
		Pins: fftypes.FFStringArray{member1NonceTwo.String()},
	}

	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{msg1, msg2},
		},
	}

	// First message should dispatch
	err := ag.processMessage(ag.ctx, batch, &fftypes.Pin{Masked: true, Sequence: 12345}, 0, msg1, bs)
	assert.NoError(t, err)

	// Second message should not (mocks have Once limit on GetMessageData to confirm)
	err = ag.processMessage(ag.ctx, batch, &fftypes.Pin{Masked: true, Sequence: 12346}, 0, msg2, bs)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestDispatchPrivateNextPinIncremented(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	org1 := newTestOrg("org1")

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil).Twice()

	groupID := fftypes.NewRandB32()
	initNPG := &nextPinGroupState{topic: "topic1", groupID: groupID}
	member1NonceOne := initNPG.calcPinHash(org1.DID, 1)
	member1NonceTwo := initNPG.calcPinHash(org1.DID, 2)
	h := sha256.New()
	h.Write([]byte("topic1"))
	context := fftypes.HashResult(h)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return([]*fftypes.NextPin{
		{Context: context, Nonce: 1 /* match member1NonceOne */, Identity: org1.DID, Hash: member1NonceOne},
	}, nil, nil)

	msg1 := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:      fftypes.MessageTypePrivate,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
			Topics:    fftypes.FFStringArray{"topic1"},
			Group:     groupID,
			SignerRef: fftypes.SignerRef{
				Author: org1.DID,
				Key:    "0x12345",
			},
		},
		Pins: fftypes.FFStringArray{member1NonceOne.String()},
	}
	msg2 := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:      fftypes.MessageTypePrivate,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
			Topics:    fftypes.FFStringArray{"topic1"},
			Group:     groupID,
			SignerRef: fftypes.SignerRef{
				Author: org1.DID,
				Key:    "0x12345",
			},
		},
		Pins: fftypes.FFStringArray{member1NonceTwo.String()},
	}

	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{msg1, msg2},
		},
	}

	// First message should dispatch
	err := ag.processMessage(ag.ctx, batch, &fftypes.Pin{Masked: true, Sequence: 12345, Signer: "0x12345"}, 0, msg1, bs)
	assert.NoError(t, err)

	// Second message should dispatch too (Twice on GetMessageData)
	err = ag.processMessage(ag.ctx, batch, &fftypes.Pin{Masked: true, Sequence: 12346, Signer: "0x12345"}, 0, msg2, bs)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestDefinitionBroadcastActionRetry(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	org1 := newTestOrg("org1")

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("HandleDefinitionBroadcast", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(definitions.ActionRetry, fmt.Errorf("pop"))

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)

	_, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:      fftypes.MessageTypeDefinition,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
			SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID},
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID()},
		},
	}, nil, nil, &fftypes.Pin{Signer: "0x12345"})
	assert.EqualError(t, err, "pop")

}

func TestDefinitionBroadcastRejectSignerLookupFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	org1 := newTestOrg("org1")

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))

	valid, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:      fftypes.MessageTypeDefinition,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
			SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID},
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID()},
		},
	}, nil, nil, &fftypes.Pin{Signer: "0x12345"})
	assert.Regexp(t, "pop", err)
	assert.False(t, valid)

	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestDefinitionBroadcastRejectSignerLookupWrongOrg(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	org1 := newTestOrg("org1")

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(newTestOrg("org2"), nil)

	valid, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:      fftypes.MessageTypeDefinition,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
			SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID},
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID()},
		},
	}, nil, nil, &fftypes.Pin{Signer: "0x12345"})
	assert.NoError(t, err)
	assert.False(t, valid)

	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestDefinitionBroadcastRejectBadSigner(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	org1 := newTestOrg("org1")

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)

	valid, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:      fftypes.MessageTypeDefinition,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
			SignerRef: fftypes.SignerRef{Key: "0x23456", Author: org1.DID},
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID()},
		},
	}, nil, nil, &fftypes.Pin{Signer: "0x12345"})
	assert.NoError(t, err)
	assert.False(t, valid)

}

func TestDefinitionBroadcastRejectUnregisteredSignerIdentityClaim(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	org1 := newTestOrg("org1")

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("HandleDefinitionBroadcast", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(definitions.ActionWait, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)

	valid, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:      fftypes.MessageTypeDefinition,
			Tag:       fftypes.SystemTagIdentityClaim,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
			SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID},
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID()},
		},
	}, nil, nil, &fftypes.Pin{Signer: "0x12345"})
	assert.NoError(t, err)
	assert.False(t, valid)

	mim.AssertExpectations(t)
	msh.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestDefinitionBroadcastActionWait(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	org1 := newTestOrg("org1")

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	msh := ag.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("HandleDefinitionBroadcast", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(definitions.ActionWait, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)

	_, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:      fftypes.MessageTypeDefinition,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
			SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID},
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID()},
		},
	}, nil, nil, &fftypes.Pin{Signer: "0x12345"})
	assert.NoError(t, err)

}

func TestAttemptMessageDispatchEventFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)
	org1 := newTestOrg("org1")

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)
	mim := ag.identity.(*identitymanagermocks.Manager)

	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	mdi.On("UpdateMessage", ag.ctx, mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", ag.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID(), SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID}},
	}, nil, bs, &fftypes.Pin{Signer: "0x12345"})
	assert.NoError(t, err)

	err = bs.RunFinalize(ag.ctx)
	assert.EqualError(t, err, "pop")

}

func TestAttemptMessageDispatchGroupInit(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)
	org1 := newTestOrg("org1")

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)
	mim := ag.identity.(*identitymanagermocks.Manager)

	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	mdi.On("UpdateMessage", ag.ctx, mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", ag.ctx, mock.Anything).Return(nil)

	_, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Type:      fftypes.MessageTypeGroupInit,
			SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID},
		},
	}, nil, bs, &fftypes.Pin{Signer: "0x12345"})
	assert.NoError(t, err)

}

func TestAttemptMessageUpdateMessageFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)
	org1 := newTestOrg("org1")

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)
	mim := ag.identity.(*identitymanagermocks.Manager)

	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	mdi.On("UpdateMessage", ag.ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := ag.attemptMessageDispatch(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID(), SignerRef: fftypes.SignerRef{Key: "0x12345", Author: org1.DID}},
	}, nil, bs, &fftypes.Pin{Signer: "0x12345"})
	assert.NoError(t, err)

	err = bs.RunFinalize(ag.ctx)
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
	go ag.batchRewindListener()

	ag.rewindBatches <- fftypes.NewUUID()
	ag.rewindBatches <- fftypes.NewUUID()
	ag.rewindBatches <- fftypes.NewUUID()
	ag.rewindBatches <- fftypes.NewUUID()

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
	go ag.batchRewindListener()

	ag.rewindBatches <- fftypes.NewUUID()
	ag.rewindBatches <- fftypes.NewUUID()
	ag.rewindBatches <- fftypes.NewUUID()
	ag.rewindBatches <- fftypes.NewUUID()

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

func TestBatchActions(t *testing.T) {
	prefinalizeCalled := false
	finalizeCalled := false

	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	bs.AddPreFinalize(func(ctx context.Context) error {
		prefinalizeCalled = true
		return nil
	})
	bs.AddFinalize(func(ctx context.Context) error {
		finalizeCalled = true
		return nil
	})

	err := bs.RunPreFinalize(context.Background())
	assert.NoError(t, err)
	assert.True(t, prefinalizeCalled)

	err = bs.RunFinalize(context.Background())
	assert.NoError(t, err)
	assert.True(t, finalizeCalled)
}

func TestBatchActionsError(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	bs.AddPreFinalize(func(ctx context.Context) error {
		return fmt.Errorf("pop")
	})
	bs.AddFinalize(func(ctx context.Context) error {
		return fmt.Errorf("pop")
	})

	err := bs.RunPreFinalize(context.Background())
	assert.EqualError(t, err, "pop")

	err = bs.RunFinalize(context.Background())
	assert.EqualError(t, err, "pop")
}

func TestProcessWithBatchActionsPreFinalizeError(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{a[1].(func(context.Context) error)(a[0].(context.Context))}
	}

	err := ag.processWithBatchState(func(ctx context.Context, actions *batchState) error {
		actions.AddPreFinalize(func(ctx context.Context) error { return fmt.Errorf("pop") })
		return nil
	})
	assert.EqualError(t, err, "pop")
}

func TestProcessWithBatchActionsSuccess(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{a[1].(func(context.Context) error)(a[0].(context.Context))}
	}

	err := ag.processWithBatchState(func(ctx context.Context, actions *batchState) error {
		actions.AddPreFinalize(func(ctx context.Context) error { return nil })
		actions.AddFinalize(func(ctx context.Context) error { return nil })
		return nil
	})
	assert.NoError(t, err)
}
