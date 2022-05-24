// Copyright © 2022 Kaleido, Inc.
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

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/blockchain/bifactory"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/database/difactory"
	"github.com/hyperledger/firefly/internal/dataexchange/dxfactory"
	"github.com/hyperledger/firefly/internal/identity/iifactory"
	"github.com/hyperledger/firefly/internal/sharedstorage/ssfactory"
	"github.com/hyperledger/firefly/internal/tokens/tifactory"
	"github.com/hyperledger/firefly/mocks/admineventsmocks"
	"github.com/hyperledger/firefly/mocks/assetmocks"
	"github.com/hyperledger/firefly/mocks/batchmocks"
	"github.com/hyperledger/firefly/mocks/batchpinmocks"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/contractmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/definitionsmocks"
	"github.com/hyperledger/firefly/mocks/eventmocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/identitymocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/namespacemocks"
	"github.com/hyperledger/firefly/mocks/networkmapmocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger/firefly/mocks/shareddownloadmocks"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/identity"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
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
	mae *admineventsmocks.Manager
	mdh *definitionsmocks.DefinitionHandler
	mns *namespacemocks.Manager
}

func (tor *testOrchestrator) cleanup(t *testing.T) {
	tor.mdi.AssertExpectations(t)
	tor.mdm.AssertExpectations(t)
	tor.mbm.AssertExpectations(t)
	tor.mba.AssertExpectations(t)
	tor.mem.AssertExpectations(t)
	tor.mnm.AssertExpectations(t)
	tor.mps.AssertExpectations(t)
	tor.mpm.AssertExpectations(t)
	tor.mbi.AssertExpectations(t)
	tor.mii.AssertExpectations(t)
	tor.mim.AssertExpectations(t)
	tor.mdx.AssertExpectations(t)
	tor.mam.AssertExpectations(t)
	tor.mti.AssertExpectations(t)
	tor.mcm.AssertExpectations(t)
	tor.mmi.AssertExpectations(t)
	tor.mom.AssertExpectations(t)
	tor.mbp.AssertExpectations(t)
	tor.mth.AssertExpectations(t)
	tor.msd.AssertExpectations(t)
	tor.mae.AssertExpectations(t)
	tor.mdh.AssertExpectations(t)
	tor.mns.AssertExpectations(t)
}

func newTestOrchestrator() *testOrchestrator {
	coreconfig.Reset()
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
		mae: &admineventsmocks.Manager{},
		mdh: &definitionsmocks.DefinitionHandler{},
		mns: &namespacemocks.Manager{},
	}
	tor.orchestrator.databases = map[string]database.Plugin{"postgres": tor.mdi}
	tor.orchestrator.data = tor.mdm
	tor.orchestrator.batch = tor.mba
	tor.orchestrator.broadcast = tor.mbm
	tor.orchestrator.events = tor.mem
	tor.orchestrator.networkmap = tor.mnm
	tor.orchestrator.sharedstoragePlugins = map[string]sharedstorage.Plugin{"ipfs": tor.mps}
	tor.orchestrator.messaging = tor.mpm
	tor.orchestrator.identity = tor.mim
	tor.orchestrator.identityPlugins = map[string]identity.Plugin{"identity": tor.mii}
	tor.orchestrator.dataexchangePlugins = map[string]dataexchange.Plugin{"ffdx": tor.mdx}
	tor.orchestrator.assets = tor.mam
	tor.orchestrator.contracts = tor.mcm
	tor.orchestrator.tokens = map[string]tokens.Plugin{"token": tor.mti}
	tor.orchestrator.blockchains = map[string]blockchain.Plugin{"ethereum": tor.mbi}
	tor.orchestrator.metrics = tor.mmi
	tor.orchestrator.operations = tor.mom
	tor.orchestrator.batchpin = tor.mbp
	tor.orchestrator.sharedDownload = tor.msd
	tor.orchestrator.adminEvents = tor.mae
	tor.orchestrator.txHelper = tor.mth
	tor.orchestrator.definitions = tor.mdh
	tor.orchestrator.namespace = tor.mns
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
	or := NewOrchestrator(true)
	assert.NotNil(t, or)
}

func TestBadDeprecatedDatabasePlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	difactory.InitConfigDeprecated(deprecatedDatabaseConfig)
	deprecatedDatabaseConfig.Set(coreconfig.PluginConfigType, "wrong")
	or.databases = nil
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10122.*wrong", err)
}

func TestBadDeprecatedDatabaseInitFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	difactory.InitConfigDeprecated(deprecatedDatabaseConfig)
	deprecatedDatabaseConfig.AddKnownKey(coreconfig.PluginConfigType, "test")
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	ctx := context.Background()
	err := or.initDeprecatedDatabasePlugin(ctx, or.mdi)
	assert.EqualError(t, err, "pop")
}

func TestDatabaseGetPlugins(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	difactory.InitConfig(databaseConfig)
	config.Set("plugins.database", []fftypes.JSONObject{{}})
	databaseConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	databaseConfig.AddKnownKey(coreconfig.PluginConfigType, "postgres")
	ctx := context.Background()
	plugins, err := or.getDatabasePlugins(ctx)
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDatabaseUnknownPlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	difactory.InitConfig(databaseConfig)
	config.Set("plugins.database", []fftypes.JSONObject{{}})
	databaseConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	databaseConfig.AddKnownKey(coreconfig.PluginConfigType, "unknown")
	ctx := context.Background()
	plugins, err := or.getDatabasePlugins(ctx)
	assert.Nil(t, plugins)
	assert.Error(t, err)
}

func TestDatabaseGetPluginsNoName(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	difactory.InitConfig(databaseConfig)
	config.Set("plugins.database", []fftypes.JSONObject{{}})
	databaseConfig.AddKnownKey(coreconfig.PluginConfigType, "postgres")
	ctx := context.Background()
	plugins, err := or.getDatabasePlugins(ctx)
	assert.Nil(t, plugins)
	assert.Error(t, err)
}

func TestDatabaseGetPluginsBadName(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	difactory.InitConfig(databaseConfig)
	config.Set("plugins.database", []fftypes.JSONObject{{}})
	databaseConfig.AddKnownKey(coreconfig.PluginConfigName, "wrong////")
	databaseConfig.AddKnownKey(coreconfig.PluginConfigType, "postgres")
	ctx := context.Background()
	err := or.initPlugins(ctx)
	assert.Error(t, err)
}

func TestDeprecatedDatabaseInitPlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	difactory.InitConfigDeprecated(deprecatedDatabaseConfig)
	deprecatedDatabaseConfig.AddKnownKey(coreconfig.PluginConfigType, "postgres")
	or.mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ctx := context.Background()
	err := or.initDeprecatedDatabasePlugin(ctx, or.mdi)
	assert.NoError(t, err)
}

func TestDatabaseInitPlugins(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	difactory.InitConfig(databaseConfig)
	config.Set("plugins.database", []fftypes.JSONObject{{}})
	databaseConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	databaseConfig.AddKnownKey(coreconfig.PluginConfigType, "postgres")
	plugins := make([]database.Plugin, 1)
	mdp := &databasemocks.Plugin{}
	mdp.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	plugins[0] = mdp
	ctx := context.Background()
	err := or.initDatabasePlugins(ctx, plugins)
	assert.NoError(t, err)
}

func TestDatabaseInitPluginFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	difactory.InitConfig(databaseConfig)
	config.Set("plugins.database", []fftypes.JSONObject{{}})
	databaseConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	databaseConfig.AddKnownKey(coreconfig.PluginConfigType, "sqlite3")
	ctx := context.Background()
	err := or.initPlugins(ctx)
	assert.Regexp(t, "FF10138.*url", err)
}

func TestDeprecatedDatabaseInitPluginFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	difactory.InitConfigDeprecated(deprecatedDatabaseConfig)
	deprecatedDatabaseConfig.AddKnownKey(coreconfig.PluginConfigType, "sqlite3")
	ctx := context.Background()
	err := or.initPlugins(ctx)
	assert.Regexp(t, "FF10138.*url", err)
}

func TestIdentityPluginMissingType(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.database = or.mdi
	or.identityPlugins = nil
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10386.*type", err)
}

func TestIdentityPluginBadName(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.database = or.mdi
	or.identityPlugins = nil
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "wrong//")
	identityConfig.AddKnownKey(coreconfig.PluginConfigType, "tbd")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF00140.*name", err)
}

func TestIdentityPluginUnknownPlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.database = or.mdi
	or.identityPlugins = nil
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	identityConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10212.*wrong", err)
}

func TestIdentityPlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.database = or.mdi
	or.identityPlugins = nil
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	identityConfig.AddKnownKey(coreconfig.PluginConfigType, "onchain")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})
	or.mns.On("Init", mock.Anything, or.mdi).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.NoError(t, err)
}

func TestBadIdentityInitFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.blockchains = nil
	config.Set("plugins.identity", []fftypes.JSONObject{{}})
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	identityConfig.AddKnownKey(coreconfig.PluginConfigType, "onchain")
	plugins := make([]identity.Plugin, 1)
	mii := &identitymocks.Plugin{}
	mii.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	plugins[0] = mii
	ctx := context.Background()
	err := or.initIdentityPlugins(ctx, plugins)
	assert.EqualError(t, err, "pop")
}

func TestBadDeprecatedBlockchainPlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	deprecatedBlockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	or.blockchains = nil
	or.databases["database_0"] = or.mdi
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10110.*wrong", err)
}

func TestDeprecatedBlockchainInitFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	bifactory.InitConfigDeprecated(deprecatedBlockchainConfig)
	deprecatedBlockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "ethereum")
	or.blockchains = nil
	or.databases["database_0"] = or.mdi
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10138.*url", err)
}

func TestBlockchainGetPlugins(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	bifactory.InitConfig(blockchainConfig)
	config.Set("plugins.blockchain", []fftypes.JSONObject{{}})
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "ethereum")
	ctx := context.Background()
	plugins, err := or.getBlockchainPlugins(ctx)
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestBlockchainGetPluginsNoType(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	bifactory.InitConfig(blockchainConfig)
	config.Set("plugins.blockchain", []fftypes.JSONObject{{}})
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	ctx := context.Background()
	_, err := or.getBlockchainPlugins(ctx)
	assert.Error(t, err)
}

func TestBlockchainGetPluginsBadName(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	bifactory.InitConfig(blockchainConfig)
	config.Set("plugins.blockchain", []fftypes.JSONObject{{}})
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigName, "wrong/////////////")
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "ethereum")
	ctx := context.Background()
	_, err := or.getBlockchainPlugins(ctx)
	assert.Error(t, err)
}

func TestBlockchainGetPluginsBadPlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	bifactory.InitConfig(blockchainConfig)
	config.Set("plugins.blockchain", []fftypes.JSONObject{{}})
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong//")
	or.blockchains = nil
	ctx := context.Background()
	err := or.initPlugins(ctx)
	assert.Error(t, err)
}

func TestBlockchainInitPlugins(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	bifactory.InitConfig(blockchainConfig)
	config.Set("plugins.blockchain", []fftypes.JSONObject{{}})
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "ethereum")
	plugins := make([]blockchain.Plugin, 1)
	mbp := &blockchainmocks.Plugin{}
	mbp.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	plugins[0] = mbp
	ctx := context.Background()
	err := or.initBlockchainPlugins(ctx, plugins)
	assert.NoError(t, err)
}

func TestDeprecatedBlockchainInitPlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	bifactory.InitConfigDeprecated(deprecatedBlockchainConfig)
	deprecatedBlockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "ethereum")
	or.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ctx := context.Background()
	err := or.initDeprecatedBlockchainPlugin(ctx, or.mbi)
	assert.NoError(t, err)
}

func TestBlockchainInitPluginsFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	bifactory.InitConfig(blockchainConfig)
	config.Set("plugins.blockchain", []fftypes.JSONObject{{}})
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "ethereum")
	blockchainConfig.AddKnownKey("addressResolver.urlTemplate", "")
	blockchainConfig.AddKnownKey("ethconnect.url", "")
	or.blockchains = nil

	ctx := context.Background()
	err := or.initPlugins(ctx)
	assert.Regexp(t, "FF10138.*url", err)
}

func TestBadSharedStoragePlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	ssfactory.InitConfig(sharedstorageConfig)
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	config.Set("plugins.sharedstorage", []fftypes.JSONObject{{}})
	or.sharedstoragePlugins = nil
	ctx := context.Background()
	plugins, err := or.getSharedStoragePlugins(ctx)
	assert.Nil(t, plugins)
	assert.Regexp(t, "FF10386.*Invalid", err)
}

func TestBadSharedStoragePluginType(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.sharedstoragePlugins = nil
	or.database = or.mdi
	ssfactory.InitConfig(sharedstorageConfig)
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigName, "sharedstorage")
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	config.Set("plugins.sharedstorage", []fftypes.JSONObject{{}})

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10134.*wrong", err)
}

func TestBadSharedStoragePluginName(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	ssfactory.InitConfig(sharedstorageConfig)
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigName, "wrong////")
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigType, "ipfs")
	config.Set("plugins.sharedstorage", []fftypes.JSONObject{{}})
	or.sharedstoragePlugins = nil
	ctx := context.Background()
	plugins, err := or.getSharedStoragePlugins(ctx)
	assert.Nil(t, plugins)
	assert.Regexp(t, "FF00140.*name", err)
}

func TestSharedStorageInitPlugins(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	ssfactory.InitConfig(sharedstorageConfig)
	config.Set("plugins.sharedstorage", []fftypes.JSONObject{{}})
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigType, "ipfs")
	plugins := make([]sharedstorage.Plugin, 1)
	mss := &sharedstoragemocks.Plugin{}
	mss.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	plugins[0] = mss
	ctx := context.Background()
	err := or.initSharedStoragePlugins(ctx, plugins)
	assert.NoError(t, err)
}

func TestSharedStorageInitPluginsFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.sharedstoragePlugins = nil
	or.database = or.mdi
	ssfactory.InitConfig(sharedstorageConfig)
	config.Set("plugins.sharedstorage", []fftypes.JSONObject{{}})
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigType, "ipfs")
	ctx := context.Background()
	err := or.initPlugins(ctx)
	assert.Regexp(t, "FF10138.*url", err)
}

func TestDeprecatedSharedStorageInitPlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	ssfactory.InitConfigDeprecated(deprecatedSharedStorageConfig)
	deprecatedSharedStorageConfig.AddKnownKey(coreconfig.PluginConfigType, "ipfs")
	or.mps.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ctx := context.Background()
	err := or.initDeprecatedSharedStoragePlugin(ctx, or.mps)
	assert.NoError(t, err)
}

func TestDeprecatedSharedStorageInitPluginFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.sharedstoragePlugins = nil
	ssfactory.InitConfigDeprecated(deprecatedSharedStorageConfig)
	deprecatedSharedStorageConfig.AddKnownKey(coreconfig.PluginConfigType, "ipfs")
	ctx := context.Background()
	err := or.initPlugins(ctx)
	assert.Regexp(t, "FF10138.*url", err)
}

func TestBadDeprecatedSharedStoragePlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	deprecatedSharedStorageConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	or.sharedstoragePlugins = nil
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10134.*Unknown", err)
}

func TestBadDataExchangePlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	dxfactory.InitConfig(dataexchangeConfig)
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong//")
	config.Set("plugins.dataexchange", []fftypes.JSONObject{{}})
	or.database = or.mdi
	or.dataexchangePlugins = nil
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10213.*wrong", err)
}

func TestDataExchangePluginBadName(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	dxfactory.InitConfig(dataexchangeConfig)
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigName, "wrong//")
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "ffdx")
	config.Set("plugins.dataexchange", []fftypes.JSONObject{{}})
	or.database = or.mdi
	or.dataexchangePlugins = nil
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF00140.*name", err)
}

func TestDataExchangePluginMissingName(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	dxfactory.InitConfig(dataexchangeConfig)
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "ffdx")
	config.Set("plugins.dataexchange", []fftypes.JSONObject{{}})
	or.database = or.mdi
	or.dataexchangePlugins = nil
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10386.*name", err)
}

func TestBadDataExchangeInitFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	dxfactory.InitConfig(dataexchangeConfig)
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "ffdx")
	config.Set("plugins.dataexchange", []fftypes.JSONObject{{}})
	or.database = or.mdi
	or.dataexchangePlugins = nil
	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return([]*core.Identity{}, nil, nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10138.*url", err)
}

func TestDeprecatedBadDataExchangeInitFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	dxfactory.InitConfigDeprecated(deprecatedDataexchangeConfig)
	deprecatedDataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "ffdx")
	or.database = or.mdi
	or.dataexchangePlugins = nil
	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return([]*core.Identity{}, nil, nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10138.*url", err)
}

func TestDeprecatedDataExchangeInit(t *testing.T) {
	or := newTestOrchestrator()
	dxfactory.InitConfigDeprecated(deprecatedDataexchangeConfig)
	deprecatedDataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "ffdx")
	deprecatedDataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "ffdx")
	deprecatedDataexchangeConfig.AddKnownKey("ffdx.url", "https://test")
	or.database = or.mdi
	or.dataexchangePlugins = nil
	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return([]*core.Identity{}, nil, nil)
	ctx := context.Background()
	err := or.initDataExchange(ctx)
	assert.NoError(t, err)
}

func TestDeprecatedBadDataExchangePlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	dxfactory.InitConfigDeprecated(deprecatedDataexchangeConfig)
	deprecatedDataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong//")
	or.database = or.mdi
	or.dataexchangePlugins = nil
	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return([]*core.Identity{}, nil, nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10213.*wrong", err)
}

func TestTokensMissingName(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	tifactory.InitConfig(tokensConfig)
	tokensConfig.AddKnownKey(coreconfig.PluginConfigType, "fftokens")
	config.Set("plugins.tokens", []fftypes.JSONObject{{}})
	or.database = or.mdi
	or.tokens = nil
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10386.*type", err)
}

func TestTokensBadName(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	tifactory.InitConfig(tokensConfig)
	tokensConfig.AddKnownKey(coreconfig.PluginConfigName, "/////////////")
	tokensConfig.AddKnownKey(coreconfig.PluginConfigType, "fftokens")
	config.Set("plugins.tokens", []fftypes.JSONObject{{}})
	or.database = or.mdi
	or.tokens = nil
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF00140.*name", err)
}

func TestBadTokensPlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	tifactory.InitConfig(tokensConfig)
	tokensConfig.AddKnownKey(coreconfig.PluginConfigName, "erc20_erc721")
	tokensConfig.AddKnownKey(coreconfig.PluginConfigType, "fftokens")
	config.Set("plugins.tokens", []fftypes.JSONObject{{}})
	or.database = or.mdi
	or.tokens = nil
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Error(t, err)
}

func TestGoodTokensPlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	tifactory.InitConfig(tokensConfig)
	tokensConfig.AddKnownKey(coreconfig.PluginConfigName, "erc20_erc721")
	tokensConfig.AddKnownKey(coreconfig.PluginConfigType, "fftokens")
	tokensConfig.AddKnownKey("fftokens.url", "test")
	config.Set("plugins.tokens", []fftypes.JSONObject{{}})
	or.database = or.mdi
	or.tokens = nil
	or.mns.On("Init", mock.Anything, or.mdi).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.NoError(t, err)
}

func TestBadDeprecatedTokensPluginNoName(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	deprecatedTokensConfig.AddKnownKey(coreconfig.PluginConfigName)
	deprecatedTokensConfig.AddKnownKey(tokens.TokensConfigPlugin, "wrong")
	config.Set("tokens", []fftypes.JSONObject{{}})
	or.database = or.mdi
	or.tokens = nil
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10273", err)
}

func TestBadDeprecatedTokensPluginInvalidName(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	deprecatedTokensConfig.AddKnownKey(coreconfig.PluginConfigName, "!wrong")
	deprecatedTokensConfig.AddKnownKey(tokens.TokensConfigPlugin, "text")
	config.Set("tokens", []fftypes.JSONObject{{}})
	or.database = or.mdi
	or.tokens = nil
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF00140.*'name'", err)
}

func TestBadDeprecatedTokensPluginNoType(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	deprecatedTokensConfig.AddKnownKey(coreconfig.PluginConfigName, "text")
	deprecatedTokensConfig.AddKnownKey(tokens.TokensConfigPlugin)
	config.Set("tokens", []fftypes.JSONObject{{}})
	or.database = or.mdi
	or.tokens = nil
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10272", err)
}

func TestGoodDeprecatedTokensPlugin(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	deprecatedTokensConfig = config.RootArray("tokens")
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	deprecatedTokensConfig.AddKnownKey(coreconfig.PluginConfigName, "test")
	deprecatedTokensConfig.AddKnownKey(tokens.TokensConfigPlugin, "fftokens")
	deprecatedTokensConfig.AddKnownKey(ffresty.HTTPConfigURL, "test")
	config.Set("tokens", []fftypes.JSONObject{{}})
	or.database = or.mdi
	or.tokens = nil
	or.mns.On("Init", mock.Anything, or.mdi).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := or.Init(ctx, cancelCtx)
	assert.NoError(t, err)
}

func TestInitMessagingComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	or.messaging = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitEventsComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	or.events = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitNetworkMapComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	or.networkmap = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitOperationComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	or.operations = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitSharedStorageDownloadComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	or.sharedDownload = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitAdminEventsInit(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.adminEvents = nil
	err := or.initPlugins(context.Background())
	assert.NoError(t, err)
}

func TestInitBatchComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	or.batch = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitBroadcastComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	or.broadcast = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitDataComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	or.data = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitIdentityComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	or.identity = nil
	or.txHelper = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitAssetsComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	or.assets = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitContractsComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	or.contracts = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitDefinitionsComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	or.definitions = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitBatchPinComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	or.batchpin = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitOperationsComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.databases = nil
	or.operations = nil
	err := or.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err)
}

func TestInitNamespaceComponentFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.namespace = nil
	err := or.initNamespaces(context.Background())
	assert.Regexp(t, "FF10166", err)
}

func TestStartBatchFail(t *testing.T) {
	coreconfig.Reset()
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.mba.On("Start").Return(fmt.Errorf("pop"))
	err := or.Start()
	assert.EqualError(t, err, "pop")
}

func TestStartTokensFail(t *testing.T) {
	coreconfig.Reset()
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.database = or.mdi
	or.mdi.On("GetNamespace", mock.Anything, "ff_system").Return(&core.Namespace{}, nil)
	or.mbi.On("ConfigureContract", &core.FireFlyContracts{}).Return(nil)
	or.mbi.On("Start").Return(nil)
	or.mba.On("Start").Return(nil)
	or.mem.On("Start").Return(nil)
	or.mbm.On("Start").Return(nil)
	or.mpm.On("Start").Return(nil)
	or.msd.On("Start").Return(nil)
	or.mom.On("Start").Return(nil)
	or.mti.On("Start").Return(fmt.Errorf("pop"))
	or.mdi.On("UpsertNamespace", mock.Anything, mock.Anything, true).Return(nil)
	err := or.Start()
	assert.EqualError(t, err, "pop")
}

func TestStartBlockchainsFail(t *testing.T) {
	coreconfig.Reset()
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.database = or.mdi
	or.mdi.On("GetNamespace", mock.Anything, "ff_system").Return(&core.Namespace{}, nil)
	or.mbi.On("ConfigureContract", &core.FireFlyContracts{}).Return(nil)
	or.mbi.On("Start").Return(fmt.Errorf("pop"))
	or.mba.On("Start").Return(nil)
	err := or.Start()
	assert.EqualError(t, err, "pop")
}

func TestStartStopOk(t *testing.T) {
	coreconfig.Reset()
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.database = or.mdi
	or.mdi.On("GetNamespace", mock.Anything, "ff_system").Return(&core.Namespace{}, nil)
	or.mbi.On("ConfigureContract", &core.FireFlyContracts{}).Return(nil)
	or.mbi.On("Start").Return(nil)
	or.mba.On("Start").Return(nil)
	or.mem.On("Start").Return(nil)
	or.mbm.On("Start").Return(nil)
	or.mpm.On("Start").Return(nil)
	or.mti.On("Start").Return(nil)
	or.mmi.On("Start").Return(nil)
	or.msd.On("Start").Return(nil)
	or.mom.On("Start").Return(nil)
	or.mba.On("WaitStop").Return(nil)
	or.mbm.On("WaitStop").Return(nil)
	or.mdm.On("WaitStop").Return(nil)
	or.msd.On("WaitStop").Return(nil)
	or.mom.On("WaitStop").Return(nil)
	or.mae.On("WaitStop").Return(nil)
	or.mdi.On("UpsertNamespace", mock.Anything, mock.Anything, true).Return(nil)
	err := or.Start()
	assert.NoError(t, err)
	or.WaitStop()
	or.WaitStop() // swallows dups
}

func TestInitOK(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.database = or.mdi
	err := config.ReadConfig("core", configDir+"/firefly.core.yaml")
	assert.NoError(t, err)
	or.mns.On("Init", mock.Anything, or.mdi).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err = or.Init(ctx, cancelCtx)
	assert.NoError(t, err)

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
	assert.Equal(t, or.mae, or.AdminEvents())
}

func TestInitOKWithMetrics(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.metrics = nil
	or.database = or.mdi
	err := config.ReadConfig("core", configDir+"/firefly.core.yaml")
	assert.NoError(t, err)
	or.mns.On("Init", mock.Anything, or.mdi).Return(nil)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err = or.Init(ctx, cancelCtx)
	assert.NoError(t, err)

	assert.Equal(t, or.mbm, or.Broadcast())
	assert.Equal(t, or.mpm, or.PrivateMessaging())
	assert.Equal(t, or.mem, or.Events())
	assert.Equal(t, or.mba, or.BatchManager())
	assert.Equal(t, or.mnm, or.NetworkMap())
	assert.Equal(t, or.mdm, or.Data())
	assert.Equal(t, or.mam, or.Assets())
	assert.Equal(t, or.mcm, or.Contracts())
	assert.Equal(t, or.mom, or.Operations())
	assert.Equal(t, or.mae, or.AdminEvents())
}

func TestInitNamespaceFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.database = or.mdi
	err := config.ReadConfig("core", configDir+"/firefly.core.yaml")
	assert.NoError(t, err)
	or.mns.On("Init", mock.Anything, or.mdi).Return(fmt.Errorf("pop"))
	ctx, cancelCtx := context.WithCancel(context.Background())
	err = or.Init(ctx, cancelCtx)
	assert.EqualError(t, err, "pop")
}

func TestInitDataExchangeGetNodesFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.database = or.mdi

	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	err := or.initDataExchange(or.ctx)
	assert.EqualError(t, err, "pop")
}

func TestInitDataExchangeWithNodes(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)
	or.database = or.mdi
	dxfactory.InitConfig(dataexchangeConfig)
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "ffdx")
	dataexchangeConfig.AddKnownKey("ffdx.url", "https://test")
	config.Set("plugins.dataexchange", []fftypes.JSONObject{{}})

	or.mdi.On("GetIdentities", mock.Anything, mock.Anything).Return([]*core.Identity{{}}, nil, nil)

	err := or.initDataExchange(or.ctx)
	assert.NoError(t, err)
}

func TestMigrateNetwork(t *testing.T) {
	or := newTestOrchestrator()
	or.blockchain = or.mbi
	verifier := &core.VerifierRef{Value: "0x123"}
	or.mim.On("GetNodeOwnerBlockchainKey", context.Background()).Return(verifier, nil)
	or.mbi.On("SubmitOperatorAction", context.Background(), mock.Anything, "0x123", "terminate").Return(nil)
	err := or.MigrateNetwork(context.Background())
	assert.NoError(t, err)
}

func TestMigrateNetworkBadKey(t *testing.T) {
	or := newTestOrchestrator()
	or.mim.On("GetNodeOwnerBlockchainKey", context.Background()).Return(nil, fmt.Errorf("pop"))
	err := or.MigrateNetwork(context.Background())
	assert.EqualError(t, err, "pop")
}
