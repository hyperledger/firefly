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

package namespace

import (
	"github.com/hyperledger/firefly-common/pkg/auth/authfactory"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly/internal/blockchain/bifactory"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/database/difactory"
	"github.com/hyperledger/firefly/internal/dataexchange/dxfactory"
	"github.com/hyperledger/firefly/internal/events/eifactory"
	"github.com/hyperledger/firefly/internal/identity/iifactory"
	"github.com/hyperledger/firefly/internal/sharedstorage/ssfactory"
	"github.com/hyperledger/firefly/internal/tokens/tifactory"
	"github.com/hyperledger/firefly/pkg/core"
)

const (
	// NamespacePredefined is the list of pre-defined namespaces
	NamespacePredefined         = "predefined"
	NamespaceMultipartyContract = "contract"
)

var (
	namespaceConfigSection = config.RootSection("namespaces")
	namespacePredefined    = namespaceConfigSection.SubArray(NamespacePredefined)

	blockchainConfig    = config.RootArray("plugins.blockchain")
	tokensConfig        = config.RootArray("plugins.tokens")
	databaseConfig      = config.RootArray("plugins.database")
	sharedstorageConfig = config.RootArray("plugins.sharedstorage")
	dataexchangeConfig  = config.RootArray("plugins.dataexchange")
	identityConfig      = config.RootArray("plugins.identity")
	authConfig          = config.RootArray("plugins.auth")
	eventsConfig        = config.RootSection("events") // still at root

	// Deprecated configs
	deprecatedTokensConfig        = config.RootArray("tokens")
	deprecatedBlockchainConfig    = config.RootSection("blockchain")
	deprecatedDatabaseConfig      = config.RootSection("database")
	deprecatedSharedStorageConfig = config.RootSection("sharedstorage")
	deprecatedDataexchangeConfig  = config.RootSection("dataexchange")
)

func InitConfig() {
	namespacePredefined.AddKnownKey(coreconfig.NamespaceName)
	namespacePredefined.AddKnownKey(coreconfig.NamespaceDescription)
	namespacePredefined.AddKnownKey(coreconfig.NamespacePlugins)
	namespacePredefined.AddKnownKey(coreconfig.NamespaceDefaultKey)
	namespacePredefined.AddKnownKey(coreconfig.NamespaceAssetKeyNormalization)

	multipartyConf := namespacePredefined.SubSection(coreconfig.NamespaceMultiparty)
	multipartyConf.AddKnownKey(coreconfig.NamespaceMultipartyEnabled)
	multipartyConf.AddKnownKey(coreconfig.NamespaceMultipartyNetworkNamespace)
	multipartyConf.AddKnownKey(coreconfig.NamespaceMultipartyOrgName)
	multipartyConf.AddKnownKey(coreconfig.NamespaceMultipartyOrgDescription)
	multipartyConf.AddKnownKey(coreconfig.NamespaceMultipartyOrgKey)
	multipartyConf.AddKnownKey(coreconfig.NamespaceMultipartyNodeName)
	multipartyConf.AddKnownKey(coreconfig.NamespaceMultipartyNodeDescription)

	contractConf := multipartyConf.SubArray(coreconfig.NamespaceMultipartyContract)
	contractConf.AddKnownKey(coreconfig.NamespaceMultipartyContractFirstEvent, string(core.SubOptsFirstEventOldest))
	contractConf.AddKnownKey(coreconfig.NamespaceMultipartyContractLocation)
	contractConf.AddKnownKey(coreconfig.NamespaceMultipartyContractOptions)

	bifactory.InitConfigDeprecated(deprecatedBlockchainConfig)
	bifactory.InitConfig(blockchainConfig)
	difactory.InitConfigDeprecated(deprecatedDatabaseConfig)
	difactory.InitConfig(databaseConfig)
	ssfactory.InitConfigDeprecated(deprecatedSharedStorageConfig)
	ssfactory.InitConfig(sharedstorageConfig)
	dxfactory.InitConfig(dataexchangeConfig)
	dxfactory.InitConfigDeprecated(deprecatedDataexchangeConfig)
	iifactory.InitConfig(identityConfig)
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	tifactory.InitConfig(tokensConfig)
	authfactory.InitConfigArray(authConfig)
	eifactory.InitConfig(eventsConfig)
}
