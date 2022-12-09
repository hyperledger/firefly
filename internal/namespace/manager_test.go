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
	"github.com/hyperledger/firefly-common/pkg/auth"
	"github.com/hyperledger/firefly-common/pkg/auth/authfactory"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/blockchain/bifactory"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/database/difactory"
	"github.com/hyperledger/firefly/internal/dataexchange/dxfactory"
	"github.com/hyperledger/firefly/internal/identity/iifactory"
	"github.com/hyperledger/firefly/internal/metrics"
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
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/events"
	"github.com/hyperledger/firefly/pkg/identity"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var testBaseConfig = `
---
namespaces:
  predefined:
    - defaultKey: 0xbEa50Ec98776beF144Fc63078e7b15291Ac64cfA
      name: default
      plugins:
        - ethereum
        - postgres
        - ffdx
        - ipfs
        - erc721
        - erc1155
        - basicauth
      multiparty:
        enabled: true
        node:
          name: node1
        org:
          name: org1
        contract:
          - firstevent: "0"
            location:
              address: 0x7359d2ecc199C48369b390522c29b77A5Af30882
plugins:
  blockchain:
    - name: ethereum
      type: ethereum
  database:
    - name: postgres
      type: postgres
  dataexchange:
    - name: ffdx
      type: ffdx
  sharedstorage:
    - ipfs:
      name: ipfs
      type: ipfs
  tokens:
    - name: erc721
      type: type1
    - name: erc1155
      type: type2
  auth:
    - name: basicauth
      type: basicauth	  
`

type testNamespaceManager struct {
	namespaceManager
	mmi *metricsmocks.Manager
	mae *spieventsmocks.Manager
	mbi *blockchainmocks.Plugin
	cmi *cachemocks.Manager
	mdi *databasemocks.Plugin
	mdx *dataexchangemocks.Plugin
	mps *sharedstoragemocks.Plugin
	mti []*tokenmocks.Plugin
	mei []*eventsmocks.Plugin
	mai *authmocks.Plugin
	mii *identitymocks.Plugin
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
	nm.mti[0].AssertExpectations(t)
	nm.mti[1].AssertExpectations(t)
	nm.mai.AssertExpectations(t)
	nm.mii.AssertExpectations(t)
	nm.mei[0].AssertExpectations(t)
	nm.mei[1].AssertExpectations(t)
	nm.mei[2].AssertExpectations(t)
	nm.mo.AssertExpectations(t)
}

func factoryMocks(m *mock.Mock, name string) {
	m.On("Name").Return(name).Maybe()
	m.On("InitConfig", mock.Anything).Maybe()
}

func newTestNamespaceManager(t *testing.T, initConfig bool) (*testNamespaceManager, func()) {
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
		mti: []*tokenmocks.Plugin{{}, {}},
		mei: []*eventsmocks.Plugin{{}, {}, {}},
		mai: &authmocks.Plugin{},
		mii: &identitymocks.Plugin{},
		mo:  &orchestratormocks.Orchestrator{},
	}
	factoryMocks(&nm.mbi.Mock, "ethereum")
	factoryMocks(&nm.mdi.Mock, "postgres")
	factoryMocks(&nm.mdx.Mock, "ffdx")
	factoryMocks(&nm.mps.Mock, "ipfs")
	factoryMocks(&nm.mti[0].Mock, "erc721")
	factoryMocks(&nm.mti[1].Mock, "erc1155")
	factoryMocks(&nm.mei[0].Mock, "system")
	factoryMocks(&nm.mei[1].Mock, "websockets")
	factoryMocks(&nm.mei[2].Mock, "webhooks")
	factoryMocks(&nm.mai.Mock, "basicauth")
	nm.namespaceManager = namespaceManager{
		ctx:                 ctx,
		cancelCtx:           cancelCtx,
		reset:               make(chan bool, 1),
		namespaces:          make(map[string]*namespace),
		plugins:             make(map[string]*plugin),
		tokenBroadcastNames: make(map[string]string),
		orchestratorFactory: func(ns *core.Namespace, config orchestrator.Config, plugins *orchestrator.Plugins, metrics metrics.Manager, cacheManager cache.Manager) orchestrator.Orchestrator {
			return nm.mo
		},
		blockchainFactory: func(ctx context.Context, pluginType string) (blockchain.Plugin, error) {
			return nm.mbi, nil
		},
		databaseFactory: func(ctx context.Context, pluginType string) (database.Plugin, error) {
			return nm.mdi, nil
		},
		dataexchangeFactory: func(ctx context.Context, pluginType string) (dataexchange.Plugin, error) {
			return nm.mdx, nil
		},
		sharedstorageFactory: func(ctx context.Context, pluginType string) (sharedstorage.Plugin, error) {
			return nm.mps, nil
		},
		tokensFactory: func(ctx context.Context, pluginType string) (tokens.Plugin, error) {
			if pluginType == "type1" {
				return nm.mti[0], nil
			}
			return nm.mti[1], nil
		},
		identityFactory: func(ctx context.Context, pluginType string) (identity.Plugin, error) {
			return nm.mii, nil
		},
		eventsFactory: func(ctx context.Context, pluginType string) (events.Plugin, error) {
			switch pluginType {
			case "system":
				return nm.mei[0], nil
			case "websockets":
				return nm.mei[1], nil
			case "webhooks":
				return nm.mei[2], nil
			default:
				panic(fmt.Errorf("Add plugin type %s to test", pluginType))
			}
		},
		authFactory: func(ctx context.Context, pluginType string) (auth.Plugin, error) {
			return nm.mai, nil
		},
	}
	nm.watchConfig = func() {
		<-ctx.Done()
	}
	nm.namespaceManager.metrics = nm.mmi
	nm.namespaceManager.adminEvents = nm.mae

	if initConfig {
		viper.SetConfigType("yaml")
		err := viper.ReadConfig(strings.NewReader(testBaseConfig))
		assert.NoError(t, err)

		nm.mo.On("Init", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				nm.namespaces["default"].Contracts.Active = &core.MultipartyContract{
					Info: core.MultipartyContractInfo{
						Version: 2,
					},
				}
			}).
			Return(nil).
			Once()

		nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil).Once()
		nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return().Once()
		nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil).Once()
		nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
		nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil).Once()
		nm.mti[0].On("Init", mock.Anything, mock.Anything, "erc721", mock.Anything).Return(nil).Once()
		nm.mti[1].On("Init", mock.Anything, mock.Anything, "erc1155", mock.Anything).Return(nil).Once()
		nm.mei[0].On("Init", mock.Anything, mock.Anything).Return(nil)
		nm.mei[1].On("Init", mock.Anything, mock.Anything).Return(nil)
		nm.mei[2].On("Init", mock.Anything, mock.Anything).Return(nil)
		nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil).Once()
		nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil).Once()
		nm.mai.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = nm.Init(nm.ctx, nm.cancelCtx, nm.reset)
		assert.NoError(t, err)
	}

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
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	nm.metricsEnabled = true

	nm.mei[0].On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mei[1].On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mei[2].On("Init", mock.Anything, mock.Anything).Return(nil)

	err := nm.Init(nm.ctx, nm.cancelCtx, nm.reset)
	assert.NoError(t, err)

	assert.Len(t, nm.plugins, 3) // events
	assert.Empty(t, nm.namespaces)
}

func TestInitAllPlugins(t *testing.T) {
	_, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()
}

func TestInitComponentsPluginsFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := nm.initPlugins(context.Background(), map[string]*plugin{
		"postgres": nm.plugins["postgres"],
	})
	assert.EqualError(t, err, "pop")
}

func TestInitBlockchainFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(fmt.Errorf("pop"))

	err := nm.initPlugins(context.Background(), map[string]*plugin{
		"ethereum": nm.plugins["ethereum"],
	})
	assert.EqualError(t, err, "pop")
}

func TestInitDataExchangeFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := nm.initPlugins(context.Background(), map[string]*plugin{
		"ffdx": nm.plugins["ffdx"],
	})
	assert.EqualError(t, err, "pop")
}

func TestInitSharedStorageFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.mps.On("Init", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := nm.initPlugins(context.Background(), map[string]*plugin{
		"ipfs": nm.plugins["ipfs"],
	})
	assert.EqualError(t, err, "pop")
}

func TestInitTokensFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.mti[0].On("Init", mock.Anything, mock.Anything, "erc721", mock.Anything).Return(fmt.Errorf("pop"))

	err := nm.initPlugins(context.Background(), map[string]*plugin{
		"erc721": nm.plugins["erc721"],
	})
	assert.EqualError(t, err, "pop")
}

func TestInitEventsFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()

	for _, mei := range nm.mei {
		mei.On("Init", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Maybe()
	}

	err := nm.Init(nm.ctx, nm.cancelCtx, nm.reset)
	assert.EqualError(t, err, "pop")
}

func TestInitAuthFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.mai.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := nm.initPlugins(context.Background(), map[string]*plugin{
		"basicauth": nm.plugins["basicauth"],
	})
	assert.EqualError(t, err, "pop")
}

func TestInitOrchestratorFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.mo.On("Init", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)

	err := nm.initNamespaces(nm.ctx, nm.namespaces)
	assert.Regexp(t, "pop", err)
}

func TestInitVersion1(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
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
	nm, cleanup := newTestNamespaceManager(t, true)
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()
	nm.metricsEnabled = true

	coreconfig.Reset()
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

	nm.namespaces, err = nm.loadNamespaces(nm.ctx, nm.dumpRootConfig(), nm.plugins)
	assert.Len(t, nm.namespaces, 2)
	assert.NoError(t, err)

	err = nm.initNamespaces(nm.ctx, nm.namespaces)
	assert.Regexp(t, "FF10421", err)

	assert.Equal(t, nm.mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))

}

func TestLegacyNamespaceConflictingPluginsTooManyPlugins(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()
	nm.metricsEnabled = true

	coreconfig.Reset()
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

	nm.namespaces, err = nm.loadNamespaces(nm.ctx, nm.dumpRootConfig(), nm.plugins)
	assert.Len(t, nm.namespaces, 2)
	assert.NoError(t, err)

	err = nm.initNamespaces(nm.ctx, nm.namespaces)
	assert.Regexp(t, "FF10421", err)

	assert.Equal(t, nm.mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))

}

func TestLegacyNamespaceMatchingPlugins(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()
	nm.metricsEnabled = true

	coreconfig.Reset()
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

	nm.namespaces, err = nm.loadNamespaces(nm.ctx, nm.dumpRootConfig(), nm.plugins)
	assert.Len(t, nm.namespaces, 2)
	assert.NoError(t, err)

	err = nm.initNamespaces(nm.ctx, nm.namespaces)
	assert.NoError(t, err)

	assert.Equal(t, nm.mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))

}

func TestInitVersion1Fail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, fmt.Errorf("pop"))

	err := nm.initNamespace(context.Background(), nm.namespaces["default"])
	assert.EqualError(t, err, "pop")
}

func TestInitNamespaceExistingUpsertFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
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
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	difactory.InitConfigDeprecated(deprecatedDatabaseConfig)
	deprecatedDatabaseConfig.Set(coreconfig.PluginConfigType, "postgres")
	plugins := make(map[string]*plugin)
	err := nm.getDatabasePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDeprecatedDatabasePluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	difactory.InitConfigDeprecated(deprecatedDatabaseConfig)
	deprecatedDatabaseConfig.Set(coreconfig.PluginConfigType, "wrong")
	nm.databaseFactory = func(ctx context.Context, pluginType string) (database.Plugin, error) {
		return nil, fmt.Errorf("pop")
	}
	err := nm.getDatabasePlugins(context.Background(), make(map[string]*plugin), nm.dumpRootConfig())
	assert.Regexp(t, "pop", err)
}

func TestDatabasePlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
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
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	difactory.InitConfig(databaseConfig)
	config.Set("plugins.database", []fftypes.JSONObject{{}})
	databaseConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	databaseConfig.AddKnownKey(coreconfig.PluginConfigType, "unknown")
	nm.databaseFactory = func(ctx context.Context, pluginType string) (database.Plugin, error) {
		return nil, fmt.Errorf("pop")
	}
	err := nm.getDatabasePlugins(context.Background(), make(map[string]*plugin), nm.dumpRootConfig())
	assert.Regexp(t, "pop", err)
}

func TestDatabasePluginBadName(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	difactory.InitConfig(databaseConfig)
	config.Set("plugins.database", []fftypes.JSONObject{{}})
	databaseConfig.AddKnownKey(coreconfig.PluginConfigName, "wrong////")
	databaseConfig.AddKnownKey(coreconfig.PluginConfigType, "postgres")
	err := nm.getDatabasePlugins(context.Background(), make(map[string]*plugin), nm.dumpRootConfig())
	assert.Regexp(t, "FF00140", err)
}

func TestIdentityPluginBadName(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "wrong//")
	identityConfig.AddKnownKey(coreconfig.PluginConfigType, "tbd")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})
	err := nm.getIdentityPlugins(context.Background(), make(map[string]*plugin), nm.dumpRootConfig())
	assert.Regexp(t, "FF00140.*name", err)
}

func TestIdentityPluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	identityConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})
	nm.identityFactory = func(ctx context.Context, pluginType string) (identity.Plugin, error) {
		return nil, fmt.Errorf("pop")
	}
	err := nm.getIdentityPlugins(context.Background(), make(map[string]*plugin), nm.dumpRootConfig())
	assert.Regexp(t, "pop", err)
}

func TestIdentityPluginNoType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "FF10386.*type", err)
}

func TestIdentityPlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
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
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	bifactory.InitConfigDeprecated(deprecatedBlockchainConfig)
	deprecatedBlockchainConfig.Set(coreconfig.PluginConfigType, "ethereum")
	plugins := make(map[string]*plugin)
	err := nm.getBlockchainPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDeprecatedBlockchainPluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	deprecatedBlockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	plugins := make(map[string]*plugin)
	nm.blockchainFactory = func(ctx context.Context, pluginType string) (blockchain.Plugin, error) {
		return nil, fmt.Errorf("pop")
	}
	err := nm.getBlockchainPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "pop", err)
}

func TestBlockchainPlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
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
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	bifactory.InitConfig(blockchainConfig)
	config.Set("plugins.blockchain", []fftypes.JSONObject{{}})
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	plugins := make(map[string]*plugin)
	err := nm.getBlockchainPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10386", err)
}

func TestBlockchainPluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	bifactory.InitConfig(blockchainConfig)
	config.Set("plugins.blockchain", []fftypes.JSONObject{{}})
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong//")

	plugins := make(map[string]*plugin)
	nm.blockchainFactory = func(ctx context.Context, pluginType string) (blockchain.Plugin, error) {
		return nil, fmt.Errorf("pop")
	}
	err := nm.getBlockchainPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "pop", err)
}

func TestDeprecatedSharedStoragePlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	ssfactory.InitConfigDeprecated(deprecatedSharedStorageConfig)
	deprecatedSharedStorageConfig.Set(coreconfig.PluginConfigType, "ipfs")
	plugins := make(map[string]*plugin)
	err := nm.getSharedStoragePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDeprecatedSharedStoragePluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	ssfactory.InitConfigDeprecated(deprecatedSharedStorageConfig)
	deprecatedSharedStorageConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	nm.sharedstorageFactory = func(ctx context.Context, pluginType string) (sharedstorage.Plugin, error) {
		return nil, fmt.Errorf("pop")
	}
	plugins := make(map[string]*plugin)
	err := nm.getSharedStoragePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "pop", err)
}

func TestSharedStoragePlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
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
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	ssfactory.InitConfig(sharedstorageConfig)
	config.Set("plugins.sharedstorage", []fftypes.JSONObject{{}})
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	plugins := make(map[string]*plugin)
	err := nm.getSharedStoragePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10386", err)
}

func TestSharedStoragePluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	ssfactory.InitConfig(sharedstorageConfig)
	config.Set("plugins.sharedstorage", []fftypes.JSONObject{{}})
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong//")

	nm.sharedstorageFactory = func(ctx context.Context, pluginType string) (sharedstorage.Plugin, error) {
		return nil, fmt.Errorf("pop")
	}
	plugins := make(map[string]*plugin)
	err := nm.getSharedStoragePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "pop", err)
}

func TestDeprecatedDataExchangePlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	dxfactory.InitConfigDeprecated(deprecatedDataexchangeConfig)
	deprecatedDataexchangeConfig.Set(coreconfig.PluginConfigType, "ffdx")
	plugins := make(map[string]*plugin)
	err := nm.getDataExchangePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestDeprecatedDataExchangePluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	dxfactory.InitConfigDeprecated(deprecatedDataexchangeConfig)
	deprecatedDataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")
	nm.dataexchangeFactory = func(ctx context.Context, pluginType string) (dataexchange.Plugin, error) {
		return nil, fmt.Errorf("pop")
	}
	plugins := make(map[string]*plugin)
	err := nm.getDataExchangePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "pop", err)
}

func TestDataExchangePlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
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
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	dxfactory.InitConfig(dataexchangeConfig)
	config.Set("plugins.dataexchange", []fftypes.JSONObject{{}})
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	plugins := make(map[string]*plugin)
	err := nm.getDataExchangePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10386", err)
}

func TestDataExchangePluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	dxfactory.InitConfig(dataexchangeConfig)
	config.Set("plugins.dataexchange", []fftypes.JSONObject{{}})
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong//")

	nm.dataexchangeFactory = func(ctx context.Context, pluginType string) (dataexchange.Plugin, error) {
		return nil, fmt.Errorf("pop")
	}
	plugins := make(map[string]*plugin)
	err := nm.getDataExchangePlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "pop", err)
}

func TestDeprecatedTokensPlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
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
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	config.Set("tokens", []fftypes.JSONObject{{}})
	deprecatedTokensConfig.AddKnownKey(tokens.TokensConfigPlugin, "fftokens")
	plugins := make(map[string]*plugin)
	err := nm.getTokensPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10273", err)
}

func TestDeprecatedTokensPluginBadName(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
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
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	config.Set("tokens", []fftypes.JSONObject{{}})
	deprecatedTokensConfig.AddKnownKey(coreconfig.PluginConfigName, "test")
	deprecatedTokensConfig.AddKnownKey(tokens.TokensConfigPlugin, "wrong")
	nm.tokensFactory = func(ctx context.Context, pluginType string) (tokens.Plugin, error) {
		return nil, fmt.Errorf("pop")
	}
	plugins := make(map[string]*plugin)
	err := nm.getTokensPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "pop", err)
}

func TestTokensPlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	tifactory.InitConfig(tokensConfig)
	config.Set("plugins.tokens", []fftypes.JSONObject{{}})
	tokensConfig.AddKnownKey(coreconfig.PluginConfigName, "erc20_erc721")
	plugins := make(map[string]*plugin)
	err := nm.getTokensPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "FF10386", err)
}

func TestTokensPluginBadType(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	tifactory.InitConfig(tokensConfig)
	config.Set("plugins.tokens", []fftypes.JSONObject{{}})
	tokensConfig.AddKnownKey(coreconfig.PluginConfigName, "erc20_erc721")
	tokensConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")

	nm.tokensFactory = func(ctx context.Context, pluginType string) (tokens.Plugin, error) {
		return nil, fmt.Errorf("pop")
	}
	plugins := make(map[string]*plugin)
	err := nm.getTokensPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "pop", err)
}

func TestTokensPluginDuplicate(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
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
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	plugins := make(map[string]*plugin)
	err := nm.getEventPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Equal(t, 3, len(plugins))
	assert.NoError(t, err)
}

func TestAuthPlugin(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
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
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	authfactory.InitConfigArray(authConfig)
	config.Set("plugins.auth", []fftypes.JSONObject{{}})
	authConfig.AddKnownKey(coreconfig.PluginConfigName, "basicauth")
	authConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")

	nm.authFactory = func(ctx context.Context, pluginType string) (auth.Plugin, error) {
		return nil, fmt.Errorf("pop")
	}
	plugins := make(map[string]*plugin)
	err := nm.getAuthPlugin(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "pop", err)
}

func TestAuthPluginInvalid(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, false)
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
	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()
	config.Set(coreconfig.EventTransportsEnabled, []string{"!unknown!"})

	nm.eventsFactory = func(ctx context.Context, pluginType string) (events.Plugin, error) {
		return nil, fmt.Errorf("pop")
	}
	plugins := make(map[string]*plugin)
	err := nm.getEventPlugins(context.Background(), plugins, nm.dumpRootConfig())
	assert.Regexp(t, "pop", err)
}

func TestInitBadNamespace(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.mbi.On("Start", mock.Anything).Return(nil)
	nm.mdx.On("Start", mock.Anything).Return(nil)
	nm.mti[0].On("Start", mock.Anything).Return(nil)
	nm.mti[1].On("Start", mock.Anything).Return(nil)
	nm.mo.On("Start", mock.Anything).Return(nil)

	err := nm.Start()
	assert.NoError(t, err)
}

func TestStartBlockchainFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.mo.On("Start", mock.Anything).Return(nil)
	nm.mbi.On("Start").Return(fmt.Errorf("pop"))

	err := nm.startNamespacesAndPlugins(nm.namespaces, map[string]*plugin{
		"ethereum": nm.plugins["ethereum"],
	})
	assert.EqualError(t, err, "pop")

}

func TestStartDataExchangeFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.mo.On("Start", mock.Anything).Return(nil)
	nm.mdx.On("Start").Return(fmt.Errorf("pop"))

	err := nm.startNamespacesAndPlugins(nm.namespaces, map[string]*plugin{
		"ffdx": nm.plugins["ffdx"],
	})
	assert.EqualError(t, err, "pop")

}

func TestStartTokensFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.mo.On("Start", mock.Anything).Return(nil)
	nm.mti[0].On("Start").Return(fmt.Errorf("pop"))

	err := nm.startNamespacesAndPlugins(nm.namespaces, map[string]*plugin{
		"erc721": nm.plugins["erc721"],
	})
	assert.EqualError(t, err, "pop")

}

func TestStartOrchestratorFail(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.mo.On("Start", mock.Anything).Return(fmt.Errorf("pop"))

	err := nm.startNamespacesAndPlugins(nm.namespaces, map[string]*plugin{})
	assert.EqualError(t, err, "pop")
}

func TestWaitStop(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.mo.On("WaitStop").Return()
	nm.mae.On("WaitStop").Return()

	nm.WaitStop()

	nm.mo.AssertExpectations(t)
	nm.mae.AssertExpectations(t)
}

func TestReset(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	childCtx, childCancel := context.WithCancel(context.Background())
	err := nm.Reset(childCtx)
	assert.NoError(t, err)
	childCancel()

	assert.True(t, <-nm.namespaceManager.reset)
}

func TestResetRejectIfConfigAutoReload(t *testing.T) {
	coreconfig.Reset()
	config.Set(coreconfig.ConfigAutoReload, true)

	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()
	config.Set(coreconfig.ConfigAutoReload, true)

	err := nm.Reset(context.Background())
	assert.Regexp(t, "FF10433", err)

}

func TestLoadMetrics(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.metrics = nil

	nm.loadManagers(context.Background())
}

func TestLoadAdminEvents(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	nm.adminEvents = nil

	nm.loadManagers(context.Background())
	assert.NotNil(t, nm.SPIEvents())
}

func TestGetNamespaces(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	results, err := nm.GetNamespaces(context.Background())
	assert.Nil(t, err)
	assert.Len(t, results, 1)
}

func TestGetOperationByNamespacedID(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
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
	nm, cleanup := newTestNamespaceManager(t, true)
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
	nm, cleanup := newTestNamespaceManager(t, true)
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
	nm, cleanup := newTestNamespaceManager(t, true)
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
	nm, cleanup := newTestNamespaceManager(t, true)
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
	nm, cleanup := newTestNamespaceManager(t, true)
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
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()
	nm.mo.On("Authorize", mock.Anything, mock.Anything).Return(nil)
	nm.namespaces["ns1"] = &namespace{
		orchestrator: nm.mo,
	}
	err := nm.Authorize(context.Background(), &fftypes.AuthReq{
		Namespace: "ns1",
	})
	assert.NoError(t, err)
}

func TestValidateNonMultipartyConfig(t *testing.T) {
	nm, cleanup := newTestNamespaceManager(t, true)
	defer cleanup()

	coreconfig.Reset()
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
