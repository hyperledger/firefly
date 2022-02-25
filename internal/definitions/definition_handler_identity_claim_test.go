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

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testOrgIdentity(t *testing.T, name string) *fftypes.Identity {
	i := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			Type:      fftypes.IdentityTypeOrg,
			Namespace: fftypes.SystemNamespace,
			Name:      name,
		},
		IdentityProfile: fftypes.IdentityProfile{
			Description: "desc",
			Profile: fftypes.JSONObject{
				"some": "profiledata",
			},
		},
		Messages: fftypes.IdentityMessages{
			Claim: fftypes.NewUUID(),
		},
	}
	var err error
	i.DID, err = i.GenerateDID(context.Background())
	assert.NoError(t, err)
	return i
}

func testCustomIdentity(t *testing.T, name string, parent *fftypes.Identity) *fftypes.Identity {
	i := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			Type:      fftypes.IdentityTypeCustom,
			Namespace: "ns1",
			Name:      name,
			Parent:    parent.ID,
		},
		IdentityProfile: fftypes.IdentityProfile{
			Description: "custom 1",
			Profile: fftypes.JSONObject{
				"some": "profiledata",
			},
		},
		Messages: fftypes.IdentityMessages{
			Claim: fftypes.NewUUID(),
		},
	}
	var err error
	i.DID, err = i.GenerateDID(context.Background())
	assert.NoError(t, err)
	return i
}

func testCustomClaimAndVerification(t *testing.T) (*fftypes.Identity, *fftypes.Identity, *fftypes.Message, *fftypes.Data, *fftypes.Message, *fftypes.Data) {
	org1 := testOrgIdentity(t, "org1")
	custom1 := testCustomIdentity(t, "custom1", org1)

	ic := &fftypes.IdentityClaim{
		Identity: custom1,
	}
	b, err := json.Marshal(&ic)
	assert.NoError(t, err)
	claimData := &fftypes.Data{
		ID:    fftypes.NewUUID(),
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	claimMsg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:     custom1.Messages.Claim,
			Type:   fftypes.MessageTypeDefinition,
			Tag:    fftypes.SystemTagIdentityClaim,
			Topics: fftypes.FFStringArray{custom1.Topic()},
			SignerRef: fftypes.SignerRef{
				Author: custom1.DID,
				Key:    "0x12345",
			},
		},
	}
	claimMsg.Hash = fftypes.NewRandB32()

	iv := &fftypes.IdentityVerification{
		Identity: custom1.IdentityBase,
		Claim: fftypes.MessageRef{
			ID:   claimMsg.Header.ID,
			Hash: claimMsg.Hash,
		},
	}
	b, err = json.Marshal(&iv)
	assert.NoError(t, err)
	verifyData := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	verifyMsg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:     fftypes.NewUUID(),
			Type:   fftypes.MessageTypeDefinition,
			Tag:    fftypes.SystemTagIdentityVerification,
			Topics: fftypes.FFStringArray{custom1.Topic()},
			SignerRef: fftypes.SignerRef{
				Author: org1.DID,
				Key:    "0x2456",
			},
		},
	}

	return custom1, org1, claimMsg, claimData, verifyMsg, verifyData
}

func TestHandleDefinitionIdentityClaimCustomWithExistingParentVerificationOk(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, verifyMsg, verifyData := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, nil)
	mdi.On("GetMessages", ctx, mock.Anything).Return([]*fftypes.Message{
		{Header: fftypes.MessageHeader{ID: fftypes.NewUUID(), Tag: "skipped missing data"}},
	}, nil, nil)
	mdi.On("UpsertIdentity", ctx, mock.MatchedBy(func(identity *fftypes.Identity) bool {
		assert.Equal(t, *claimMsg.Header.ID, *identity.Messages.Claim)
		assert.Equal(t, *verifyMsg.Header.ID, *identity.Messages.Verification)
		return true
	}), database.UpsertOptimizationNew).Return(nil)
	mdi.On("UpsertVerifier", ctx, mock.MatchedBy(func(verifier *fftypes.Verifier) bool {
		assert.Equal(t, fftypes.VerifierTypeEthAddress, verifier.Type)
		assert.Equal(t, "0x12345", verifier.Value)
		assert.Equal(t, *custom1.ID, *verifier.Identity)
		return true
	}), database.UpsertOptimizationNew).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(event *fftypes.Event) bool {
		return event.Type == fftypes.EventTypeIdentityConfirmed
	})).Return(nil)

	mdm := dh.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ctx, mock.Anything, true).Return([]*fftypes.Data{verifyData}, false, nil).Once()
	mdm.On("GetMessageData", ctx, mock.Anything, true).Return([]*fftypes.Data{verifyData}, true, nil)

	bs.pendingConfirms[*verifyMsg.Header.ID] = verifyMsg

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionConfirm, action)
	assert.NoError(t, err)

	err = bs.finalizers[0](ctx)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)

}

func TestHandleDefinitionIdentityClaimIdempotentReplay(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, verifyMsg, verifyData := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, custom1.ID).Return(custom1, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, "ns1", "0x12345").Return(&fftypes.Verifier{
		Identity:  custom1.ID,
		Namespace: "ns1",
		VerifierRef: fftypes.VerifierRef{
			Type:  fftypes.VerifierTypeEthAddress,
			Value: "0x12345",
		},
	}, nil)
	mdi.On("GetMessages", ctx, mock.Anything).Return([]*fftypes.Message{
		{Header: fftypes.MessageHeader{ID: fftypes.NewUUID(), Tag: "skipped missing data"}},
	}, nil, nil)
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(event *fftypes.Event) bool {
		return event.Type == fftypes.EventTypeIdentityConfirmed
	})).Return(nil)

	mdm := dh.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ctx, mock.Anything, true).Return([]*fftypes.Data{verifyData}, false, nil).Once()
	mdm.On("GetMessageData", ctx, mock.Anything, true).Return([]*fftypes.Data{verifyData}, true, nil)

	bs.pendingConfirms[*verifyMsg.Header.ID] = verifyMsg

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionConfirm, action)
	assert.NoError(t, err)

	err = bs.finalizers[0](ctx)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestHandleDefinitionIdentityClaimFailInsertIdentity(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, verifyMsg, verifyData := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, nil)
	mdi.On("GetMessages", ctx, mock.Anything).Return([]*fftypes.Message{}, nil, nil)
	mdi.On("UpsertVerifier", ctx, mock.Anything, database.UpsertOptimizationNew).Return(nil)
	mdi.On("UpsertIdentity", ctx, mock.Anything, database.UpsertOptimizationNew).Return(fmt.Errorf("pop"))

	mdm := dh.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ctx, mock.Anything, true).Return([]*fftypes.Data{verifyData}, true, nil)

	bs.pendingConfirms[*verifyMsg.Header.ID] = verifyMsg

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionRetry, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimVerificationDataFail(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, verifyMsg, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, nil)
	mdi.On("GetMessages", ctx, mock.Anything).Return([]*fftypes.Message{}, nil, nil)

	mdm := dh.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ctx, mock.Anything, true).Return(nil, false, fmt.Errorf("pop"))

	bs.pendingConfirms[*verifyMsg.Header.ID] = verifyMsg

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionRetry, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimVerificationMissingData(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, verifyMsg, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, nil)
	mdi.On("GetMessages", ctx, mock.Anything).Return([]*fftypes.Message{}, nil, nil)

	mdm := dh.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ctx, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)

	bs.pendingConfirms[*verifyMsg.Header.ID] = verifyMsg

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionConfirm, action)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimFailInsertVerifier(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, verifyMsg, verifyData := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, nil)
	mdi.On("GetMessages", ctx, mock.Anything).Return([]*fftypes.Message{}, nil, nil)
	mdi.On("UpsertVerifier", ctx, mock.Anything, database.UpsertOptimizationNew).Return(fmt.Errorf("pop"))

	mdm := dh.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ctx, mock.Anything, true).Return([]*fftypes.Data{verifyData}, true, nil)

	bs.pendingConfirms[*verifyMsg.Header.ID] = verifyMsg

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionRetry, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimCustomMissingParentVerificationOk(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, nil)
	mdi.On("GetMessages", ctx, mock.Anything).Return([]*fftypes.Message{}, nil, nil)

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionConfirm, action) // Just wait for the verification to come in later
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimCustomParentVerificationFail(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, nil)
	mdi.On("GetMessages", ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionRetry, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimVerifierClash(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, "ns1", "0x12345").Return(&fftypes.Verifier{
		ID: fftypes.NewUUID(),
	}, nil)

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimVerifierError(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, fmt.Errorf("pop"))

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionRetry, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimIdentityClash(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(&fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID: fftypes.NewUUID(),
		},
	}, nil)

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimIdentityError(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, custom1.ID).Return(nil, fmt.Errorf("pop"))

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionRetry, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityMissingAuthor(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)
	claimMsg.Header.Author = ""

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimBadSignature(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)
	claimMsg.Header.Author = org1.DID // should be the child for the claim

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(org1, false, nil)

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityVerifyChainFail(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)
	claimMsg.Header.Author = org1.DID // should be the child for the claim

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(nil, true, fmt.Errorf("pop"))

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionRetry, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityVerifyChainInvalid(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, _, _ := testCustomClaimAndVerification(t)
	claimMsg.Header.Author = org1.DID // should be the child for the claim

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, custom1).Return(nil, false, fmt.Errorf("wrong"))

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{claimData}, fftypes.NewUUID())
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityClaimBadData(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	_, org1, claimMsg, _, _, _ := testCustomClaimAndVerification(t)
	claimMsg.Header.Author = org1.DID // should be the child for the claim

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, claimMsg, []*fftypes.Data{}, fftypes.NewUUID())
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	bs.assertNoFinalizers()
}
