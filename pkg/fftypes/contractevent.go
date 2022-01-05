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

type ContractEvent struct {
	ID           *UUID      `json:"id,omitempty"`
	Sequence     int64      `json:"sequence"`
	Namespace    string     `json:"namespace,omitempty"`
	Name         string     `json:"name,omitempty"`
	Subscription *UUID      `json:"subscription,omitempty"`
	Outputs      JSONObject `json:"outputs,omitempty"`
	Info         JSONObject `json:"info,omitempty"`
	Timestamp    *FFTime    `json:"timestamp,omitempty"`
}
