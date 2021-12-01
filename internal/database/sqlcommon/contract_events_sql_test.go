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

package sqlcommon

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestContractEventsE2EWithDB(t *testing.T) {

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new event entry
	contractID := fftypes.NewUUID()
	eventID := fftypes.NewUUID()
	event := &fftypes.FFIEvent{
		ID:   eventID,
		Name: "Changed",
		Params: fftypes.FFIParams{
			{
				Name:         "value",
				Type:         "integer",
				InternalType: "uint256",
			},
		},
	}

	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionContractEvents, fftypes.ChangeEventTypeCreated, "ns", eventID).Return()

	err := s.UpsertContractEvent(ctx, "ns", contractID, event)
	assert.NoError(t, err)

	// Query back the event (by name)
	eventRead, err := s.GetContractEventByName(ctx, "ns", contractID, "Changed")
	assert.NoError(t, err)
	assert.NotNil(t, eventRead)
	eventJson, _ := json.Marshal(&event)
	eventReadJson, _ := json.Marshal(&eventRead)
	assert.Equal(t, string(eventJson), string(eventReadJson))

	// Query back the event (by query filter)
	fb := database.ContractEventQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("id", eventRead.ID.String()),
		fb.Eq("name", eventRead.Name),
	)
	events, res, err := s.GetContractEvents(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, int64(1), *res.TotalCount)
	eventReadJson, _ = json.Marshal(events[0])
	assert.Equal(t, string(eventJson), string(eventReadJson))

	s.callbacks.AssertExpectations(t)
}
