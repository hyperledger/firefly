// Copyright © 2021 Kaleido, Inc.
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

import (
	"crypto/sha256"
	"encoding/json"

	"github.com/google/uuid"
)

type TransactionType string

const (
	TransactionTypeNone TransactionType = "none"
	TransactionTypePin  TransactionType = "pin"
)

type TransactionStatus string

const (
	// TransactionStatusPending the transaction has been submitted
	TransactionStatusPending TransactionStatus = "pending"
	// TransactionStatusConfirmed the transaction is considered final per the rules of the blockchain technnology
	TransactionStatusConfirmed TransactionStatus = "confirmed"
	// TransactionStatusFailed the transaction has encountered, and is unlikely to ever become final on the blockchain. However, it is not impossible it will still be mined.
	TransactionStatusFailed TransactionStatus = "error"
)

type TransactionRef struct {
	Type TransactionType `json:"type"`
	// ID is a direct reference to a submitted transaction
	ID *uuid.UUID `json:"id,omitempty"`
}

type TransactionSubject struct {
	Author    string          `json:"author"`
	Namespace string          `json:"namespace,omitempty"`
	Type      TransactionType `json:"type"`
	Message   *uuid.UUID      `json:"message,omitempty"`
	Batch     *uuid.UUID      `json:"batch,omitempty"`
}

func (t *TransactionSubject) Hash() *Bytes32 {
	b, _ := json.Marshal(&t)
	var b32 Bytes32 = sha256.Sum256(b)
	return &b32
}

// Transactions are (blockchain) transactions that were submitted by this
// node, with the correlation information to look them up on the underlying
// ledger technology
type Transaction struct {
	ID         *uuid.UUID         `json:"id,omitempty"`
	Hash       *Bytes32           `json:"hash"`
	Subject    TransactionSubject `json:"subject"`
	Sequence   int64              `json:"sequence,omitempty"`
	Created    *FFTime            `json:"created"`
	Status     TransactionStatus  `json:"status"`
	ProtocolID string             `json:"protocolId,omitempty"`
	Confirmed  *FFTime            `json:"confirmed,omitempty"`
	Info       JSONObject         `json:"info,omitempty"`
}
