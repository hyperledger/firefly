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
	"github.com/hyperledger/firefly/mocks/assetmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newPoolDefinition() *core.TokenPoolDefinition {
	pool := &core.TokenPool{
		ID:          fftypes.NewUUID(),
		Namespace:   "ns1",
		Name:        "name1",
		NetworkName: "name1",
		Type:        core.TokenTypeFungible,
		Locator:     "12345",
		Symbol:      "COIN",
		TX: core.TransactionRef{
			Type: core.TransactionTypeTokenPool,
			ID:   fftypes.NewUUID(),
		},
		Connector: "remote1",
		Published: true,
	}
	return &core.TokenPoolDefinition{
		Pool: pool,
	}
}

func buildPoolDefinitionMessage(definition *core.TokenPoolDefinition) (*core.Message, core.DataArray, error) {
	msg := &core.Message{
		Header: core.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: core.SystemTagDefinePool,
			SignerRef: core.SignerRef{
				Author: "firefly:org1",
			},
		},
	}
	b, err := json.Marshal(definition)
	if err != nil {
		return nil, nil, err
	}
	data := core.DataArray{{
		Value: fftypes.JSONAnyPtrBytes(b),
	}}
	return msg, data, nil
}

func TestHandleDefinitionBroadcastTokenPoolActivateOK(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	definition := newPoolDefinition()
	pool := definition.Pool
	msg, data, err := buildPoolDefinitionMessage(definition)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mam := sh.assets.(*assetmocks.Manager)
	mim := sh.identity.(*identitymanagermocks.Manager)
	mdi.On("InsertOrGetTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID && p.Connector == "connector1"
	})).Return(nil, nil)
	mam.On("ActivateTokenPool", context.Background(), mock.AnythingOfType("*core.TokenPool")).Return(nil)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: core.ActionWait, CustomCorrelator: pool.ID}, action)
	assert.NoError(t, err)

	err = bs.RunPreFinalize(context.Background())
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mam.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolBadConnector(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	definition := newPoolDefinition()
	pool := definition.Pool
	pool.NetworkName = "//bad"
	msg, data, err := buildPoolDefinitionMessage(definition)
	assert.NoError(t, err)

	mim := sh.identity.(*identitymanagermocks.Manager)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: core.ActionReject, CustomCorrelator: pool.ID}, action)
	assert.Regexp(t, "FF10403", err)

	err = bs.RunPreFinalize(context.Background())
	assert.NoError(t, err)

	mim.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolNameExists(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	definition := newPoolDefinition()
	pool := definition.Pool
	msg, data, err := buildPoolDefinitionMessage(definition)
	assert.NoError(t, err)

	existing := &core.TokenPool{
		Name: "name1",
	}

	mdi := sh.database.(*databasemocks.Plugin)
	mim := sh.identity.(*identitymanagermocks.Manager)
	mdi.On("InsertOrGetTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID && p.Name == "name1"
	})).Return(existing, nil)
	mdi.On("InsertOrGetTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID && p.Name == "name1-1"
	})).Return(nil, nil)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: core.ActionWait, CustomCorrelator: pool.ID}, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestHandleDefinitionLocalTokenPoolNameExists(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	definition := newPoolDefinition()
	pool := definition.Pool
	pool.Published = false

	existing := &core.TokenPool{
		Name: "name1",
	}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("InsertOrGetTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Name == "name1"
	})).Return(existing, nil)

	action, err := sh.handleTokenPoolDefinition(context.Background(), &bs.BatchState, pool, true)
	assert.Equal(t, HandlerResult{Action: core.ActionReject, CustomCorrelator: pool.ID}, action)
	assert.Error(t, err)

	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionBroadcastTokenPoolNetworkNameExists(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	definition := newPoolDefinition()
	pool := definition.Pool
	msg, data, err := buildPoolDefinitionMessage(definition)
	assert.NoError(t, err)

	existing := &core.TokenPool{
		NetworkName: "name1",
	}

	mdi := sh.database.(*databasemocks.Plugin)
	mim := sh.identity.(*identitymanagermocks.Manager)
	mdi.On("InsertOrGetTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(existing, nil)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: core.ActionReject, CustomCorrelator: pool.ID}, action)
	assert.Error(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionBroadcastTokenPoolExistingConfirmed(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	definition := newPoolDefinition()
	pool := definition.Pool
	msg, data, err := buildPoolDefinitionMessage(definition)
	assert.NoError(t, err)

	existing := &core.TokenPool{
		ID:      pool.ID,
		State:   core.TokenPoolStateConfirmed,
		Message: msg.Header.ID,
	}

	mdi := sh.database.(*databasemocks.Plugin)
	mim := sh.identity.(*identitymanagermocks.Manager)
	mdi.On("InsertOrGetTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(existing, nil)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: core.ActionConfirm, CustomCorrelator: pool.ID}, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolExistingWaiting(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	definition := newPoolDefinition()
	pool := definition.Pool
	msg, data, err := buildPoolDefinitionMessage(definition)
	assert.NoError(t, err)

	existing := &core.TokenPool{
		ID:      pool.ID,
		State:   core.TokenPoolStatePending,
		Message: msg.Header.ID,
	}

	mdi := sh.database.(*databasemocks.Plugin)
	mim := sh.identity.(*identitymanagermocks.Manager)
	mdi.On("InsertOrGetTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(existing, nil)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: core.ActionWait, CustomCorrelator: pool.ID}, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolExistingPublish(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	definition := newPoolDefinition()
	pool := definition.Pool
	msg, data, err := buildPoolDefinitionMessage(definition)
	assert.NoError(t, err)

	existing := &core.TokenPool{
		ID:    pool.ID,
		State: core.TokenPoolStateConfirmed,
		Name:  "existing-pool",
	}
	newPool := *pool
	newPool.Name = existing.Name
	newPool.Connector = "connector1"
	newPool.Published = true
	newPool.Message = msg.Header.ID
	newPool.State = core.TokenPoolStatePending

	mdi := sh.database.(*databasemocks.Plugin)
	mim := sh.identity.(*identitymanagermocks.Manager)

	mdi.On("InsertOrGetTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(existing, nil)
	mdi.On("UpsertTokenPool", context.Background(), &newPool, database.UpsertOptimizationExisting).Return(nil)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: core.ActionConfirm, CustomCorrelator: pool.ID}, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolExistingPublishUpsertFail(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	definition := newPoolDefinition()
	pool := definition.Pool
	msg, data, err := buildPoolDefinitionMessage(definition)
	assert.NoError(t, err)

	existing := &core.TokenPool{
		ID:    pool.ID,
		State: core.TokenPoolStateConfirmed,
		Name:  "existing-pool",
	}
	newPool := *pool
	newPool.Name = existing.Name
	newPool.Connector = "connector1"
	newPool.Published = true
	newPool.Message = msg.Header.ID
	newPool.State = core.TokenPoolStatePending

	mdi := sh.database.(*databasemocks.Plugin)
	mim := sh.identity.(*identitymanagermocks.Manager)

	mdi.On("InsertOrGetTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(existing, nil)
	mdi.On("UpsertTokenPool", context.Background(), &newPool, database.UpsertOptimizationExisting).Return(fmt.Errorf("pop"))
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: core.ActionRetry, CustomCorrelator: pool.ID}, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolExistingPublishOrgFail(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	definition := newPoolDefinition()
	pool := definition.Pool
	msg, data, err := buildPoolDefinitionMessage(definition)
	assert.NoError(t, err)

	existing := &core.TokenPool{
		ID:    pool.ID,
		State: core.TokenPoolStateConfirmed,
		Name:  "existing-pool",
	}
	newPool := *pool
	newPool.Name = existing.Name
	newPool.Connector = "connector1"
	newPool.Published = true
	newPool.Message = msg.Header.ID
	newPool.State = core.TokenPoolStatePending

	mim := sh.identity.(*identitymanagermocks.Manager)

	mim.On("GetMultipartyRootOrg", context.Background()).Return(nil, fmt.Errorf("pop"))

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: core.ActionRetry}, action)
	assert.EqualError(t, err, "pop")

	mim.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolExistingPublishOrgMismatch(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	definition := newPoolDefinition()
	pool := definition.Pool
	msg, data, err := buildPoolDefinitionMessage(definition)
	assert.NoError(t, err)

	existing := &core.TokenPool{
		ID:    pool.ID,
		State: core.TokenPoolStateConfirmed,
		Name:  "existing-pool",
	}
	newPool := *pool
	newPool.Name = existing.Name
	newPool.Connector = "connector1"
	newPool.Published = true
	newPool.Message = msg.Header.ID
	newPool.State = core.TokenPoolStatePending

	mdi := sh.database.(*databasemocks.Plugin)
	mim := sh.identity.(*identitymanagermocks.Manager)

	mdi.On("InsertOrGetTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(existing, nil)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org2",
		},
	}, nil)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: core.ActionReject, CustomCorrelator: pool.ID}, action)
	assert.Regexp(t, "FF10407", err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolFailUpsert(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	definition := newPoolDefinition()
	pool := definition.Pool
	msg, data, err := buildPoolDefinitionMessage(definition)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mim := sh.identity.(*identitymanagermocks.Manager)
	mdi.On("InsertOrGetTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(nil, fmt.Errorf("pop"))
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: core.ActionRetry}, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionBroadcastTokenPoolActivateFail(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	definition := newPoolDefinition()
	pool := definition.Pool
	msg, data, err := buildPoolDefinitionMessage(definition)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mam := sh.assets.(*assetmocks.Manager)
	mim := sh.identity.(*identitymanagermocks.Manager)
	mdi.On("InsertOrGetTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(nil, nil)
	mam.On("ActivateTokenPool", context.Background(), mock.AnythingOfType("*core.TokenPool")).Return(fmt.Errorf("pop"))
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: core.ActionWait, CustomCorrelator: pool.ID}, action)
	assert.NoError(t, err)

	err = bs.RunPreFinalize(context.Background())
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mam.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolValidateFail(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	definition := &core.TokenPoolDefinition{
		Pool: &core.TokenPool{},
	}
	msg, data, err := buildPoolDefinitionMessage(definition)
	assert.NoError(t, err)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: core.ActionReject}, action)
	assert.Error(t, err)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionBroadcastTokenPoolBadMessage(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	msg := &core.Message{
		Header: core.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: core.SystemTagDefinePool,
		},
	}

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, nil, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: core.ActionReject}, action)
	assert.Error(t, err)
	bs.assertNoFinalizers()
}
