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

package fftypes

// OpType describes mechanical steps in the process that have to be performed,
// might be asynchronous, and have results in the back-end systems that might need
// to be correlated with messages by operators.
type OpType = FFEnum

var (
	// OpTypeBlockchainBatchPin is a blockchain transaction to pin a batch
	OpTypeBlockchainBatchPin = ffEnum("optype", "blockchain_batch_pin")
	// OpTypeBlockchainInvoke is a smart contract invoke
	OpTypeBlockchainInvoke = ffEnum("optype", "blockchain_invoke")
	// OpTypeSharedStorageBatchBroadcast is a shared storage operation to store broadcast data
	OpTypeSharedStorageBatchBroadcast = ffEnum("optype", "sharedstorage_batch_broadcast")
	// OpTypeDataExchangeBatchSend is a private send
	OpTypeDataExchangeBatchSend = ffEnum("optype", "dataexchange_batch_send")
	// OpTypeDataExchangeBlobSend is a private send
	OpTypeDataExchangeBlobSend = ffEnum("optype", "dataexchange_blob_send")
	// OpTypeTokenCreatePool is a token pool creation
	OpTypeTokenCreatePool = ffEnum("optype", "token_create_pool")
	// OpTypeTokenActivatePool is a token pool activation
	OpTypeTokenActivatePool = ffEnum("optype", "token_activate_pool")
	// OpTypeTokenTransfer is a token transfer
	OpTypeTokenTransfer = ffEnum("optype", "token_transfer")
	// OpTypeTokenApproval is a token approval
	OpTypeTokenApproval = ffEnum("optype", "token_approval")
)

// OpStatus is the current status of an operation
type OpStatus string

const (
	// OpStatusPending indicates the operation has been submitted, but is not yet confirmed as successful or failed
	OpStatusPending OpStatus = "Pending"
	// OpStatusSucceeded the infrastructure runtime has returned success for the operation
	OpStatusSucceeded OpStatus = "Succeeded"
	// OpStatusFailed happens when an error is reported by the infrastructure runtime
	OpStatusFailed OpStatus = "Failed"
)

type Named interface {
	Name() string
}

// NewOperation creates a new operation in a transaction
func NewOperation(plugin Named, namespace string, tx *UUID, opType OpType) *Operation {
	now := Now()
	return &Operation{
		ID:          NewUUID(),
		Namespace:   namespace,
		Plugin:      plugin.Name(),
		Transaction: tx,
		Type:        opType,
		Status:      OpStatusPending,
		Created:     now,
		Updated:     now,
	}
}

// Operation is a description of an action performed as part of a transaction submitted by this node
type Operation struct {
	ID          *UUID      `json:"id"`
	Namespace   string     `json:"namespace"`
	Transaction *UUID      `json:"tx"`
	Type        OpType     `json:"type" ffenum:"optype"`
	Status      OpStatus   `json:"status"`
	Error       string     `json:"error,omitempty"`
	Plugin      string     `json:"plugin"`
	Input       JSONObject `json:"input,omitempty"`
	Output      JSONObject `json:"output,omitempty"`
	Created     *FFTime    `json:"created,omitempty"`
	Updated     *FFTime    `json:"updated,omitempty"`
	Retry       *UUID      `json:"retry,omitempty"`
}

// PreparedOperation is an operation that has gathered all the raw data ready to send to a plugin
// It is never stored, but it should always be possible for the owning Manager to generate a
// PreparedOperation from an Operation. Data is defined by the Manager, but should be JSON-serializable
// to support inspection and debugging.
type PreparedOperation struct {
	ID   *UUID       `json:"id"`
	Type OpType      `json:"type" ffenum:"optype"`
	Data interface{} `json:"data"`
}
