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

package fftypes

// OpType describes mechanical steps in the process that have to be performed,
// might be asynchronous, and have results in the back-end systems that might need
// to be correlated with messages by operators.
type OpType = FFEnum

var (
	// OpTypeBlockchainBatchPin is a blockchain transaction to pin a batch
	OpTypeBlockchainBatchPin OpType = ffEnum("optype", "blockchain_batch_pin")
	// OpTypePublicStorageBatchBroadcast is a public storage operation to store broadcast data
	OpTypePublicStorageBatchBroadcast OpType = ffEnum("optype", "publicstorage_batch_broadcast")
	// OpTypeDataExchangeBatchSend is a private send
	OpTypeDataExchangeBatchSend OpType = ffEnum("optype", "dataexchange_batch_send")
	// OpTypeDataExchangeBlobSend is a private send
	OpTypeDataExchangeBlobSend OpType = ffEnum("optype", "dataexchange_blob_send")
	// OpTypeTokensCreatePool is a token pool creation
	OpTypeTokensCreatePool OpType = ffEnum("optype", "tokens_create_pool")
)

// OpStatus is the current status of an operation
type OpStatus string

const (
	// OpStatusPending indicates the operation has been submitted, but is not yet confirmed as successful or failed
	OpStatusPending OpStatus = "Pending"
	// OpStatusSucceeded the infrastructure runtime has returned success for the operation.
	OpStatusSucceeded OpStatus = "Succeeded"
	// OpStatusFailed happens when an error is reported by the infrastructure runtime
	OpStatusFailed OpStatus = "Failed"
)

type Named interface {
	Name() string
}

// NewTXOperation creates a new operation for a transaction
func NewTXOperation(plugin Named, namespace string, tx *UUID, backendID string, opType OpType, opStatus OpStatus, member string) *Operation {
	return &Operation{
		ID:          NewUUID(),
		Namespace:   namespace,
		Plugin:      plugin.Name(),
		BackendID:   backendID,
		Transaction: tx,
		Type:        opType,
		Member:      member,
		Status:      opStatus,
		Created:     Now(),
	}
}

// Operation is a description of an action performed as part of a transaction submitted by this node
type Operation struct {
	ID          *UUID      `json:"id"`
	Namespace   string     `json:"namespace"`
	Transaction *UUID      `json:"tx"`
	Type        OpType     `json:"type" ffenum:"optype"`
	Member      string     `json:"member,omitempty"`
	Status      OpStatus   `json:"status"`
	Error       string     `json:"error,omitempty"`
	Plugin      string     `json:"plugin"`
	BackendID   string     `json:"backendId"`
	Info        JSONObject `json:"info,omitempty"`
	Created     *FFTime    `json:"created,omitempty"`
	Updated     *FFTime    `json:"updated,omitempty"`
}
