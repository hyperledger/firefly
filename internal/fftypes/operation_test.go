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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakePlugin struct{}

func (f *fakePlugin) Name() string { return "fake" }

func TestNewPendingMessageOp(t *testing.T) {
	msg := &Message{
		Header: MessageHeader{
			ID:        NewUUID(),
			Namespace: "ns1",
		},
	}
	op := NewMessageOp(&fakePlugin{}, "testBackend", msg, OpTypePublicStorageBatchBroadcast, OpDirectionOutbound, OpStatusPending, "recipient")
	assert.Equal(t, Operation{
		ID:        op.ID,
		Plugin:    "fake",
		BackendID: "testBackend",
		Namespace: "ns1",
		Message:   msg.Header.ID,
		Data:      nil,
		Type:      OpTypePublicStorageBatchBroadcast,
		Direction: OpDirectionOutbound,
		Recipient: "recipient",
		Status:    OpStatusPending,
		Created:   op.Created,
	}, *op)
}

func TestMessageDataOp(t *testing.T) {
	msg := &Message{
		Header: MessageHeader{
			ID:        NewUUID(),
			Namespace: "ns1",
		},
		Data: DataRefs{
			{ID: NewUUID(), Hash: NewRandB32()},
		},
	}
	op := NewMessageDataOp(&fakePlugin{}, "testBackend", msg, 0, OpTypePublicStorageBatchBroadcast, OpDirectionOutbound, OpStatusSucceeded, "recipient")
	assert.Equal(t, Operation{
		ID:        op.ID,
		Plugin:    "fake",
		BackendID: "testBackend",
		Namespace: "ns1",
		Message:   msg.Header.ID,
		Data:      msg.Data[0].ID,
		Type:      OpTypePublicStorageBatchBroadcast,
		Direction: OpDirectionOutbound,
		Recipient: "recipient",
		Status:    OpStatusSucceeded,
		Created:   op.Created,
	}, *op)
}
