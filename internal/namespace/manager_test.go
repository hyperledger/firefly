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
	mmi  *metricsmocks.Manager
	mae  *spieventsmocks.Manager
	mbi  *blockchainmocks.Plugin
	cmi  *cachemocks.Manager
	mdi  *databasemocks.Plugin
	mdx  *dataexchangemocks.Plugin
	mps  *sharedstoragemocks.Plugin
	mti  *tokenmocks.Plugin
	mev  *eventsmocks.Plugin
	auth *authmocks.Plugin
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
	nm.auth.AssertExpectations(t)
}

func newTestNamespaceManager(resetConfig bool) *testNamespaceManager {
	if resetConfig {
		coreconfig.Reset()
		InitConfig(true)
		namespaceConfig.AddKnownKey("predefined.0.multiparty.enabled", true)
	}
	nm := &testNamespaceManager{
		mmi:  &metricsmocks.Manager{},
		mae:  &spieventsmocks.Manager{},
		mbi:  &blockchainmocks.Plugin{},
		cmi:  &cachemocks.Manager{},
		mdi:  &databasemocks.Plugin{},
		mdx:  &dataexchangemocks.Plugin{},
		mps:  &sharedstoragemocks.Plugin{},
		mti:  &tokenmocks.Plugin{},
		mev:  &eventsmocks.Plugin{},
		auth: &authmocks.Plugin{},
		namespaceManager: namespaceManager{
			reset:               make(chan bool, 1),
			namespaces:          make(map[string]*namespace),
			pluginNames:         make(map[string]bool),
			tokenBroadcastNames: make(map[string]string),
		},
	}
	nm.plugins.blockchain = map[string]blockchainPlugin{
		"ethereum": {plugin: nm.mbi},
	}
	nm.plugins.database = map[string]databasePlugin{
		"postgres": {plugin: nm.mdi},
	}
	nm.plugins.dataexchange = map[string]dataExchangePlugin{
		"ffdx": {plugin: nm.mdx},
	}
	nm.plugins.sharedstorage = map[string]sharedStoragePlugin{
		"ipfs": {plugin: nm.mps},
	}
	nm.plugins.identity = map[string]identityPlugin{
		"tbd": {plugin: &identitymocks.Plugin{}},
	}
	nm.plugins.tokens = map[string]tokensPlugin{
		"erc721":  {plugin: nm.mti},
		"erc1155": {plugin: nm.mti},
	}
	nm.plugins.events = map[string]eventsPlugin{
		"websockets": {plugin: nm.mev},
	}
	nm.plugins.auth = map[string]authPlugin{
		"basicauth": {plugin: nm.auth},
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
	nm.metricsEnabled = true

	mo := &orchestratormocks.Orchestrator{}
	mo.On("Init", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			nm.namespaces["default"].Contracts.Active = &core.MultipartyContract{
				Info: core.MultipartyContractInfo{
					Version: 2,
				},
			}
		}).
		Return(nil)
	nm.utOrchestrator = mo

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
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.NoError(t, err)

	assert.Equal(t, mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))
	assert.Nil(t, nm.Orchestrator(core.LegacySystemNamespace))

	mo.AssertExpectations(t)
}

func TestInitDatabaseFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	nm.utOrchestrator = &orchestratormocks.Orchestrator{}

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.EqualError(t, err, "pop")
}

func TestInitBlockchainFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	nm.utOrchestrator = &orchestratormocks.Orchestrator{}

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(fmt.Errorf("pop"))

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.EqualError(t, err, "pop")
}

func TestInitDataExchangeFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	nm.utOrchestrator = &orchestratormocks.Orchestrator{}

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.EqualError(t, err, "pop")
}

func TestInitSharedStorageFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	nm.utOrchestrator = &orchestratormocks.Orchestrator{}

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.EqualError(t, err, "pop")
}

func TestInitTokensFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	nm.utOrchestrator = &orchestratormocks.Orchestrator{}

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, mock.Anything, nil).Return(fmt.Errorf("pop"))

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.EqualError(t, err, "pop")
}

func TestInitEventsFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	nm.utOrchestrator = &orchestratormocks.Orchestrator{}

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.EqualError(t, err, "pop")
}

func TestInitAuthFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	nm.utOrchestrator = &orchestratormocks.Orchestrator{}

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.EqualError(t, err, "pop")
}

func TestInitOrchestratorFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	nm.mdi.On("Capabilities").Return(&database.Capabilities{
		Concurrency: true,
	})
	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mbi.On("GetAndConvertDeprecatedContractConfig", mock.Anything).Return(nil, "", fmt.Errorf("pop"))
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "pop", err)
}

func TestInitVersion1(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	mo := &orchestratormocks.Orchestrator{}
	mo.On("Init", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			nm.namespaces["default"].Contracts.Active = &core.MultipartyContract{
				Location: fftypes.JSONAnyPtr("{}"),
				Info: core.MultipartyContractInfo{
					Version: 1,
				},
			}
		}).
		Return(nil).Once()
	mo.On("Init", mock.Anything, mock.Anything).Return(nil).Once()
	nm.utOrchestrator = mo

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, core.LegacySystemNamespace).Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.NoError(t, err)

	assert.Equal(t, mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))
	assert.NotNil(t, nm.Orchestrator(core.LegacySystemNamespace))

	mo.AssertExpectations(t)
}

func TestInitFFSystemWithTerminatedV1Contract(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	nm.metricsEnabled = true

	mo := &orchestratormocks.Orchestrator{}
	mo.On("Init", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
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
		Return(nil)
	nm.utOrchestrator = mo

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, core.LegacySystemNamespace).Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.NoError(t, err)

	assert.Equal(t, mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))

	mo.AssertExpectations(t)
}

func TestLegacyNamespaceConflictingPlugins(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
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

	mo := &orchestratormocks.Orchestrator{}
	mo.On("Init", mock.Anything, mock.Anything).
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
	nm.utOrchestrator = mo

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, "ns2").Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err = nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "FF10421", err)

	assert.Equal(t, mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))

	mo.AssertExpectations(t)
}

func TestLegacyNamespaceConflictingPluginsTooManyPlugins(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
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

	mo := &orchestratormocks.Orchestrator{}
	mo.On("Init", mock.Anything, mock.Anything).
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
	nm.utOrchestrator = mo

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, "ns2").Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err = nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "FF10421", err)

	assert.Equal(t, mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))

	mo.AssertExpectations(t)
}

func TestLegacyNamespaceMatchingPlugins(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
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

	mo := &orchestratormocks.Orchestrator{}
	mo.On("Init", mock.Anything, mock.Anything).
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
	nm.utOrchestrator = mo

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, "ns2").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, core.LegacySystemNamespace).Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err = nm.Init(ctx, cancelCtx, nm.reset)
	assert.NoError(t, err)

	assert.Equal(t, mo, nm.Orchestrator("default"))
	assert.Nil(t, nm.Orchestrator("unknown"))

	mo.AssertExpectations(t)
}

func TestInitVersion1Fail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	mo := &orchestratormocks.Orchestrator{}
	mo.On("Init", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			nm.namespaces["default"].Contracts.Active = &core.MultipartyContract{
				Location: fftypes.JSONAnyPtr("{}"),
				Info: core.MultipartyContractInfo{
					Version: 1,
				},
			}
		}).
		Return(nil).Once()
	mo.On("Init", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Once()
	nm.utOrchestrator = mo

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, core.LegacySystemNamespace).Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.EqualError(t, err, "pop")

	mo.AssertExpectations(t)
}

func TestInitNamespaceQueryFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	ns := &namespace{
		Namespace: core.Namespace{
			Name: "default",
		},
		plugins: []string{"postgres"},
	}

	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(nil, fmt.Errorf("pop"))

	err := nm.initNamespace(context.Background(), ns)
	assert.EqualError(t, err, "pop")
}

func TestInitNamespaceExistingUpsertFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	ns := &namespace{
		Namespace: core.Namespace{
			Name: "default",
		},
		plugins: []string{"postgres"},
	}
	existing := &core.Namespace{
		NetworkName: "ns1",
	}

	nm.mdi.On("GetNamespace", mock.Anything, "default").Return(existing, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(fmt.Errorf("pop"))

	err := nm.initNamespace(context.Background(), ns)
	assert.EqualError(t, err, "pop")
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
	nm.plugins.database = nil
	difactory.InitConfig(databaseConfig)
	config.Set("plugins.database", []fftypes.JSONObject{{}})
	databaseConfig.AddKnownKey(coreconfig.PluginConfigName, "wrong////")
	databaseConfig.AddKnownKey(coreconfig.PluginConfigType, "postgres")

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
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
	nm.plugins.identity = nil
	iifactory.InitConfig(identityConfig)
	identityConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	config.Set("plugins.identity", []fftypes.JSONObject{{}})

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
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
	nm.plugins.blockchain = nil
	bifactory.InitConfig(blockchainConfig)
	config.Set("plugins.blockchain", []fftypes.JSONObject{{}})
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	blockchainConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong//")

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
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
	nm.plugins.sharedstorage = nil
	ssfactory.InitConfig(sharedstorageConfig)
	config.Set("plugins.sharedstorage", []fftypes.JSONObject{{}})
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	sharedstorageConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong//")

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
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
	nm.plugins.dataexchange = nil
	dxfactory.InitConfig(dataexchangeConfig)
	config.Set("plugins.dataexchange", []fftypes.JSONObject{{}})
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigName, "flapflip")
	dataexchangeConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong//")

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
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

func TestTokensPluginDuplicateBroadcastName(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
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

	plugins, err := nm.getTokensPlugins(context.Background())
	assert.Nil(t, plugins)
	assert.Regexp(t, "FF10419", err)
}

func TestMultipleTokensPluginsWithBroadcastName(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
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

	plugins, err := nm.getTokensPlugins(context.Background())
	assert.Equal(t, 2, len(plugins))
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
	nm.plugins.tokens = nil
	tifactory.InitConfig(tokensConfig)
	config.Set("plugins.tokens", []fftypes.JSONObject{{}})
	tokensConfig.AddKnownKey(coreconfig.PluginConfigName, "erc20_erc721")
	tokensConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
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

func TestEventsPluginDefaults(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	nm.plugins.events = nil
	plugins, err := nm.getEventPlugins(context.Background())
	assert.Equal(t, 3, len(plugins))
	assert.NoError(t, err)
}

func TestAuthPlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	authfactory.InitConfigArray(authConfig)
	config.Set("plugins.auth", []fftypes.JSONObject{{}})
	authConfig.AddKnownKey(coreconfig.PluginConfigName, "basicauth")
	authConfig.AddKnownKey(coreconfig.PluginConfigType, "basic")
	plugins, err := nm.getAuthPlugin(context.Background())
	assert.Equal(t, 1, len(plugins))
	assert.NoError(t, err)
}

func TestAuthPluginBadType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	nm.plugins.auth = nil
	authfactory.InitConfigArray(authConfig)
	config.Set("plugins.auth", []fftypes.JSONObject{{}})
	authConfig.AddKnownKey(coreconfig.PluginConfigName, "basicauth")
	authConfig.AddKnownKey(coreconfig.PluginConfigType, "wrong")

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.Error(t, err)
}

func TestAuthPluginInvalid(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	authfactory.InitConfigArray(authConfig)
	config.Set("plugins.auth", []fftypes.JSONObject{{}})
	authConfig.AddKnownKey(coreconfig.PluginConfigName, "bad name not allowed")
	authConfig.AddKnownKey(coreconfig.PluginConfigType, "basic")
	plugins, err := nm.getAuthPlugin(context.Background())
	assert.Equal(t, 0, len(plugins))
	assert.Error(t, err)
}

func TestEventsPluginBadType(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
	nm.plugins.events = nil
	config.Set(coreconfig.EventTransportsEnabled, []string{"!unknown!"})

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := nm.Init(ctx, cancelCtx, nm.reset)
	assert.Error(t, err)
}

func TestInitBadNamespace(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	nm.utOrchestrator = &orchestratormocks.Orchestrator{}

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

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

func TestLoadNamespacesReservedNetworkName(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

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

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10388", err)
}

func TestLoadNamespacesDuplicate(t *testing.T) {
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

func TestLoadNamespacesNoName(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    predefined:
    - plugins: [postgres]
    `))
	assert.NoError(t, err)

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10166", err)
}

func TestLoadNamespacesNoDefault(t *testing.T) {
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

	err = nm.loadNamespaces(context.Background())
	assert.Regexp(t, "FF10166", err)
}

func TestLoadNamespacesUseDefaults(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

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

	err = nm.loadNamespaces(context.Background())
	assert.NoError(t, err)
	assert.Len(t, nm.namespaces, 1)
	assert.Equal(t, "oldest", nm.namespaces["ns1"].config.Multiparty.Contracts[0].FirstEvent)
}

func TestLoadNamespacesNonMultipartyNoDatabase(t *testing.T) {
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

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err = nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "FF10392", err)
}

func TestLoadNamespacesMultipartyUnknownPlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

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

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err = nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "FF10390.*unknown", err)
}

func TestLoadNamespacesMultipartyMultipleBlockchains(t *testing.T) {
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

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err = nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "FF10394.*blockchain", err)
}

func TestLoadNamespacesMultipartyMultipleDX(t *testing.T) {
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

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err = nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "FF10394.*dataexchange", err)
}

func TestLoadNamespacesMultipartyMultipleSS(t *testing.T) {
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

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err = nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "FF10394.*sharedstorage", err)
}

func TestLoadNamespacesMultipartyMultipleDB(t *testing.T) {
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

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err = nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "FF10394.*database", err)
}

func TestInitNamespacesMultipartyWithAuth(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

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

	err = nm.loadNamespaces(context.Background())
	assert.NoError(t, err)
}

func TestLoadNamespacesNonMultipartyWithAuth(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

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

	err = nm.loadNamespaces(context.Background())
	assert.NoError(t, err)
}

func TestLoadNamespacesMultipartyContract(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

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

	err = nm.loadNamespaces(context.Background())
	assert.NoError(t, err)
}

func TestLoadNamespacesMultipartyContractBadLocation(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	fail := make(chan string)
	namespaceConfig.AddKnownKey("predefined.0.multiparty.contract.0.location.address", fail)

	err := nm.loadNamespaces(context.Background())
	assert.Regexp(t, "json:", err)
}

func TestLoadNamespacesNonMultipartyMultipleDB(t *testing.T) {
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

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err = nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "FF10394.*database", err)
}

func TestLoadNamespacesNonMultipartyMultipleBlockchains(t *testing.T) {
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

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err = nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "FF10394.*blockchain", err)
}

func TestLoadNamespacesMultipartyMissingPlugins(t *testing.T) {
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

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err = nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "FF10391", err)
}

func TestLoadNamespacesNonMultipartyUnknownPlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      plugins: [basicauth, erc721, bad]
  `))
	assert.NoError(t, err)

	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc721", nil).Return(nil)
	nm.mti.On("Init", mock.Anything, mock.Anything, "erc1155", nil).Return(nil)
	nm.mev.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.auth.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err = nm.Init(ctx, cancelCtx, nm.reset)
	assert.Regexp(t, "FF10390.*unknown", err)
}

func TestLoadNamespacesNonMultipartyTokens(t *testing.T) {
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
	nm.plugins.blockchain = nil
	nm.plugins.dataexchange = nil
	nm.plugins.tokens = nil
	nm.metricsEnabled = true

	mo.On("Start", mock.Anything).Return(nil)

	err := nm.Start()
	assert.NoError(t, err)

	mo.AssertExpectations(t)
}

func TestStartBlockchainFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	mo := &orchestratormocks.Orchestrator{}
	nm.namespaces = map[string]*namespace{
		"ns": {
			orchestrator: mo,
		},
	}

	mo.On("Start", mock.Anything).Return(nil)
	nm.mbi.On("Start").Return(fmt.Errorf("pop"))

	err := nm.Start()
	assert.EqualError(t, err, "pop")

	mo.AssertExpectations(t)

}

func TestStartDataExchangeFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	mo := &orchestratormocks.Orchestrator{}
	nm.namespaces = map[string]*namespace{
		"ns": {
			orchestrator: mo,
		},
	}
	nm.plugins.blockchain = nil

	mo.On("Start", mock.Anything).Return(nil)
	nm.mdx.On("Start").Return(fmt.Errorf("pop"))

	err := nm.Start()
	assert.EqualError(t, err, "pop")

	mo.AssertExpectations(t)

}

func TestStartTokensFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	mo := &orchestratormocks.Orchestrator{}
	nm.namespaces = map[string]*namespace{
		"ns": {
			orchestrator: mo,
		},
	}
	nm.plugins.blockchain = nil
	nm.plugins.dataexchange = nil

	mo.On("Start", mock.Anything).Return(nil)
	nm.mti.On("Start").Return(fmt.Errorf("pop"))

	err := nm.Start()
	assert.EqualError(t, err, "pop")

}

func TestStartOrchestratorFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	mo := &orchestratormocks.Orchestrator{}
	nm.namespaces = map[string]*namespace{
		"ns": {orchestrator: mo},
	}

	mo.On("Start", mock.Anything).Return(fmt.Errorf("pop"))

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
	mae := nm.adminEvents.(*spieventsmocks.Manager)

	mo.On("WaitStop").Return()
	mae.On("WaitStop").Return()

	nm.WaitStop()

	mo.AssertExpectations(t)
	mae.AssertExpectations(t)
}

func TestReset(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	childCtx, childCancel := context.WithCancel(context.Background())
	nm.Reset(childCtx)
	childCancel()

	assert.True(t, <-nm.namespaceManager.reset)
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
	assert.NotNil(t, nm.SPIEvents())
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

func TestGetOperationByNamespacedID(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

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
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	mo := &orchestratormocks.Orchestrator{}
	nm.namespaces = map[string]*namespace{
		"default": {orchestrator: mo},
	}

	_, err := nm.GetOperationByNamespacedID(context.Background(), "default:bad")
	assert.Regexp(t, "FF00138", err)

	mo.AssertExpectations(t)
}

func TestGetOperationByNamespacedIDNoOrchestrator(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

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
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

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
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	mo := &orchestratormocks.Orchestrator{}
	nm.namespaces = map[string]*namespace{
		"default": {orchestrator: mo},
	}

	err := nm.ResolveOperationByNamespacedID(context.Background(), "default:bad", &core.OperationUpdateDTO{})
	assert.Regexp(t, "FF00138", err)

	mo.AssertExpectations(t)
}

func TestResolveOperationByNamespacedIDNoOrchestrator(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

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
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)
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

	_, err = nm.validateNonMultipartyConfig(context.Background(), "ns1", []string{"postgres", "erc721"})
	assert.NoError(t, err)
}
