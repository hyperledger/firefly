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

package events

import (
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/assetmocks"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTokenPoolCreatedIgnore(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
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

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil, nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(operations, nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokenPoolCreated(mti, pool, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestTokenPoolCreatedIgnoreNoTX(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	pool := &tokens.TokenPool{
		Type:          fftypes.TokenTypeFungible,
		ProtocolID:    "123",
		Key:           "0x0",
		TransactionID: nil,
		Connector:     "erc1155",
	}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokenPoolCreated(mti, pool, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestTokenPoolCreatedConfirm(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
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
		Message:   fftypes.NewUUID(),
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
	storedMessage := &fftypes.Message{
		BatchID: fftypes.NewUUID(),
	}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(storedPool, nil).Times(2)
	mdi.On("GetTransactionByID", em.ctx, txID).Return(storedTX, nil)
	mdi.On("UpsertTransaction", em.ctx, mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return *tx.Subject.Reference == *storedTX.Subject.Reference
	}), false).Return(nil)
	mdi.On("UpsertTokenPool", em.ctx, storedPool).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypePoolConfirmed && *e.Reference == *storedPool.ID
	})).Return(nil)
	mdi.On("GetMessageByID", em.ctx, storedPool.Message).Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetMessageByID", em.ctx, storedPool.Message).Return(storedMessage, nil).Once()

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokenPoolCreated(mti, chainPool, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestTokenPoolCreatedAlreadyConfirmed(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
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

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(storedPool, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokenPoolCreated(mti, chainPool, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestTokenPoolCreatedMigrate(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mam := em.assets.(*assetmocks.Manager)
	mti := &tokenmocks.Plugin{}

	txID := fftypes.NewUUID()
	chainPool := &tokens.TokenPool{
		Type:       fftypes.TokenTypeFungible,
		ProtocolID: "123",
		Key:        "0x0",
		Connector:  "magic-tokens",
	}
	storedPool := &fftypes.TokenPool{
		Namespace: "ns1",
		ID:        fftypes.NewUUID(),
		Key:       chainPool.Key,
		State:     fftypes.TokenPoolStateUnknown,
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
	storedMessage := &fftypes.Message{
		BatchID: fftypes.NewUUID(),
	}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "magic-tokens", "123").Return(storedPool, nil).Times(3)
	mdi.On("GetTransactionByID", em.ctx, storedPool.TX.ID).Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetTransactionByID", em.ctx, storedPool.TX.ID).Return(storedTX, nil).Times(3)
	mdi.On("UpsertTransaction", em.ctx, mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return *tx.Subject.Reference == *storedTX.Subject.Reference
	}), false).Return(nil).Once()
	mdi.On("UpsertTokenPool", em.ctx, storedPool).Return(nil).Once()
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypePoolConfirmed && *e.Reference == *storedPool.ID
	})).Return(nil).Once()
	mam.On("ActivateTokenPool", em.ctx, storedPool, storedTX).Return(fmt.Errorf("pop")).Once()
	mam.On("ActivateTokenPool", em.ctx, storedPool, storedTX).Return(nil).Once()
	mdi.On("GetMessageByID", em.ctx, storedPool.Message).Return(storedMessage, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokenPoolCreated(mti, chainPool, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mam.AssertExpectations(t)
}

func TestConfirmPoolTxFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)

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

	mdi.On("GetTransactionByID", em.ctx, txID).Return(nil, fmt.Errorf("pop"))

	info := fftypes.JSONObject{"some": "info"}
	err := em.confirmPool(em.ctx, storedPool, "tx1", info)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestConfirmPoolUpsertFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)

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

	mdi.On("GetTransactionByID", em.ctx, txID).Return(storedTX, nil)
	mdi.On("UpsertTransaction", em.ctx, mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return *tx.Subject.Reference == *storedTX.Subject.Reference
	}), false).Return(nil)
	mdi.On("UpsertTokenPool", em.ctx, storedPool).Return(fmt.Errorf("pop"))

	info := fftypes.JSONObject{"some": "info"}
	err := em.confirmPool(em.ctx, storedPool, "tx1", info)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestTokenPoolCreatedAnnounce(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}
	mbm := em.broadcast.(*broadcastmocks.Manager)

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
	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil).Times(2)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop")).Once()
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(operations, nil, nil).Once()
	mdi.On("InsertOperation", em.ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Type == fftypes.OpTypeTokenAnnouncePool
	})).Return(nil)
	mbm.On("BroadcastTokenPool", em.ctx, "test-ns", mock.MatchedBy(func(pool *fftypes.TokenPoolAnnouncement) bool {
		return pool.Pool.Namespace == "test-ns" && pool.Pool.Name == "my-pool" && *pool.Pool.ID == *poolID
	}), false).Return(nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokenPoolCreated(mti, pool, "tx1", info)
	assert.NoError(t, err)

	mti.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestTokenPoolCreatedAnnounceBadOpInputID(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
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

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(operations, nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokenPoolCreated(mti, pool, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestTokenPoolCreatedAnnounceBadOpInputNS(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
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

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(operations, nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokenPoolCreated(mti, pool, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}
