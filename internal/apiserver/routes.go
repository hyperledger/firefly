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
	"context"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffapi"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/orchestrator"
	"github.com/hyperledger/firefly/pkg/database"
)

type coreRequest struct {
	or         orchestrator.Orchestrator
	ctx        context.Context
	filter     database.AndFilter
	apiBaseURL string
}

type coreExtensions struct {
	FilterFactory         database.QueryFactory
	CoreJSONHandler       func(r *ffapi.APIRequest, cr *coreRequest) (output interface{}, err error)
	CoreFormUploadHandler func(r *ffapi.APIRequest, cr *coreRequest) (output interface{}, err error)
}

var routes = append(
	globalRoutes([]*ffapi.Route{
		getNamespace,
		getNamespaces,
	}),
	namespacedRoutes([]*ffapi.Route{
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
		getIdentityByDID,
		getIdentityByID,
		getIdentityDID,
		getIdentityVerifiers,
		getMsgByID,
		getMsgData,
		getMsgEvents,
		getMsgs,
		getMsgTxn,
		getNetworkDIDDocByDID,
		getNetworkIdentities,
		getNetworkIdentityByDID,
		getNetworkNode,
		getNetworkNodes,
		getNetworkOrg,
		getNetworkOrgs,
		getOpByID,
		getOps,
		getPins,
		getStatus,
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
		postNetworkAction,
		postNewContractAPI,
		postNewContractInterface,
		postNewContractListener,
		postNewDatatype,
		postNewIdentity,
		postNewMessageBroadcast,
		postNewMessagePrivate,
		postNewMessageRequestReply,
		postNewSubscription,
		postNewOrganization,
		postNewOrganizationSelf,
		postNodesSelf,
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

func globalRoutes(routes []*ffapi.Route) []*ffapi.Route {
	for _, route := range routes {
		route.Tag = "Global"
	}
	return routes
}

func namespacedRoutes(routes []*ffapi.Route) []*ffapi.Route {
	newRoutes := make([]*ffapi.Route, len(routes))
	for i, route := range routes {
		route.Tag = "Default Namespace"

		routeCopy := *route
		routeCopy.Name += "Namespace"
		routeCopy.Path = "namespaces/{ns}/" + route.Path
		routeCopy.PathParams = append(routeCopy.PathParams, &ffapi.PathParam{
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
