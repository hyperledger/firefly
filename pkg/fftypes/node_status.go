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

// NodeStatus is a set of information that represents the health, and identity of a node
type NodeStatus struct {
	Node     NodeStatusNode     `json:"node"`
	Org      NodeStatusOrg      `json:"org"`
	Defaults NodeStatusDefaults `json:"defaults"`
}

// NodeStatusNode is the information about the local node, returned in the node status
type NodeStatusNode struct {
	Name       string `json:"name"`
	Registered bool   `json:"registered"`
	ID         *UUID  `json:"id,omitempty"`
}

// NodeStatusOrg is the information about the node owning org, returned in the node status
type NodeStatusOrg struct {
	Name       string `json:"name"`
	Registered bool   `json:"registered"`
	Identity   string `json:"identity,omitempty"`
	ID         *UUID  `json:"id,omitempty"`
}

// NodeStatusDefaults is information about core configuration th
type NodeStatusDefaults struct {
	Namespace string `json:"namespace"`
}
