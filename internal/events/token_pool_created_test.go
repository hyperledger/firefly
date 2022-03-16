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

	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/mocks/assetmocks"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
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
	info := fftypes.JSONObject{"some": "info"}
	pool := &tokens.TokenPool{
		Type:          fftypes.TokenTypeFungible,
		ProtocolID:    "123",
		TransactionID: txID,
		Connector:     "erc1155",
		Event: blockchain.Event{
			BlockchainTXID: "0xffffeeee",
			ProtocolID:     "tx1",
			Info:           info,
		},
	}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil, nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(operations, nil, nil)

	err := em.TokenPoolCreated(mti, pool)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestTokenPoolCreatedIgnoreNoTX(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	info := fftypes.JSONObject{"some": "info"}
	pool := &tokens.TokenPool{
		Type:          fftypes.TokenTypeFungible,
		ProtocolID:    "123",
		TransactionID: nil,
		Connector:     "erc1155",
		Event: blockchain.Event{
			BlockchainTXID: "0xffffeeee",
			ProtocolID:     "tx1",
			Info:           info,
		},
	}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil, nil)

	err := em.TokenPoolCreated(mti, pool)
	assert.NoError(t, err)
}

func TestTokenPoolCreatedConfirm(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}
	mth := em.txHelper.(*txcommonmocks.Helper)
	mdm := em.data.(*datamocks.Manager)

	opID := fftypes.NewUUID()
	txID := fftypes.NewUUID()
	info1 := fftypes.JSONObject{"pool": "info"}
	info2 := fftypes.JSONObject{"block": "info"}
	chainPool := &tokens.TokenPool{
		Type:          fftypes.TokenTypeFungible,
		ProtocolID:    "123",
		Connector:     "erc1155",
		TransactionID: txID,
		Standard:      "ERC1155",
		Info:          info1,
		Event: blockchain.Event{
			BlockchainTXID: "0xffffeeee",
			Name:           "TokenPool",
			ProtocolID:     "tx1",
			Info:           info2,
		},
	}
	storedPool := &fftypes.TokenPool{
		Namespace: "ns1",
		ID:        fftypes.NewUUID(),
		State:     fftypes.TokenPoolStatePending,
		Message:   fftypes.NewUUID(),
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   txID,
		},
	}
	storedMessage := &fftypes.Message{
		BatchID: fftypes.NewUUID(),
	}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(storedPool, nil).Times(2)
	mdi.On("InsertBlockchainEvent", em.ctx, mock.MatchedBy(func(e *fftypes.BlockchainEvent) bool {
		return e.Name == chainPool.Event.Name
	})).Return(nil).Once()
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeBlockchainEventReceived
	})).Return(nil).Once()
	mdi.On("GetOperations", em.ctx, mock.Anything).Return([]*fftypes.Operation{{
		ID: opID,
	}}, nil, nil)
	mdi.On("ResolveOperation", em.ctx, opID, fftypes.OpStatusSucceeded, "", mock.Anything).Return(nil)
	mth.On("PersistTransaction", mock.Anything, "ns1", txID, fftypes.TransactionTypeTokenPool, "0xffffeeee").Return(true, nil).Once()
	mdi.On("UpsertTokenPool", em.ctx, storedPool).Return(nil).Once()
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypePoolConfirmed && *e.Reference == *storedPool.ID
	})).Return(nil).Once()
	mdm.On("GetMessageWithDataCached", em.ctx, storedPool.Message, data.CRORequireBatchID).Return(nil, nil, false, fmt.Errorf("pop")).Once()
	mdm.On("GetMessageWithDataCached", em.ctx, storedPool.Message, data.CRORequireBatchID).Return(storedMessage, nil, true, nil).Once()

	err := em.TokenPoolCreated(mti, chainPool)
	assert.NoError(t, err)

	assert.Equal(t, "ERC1155", storedPool.Standard)
	assert.Equal(t, info1, storedPool.Info)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestTokenPoolCreatedAlreadyConfirmed(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	txID := fftypes.NewUUID()
	info := fftypes.JSONObject{"some": "info"}
	chainPool := &tokens.TokenPool{
		Type:          fftypes.TokenTypeFungible,
		ProtocolID:    "123",
		Connector:     "erc1155",
		TransactionID: txID,
		Event: blockchain.Event{
			BlockchainTXID: "0xffffeeee",
			ProtocolID:     "tx1",
			Info:           info,
		},
	}
	storedPool := &fftypes.TokenPool{
		Namespace: "ns1",
		ID:        fftypes.NewUUID(),
		State:     fftypes.TokenPoolStateConfirmed,
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   txID,
		},
	}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(storedPool, nil)

	err := em.TokenPoolCreated(mti, chainPool)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestTokenPoolCreatedMigrate(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mam := em.assets.(*assetmocks.Manager)
	mti := &tokenmocks.Plugin{}
	mth := em.txHelper.(*txcommonmocks.Helper)
	mdm := em.data.(*datamocks.Manager)

	txID := fftypes.NewUUID()
	info := fftypes.JSONObject{"some": "info"}
	chainPool := &tokens.TokenPool{
		Type:          fftypes.TokenTypeFungible,
		ProtocolID:    "123",
		Connector:     "magic-tokens",
		TransactionID: txID,
		Event: blockchain.Event{
			BlockchainTXID: "0xffffeeee",
			ProtocolID:     "tx1",
			Info:           info,
		},
	}
	storedPool := &fftypes.TokenPool{
		Namespace: "ns1",
		ID:        fftypes.NewUUID(),
		State:     fftypes.TokenPoolStateUnknown,
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   txID,
		},
	}
	storedMessage := &fftypes.Message{
		BatchID: fftypes.NewUUID(),
	}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "magic-tokens", "123").Return(storedPool, nil).Times(2)
	mdi.On("InsertBlockchainEvent", em.ctx, mock.MatchedBy(func(e *fftypes.BlockchainEvent) bool {
		return e.Name == chainPool.Event.Name
	})).Return(nil).Once()
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeBlockchainEventReceived
	})).Return(nil).Once()
	mdi.On("GetOperations", em.ctx, mock.Anything).Return([]*fftypes.Operation{}, nil, nil)
	mth.On("PersistTransaction", mock.Anything, "ns1", txID, fftypes.TransactionTypeTokenPool, "0xffffeeee").Return(true, nil).Once()
	mdi.On("UpsertTokenPool", em.ctx, storedPool).Return(nil).Once()
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypePoolConfirmed && *e.Reference == *storedPool.ID
	})).Return(nil).Once()
	mam.On("ActivateTokenPool", em.ctx, storedPool, info).Return(fmt.Errorf("pop")).Once()
	mam.On("ActivateTokenPool", em.ctx, storedPool, info).Return(nil).Once()
	mdm.On("GetMessageWithDataCached", em.ctx, storedPool.Message, data.CRORequireBatchID).Return(storedMessage, nil, true, nil).Once()

	err := em.TokenPoolCreated(mti, chainPool)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mam.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestConfirmPoolBlockchainEventFail(t *testing.T) {
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
	event := &blockchain.Event{
		Name: "TokenPool",
	}

	mdi.On("InsertBlockchainEvent", em.ctx, mock.MatchedBy(func(e *fftypes.BlockchainEvent) bool {
		return e.Name == event.Name
	})).Return(fmt.Errorf("pop"))

	err := em.confirmPool(em.ctx, storedPool, event, "0xffffeeee")
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestConfirmPoolGetOpsFail(t *testing.T) {
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
	event := &blockchain.Event{
		Name: "TokenPool",
	}

	mdi.On("InsertBlockchainEvent", em.ctx, mock.MatchedBy(func(e *fftypes.BlockchainEvent) bool {
		return e.Name == event.Name
	})).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeBlockchainEventReceived
	})).Return(nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	err := em.confirmPool(em.ctx, storedPool, event, "0xffffeeee")
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestConfirmPoolResolveOpFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)

	opID := fftypes.NewUUID()
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
	event := &blockchain.Event{
		Name: "TokenPool",
	}

	mdi.On("InsertBlockchainEvent", em.ctx, mock.MatchedBy(func(e *fftypes.BlockchainEvent) bool {
		return e.Name == event.Name
	})).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeBlockchainEventReceived
	})).Return(nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return([]*fftypes.Operation{{
		ID: opID,
	}}, nil, nil)
	mdi.On("ResolveOperation", em.ctx, opID, fftypes.OpStatusSucceeded, "", mock.Anything).Return(fmt.Errorf("pop"))

	err := em.confirmPool(em.ctx, storedPool, event, "0xffffeeee")
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestConfirmPoolTxFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mth := em.txHelper.(*txcommonmocks.Helper)

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
	event := &blockchain.Event{
		Name: "TokenPool",
	}

	mdi.On("InsertBlockchainEvent", em.ctx, mock.MatchedBy(func(e *fftypes.BlockchainEvent) bool {
		return e.Name == event.Name
	})).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeBlockchainEventReceived
	})).Return(nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return([]*fftypes.Operation{}, nil, nil)
	mth.On("PersistTransaction", mock.Anything, "ns1", txID, fftypes.TransactionTypeTokenPool, "0xffffeeee").Return(false, fmt.Errorf("pop"))

	err := em.confirmPool(em.ctx, storedPool, event, "0xffffeeee")
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestConfirmPoolUpsertFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mth := em.txHelper.(*txcommonmocks.Helper)

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
	event := &blockchain.Event{
		Name: "TokenPool",
	}

	mdi.On("InsertBlockchainEvent", em.ctx, mock.MatchedBy(func(e *fftypes.BlockchainEvent) bool {
		return e.Name == event.Name
	})).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeBlockchainEventReceived
	})).Return(nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return([]*fftypes.Operation{}, nil, nil)
	mth.On("PersistTransaction", mock.Anything, "ns1", txID, fftypes.TransactionTypeTokenPool, "0xffffeeee").Return(true, nil).Once()
	mdi.On("UpsertTokenPool", em.ctx, storedPool).Return(fmt.Errorf("pop"))

	err := em.confirmPool(em.ctx, storedPool, event, "0xffffeeee")
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
	info := fftypes.JSONObject{"some": "info"}
	pool := &tokens.TokenPool{
		Type:          fftypes.TokenTypeFungible,
		ProtocolID:    "123",
		TransactionID: txID,
		Connector:     "erc1155",
		Event: blockchain.Event{
			BlockchainTXID: "0xffffeeee",
			ProtocolID:     "tx1",
			Info:           info,
		},
	}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil).Times(2)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop")).Once()
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(operations, nil, nil).Once()
	mbm.On("BroadcastTokenPool", em.ctx, "test-ns", mock.MatchedBy(func(pool *fftypes.TokenPoolAnnouncement) bool {
		return pool.Pool.Namespace == "test-ns" && pool.Pool.Name == "my-pool" && *pool.Pool.ID == *poolID
	}), false).Return(nil, nil)

	err := em.TokenPoolCreated(mti, pool)
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
	info := fftypes.JSONObject{"some": "info"}
	pool := &tokens.TokenPool{
		Type:          fftypes.TokenTypeFungible,
		ProtocolID:    "123",
		TransactionID: txID,
		Connector:     "erc1155",
		Event: blockchain.Event{
			BlockchainTXID: "0xffffeeee",
			ProtocolID:     "tx1",
			Info:           info,
		},
	}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(operations, nil, nil)

	err := em.TokenPoolCreated(mti, pool)
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
	info := fftypes.JSONObject{"some": "info"}
	pool := &tokens.TokenPool{
		Type:          fftypes.TokenTypeFungible,
		ProtocolID:    "123",
		TransactionID: txID,
		Connector:     "erc1155",
		Event: blockchain.Event{
			BlockchainTXID: "0xffffeeee",
			ProtocolID:     "tx1",
			Info:           info,
		},
	}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(operations, nil, nil)

	err := em.TokenPoolCreated(mti, pool)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}
