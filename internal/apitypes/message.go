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

package apitypes

import "github.com/google/uuid"

type MessageType string

const (
	MessageTypeBroadcast MessageType = "broadcast"
	MessageTypePrivate   MessageType = "private"
)

type messageCommon struct {
	// Hashed fields
	ID       *uuid.UUID `json:"id,omitempty"`
	Type     string     `json:"type"`
	Author   string     `json:"author,omitempty"`
	Created  uint64     `json:"created,omitempty"`
	Topic    string     `json:"topic,omitempty"`
	Context  string     `json:"context,omitempty"`
	Group    *uuid.UUID `json:"group,omitempty"`
	CID      *uuid.UUID `json:"cid,omitempty"`
	DataHash string     `json:"datahash,omitempty"`
	// Unhashed fields
	Hash      string `json:"hash,omitempty"`
	Confirmed uint64 `json:"confirmed,omitempty"`
}

type MessageExpanded struct {
	messageCommon
	TX   *Transaction    `json:"tx,omitempty"`
	Data []*DataExpanded `json:"data"`
}

type MessageRefsOnly struct {
	messageCommon
	TX   *uuid.UUID `json:"tx,omitempty"`
	Data []DataRef  `json:"data"`
}
