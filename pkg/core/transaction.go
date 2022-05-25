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

package core

import "github.com/hyperledger/firefly-common/pkg/fftypes"

type TransactionType = FFEnum

const transactionBaseSizeEstimate = int64(256)

var (
	// TransactionTypeNone deprecated - replaced by TransactionTypeUnpinned
	TransactionTypeNone = ffEnum("txtype", "none")
	// TransactionTypeUnpinned indicates the message will be sent without pinning any evidence to the blockchain. Not supported for broadcast. The FireFly transaction will be used to track the sends to all group members.
	TransactionTypeUnpinned = ffEnum("txtype", "unpinned")
	// TransactionTypeBatchPin represents a pinning transaction, that verifies the originator of the data, and sequences the event deterministically between parties
	TransactionTypeBatchPin = ffEnum("txtype", "batch_pin")
	// TransactionTypeTokenPool represents a token pool creation
	TransactionTypeTokenPool = ffEnum("txtype", "token_pool")
	// TransactionTypeTokenTransfer represents a token transfer
	TransactionTypeTokenTransfer = ffEnum("txtype", "token_transfer")
	// TransactionTypeContractInvoke is a smart contract invoke
	TransactionTypeContractInvoke = ffEnum("txtype", "contract_invoke")
	// TransactionTypeTokenTransfer represents a token approval
	TransactionTypeTokenApproval = ffEnum("txtype", "token_approval")
)

// TransactionRef refers to a transaction, in other types
type TransactionRef struct {
	Type TransactionType `ffstruct:"Transaction" json:"type"`
	ID   *fftypes.UUID   `ffstruct:"Transaction" json:"id,omitempty"`
}

// BlockchainTransactionRef refers to a transaction and a transaction blockchain ID, in other types
type BlockchainTransactionRef struct {
	Type         TransactionType `ffstruct:"Transaction" json:"type,omitempty"`
	ID           *fftypes.UUID   `ffstruct:"Transaction" json:"id,omitempty"`
	BlockchainID string          `ffstruct:"Transaction" json:"blockchainId,omitempty"`
}

// Transaction is a unit of work sent or received by this node
// It serves as a container for one or more Operations, BlockchainEvents, and other related objects
type Transaction struct {
	ID            *fftypes.UUID   `ffstruct:"Transaction" json:"id,omitempty"`
	Namespace     string          `ffstruct:"Transaction" json:"namespace,omitempty"`
	Type          TransactionType `ffstruct:"Transaction" json:"type" ffenum:"txtype"`
	Created       *fftypes.FFTime `ffstruct:"Transaction" json:"created"`
	BlockchainIDs FFStringArray   `ffstruct:"Transaction" json:"blockchainIds,omitempty"`
}

type TransactionStatusType string

var (
	TransactionStatusTypeOperation       TransactionStatusType = "Operation"
	TransactionStatusTypeBlockchainEvent TransactionStatusType = "BlockchainEvent"
	TransactionStatusTypeBatch           TransactionStatusType = "Batch"
	TransactionStatusTypeTokenPool       TransactionStatusType = "TokenPool"
	TransactionStatusTypeTokenTransfer   TransactionStatusType = "TokenTransfer"
	TransactionStatusTypeTokenApproval   TransactionStatusType = "TokenApproval"
)

type TransactionStatusDetails struct {
	Type      TransactionStatusType `ffstruct:"TransactionStatusDetails" json:"type"`
	SubType   string                `ffstruct:"TransactionStatusDetails" json:"subtype,omitempty"`
	Status    OpStatus              `ffstruct:"TransactionStatusDetails" json:"status"`
	Timestamp *fftypes.FFTime       `ffstruct:"TransactionStatusDetails" json:"timestamp,omitempty"`
	ID        *fftypes.UUID         `ffstruct:"TransactionStatusDetails" json:"id,omitempty"`
	Error     string                `ffstruct:"TransactionStatusDetails" json:"error,omitempty"`
	Info      fftypes.JSONObject    `ffstruct:"TransactionStatusDetails" json:"info,omitempty"`
}

type TransactionStatus struct {
	Status  OpStatus                    `ffstruct:"TransactionStatus" json:"status"`
	Details []*TransactionStatusDetails `ffstruct:"TransactionStatus" json:"details"`
}

func (tx *Transaction) Size() int64 {
	return transactionBaseSizeEstimate // currently a static size assessment for caching
}
