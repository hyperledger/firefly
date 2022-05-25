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
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testNamespaceManager struct {
	namespaceManager
	mdi *databasemocks.Plugin
	mbi *blockchainmocks.Plugin
	mps *sharedstoragemocks.Plugin
	mdx *dataexchangemocks.Plugin
	mti *tokenmocks.Plugin
}

func (nm *testNamespaceManager) cleanup(t *testing.T) {
	nm.mdi.AssertExpectations(t)
}

func newTestNamespaceManager(resetConfig bool) *testNamespaceManager {
	if resetConfig {
		coreconfig.Reset()
		InitConfig(true)
	}
	nm := &testNamespaceManager{
		mdi: &databasemocks.Plugin{},
		namespaceManager: namespaceManager{
			nsConfig:      buildNamespaceMap(context.Background()),
			bcPlugins:     map[string]blockchain.Plugin{"ethereum": &blockchainmocks.Plugin{}, "fabric": &blockchainmocks.Plugin{}},
			dbPlugins:     map[string]database.Plugin{"postgres": &databasemocks.Plugin{}, "sqlite3": &databasemocks.Plugin{}},
			dxPlugins:     map[string]dataexchange.Plugin{"ffdx": &dataexchangemocks.Plugin{}, "ffdx2": &dataexchangemocks.Plugin{}},
			ssPlugins:     map[string]sharedstorage.Plugin{"ipfs": &sharedstoragemocks.Plugin{}, "ipfs2": &sharedstoragemocks.Plugin{}},
			tokensPlugins: map[string]tokens.Plugin{"erc721": &tokenmocks.Plugin{}},
		},
	}
	return nm
}

func TestNewNamespaceManager(t *testing.T) {
	bc := map[string]blockchain.Plugin{"ethereum": &blockchainmocks.Plugin{}}
	db := map[string]database.Plugin{"postgres": &databasemocks.Plugin{}}
	dx := map[string]dataexchange.Plugin{"ffdx": &dataexchangemocks.Plugin{}}
	ss := map[string]sharedstorage.Plugin{"ipfs": &sharedstoragemocks.Plugin{}}
	tokens := map[string]tokens.Plugin{"erc721": &tokenmocks.Plugin{}}

	nm := NewNamespaceManager(context.Background(), bc, db, dx, ss, tokens)
	assert.NotNil(t, nm)
}

func TestInit(t *testing.T) {
	coreconfig.Reset()
	nm := newTestNamespaceManager(false)
	defer nm.cleanup(t)

	nm.Init(context.Background(), nm.mdi)
}

func TestInitNamespacesBadName(t *testing.T) {
	coreconfig.Reset()
	namespaceConfig.AddKnownKey("predefined.0."+coreconfig.NamespaceName, "!Badness")
	nm := newTestNamespaceManager(false)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("test")

	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "!Badness")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "test")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "multiparty")
	utTestConfig.AddKnownKey(coreconfig.NamespaceDescription, "test description")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "postgres", "ffdx", "ipfs", "erc721"})
	nm.nsConfig = map[string]config.Section{"!Badness": utTestConfig}

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF00140", err)
}

func TestInitNamespacesReservedName(t *testing.T) {
	coreconfig.Reset()
	nm := newTestNamespaceManager(false)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("test")

	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "ff_system")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "test")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "multiparty")
	utTestConfig.AddKnownKey(coreconfig.NamespaceDescription, "test description")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "postgres", "ffdx", "ipfs", "erc721"})
	nm.nsConfig = map[string]config.Section{"ff_system": utTestConfig}

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10388", err)
}

func TestInitNamespacesGetFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "multiparty")
	utTestConfig.AddKnownKey(coreconfig.NamespaceDescription, "test description")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "postgres", "ffdx", "ipfs", "erc721"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	nm.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "pop", err)
}

func TestInitNamespacesUpsertFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "multiparty")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "postgres", "ffdx", "ipfs", "erc721"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	nm.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.Anything, true).Return(fmt.Errorf("pop"))
	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "pop", err)
}

func TestInitNamespacesMultipartyUnknownPlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "multiparty")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "postgres", "ffdx", "ipfs", "erc721", "bad_unknown_plugin"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10390.*unknown", err)
}

func TestInitNamespacesMultipartyMultipleBlockchains(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "multiparty")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "postgres", "ffdx", "ipfs", "erc721", "fabric"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10394", err)
}

func TestInitNamespacesMultipartyMultipleDX(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "multiparty")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "postgres", "ffdx", "ipfs", "erc721", "ffdx2"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10394", err)
}

func TestInitNamespacesMultipartyMultipleSS(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "multiparty")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "postgres", "ffdx", "ipfs", "erc721", "ipfs2"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10394", err)
}

func TestInitNamespacesMultipartyMultipleDB(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "multiparty")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "postgres", "ffdx", "ipfs", "erc721", "sqlite3"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10394", err)
}

func TestInitNamespacesDeprecatedConfigMultipleBlockchains(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "multiparty")
	utTestConfig.AddKnownKey(coreconfig.NamespaceDescription)
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	// nm.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, nil)
	// nm.mdi.On("UpsertNamespace", mock.Anything, mock.Anything, true).Return(nil)

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10394", err)
}

func TestInitNamespacesGatewayMultipleDB(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "gateway")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "postgres", "sqlite3"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10394", err)
}

func TestInitNamespacesGatewayMultipleBlockchains(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "gateway")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "postgres", "fabric"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10394", err)
}

func TestInitNamespacesMultipartyMissingPlugins(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "multiparty")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "postgres"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10391", err)
}

func TestInitNamespacesGatewayWithDX(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "gateway")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "ffdx"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10393", err)
}

func TestInitNamespacesGatewayWithSharedStorage(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "gateway")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "ipfs"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10393", err)
}

func TestInitNamespacesGatewayUnknownPlugin(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "gateway")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "bad_unknown_plugin"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10390.*unknown", err)
}

func TestInitNamespacesGatewayNoDB(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "gateway")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10392", err)
}

func TestInitNamespacesUnknownMode(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "not a real mode")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "postgres", "ffdx", "ipfs", "erc721"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10389", err)
}

func TestInitNamespacesUpsertNotNeeded(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	utTestConfig := config.RootSection("default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceRemoteName, "default")
	utTestConfig.AddKnownKey(coreconfig.NamespaceMode, "multiparty")
	utTestConfig.AddKnownKey(coreconfig.NamespaceDescription, "test description")
	utTestConfig.AddKnownKey(coreconfig.NamespacePlugins, []string{"ethereum", "postgres", "ffdx", "ipfs", "erc721"})
	nm.nsConfig = map[string]config.Section{"default": utTestConfig}

	nm.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(&core.Namespace{
		Type: core.NamespaceTypeBroadcast, // any broadcasted NS will not be updated
	}, nil)
	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.NoError(t, err)
}

func TestInitNamespacesDefaultMissing(t *testing.T) {
	coreconfig.Reset()
	config.Set(coreconfig.NamespacesPredefined, fftypes.JSONObjectArray{})

	nm := newTestNamespaceManager(false)
	defer nm.cleanup(t)

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10166", err)
}

func TestInitNamespacesDupName(t *testing.T) {
	coreconfig.Reset()
	InitConfig(false)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
  namespaces:
    default: ns1
    predefined:
    - name: ns1
      remoteName: ns1
      mode: gateway
      org:
        key: 0x123456
      plugins:
      - sqlite3
      - ethereum
      - erc721
    - name: ns2
      remoteName: ns2
      mode: gateway
      org:
        key: 0x223456
      plugins:
      - sqlite3
      - ethereum
      - erc721
    - name: ns2
      remoteName: ns2
      mode: gateway
      org:
        key: 0x223456
      plugins:
      - sqlite3
      - ethereum
      - erc721
  `))
	assert.NoError(t, err)

	nm := newTestNamespaceManager(false)
	defer nm.cleanup(t)

	nsList, err := nm.getPredefinedNamespaces(context.Background())
	assert.NoError(t, err)
	assert.Len(t, nsList, 3)
	names := make([]string, len(nsList))
	for i, ns := range nsList {
		names[i] = ns.Name
	}
	assert.Contains(t, names, core.SystemNamespace)
	assert.Contains(t, names, "ns1")
	assert.Contains(t, names, "ns2")
}
