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

type TransactionType = FFEnum

var (
	// TransactionTypeNone deprecated - replaced by TransactionTypeUnpinned
	TransactionTypeNone TransactionType = ffEnum("txtype", "none")
	// TransactionTypeUnpinned indicates the message will be sent without pinning any evidence to the blockchain. Not supported for broadcast. The FireFly transaction will be used to track the sends to all group members.
	TransactionTypeUnpinned TransactionType = ffEnum("txtype", "unpinned")
	// TransactionTypeBatchPin represents a pinning transaction, that verifies the originator of the data, and sequences the event deterministically between parties
	TransactionTypeBatchPin TransactionType = ffEnum("txtype", "batch_pin")
	// TransactionTypeTokenPool represents a token pool creation
	TransactionTypeTokenPool TransactionType = ffEnum("txtype", "token_pool")
	// TransactionTypeTokenTransfer represents a token transfer
	TransactionTypeTokenTransfer TransactionType = ffEnum("txtype", "token_transfer")
	// TransactionTypeContractInvoke is a smart contract invoke
	TransactionTypeContractInvoke OpType = ffEnum("txtype", "contract_invoke")
)

// TransactionRef refers to a transaction, in other types
type TransactionRef struct {
	Type TransactionType `json:"type"`
	ID   *UUID           `json:"id,omitempty"`
}

// Transaction is a unit of work sent or received by this node
// It serves as a container for one or more Operations, BlockchainEvents, and other related objects
type Transaction struct {
	ID            *UUID           `json:"id,omitempty"`
	Namespace     string          `json:"namespace,omitempty"`
	Type          TransactionType `json:"type" ffenum:"txtype"`
	Created       *FFTime         `json:"created"`
	BlockchainIDs FFStringArray   `json:"blockchainIds,omitempty"`
}

type TransactionStatusType string

var (
	TransactionStatusTypeOperation       TransactionStatusType = "Operation"
	TransactionStatusTypeBlockchainEvent TransactionStatusType = "BlockchainEvent"
	TransactionStatusTypeBatch           TransactionStatusType = "Batch"
	TransactionStatusTypeTokenPool       TransactionStatusType = "TokenPool"
	TransactionStatusTypeTokenTransfer   TransactionStatusType = "TokenTransfer"
)

type TransactionStatusDetails struct {
	Type      TransactionStatusType `json:"type"`
	SubType   string                `json:"subtype,omitempty"`
	Status    OpStatus              `json:"status"`
	Timestamp *FFTime               `json:"timestamp,omitempty"`
	ID        *UUID                 `json:"id,omitempty"`
	Error     string                `json:"error,omitempty"`
	Info      JSONObject            `json:"info,omitempty"`
}

type TransactionStatus struct {
	Status  OpStatus                    `json:"status"`
	Details []*TransactionStatusDetails `json:"details"`
}
