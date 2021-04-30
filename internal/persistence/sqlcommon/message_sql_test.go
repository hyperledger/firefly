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

package sqlcommon

import (
	"context"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestUpsertMessage(t *testing.T) {

	s := &SQLCommon{}
	ctx := context.Background()
	InitSQLCommon(ctx, s, ensureTestDB(t), nil)

	// Create a new message
	msgId := uuid.New()
	dataId1 := uuid.New()
	dataId2 := uuid.New()
	randB32 := fftypes.NewRandB32()
	msg := &fftypes.MessageRefsOnly{
		MessageBase: fftypes.MessageBase{
			ID:        &msgId,
			CID:       nil,
			Type:      fftypes.MessageTypeBroadcast,
			Author:    "0x12345",
			Created:   time.Now().UnixNano(),
			Namespace: "ns12345",
			Topic:     "topic1",
			Context:   "context1",
			Group:     nil,
			DataHash:  &randB32,
			Hash:      &randB32,
			Confirmed: 0,
		},
		TX: nil,
		Data: []fftypes.DataRef{
			{ID: &dataId1},
			{ID: &dataId2},
		},
	}
	err := s.UpsertMessage(ctx, msg)
	assert.NoError(t, err)

	// Check we get the exact same message back - note data gets sorted automatically on retrieve
	sort.Sort(msg.Data)
	msgRead, err := s.GetMessageById(ctx, &msgId)
	assert.NoError(t, err)
	msgJson, _ := json.Marshal(&msg)
	msgReadJson, _ := json.Marshal(&msgRead)
	assert.Equal(t, string(msgJson), string(msgReadJson))

	// Update the message (this is testing what's possible at the persistence layer,
	// and does not account for the verification that happens at the higher level)
	dataId3 := uuid.New()
	cid := uuid.New()
	gid := uuid.New()
	msgUpdated := &fftypes.MessageRefsOnly{
		MessageBase: fftypes.MessageBase{
			ID:        &msgId,
			CID:       &cid,
			Type:      fftypes.MessageTypeBroadcast,
			Author:    "0x12345",
			Created:   time.Now().UnixNano(),
			Namespace: "ns12345",
			Topic:     "topic1",
			Context:   "context1",
			Group:     &gid,
			DataHash:  &randB32,
			Hash:      &randB32,
			Confirmed: time.Now().UnixNano(),
		},
		TX: nil,
		Data: []fftypes.DataRef{
			{ID: &dataId1},
			{ID: &dataId3},
		},
	}
	err = s.UpsertMessage(context.Background(), msgUpdated)
	assert.NoError(t, err)

	// Check we get the exact same message back - note the removal of one of the data elements
	sort.Sort(msgUpdated.Data)
	msgRead, err = s.GetMessageById(ctx, &msgId)
	assert.NoError(t, err)
	msgJson, _ = json.Marshal(&msgUpdated)
	msgReadJson, _ = json.Marshal(&msgRead)
	assert.Equal(t, string(msgJson), string(msgReadJson))

}
