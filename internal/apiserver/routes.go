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

import "github.com/hyperledger/firefly/internal/oapispec"

const emptyObjectSchema = `{"type": "object"}`

var routes = []*oapispec.Route{
	deleteContractListener,
	deleteSubscription,
	getBatchByID,
	getBatches,
	getBlockchainEventByID,
	getBlockchainEvents,
	getChartHistogram,
	getContractAPIByName,
	getContractAPIs,
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
	getMsgOps,
	getMsgs,
	getMsgTxn,
	getNamespace,
	getNamespaces,
	getNetworkNode,
	getNetworkNodes,
	getNetworkOrg,
	getNetworkOrgs,
	getOpByID,
	getOps,
	getStatus,
	getStatusBatchManager,
	getStatusPins,
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
	postContractInterfaceGenerate,
	postContractInterfaceInvoke,
	postContractInterfaceQuery,
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
	postNewNamespace,
	postNewOrganization,
	postNewOrganizationSelf,
	postNewSubscription,
	postNodesSelf,
	postOpRetry,
	postTokenApproval,
	postTokenBurn,
	postTokenMint,
	postTokenPool,
	postTokenTransfer,
	putContractAPI,
	putSubscription,
}
