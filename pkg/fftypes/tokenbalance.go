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
	Pool       *UUID   `json:"pool,omitempty"`
	TokenIndex string  `json:"tokenIndex,omitempty"`
	Connector  string  `json:"connector,omitempty"`
	Namespace  string  `json:"namespace,omitempty"`
	Key        string  `json:"key,omitempty"`
	Balance    BigInt  `json:"balance"`
	Updated    *FFTime `json:"updated,omitempty"`
}

func TokenBalanceIdentifier(pool *UUID, tokenIndex, identity string) string {
	return pool.String() + ":" + tokenIndex + ":" + identity
}

func (t *TokenBalance) Identifier() string {
	return TokenBalanceIdentifier(t.Pool, t.TokenIndex, t.Key)
}

// Currently these types are just filtered views of TokenBalance.
// If more fields/aggregation become needed, they might merit a new table in the database.
type TokenAccount struct {
	Key string `json:"key,omitempty"`
}
type TokenAccountPool struct {
	Pool *UUID `json:"pool,omitempty"`
}
