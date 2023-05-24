// Copyright © 2023 Kaleido, Inc.
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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/assetmocks"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastTokenPoolInvalid(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mdm := ds.data.(*datamocks.Manager)

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "",
		Name:      "",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Published: true,
	}

	err := ds.DefineTokenPool(context.Background(), pool, false)
	assert.Regexp(t, "FF10420", err)

	mdm.AssertExpectations(t)
}

func TestBroadcastTokenPoolInvalidNonMultiparty(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = false

	mdm := ds.data.(*datamocks.Manager)

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "",
		Name:      "",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Published: false,
	}

	err := ds.DefineTokenPool(context.Background(), pool, false)
	assert.Regexp(t, "FF00140", err)

	mdm.AssertExpectations(t)
}

func TestBroadcastTokenPoolPublishNonMultiparty(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = false

	mdm := ds.data.(*datamocks.Manager)

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "",
		Name:      "",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Published: true,
	}

	err := ds.DefineTokenPool(context.Background(), pool, false)
	assert.Regexp(t, "FF10414", err)

	mdm.AssertExpectations(t)
}

func TestBroadcastTokenPoolInvalidNameMultiparty(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mdm := ds.data.(*datamocks.Manager)

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "",
		Name:      "",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Connector: "connector1",
		Published: true,
	}

	err := ds.DefineTokenPool(context.Background(), pool, false)
	assert.Regexp(t, "FF00140", err)

	mdm.AssertExpectations(t)
}

func TestDefineTokenPoolOk(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mdi := ds.database.(*databasemocks.Plugin)
	mdm := ds.data.(*datamocks.Manager)
	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &syncasyncmocks.Sender{}

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "mypool",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Connector: "connector1",
		Published: true,
	}

	mim.On("GetMultipartyRootOrg", ds.ctx).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)
	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(nil)
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("Send", ds.ctx).Return(nil)
	mdi.On("GetTokenPoolByNetworkName", ds.ctx, "ns1", "mypool").Return(nil, nil)

	err := ds.DefineTokenPool(ds.ctx, pool, false)
	assert.NoError(t, err)

	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestDefineTokenPoolkONonMultiparty(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = false

	mdb := ds.database.(*databasemocks.Plugin)
	mam := ds.assets.(*assetmocks.Manager)

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "mypool",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Connector: "connector1",
		Active:    true,
		Published: false,
	}

	mdb.On("InsertOrGetTokenPool", mock.Anything, pool).Return(nil, nil)
	mam.On("ActivateTokenPool", mock.Anything, pool).Return(nil)

	err := ds.DefineTokenPool(context.Background(), pool, false)
	assert.NoError(t, err)

	mdb.AssertExpectations(t)
	mam.AssertExpectations(t)
}

func TestDefineTokenPoolNonMultipartyTokenPoolFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	mdi := ds.database.(*databasemocks.Plugin)

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "mypool",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Connector: "connector1",
		Published: false,
	}

	mdi.On("InsertOrGetTokenPool", mock.Anything, pool).Return(nil, fmt.Errorf("pop"))

	err := ds.DefineTokenPool(context.Background(), pool, false)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestDefineTokenPoolBadName(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "///bad/////",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Connector: "connector1",
		Published: false,
	}

	err := ds.DefineTokenPool(context.Background(), pool, false)
	assert.Regexp(t, "FF00140", err)
}

func TestPublishTokenPool(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mdi := ds.database.(*databasemocks.Plugin)
	mam := ds.assets.(*assetmocks.Manager)
	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &syncasyncmocks.Sender{}

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "pool1",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Connector: "connector1",
		Published: false,
	}

	mam.On("GetTokenPoolByNameOrID", mock.Anything, "pool1").Return(pool, nil)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)
	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(nil)
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("Prepare", context.Background()).Return(nil)
	mms.On("Send", context.Background()).Return(nil)
	mdi.On("GetTokenPoolByNetworkName", mock.Anything, "ns1", "pool-shared").Return(nil, nil)
	mockRunAsGroupPassthrough(mdi)

	result, err := ds.PublishTokenPool(context.Background(), "pool1", "pool-shared", false)
	assert.NoError(t, err)
	assert.Equal(t, pool, result)
	assert.True(t, pool.Published)

	mdi.AssertExpectations(t)
	mam.AssertExpectations(t)
	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestPublishTokenPoolNonMultiparty(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = false

	_, err := ds.PublishTokenPool(context.Background(), "pool1", "pool-shared", false)
	assert.Regexp(t, "FF10414", err)
}

func TestPublishTokenPoolAlreadyPublished(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mam := ds.assets.(*assetmocks.Manager)
	mdi := ds.database.(*databasemocks.Plugin)

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "pool1",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Connector: "connector1",
		Published: true,
	}

	mam.On("GetTokenPoolByNameOrID", mock.Anything, "pool1").Return(pool, nil)
	mockRunAsGroupPassthrough(mdi)

	_, err := ds.PublishTokenPool(context.Background(), "pool1", "pool-shared", false)
	assert.Regexp(t, "FF10450", err)

	mam.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestPublishTokenPoolQueryFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mam := ds.assets.(*assetmocks.Manager)
	mdi := ds.database.(*databasemocks.Plugin)

	mam.On("GetTokenPoolByNameOrID", mock.Anything, "pool1").Return(nil, fmt.Errorf("pop"))
	mockRunAsGroupPassthrough(mdi)

	_, err := ds.PublishTokenPool(context.Background(), "pool1", "pool-shared", false)
	assert.EqualError(t, err, "pop")

	mam.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestPublishTokenPoolNetworkNameError(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mam := ds.assets.(*assetmocks.Manager)
	mdi := ds.database.(*databasemocks.Plugin)

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "pool1",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Connector: "connector1",
		Published: false,
	}

	mam.On("GetTokenPoolByNameOrID", mock.Anything, "pool1").Return(pool, nil)
	mdi.On("GetTokenPoolByNetworkName", mock.Anything, "ns1", "pool-shared").Return(nil, fmt.Errorf("pop"))
	mockRunAsGroupPassthrough(mdi)

	_, err := ds.PublishTokenPool(context.Background(), "pool1", "pool-shared", false)
	assert.EqualError(t, err, "pop")

	mam.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestPublishTokenPoolNetworkNameConflict(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mam := ds.assets.(*assetmocks.Manager)
	mdi := ds.database.(*databasemocks.Plugin)

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "pool1",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Connector: "connector1",
		Published: false,
	}

	mam.On("GetTokenPoolByNameOrID", mock.Anything, "pool1").Return(pool, nil)
	mdi.On("GetTokenPoolByNetworkName", mock.Anything, "ns1", "pool-shared").Return(&core.TokenPool{}, nil)
	mockRunAsGroupPassthrough(mdi)

	_, err := ds.PublishTokenPool(context.Background(), "pool1", "pool-shared", false)
	assert.Regexp(t, "FF10448", err)

	mam.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestPublishTokenPoolResolveFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mam := ds.assets.(*assetmocks.Manager)
	mim := ds.identity.(*identitymanagermocks.Manager)
	mdi := ds.database.(*databasemocks.Plugin)

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "pool1",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Connector: "connector1",
		Published: false,
	}

	mam.On("GetTokenPoolByNameOrID", mock.Anything, "pool1").Return(pool, nil)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(nil, fmt.Errorf("pop"))
	mdi.On("GetTokenPoolByNetworkName", mock.Anything, "ns1", "pool-shared").Return(nil, nil)
	mockRunAsGroupPassthrough(mdi)

	_, err := ds.PublishTokenPool(context.Background(), "pool1", "pool-shared", false)
	assert.EqualError(t, err, "pop")

	mam.AssertExpectations(t)
	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestPublishTokenPoolPrepareFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mdi := ds.database.(*databasemocks.Plugin)
	mam := ds.assets.(*assetmocks.Manager)
	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &syncasyncmocks.Sender{}

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "pool1",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Connector: "connector1",
		Published: false,
	}

	mam.On("GetTokenPoolByNameOrID", mock.Anything, "pool1").Return(pool, nil)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)
	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(nil)
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("Prepare", context.Background()).Return(fmt.Errorf("pop"))
	mdi.On("GetTokenPoolByNetworkName", mock.Anything, "ns1", "pool-shared").Return(nil, nil)
	mockRunAsGroupPassthrough(mdi)

	_, err := ds.PublishTokenPool(context.Background(), "pool1", "pool-shared", false)
	assert.EqualError(t, err, "pop")
	assert.True(t, pool.Published)

	mdi.AssertExpectations(t)
	mam.AssertExpectations(t)
	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestPublishTokenPoolSendFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mdi := ds.database.(*databasemocks.Plugin)
	mam := ds.assets.(*assetmocks.Manager)
	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &syncasyncmocks.Sender{}

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "pool1",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Connector: "connector1",
		Published: false,
	}

	mam.On("GetTokenPoolByNameOrID", mock.Anything, "pool1").Return(pool, nil)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)
	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(nil)
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("Prepare", context.Background()).Return(nil)
	mms.On("Send", context.Background()).Return(fmt.Errorf("pop"))
	mdi.On("GetTokenPoolByNetworkName", mock.Anything, "ns1", "pool-shared").Return(nil, nil)
	mockRunAsGroupPassthrough(mdi)

	_, err := ds.PublishTokenPool(context.Background(), "pool1", "pool-shared", false)
	assert.EqualError(t, err, "pop")
	assert.True(t, pool.Published)

	mdi.AssertExpectations(t)
	mam.AssertExpectations(t)
	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestPublishTokenPoolConfirm(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mdi := ds.database.(*databasemocks.Plugin)
	mam := ds.assets.(*assetmocks.Manager)
	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &syncasyncmocks.Sender{}

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "pool1",
		Type:      core.TokenTypeNonFungible,
		Locator:   "N1",
		Symbol:    "COIN",
		Connector: "connector1",
		Published: false,
	}

	mam.On("GetTokenPoolByNameOrID", mock.Anything, "pool1").Return(pool, nil)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)
	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(nil)
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("Prepare", context.Background()).Return(nil)
	mms.On("SendAndWait", context.Background()).Return(nil)
	mdi.On("GetTokenPoolByNetworkName", mock.Anything, "ns1", "pool-shared").Return(nil, nil)
	mockRunAsGroupPassthrough(mdi)

	_, err := ds.PublishTokenPool(context.Background(), "pool1", "pool-shared", true)
	assert.NoError(t, err)
	assert.True(t, pool.Published)

	mdi.AssertExpectations(t)
	mam.AssertExpectations(t)
	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}
