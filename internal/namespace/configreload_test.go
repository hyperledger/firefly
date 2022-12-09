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
	"strings"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var exampleConfig1base = `
---
http:
  address: 0.0.0.0
  port: 5000
  publicURL: https://myfirefly.example.com
log:
  level: debug
metrics:
  enabled: true
  address: 127.0.0.1
  port: 6000
  path: /metrics
namespaces:
  default: ns1
  predefined:
    - defaultKey: 0xbEa50Ec98776beF144Fc63078e7b15291Ac64cfA
      name: ns1
      plugins:
        - sharedstorage-ns1
        - database0
        - blockchain-ns1
        - ff-dx
        - erc1155-ns1
        - erc20_erc721-ns1
        - test_user_auth-ns1
      multiparty:
        enabled: true
        node:
          name: node1
        org:
          name: org1
          key: 0xbEa50Ec98776beF144Fc63078e7b15291Ac64cfA
        contract:
          - firstevent: "0"
            location:
              address: 0x7359d2ecc199C48369b390522c29b77A5Af30882
    - defaultKey: 0x630659A26fa005d50Fa9706D8a4e242fd4169A61
      name: ns2
      plugins:
        - database0
        - blockchain-ns2
        - test_user_auth-ns2
      multiparty: {}
node: {}
plugins:
  blockchain:
    - ethereum:
        ethconnect:
          topic: "0"
          url: https://ethconnect1.example.com:5000
      name: blockchain-ns1
      type: ethereum
    - ethereum:
        ethconnect:
          topic: "0"
          url: https://ethconnect2.example.com:5000
      name: blockchain-ns2
      type: ethereum
  database:
    - name: database0
      postgres:
        url: postgres://postgrs.example.com:5432/firefly?sslmode=require
      type: postgres
  dataexchange:
    - ffdx:
        url: https://ffdx:3000
        initEnabled: true
      name: ff-dx
      type: ffdx
  sharedstorage:
    - ipfs:
        api:
          url: https://ipfs1.example.com
          auth:
            username: someuser
            password: somepass
        gateway:
          url: https://ipfs1.example.com
          auth:
          username: someuser
          password: somepass
      name: sharedstorage-ns1
      type: ipfs
    - ipfs:
        api:
          url: https://ipfs2.example.com
        gateway:
          url: https://ipfs2.example.com
      name: sharedstorage-ns2
      type: ipfs
  tokens:
    - fftokens:
        url: https://ff-tokens-ns1-erc1155:3000
      name: erc1155-ns1
      type: fftokens
    - fftokens:
        url: https://ff-tokens-ns1-erc20-erc721:3000
      name: erc20_erc721-ns1
      type: fftokens
    - fftokens:
        url: https://ff-tokens-ns2-erc20-erc721:3000
      name: erc20_erc721-ns2
      type: fftokens
  auth:
    - name: test_user_auth-ns1
      type: basic
      basic:
        passwordfile: /etc/firefly/test_users
    - name: test_user_auth-ns2
      type: basic
      basic:
        passwordfile: /etc/firefly/test_users              
ui:
  path: ./frontend
`

// here we deliberately make a bunch of ordering changes in fields,
// but the only actual difference is the creation of a third namespace,
// and some NEW plugins
var exampleConfig2extraNS = `
---
http:
  address: 0.0.0.0
  port: 5000
  publicURL: https://myfirefly.example.com
ui:
  path: ./frontend  
namespaces:
  default: ns1
  predefined:
    - defaultKey: 0xbEa50Ec98776beF144Fc63078e7b15291Ac64cfA
      name: ns1
      plugins:
        - sharedstorage-ns1
        - database0
        - blockchain-ns1
        - ff-dx
        - erc1155-ns1
        - erc20_erc721-ns1
        - test_user_auth-ns1
      multiparty:
        enabled: true
        node:
          name: node1
        org:
          name: org1
          key: 0xbEa50Ec98776beF144Fc63078e7b15291Ac64cfA
        contract:
          - firstevent: "0"
            location:
              address: 0x7359d2ecc199C48369b390522c29b77A5Af30882
    - defaultKey: 0x630659A26fa005d50Fa9706D8a4e242fd4169A61
      plugins:
        - database0
        - blockchain-ns2
        - test_user_auth-ns2
      name: ns2
      multiparty: {}
    - defaultKey: 0xF49C223038FA129c2Ba23D5c5f3Cdb50120F3EDe
      name: ns3
      plugins:
        - database0
        - blockchain-ns3
        - test_user_auth-ns3
      multiparty: {}      
node: {}
log:
  level: debug
metrics:
  enabled: true
  address: 127.0.0.1
  port: 6000
  path: /metrics
plugins:
  blockchain:
    - ethereum:
        ethconnect:
          topic: "0"
          url: https://ethconnect1.example.com:5000
      name: blockchain-ns1
      type: ethereum
    - ethereum:
        ethconnect:
          topic: "0"
          url: https://ethconnect2.example.com:5000
      name: blockchain-ns2
      type: ethereum
    - ethereum:
      ethconnect:
        topic: "0"
        url: https://ethconnect3.example.com:5000
      name: blockchain-ns3
      type: ethereum
  database:
    - name: database0
      type: postgres
      postgres:
        url: postgres://postgrs.example.com:5432/firefly?sslmode=require
  dataexchange:
    - ffdx:
        url: https://ffdx:3000
        initEnabled: true
      name: ff-dx
      type: ffdx
  sharedstorage:
    - ipfs:
        gateway:
          url: https://ipfs1.example.com
          auth:
          username: someuser
          password: somepass
        api:
          url: https://ipfs1.example.com
          auth:
            username: someuser
            password: somepass
      type: ipfs
      name: sharedstorage-ns1
    - ipfs:
        api:
          url: https://ipfs2.example.com
        gateway:
          url: https://ipfs2.example.com
      name: sharedstorage-ns2
      type: ipfs
  tokens:
    - fftokens:
        url: https://ff-tokens-ns1-erc1155:3000
      type: fftokens
      name: erc1155-ns1
    - fftokens:
        url: https://ff-tokens-ns1-erc20-erc721:3000
      type: fftokens
      name: erc20_erc721-ns1
    - fftokens:
        url: https://ff-tokens-ns2-erc20-erc721:3000
      type: fftokens
      name: erc20_erc721-ns2
  auth:
    - name: test_user_auth-ns1
      basic:
        passwordfile: /etc/firefly/test_users
      type: basic
    - name: test_user_auth-ns2
      basic:
        passwordfile: /etc/firefly/test_users
      type: basic
    - name: test_user_auth-ns3
      type: basic
      basic:
        passwordfile: /etc/firefly/test_users
`

func mockInitConfig(nm *testNamespaceManager) {
	nm.mdi.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("SetHandler", database.GlobalHandler, mock.Anything).Return()
	nm.mbi.On("Init", mock.Anything, mock.Anything, mock.Anything, nm.mmi, mock.Anything).Return(nil)
	nm.mdx.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mps.On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mti[1].On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mei[0].On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mei[1].On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mei[2].On("Init", mock.Anything, mock.Anything).Return(nil)
	nm.mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, "ns2").Return(nil, nil)
	nm.mdi.On("GetNamespace", mock.Anything, "ns3").Return(nil, nil).Maybe()
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.AnythingOfType("*core.Namespace"), true).Return(nil)
	nm.mai.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mbi.On("Start").Return(nil)
	nm.mdx.On("Start").Return(nil)
	nm.mti[1].On("Start").Return(nil)

	nm.mo.On("Init", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			if nm.namespaces["ns1"] != nil && nm.namespaces["ns1"].Contracts != nil {
				nm.namespaces["ns1"].Contracts.Active = &core.MultipartyContract{
					Info: core.MultipartyContractInfo{
						Version: 2,
					},
				}
			}
		}).
		Return(nil)
	nm.mo.On("Start").Return(nil)
}

func TestConfigReload1to2(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	nm, cleanup := newTestNamespaceManager(t, false)
	defer cleanup()

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(exampleConfig1base))
	assert.NoError(t, err)

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	mockInitConfig(nm)

	err = nm.Init(ctx, cancelCtx, make(chan bool))
	assert.NoError(t, err)

	err = nm.Start()
	assert.NoError(t, err)

	originalPlugins := nm.plugins
	originalPluginHashes := make(map[string]*fftypes.Bytes32)
	for k, v := range originalPlugins {
		originalPluginHashes[k] = v.configHash
	}
	originaNS := nm.namespaces
	originalNSHashes := make(map[string]*fftypes.Bytes32)
	for k, v := range originaNS {
		originalNSHashes[k] = v.configHash
	}

	coreconfig.Reset()
	initAllConfig()
	viper.SetConfigType("yaml")
	err = viper.ReadConfig(strings.NewReader(exampleConfig2extraNS))
	assert.NoError(t, err)

	// Drive the config reload
	nm.configReloaded(nm.ctx)

	// Check that we didn't cancel the context
	select {
	case <-nm.ctx.Done():
		assert.Fail(t, "Error occurred in config reload")
	default:
	}

	// Check none of the plugins reloaded
	for name := range originalPlugins {
		assert.True(t, originalPlugins[name] == nm.plugins[name], name)
		assert.Equal(t, originalPluginHashes[name], nm.plugins[name].configHash, name)
	}

	// Check we have two more than before
	assert.Len(t, nm.plugins, len(originalPlugins)+2)
	assert.NotNil(t, nm.plugins["blockchain-ns3"])
	assert.NotNil(t, nm.plugins["test_user_auth-ns3"])

	// Check none of the namespaces reloaded
	for name := range originaNS {
		assert.True(t, originaNS[name] == nm.namespaces[name], name)
		assert.Equal(t, originalNSHashes[name], nm.namespaces[name].configHash, name)
	}

	// Check we have one more than before
	assert.Len(t, nm.namespaces, len(originaNS)+1)
	assert.NotNil(t, nm.namespaces["ns3"])

}
