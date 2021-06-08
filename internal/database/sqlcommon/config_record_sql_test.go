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

	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestConfigRecordE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s := newQLTestProvider(t)
	defer s.Close()
	ctx := context.Background()

	// Create a new namespace entry
	configRecord := &fftypes.ConfigRecord{
		Key:   "foo",
		Value: fftypes.Byteable(`{"foo":"bar"}`),
	}
	err := s.UpsertConfigRecord(ctx, configRecord, true)
	assert.NoError(t, err)

	// Check we get the exact same config record back
	configRecordRead, err := s.GetConfigRecord(ctx, configRecord.Key)
	assert.NoError(t, err)
	assert.NotNil(t, configRecordRead)
	configRecordJson, _ := json.Marshal(&configRecord)
	configRecordReadJson, _ := json.Marshal(&configRecordRead)
	assert.Equal(t, string(configRecordJson), string(configRecordReadJson))

	// Update the config record (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	configRecordUpdated := &fftypes.ConfigRecord{
		Key:   "foo",
		Value: fftypes.Byteable(`{"fiz":"buzz"}`),
	}
	err = s.UpsertConfigRecord(context.Background(), configRecordUpdated, true)
	assert.NoError(t, err)

	// Check we get the exact same data back
	configRecordRead, err = s.GetConfigRecord(ctx, configRecord.Key)
	assert.NoError(t, err)
	configRecordJson, _ = json.Marshal(&configRecordUpdated)
	configRecordReadJson, _ = json.Marshal(&configRecordRead)
	assert.Equal(t, string(configRecordJson), string(configRecordReadJson))

	// Query back the config record
	fb := database.ConfigRecordQueryFactory.NewFilter(ctx)
	filter := fb.And()
	configRecordRes, err := s.GetConfigRecords(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configRecordRes))
	configRecordReadJson, _ = json.Marshal(configRecordRes[0])
	assert.Equal(t, string(configRecordJson), string(configRecordReadJson))
}
