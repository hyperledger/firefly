// Copyright © 2021 Kaleido, Inc.
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
	postNewDatatype,
	postNewNamespace,
	postNewMessageBroadcast,
	postNewMessagePrivate,
	postNewMessageRequestReply,
	postNodesSelf,
	postNewOrganization,
	postNewOrganizationSelf,

	postBroadcastDatatype,
	postBroadcastMessage,
	postBroadcastNamespace,
	postData,
	postNewSubscription,
	postRegisterOrg,
	postRegisterNode,
	postRegisterNodeOrg,
	postRequestMessage,
	postSendMessage,

	putSubscription,

	deleteSubscription,

	getBatchByID,
	getBatches,
	getData,
	getDataBlob,
	getDataByID,
	getDatatypeByID,
	getDatatypeByName,
	getDatatypes,
	getDataMsgs,
	getEventByID,
	getEvents,
	getMsgByID,
	getMsgData,
	getMsgEvents,
	getMsgOps,
	getMsgTxn,
	getMsgs,
	getNetworkOrg,
	getNetworkOrgs,
	getNetworkNode,
	getNetworkNodes,
	getNamespace,
	getNamespaces,
	getOpByID,
	getOps,
	getStatus,
	getSubscriptionByID,
	getSubscriptions,
	getTxnByID,
	getTxnOps,
	getTxns,

	postTokenPoolByType,
	getTokenPools,
	getTokenPoolsByType,
	getTokenPoolByNameOrID,
	getTokenPoolByName,
	getTokenAccounts,
	getTokenAccountsByPool,
	getTokenTransfers,
	getTokenTransfersByPool,
	getTokenTransferByID,
	postTokenMint,
	postTokenMintByType,
	postTokenBurn,
	postTokenBurnByType,
	postTokenTransfer,
	postTokenTransferByType,
	getTokenConnectors,
}
