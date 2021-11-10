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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTokenPoolCreatedIgnore(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	mdi := am.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	txID := fftypes.NewUUID()
	operations := []*fftypes.Operation{}
	pool := &tokens.TokenPool{
		Type:          fftypes.TokenTypeFungible,
		ProtocolID:    "123",
		Key:           "0x0",
		TransactionID: txID,
		Connector:     "erc1155",
	}

	mdi.On("GetTokenPoolByProtocolID", am.ctx, "erc1155", "123").Return(nil, nil, nil)
	mdi.On("GetOperations", am.ctx, mock.Anything).Return(operations, nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := am.TokenPoolCreated(mti, pool, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestTokenPoolCreatedConfirm(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	mdi := am.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	txID := fftypes.NewUUID()
	chainPool := &tokens.TokenPool{
		Type:       fftypes.TokenTypeFungible,
		ProtocolID: "123",
		Key:        "0x0",
		Connector:  "erc1155",
	}
	storedPool := &fftypes.TokenPool{
		Namespace: "ns1",
		ID:        fftypes.NewUUID(),
		Key:       chainPool.Key,
		State:     fftypes.TokenPoolStatePending,
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   txID,
		},
	}
	storedTX := &fftypes.Transaction{
		Subject: fftypes.TransactionSubject{
			Namespace: "ns1",
			Reference: storedPool.ID,
			Signer:    storedPool.Key,
			Type:      fftypes.TransactionTypeTokenPool,
		},
	}

	mdi.On("GetTokenPoolByProtocolID", am.ctx, "erc1155", "123").Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetTokenPoolByProtocolID", am.ctx, "erc1155", "123").Return(storedPool, nil).Once()
	mdi.On("GetTransactionByID", am.ctx, txID).Return(storedTX, nil)
	mdi.On("UpsertTransaction", am.ctx, mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return *tx.Subject.Reference == *storedTX.Subject.Reference
	}), false).Return(nil)
	mdi.On("UpsertTokenPool", am.ctx, storedPool).Return(nil)
	mdi.On("InsertEvent", am.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypePoolConfirmed && *e.Reference == *storedPool.ID
	})).Return(nil)

	info := fftypes.JSONObject{"some": "info"}
	err := am.TokenPoolCreated(mti, chainPool, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestTokenPoolCreatedAlreadyConfirmed(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	mdi := am.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	txID := fftypes.NewUUID()
	chainPool := &tokens.TokenPool{
		Type:       fftypes.TokenTypeFungible,
		ProtocolID: "123",
		Key:        "0x0",
		Connector:  "erc1155",
	}
	storedPool := &fftypes.TokenPool{
		Namespace: "ns1",
		ID:        fftypes.NewUUID(),
		Key:       chainPool.Key,
		State:     fftypes.TokenPoolStateConfirmed,
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   txID,
		},
	}

	mdi.On("GetTokenPoolByProtocolID", am.ctx, "erc1155", "123").Return(storedPool, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := am.TokenPoolCreated(mti, chainPool, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestConfirmPoolTxFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	mdi := am.database.(*databasemocks.Plugin)

	txID := fftypes.NewUUID()
	storedPool := &fftypes.TokenPool{
		Namespace: "ns1",
		ID:        fftypes.NewUUID(),
		Key:       "0x0",
		State:     fftypes.TokenPoolStatePending,
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   txID,
		},
	}

	mdi.On("GetTransactionByID", am.ctx, txID).Return(nil, fmt.Errorf("pop"))

	info := fftypes.JSONObject{"some": "info"}
	err := am.confirmPool(am.ctx, storedPool, "tx1", info)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestConfirmPoolUpsertFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	mdi := am.database.(*databasemocks.Plugin)

	txID := fftypes.NewUUID()
	storedPool := &fftypes.TokenPool{
		Namespace: "ns1",
		ID:        fftypes.NewUUID(),
		Key:       "0x0",
		State:     fftypes.TokenPoolStatePending,
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   txID,
		},
	}
	storedTX := &fftypes.Transaction{
		Subject: fftypes.TransactionSubject{
			Namespace: "ns1",
			Reference: storedPool.ID,
			Signer:    storedPool.Key,
			Type:      fftypes.TransactionTypeTokenPool,
		},
	}

	mdi.On("GetTransactionByID", am.ctx, txID).Return(storedTX, nil)
	mdi.On("UpsertTransaction", am.ctx, mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return *tx.Subject.Reference == *storedTX.Subject.Reference
	}), false).Return(nil)
	mdi.On("UpsertTokenPool", am.ctx, storedPool).Return(fmt.Errorf("pop"))

	info := fftypes.JSONObject{"some": "info"}
	err := am.confirmPool(am.ctx, storedPool, "tx1", info)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestTokenPoolCreatedAnnounce(t *testing.T) {
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
	pool := &tokens.TokenPool{
		Type:          fftypes.TokenTypeFungible,
		ProtocolID:    "123",
		Key:           "0x0",
		TransactionID: txID,
		Connector:     "erc1155",
	}

	mti.On("Name").Return("mock-tokens")
	mdi.On("GetTokenPoolByProtocolID", am.ctx, "erc1155", "123").Return(nil, nil).Times(2)
	mdi.On("GetOperations", am.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop")).Once()
	mdi.On("GetOperations", am.ctx, mock.Anything).Return(operations, nil, nil).Once()
	mdi.On("UpsertOperation", am.ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Type == fftypes.OpTypeTokenAnnouncePool
	}), false).Return(nil)
	mbm.On("BroadcastTokenPool", am.ctx, "test-ns", mock.MatchedBy(func(pool *fftypes.TokenPoolAnnouncement) bool {
		return pool.Pool.Namespace == "test-ns" && pool.Pool.Name == "my-pool" && *pool.Pool.ID == *poolID
	}), false).Return(nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := am.TokenPoolCreated(mti, pool, "tx1", info)
	assert.NoError(t, err)

	mti.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestTokenPoolCreatedAnnounceBadOpInputID(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	mdi := am.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	txID := fftypes.NewUUID()
	operations := []*fftypes.Operation{
		{
			ID:    fftypes.NewUUID(),
			Type:  fftypes.OpTypeTokenCreatePool,
			Input: fftypes.JSONObject{},
		},
	}
	pool := &tokens.TokenPool{
		Type:          fftypes.TokenTypeFungible,
		ProtocolID:    "123",
		Key:           "0x0",
		TransactionID: txID,
		Connector:     "erc1155",
	}

	mdi.On("GetTokenPoolByProtocolID", am.ctx, "erc1155", "123").Return(nil, nil)
	mdi.On("GetOperations", am.ctx, mock.Anything).Return(operations, nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := am.TokenPoolCreated(mti, pool, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestTokenPoolCreatedAnnounceBadOpInputNS(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	mdi := am.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	txID := fftypes.NewUUID()
	operations := []*fftypes.Operation{
		{
			ID:   fftypes.NewUUID(),
			Type: fftypes.OpTypeTokenCreatePool,
			Input: fftypes.JSONObject{
				"id": fftypes.NewUUID().String(),
			},
		},
	}
	pool := &tokens.TokenPool{
		Type:          fftypes.TokenTypeFungible,
		ProtocolID:    "123",
		Key:           "0x0",
		TransactionID: txID,
		Connector:     "erc1155",
	}

	mdi.On("GetTokenPoolByProtocolID", am.ctx, "erc1155", "123").Return(nil, nil)
	mdi.On("GetOperations", am.ctx, mock.Anything).Return(operations, nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := am.TokenPoolCreated(mti, pool, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}
