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

package namespace

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/blockchain/bifactory"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/database/difactory"
	"github.com/hyperledger/firefly/internal/dataexchange/dxfactory"
	"github.com/hyperledger/firefly/internal/identity/iifactory"
	"github.com/hyperledger/firefly/internal/orchestrator"
	"github.com/hyperledger/firefly/internal/sharedstorage/ssfactory"
	"github.com/hyperledger/firefly/internal/tokens/tifactory"
	"github.com/hyperledger/firefly/mocks/admineventsmocks"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/identitymocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/orchestratormocks"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testNamespaceManager struct {
	namespaceManager
	mmi *metricsmocks.Manager
	mae *admineventsmocks.Manager
}

func (nm *testNamespaceManager) cleanup(t *testing.T) {
}

func newTestNamespaceManager(resetConfig bool) *testNamespaceManager {
	if resetConfig {
		coreconfig.Reset()
		InitConfig(true)
		namespaceConfig.AddKnownKey("predefined.0.multiparty.enabled", true)
	}
	nm := &testNamespaceManager{
		mmi: &metricsmocks.Manager{},
		mae: &admineventsmocks.Manager{},
		namespaceManager: namespaceManager{
			namespaces:  make(map[string]*namespace),
			pluginNames: make(map[string]bool),
			blockchains: map[string]orchestrator.BlockchainPlugin{
				"ethereum": {Plugin: &blockchainmocks.Plugin{}},
			},
			databases: map[string]orchestrator.DatabasePlugin{
				"postgres": {Plugin: &databasemocks.Plugin{}},
			},
			dataexchanges: map[string]orchestrator.DataexchangePlugin{
				"ffdx": {Plugin: &dataexchangemocks.Plugin{}},
			},
			sharedstorages: map[string]orchestrator.SharedStoragePlugin{
				"ipfs": {Plugin: &sharedstoragemocks.Plugin{}},
			},
			identities: map[string]orchestrator.IdentityPlugin{
				"tbd": {Plugin: &identitymocks.Plugin{}},
			},
			tokens: map[string]orchestrator.TokensPlugin{
				"erc721": {Plugin: &tokenmocks.Plugin{}},
			},
		},
	}
	nm.namespaceManager.metrics = nm.mmi
	nm.namespaceManager.adminEvents = nm.mae
	return nm
}

func TestNewNamespaceManager(t *testing.T) {
	nm := NewNamespaceManager(true)
	assert.NotNil(t, nm)
}

func TestInit(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	mo := &orchestratormocks.Orchestrator{}
	mo.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.utOrchestrator = mo

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx)
	assert.NoError(t, err)

	assert.Equal(t, mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))
}

func TestInitFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	mdi := nm.databases["postgres"].Plugin.(*databasemocks.Plugin)
	mdi.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestDeprecatedDatabasePlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	difactory.InitConfigDeprecated(deprecatedDatabaseConfig)
	deprecatedDatabaseConfig.Set(coreconfig.PluginConfigType, "postgres")
	plugins, err := nm.getDatabasePlugins(context.Background())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDeprecatedDatabasePluginBadType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	difactory.InitConfigDeprecated(deprecatedDatabaseConfig)
	deprecatedDatabaseConfig.Set(coreconfig.PluginConfigType, "wrong")
	_, err := nm.getDatabasePlugins(context.Background())
	assert.Regexp(t, "FF10122.*wrong", err)
}

func TestDatabasePlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	difactory.InitConfig(databaseConfig)
	config.Set("plugins.database", []fftypes.JSONObject{{}})
	databaseConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	databaseConfig.AddKnownKey(coreconfig.PluginConfigType, "postgres")
	plugins, err := nm.getDatabasePlugins(context.Background())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDatabasePluginBadType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	difactory.InitConfig(databaseConfig)
	config.Set("plugins.database", []fftypes.JSONObject{{}})
	databaseConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	databaseConfig.AddKnownKey(coreconfig.PluginConfigType, "unknown")
	plugins, err := nm.getDatabasePlugins(context.Background())
	assert.Nil(t, plugins)
	assert.Error(t, err)
}

func TestDatabasePluginBadName(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	nm.databases = nil
	difactory.InitConfig(databaseConfig)
	config.Set("plugins.database", []fftypes.JSONObject{{}})
	databaseConfig.AddKnownKey(coreconfig.PluginConfigName, "wrong////")
	databaseConfig.AddKnownKey(coreconfig.PluginConfigType, "postgres")
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx)
	assert.Error(t, err)
}

func TestIdentityPluginBadName(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "wrong//")
	identityConfig.AddKnownKey(coreconfig.PluginConfigType, "tbd")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})
	_, err := nm.getIdentityPlugins(context.Background())
	assert.Regexp(t, "FF00140.*name", err)
}

func TestIdentityPluginBadType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	identityConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})
	_, err := nm.getIdentityPlugins(context.Background())
	assert.Regexp(t, "FF10212.*wrong", err)
}

func TestIdentityPluginNoType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	nm.identities = nil
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10386.*type", err)
}

func TestIdentityPlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	identityConfig.AddKnownKey(coreconfig.PluginConfigType, "onchain")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})
	plugins, err := nm.getIdentityPlugins(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, len(plugins))
}

func TestDeprecatedBlockchainPlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	bifactory.InitConfigDeprecated(deprecatedBlockchainConfig)
	deprecatedBlockchainConfig.Set(coreconfig.PluginConfigType, "ethereum")
	plugins, err := nm.getBlockchainPlugins(context.Background())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDeprecatedBlockchainPluginBadType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	deprecatedBlockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	_, err := nm.getBlockchainPlugins(context.Background())
	assert.Regexp(t, "FF10110.*wrong", err)
}

func TestBlockchainPlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	bifactory.InitConfig(blockchainConfig)
	config.Set("plugins.blockchain", []fftypes.JSONObject{{}})
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "ethereum")
	plugins, err := nm.getBlockchainPlugins(context.Background())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestBlockchainPluginNoType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	bifactory.InitConfig(blockchainConfig)
	config.Set("plugins.blockchain", []fftypes.JSONObject{{}})
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	_, err := nm.getBlockchainPlugins(context.Background())
	assert.Error(t, err)
}

func TestBlockchainPluginBadType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	nm.blockchains = nil
	bifactory.InitConfig(blockchainConfig)
	config.Set("plugins.blockchain", []fftypes.JSONObject{{}})
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong//")
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx)
	assert.Error(t, err)
}

func TestDeprecatedSharedStoragePlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	ssfactory.InitConfigDeprecated(deprecatedSharedStorageConfig)
	deprecatedSharedStorageConfig.Set(coreconfig.PluginConfigType, "ipfs")
	plugins, err := nm.getSharedStoragePlugins(context.Background())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDeprecatedSharedStoragePluginBadType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	ssfactory.InitConfigDeprecated(deprecatedSharedStorageConfig)
	deprecatedSharedStorageConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	_, err := nm.getSharedStoragePlugins(context.Background())
	assert.Regexp(t, "FF10134.*wrong", err)
}

func TestSharedStoragePlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	ssfactory.InitConfig(sharedstorageConfig)
	config.Set("plugins.sharedstorage", []fftypes.JSONObject{{}})
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigType, "ipfs")
	plugins, err := nm.getSharedStoragePlugins(context.Background())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestSharedStoragePluginNoType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	ssfactory.InitConfig(sharedstorageConfig)
	config.Set("plugins.sharedstorage", []fftypes.JSONObject{{}})
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	_, err := nm.getSharedStoragePlugins(context.Background())
	assert.Error(t, err)
}

func TestSharedStoragePluginBadType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	nm.sharedstorages = nil
	ssfactory.InitConfig(sharedstorageConfig)
	config.Set("plugins.sharedstorage", []fftypes.JSONObject{{}})
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong//")
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx)
	assert.Error(t, err)
}

func TestDeprecatedDataExchangePlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	dxfactory.InitConfigDeprecated(deprecatedDataexchangeConfig)
	deprecatedDataexchangeConfig.Set(coreconfig.PluginConfigType, "ffdx")
	plugins, err := nm.getDataExchangePlugins(context.Background())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDeprecatedDataExchangePluginBadType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	dxfactory.InitConfigDeprecated(deprecatedDataexchangeConfig)
	deprecatedDataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	_, err := nm.getDataExchangePlugins(context.Background())
	assert.Regexp(t, "FF10213.*wrong", err)
}

func TestDataExchangePlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	dxfactory.InitConfig(dataexchangeConfig)
	config.Set("plugins.dataexchange", []fftypes.JSONObject{{}})
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "ffdx")
	plugins, err := nm.getDataExchangePlugins(context.Background())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDataExchangePluginNoType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	dxfactory.InitConfig(dataexchangeConfig)
	config.Set("plugins.dataexchange", []fftypes.JSONObject{{}})
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	_, err := nm.getDataExchangePlugins(context.Background())
	assert.Error(t, err)
}

func TestDataExchangePluginBadType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	nm.dataexchanges = nil
	dxfactory.InitConfig(dataexchangeConfig)
	config.Set("plugins.dataexchange", []fftypes.JSONObject{{}})
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong//")
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx)
	assert.Error(t, err)
}

func TestDeprecatedTokensPlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	config.Set("tokens", []fftypes.JSONObject{{}})
	deprecatedTokensConfig.AddKnownKey(coreconfig.PluginConfigName, "test")
	deprecatedTokensConfig.AddKnownKey(tokens.TokensConfigPlugin, "fftokens")
	plugins, err := nm.getTokensPlugins(context.Background())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDeprecatedTokensPluginNoName(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	config.Set("tokens", []fftypes.JSONObject{{}})
	deprecatedTokensConfig.AddKnownKey(tokens.TokensConfigPlugin, "fftokens")
	_, err := nm.getTokensPlugins(context.Background())
	assert.Regexp(t, "FF10273", err)
}

func TestDeprecatedTokensPluginBadName(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	config.Set("tokens", []fftypes.JSONObject{{}})
	deprecatedTokensConfig.AddKnownKey(coreconfig.PluginConfigName, "BAD!")
	deprecatedTokensConfig.AddKnownKey(tokens.TokensConfigPlugin, "fftokens")
	_, err := nm.getTokensPlugins(context.Background())
	assert.Regexp(t, "FF00140", err)
}

func TestDeprecatedTokensPluginBadType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	config.Set("tokens", []fftypes.JSONObject{{}})
	deprecatedTokensConfig.AddKnownKey(coreconfig.PluginConfigName, "test")
	deprecatedTokensConfig.AddKnownKey(tokens.TokensConfigPlugin, "wrong")
	_, err := nm.getTokensPlugins(context.Background())
	assert.Regexp(t, "FF10272.*wrong", err)
}

func TestTokensPlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	tifactory.InitConfig(tokensConfig)
	config.Set("plugins.tokens", []fftypes.JSONObject{{}})
	tokensConfig.AddKnownKey(coreconfig.PluginConfigName, "erc20_erc721")
	tokensConfig.AddKnownKey(coreconfig.PluginConfigType, "fftokens")
	plugins, err := nm.getTokensPlugins(context.Background())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestTokensPluginNoType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	tifactory.InitConfig(tokensConfig)
	config.Set("plugins.tokens", []fftypes.JSONObject{{}})
	tokensConfig.AddKnownKey(coreconfig.PluginConfigName, "erc20_erc721")
	_, err := nm.getTokensPlugins(context.Background())
	assert.Error(t, err)
}

func TestTokensPluginBadType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	nm.tokens = nil
	tifactory.InitConfig(tokensConfig)
	config.Set("plugins.tokens", []fftypes.JSONObject{{}})
	tokensConfig.AddKnownKey(coreconfig.PluginConfigName, "erc20_erc721")
	tokensConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx)
	assert.Error(t, err)
}

func TestTokensPluginDuplicate(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	nm.pluginNames["erc20_erc721"] = true
	tifactory.InitConfig(tokensConfig)
	config.Set("plugins.tokens", []fftypes.JSONObject{{}})
	tokensConfig.AddKnownKey(coreconfig.PluginConfigName, "erc20_erc721")
	tokensConfig.AddKnownKey(coreconfig.PluginConfigType, "fftokens")
	_, err := nm.getTokensPlugins(context.Background())
	assert.Regexp(t, "FF10395", err)
}

func TestInitNamespacesBadName(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: "!Badness"
    predefined:
    - name: "!Badness"
    `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF00140", err)
}

func TestInitNamespacesReservedName(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ff_system
    `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10388", err)
}

func TestInitNamespacesDuplicate(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [postgres]
    - name: ns1
      plugins: [postgres]
    `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.NoError(t, err)
	assert.Len(t, nm.namespaces, 1)
}

func TestInitNamespacesNoDefault(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns2
      plugins: [postgres]
  `))
	assert.NoError(t, err)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err = nm.Init(ctx, cancelCtx)
	assert.Regexp(t, "FF10166", err)
}

func TestInitNamespacesUseDefaults(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
  org:
    name: org1
  `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.NoError(t, err)
	assert.Len(t, nm.namespaces, 1)
}

func TestInitNamespacesGatewayNoDatabase(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: []
  `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10392", err)
}

func TestInitNamespacesMultipartyUnknownPlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [bad]
      multiparty:
        enabled: true
  `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10390.*unknown", err)
}

func TestInitNamespacesMultipartyMultipleBlockchains(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [ethereum, ethereum]
      multiparty:
        enabled: true
  `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10394.*blockchain", err)
}

func TestInitNamespacesMultipartyMultipleDX(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [ffdx, ffdx]
      multiparty:
        enabled: true
  `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10394.*dataexchange", err)
}

func TestInitNamespacesMultipartyMultipleSS(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [ipfs, ipfs]
      multiparty:
        enabled: true
  `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10394.*sharedstorage", err)
}

func TestInitNamespacesMultipartyMultipleDB(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [postgres, postgres]
      multiparty:
        enabled: true
  `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10394.*database", err)
}

func TestInitNamespacesGatewayMultipleDB(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [postgres, postgres]
  `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10394.*database", err)
}

func TestInitNamespacesGatewayMultipleBlockchains(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [ethereum, ethereum]
  `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10394.*blockchain", err)
}

func TestInitNamespacesMultipartyMissingPlugins(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [postgres]
      multiparty:
        enabled: true
  `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10391", err)
}

func TestInitNamespacesGatewayWithDX(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [ffdx]
  `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10393", err)
}

func TestInitNamespacesGatewayWithSharedStorage(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [ipfs]
  `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10393", err)
}

func TestInitNamespacesGatewayUnknownPlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [bad]
  `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10390.*unknown", err)
}

func TestInitNamespacesGatewayTokens(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [postgres, erc721]
  `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.NoError(t, err)
}

func TestStart(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	mo := &orchestratormocks.Orchestrator{}
	nm.namespaces = map[string]*namespace{
		"ns": {orchestrator: mo},
	}
	nm.metricsEnabled = true

	mo.On("Start").Return(nil)

	err := nm.Start()
	assert.NoError(t, err)

	mo.AssertExpectations(t)
}

func TestStartFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	mo := &orchestratormocks.Orchestrator{}
	nm.namespaces = map[string]*namespace{
		"ns": {orchestrator: mo},
	}

	mo.On("Start").Return(fmt.Errorf("pop"))

	err := nm.Start()
	assert.EqualError(t, err, "pop")

	mo.AssertExpectations(t)
}

func TestWaitStop(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	mo := &orchestratormocks.Orchestrator{}
	nm.namespaces = map[string]*namespace{
		"ns": {orchestrator: mo},
	}
	mae := nm.adminEvents.(*admineventsmocks.Manager)

	mo.On("WaitStop").Return()
	mae.On("WaitStop").Return()

	nm.WaitStop()

	mo.AssertExpectations(t)
	mae.AssertExpectations(t)
}

func TestLoadMetrics(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	nm.metrics = nil

	err := nm.loadPlugins(context.Background())
	assert.NoError(t, err)
}

func TestLoadAdminEvents(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	nm.adminEvents = nil

	err := nm.loadPlugins(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, nm.AdminEvents())
}

func TestGetNamespaces(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	nm.namespaces = map[string]*namespace{
		"default": {},
	}

	results, err := nm.GetNamespaces(context.Background())
	assert.Nil(t, err)
	assert.Len(t, results, 1)
}
