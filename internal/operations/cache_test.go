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
package operations

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRunWithOperationCache(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	op1 := &core.Operation{
		ID:     fftypes.NewUUID(),
		Type:   core.OpTypeBlockchainPinBatch,
		Input:  fftypes.JSONObject{"batch": "1"},
		Status: core.OpStatusFailed,
	}
	op1Copy := &core.Operation{
		ID:     fftypes.NewUUID(),
		Type:   core.OpTypeBlockchainPinBatch,
		Input:  fftypes.JSONObject{"batch": "1"},
		Status: core.OpStatusPending,
	}
	op2 := &core.Operation{
		ID:     fftypes.NewUUID(),
		Type:   core.OpTypeBlockchainPinBatch,
		Input:  fftypes.JSONObject{"batch": "2"},
		Status: core.OpStatusFailed,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("InsertOperation", mock.Anything, op1).Return(nil).Once()
	mdi.On("InsertOperation", mock.Anything, op2).Return(nil).Once()

	err := RunWithOperationCache(context.Background(), func(ctx context.Context) error {
		if err := om.AddOrReuseOperation(ctx, op1); err != nil {
			return err
		}
		if err := om.AddOrReuseOperation(ctx, op1Copy); err != nil {
			return err
		}
		return om.AddOrReuseOperation(ctx, op2)
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestRunWithOperationCacheFail(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	op1 := &core.Operation{
		ID:     fftypes.NewUUID(),
		Type:   core.OpTypeBlockchainPinBatch,
		Input:  fftypes.JSONObject{"batch": "1"},
		Status: core.OpStatusFailed,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("InsertOperation", mock.Anything, op1).Return(fmt.Errorf("pop")).Once()

	err := RunWithOperationCache(context.Background(), func(ctx context.Context) error {
		return om.AddOrReuseOperation(ctx, op1)
	})
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestAddOrReuseOperationNoCache(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	op1 := &core.Operation{
		ID:     fftypes.NewUUID(),
		Type:   core.OpTypeBlockchainPinBatch,
		Input:  fftypes.JSONObject{"batch": "1"},
		Status: core.OpStatusFailed,
	}
	op2 := &core.Operation{
		ID:     fftypes.NewUUID(),
		Type:   core.OpTypeBlockchainPinBatch,
		Input:  fftypes.JSONObject{"batch": "1"},
		Status: core.OpStatusPending,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("InsertOperation", ctx, op1).Return(nil).Once()
	mdi.On("InsertOperation", ctx, op2).Return(nil).Once()

	err := om.AddOrReuseOperation(ctx, op1)
	assert.NoError(t, err)
	err = om.AddOrReuseOperation(ctx, op2)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestGetCacheKeyBadJSON(t *testing.T) {
	op := &core.Operation{
		Input: fftypes.JSONObject{
			"test": map[bool]bool{true: false},
		},
	}
	_, err := getCacheKey(op)
	assert.Error(t, err)
}
