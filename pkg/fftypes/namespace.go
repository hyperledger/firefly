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

// NamespaceType describes when the namespace was created from local configuration, or broadcast through the network
type NamespaceType string

const (
	// NamespaceTypeLocal is a namespace that only exists because it was defined in the local configuration of the node
	NamespaceTypeLocal NamespaceType = "local"
	// NamespaceTypeBroadcast is a namespace that was broadcast through the network. Broadcast namespaces can overwrite a local namespace
	NamespaceTypeBroadcast NamespaceType = "broadcast"
)

// Namespace is a isolate set of named resources, to allow multiple applications to co-exist in the same network, with the same named objects.
// Can be used for use case segregation, or multi-tenancy.
type Namespace struct {
	ID          *UUID         `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Type        NamespaceType `json:"type"`
	Created     *FFTime       `json:"created"`
	Confirmed   *FFTime       `json:"confirmed"`
}
