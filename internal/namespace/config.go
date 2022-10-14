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
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/pkg/core"
)

const (
	// NamespacePredefined is the list of pre-defined namespaces
	NamespacePredefined         = "predefined"
	NamespaceMultipartyContract = "contract"
)

var (
	namespaceConfig     = config.RootSection("namespaces")
	namespacePredefined = namespaceConfig.SubArray(NamespacePredefined)
)

func InitConfig(withDefaults bool) {
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

	if withDefaults {
		namespaceConfig.AddKnownKey(NamespacePredefined+".0."+coreconfig.NamespaceName, "default")
		namespaceConfig.AddKnownKey(NamespacePredefined+".0."+coreconfig.NamespaceDescription, "Default predefined namespace")
		namespaceConfig.AddKnownKey(NamespacePredefined+".0."+coreconfig.NamespaceAssetKeyNormalization, "blockchain_plugin")
	}
}
