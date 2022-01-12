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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBatchManifest(t *testing.T) {

	tw := TransportWrapper{
		Type: TransportPayloadTypeBatch,
		Batch: &Batch{
			Payload: BatchPayload{
				Messages: []*Message{
					{Header: MessageHeader{ID: NewUUID()}, Hash: NewRandB32()},
					{Header: MessageHeader{ID: NewUUID()}, Hash: NewRandB32()},
				},
				Data: []*Data{
					{ID: NewUUID(), Hash: NewRandB32()},
					{ID: NewUUID(), Hash: NewRandB32()},
				},
			},
		},
	}
	tm := tw.Manifest()
	assert.Equal(t, 2, len(tm.Messages))
	assert.Equal(t, tw.Batch.Payload.Messages[0].Header.ID.String(), tm.Messages[0].ID.String())
	assert.Equal(t, tw.Batch.Payload.Messages[1].Header.ID.String(), tm.Messages[1].ID.String())
	assert.Equal(t, tw.Batch.Payload.Messages[0].Hash.String(), tm.Messages[0].Hash.String())
	assert.Equal(t, tw.Batch.Payload.Messages[1].Hash.String(), tm.Messages[1].Hash.String())
	assert.Equal(t, 2, len(tm.Data))
	assert.Equal(t, tw.Batch.Payload.Data[0].ID.String(), tm.Data[0].ID.String())
	assert.Equal(t, tw.Batch.Payload.Data[1].ID.String(), tm.Data[1].ID.String())
	assert.Equal(t, tw.Batch.Payload.Data[0].Hash.String(), tm.Data[0].Hash.String())
	assert.Equal(t, tw.Batch.Payload.Data[1].Hash.String(), tm.Data[1].Hash.String())

}

func TestSingleMessageManifest(t *testing.T) {

	tw := TransportWrapper{
		Type:    TransportPayloadTypeMessage,
		Message: &Message{Header: MessageHeader{ID: NewUUID()}, Hash: NewRandB32()},
		Data: []*Data{
			{ID: NewUUID(), Hash: NewRandB32()},
			{ID: NewUUID(), Hash: NewRandB32()},
		},
	}
	tm := tw.Manifest()
	assert.Equal(t, 1, len(tm.Messages))
	assert.Equal(t, tw.Message.Header.ID.String(), tm.Messages[0].ID.String())
	assert.Equal(t, tw.Message.Hash.String(), tm.Messages[0].Hash.String())
	assert.Equal(t, 2, len(tm.Data))
	assert.Equal(t, tw.Data[0].ID.String(), tm.Data[0].ID.String())
	assert.Equal(t, tw.Data[1].ID.String(), tm.Data[1].ID.String())
	assert.Equal(t, tw.Data[0].Hash.String(), tm.Data[0].Hash.String())
	assert.Equal(t, tw.Data[1].Hash.String(), tm.Data[1].Hash.String())

}

func TestUnknownManifest(t *testing.T) {

	tw := TransportWrapper{}
	assert.Nil(t, tw.Manifest())

}

func TestNillBatchManifest(t *testing.T) {

	tw := TransportWrapper{
		Type: TransportPayloadTypeBatch,
	}
	assert.Nil(t, tw.Manifest())

}
