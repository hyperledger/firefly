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

package assets

import (
	"testing"

	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTokenPoolCreatedSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	mdi := am.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}
	mbm := am.broadcast.(*broadcastmocks.Manager)

	poolID := fftypes.NewUUID()
	txID := fftypes.NewUUID()
	operations := []*fftypes.Operation{
		{
			ID: fftypes.NewUUID(),
			Input: fftypes.JSONObject{
				"id":        poolID.String(),
				"namespace": "test-ns",
				"name":      "my-pool",
			},
		},
	}

	mti.On("Name").Return("mock-tokens")
	mdi.On("GetOperations", am.ctx, mock.Anything).Return(operations, nil, nil)
	mdi.On("GetTransactionByID", mock.Anything, txID).Return(nil, nil)
	mdi.On("UpsertTransaction", am.ctx, mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenPool
	}), false).Return(nil)
	mdi.On("UpsertOperation", am.ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Type == fftypes.OpTypeTokenAnnouncePool
	}), false).Return(nil)
	mbm.On("BroadcastTokenPool", am.ctx, "test-ns", mock.MatchedBy(func(pool *fftypes.TokenPoolAnnouncement) bool {
		return pool.Namespace == "test-ns" && pool.Name == "my-pool" && *pool.ID == *poolID
	}), false).Return(nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := am.TokenPoolCreated(mti, fftypes.TokenTypeFungible, txID, "123", "0x0", "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestTokenPoolCreatedOpNotFound(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	mdi := am.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}
	mbm := am.broadcast.(*broadcastmocks.Manager)

	txID := fftypes.NewUUID()
	operations := []*fftypes.Operation{}

	mti.On("Name").Return("mock-tokens")
	mdi.On("GetOperations", am.ctx, mock.Anything).Return(operations, nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := am.TokenPoolCreated(mti, fftypes.TokenTypeFungible, txID, "123", "0x0", "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestTokenPoolMissingID(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	mdi := am.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}
	mbm := am.broadcast.(*broadcastmocks.Manager)

	txID := fftypes.NewUUID()
	operations := []*fftypes.Operation{
		{
			ID:    fftypes.NewUUID(),
			Input: fftypes.JSONObject{},
		},
	}

	mti.On("Name").Return("mock-tokens")
	mdi.On("GetOperations", am.ctx, mock.Anything).Return(operations, nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := am.TokenPoolCreated(mti, fftypes.TokenTypeFungible, txID, "123", "0x0", "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestTokenPoolCreatedMissingNamespace(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	mdi := am.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}
	mbm := am.broadcast.(*broadcastmocks.Manager)

	poolID := fftypes.NewUUID()
	txID := fftypes.NewUUID()
	operations := []*fftypes.Operation{
		{
			ID: fftypes.NewUUID(),
			Input: fftypes.JSONObject{
				"id": poolID.String(),
			},
		},
	}

	mti.On("Name").Return("mock-tokens")
	mdi.On("GetOperations", am.ctx, mock.Anything).Return(operations, nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := am.TokenPoolCreated(mti, fftypes.TokenTypeFungible, txID, "123", "0x0", "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestTokenPoolCreatedUpsertFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	mdi := am.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}
	mbm := am.broadcast.(*broadcastmocks.Manager)

	poolID := fftypes.NewUUID()
	txID := fftypes.NewUUID()
	operations := []*fftypes.Operation{
		{
			ID: fftypes.NewUUID(),
			Input: fftypes.JSONObject{
				"id":        poolID.String(),
				"namespace": "test-ns",
				"name":      "my-pool",
			},
		},
	}

	mti.On("Name").Return("mock-tokens")
	mdi.On("GetOperations", am.ctx, mock.Anything).Return(operations, nil, nil)
	mdi.On("GetTransactionByID", mock.Anything, txID).Return(nil, nil)
	mdi.On("UpsertTransaction", am.ctx, mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenPool
	}), false).Return(database.HashMismatch)

	info := fftypes.JSONObject{"some": "info"}
	err := am.TokenPoolCreated(mti, fftypes.TokenTypeFungible, txID, "123", "0x0", "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mbm.AssertExpectations(t)
}
