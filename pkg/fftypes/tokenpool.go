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

type TokenType = LowerCasedType

const (
	TokenTypeFungible    TokenType = "fungible"
	TokenTypeNonFungible TokenType = "nonfungible"
)

type TokenPoolCreate struct {
	BaseURI string    `json:"base_uri"`
	Type    TokenType `json:"type"`
}

type TokenPool struct {
	BaseURI string    `json:"base_uri"`
	Type    TokenType `json:"type"`
	PoolID  string    `json:"pool_id"`
}

type TokenMint struct {
	PoolID    string `json:"pool_id"`
	Recipient string `json:"recipient"`
	Amount    int    `json:"amount,omitempty"`
}
