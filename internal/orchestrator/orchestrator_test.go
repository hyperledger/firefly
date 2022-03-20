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

package orchestrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/dataexchange/dxfactory"
	"github.com/hyperledger/firefly/internal/restclient"
	"github.com/hyperledger/firefly/internal/tokens/tifactory"
	"github.com/hyperledger/firefly/mocks/assetmocks"
	"github.com/hyperledger/firefly/mocks/batchmocks"
	"github.com/hyperledger/firefly/mocks/batchpinmocks"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/contractmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/eventmocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/identitymocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/networkmapmocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger/firefly/mocks/shareddownloadmocks"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const configDir = "../../test/data/config"

type testOrchestrator struct {
	orchestrator

	mdi *databasemocks.Plugin
	mdm *datamocks.Manager
	mbm *broadcastmocks.Manager
	mba *batchmocks.Manager
	mem *eventmocks.EventManager
	mnm *networkmapmocks.Manager
	mps *sharedstoragemocks.Plugin
	mpm *privatemessagingmocks.Manager
	mbi *blockchainmocks.Plugin
	mii *identitymocks.Plugin
	mim *identitymanagermocks.Manager
	mdx *dataexchangemocks.Plugin
	mam *assetmocks.Manager
	mti *tokenmocks.Plugin
	mcm *contractmocks.Manager
	mmi *metricsmocks.Manager
	mom *operationmocks.Manager
	mbp *batchpinmocks.Submitter
	mth *txcommonmocks.Helper
	msd *shareddownloadmocks.Manager
}

func newTestOrchestrator() *testOrchestrator {
	config.Reset()
	ctx, cancel := context.WithCancel(context.Background())
	tor := &testOrchestrator{
		orchestrator: orchestrator{
			ctx:       ctx,
			cancelCtx: cancel,
		},
		mdi: &databasemocks.Plugin{},
		mdm: &datamocks.Manager{},
		mbm: &broadcastmocks.Manager{},
		mba: &batchmocks.Manager{},
		mem: &eventmocks.EventManager{},
		mnm: &networkmapmocks.Manager{},
		mps: &sharedstoragemocks.Plugin{},
		mpm: &privatemessagingmocks.Manager{},
		mbi: &blockchainmocks.Plugin{},
		mii: &identitymocks.Plugin{},
		mim: &identitymanagermocks.Manager{},
		mdx: &dataexchangemocks.Plugin{},
		mam: &assetmocks.Manager{},
		mti: &tokenmocks.Plugin{},
		mcm: &contractmocks.Manager{},
		mmi: &metricsmocks.Manager{},
		mom: &operationmocks.Manager{},
		mbp: &batchpinmocks.Submitter{},
		mth: &txcommonmocks.Helper{},
		msd: &shareddownloadmocks.Manager{},
	}
	tor.orchestrator.database = tor.mdi
	tor.orchestrator.data = tor.mdm
	tor.orchestrator.batch = tor.mba
	tor.orchestrator.broadcast = tor.mbm
	tor.orchestrator.events = tor.mem
	tor.orchestrator.networkmap = tor.mnm
	tor.orchestrator.sharedstorage = tor.mps
	tor.orchestrator.messaging = tor.mpm
	tor.orchestrator.blockchain = tor.mbi
	tor.orchestrator.identity = tor.mim
	tor.orchestrator.identityPlugin = tor.mii
	tor.orchestrator.dataexchange = tor.mdx
	tor.orchestrator.assets = tor.mam
	tor.orchestrator.contracts = tor.mcm
	tor.orchestrator.tokens = map[string]tokens.Plugin{"token": tor.mti}
	tor.orchestrator.metrics = tor.mmi
	tor.orchestrator.operations = tor.mom
	tor.orchestrator.batchpin = tor.mbp
	tor.orchestrator.sharedDownload = tor.msd
	tor.orchestrator.txHelper = tor.mth
	tor.mdi.On("Name").Return("mock-di").Maybe()
	tor.mem.On("Name").Return("mock-ei").Maybe()
	tor.mps.On("Name").Return("mock-ps").Maybe()
	tor.mbi.On("Name").Return("mock-bi").Maybe()
	tor.mii.On("Name").Return("mock-ii").Maybe()
	tor.mdx.On("Name").Return("mock-dx").Maybe()
	tor.mam.On("Name").Return("mock-am").Maybe()
	tor.mti.On("Name").Return("mock-tk").Maybe()
	tor.mcm.On("Name").Return("mock-cm").Maybe()
	tor.mmi.On("Name").Return("mock-mm").Maybe()
	return tor
}

func TestNewOrchestrator(t *testing.T) {
	or := NewOrchestrator()
	assert.NotNil(t, or)
}

func TestBadDatabasePlugin(t *testing.T) {
	or := newTestOrchestrator()
	config.Set(config.DatabaseType, "wrong")
	or.database = nil
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10122.*wrong", err)
}

func TestBadDatabaseInitFail(t *testing.T) {
	or := newTestOrchestrator()
	config.Set(config.DatabaseType, "wrong")
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.EqualError(t, err, "pop")
}

func TestBadDatabasePreInitMode(t *testing.T) {
	or := newTestOrchestrator()
	config.Set(config.AdminPreinit, true)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.NoError(t, err)
	err = or.Start()
	assert.NoError(t, err)
}

func TestBadIdentityPlugin(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	config.Set(config.IdentityType, "wrong")
	or.identityPlugin = nil
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10212.*wrong", err)
}

func TestBadIdentityInitFail(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.EqualError(t, err, "pop")
}

func TestBadBlockchainPlugin(t *testing.T) {
	or := newTestOrchestrator()
	config.Set(config.BlockchainType, "wrong")
	or.blockchain = nil
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10110.*wrong", err)
}

func TestBlockchainInitFail(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.EqualError(t, err, "pop")
}

func TestBlockchainInitGetConfigRecordsFail(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.EqualError(t, err, "pop")
}

func TestBlockchainInitMergeConfigRecordsFail(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{
		{
			Key:   "pizza.toppings",
			Value: fftypes.JSONAnyPtr("cheese, pepperoni, mushrooms"),
		},
	}, nil, nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.EqualError(t, err, "invalid character 'c' looking for beginning of value")
}

func TestBadSharedStoragePlugin(t *testing.T) {
	or := newTestOrchestrator()
	config.Set(config.SharedStorageType, "wrong")
	or.sharedstorage = nil
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10134.*wrong", err)
}

func TestBadSharedStoragePluginOldConfig(t *testing.T) {
	or := newTestOrchestrator()
	config.Set(config.PublicStorageType, "wrong")
	or.sharedstorage = nil
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10134.*wrong", err)
}

func TestBadSharedStorageInitFail(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mps.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.EqualError(t, err, "pop")
}

func TestBadDataExchangePlugin(t *testing.T) {
	or := newTestOrchestrator()
	config.Set(config.DataexchangeType, "wrong")
	or.dataexchange = nil
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mps.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10213.*wrong", err)
}

func TestBadDataExchangeInitFail(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mps.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return([]*fftypes.Identity{}, nil, nil)
	or.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.EqualError(t, err, "pop")
}

func TestDataExchangePluginOldName(t *testing.T) {
	or := newTestOrchestrator()
	dxfactory.InitPrefix(dataexchangeConfig)
	config.Set(config.DataexchangeType, "https")
	dataexchangeConfig.SubPrefix("https").Set(restclient.HTTPConfigURL, "http://test")
	or.dataexchange = nil
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mps.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return([]*fftypes.Identity{}, nil, nil)
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.EqualError(t, err, "pop")
}

func TestBadTokensPlugin(t *testing.T) {
	or := newTestOrchestrator()
	tifactory.InitPrefix(tokensConfig)
	tokensConfig.AddKnownKey(tokens.TokensConfigName, "text")
	tokensConfig.AddKnownKey(tokens.TokensConfigConnector, "wrong")
	config.Set("tokens", []fftypes.JSONObject{{}})
	or.tokens = nil
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mps.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return([]*fftypes.Identity{}, nil, nil)
	or.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, nil)
	or.mdi.On("UpsertNamespace", mock.Anything, mock.Anything, true).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10272.*wrong", err)
}

func TestBadTokensPluginNoConnector(t *testing.T) {
	or := newTestOrchestrator()
	tokensConfig.AddKnownKey(tokens.TokensConfigName, "test")
	tokensConfig.AddKnownKey(tokens.TokensConfigConnector)
	tokensConfig.AddKnownKey(tokens.TokensConfigPlugin)
	config.Set("tokens", []fftypes.JSONObject{{}})
	or.tokens = nil
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("VerifyIdentitySyntax", mock.Anything, mock.Anything, mock.Anything).Return("", nil)
	or.mps.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return([]*fftypes.Identity{}, nil, nil)
	or.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, nil)
	or.mdi.On("UpsertNamespace", mock.Anything, mock.Anything, true).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10273", err)
}

func TestBadTokensPluginNoName(t *testing.T) {
	or := newTestOrchestrator()
	tifactory.InitPrefix(tokensConfig)
	tokensConfig.AddKnownKey(tokens.TokensConfigName)
	tokensConfig.AddKnownKey(tokens.TokensConfigConnector, "wrong")
	config.Set("tokens", []fftypes.JSONObject{{}})
	or.tokens = nil
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mps.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return([]*fftypes.Identity{}, nil, nil)
	or.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, nil)
	or.mdi.On("UpsertNamespace", mock.Anything, mock.Anything, true).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10273", err)
}

func TestBadTokensPluginInvalidName(t *testing.T) {
	or := newTestOrchestrator()
	tifactory.InitPrefix(tokensConfig)
	tokensConfig.AddKnownKey(tokens.TokensConfigName, "!wrong")
	tokensConfig.AddKnownKey(tokens.TokensConfigConnector, "text")
	config.Set("tokens", []fftypes.JSONObject{{}})
	or.tokens = nil
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mps.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return([]*fftypes.Identity{}, nil, nil)
	or.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, nil)
	or.mdi.On("UpsertNamespace", mock.Anything, mock.Anything, true).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10131.*'name'", err)
}

func TestBadTokensPluginNoType(t *testing.T) {
	or := newTestOrchestrator()
	tifactory.InitPrefix(tokensConfig)
	tokensConfig.AddKnownKey(tokens.TokensConfigName, "text")
	tokensConfig.AddKnownKey(tokens.TokensConfigConnector)
	config.Set("tokens", []fftypes.JSONObject{{}})
	or.tokens = nil
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("VerifyIdentitySyntax", mock.Anything, mock.Anything, mock.Anything).Return("", nil)
	or.mps.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return([]*fftypes.Identity{}, nil, nil)
	or.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, nil)
	or.mdi.On("UpsertNamespace", mock.Anything, mock.Anything, true).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10273", err)
}

func TestGoodTokensPlugin(t *testing.T) {
	or := newTestOrchestrator()
	tokensConfig = config.NewPluginConfig("tokens").Array()
	tifactory.InitPrefix(tokensConfig)
	tokensConfig.AddKnownKey(tokens.TokensConfigName, "test")
	tokensConfig.AddKnownKey(tokens.TokensConfigConnector, "https")
	tokensConfig.AddKnownKey(restclient.HTTPConfigURL, "test")
	config.Set("tokens", []fftypes.JSONObject{{}})
	or.tokens = nil
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mps.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return([]*fftypes.Identity{}, nil, nil)
	or.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, nil)
	or.mdi.On("UpsertNamespace", mock.Anything, mock.Anything, true).Return(nil)
	or.mti.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.NoError(t, err)
}

func TestInitMessagingComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	or.database = nil
	or.messaging = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitEventsComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	or.database = nil
	or.events = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitNetworkMapComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	or.database = nil
	or.networkmap = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitSharedStorageDownloadComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	or.database = nil
	or.sharedDownload = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitBatchComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	or.database = nil
	or.batch = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitBroadcastComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	or.database = nil
	or.broadcast = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitDataComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	or.database = nil
	or.data = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitMetricsComponent(t *testing.T) {
	or := newTestOrchestrator()
	or.metrics = nil
	or.initComponents(context.Background())
}

func TestInitIdentityComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	or.database = nil
	or.identity = nil
	or.txHelper = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitAssetsComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	or.database = nil
	or.assets = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitContractsComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	or.database = nil
	or.contracts = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitBatchPinComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	or.database = nil
	or.batchpin = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitOperationsComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	or.database = nil
	or.operations = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestStartBatchFail(t *testing.T) {
	config.Reset()
	or := newTestOrchestrator()
	or.mba.On("Start").Return(fmt.Errorf("pop"))
	or.mbi.On("Start").Return(nil)
	err := or.Start()
	assert.EqualError(t, err, "pop")
}

func TestStartTokensFail(t *testing.T) {
	config.Reset()
	or := newTestOrchestrator()
	or.mbi.On("Start").Return(nil)
	or.mba.On("Start").Return(nil)
	or.mem.On("Start").Return(nil)
	or.mbm.On("Start").Return(nil)
	or.mpm.On("Start").Return(nil)
	or.mam.On("Start").Return(nil)
	or.msd.On("Start").Return(nil)
	or.mti.On("Start").Return(fmt.Errorf("pop"))
	err := or.Start()
	assert.EqualError(t, err, "pop")
}

func TestStartStopOk(t *testing.T) {
	config.Reset()
	or := newTestOrchestrator()
	or.mbi.On("Start").Return(nil)
	or.mba.On("Start").Return(nil)
	or.mem.On("Start").Return(nil)
	or.mbm.On("Start").Return(nil)
	or.mpm.On("Start").Return(nil)
	or.mam.On("Start").Return(nil)
	or.mti.On("Start").Return(nil)
	or.mmi.On("Start").Return(nil)
	or.msd.On("Start").Return(nil)
	or.mbi.On("WaitStop").Return(nil)
	or.mba.On("WaitStop").Return(nil)
	or.mem.On("WaitStop").Return(nil)
	or.mbm.On("WaitStop").Return(nil)
	or.mam.On("WaitStop").Return(nil)
	or.mti.On("WaitStop").Return(nil)
	or.mdm.On("WaitStop").Return(nil)
	or.msd.On("WaitStop").Return(nil)
	err := or.Start()
	assert.NoError(t, err)
	or.WaitStop()
	or.WaitStop() // swallows dups
}

func TestInitNamespacesBadName(t *testing.T) {
	or := newTestOrchestrator()
	config.Reset()
	config.Set(config.NamespacesPredefined, fftypes.JSONObjectArray{
		{"name": "!Badness"},
	})
	err := or.initNamespaces(context.Background())
	assert.Regexp(t, "FF10131", err)
}

func TestInitNamespacesGetFail(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	err := or.initNamespaces(context.Background())
	assert.Regexp(t, "pop", err)
}

func TestInitNamespacesUpsertFail(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, nil)
	or.mdi.On("UpsertNamespace", mock.Anything, mock.Anything, true).Return(fmt.Errorf("pop"))
	err := or.initNamespaces(context.Background())
	assert.Regexp(t, "pop", err)
}

func TestInitNamespacesUpsertNotNeeded(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(&fftypes.Namespace{
		Type: fftypes.NamespaceTypeBroadcast, // any broadcasted NS will not be updated
	}, nil)
	err := or.initNamespaces(context.Background())
	assert.NoError(t, err)
}

func TestInitNamespacesDefaultMissing(t *testing.T) {
	or := newTestOrchestrator()
	config.Set(config.NamespacesPredefined, fftypes.JSONObjectArray{})
	err := or.initNamespaces(context.Background())
	assert.Regexp(t, "FF10166", err)
}

func TestInitNamespacesDupName(t *testing.T) {
	or := newTestOrchestrator()
	config.Set(config.NamespacesPredefined, fftypes.JSONObjectArray{
		{"name": "ns1"},
		{"name": "ns2"},
		{"name": "ns2"},
	})
	config.Set(config.NamespacesDefault, "ns1")
	nsList, err := or.getPrefdefinedNamespaces(context.Background())
	assert.NoError(t, err)
	assert.Len(t, nsList, 3)
	assert.Equal(t, fftypes.SystemNamespace, nsList[0].Name)
	assert.Equal(t, "ns1", nsList[1].Name)
	assert.Equal(t, "ns2", nsList[2].Name)
}

func TestInitOK(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{}, nil, nil)
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mii.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mps.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return([]*fftypes.Identity{}, nil, nil)
	or.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, nil)
	or.mdi.On("UpsertNamespace", mock.Anything, mock.Anything, true).Return(nil)
	or.mti.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	or.mmi.On("Init").Return(nil)
	err := config.ReadConfig(configDir + "/firefly.core.yaml")
	assert.NoError(t, err)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err = or.Init(ctx, cancelCtx)
	assert.NoError(t, err)

	assert.False(t, or.IsPreInit())
	assert.Equal(t, or.mbm, or.Broadcast())
	assert.Equal(t, or.mpm, or.PrivateMessaging())
	assert.Equal(t, or.mem, or.Events())
	assert.Equal(t, or.mba, or.BatchManager())
	assert.Equal(t, or.mnm, or.NetworkMap())
	assert.Equal(t, or.mdm, or.Data())
	assert.Equal(t, or.mam, or.Assets())
	assert.Equal(t, or.mcm, or.Contracts())
	assert.Equal(t, or.mmi, or.Metrics())
	assert.Equal(t, or.mom, or.Operations())
}

func TestInitDataExchangeGetNodesFail(t *testing.T) {
	or := newTestOrchestrator()

	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	err := or.initDataExchange(or.ctx)
	assert.EqualError(t, err, "pop")
}

func TestInitDataExchangeWithNodes(t *testing.T) {
	or := newTestOrchestrator()

	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return([]*fftypes.Identity{{}}, nil, nil)
	or.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := or.initDataExchange(or.ctx)
	assert.NoError(t, err)
}
