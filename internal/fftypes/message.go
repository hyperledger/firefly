// Copyright © 2021 Kaleido, Inc.
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

import "github.com/google/uuid"

type MessageType string

const (
	MessageTypeDefinition MessageType = "definition"
	MessageTypeBroadcast  MessageType = "broadcast"
	MessageTypePrivate    MessageType = "private"
)

// MessageBase is the raw message, without any data relationships
type MessageBase struct {
	// Hashed fields
	ID        *uuid.UUID  `json:"id,omitempty"`
	CID       *uuid.UUID  `json:"cid,omitempty"`
	Type      MessageType `json:"type"`
	Author    string      `json:"author,omitempty"`
	Created   int64       `json:"created,omitempty"`
	Namespace string      `json:"namespace,omitempty"`
	Topic     string      `json:"topic,omitempty"`
	Context   string      `json:"context,omitempty"`
	Group     *uuid.UUID  `json:"group,omitempty"`
	DataHash  *Bytes32    `json:"datahash,omitempty"`
	// Unhashed fields
	Hash      *Bytes32 `json:"hash,omitempty"`
	Confirmed int64    `json:"confirmed,omitempty"`
}

type MessageExpanded struct {
	MessageBase
	TX   *Transaction `json:"tx"`
	Data []*Data      `json:"data"`
}

type MessageRefsOnly struct {
	MessageBase
	TX   TransactionRef  `json:"tx,omitempty"`
	Data DataRefSortable `json:"data"`
}
