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

type TokenApprovalInput struct {
	TokenApproval
	Pool string `json:"pool,omitempty"`
}

type TokenApproval struct {
	LocalID         *UUID          `json:"localId,omitempty"`
	Pool            *UUID          `json:"pool,omitempty"`
	TokenIndex      string         `json:"tokenIndex,omitempty"`
	Connector       string         `json:"connector,omitempty"`
	Key             string         `json:"key,omitempty"`
	Operator        string         `json:"operator,omitempty"`
	Approved        bool           `json:"approved"`
	Info            JSONObject     `json:"info,omitempty"`
	Namespace       string         `json:"namespace,omitempty"`
	ProtocolID      string         `json:"protocolId,omitempty"`
	Created         *FFTime        `json:"created,omitempty"`
	TX              TransactionRef `json:"tx"`
	BlockchainEvent *UUID          `json:"blockchainEvent,omitempty"`
	Config          JSONObject     `json:"config,omitempty"` // for REST calls only (not stored)
}
