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

// ChangeEventType
type ChangeEventType string

const (
	ChangeEventTypeCreated ChangeEventType = "created"
	ChangeEventTypeUpdated ChangeEventType = "updated" // note bulk updates might not results in change events.
	ChangeEventTypeDeleted ChangeEventType = "deleted"
)

// ChangeEvent is a change to the local FireFly core node.
type ChangeEvent struct {
	// The resource collection where the changed resource exists
	Collection string `json:"collection"`
	// The type of event
	Type ChangeEventType `json:"type"`
	// Namespace is set if there is a namespace associated with the changed resource
	Namespace string `json:"namespace,omitempty"`
	// UUID is set if the resource is identified by ID
	ID *UUID `json:"id,omitempty"`
	// Hash is set if the resource is identified primarily by hash (groups is currently the only example)
	Hash *Bytes32 `json:"hash,omitempty"`
	// Sequence is set if there is a local ordered sequence associated with the changed resource
	Sequence *int64 `json:"sequence,omitempty"`
}
