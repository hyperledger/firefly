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

// NodeStatus is a set of information that represents the health, and identity of a node
type NodeStatus struct {
	Node     NodeStatusNode     `ffstruct:"NodeStatus" json:"node"`
	Org      NodeStatusOrg      `ffstruct:"NodeStatus" json:"org"`
	Defaults NodeStatusDefaults `ffstruct:"NodeStatus" json:"defaults"`
	Plugins  NodeStatusPlugins  `ffstruct:"NodeStatus" json:"plugins"`
}

// NodeStatusNode is the information about the local node, returned in the node status
type NodeStatusNode struct {
	Name       string `ffstruct:"NodeStatusNode" json:"name"`
	Registered bool   `ffstruct:"NodeStatusNode" json:"registered"`
	ID         *UUID  `ffstruct:"NodeStatusNode" json:"id,omitempty"`
}

// NodeStatusOrg is the information about the node owning org, returned in the node status
type NodeStatusOrg struct {
	Name       string         `ffstruct:"NodeStatusOrg" json:"name"`
	Registered bool           `ffstruct:"NodeStatusOrg" json:"registered"`
	DID        string         `ffstruct:"NodeStatusOrg" json:"did,omitempty"`
	ID         *UUID          `ffstruct:"NodeStatusOrg" json:"id,omitempty"`
	Verifiers  []*VerifierRef `ffstruct:"NodeStatusOrg" json:"verifiers,omitempty"`
}

// NodeStatusDefaults is information about core configuration th
type NodeStatusDefaults struct {
	Namespace string `ffstruct:"NodeStatusDefaults" json:"namespace"`
}

// NodeStatusPlugins is a map of plugins configured on the node
type NodeStatusPlugins struct {
	Blockchain    []*NodeStatusPlugin `ffstruct:"NodeStatusPlugins" json:"blockchain"`
	Database      []*NodeStatusPlugin `ffstruct:"NodeStatusPlugins" json:"database"`
	DataExchange  []*NodeStatusPlugin `ffstruct:"NodeStatusPlugins" json:"dataExchange"`
	Identity      []*NodeStatusPlugin `ffstruct:"NodeStatusPlugins" json:"identity"`
	SharedStorage []*NodeStatusPlugin `ffstruct:"NodeStatusPlugins" json:"sharedStorage"`
	Tokens        []*NodeStatusPlugin `ffstruct:"NodeStatusPlugins" json:"tokens"`
}

// NodeStatusPlugin is information about a plugin
type NodeStatusPlugin struct {
	Connection string `ffstruct:"NodeStatusPlugin" json:"connection"`
}
