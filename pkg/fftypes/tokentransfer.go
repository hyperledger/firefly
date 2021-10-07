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

type TokenTransferType = FFEnum

var (
	TokenTransferTypeMint     TokenType = ffEnum("tokentransfertype", "mint")
	TokenTransferTypeBurn     TokenType = ffEnum("tokentransfertype", "burn")
	TokenTransferTypeTransfer TokenType = ffEnum("tokentransfertype", "transfer")
)

type TokenTransfer struct {
	Type           TokenTransferType `json:"type" ffenum:"tokentransfertype"`
	LocalID        *UUID             `json:"localId,omitempty"`
	PoolProtocolID string            `json:"poolProtocolId,omitempty"`
	TokenIndex     string            `json:"tokenIndex,omitempty"`
	Key            string            `json:"key,omitempty"`
	From           string            `json:"from,omitempty"`
	To             string            `json:"to,omitempty"`
	Amount         int64             `json:"amount,omitempty"`
	ProtocolID     string            `json:"protocolId,omitempty"`
	MessageHash    *Bytes32          `json:"messageHash,omitempty"`
	Created        *FFTime           `json:"created,omitempty"`
	TX             TransactionRef    `json:"tx,omitempty"`
}
