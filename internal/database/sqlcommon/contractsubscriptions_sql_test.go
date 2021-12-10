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

func TestContractSubscriptionE2EWithDB(t *testing.T) {
	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new contract subscription entry
	location := fftypes.JSONObject{"path": "my-api"}
	locationJson, _ := json.Marshal(location)
	sub := &fftypes.ContractSubscription{
		ID:         fftypes.NewUUID(),
		Interface:  fftypes.NewUUID(),
		Event:      fftypes.NewUUID(),
		Namespace:  "ns",
		ProtocolID: "sb-123",
		Location:   locationJson,
	}
	subJson, _ := json.Marshal(&sub)

	err := s.UpsertContractSubscription(ctx, sub)
	assert.NoError(t, err)

	// Query back the subscription (by query filter)
	fb := database.ContractSubscriptionQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("protocolid", sub.ProtocolID),
	)
	subs, res, err := s.GetContractSubscriptions(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(subs))
	assert.Equal(t, int64(1), *res.TotalCount)
	subReadJson, _ := json.Marshal(subs[0])
	assert.Equal(t, string(subJson), string(subReadJson))

	// Update the subscription
	sub.Location = []byte("{}")
	subJson, _ = json.Marshal(&sub)
	err = s.UpsertContractSubscription(ctx, sub)
	assert.NoError(t, err)

	// Query back the subscription (by query filter)
	filter = fb.And(
		fb.Eq("protocolid", sub.ProtocolID),
	)
	subs, res, err = s.GetContractSubscriptions(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(subs))
	assert.Equal(t, int64(1), *res.TotalCount)
	subReadJson, _ = json.Marshal(subs[0])
	assert.Equal(t, string(subJson), string(subReadJson))
}
