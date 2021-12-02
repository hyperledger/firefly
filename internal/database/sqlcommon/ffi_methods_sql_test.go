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

func TestFFIMethodsE2EWithDB(t *testing.T) {

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new method entry
	contractID := fftypes.NewUUID()
	methodID := fftypes.NewUUID()
	method := &fftypes.FFIMethod{
		ID:   methodID,
		Name: "Set",
		Params: fftypes.FFIParams{
			{
				Name:         "value",
				Type:         "integer",
				InternalType: "uint256",
			},
		},
		Returns: fftypes.FFIParams{
			{
				Name:         "value",
				Type:         "integer",
				InternalType: "uint256",
			},
		},
	}

	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionFFIMethods, fftypes.ChangeEventTypeCreated, "ns", methodID).Return()
	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionFFIMethods, fftypes.ChangeEventTypeUpdated, "ns", methodID).Return()

	err := s.UpsertFFIMethod(ctx, "ns", contractID, method)
	assert.NoError(t, err)

	// Query back the method (by name)
	methodRead, err := s.GetFFIMethod(ctx, "ns", contractID, "Set")
	assert.NoError(t, err)
	assert.NotNil(t, methodRead)
	methodJson, _ := json.Marshal(&method)
	methodReadJson, _ := json.Marshal(&methodRead)
	assert.Equal(t, string(methodJson), string(methodReadJson))

	// Query back the method (by query filter)
	fb := database.FFIMethodQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("id", methodRead.ID.String()),
		fb.Eq("name", methodRead.Name),
	)
	methods, res, err := s.GetFFIMethods(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(methods))
	assert.Equal(t, int64(1), *res.TotalCount)
	methodReadJson, _ = json.Marshal(methods[0])
	assert.Equal(t, string(methodJson), string(methodReadJson))

	// Update method
	method.Params = fftypes.FFIParams{}
	err = s.UpsertFFIMethod(ctx, "ns", contractID, method)
	assert.NoError(t, err)

	// Query back the method (by name)
	methodRead, err = s.GetFFIMethod(ctx, "ns", contractID, "Set")
	assert.NoError(t, err)
	assert.NotNil(t, methodRead)
	methodJson, _ = json.Marshal(&method)
	methodReadJson, _ = json.Marshal(&methodRead)
	assert.Equal(t, string(methodJson), string(methodReadJson))

	s.callbacks.AssertExpectations(t)
}
