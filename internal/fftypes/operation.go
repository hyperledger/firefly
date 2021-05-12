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

type OpType string

const (
	OpTypeMessagePin       OpType = "message_pin"
	OpTypeMessageSend      OpType = "message_transfer"
	OpTypeMessageBroadcast OpType = "message_broacast"
	OpTypeDataSend         OpType = "data_transfer"
	OpTypeDataBroadcast    OpType = "data_broadcast"
)

type OpDirection string

const (
	OpDirectionInbound  OpDirection = "inbound"
	OpDirectionOutbound OpDirection = "outbound"
)

type OpStatus string

const (
	OpStatusPending   OpStatus = "pending"
	OpStatusSucceeded OpStatus = "succeeded"
	OpStatusFailed    OpStatus = "failed"
)

type Operation struct {
	ID        *uuid.UUID  `json:"id"`
	Namespace string      `json:"namespace,omitempty"`
	Message   *uuid.UUID  `json:"message"`
	Data      *uuid.UUID  `json:"data,omitempty"`
	Type      OpType      `json:"type"`
	Direction OpDirection `json:"direction"`
	Recipient string      `json:"recipient,omitempty"`
	Status    OpStatus    `json:"status"`
	Error     string      `json:"error,omitempty"`
	Plugin    string      `json:"plugin"`
	BackendID string      `json:"backendId"`
	Created   int64       `json:"created,omitempty"`
	Updated   int64       `json:"updated,omitempty"`
}
