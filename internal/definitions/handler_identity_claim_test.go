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

package definitions

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testOrgIdentity(t *testing.T, name string) *core.Identity {
	i := &core.Identity{
		IdentityBase: core.IdentityBase{
			ID:        fftypes.NewUUID(),
			Type:      core.IdentityTypeOrg,
			Namespace: "ns1",
			Name:      name,
		},
		IdentityProfile: core.IdentityProfile{
			Description: "desc",
			Profile: fftypes.JSONObject{
				"some": "profiledata",
			},
		},
		Messages: core.IdentityMessages{
			Claim: fftypes.NewUUID(),
		},
	}
	var err error
	i.DID, err = i.GenerateDID(context.Background())
	assert.NoError(t, err)
	return i
}

func testCustomIdentity(t *testing.T, name string, parent *core.Identity) *core.Identity {
	i := &core.Identity{
		IdentityBase: core.IdentityBase{
			ID:        fftypes.NewUUID(),
			Type:      core.IdentityTypeCustom,
			Namespace: "ns1",
			Name:      name,
			Parent:    parent.ID,
		},
		IdentityProfile: core.IdentityProfile{
			Description: "custom 1",
			Profile: fftypes.JSONObject{
				"some": "profiledata",
			},
		},
		Messages: core.IdentityMessages{
			Claim: fftypes.NewUUID(),
		},
	}
	var err error
	i.DID, err = i.GenerateDID(context.Background())
	assert.NoError(t, err)
	return i
}

func testCustomClaimAndVerification(t *testing.T) (*core.Identity, *core.Identity, *core.Message, *core.Data, *core.Message, *core.Data) {
	org1 := testOrgIdentity(t, "org1")
	custom1 := testCustomIdentity(t, "custom1", org1)

	ic := &core.IdentityClaim{
		Identity: custom1,
	}
	b, err := json.Marshal(&ic)
	assert.NoError(t, err)
	claimData := &core.Data{
		ID:    fftypes.NewUUID(),
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	claimMsg := &core.Message{
		Header: core.MessageHeader{
			Namespace: "ns1",
			ID:        custom1.Messages.Claim,
			Type:      core.MessageTypeDefinition,
			Tag:       core.SystemTagIdentityClaim,
			Topics:    fftypes.FFStringArray{custom1.Topic()},
			SignerRef: core.SignerRef{
				Author: custom1.DID,
				Key:    "0x12345",
			},
		},
	}
	claimMsg.Hash = fftypes.NewRandB32()

	iv := &core.IdentityVerification{
		Identity: custom1.IdentityBase,
		Claim: core.MessageRef{
			ID:   claimMsg.Header.ID,
			Hash: claimMsg.Hash,
		},
	}
	b, err = json.Marshal(&iv)
	assert.NoError(t, err)
	verifyData := &core.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	verifyMsg := &core.Message{
		Header: core.MessageHeader{
			ID:     fftypes.NewUUID(),
			Type:   core.MessageTypeDefinition,
			Tag:    core.SystemTagIdentityVerification,
			Topics: fftypes.FFStringArray{custom1.Topic()},
			SignerRef: core.SignerRef{
				Author: org1.DID,
				Key:    "0x2456",
			},
		},
	}

	return custom1, org1, claimMsg, claimData, verifyMsg, verifyData
}

func TestHandleDefinitionIdentityClaimCustomWithExistingParentVerificationOk(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, verifyMsg, verifyData := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, "ns1", custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, core.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, nil)
	mdi.On("GetMessages", ctx, "ns1", mock.Anything).Return([]*core.Message{
		{Header: core.MessageHeader{ID: fftypes.NewUUID(), Tag: "skipped missing data"}},
	}, nil, nil)
	mdi.On("UpsertIdentity", ctx, mock.MatchedBy(func(identity *core.Identity) bool {
		assert.Equal(t, *claimMsg.Header.ID, *identity.Messages.Claim)
		assert.Equal(t, *verifyMsg.Header.ID, *identity.Messages.Verification)
		return true
	}), database.UpsertOptimizationNew).Return(nil)
	mdi.On("UpsertVerifier", ctx, mock.MatchedBy(func(verifier *core.Verifier) bool {
		assert.Equal(t, core.VerifierTypeEthAddress, verifier.Type)
		assert.Equal(t, "0x12345", verifier.Value)
		assert.Equal(t, *custom1.ID, *verifier.Identity)
		return true
	}), database.UpsertOptimizationNew).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(event *core.Event) bool {
		return event.Type == core.EventTypeIdentityConfirmed
	})).Return(nil)

	mdm := dh.data.(*datamocks.Manager)
	mdm.On("GetMessageDataCached", ctx, mock.Anything).Return(core.DataArray{verifyData}, false, nil).Once()
	mdm.On("GetMessageDataCached", ctx, mock.Anything).Return(core.DataArray{verifyData}, true, nil)

	dh.multiparty = true

	bs.AddPendingConfirm(verifyMsg.Header.ID, verifyMsg)

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionConfirm}, action)
	assert.NoError(t, err)
	assert.Equal(t, bs.ConfirmedDIDClaims, []string{custom1.DID})

	err = bs.RunFinalize(ctx)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)

}

func TestHandleDefinitionIdentityClaimIdempotentReplay(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, verifyMsg, verifyData := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, "ns1", custom1.ID).Return(custom1, nil)
	mdi.On("GetVerifierByValue", ctx, core.VerifierTypeEthAddress, "ns1", "0x12345").Return(&core.Verifier{
		Identity:  custom1.ID,
		Namespace: "ns1",
		VerifierRef: core.VerifierRef{
			Type:  core.VerifierTypeEthAddress,
			Value: "0x12345",
		},
	}, nil)
	mdi.On("GetMessages", ctx, "ns1", mock.Anything).Return([]*core.Message{
		{Header: core.MessageHeader{ID: fftypes.NewUUID(), Tag: "skipped missing data"}},
	}, nil, nil)
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(event *core.Event) bool {
		return event.Type == core.EventTypeIdentityConfirmed
	})).Return(nil)

	mdm := dh.data.(*datamocks.Manager)
	mdm.On("GetMessageDataCached", ctx, mock.Anything).Return(core.DataArray{verifyData}, false, nil).Once()
	mdm.On("GetMessageDataCached", ctx, mock.Anything).Return(core.DataArray{verifyData}, true, nil)

	dh.multiparty = true

	bs.AddPendingConfirm(verifyMsg.Header.ID, verifyMsg)

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionConfirm}, action)
	assert.NoError(t, err)

	err = bs.RunFinalize(ctx)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestHandleDefinitionIdentityClaimFailInsertIdentity(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, verifyMsg, verifyData := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, "ns1", custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, core.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, nil)
	mdi.On("GetMessages", ctx, "ns1", mock.Anything).Return([]*core.Message{}, nil, nil)
	mdi.On("UpsertVerifier", ctx, mock.Anything, database.UpsertOptimizationNew).Return(nil)
	mdi.On("UpsertIdentity", ctx, mock.Anything, database.UpsertOptimizationNew).Return(fmt.Errorf("pop"))

	mdm := dh.data.(*datamocks.Manager)
	mdm.On("GetMessageDataCached", ctx, mock.Anything).Return(core.DataArray{verifyData}, true, nil)

	dh.multiparty = true

	bs.AddPendingConfirm(verifyMsg.Header.ID, verifyMsg)

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimVerificationDataFail(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, verifyMsg, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, "ns1", custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, core.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, nil)
	mdi.On("GetMessages", ctx, "ns1", mock.Anything).Return([]*core.Message{}, nil, nil)

	mdm := dh.data.(*datamocks.Manager)
	mdm.On("GetMessageDataCached", ctx, mock.Anything).Return(nil, false, fmt.Errorf("pop"))

	dh.multiparty = true

	bs.AddPendingConfirm(verifyMsg.Header.ID, verifyMsg)

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimVerificationMissingData(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, verifyMsg, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, "ns1", custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, core.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, nil)
	mdi.On("GetMessages", ctx, "ns1", mock.Anything).Return([]*core.Message{}, nil, nil)

	mdm := dh.data.(*datamocks.Manager)
	mdm.On("GetMessageDataCached", ctx, mock.Anything).Return(core.DataArray{}, true, nil)

	dh.multiparty = true

	bs.AddPendingConfirm(verifyMsg.Header.ID, verifyMsg)

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionConfirm}, action)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimFailInsertVerifier(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, verifyMsg, verifyData := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, "ns1", custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, core.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, nil)
	mdi.On("GetMessages", ctx, "ns1", mock.Anything).Return([]*core.Message{}, nil, nil)
	mdi.On("UpsertVerifier", ctx, mock.Anything, database.UpsertOptimizationNew).Return(fmt.Errorf("pop"))

	mdm := dh.data.(*datamocks.Manager)
	mdm.On("GetMessageDataCached", ctx, mock.Anything).Return(core.DataArray{verifyData}, true, nil)

	dh.multiparty = true

	bs.AddPendingConfirm(verifyMsg.Header.ID, verifyMsg)

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimCustomMissingParentVerificationOk(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, "ns1", custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, core.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, nil)
	mdi.On("GetMessages", ctx, "ns1", mock.Anything).Return([]*core.Message{}, nil, nil)

	dh.multiparty = true

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionConfirm}, action) // Just wait for the verification to come in later
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimCustomParentVerificationFail(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, "ns1", custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, core.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, nil)
	mdi.On("GetMessages", ctx, "ns1", mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	dh.multiparty = true

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimVerifierClash(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, "ns1", custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, core.VerifierTypeEthAddress, "ns1", "0x12345").Return(&core.Verifier{
		Hash: fftypes.NewRandB32(),
	}, nil)

	dh.multiparty = true

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.Error(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimVerifierError(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, "ns1", custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, core.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, fmt.Errorf("pop"))

	dh.multiparty = true

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimIdentityClash(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID: fftypes.NewUUID(),
		},
	}, nil)

	dh.multiparty = true

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.Error(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimIdentityError(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, "ns1", custom1.ID).Return(nil, fmt.Errorf("pop"))

	dh.multiparty = true

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityMissingAuthor(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)
	claimMsg.Header.Author = ""

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	dh.multiparty = true

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.Error(t, err)

	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimBadSignature(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)
	claimMsg.Header.Author = org1.DID // should be the child for the claim

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	dh.multiparty = true

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.Error(t, err)

	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityVerifyChainFail(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)
	claimMsg.Header.Author = org1.DID // should be the child for the claim

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(nil, true, fmt.Errorf("pop"))

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityVerifyChainInvalid(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)
	claimMsg.Header.Author = org1.DID // should be the child for the claim

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(nil, false, fmt.Errorf("wrong"))

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{claimData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionWait}, action)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimBadData(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	_, org1, claimMsg, _, _, _ := testCustomClaimAndVerification(t)
	claimMsg.Header.Author = org1.DID // should be the child for the claim

	action, err := dh.HandleDefinitionBroadcast(ctx, &bs.BatchState, claimMsg, core.DataArray{}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.Error(t, err)

	bs.assertNoFinalizers()
}
