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

type TokenBalance struct {
	PoolProtocolID string  `json:"poolProtocolId,omitempty"`
	TokenIndex     string  `json:"tokenIndex,omitempty"`
	Connector      string  `json:"connector,omitempty"`
	Namespace      string  `json:"namespace,omitempty"`
	Key            string  `json:"key,omitempty"`
	Balance        BigInt  `json:"balance"`
	Updated        *FFTime `json:"updated,omitempty"`
}

func TokenBalanceIdentifier(protocolID, tokenIndex, identity string) string {
	return protocolID + ":" + tokenIndex + ":" + identity
}

func (t *TokenBalance) Identifier() string {
	return TokenBalanceIdentifier(t.PoolProtocolID, t.TokenIndex, t.Key)
}

// Currently this type is just a filtered view of TokenBalance.
// If more fields/aggregation become needed, this may need its own table in the database.
type TokenAccount struct {
	Key string `json:"key,omitempty"`
}
