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

type TokenAccount struct {
	ProtocolID string `json:"protocolId,omitempty"`
	TokenIndex string `json:"tokenIndex,omitempty"`
	Identity   string `json:"identity,omitempty"`
	Balance    int64  `json:"balance"`
}

func TokenAccountIdentifier(protocolID, tokenIndex, identity string) string {
	return protocolID + ":" + tokenIndex + ":" + identity
}

func (a *TokenAccount) Identifier() string {
	return TokenAccountIdentifier(a.ProtocolID, a.TokenIndex, a.Identity)
}
