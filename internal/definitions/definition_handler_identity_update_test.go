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
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testIdentityUpdate(t *testing.T) (*fftypes.Identity, *fftypes.Message, *fftypes.Data, *fftypes.IdentityUpdate) {
	org1 := testOrgIdentity(t, "org1")
	org1.Parent = fftypes.NewUUID() // Not involved in verification for updates, just must not change

	iu := &fftypes.IdentityUpdate{
		Identity: org1.IdentityBase,
		Updates: fftypes.IdentityProfile{
			Profile: fftypes.JSONObject{
				"new": "profile",
			},
			Description: "new description",
		},
	}
	b, err := json.Marshal(&iu)
	assert.NoError(t, err)
	updateData := &fftypes.Data{
		ID:    fftypes.NewUUID(),
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	updateMsg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:     fftypes.NewUUID(),
			Type:   fftypes.MessageTypeDefinition,
			Tag:    fftypes.SystemTagIdentityUpdate,
			Topics: fftypes.FFStringArray{org1.Topic()},
			SignerRef: fftypes.SignerRef{
				Author: org1.DID,
				Key:    "0x12345",
			},
		},
	}

	return org1, updateMsg, updateData, iu
}

func TestHandleDefinitionIdentityUpdateOk(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	org1, updateMsg, updateData, iu := testIdentityUpdate(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", ctx, org1.ID).Return(org1, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("UpsertIdentity", ctx, mock.MatchedBy(func(identity *fftypes.Identity) bool {
		assert.Equal(t, *updateMsg.Header.ID, *identity.Messages.Update)
		assert.Equal(t, org1.IdentityBase, identity.IdentityBase)
		assert.Equal(t, iu.Updates, identity.IdentityProfile)
		return true
	}), database.UpsertOptimizationExisting).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(event *fftypes.Event) bool {
		return event.Type == fftypes.EventTypeIdentityUpdated
	})).Return(nil)

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, updateMsg, []*fftypes.Data{updateData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionConfirm}, action)
	assert.NoError(t, err)

	err = bs.finalizers[0](ctx)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleDefinitionIdentityUpdateUpsertFail(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	org1, updateMsg, updateData, _ := testIdentityUpdate(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", ctx, org1.ID).Return(org1, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("UpsertIdentity", ctx, mock.Anything, database.UpsertOptimizationExisting).Return(fmt.Errorf("pop"))

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, updateMsg, []*fftypes.Data{updateData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityInvalidIdentity(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	org1, updateMsg, updateData, _ := testIdentityUpdate(t)
	updateMsg.Header.Author = "wrong"

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", ctx, org1.ID).Return(org1, nil)

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, updateMsg, []*fftypes.Data{updateData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityNotFound(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	org1, updateMsg, updateData, _ := testIdentityUpdate(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", ctx, org1.ID).Return(nil, nil)

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, updateMsg, []*fftypes.Data{updateData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityLookupFail(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	org1, updateMsg, updateData, _ := testIdentityUpdate(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", ctx, org1.ID).Return(nil, fmt.Errorf("pop"))

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, updateMsg, []*fftypes.Data{updateData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityValidateFail(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	org1 := testOrgIdentity(t, "org1")
	iu := &fftypes.IdentityUpdate{
		Identity: org1.IdentityBase,
	}
	iu.Identity.DID = "wrong"
	b, err := json.Marshal(&iu)
	assert.NoError(t, err)
	updateData := &fftypes.Data{
		ID:    fftypes.NewUUID(),
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	updateMsg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:     fftypes.NewUUID(),
			Type:   fftypes.MessageTypeDefinition,
			Tag:    fftypes.SystemTagIdentityUpdate,
			Topics: fftypes.FFStringArray{org1.Topic()},
			SignerRef: fftypes.SignerRef{
				Author: org1.DID,
				Key:    "0x12345",
			},
		},
	}

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, updateMsg, []*fftypes.Data{updateData}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.NoError(t, err)

	bs.assertNoFinalizers()
}

func TestHandleDefinitionIdentityMissingData(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	ctx := context.Background()

	org1 := testOrgIdentity(t, "org1")
	updateMsg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:     fftypes.NewUUID(),
			Type:   fftypes.MessageTypeDefinition,
			Tag:    fftypes.SystemTagIdentityUpdate,
			Topics: fftypes.FFStringArray{org1.Topic()},
			SignerRef: fftypes.SignerRef{
				Author: org1.DID,
				Key:    "0x12345",
			},
		},
	}

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, updateMsg, []*fftypes.Data{}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.NoError(t, err)

	bs.assertNoFinalizers()
}
