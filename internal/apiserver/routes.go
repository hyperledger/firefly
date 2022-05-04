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

package apiserver

import (
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/oapispec"
)

var routes = append(
	globalRoutes([]*oapispec.Route{
		getDIDDocByDID,
		getIdentityByDID,
		getNamespace,
		getNamespaces,
		getNetworkIdentities,
		getNetworkNode,
		getNetworkNodes,
		getNetworkOrg,
		getNetworkOrgs,
		getStatus,
		getStatusBatchManager,
		getStatusPins,
		getStatusWebSockets,
		postNewNamespace,
		postNewOrganization,
		postNewOrganizationSelf,
		postNodesSelf,
	}),
	namespacedRoutes([]*oapispec.Route{
		deleteContractListener,
		deleteSubscription,
		getBatchByID,
		getBatches,
		getBlockchainEventByID,
		getBlockchainEvents,
		getChartHistogram,
		getContractAPIByName,
		getContractAPIInterface,
		getContractAPIs,
		getContractAPIListeners,
		getContractInterface,
		getContractInterfaceNameVersion,
		getContractInterfaces,
		getContractListenerByNameOrID,
		getContractListeners,
		getData,
		getDataBlob,
		getDataByID,
		getDataMsgs,
		getDatatypeByName,
		getDatatypes,
		getEventByID,
		getEvents,
		getGroupByHash,
		getGroups,
		getIdentities,
		getIdentityByID,
		getIdentityDID,
		getIdentityVerifiers,
		getMsgByID,
		getMsgData,
		getMsgEvents,
		getMsgs,
		getMsgTxn,
		getOpByID,
		getOps,
		getSubscriptionByID,
		getSubscriptions,
		getTokenAccountPools,
		getTokenAccounts,
		getTokenApprovals,
		getTokenBalances,
		getTokenConnectors,
		getTokenPoolByNameOrID,
		getTokenPools,
		getTokenTransferByID,
		getTokenTransfers,
		getTxnBlockchainEvents,
		getTxnByID,
		getTxnOps,
		getTxns,
		getTxnStatus,
		getVerifierByID,
		getVerifiers,
		patchUpdateIdentity,
		postContractAPIInvoke,
		postContractAPIQuery,
		postContractAPIListeners,
		postContractInterfaceGenerate,
		postContractInvoke,
		postContractQuery,
		postData,
		postNewContractAPI,
		postNewContractInterface,
		postNewContractListener,
		postNewDatatype,
		postNewIdentity,
		postNewMessageBroadcast,
		postNewMessagePrivate,
		postNewMessageRequestReply,
		postNewSubscription,
		postOpRetry,
		postTokenApproval,
		postTokenBurn,
		postTokenMint,
		postTokenPool,
		postTokenTransfer,
		putContractAPI,
		putSubscription,
	})...,
)

func globalRoutes(routes []*oapispec.Route) []*oapispec.Route {
	for _, route := range routes {
		route.Tag = "Global"
	}
	return routes
}

func namespacedRoutes(routes []*oapispec.Route) []*oapispec.Route {
	newRoutes := make([]*oapispec.Route, len(routes))
	for i, route := range routes {
		route.Tag = "Default Namespace"

		routeCopy := *route
		routeCopy.Name += "Namespace"
		routeCopy.Path = "namespaces/{ns}/" + route.Path
		routeCopy.PathParams = append(routeCopy.PathParams, &oapispec.PathParam{
			Name: "ns", ExampleFromConf: coreconfig.NamespacesDefault, Description: coremsgs.APIParamsNamespace,
		})
		routeCopy.Tag = "Non-Default Namespace"
		newRoutes[i] = &routeCopy
	}
	return append(routes, newRoutes...)
}

func extractNamespace(pathParams map[string]string) string {
	if ns, ok := pathParams["ns"]; ok {
		return ns
	}
	return config.GetString(coreconfig.NamespacesDefault)
}
