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
	Pool string `ffstruct:"TokenApprovalInput" json:"pool,omitempty"`
}

type TokenApproval struct {
	LocalID         *UUID          `ffstruct:"TokenApproval" json:"localId,omitempty" ffexcludeinput:"true"`
	Pool            *UUID          `ffstruct:"TokenApproval" json:"pool,omitempty"`
	TokenIndex      string         `ffstruct:"TokenApproval" json:"tokenIndex,omitempty"`
	Connector       string         `ffstruct:"TokenApproval" json:"connector,omitempty"`
	Key             string         `ffstruct:"TokenApproval" json:"key,omitempty"`
	Operator        string         `ffstruct:"TokenApproval" json:"operator,omitempty"`
	Approved        bool           `ffstruct:"TokenApproval" json:"approved"`
	Info            JSONObject     `ffstruct:"TokenApproval" json:"info,omitempty"`
	Namespace       string         `ffstruct:"TokenApproval" json:"namespace,omitempty"`
	ProtocolID      string         `ffstruct:"TokenApproval" json:"protocolId,omitempty" ffexcludeinput:"true"`
	Created         *FFTime        `ffstruct:"TokenApproval" json:"created,omitempty" ffexcludeinput:"true"`
	TX              TransactionRef `ffstruct:"TokenApproval" json:"tx" ffexcludeinput:"true"`
	BlockchainEvent *UUID          `ffstruct:"TokenApproval" json:"blockchainEvent,omitempty" ffexcludeinput:"true"`
	Config          JSONObject     `ffstruct:"TokenApproval" json:"config,omitempty" ffexcludeoutput:"true"` // for REST calls only (not stored)
}
