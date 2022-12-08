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

	"github.com/hyperledger/firefly-common/mocks/authmocks"
	"github.com/hyperledger/firefly-common/pkg/auth/authfactory"
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
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/cachemocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/eventsmocks"
	"github.com/hyperledger/firefly/mocks/identitymocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/mocks/orchestratormocks"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/mocks/spieventsmocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testNamespaceManager struct {
	namespaceManager
	mmi *metricsmocks.Manager
	mae *spieventsmocks.Manager
	mbi *blockchainmocks.Plugin
	cmi *cachemocks.Manager
	mdi *databasemocks.Plugin
	mdx *dataexchangemocks.Plugin
	mps *sharedstoragemocks.Plugin
	mti *tokenmocks.Plugin
	mev *eventsmocks.Plugin
	mai *authmocks.Plugin
	mo  *orchestratormocks.Orchestrator
}

func (nm *testNamespaceManager) cleanup(t *testing.T) {
	nm.mmi.AssertExpectations(t)
	nm.mae.AssertExpectations(t)
	nm.mbi.AssertExpectations(t)
	nm.cmi.AssertExpectations(t)
	nm.mdi.AssertExpectations(t)
	nm.mdx.AssertExpectations(t)
	nm.mps.AssertExpectations(t)
	nm.mti.AssertExpectations(t)
	nm.mai.AssertExpectations(t)
	nm.mo.AssertExpectations(t)
}

func newTestNamespaceManager(t *testing.T) (*testNamespaceManager, func()) {
	coreconfig.Reset()
	initAllConfig()
	ctx, cancelCtx := context.WithCancel(context.Background())
	nm := &testNamespaceManager{
		mmi: &metricsmocks.Manager{},
		mae: &spieventsmocks.Manager{},
		mbi: &blockchainmocks.Plugin{},
		cmi: &cachemocks.Manager{},
		mdi: &databasemocks.Plugin{},
		mdx: &dataexchangemocks.Plugin{},
		mps: &sharedstoragemocks.Plugin{},
		mti: &tokenmocks.Plugin{},
		mev: &eventsmocks.Plugin{},
		mai: &authmocks.Plugin{},
		mo:  &orchestratormocks.Orchestrator{},
	}
	nm.namespaceManager = namespaceManager{
		ctx:                 ctx,
		cancelCtx:           cancelCtx,
		reset:               make(chan bool, 1),
		namespaces:          make(map[string]*namespace),
		plugins:             make(map[string]*plugin),
		tokenBroadcastNames: make(map[string]string),
		utOrchestrator:      nm.mo,
	}
	nm.watchConfig = func() {
		<-ctx.Done()
	}
	nm.plugins = map[string]*plugin{
		"ethereum": {
			name:       "ethereum",
			category:   pluginCategoryBlockchain,
			pluginType: "ethereum",
			blockchain: nm.mbi,
		},
		"postgres": {
			name:       "postgres",
			category:   pluginCategoryDatabase,
			pluginType: "postgres",
			database:   nm.mdi,
		},
		"ffdx": {
			name:         "ffdx",
			category:     pluginCategoryDataexchange,
			pluginType:   "ffdx",
			dataexchange: nm.mdx,
		},
		"ipfs": {
			name:          "ipfs",
			category:      pluginCategorySharedstorage,
			pluginType:    "ipfs",
			sharedstorage: nm.mps,
		},
		"tbd": {
			name:       "tbd",
			category:   pluginCategoryIdentity,
			pluginType: "tbd",
			identity:   &identitymocks.Plugin{},
		},
		"erc721": {
			name:       "erc721",
			category:   pluginCategoryTokens,
			pluginType: "erc721",
			tokens:     nm.mti,
		},
		"erc1155": {
			name:       "erc1155",
			category:   pluginCategoryTokens,
			pluginType: "erc1155",
			tokens:     nm.mti,
		},
		"websockets": {
			name:       "websockets",
			category:   pluginCategoryEvents,
			pluginType: "websockets",
			events:     nm.mev,
		},
		"basicauth": {
			name:       "basicauth",
			category:   pluginCategoryAuth,
			pluginType: "basicauth",
			auth:       nm.mai,
		},
	}
	nm.namespaces = map[string]*namespace{
		"default": {
			Namespace: core.Namespace{
				Name: "default",
			},
			ctx:          nm.ctx,
			cancelCtx:    nm.cancelCtx,
			orchestrator: nm.mo,
			pluginNames: []string{
				"ethereum",
				"postgres",
				"ffdx",
				"ipfs",
				"tbd",
				"erc721",
				"erc1155",
				"basicauth",
			},
			plugins: &orchestrator.Plugins{
				Blockchain: orchestrator.BlockchainPlugin{
					Name:   "ethereum",
					Plugin: nm.plugins["ethereum"].blockchain,
				},
				Database: orchestrator.DatabasePlugin{
					Name:   "postgres",
					Plugin: nm.plugins["postgres"].database,
				},
				DataExchange: orchestrator.DataExchangePlugin{
					Name:   "ffdx",
					Plugin: nm.plugins["ffdx"].dataexchange,
				},
				SharedStorage: orchestrator.SharedStoragePlugin{
					Name:   "ipfs",
					Plugin: nm.plugins["ipfs"].sharedstorage,
				},
				Tokens: []orchestrator.TokensPlugin{
					{
						Name:   "erc721",
						Plugin: nm.plugins["erc721"].tokens,
					},
					{
						Name:   "erc1155",
						Plugin: nm.plugins["erc1155"].tokens,
					},
				},
			},
		},
	}
	nm.namespaceManager.metrics = nm.mmi
	nm.namespaceManager.adminEvents = nm.mae

	return nm, func() {
		if a := recover(); a != nil {
			panic(a)
		}
		nm.cancelCtx()
		nm.cleanup(t)
	}
}

func TestNewNamespaceManager(t *testing.T) {
	nm := NewNamespaceManager()
	assert.NotNil(t, nm)
}

func TestInitEmpty(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	nm.metricsEnabled = true

	err := nm.Init(nm.ctx, nm.cancelCtx, nm.reset)
	assert.NoError(t, err)

	assert.Len(t, nm.plugins, 3)
	assert.NotNil(t, nm.plugins["system"])
	assert.NotNil(t, nm.plugins["webhooks"])
	assert.NotNil(t, nm.plugins["websockets"])
	assert.Empty(t, nm.namespaces)
}

func TestInitAllPlugins(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	nm.metricsEnabled = true

	nm.mo.On("Init", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			nm.namespaces["default"].Contracts.Active = &core.MultipartyContract{
				Info: core.MultipartyContractInfo{
					Version: 2,
				},
			}
		}).
		Return(nil)

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)
	nm.mai.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := nm.initComponents(nm.ctx)
	assert.NoError(t, err)

	assert.Equal(t, nm.mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))
	assert.Nil(t, nm.Orchestrator(core.LegacySystemNamespace))
}

func TestInitComponentsPluginsFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mai.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	nm.namespaces = map[string]*namespace{}
	nm.plugins = map[string]*plugin{
		"basicauth": nm.plugins["basicauth"],
	}
	err := nm.initComponents(context.Background())
	assert.Regexp(t, "pop", err)
}

func TestInitDatabaseFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := nm.initPlugins(context.Background(), map[string]*plugin{
		"postgres": nm.plugins["postgres"],
	})
	assert.EqualError(t, err, "pop")
}

func TestInitBlockchainFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(fmt.Errorf("pop"))

	err := nm.initPlugins(context.Background(), map[string]*plugin{
		"ethereum": nm.plugins["ethereum"],
	})
	assert.EqualError(t, err, "pop")
}

func TestInitDataExchangeFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := nm.initPlugins(context.Background(), map[string]*plugin{
		"ffdx": nm.plugins["ffdx"],
	})
	assert.EqualError(t, err, "pop")
}

func TestInitSharedStorageFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mps.On("Init", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := nm.initPlugins(context.Background(), map[string]*plugin{
		"ipfs": nm.plugins["ipfs"],
	})
	assert.EqualError(t, err, "pop")
}

func TestInitTokensFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mti.On("Init", mock.Anything, mock.Anything, mock.Anything, nil).Return(fmt.Errorf("pop"))

	err := nm.initPlugins(context.Background(), map[string]*plugin{
		"erc1155": nm.plugins["erc1155"],
	})
	assert.EqualError(t, err, "pop")
}

func TestInitEventsFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mev.On("Init", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := nm.initPlugins(context.Background(), map[string]*plugin{
		"websockets": nm.plugins["websockets"],
	})
	assert.EqualError(t, err, "pop")
}

func TestInitAuthFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mai.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := nm.initPlugins(context.Background(), map[string]*plugin{
		"basicauth": nm.plugins["basicauth"],
	})
	assert.EqualError(t, err, "pop")
}

func TestInitOrchestratorFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mo.On("Init", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)

	err := nm.initNamespaces(nm.ctx, nm.namespaces)
	assert.Regexp(t, "pop", err)
}

func TestInitVersion1(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mo.On("Init", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			nm.namespaces["default"].config.Multiparty.Enabled = true
			nm.namespaces["default"].Contracts.Active = &core.MultipartyContract{
				Location: fftypes.JSONAnyPtr("{}"),
				Info: core.MultipartyContractInfo{
					Version: 1,
				},
			}
		}).
		Return(nil).Twice() // legacy system namespace
	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, core.LegacySystemNamespace).Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)

	err := nm.initNamespaces(nm.ctx, nm.namespaces)
	assert.NoError(t, err)

	assert.Equal(t, nm.mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))
	assert.NotNil(t, nm.Orchestrator(core.LegacySystemNamespace))

}

func TestInitFFSystemWithTerminatedV1Contract(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	nm.metricsEnabled = true

	nm.mo.On("Init", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			nm.namespaces["default"].config.Multiparty.Enabled = true
			nm.namespaces["default"].Contracts = &core.MultipartyContracts{
				Active: &core.MultipartyContract{
					Info: core.MultipartyContractInfo{
						Version: 2,
					},
				},
				Terminated: []*core.MultipartyContract{{
					Location: fftypes.JSONAnyPtr("{}"),
					Info: core.MultipartyContractInfo{
						Version: 1,
					},
				}},
			}
		}).
		Return(nil).Twice() // legacy system namespace

	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, core.LegacySystemNamespace).Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)

	err := nm.initNamespaces(nm.ctx, nm.namespaces)
	assert.NoError(t, err)

	assert.Equal(t, nm.mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))

}

func TestLegacyNamespaceConflictingPlugins(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	nm.metricsEnabled = true

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: default
    predefined:
    - name: default
      plugins: [ethereum, postgres, erc721, ipfs, ffdx]
      multiparty:
        enabled: true
        networkNamespace: default
    - name: ns2
      plugins: [ethereum, postgres, erc1155, ipfs, ffdx]
      multiparty:
        enabled: true
  `))
	assert.NoError(t, err)

	nm.mo.On("Init", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			nm.namespaces["default"].Contracts = &core.MultipartyContracts{
				Active: &core.MultipartyContract{
					Location: fftypes.JSONAnyPtr("{}"),
					Info: core.MultipartyContractInfo{
						Version: 1,
					},
				},
			}
			nm.namespaces["ns2"].Contracts = &core.MultipartyContracts{
				Active: &core.MultipartyContract{
					Location: fftypes.JSONAnyPtr("{}"),
					Info: core.MultipartyContractInfo{
						Version: 1,
					},
				},
			}
		}).
		Return(nil)

	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, "ns2").Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)

	nm.namespaces, err = nm.loadNamespaces(nm.ctx, viper.AllSettings(), nm.plugins)
	assert.Len(t, nm.namespaces, 2)
	assert.NoError(t, err)

	err = nm.initNamespaces(nm.ctx, nm.namespaces)
	assert.Regexp(t, "FF10421", err)

	assert.Equal(t, nm.mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))

}

func TestLegacyNamespaceConflictingPluginsTooManyPlugins(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	nm.metricsEnabled = true

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: default
    predefined:
    - name: default
      plugins: [ethereum, postgres, erc721, ipfs, ffdx]
      multiparty:
        enabled: true
    - name: ns2
      plugins: [ethereum, postgres, erc1155, erc721, ipfs, ffdx]
      multiparty:
        enabled: true
  `))
	assert.NoError(t, err)

	nm.mo.On("Init", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			nm.namespaces["default"].Contracts = &core.MultipartyContracts{
				Active: &core.MultipartyContract{
					Location: fftypes.JSONAnyPtr("{}"),
					Info: core.MultipartyContractInfo{
						Version: 1,
					},
				},
			}
			nm.namespaces["ns2"].Contracts = &core.MultipartyContracts{
				Active: &core.MultipartyContract{
					Location: fftypes.JSONAnyPtr("{}"),
					Info: core.MultipartyContractInfo{
						Version: 1,
					},
				},
			}
		}).
		Return(nil)

	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, "ns2").Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)

	nm.namespaces, err = nm.loadNamespaces(nm.ctx, viper.AllSettings(), nm.plugins)
	assert.Len(t, nm.namespaces, 2)
	assert.NoError(t, err)

	err = nm.initNamespaces(nm.ctx, nm.namespaces)
	assert.Regexp(t, "FF10421", err)

	assert.Equal(t, nm.mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))

}

func TestLegacyNamespaceMatchingPlugins(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	nm.metricsEnabled = true

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: default
    predefined:
    - name: default
      plugins: [ethereum, postgres, erc721, ipfs, ffdx]
      multiparty:
        enabled: true
    - name: ns2
      plugins: [ethereum, postgres, erc721, ipfs, ffdx]
      multiparty:
        enabled: true
  `))
	assert.NoError(t, err)

	nm.mo.On("Init", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			nm.namespaces["default"].Contracts = &core.MultipartyContracts{
				Active: &core.MultipartyContract{
					Location: fftypes.JSONAnyPtr("{}"),
					Info: core.MultipartyContractInfo{
						Version: 1,
					},
				},
			}
			nm.namespaces["ns2"].Contracts = &core.MultipartyContracts{
				Active: &core.MultipartyContract{
					Location: fftypes.JSONAnyPtr("{}"),
					Info: core.MultipartyContractInfo{
						Version: 1,
					},
				},
			}
		}).
		Return(nil)

	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, "ns2").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, core.LegacySystemNamespace).Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)

	nm.namespaces, err = nm.loadNamespaces(nm.ctx, viper.AllSettings(), nm.plugins)
	assert.Len(t, nm.namespaces, 2)
	assert.NoError(t, err)

	err = nm.initNamespaces(nm.ctx, nm.namespaces)
	assert.NoError(t, err)

	assert.Equal(t, nm.mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))

}

func TestInitVersion1Fail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mo.On("Init", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			nm.namespaces["default"].config.Multiparty.Enabled = true
			nm.namespaces["default"].Contracts.Active = &core.MultipartyContract{
				Location: fftypes.JSONAnyPtr("{}"),
				Info: core.MultipartyContractInfo{
					Version: 1,
				},
			}
		}).
		Return(nil).Once()
	nm.mo.On("Init", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Once()

	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, core.LegacySystemNamespace).Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)

	err := nm.initNamespaces(nm.ctx, nm.namespaces)
	assert.EqualError(t, err, "pop")

}

func TestInitNamespaceQueryFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, fmt.Errorf("pop"))

	err := nm.initNamespace(context.Background(), nm.namespaces["default"])
	assert.EqualError(t, err, "pop")
}

func TestInitNamespaceExistingUpsertFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	existing := &core.Namespace{
		NetworkName: "ns1",
	}

	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(existing, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(fmt.Errorf("pop"))

	err := nm.initNamespace(context.Background(), nm.namespaces["default"])
	assert.EqualError(t, err, "pop")
}

func TestDeprecatedDatabasePlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	difactory.InitConfigDeprecated(deprecatedDatabaseConfig)
	deprecatedDatabaseConfig.Set(coreconfig.PluginConfigType, "postgres")
	plugins := make(map[string]*plugin)
	err := nm.getDatabasePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDeprecatedDatabasePluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	difactory.InitConfigDeprecated(deprecatedDatabaseConfig)
	deprecatedDatabaseConfig.Set(coreconfig.PluginConfigType, "wrong")
	err := nm.getDatabasePlugins(context.Background(), make(map[string]*plugin), nm.dumpRootConfig())
	assert.Regexp(t, "FF10122.*wrong", err)
}

func TestDatabasePlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	difactory.InitConfig(databaseConfig)
	config.Set("plugins.database", []fftypes.JSONObject{{}})
	databaseConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	databaseConfig.AddKnownKey(coreconfig.PluginConfigType, "postgres")
	plugins := make(map[string]*plugin)
	err := nm.getDatabasePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDatabasePluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	difactory.InitConfig(databaseConfig)
	config.Set("plugins.database", []fftypes.JSONObject{{}})
	databaseConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	databaseConfig.AddKnownKey(coreconfig.PluginConfigType, "unknown")
	err := nm.getDatabasePlugins(context.Background(), make(map[string]*plugin), nm.dumpRootConfig())
	assert.Regexp(t, "FF10122.*unknown", err)
}

func TestDatabasePluginBadName(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	difactory.InitConfig(databaseConfig)
	config.Set("plugins.database", []fftypes.JSONObject{{}})
	databaseConfig.AddKnownKey(coreconfig.PluginConfigName, "wrong////")
	databaseConfig.AddKnownKey(coreconfig.PluginConfigType, "postgres")
	err := nm.getDatabasePlugins(context.Background(), make(map[string]*plugin), nm.dumpRootConfig())
	assert.Regexp(t, "FF00140", err)
}

func TestIdentityPluginBadName(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "wrong//")
	identityConfig.AddKnownKey(coreconfig.PluginConfigType, "tbd")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})
	err := nm.getIdentityPlugins(context.Background(), make(map[string]*plugin), nm.dumpRootConfig())
	assert.Regexp(t, "FF00140.*name", err)
}

func TestIdentityPluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	identityConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})
	err := nm.getIdentityPlugins(context.Background(), make(map[string]*plugin), nm.dumpRootConfig())
	assert.Regexp(t, "FF10212.*wrong", err)
}

func TestIdentityPluginNoType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "FF10386.*type", err)
}

func TestIdentityPlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	identityConfig.AddKnownKey(coreconfig.PluginConfigType, "onchain")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})
	plugins := make(map[string]*plugin)
	err := nm.getIdentityPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.NoError(t, err)
	assert.Equal(t, 1, len(plugins))
}

func TestDeprecatedBlockchainPlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	bifactory.InitConfigDeprecated(deprecatedBlockchainConfig)
	deprecatedBlockchainConfig.Set(coreconfig.PluginConfigType, "ethereum")
	plugins := make(map[string]*plugin)
	err := nm.getBlockchainPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDeprecatedBlockchainPluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	deprecatedBlockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	plugins := make(map[string]*plugin)
	err := nm.getBlockchainPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10110.*wrong", err)
}

func TestBlockchainPlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	bifactory.InitConfig(blockchainConfig)
	config.Set("plugins.blockchain", []fftypes.JSONObject{{}})
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "ethereum")
	plugins := make(map[string]*plugin)
	err := nm.getBlockchainPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestBlockchainPluginNoType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	bifactory.InitConfig(blockchainConfig)
	config.Set("plugins.blockchain", []fftypes.JSONObject{{}})
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	plugins := make(map[string]*plugin)
	err := nm.getBlockchainPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10386", err)
}

func TestBlockchainPluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	bifactory.InitConfig(blockchainConfig)
	config.Set("plugins.blockchain", []fftypes.JSONObject{{}})
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong//")

	plugins := make(map[string]*plugin)
	err := nm.getBlockchainPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10110", err)
}

func TestDeprecatedSharedStoragePlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	ssfactory.InitConfigDeprecated(deprecatedSharedStorageConfig)
	deprecatedSharedStorageConfig.Set(coreconfig.PluginConfigType, "ipfs")
	plugins := make(map[string]*plugin)
	err := nm.getSharedStoragePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDeprecatedSharedStoragePluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	ssfactory.InitConfigDeprecated(deprecatedSharedStorageConfig)
	deprecatedSharedStorageConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	plugins := make(map[string]*plugin)
	err := nm.getSharedStoragePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10134.*wrong", err)
}

func TestSharedStoragePlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	ssfactory.InitConfig(sharedstorageConfig)
	config.Set("plugins.sharedstorage", []fftypes.JSONObject{{}})
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigType, "ipfs")
	plugins := make(map[string]*plugin)
	err := nm.getSharedStoragePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestSharedStoragePluginNoType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	ssfactory.InitConfig(sharedstorageConfig)
	config.Set("plugins.sharedstorage", []fftypes.JSONObject{{}})
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	plugins := make(map[string]*plugin)
	err := nm.getSharedStoragePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10386", err)
}

func TestSharedStoragePluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	ssfactory.InitConfig(sharedstorageConfig)
	config.Set("plugins.sharedstorage", []fftypes.JSONObject{{}})
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong//")

	plugins := make(map[string]*plugin)
	err := nm.getSharedStoragePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10134", err)
}

func TestDeprecatedDataExchangePlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	dxfactory.InitConfigDeprecated(deprecatedDataexchangeConfig)
	deprecatedDataexchangeConfig.Set(coreconfig.PluginConfigType, "ffdx")
	plugins := make(map[string]*plugin)
	err := nm.getDataExchangePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDeprecatedDataExchangePluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	dxfactory.InitConfigDeprecated(deprecatedDataexchangeConfig)
	deprecatedDataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	plugins := make(map[string]*plugin)
	err := nm.getDataExchangePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10213.*wrong", err)
}

func TestDataExchangePlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	dxfactory.InitConfig(dataexchangeConfig)
	config.Set("plugins.dataexchange", []fftypes.JSONObject{{}})
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "ffdx")
	plugins := make(map[string]*plugin)
	err := nm.getDataExchangePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDataExchangePluginNoType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	dxfactory.InitConfig(dataexchangeConfig)
	config.Set("plugins.dataexchange", []fftypes.JSONObject{{}})
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	plugins := make(map[string]*plugin)
	err := nm.getDataExchangePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10386", err)
}

func TestDataExchangePluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	dxfactory.InitConfig(dataexchangeConfig)
	config.Set("plugins.dataexchange", []fftypes.JSONObject{{}})
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong//")

	plugins := make(map[string]*plugin)
	err := nm.getDataExchangePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10213", err)
}

func TestDeprecatedTokensPlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	config.Set("tokens", []fftypes.JSONObject{{}})
	deprecatedTokensConfig.AddKnownKey(coreconfig.PluginConfigName, "test")
	deprecatedTokensConfig.AddKnownKey(tokens.TokensConfigPlugin, "fftokens")
	plugins := make(map[string]*plugin)
	err := nm.getTokensPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDeprecatedTokensPluginNoName(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	config.Set("tokens", []fftypes.JSONObject{{}})
	deprecatedTokensConfig.AddKnownKey(tokens.TokensConfigPlugin, "fftokens")
	plugins := make(map[string]*plugin)
	err := nm.getTokensPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10273", err)
}

func TestDeprecatedTokensPluginBadName(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	config.Set("tokens", []fftypes.JSONObject{{}})
	deprecatedTokensConfig.AddKnownKey(coreconfig.PluginConfigName, "BAD!")
	deprecatedTokensConfig.AddKnownKey(tokens.TokensConfigPlugin, "fftokens")
	plugins := make(map[string]*plugin)
	err := nm.getTokensPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF00140", err)
}

func TestDeprecatedTokensPluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	config.Set("tokens", []fftypes.JSONObject{{}})
	deprecatedTokensConfig.AddKnownKey(coreconfig.PluginConfigName, "test")
	deprecatedTokensConfig.AddKnownKey(tokens.TokensConfigPlugin, "wrong")
	plugins := make(map[string]*plugin)
	err := nm.getTokensPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10272.*wrong", err)
}

func TestTokensPlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	tifactory.InitConfig(tokensConfig)
	config.Set("plugins.tokens", []fftypes.JSONObject{{}})
	tokensConfig.AddKnownKey(coreconfig.PluginConfigName, "erc20_erc721")
	tokensConfig.AddKnownKey(coreconfig.PluginConfigType, "fftokens")
	plugins := make(map[string]*plugin)
	err := nm.getTokensPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestTokensPluginDuplicateBroadcastName(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	tifactory.InitConfig(tokensConfig)
	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  plugins:
    tokens:
    - name: test1
      broadcastName: remote1
      type: fftokens
      fftokens:
        url: http://tokens:3000
    - name: test2
      broadcastName: remote1
      type: fftokens
      fftokens:
        url: http://tokens:3000
  `))
	assert.NoError(t, err)

	plugins := make(map[string]*plugin)
	err = nm.getTokensPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10419", err)
}

func TestMultipleTokensPluginsWithBroadcastName(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	tifactory.InitConfig(tokensConfig)
	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  plugins:
    tokens:
    - name: test1
      broadcastName: remote1
      type: fftokens
      fftokens:
        url: http://tokens:3000
    - name: test2
      broadcastName: remote2
      type: fftokens
      fftokens:
        url: http://tokens:3000
  `))
	assert.NoError(t, err)

	plugins := make(map[string]*plugin)
	err = nm.getTokensPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 2, len(plugins))
	assert.NoError(t, err)
}

func TestTokensPluginNoType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	tifactory.InitConfig(tokensConfig)
	config.Set("plugins.tokens", []fftypes.JSONObject{{}})
	tokensConfig.AddKnownKey(coreconfig.PluginConfigName, "erc20_erc721")
	plugins := make(map[string]*plugin)
	err := nm.getTokensPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10386", err)
}

func TestTokensPluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	tifactory.InitConfig(tokensConfig)
	config.Set("plugins.tokens", []fftypes.JSONObject{{}})
	tokensConfig.AddKnownKey(coreconfig.PluginConfigName, "erc20_erc721")
	tokensConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")

	plugins := make(map[string]*plugin)
	err := nm.getTokensPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10272", err)
}

func TestTokensPluginDuplicate(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	tifactory.InitConfig(tokensConfig)
	config.Set("plugins.tokens", []fftypes.JSONObject{{}})
	tokensConfig.AddKnownKey(coreconfig.PluginConfigName, "erc20_erc721")
	tokensConfig.AddKnownKey(coreconfig.PluginConfigType, "fftokens")
	plugins := make(map[string]*plugin)
	plugins["erc20_erc721"] = &plugin{}
	err := nm.getTokensPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10395", err)
}

func TestEventsPluginDefaults(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	plugins := make(map[string]*plugin)
	err := nm.getEventPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 3, len(plugins))
	assert.NoError(t, err)
}

func TestAuthPlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	authfactory.InitConfigArray(authConfig)
	config.Set("plugins.auth", []fftypes.JSONObject{{}})
	authConfig.AddKnownKey(coreconfig.PluginConfigName, "basicauth")
	authConfig.AddKnownKey(coreconfig.PluginConfigType, "basic")
	plugins := make(map[string]*plugin)
	err := nm.getAuthPlugin(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestAuthPluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	authfactory.InitConfigArray(authConfig)
	config.Set("plugins.auth", []fftypes.JSONObject{{}})
	authConfig.AddKnownKey(coreconfig.PluginConfigName, "basicauth")
	authConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")

	plugins := make(map[string]*plugin)
	err := nm.getAuthPlugin(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF00168", err)
}

func TestAuthPluginInvalid(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	authfactory.InitConfigArray(authConfig)
	config.Set("plugins.auth", []fftypes.JSONObject{{}})
	authConfig.AddKnownKey(coreconfig.PluginConfigName, "bad name not allowed")
	authConfig.AddKnownKey(coreconfig.PluginConfigType, "basic")
	plugins := make(map[string]*plugin)
	err := nm.getAuthPlugin(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF00140", err)
}

func TestEventsPluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	config.Set(coreconfig.EventTransportsEnabled, []string{"!unknown!"})

	plugins := make(map[string]*plugin)
	err := nm.getEventPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10172", err)
}

func TestInitBadNamespace(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: "!Badness"
    predefined:
    - name: "!Badness"
    `))
	assert.NoError(t, err)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err = nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "FF00140", err)
}

func TestLoadNamespacesReservedName(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ff_system
    `))
	assert.NoError(t, err)

	_, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.Regexp(t, "FF10388", err)
}

func TestLoadNamespacesReservedNetworkName(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      multiparty:
        enabled: true
        networknamespace: ff_system
    `))
	assert.NoError(t, err)

	_, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.Regexp(t, "FF10388", err)
}

func TestLoadNamespacesDuplicate(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

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

	newNS, err := nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.NoError(t, err)
	assert.Len(t, newNS, 1)
}

func TestLoadNamespacesNoName(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    predefined:
    - plugins: [postgres]
    `))
	assert.NoError(t, err)

	_, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.Regexp(t, "FF10166", err)
}

func TestLoadNamespacesNoDefault(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns2
      plugins: [postgres]
  `))
	assert.NoError(t, err)

	_, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.Regexp(t, "FF10166", err)
}

func TestLoadNamespacesUseDefaults(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      multiparty:
        contract:
        - location:
          address: 0x1234
  org:
    name: org1
  node:
    name: node1
  `))
	assert.NoError(t, err)

	newNS, err := nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.NoError(t, err)
	assert.Len(t, newNS, 1)
	assert.Equal(t, "oldest", newNS["ns1"].config.Multiparty.Contracts[0].FirstEvent)
}

func TestLoadNamespacesNonMultipartyNoDatabase(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: []
  `))
	assert.NoError(t, err)

	nm.namespaces, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.Regexp(t, "FF10392", err)

}

func TestLoadNamespacesMultipartyUnknownPlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [basicauth, bad]
      multiparty:
        enabled: true
  `))
	assert.NoError(t, err)

	nm.namespaces, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.Regexp(t, "FF10390.*unknown", err)
}

func TestLoadNamespacesMultipartyMultipleBlockchains(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

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

	nm.namespaces, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.Regexp(t, "FF10394.*blockchain", err)
}

func TestLoadNamespacesMultipartyMultipleDX(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

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

	nm.namespaces, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.Regexp(t, "FF10394.*dataexchange", err)
}

func TestLoadNamespacesMultipartyMultipleSS(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

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

	nm.namespaces, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.Regexp(t, "FF10394.*sharedstorage", err)
}

func TestLoadNamespacesMultipartyMultipleDB(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

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

	nm.namespaces, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.Regexp(t, "FF10394.*database", err)
}

func TestInitNamespacesMultipartyWithAuth(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [ethereum, postgres, ipfs, ffdx, basicauth]
      multiparty:
        enabled: true
  `))
	assert.NoError(t, err)

	nm.namespaces, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.NoError(t, err)
}

func TestLoadNamespacesNonMultipartyWithAuth(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [ethereum, postgres, basicauth]
      multiparty:
        enabled: false
  `))
	assert.NoError(t, err)

	nm.namespaces, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.NoError(t, err)
}

func TestLoadNamespacesMultipartyContract(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      multiparty:
        enabled: true
        contract:
        - location:
            address: 0x4ae50189462b0e5d52285f59929d037f790771a6
          firstEvent: oldest
  `))
	assert.NoError(t, err)

	nm.namespaces, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.NoError(t, err)
}

func TestLoadNamespacesNonMultipartyMultipleDB(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [postgres, postgres]
  `))
	assert.NoError(t, err)

	nm.namespaces, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.Regexp(t, "FF10394.*database", err)
}

func TestLoadNamespacesNonMultipartyMultipleBlockchains(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [ethereum, ethereum]
  `))
	assert.NoError(t, err)

	nm.namespaces, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.Regexp(t, "FF10394.*blockchain", err)
}

func TestLoadNamespacesMultipartyMissingPlugins(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

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

	nm.namespaces, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.Regexp(t, "FF10391", err)
}

func TestLoadNamespacesNonMultipartyUnknownPlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [basicauth, erc721, bad]
  `))
	assert.NoError(t, err)

	nm.namespaces, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.Regexp(t, "FF10390.*unknown", err)
}

func TestLoadNamespacesNonMultipartyTokens(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [postgres, erc721]
  `))
	assert.NoError(t, err)

	nm.namespaces, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.NoError(t, err)
}

func TestStart(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mbi.On("Start", mock.Anything).Return(nil)
	nm.mdx.On("Start", mock.Anything).Return(nil)
	nm.mti.On("Start", mock.Anything).Return(nil)
	nm.mo.On("Start", mock.Anything).Return(nil)

	err := nm.Start()
	assert.NoError(t, err)
}

func TestStartBlockchainFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mo.On("Start", mock.Anything).Return(nil)
	nm.mbi.On("Start").Return(fmt.Errorf("pop"))

	err := nm.startNamespacesAndPlugins(nm.namespaces, map[string]*plugin{
		"ethereum": nm.plugins["ethereum"],
	})
	assert.EqualError(t, err, "pop")

}

func TestStartDataExchangeFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mo.On("Start", mock.Anything).Return(nil)
	nm.mdx.On("Start").Return(fmt.Errorf("pop"))

	err := nm.startNamespacesAndPlugins(nm.namespaces, map[string]*plugin{
		"ffdx": nm.plugins["ffdx"],
	})
	assert.EqualError(t, err, "pop")

}

func TestStartTokensFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mo.On("Start", mock.Anything).Return(nil)
	nm.mti.On("Start").Return(fmt.Errorf("pop"))

	err := nm.startNamespacesAndPlugins(nm.namespaces, map[string]*plugin{
		"erc1155": nm.plugins["erc1155"],
	})
	assert.EqualError(t, err, "pop")

}

func TestStartOrchestratorFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mo.On("Start", mock.Anything).Return(fmt.Errorf("pop"))

	err := nm.startNamespacesAndPlugins(nm.namespaces, map[string]*plugin{})
	assert.EqualError(t, err, "pop")
}

func TestWaitStop(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.mo.On("WaitStop").Return()
	nm.mae.On("WaitStop").Return()

	nm.WaitStop()

	nm.mo.AssertExpectations(t)
	nm.mae.AssertExpectations(t)
}

func TestReset(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	childCtx, childCancel := context.WithCancel(context.Background())
	err := nm.Reset(childCtx)
	assert.NoError(t, err)
	childCancel()

	assert.True(t, <-nm.namespaceManager.reset)
}

func TestResetRejectIfConfigAutoReload(t *testing.T) {
	config.RootConfigReset()
	config.Set(coreconfig.ConfigAutoReload, true)

	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	config.Set(coreconfig.ConfigAutoReload, true)

	err := nm.Reset(context.Background())
	assert.Regexp(t, "FF10433", err)

}

func TestLoadMetrics(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.metrics = nil

	nm.loadManagers(context.Background())
}

func TestLoadAdminEvents(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	nm.adminEvents = nil

	nm.loadManagers(context.Background())
	assert.NotNil(t, nm.SPIEvents())
}

func TestGetNamespaces(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	results, err := nm.GetNamespaces(context.Background())
	assert.Nil(t, err)
	assert.Len(t, results, 1)
}

func TestGetOperationByNamespacedID(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	mo := &orchestratormocks.Orchestrator{}
	nm.namespaces = map[string]*namespace{
		"default": {orchestrator: mo},
	}

	opID := fftypes.NewUUID()
	mo.On("GetOperationByID", context.Background(), opID.String()).Return(nil, nil)

	op, err := nm.GetOperationByNamespacedID(context.Background(), "default:"+opID.String())
	assert.Nil(t, err)
	assert.Nil(t, op)

	mo.AssertExpectations(t)
}

func TestGetOperationByNamespacedIDBadID(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	mo := &orchestratormocks.Orchestrator{}
	nm.namespaces = map[string]*namespace{
		"default": {orchestrator: mo},
	}

	_, err := nm.GetOperationByNamespacedID(context.Background(), "default:bad")
	assert.Regexp(t, "FF00138", err)

	mo.AssertExpectations(t)
}

func TestGetOperationByNamespacedIDNoOrchestrator(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	mo := &orchestratormocks.Orchestrator{}
	nm.namespaces = map[string]*namespace{
		"default": {orchestrator: mo},
	}

	opID := fftypes.NewUUID()

	_, err := nm.GetOperationByNamespacedID(context.Background(), "bad:"+opID.String())
	assert.Regexp(t, "FF10109", err)

	mo.AssertExpectations(t)
}

func TestResolveOperationByNamespacedID(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	mo := &orchestratormocks.Orchestrator{}
	mom := &operationmocks.Manager{}
	nm.namespaces = map[string]*namespace{
		"default": {orchestrator: mo},
	}

	opID := fftypes.NewUUID()
	mo.On("Operations").Return(mom)
	mom.On("ResolveOperationByID", context.Background(), opID, mock.Anything).Return(nil)

	err := nm.ResolveOperationByNamespacedID(context.Background(), "default:"+opID.String(), &core.OperationUpdateDTO{})
	assert.Nil(t, err)

	mo.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestResolveOperationByNamespacedIDBadID(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	mo := &orchestratormocks.Orchestrator{}
	nm.namespaces = map[string]*namespace{
		"default": {orchestrator: mo},
	}

	err := nm.ResolveOperationByNamespacedID(context.Background(), "default:bad", &core.OperationUpdateDTO{})
	assert.Regexp(t, "FF00138", err)

	mo.AssertExpectations(t)
}

func TestResolveOperationByNamespacedIDNoOrchestrator(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	mo := &orchestratormocks.Orchestrator{}
	nm.namespaces = map[string]*namespace{
		"default": {orchestrator: mo},
	}

	opID := fftypes.NewUUID()

	err := nm.ResolveOperationByNamespacedID(context.Background(), "bad:"+opID.String(), &core.OperationUpdateDTO{})
	assert.Regexp(t, "FF10109", err)

	mo.AssertExpectations(t)
}

func TestAuthorize(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()
	mo := &orchestratormocks.Orchestrator{}
	mo.On("Authorize", mock.Anything, mock.Anything).Return(nil)
	nm.namespaces["ns1"] = &namespace{
		orchestrator: mo,
	}
	nm.utOrchestrator = mo
	err := nm.Authorize(context.Background(), &fftypes.AuthReq{
		Namespace: "ns1",
	})
	assert.NoError(t, err)

	mo.AssertExpectations(t)
}

func TestValidateNonMultipartyConfig(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
    namespaces:
      default: ns1
      predefined:
      - name: ns1
        plugins: [postgres, erc721]
    `))
	assert.NoError(t, err)

	_, err = nm.loadNamespaces(context.Background(), nm.dumpRootConfig(), nm.plugins)
	assert.NoError(t, err)
}
