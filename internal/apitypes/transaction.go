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

package apitypes

import "github.com/google/uuid"

type TransactionType string

const (
	TransactionTypeNone MessageType = "none"
	TransactionTypePin  MessageType = "pin"
)

type TransactionState string

const (
	// TransactionStateSubmitted the transaction has been submitted
	TransactionStateSubmitted TransactionState = "submitted"
	// TransactionStateSubmitted the transaction is considered final per the rules of the blockchain technnology
	TransactionStateConfirmed TransactionState = "confirmed"
	// TransactionStateSubmitted the transaction has encountered, and is unlikely to ever become final on the blockchain. However, it is not impossible it will still be mined.
	TransactionStateFailed TransactionState = "error"
)

type Transaction struct {
	ID         *uuid.UUID             `json:"id,omitempty"`
	TrackingID string                 `json:"trackingId,omitempty"`
	Sequence   string                 `json:"sequence,omitempty"`
	Info       map[string]interface{} `json:"info,omitempty"`
}
