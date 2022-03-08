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

func TestHandleDefinitionIdentityVerificationWithExistingClaimOk(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	custom1, org1, claimMsg, claimData, verifyMsg, verifyData := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", ctx, org1.ID).Return(org1, nil)
	mim.On("VerifyIdentityChain", ctx, mock.Anything).Return(custom1, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", ctx, claimMsg.Header.ID).Return(nil, nil) // Simulate pending confirm in same pin batch
	mdi.On("GetIdentityByName", ctx, custom1.Type, custom1.Namespace, custom1.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, custom1.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, "ns1", "0x12345").Return(nil, nil)
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
	mdm.On("GetMessageDataCached", ctx, mock.Anything).Return([]*fftypes.Data{claimData}, true, nil)

	bs.pendingConfirms[*claimMsg.Header.ID] = claimMsg

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, verifyMsg, []*fftypes.Data{verifyData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionConfirm}, action)
	assert.NoError(t, err)

	err = bs.finalizers[0](ctx)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestHandleDefinitionIdentityVerificationIncompleteClaimData(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	_, org1, claimMsg, _, verifyMsg, verifyData := testCustomClaimAndVerification(t)
	claimMsg.State = fftypes.MessageStateConfirmed

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", ctx, org1.ID).Return(org1, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", ctx, claimMsg.Header.ID).Return(claimMsg, nil)

	mdm := dh.data.(*datamocks.Manager)
	mdm.On("GetMessageDataCached", ctx, mock.Anything).Return([]*fftypes.Data{}, false, nil)

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, verifyMsg, []*fftypes.Data{verifyData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionConfirm}, action)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityVerificationClaimDataFail(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	_, org1, claimMsg, _, verifyMsg, verifyData := testCustomClaimAndVerification(t)
	claimMsg.State = fftypes.MessageStateConfirmed

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", ctx, org1.ID).Return(org1, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", ctx, claimMsg.Header.ID).Return(claimMsg, nil)

	mdm := dh.data.(*datamocks.Manager)
	mdm.On("GetMessageDataCached", ctx, mock.Anything).Return(nil, false, fmt.Errorf("pop"))

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, verifyMsg, []*fftypes.Data{verifyData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityVerificationClaimHashMismatchl(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	_, org1, claimMsg, _, verifyMsg, verifyData := testCustomClaimAndVerification(t)
	claimMsg.State = fftypes.MessageStateConfirmed
	claimMsg.Hash = fftypes.NewRandB32()

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", ctx, org1.ID).Return(org1, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", ctx, claimMsg.Header.ID).Return(claimMsg, nil)

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, verifyMsg, []*fftypes.Data{verifyData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityVerificationBeforeClaim(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	_, org1, claimMsg, _, verifyMsg, verifyData := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", ctx, org1.ID).Return(org1, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", ctx, claimMsg.Header.ID).Return(nil, nil)

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, verifyMsg, []*fftypes.Data{verifyData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionConfirm}, action)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityVerificationClaimLookupFail(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	_, org1, claimMsg, _, verifyMsg, verifyData := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", ctx, org1.ID).Return(org1, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", ctx, claimMsg.Header.ID).Return(nil, fmt.Errorf("pop"))

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, verifyMsg, []*fftypes.Data{verifyData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityVerificationWrongSigner(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	_, org1, _, _, verifyMsg, verifyData := testCustomClaimAndVerification(t)
	verifyMsg.Header.Author = "wrong"

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", ctx, org1.ID).Return(org1, nil)

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, verifyMsg, []*fftypes.Data{verifyData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityVerificationCheckParentNotFound(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	_, org1, _, _, verifyMsg, verifyData := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", ctx, org1.ID).Return(nil, nil)

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, verifyMsg, []*fftypes.Data{verifyData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityVerificationCheckParentFail(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	_, org1, _, _, verifyMsg, verifyData := testCustomClaimAndVerification(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", ctx, org1.ID).Return(nil, fmt.Errorf("pop"))

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, verifyMsg, []*fftypes.Data{verifyData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityVerificationInvalidPayload(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	iv := fftypes.IdentityVerification{
		Identity: testOrgIdentity(t, "org1").IdentityBase,
		// Missing message claim info
	}
	b, err := json.Marshal(&iv)
	assert.NoError(t, err)
	emptyObjectData := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:   fftypes.NewUUID(),
			Type: fftypes.MessageTypeBroadcast,
			Tag:  fftypes.SystemTagIdentityVerification,
		},
	}, []*fftypes.Data{emptyObjectData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.NoError(t, err)

	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityVerificationInvalidData(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:   fftypes.NewUUID(),
			Type: fftypes.MessageTypeBroadcast,
			Tag:  fftypes.SystemTagIdentityVerification,
		},
	}, []*fftypes.Data{}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.NoError(t, err)

	bs.assertNoFinalizers()
}
