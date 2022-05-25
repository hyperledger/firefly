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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/definitions"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/definitionsmocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestAggregatorCommon(metrics bool) (*aggregator, func()) {
	coreconfig.Reset()
	logrus.SetLevel(logrus.DebugLevel)
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mpm := &privatemessagingmocks.Manager{}
	msh := &definitionsmocks.DefinitionHandler{}
	mim := &identitymanagermocks.Manager{}
	mmi := &metricsmocks.Manager{}
	mbi := &blockchainmocks.Plugin{}
	if metrics {
		mmi.On("MessageConfirmed", mock.Anything, core.EventTypeMessageConfirmed).Return()
	}
	mmi.On("IsMetricsEnabled").Return(metrics)
	mbi.On("VerifierType").Return(core.VerifierTypeEthAddress)
	ctx, cancel := context.WithCancel(context.Background())
	ag := newAggregator(ctx, mdi, mbi, mpm, msh, mim, mdm, newEventNotifier(ctx, "ut"), mmi)
	return ag, func() {
		cancel()
		ag.batchCache.Stop()
	}
}

func newTestAggregatorWithMetrics() (*aggregator, func()) {
	return newTestAggregatorCommon(true)
}

func newTestAggregator() (*aggregator, func()) {
	return newTestAggregatorCommon(false)
}

func newTestManifest(mType core.MessageType, groupID *fftypes.Bytes32) (*core.Message, *core.Message, *core.Identity, *core.BatchManifest) {
	org1 := newTestOrg("org1")

	msg1 := &core.Message{
		Header: core.MessageHeader{
			Type:      mType,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
			Group:     groupID,
			Topics:    core.FFStringArray{"topic1"},
			SignerRef: core.SignerRef{Key: "0x12345", Author: org1.DID},
		},
		Data: core.DataRefs{
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
		},
	}
	msg2 := &core.Message{
		Header: core.MessageHeader{
			Type:      mType,
			ID:        fftypes.NewUUID(),
			Group:     groupID,
			Namespace: "any",
			Topics:    core.FFStringArray{"topic1"},
			SignerRef: core.SignerRef{Key: "0x12345", Author: org1.DID},
		},
		Data: core.DataRefs{
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
		},
	}

	return msg1, msg2, org1, &core.BatchManifest{
		Version: 1,
		ID:      fftypes.NewUUID(),
		TX: core.TransactionRef{
			Type: core.TransactionTypeBatchPin,
			ID:   fftypes.NewUUID(),
		},
		Messages: []*core.MessageManifestEntry{
			{
				MessageRef: core.MessageRef{
					ID:   msg1.Header.ID,
					Hash: msg1.Hash,
				},
				Topics: len(msg1.Header.Topics),
			},
			{
				MessageRef: core.MessageRef{
					ID:   msg2.Header.ID,
					Hash: msg2.Hash,
				},
				Topics: len(msg2.Header.Topics),
			},
		},
	}
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
	mpm := ag.messaging.(*privatemessagingmocks.Manager)
	mim := ag.identity.(*identitymanagermocks.Manager)

	mim.On("FindIdentityForVerifier", ag.ctx, []core.IdentityType{core.IdentityTypeOrg, core.IdentityTypeCustom}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: member2key,
	}).Return(member2org, nil)

	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: batchID,
		},
		Payload: core.BatchPayload{
			Messages: []*core.Message{
				{
					Header: core.MessageHeader{
						ID:        msgID,
						Group:     groupID,
						Namespace: "ns1",
						Topics:    []string{topic},
						SignerRef: core.SignerRef{
							Author: member2org.DID,
							Key:    member2key,
						},
					},
					Pins: []string{fmt.Sprintf("%s:%.9d", member2NonceZero, 0)},
					Data: core.DataRefs{
						{ID: fftypes.NewUUID()},
					},
				},
			},
		},
	}
	bp, _ := batch.Confirmed()

	// Get the batch
	mdi.On("GetBatchByID", ag.ctx, batchID).Return(bp, nil)
	// Look for existing nextpins - none found, first on context
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return([]*core.NextPin{}, nil, nil).Once()
	// Get the group members
	mpm.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(&core.Group{
		GroupIdentity: core.GroupIdentity{
			Members: core.Members{
				{Identity: member1org.DID},
				{Identity: member2org.DID},
			},
		},
	}, nil)
	// Look for any earlier pins - none found
	mdi.On("GetPins", ag.ctx, mock.Anything).Return([]*core.Pin{}, nil, nil).Once()
	// Insert all the zero pins
	mdi.On("InsertNextPin", ag.ctx, mock.MatchedBy(func(np *core.NextPin) bool {
		assert.Equal(t, *np.Context, *contextUnmasked)
		np.Sequence = 10011
		return *np.Hash == *member1NonceZero && np.Nonce == 0
	})).Return(nil).Once()
	mdi.On("InsertNextPin", ag.ctx, mock.MatchedBy(func(np *core.NextPin) bool {
		assert.Equal(t, *np.Context, *contextUnmasked)
		np.Sequence = 10012
		return *np.Hash == *member2NonceOne && np.Nonce == 1
	})).Return(nil).Once()
	// Validate the message is ok
	mdm.On("GetMessageWithDataCached", ag.ctx, batch.Payload.Messages[0].Header.ID, data.CRORequirePins).Return(batch.Payload.Messages[0], core.DataArray{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	mdm.On("UpdateMessageStateIfCached", ag.ctx, mock.Anything, core.MessageStateConfirmed, mock.Anything).Return()
	// Insert the confirmed event
	mdi.On("InsertEvent", ag.ctx, mock.MatchedBy(func(e *core.Event) bool {
		return *e.Reference == *msgID && e.Type == core.EventTypeMessageConfirmed
	})).Return(nil)
	// Set the pin to dispatched
	mdi.On("UpdatePins", ag.ctx, mock.Anything, mock.Anything).Return(nil)
	// Update the message
	mdi.On("UpdateMessages", ag.ctx, mock.Anything, mock.MatchedBy(func(u database.Update) bool {
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

	err := ag.processPins(ag.ctx, []*core.Pin{
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

	assert.NotNil(t, bs.GetPendingConfirm()[*msgID])

	// Confirm the offset
	assert.Equal(t, int64(10001), <-ag.eventPoller.offsetCommitted)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mpm.AssertExpectations(t)
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

	mim.On("FindIdentityForVerifier", ag.ctx, []core.IdentityType{core.IdentityTypeOrg, core.IdentityTypeCustom}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: member2key,
	}).Return(member2org, nil)

	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{a[1].(func(context.Context) error)(a[0].(context.Context))}
	}

	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: batchID,
		},
		Payload: core.BatchPayload{
			Messages: []*core.Message{
				{
					Header: core.MessageHeader{
						ID:        msgID,
						Group:     groupID,
						Namespace: "ns1",
						Topics:    []string{topic},
						SignerRef: core.SignerRef{
							Author: member2org.DID,
							Key:    member2key,
						},
					},
					Pins: []string{member2Nonce500.String()},
					Data: core.DataRefs{
						{ID: fftypes.NewUUID()},
					},
				},
			},
		},
	}
	bp, _ := batch.Confirmed()

	// Get the batch
	mdi.On("GetBatchByID", ag.ctx, batchID).Return(bp, nil)
	// Look for existing nextpins - none found, first on context
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return([]*core.NextPin{
		{Context: contextUnmasked, Identity: member1org.DID, Hash: member1Nonce100, Nonce: 100, Sequence: 929},
		{Context: contextUnmasked, Identity: member2org.DID, Hash: member2Nonce500, Nonce: 500, Sequence: 424},
	}, nil, nil).Once()
	// Validate the message is ok
	mdm.On("GetMessageWithDataCached", ag.ctx, batch.Payload.Messages[0].Header.ID, data.CRORequirePins).Return(batch.Payload.Messages[0], core.DataArray{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	mdm.On("UpdateMessageStateIfCached", ag.ctx, mock.Anything, core.MessageStateConfirmed, mock.Anything).Return()
	// Insert the confirmed event
	mdi.On("InsertEvent", ag.ctx, mock.MatchedBy(func(e *core.Event) bool {
		return *e.Reference == *msgID && e.Type == core.EventTypeMessageConfirmed
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
	mdi.On("UpdateMessages", ag.ctx, mock.Anything, mock.Anything).Return(nil)

	_, err := ag.processPinsEventsHandler([]core.LocallySequenced{
		&core.Pin{
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

	// Confirm the offset
	assert.Equal(t, int64(10001), <-ag.eventPoller.offsetCommitted)

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

	mim.On("FindIdentityForVerifier", ag.ctx, []core.IdentityType{core.IdentityTypeOrg, core.IdentityTypeCustom}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: member1key,
	}).Return(member1org, nil)

	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: batchID,
		},
		Payload: core.BatchPayload{
			Messages: []*core.Message{
				{
					Header: core.MessageHeader{
						ID:        msgID,
						Topics:    []string{topic},
						Namespace: "ns1",
						SignerRef: core.SignerRef{
							Author: member1org.DID,
							Key:    member1key,
						},
					},
					Data: core.DataRefs{
						{ID: fftypes.NewUUID()},
					},
				},
			},
		},
	}
	bp, _ := batch.Confirmed()

	// Get the batch
	mdi.On("GetBatchByID", ag.ctx, batchID).Return(bp, nil)
	// Do not resolve any pins earlier
	mdi.On("GetPins", mock.Anything, mock.Anything).Return([]*core.Pin{}, nil, nil)
	// Validate the message is ok
	mdm.On("GetMessageWithDataCached", ag.ctx, batch.Payload.Messages[0].Header.ID, data.CRORequirePublicBlobRefs).Return(batch.Payload.Messages[0], core.DataArray{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	mdm.On("UpdateMessageStateIfCached", ag.ctx, mock.Anything, core.MessageStateConfirmed, mock.Anything).Return()
	// Insert the confirmed event
	mdi.On("InsertEvent", ag.ctx, mock.MatchedBy(func(e *core.Event) bool {
		return *e.Reference == *msgID && e.Type == core.EventTypeMessageConfirmed
	})).Return(nil)
	// Set the pin to dispatched
	mdi.On("UpdatePins", ag.ctx, mock.Anything, mock.Anything).Return(nil)
	// Update the message
	mdi.On("UpdateMessages", ag.ctx, mock.Anything, mock.Anything).Return(nil)

	err := ag.processPins(ag.ctx, []*core.Pin{
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

	// Confirm the offset
	assert.Equal(t, int64(10001), <-ag.eventPoller.offsetCommitted)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestAggregationMigratedBroadcast(t *testing.T) {

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

	mim.On("FindIdentityForVerifier", ag.ctx, []core.IdentityType{core.IdentityTypeOrg, core.IdentityTypeCustom}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: member1key,
	}).Return(member1org, nil)

	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: batchID,
		},
		Payload: core.BatchPayload{
			Messages: []*core.Message{
				{
					Header: core.MessageHeader{
						ID:        msgID,
						Topics:    []string{topic},
						Namespace: "ns1",
						SignerRef: core.SignerRef{
							Author: member1org.DID,
							Key:    member1key,
						},
					},
					Data: core.DataRefs{
						{ID: fftypes.NewUUID()},
					},
				},
			},
		},
	}
	payloadBinary, err := json.Marshal(&batch.Payload)
	assert.NoError(t, err)
	bp := &core.BatchPersisted{
		TX:          batch.Payload.TX,
		BatchHeader: batch.BatchHeader,
		Manifest:    fftypes.JSONAnyPtr(string(payloadBinary)),
	}

	// Get the batch
	mdi.On("GetBatchByID", ag.ctx, batchID).Return(bp, nil)
	// Do not resolve any pins earlier
	mdi.On("GetPins", mock.Anything, mock.Anything).Return([]*core.Pin{}, nil, nil)
	// Validate the message is ok
	mdm.On("GetMessageWithDataCached", ag.ctx, batch.Payload.Messages[0].Header.ID, data.CRORequirePublicBlobRefs).Return(batch.Payload.Messages[0], core.DataArray{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	mdm.On("UpdateMessageStateIfCached", ag.ctx, mock.Anything, core.MessageStateConfirmed, mock.Anything).Return()
	// Insert the confirmed event
	mdi.On("InsertEvent", ag.ctx, mock.MatchedBy(func(e *core.Event) bool {
		return *e.Reference == *msgID && e.Type == core.EventTypeMessageConfirmed
	})).Return(nil)
	// Set the pin to dispatched
	mdi.On("UpdatePins", ag.ctx, mock.Anything, mock.Anything).Return(nil)
	// Update the message
	mdi.On("UpdateMessages", ag.ctx, mock.Anything, mock.Anything).Return(nil)

	err = ag.processPins(ag.ctx, []*core.Pin{
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

	// Confirm the offset
	assert.Equal(t, int64(10001), <-ag.eventPoller.offsetCommitted)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestAggregationMigratedBroadcastNilMessageID(t *testing.T) {

	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	// Generate some pin data
	member1org := newTestOrg("org1")
	member1key := "0x12345"
	topic := "some-topic"
	batchID := fftypes.NewUUID()
	h := sha256.New()
	h.Write([]byte(topic))
	contextUnmasked := fftypes.HashResult(h)

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)
	mim := ag.identity.(*identitymanagermocks.Manager)

	mim.On("FindIdentityForVerifier", ag.ctx, []core.IdentityType{core.IdentityTypeOrg, core.IdentityTypeCustom}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: member1key,
	}).Return(member1org, nil)

	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: batchID,
		},
		Payload: core.BatchPayload{
			Messages: []*core.Message{{
				Header: core.MessageHeader{
					Topics: core.FFStringArray{"topic1"},
				},
			}},
		},
	}
	payloadBinary, err := json.Marshal(&batch.Payload)
	assert.NoError(t, err)
	bp := &core.BatchPersisted{
		TX:          batch.Payload.TX,
		BatchHeader: batch.BatchHeader,
		Manifest:    fftypes.JSONAnyPtr(string(payloadBinary)),
	}

	mdi.On("GetBatchByID", ag.ctx, batchID).Return(bp, nil)

	err = ag.processPins(ag.ctx, []*core.Pin{
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

	// Confirm the offset
	assert.Equal(t, int64(10001), <-ag.eventPoller.offsetCommitted)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestAggregationMigratedBroadcastInvalid(t *testing.T) {

	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	// Generate some pin data
	member1org := newTestOrg("org1")
	member1key := "0x12345"
	topic := "some-topic"
	batchID := fftypes.NewUUID()
	h := sha256.New()
	h.Write([]byte(topic))
	contextUnmasked := fftypes.HashResult(h)

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)
	mim := ag.identity.(*identitymanagermocks.Manager)

	mim.On("FindIdentityForVerifier", ag.ctx, []core.IdentityType{core.IdentityTypeOrg, core.IdentityTypeCustom}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: member1key,
	}).Return(member1org, nil)

	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: batchID,
		},
		Payload: core.BatchPayload{
			Messages: []*core.Message{{
				Header: core.MessageHeader{
					Topics: core.FFStringArray{"topic1"},
				},
			}},
		},
	}
	bp := &core.BatchPersisted{
		TX:          batch.Payload.TX,
		BatchHeader: batch.BatchHeader,
		Manifest:    fftypes.JSONAnyPtr("{}"),
	}

	mdi.On("GetBatchByID", ag.ctx, batchID).Return(bp, nil)

	err := ag.processPins(ag.ctx, []*core.Pin{
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

	// Confirm the offset
	assert.Equal(t, int64(10001), <-ag.eventPoller.offsetCommitted)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestShutdownOnCancel(t *testing.T) {
	ag, cancel := newTestAggregator()
	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetOffset", mock.Anything, core.OffsetTypeAggregator, aggregatorOffsetName).Return(&core.Offset{
		Type:    core.OffsetTypeAggregator,
		Name:    aggregatorOffsetName,
		Current: 12345,
		RowID:   333333,
	}, nil)
	mdi.On("GetPins", mock.Anything, mock.Anything, mock.Anything).Return([]*core.Pin{}, nil, nil)
	ag.start()
	assert.Equal(t, int64(12345), ag.eventPoller.pollingOffset)
	ag.eventPoller.eventNotifier.newEvents <- 12345
	cancel()
	<-ag.eventPoller.closed
	<-ag.rewinder.loop1Done
	<-ag.rewinder.loop2Done
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

	_, err := ag.processPinsEventsHandler([]core.LocallySequenced{
		&core.Pin{
			Batch: fftypes.NewUUID(),
		},
	})
	assert.Regexp(t, "pop", err)
}

func TestGetPins(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetPins", ag.ctx, mock.Anything).Return([]*core.Pin{
		{Sequence: 12345},
	}, nil, nil)

	lc, err := ag.getPins(ag.ctx, database.EventQueryFactory.NewFilter(ag.ctx).Gte("sequence", 12345), 12345)
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), lc[0].LocalSequence())
}

func TestProcessPinsMissingBatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", ag.ctx, mock.Anything).Return(nil, nil)

	err := ag.processPins(ag.ctx, []*core.Pin{
		{Sequence: 12345, Batch: fftypes.NewUUID()},
	}, bs)
	assert.NoError(t, err)

	// Confirm the offset
	assert.Equal(t, int64(12345), <-ag.eventPoller.offsetCommitted)

}

func TestProcessPinsMissingNoMsg(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: fftypes.NewUUID(),
		},
		Payload: core.BatchPayload{
			Messages: []*core.Message{
				{Header: core.MessageHeader{ID: fftypes.NewUUID()}},
			},
		},
	}
	bp, _ := batch.Confirmed()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", ag.ctx, mock.Anything).Return(bp, nil)

	err := ag.processPins(ag.ctx, []*core.Pin{
		{Sequence: 12345, Batch: fftypes.NewUUID(), Index: 25},
	}, bs)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)

	// Confirm the offset
	assert.Equal(t, int64(12345), <-ag.eventPoller.offsetCommitted)

}

func TestProcessPinsBadMsgHeader(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: fftypes.NewUUID(),
		},
		Payload: core.BatchPayload{
			Messages: []*core.Message{
				{Header: core.MessageHeader{
					ID:     nil, /* missing */
					Topics: core.FFStringArray{"topic1"},
				}},
			},
		},
	}
	bp, _ := batch.Confirmed()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", ag.ctx, mock.Anything).Return(bp, nil)

	err := ag.processPins(ag.ctx, []*core.Pin{
		{Sequence: 12345, Batch: fftypes.NewUUID(), Index: 0},
	}, bs)
	assert.NoError(t, err)

	// Confirm the offset
	assert.Equal(t, int64(12345), <-ag.eventPoller.offsetCommitted)

	mdi.AssertExpectations(t)

}

func TestProcessSkipDupMsg(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	batchID := fftypes.NewUUID()
	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: batchID,
		},
		Payload: core.BatchPayload{
			Messages: []*core.Message{
				{Header: core.MessageHeader{
					ID:     fftypes.NewUUID(),
					Topics: core.FFStringArray{"topic1", "topic2"},
				}},
			},
		},
	}
	bp, _ := batch.Confirmed()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", ag.ctx, mock.Anything).Return(bp, nil).Once()
	mdi.On("GetPins", mock.Anything, mock.Anything).Return([]*core.Pin{
		{Sequence: 1111}, // blocks the context
	}, nil, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", ag.ctx, mock.Anything, data.CRORequirePublicBlobRefs).Return(batch.Payload.Messages[0], nil, true, nil)

	err := ag.processPins(ag.ctx, []*core.Pin{
		{Sequence: 12345, Batch: batchID, Index: 0, Hash: fftypes.NewRandB32()},
		{Sequence: 12345, Batch: batchID, Index: 1, Hash: fftypes.NewRandB32()},
	}, bs)
	assert.NoError(t, err)

	// Confirm the offset
	assert.Equal(t, int64(12345), <-ag.eventPoller.offsetCommitted)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)

}

func TestProcessMsgFailGetPins(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	batchID := fftypes.NewUUID()
	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: batchID,
		},
		Payload: core.BatchPayload{
			Messages: []*core.Message{
				{Header: core.MessageHeader{
					ID:     fftypes.NewUUID(),
					Topics: core.FFStringArray{"topic1"},
				}},
			},
		},
	}
	bp, _ := batch.Confirmed()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", ag.ctx, mock.Anything).Return(bp, nil).Once()
	mdi.On("GetPins", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", ag.ctx, mock.Anything, data.CRORequirePublicBlobRefs).Return(batch.Payload.Messages[0], nil, true, nil)

	err := ag.processPins(ag.ctx, []*core.Pin{
		{Sequence: 12345, Batch: batchID, Index: 0, Hash: fftypes.NewRandB32()},
	}, bs)
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestProcessMsgFailData(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", ag.ctx, mock.Anything, data.CRORequirePins).Return(nil, nil, false, fmt.Errorf("pop"))

	err := ag.processMessage(ag.ctx, &core.BatchManifest{}, &core.Pin{Masked: true, Sequence: 12345}, 10, &core.MessageManifestEntry{}, nil)
	assert.Regexp(t, "pop", err)

	mdm.AssertExpectations(t)
}

func TestProcessMsgFailMissingData(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", ag.ctx, mock.Anything, data.CRORequirePins).Return(&core.Message{Header: core.MessageHeader{ID: fftypes.NewUUID()}}, nil, false, nil)

	err := ag.processMessage(ag.ctx, &core.BatchManifest{}, &core.Pin{Masked: true, Sequence: 12345}, 10, &core.MessageManifestEntry{}, nil)
	assert.NoError(t, err)

	mdm.AssertExpectations(t)
}

func TestProcessMsgFailMissingGroup(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", ag.ctx, mock.Anything, data.CRORequirePins).Return(&core.Message{Header: core.MessageHeader{ID: fftypes.NewUUID()}}, nil, true, nil)

	err := ag.processMessage(ag.ctx, &core.BatchManifest{}, &core.Pin{Masked: true, Sequence: 12345}, 10, &core.MessageManifestEntry{}, nil)
	assert.NoError(t, err)

	mdm.AssertExpectations(t)
}

func TestProcessMsgFailBadPin(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	msg := &core.Message{
		Header: core.MessageHeader{
			ID:     fftypes.NewUUID(),
			Group:  fftypes.NewRandB32(),
			Topics: core.FFStringArray{"topic1"},
		},
		Hash: fftypes.NewRandB32(),
		Pins: core.FFStringArray{"!Wrong"},
	}

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", ag.ctx, mock.Anything, data.CRORequirePins).Return(msg, nil, true, nil)

	err := ag.processMessage(ag.ctx, &core.BatchManifest{}, &core.Pin{Masked: true, Sequence: 12345}, 10, &core.MessageManifestEntry{
		MessageRef: core.MessageRef{
			ID:   msg.Header.ID,
			Hash: msg.Hash,
		},
		Topics: len(msg.Header.Topics),
	}, newBatchState(ag))
	assert.NoError(t, err)

	mdm.AssertExpectations(t)

}

func TestProcessMsgFailGetNextPins(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	msg := &core.Message{
		Header: core.MessageHeader{
			ID:     fftypes.NewUUID(),
			Group:  fftypes.NewRandB32(),
			Topics: core.FFStringArray{"topic1"},
		},
		Pins: core.FFStringArray{fftypes.NewRandB32().String()},
	}

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", ag.ctx, mock.Anything, data.CRORequirePins).Return(msg, nil, true, nil)

	err := ag.processMessage(ag.ctx, &core.BatchManifest{}, &core.Pin{Masked: true, Sequence: 12345}, 10, &core.MessageManifestEntry{
		MessageRef: core.MessageRef{
			ID:   msg.Header.ID,
			Hash: msg.Hash,
		},
		Topics: len(msg.Header.Topics),
	}, newBatchState(ag))
	assert.EqualError(t, err, "pop")

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestProcessMsgFailDispatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetPins", ag.ctx, mock.Anything).Return([]*core.Pin{}, nil, nil)

	msg := &core.Message{
		Header: core.MessageHeader{
			ID:     fftypes.NewUUID(),
			Topics: core.FFStringArray{"topic1"},
			SignerRef: core.SignerRef{
				Key: "0x12345",
			},
		},
		Pins: core.FFStringArray{fftypes.NewRandB32().String()},
	}

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", ag.ctx, mock.Anything, data.CRORequirePublicBlobRefs).Return(msg, nil, true, nil)

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := ag.processMessage(ag.ctx, &core.BatchManifest{}, &core.Pin{Sequence: 12345, Signer: "0x12345"}, 10, &core.MessageManifestEntry{
		MessageRef: core.MessageRef{
			ID:   msg.Header.ID,
			Hash: msg.Hash,
		},
		Topics: len(msg.Header.Topics),
	}, newBatchState(ag))
	assert.EqualError(t, err, "pop")

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)

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

	msg := &core.Message{
		Header: core.MessageHeader{
			ID:        fftypes.NewUUID(),
			Group:     fftypes.NewRandB32(),
			Topics:    core.FFStringArray{"topic1"},
			Namespace: "ns1",
			SignerRef: core.SignerRef{
				Author: org1.DID,
				Key:    "0x12345",
			},
		},
		Pins: core.FFStringArray{pin.String()},
	}

	mim.On("FindIdentityForVerifier", ag.ctx, []core.IdentityType{core.IdentityTypeOrg, core.IdentityTypeCustom}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x12345",
	}).Return(org1, nil)
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return([]*core.NextPin{
		{Context: fftypes.NewRandB32(), Hash: pin, Identity: org1.DID},
	}, nil, nil)
	mdm.On("GetMessageWithDataCached", ag.ctx, mock.Anything, data.CRORequirePins).Return(msg, nil, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(false, nil)
	mdi.On("InsertEvent", ag.ctx, mock.Anything).Return(nil)
	mdi.On("UpdateMessages", ag.ctx, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpdateNextPin", ag.ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := ag.processMessage(ag.ctx, &core.BatchManifest{
		ID: fftypes.NewUUID(),
	}, &core.Pin{Masked: true, Sequence: 12345, Signer: "0x12345"}, 10, &core.MessageManifestEntry{
		MessageRef: core.MessageRef{
			ID:   msg.Header.ID,
			Hash: msg.Hash,
		},
		Topics: len(msg.Header.Topics),
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
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return([]*core.NextPin{
		{Context: fftypes.NewRandB32(), Hash: pin},
	}, nil, nil)

	bs := newBatchState(ag)
	_, err := bs.CheckMaskedContextReady(ag.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:     fftypes.NewUUID(),
			Group:  fftypes.NewRandB32(),
			Tag:    core.SystemTagDefineDatatype,
			Topics: core.FFStringArray{"topic1"},
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), "12345")
	assert.NoError(t, err)

}

func TestAttemptContextInitGetGroupByIDFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mpm := ag.messaging.(*privatemessagingmocks.Manager)
	mpm.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	bs := newBatchState(ag)
	_, err := bs.attemptContextInit(ag.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: fftypes.NewRandB32(),
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), fftypes.NewRandB32())
	assert.EqualError(t, err, "pop")

	mpm.AssertExpectations(t)
}

func TestAttemptContextInitGroupNotFound(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mpm := ag.messaging.(*privatemessagingmocks.Manager)
	mpm.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(nil, nil)

	bs := newBatchState(ag)
	_, err := bs.attemptContextInit(ag.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: fftypes.NewRandB32(),
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), fftypes.NewRandB32())
	assert.NoError(t, err)

	mpm.AssertExpectations(t)
}

func TestAttemptContextInitAuthorMismatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	groupID := fftypes.NewRandB32()
	initNPG := &nextPinGroupState{topic: "topic1", groupID: groupID}
	zeroHash := initNPG.calcPinHash("author2", 0)
	mpm := ag.messaging.(*privatemessagingmocks.Manager)
	mpm.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(&core.Group{
		GroupIdentity: core.GroupIdentity{
			Members: core.Members{
				{Identity: "author2"},
			},
		},
	}, nil)

	bs := newBatchState(ag)
	_, err := bs.attemptContextInit(ag.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: groupID,
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), zeroHash)
	assert.NoError(t, err)

	mpm.AssertExpectations(t)
}

func TestAttemptContextInitNoMatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	groupID := fftypes.NewRandB32()
	mpm := ag.messaging.(*privatemessagingmocks.Manager)
	mpm.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(&core.Group{
		GroupIdentity: core.GroupIdentity{
			Members: core.Members{
				{Identity: "author2"},
			},
		},
	}, nil)

	bs := newBatchState(ag)
	_, err := bs.attemptContextInit(ag.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: groupID,
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), fftypes.NewRandB32())
	assert.NoError(t, err)

	mpm.AssertExpectations(t)
}

func TestAttemptContextInitGetPinsFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	groupID := fftypes.NewRandB32()
	initNPG := &nextPinGroupState{topic: "topic1", groupID: groupID}
	zeroHash := initNPG.calcPinHash("author1", 0)
	mpm := ag.messaging.(*privatemessagingmocks.Manager)
	mdi := ag.database.(*databasemocks.Plugin)
	mpm.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(&core.Group{
		GroupIdentity: core.GroupIdentity{
			Members: core.Members{
				{Identity: "author1"},
			},
		},
	}, nil)
	mdi.On("GetPins", ag.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	bs := newBatchState(ag)
	_, err := bs.attemptContextInit(ag.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: groupID,
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), zeroHash)
	assert.EqualError(t, err, "pop")

	mpm.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAttemptContextInitGetPinsBlocked(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	groupID := fftypes.NewRandB32()
	initNPG := &nextPinGroupState{topic: "topic1", groupID: groupID}
	zeroHash := initNPG.calcPinHash("author1", 0)
	mdi := ag.database.(*databasemocks.Plugin)
	mpm := ag.messaging.(*privatemessagingmocks.Manager)
	mpm.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(&core.Group{
		GroupIdentity: core.GroupIdentity{
			Members: core.Members{
				{Identity: "author1"},
			},
		},
	}, nil)
	mdi.On("GetPins", ag.ctx, mock.Anything).Return([]*core.Pin{
		{Sequence: 12345},
	}, nil, nil)

	bs := newBatchState(ag)
	np, err := bs.attemptContextInit(ag.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: groupID,
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), zeroHash)
	assert.NoError(t, err)
	assert.Nil(t, np)

	mpm.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAttemptContextInitInsertPinsFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	groupID := fftypes.NewRandB32()
	initNPG := &nextPinGroupState{topic: "topic1", groupID: groupID}
	zeroHash := initNPG.calcPinHash("author1", 0)
	mdi := ag.database.(*databasemocks.Plugin)
	mpm := ag.messaging.(*privatemessagingmocks.Manager)
	mpm.On("ResolveInitGroup", ag.ctx, mock.Anything).Return(&core.Group{
		GroupIdentity: core.GroupIdentity{
			Members: core.Members{
				{Identity: "author1"},
			},
		},
	}, nil)
	mdi.On("GetPins", ag.ctx, mock.Anything).Return([]*core.Pin{}, nil, nil)
	mdi.On("InsertNextPin", ag.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	bs := newBatchState(ag)
	np, err := bs.attemptContextInit(ag.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:    fftypes.NewUUID(),
			Group: groupID,
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	}, "topic1", 12345, fftypes.NewRandB32(), zeroHash)
	assert.NoError(t, err)
	assert.NotNil(t, np)
	err = bs.RunFinalize(ag.ctx)
	assert.EqualError(t, err, "pop")

	mpm.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAttemptMessageDispatchFailValidateData(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdm := ag.data.(*datamocks.Manager)
	mim := ag.identity.(*identitymanagermocks.Manager)

	org1 := newTestOrg("org1")
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return(core.DataArray{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(false, fmt.Errorf("pop"))

	_, _, err := ag.attemptMessageDispatch(ag.ctx, &core.Message{
		Header: core.MessageHeader{ID: fftypes.NewUUID(), SignerRef: core.SignerRef{Key: "0x12345", Author: org1.DID}},
		Data: core.DataRefs{
			{ID: fftypes.NewUUID()},
		},
	}, core.DataArray{}, nil, &batchState{}, &core.Pin{Signer: "0x12345"})
	assert.EqualError(t, err, "pop")

}

func TestAttemptMessageDispatchMissingBlobs(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	blobHash := fftypes.NewRandB32()

	mim := ag.identity.(*identitymanagermocks.Manager)

	org1 := newTestOrg("org1")
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", ag.ctx, blobHash).Return(nil, nil)

	_, dispatched, err := ag.attemptMessageDispatch(ag.ctx, &core.Message{
		Header: core.MessageHeader{ID: fftypes.NewUUID(), SignerRef: core.SignerRef{Key: "0x12345", Author: org1.DID}},
	}, core.DataArray{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32(), Blob: &core.BlobRef{
			Hash:   blobHash,
			Public: "public-ref",
		}},
	}, nil, &batchState{}, &core.Pin{Signer: "0x12345"})
	assert.NoError(t, err)
	assert.False(t, dispatched)

}

func TestAttemptMessageDispatchMissingTransfers(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mim := ag.identity.(*identitymanagermocks.Manager)

	org1 := newTestOrg("org1")
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)
	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetTokenTransfers", ag.ctx, mock.Anything).Return([]*core.TokenTransfer{}, nil, nil)

	msg := &core.Message{
		Header: core.MessageHeader{
			ID:   fftypes.NewUUID(),
			Type: core.MessageTypeTransferBroadcast,
			SignerRef: core.SignerRef{
				Author: org1.DID,
				Key:    "0x12345",
			},
		},
	}
	msg.Hash = msg.Header.Hash()
	_, dispatched, err := ag.attemptMessageDispatch(ag.ctx, msg, core.DataArray{}, nil, &batchState{}, &core.Pin{Signer: "0x12345"})
	assert.NoError(t, err)
	assert.False(t, dispatched)

	mdi.AssertExpectations(t)
}

func TestAttemptMessageDispatchGetTransfersFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mim := ag.identity.(*identitymanagermocks.Manager)

	org1 := newTestOrg("org1")
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetTokenTransfers", ag.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	msg := &core.Message{
		Header: core.MessageHeader{
			ID:        fftypes.NewUUID(),
			Type:      core.MessageTypeTransferBroadcast,
			SignerRef: core.SignerRef{Key: "0x12345", Author: org1.DID},
		},
	}
	msg.Hash = msg.Header.Hash()
	_, dispatched, err := ag.attemptMessageDispatch(ag.ctx, msg, core.DataArray{}, nil, &batchState{}, &core.Pin{Signer: "0x12345"})
	assert.EqualError(t, err, "pop")
	assert.False(t, dispatched)

	mdi.AssertExpectations(t)
}

func TestAttemptMessageDispatchTransferMismatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	org1 := newTestOrg("org1")

	msg := &core.Message{
		Header: core.MessageHeader{
			ID:        fftypes.NewUUID(),
			Type:      core.MessageTypeTransferBroadcast,
			SignerRef: core.SignerRef{Key: "0x12345", Author: org1.DID},
		},
	}
	msg.Hash = msg.Header.Hash()

	transfers := []*core.TokenTransfer{{
		Message:     msg.Header.ID,
		MessageHash: fftypes.NewRandB32(),
	}}

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetTokenTransfers", ag.ctx, mock.Anything).Return(transfers, nil, nil)

	_, dispatched, err := ag.attemptMessageDispatch(ag.ctx, msg, core.DataArray{}, nil, &batchState{}, &core.Pin{Signer: "0x12345"})
	assert.NoError(t, err)
	assert.False(t, dispatched)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestDefinitionBroadcastActionRejectCustomCorrelator(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	org1 := newTestOrg("org1")

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	customCorrelator := fftypes.NewUUID()
	msh := ag.definitions.(*definitionsmocks.DefinitionHandler)
	msh.On("HandleDefinitionBroadcast", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(definitions.HandlerResult{Action: definitions.ActionReject, CustomCorrelator: customCorrelator}, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return(core.DataArray{}, true, nil)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("UpdateMessages", ag.ctx, mock.Anything, mock.MatchedBy(func(u database.Update) bool {
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
	mdi.On("InsertEvent", ag.ctx, mock.MatchedBy(func(event *core.Event) bool {
		return event.Correlator.Equals(customCorrelator)
	})).Return(nil)

	_, _, err := ag.attemptMessageDispatch(ag.ctx, &core.Message{
		Header: core.MessageHeader{
			Type:      core.MessageTypeDefinition,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
			SignerRef: core.SignerRef{Key: "0x12345", Author: org1.DID},
			Tag:       core.SystemTagDefineDatatype,
			Topics:    core.FFStringArray{"topic1"},
		},
		Data: core.DataRefs{
			{ID: fftypes.NewUUID()},
		},
	}, core.DataArray{}, nil, bs, &core.Pin{Signer: "0x12345"})
	assert.NoError(t, err)
	err = bs.RunFinalize(ag.ctx)
	assert.NoError(t, err)
}

func TestDefinitionBroadcastInvalidSigner(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	org1 := newTestOrg("org1")

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return(core.DataArray{}, true, nil)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("UpdateMessages", ag.ctx, mock.Anything, mock.MatchedBy(func(u database.Update) bool {
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

	_, _, err := ag.attemptMessageDispatch(ag.ctx, &core.Message{
		Header: core.MessageHeader{
			Type:      core.MessageTypeDefinition,
			ID:        fftypes.NewUUID(),
			Namespace: "any",
			SignerRef: core.SignerRef{Key: "0x12345", Author: org1.DID},
		},
		Data: core.DataRefs{
			{ID: fftypes.NewUUID()},
		},
	}, core.DataArray{}, nil, bs, &core.Pin{Signer: "0x12345"})
	assert.NoError(t, err)
}

func TestDispatchBroadcastQueuesLaterDispatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	msg1, msg2, org1, manifest := newTestManifest(core.MessageTypeDefinition, nil)

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", ag.ctx, msg1.Header.ID, data.CRORequirePublicBlobRefs).Return(msg1, core.DataArray{}, true, nil).Once()
	mdm.On("GetMessageWithDataCached", ag.ctx, msg2.Header.ID, data.CRORequirePublicBlobRefs).Return(msg2, core.DataArray{}, true, nil).Once()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetPins", ag.ctx, mock.Anything).Return([]*core.Pin{}, nil, nil)

	// First message should dispatch
	err := ag.processMessage(ag.ctx, manifest, &core.Pin{Sequence: 12345}, 0, manifest.Messages[0], bs)
	assert.NoError(t, err)

	// Second message should not (mocks have Once limit on GetMessageData to confirm)
	err = ag.processMessage(ag.ctx, manifest, &core.Pin{Sequence: 12346}, 0, manifest.Messages[1], bs)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestDispatchPrivateQueuesLaterDispatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	groupID := fftypes.NewRandB32()
	msg1, msg2, org1, manifest := newTestManifest(core.MessageTypePrivate, groupID)

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", ag.ctx, msg1.Header.ID, data.CRORequirePins).Return(msg1, core.DataArray{}, true, nil).Once()
	mdm.On("GetMessageWithDataCached", ag.ctx, msg2.Header.ID, data.CRORequirePins).Return(msg2, core.DataArray{}, true, nil).Once()

	initNPG := &nextPinGroupState{topic: "topic1", groupID: groupID}
	member1NonceOne := initNPG.calcPinHash("org1", 1)
	member1NonceTwo := initNPG.calcPinHash("org1", 2)
	h := sha256.New()
	h.Write([]byte("topic1"))
	context := fftypes.HashResult(h)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return([]*core.NextPin{
		{Context: context, Nonce: 1 /* match member1NonceOne */, Identity: org1.DID, Hash: member1NonceOne},
	}, nil, nil)

	msg1.Pins = core.FFStringArray{member1NonceOne.String()}
	msg2.Pins = core.FFStringArray{member1NonceTwo.String()}

	// First message should dispatch
	err := ag.processMessage(ag.ctx, manifest, &core.Pin{Masked: true, Sequence: 12345}, 0, manifest.Messages[0], bs)
	assert.NoError(t, err)

	// Second message should not (mocks have Once limit on GetMessageData to confirm)
	err = ag.processMessage(ag.ctx, manifest, &core.Pin{Masked: true, Sequence: 12346}, 0, manifest.Messages[1], bs)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestDispatchPrivateNextPinIncremented(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	groupID := fftypes.NewRandB32()
	msg1, msg2, org1, manifest := newTestManifest(core.MessageTypePrivate, groupID)

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", ag.ctx, msg1.Header.ID, data.CRORequirePins).Return(msg1, core.DataArray{}, true, nil).Once()
	mdm.On("GetMessageWithDataCached", ag.ctx, msg2.Header.ID, data.CRORequirePins).Return(msg2, core.DataArray{}, true, nil).Once()
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)

	initNPG := &nextPinGroupState{topic: "topic1", groupID: groupID}
	member1NonceOne := initNPG.calcPinHash(org1.DID, 1)
	member1NonceTwo := initNPG.calcPinHash(org1.DID, 2)
	h := sha256.New()
	h.Write([]byte("topic1"))
	context := fftypes.HashResult(h)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetNextPins", ag.ctx, mock.Anything).Return([]*core.NextPin{
		{Context: context, Nonce: 1 /* match member1NonceOne */, Identity: org1.DID, Hash: member1NonceOne},
	}, nil, nil)

	msg1.Pins = core.FFStringArray{member1NonceOne.String()}
	msg2.Pins = core.FFStringArray{member1NonceTwo.String()}

	// First message should dispatch
	err := ag.processMessage(ag.ctx, manifest, &core.Pin{Masked: true, Sequence: 12345, Signer: "0x12345"}, 0, manifest.Messages[0], bs)
	assert.NoError(t, err)

	// Second message should dispatch too (Twice on GetMessageData)
	err = ag.processMessage(ag.ctx, manifest, &core.Pin{Masked: true, Sequence: 12346, Signer: "0x12345"}, 0, manifest.Messages[1], bs)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestDefinitionBroadcastActionRetry(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	msg1, _, org1, _ := newTestManifest(core.MessageTypeDefinition, nil)

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	msh := ag.definitions.(*definitionsmocks.DefinitionHandler)
	msh.On("HandleDefinitionBroadcast", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(definitions.HandlerResult{Action: definitions.ActionRetry}, fmt.Errorf("pop"))

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", ag.ctx, mock.Anything).Return(msg1, core.DataArray{}, true, nil)

	_, _, err := ag.attemptMessageDispatch(ag.ctx, msg1, nil, nil, &batchState{}, &core.Pin{Signer: "0x12345"})
	assert.EqualError(t, err, "pop")

}

func TestDefinitionBroadcastRejectSignerLookupFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	msg1, _, _, _ := newTestManifest(core.MessageTypeDefinition, nil)

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, valid, err := ag.attemptMessageDispatch(ag.ctx, msg1, nil, nil, &batchState{}, &core.Pin{Signer: "0x12345"})
	assert.Regexp(t, "pop", err)
	assert.False(t, valid)

	mim.AssertExpectations(t)
}

func TestDefinitionBroadcastRejectSignerLookupWrongOrg(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	msg1, _, _, _ := newTestManifest(core.MessageTypeDefinition, nil)

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(newTestOrg("org2"), nil)

	_, valid, err := ag.attemptMessageDispatch(ag.ctx, msg1, nil, nil, &batchState{}, &core.Pin{Signer: "0x12345"})
	assert.NoError(t, err)
	assert.False(t, valid)

	mim.AssertExpectations(t)
}

func TestDefinitionBroadcastRejectBadSigner(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	msg1, _, org1, _ := newTestManifest(core.MessageTypeDefinition, nil)
	msg1.Header.SignerRef = core.SignerRef{Key: "0x23456", Author: org1.DID}

	_, valid, err := ag.attemptMessageDispatch(ag.ctx, msg1, nil, nil, &batchState{}, &core.Pin{Signer: "0x12345"})
	assert.NoError(t, err)
	assert.False(t, valid)

}

func TestDefinitionBroadcastParkUnregisteredSignerIdentityClaim(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	msg1, _, _, _ := newTestManifest(core.MessageTypeDefinition, nil)
	msg1.Header.Tag = core.SystemTagIdentityClaim

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	msh := ag.definitions.(*definitionsmocks.DefinitionHandler)
	msh.On("HandleDefinitionBroadcast", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(definitions.HandlerResult{Action: definitions.ActionWait}, nil)

	newState, valid, err := ag.attemptMessageDispatch(ag.ctx, msg1, nil, nil, &batchState{}, &core.Pin{Signer: "0x12345"})
	assert.NoError(t, err)
	assert.False(t, valid)
	assert.Empty(t, newState)

	mim.AssertExpectations(t)
	msh.AssertExpectations(t)
}

func TestDefinitionBroadcastRootUnregisteredOk(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	msg1, _, _, _ := newTestManifest(core.MessageTypeDefinition, nil)

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	_, valid, err := ag.attemptMessageDispatch(ag.ctx, msg1, nil, nil, &batchState{}, &core.Pin{Signer: "0x12345"})
	assert.NoError(t, err)
	assert.False(t, valid)

	mim.AssertExpectations(t)
}

func TestDefinitionBroadcastActionWait(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	msg1, _, org1, _ := newTestManifest(core.MessageTypeDefinition, nil)

	mim := ag.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)

	msh := ag.definitions.(*definitionsmocks.DefinitionHandler)
	msh.On("HandleDefinitionBroadcast", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(definitions.HandlerResult{Action: definitions.ActionWait}, nil)

	_, _, err := ag.attemptMessageDispatch(ag.ctx, msg1, nil, nil, &batchState{}, &core.Pin{Signer: "0x12345"})
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	msh.AssertExpectations(t)

}

func TestAttemptMessageDispatchEventFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)
	msg1, _, org1, _ := newTestManifest(core.MessageTypeBroadcast, nil)

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)
	mim := ag.identity.(*identitymanagermocks.Manager)

	mim.On("FindIdentityForVerifier", ag.ctx, mock.Anything, mock.Anything, mock.Anything).Return(org1, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	mdi.On("InsertEvent", ag.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	_, _, err := ag.attemptMessageDispatch(ag.ctx, msg1, core.DataArray{
		&core.Data{ID: msg1.Data[0].ID},
	}, nil, bs, &core.Pin{Signer: "0x12345"})
	assert.NoError(t, err)

	err = bs.RunFinalize(ag.ctx)
	assert.EqualError(t, err, "pop")

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)

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
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return(core.DataArray{}, true, nil)
	mdm.On("ValidateAll", ag.ctx, mock.Anything).Return(true, nil)
	mdi.On("InsertEvent", ag.ctx, mock.Anything).Return(nil)

	_, _, err := ag.attemptMessageDispatch(ag.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:        fftypes.NewUUID(),
			Type:      core.MessageTypeGroupInit,
			SignerRef: core.SignerRef{Key: "0x12345", Author: org1.DID},
		},
	}, nil, nil, bs, &core.Pin{Signer: "0x12345"})
	assert.NoError(t, err)

}

func TestRewindOffchainBatchesNoBatches(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("UpdateMessages", ag.ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	rewind, offset := ag.rewindOffchainBatches()
	assert.False(t, rewind)
	assert.Equal(t, int64(0), offset)
}

func TestRewindOffchainBatchesBatchesNoRewind(t *testing.T) {
	config.Set(coreconfig.EventAggregatorBatchSize, 10)

	ag, cancel := newTestAggregator()
	defer cancel()

	rewind, offset := ag.rewindOffchainBatches()
	assert.False(t, rewind)
	assert.Equal(t, int64(0), offset)
}

func TestRewindOffchainBatchesAndTXRewind(t *testing.T) {
	config.Set(coreconfig.EventAggregatorBatchSize, 10)

	ag, cancel := newTestAggregator()
	defer cancel()

	batchID := fftypes.NewUUID()
	ag.rewinder.readyRewinds = map[fftypes.UUID]bool{
		*batchID: true,
	}

	mdi := ag.database.(*databasemocks.Plugin)
	mockRunAsGroupPassthrough(mdi)
	mdi.On("GetBatchIDs", ag.ctx, mock.Anything, mock.Anything).Return([]*fftypes.UUID{
		fftypes.NewUUID(),
	}, nil)
	mdi.On("GetPins", ag.ctx, mock.Anything, mock.Anything).Return([]*core.Pin{
		{Sequence: 12345, Batch: fftypes.NewUUID()},
	}, nil, nil)

	rewind, offset := ag.rewindOffchainBatches()
	assert.True(t, rewind)
	assert.Equal(t, int64(12344) /* one before the batch */, offset)

}

func TestRewindOffchainBatchesError(t *testing.T) {
	config.Set(coreconfig.EventAggregatorBatchSize, 10)

	ag, cancel := newTestAggregator()
	cancel()

	batchID := fftypes.NewUUID()
	ag.rewinder.readyRewinds = map[fftypes.UUID]bool{
		*batchID: true,
	}

	mdi := ag.database.(*databasemocks.Plugin)
	mockRunAsGroupPassthrough(mdi)
	mdi.On("GetBatchIDs", ag.ctx, mock.Anything, mock.Anything).Return([]*fftypes.UUID{
		fftypes.NewUUID(),
	}, nil)
	mdi.On("GetPins", ag.ctx, mock.Anything, mock.Anything).Return([]*core.Pin{
		{Sequence: 12345, Batch: fftypes.NewUUID()},
	}, nil, fmt.Errorf("pop"))

	// We will return 0, as context ended so retry will exit after error
	rewind, offset := ag.rewindOffchainBatches()
	assert.False(t, rewind)
	assert.Equal(t, int64(0), offset)

}

func TestResolveBlobsNoop(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	resolved, err := ag.resolveBlobs(ag.ctx, core.DataArray{
		{ID: fftypes.NewUUID(), Blob: &core.BlobRef{}},
	})

	assert.NoError(t, err)
	assert.True(t, resolved)
}

func TestResolveBlobsErrorGettingHash(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", ag.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	resolved, err := ag.resolveBlobs(ag.ctx, core.DataArray{
		{ID: fftypes.NewUUID(), Blob: &core.BlobRef{
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

	resolved, err := ag.resolveBlobs(ag.ctx, core.DataArray{
		{ID: fftypes.NewUUID(), Blob: &core.BlobRef{
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
	mdi.On("GetBlobMatchingHash", ag.ctx, mock.Anything).Return(&core.Blob{}, nil)

	resolved, err := ag.resolveBlobs(ag.ctx, core.DataArray{
		{ID: fftypes.NewUUID(), Blob: &core.BlobRef{
			Hash: fftypes.NewRandB32(),
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

func TestProcessWithBatchRewindsSuccess(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{a[1].(func(context.Context) error)(a[0].(context.Context))}
	}

	err := ag.processWithBatchState(func(ctx context.Context, actions *batchState) error {
		actions.DIDClaimConfirmed("did:firefly:org/test")
		return nil
	})
	assert.NoError(t, err)
}

func TestProcessWithBatchActionsFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Once()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{a[1].(func(context.Context) error)(a[0].(context.Context))}
	}
	mdi.On("RunAsGroup", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := ag.processWithBatchState(func(ctx context.Context, actions *batchState) error {
		actions.AddPreFinalize(func(ctx context.Context) error { return nil })
		return nil
	})
	assert.EqualError(t, err, "pop")
}

func TestExtractManifestFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	manifest := ag.extractManifest(ag.ctx, &core.BatchPersisted{
		Manifest: fftypes.JSONAnyPtr("!wrong"),
	})

	assert.Nil(t, manifest)
}

func TestExtractManifestBadVersion(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	manifest := ag.extractManifest(ag.ctx, &core.BatchPersisted{
		Manifest: fftypes.JSONAnyPtr(`{"version":999}`),
	})

	assert.Nil(t, manifest)
}

func TestMigrateManifestFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	manifest := ag.migrateManifest(ag.ctx, &core.BatchPersisted{
		Manifest: fftypes.JSONAnyPtr("!wrong"),
	})

	assert.Nil(t, manifest)
}

func TestBatchCaching(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	persisted, expectedManifest := batch.Confirmed()

	pin := &core.Pin{
		Batch:     batch.ID,
		BatchHash: batch.Hash,
	}

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", ag.ctx, batch.ID).Return(persisted, nil).Once() // to prove caching

	batchRetrieved, manifest, err := ag.GetBatchForPin(ag.ctx, pin)
	assert.NoError(t, err)
	assert.Equal(t, persisted, batchRetrieved)
	assert.Equal(t, expectedManifest, manifest)

	batchRetrieved, manifest, err = ag.GetBatchForPin(ag.ctx, pin)
	assert.NoError(t, err)
	assert.Equal(t, persisted, batchRetrieved)
	assert.Equal(t, expectedManifest, manifest)

}

func TestGetBatchForPinHashMismatch(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	persisted, _ := batch.Confirmed()
	pin := &core.Pin{
		Batch:     batch.ID,
		BatchHash: fftypes.NewRandB32(),
	}

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", ag.ctx, batch.ID).Return(persisted, nil)

	batchRetrieved, manifest, err := ag.GetBatchForPin(ag.ctx, pin)
	assert.Nil(t, batchRetrieved)
	assert.Nil(t, manifest)
	assert.Nil(t, err)

}
