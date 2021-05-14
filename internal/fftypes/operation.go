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

// OpType describes mechanical steps in the process that have to be performed,
// might be asynchronous, and have results in the back-end systems that might need
// to be correlated with messages by operators.
type OpType string

const (
	OpTypeBlockchainBatchPin  OpType = "blockchain_batch_pin"
	OpTypeP2PFSBatchBroadcast OpType = "p2pfs_batch_broadcast"
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

type Named interface {
	Name() string
}

func NewMessageOp(plugin Named, backendID string, msg *Message, opType OpType, opDir OpDirection, opStatus OpStatus, recipient string) *Operation {
	return &Operation{
		ID:        NewUUID(),
		Plugin:    plugin.Name(),
		BackendID: backendID,
		Namespace: msg.Header.Namespace,
		Message:   msg.Header.ID,
		Data:      nil,
		Type:      opType,
		Direction: opDir,
		Recipient: recipient,
		Status:    opStatus,
		Created:   NowMillis(),
	}
}

func NewMessageDataOp(plugin Named, backendID string, msg *Message, dataIdx int, opType OpType, opDir OpDirection, opStatus OpStatus, recipient string) *Operation {
	return &Operation{
		ID:        NewUUID(),
		Plugin:    plugin.Name(),
		BackendID: backendID,
		Namespace: msg.Header.Namespace,
		Message:   msg.Header.ID,
		Data:      msg.Data[dataIdx].ID,
		Type:      opType,
		Direction: opDir,
		Recipient: recipient,
		Status:    opStatus,
		Created:   NowMillis(),
	}
}

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
