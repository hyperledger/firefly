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

package syshandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/assetmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleSystemBroadcastTokenPoolSelfOk(t *testing.T) {
	sh := newTestSystemHandlers(t)

	pool := &fftypes.TokenPool{
		ID:         fftypes.NewUUID(),
		Namespace:  "ns1",
		Name:       "name1",
		Type:       fftypes.TokenTypeFungible,
		ProtocolID: "12345",
		Symbol:     "COIN",
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   fftypes.NewUUID(),
		},
	}
	announce := &fftypes.TokenPoolAnnouncement{
		Pool:         pool,
		ProtocolTxID: "tx123",
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: string(fftypes.SystemTagDefinePool),
		},
	}
	b, err := json.Marshal(&announce)
	assert.NoError(t, err)
	data := []*fftypes.Data{{
		Value: fftypes.Byteable(b),
	}}
	opID := fftypes.NewUUID()
	operations := []*fftypes.Operation{{ID: opID}}
	tx := &fftypes.Transaction{ProtocolID: "tx123"}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", context.Background(), mock.Anything).Return(operations, nil, nil)
	mdi.On("UpdateOperation", context.Background(), opID, mock.Anything).Return(nil)
	mdi.On("GetTransactionByID", context.Background(), pool.TX.ID).Return(tx, nil)
	mdi.On("UpsertTransaction", context.Background(), tx, false).Return(nil)
	mdi.On("UpsertTokenPool", context.Background(), mock.MatchedBy(func(p *fftypes.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(nil)
	mdi.On("InsertEvent", context.Background(), mock.MatchedBy(func(event *fftypes.Event) bool {
		return *event.Reference == *pool.ID && event.Namespace == pool.Namespace && event.Type == fftypes.EventTypePoolConfirmed
	})).Return(nil)

	valid, err := sh.HandleSystemBroadcast(context.Background(), msg, data)
	assert.True(t, valid)
	assert.NoError(t, err)
	assert.Equal(t, fftypes.OpStatusSucceeded, tx.Status)

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastTokenPoolSelfUpdateOpFail(t *testing.T) {
	sh := newTestSystemHandlers(t)

	announce := &fftypes.TokenPoolAnnouncement{
		Pool: &fftypes.TokenPool{
			ID:         fftypes.NewUUID(),
			Namespace:  "ns1",
			Name:       "name1",
			Type:       fftypes.TokenTypeFungible,
			ProtocolID: "12345",
			Symbol:     "COIN",
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeTokenPool,
				ID:   fftypes.NewUUID(),
			},
		},
		ProtocolTxID: "tx123",
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: string(fftypes.SystemTagDefinePool),
		},
	}
	b, err := json.Marshal(&announce)
	assert.NoError(t, err)
	data := []*fftypes.Data{{
		Value: fftypes.Byteable(b),
	}}
	opID := fftypes.NewUUID()
	operations := []*fftypes.Operation{{ID: opID}}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", context.Background(), mock.Anything).Return(operations, nil, nil)
	mdi.On("UpdateOperation", context.Background(), opID, mock.Anything).Return(fmt.Errorf("pop"))

	valid, err := sh.HandleSystemBroadcast(context.Background(), msg, data)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastTokenPoolSelfGetTXFail(t *testing.T) {
	sh := newTestSystemHandlers(t)

	pool := &fftypes.TokenPool{
		ID:         fftypes.NewUUID(),
		Namespace:  "ns1",
		Name:       "name1",
		Type:       fftypes.TokenTypeFungible,
		ProtocolID: "12345",
		Symbol:     "COIN",
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   fftypes.NewUUID(),
		},
	}
	announce := &fftypes.TokenPoolAnnouncement{
		Pool:         pool,
		ProtocolTxID: "tx123",
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: string(fftypes.SystemTagDefinePool),
		},
	}
	b, err := json.Marshal(&announce)
	assert.NoError(t, err)
	data := []*fftypes.Data{{
		Value: fftypes.Byteable(b),
	}}
	opID := fftypes.NewUUID()
	operations := []*fftypes.Operation{{ID: opID}}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", context.Background(), mock.Anything).Return(operations, nil, nil)
	mdi.On("UpdateOperation", context.Background(), opID, mock.Anything).Return(nil)
	mdi.On("GetTransactionByID", context.Background(), pool.TX.ID).Return(nil, fmt.Errorf("pop"))

	valid, err := sh.HandleSystemBroadcast(context.Background(), msg, data)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastTokenPoolSelfTXMismatch(t *testing.T) {
	sh := newTestSystemHandlers(t)

	pool := &fftypes.TokenPool{
		ID:         fftypes.NewUUID(),
		Namespace:  "ns1",
		Name:       "name1",
		Type:       fftypes.TokenTypeFungible,
		ProtocolID: "12345",
		Symbol:     "COIN",
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   fftypes.NewUUID(),
		},
	}
	announce := &fftypes.TokenPoolAnnouncement{
		Pool:         pool,
		ProtocolTxID: "tx123",
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: string(fftypes.SystemTagDefinePool),
		},
	}
	b, err := json.Marshal(&announce)
	assert.NoError(t, err)
	data := []*fftypes.Data{{
		Value: fftypes.Byteable(b),
	}}
	opID := fftypes.NewUUID()
	operations := []*fftypes.Operation{{ID: opID}}
	tx := &fftypes.Transaction{ProtocolID: "bad"}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", context.Background(), mock.Anything).Return(operations, nil, nil)
	mdi.On("UpdateOperation", context.Background(), opID, mock.Anything).Return(nil)
	mdi.On("GetTransactionByID", context.Background(), pool.TX.ID).Return(tx, nil)
	mdi.On("InsertEvent", context.Background(), mock.MatchedBy(func(event *fftypes.Event) bool {
		return *event.Reference == *pool.ID && event.Namespace == pool.Namespace && event.Type == fftypes.EventTypePoolRejected
	})).Return(nil)

	valid, err := sh.HandleSystemBroadcast(context.Background(), msg, data)
	assert.False(t, valid)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastTokenPoolSelfUpdateTXFail(t *testing.T) {
	sh := newTestSystemHandlers(t)

	pool := &fftypes.TokenPool{
		ID:         fftypes.NewUUID(),
		Namespace:  "ns1",
		Name:       "name1",
		Type:       fftypes.TokenTypeFungible,
		ProtocolID: "12345",
		Symbol:     "COIN",
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   fftypes.NewUUID(),
		},
	}
	announce := &fftypes.TokenPoolAnnouncement{
		Pool:         pool,
		ProtocolTxID: "tx123",
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: string(fftypes.SystemTagDefinePool),
		},
	}
	b, err := json.Marshal(&announce)
	assert.NoError(t, err)
	data := []*fftypes.Data{{
		Value: fftypes.Byteable(b),
	}}
	opID := fftypes.NewUUID()
	operations := []*fftypes.Operation{{ID: opID}}
	tx := &fftypes.Transaction{ProtocolID: "tx123"}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", context.Background(), mock.Anything).Return(operations, nil, nil)
	mdi.On("UpdateOperation", context.Background(), opID, mock.Anything).Return(nil)
	mdi.On("GetTransactionByID", context.Background(), pool.TX.ID).Return(tx, nil)
	mdi.On("UpsertTransaction", context.Background(), tx, false).Return(fmt.Errorf("pop"))

	valid, err := sh.HandleSystemBroadcast(context.Background(), msg, data)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastTokenPoolOk(t *testing.T) {
	sh := newTestSystemHandlers(t)

	pool := &fftypes.TokenPool{
		ID:         fftypes.NewUUID(),
		Namespace:  "ns1",
		Name:       "name1",
		Type:       fftypes.TokenTypeFungible,
		ProtocolID: "12345",
		Symbol:     "COIN",
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   fftypes.NewUUID(),
		},
	}
	announce := &fftypes.TokenPoolAnnouncement{
		Pool:         pool,
		ProtocolTxID: "tx123",
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: string(fftypes.SystemTagDefinePool),
		},
	}
	b, err := json.Marshal(&announce)
	assert.NoError(t, err)
	data := []*fftypes.Data{{
		Value: fftypes.Byteable(b),
	}}
	operations := []*fftypes.Operation{}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", context.Background(), mock.Anything).Return(operations, nil, nil)
	mdi.On("GetTransactionByID", context.Background(), pool.TX.ID).Return(nil, nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(t *fftypes.Transaction) bool {
		return t.Subject.Type == fftypes.TransactionTypeTokenPool && *t.Subject.Reference == *pool.ID
	}), false).Return(nil)
	mdi.On("UpsertTokenPool", context.Background(), mock.MatchedBy(func(p *fftypes.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(nil)
	mdi.On("InsertEvent", context.Background(), mock.MatchedBy(func(event *fftypes.Event) bool {
		return *event.Reference == *pool.ID && event.Namespace == pool.Namespace && event.Type == fftypes.EventTypePoolConfirmed
	})).Return(nil)

	mam := sh.assets.(*assetmocks.Manager)
	mam.On("ValidateTokenPoolTx", context.Background(), mock.MatchedBy(func(p *fftypes.TokenPool) bool {
		return *p.ID == *pool.ID
	}), "tx123").Return(nil)

	valid, err := sh.HandleSystemBroadcast(context.Background(), msg, data)
	assert.True(t, valid)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mam.AssertExpectations(t)
}

func TestHandleSystemBroadcastTokenPoolValidateTxFail(t *testing.T) {
	sh := newTestSystemHandlers(t)

	pool := &fftypes.TokenPool{
		ID:         fftypes.NewUUID(),
		Namespace:  "ns1",
		Name:       "name1",
		Type:       fftypes.TokenTypeFungible,
		ProtocolID: "12345",
		Symbol:     "COIN",
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   fftypes.NewUUID(),
		},
	}
	announce := &fftypes.TokenPoolAnnouncement{
		Pool:         pool,
		ProtocolTxID: "tx123",
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: string(fftypes.SystemTagDefinePool),
		},
	}
	b, err := json.Marshal(&announce)
	assert.NoError(t, err)
	data := []*fftypes.Data{{
		Value: fftypes.Byteable(b),
	}}
	operations := []*fftypes.Operation{}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", context.Background(), mock.Anything).Return(operations, nil, nil)

	mam := sh.assets.(*assetmocks.Manager)
	mam.On("ValidateTokenPoolTx", context.Background(), mock.MatchedBy(func(p *fftypes.TokenPool) bool {
		return *p.ID == *pool.ID
	}), "tx123").Return(fmt.Errorf("pop"))

	valid, err := sh.HandleSystemBroadcast(context.Background(), msg, data)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mam.AssertExpectations(t)
}

func TestHandleSystemBroadcastTokenPoolBadTX(t *testing.T) {
	sh := newTestSystemHandlers(t)

	pool := &fftypes.TokenPool{
		ID:         fftypes.NewUUID(),
		Namespace:  "ns1",
		Name:       "name1",
		Type:       fftypes.TokenTypeFungible,
		ProtocolID: "12345",
		Symbol:     "COIN",
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   nil,
		},
	}
	announce := &fftypes.TokenPoolAnnouncement{
		Pool:         pool,
		ProtocolTxID: "tx123",
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: string(fftypes.SystemTagDefinePool),
		},
	}
	b, err := json.Marshal(&announce)
	assert.NoError(t, err)
	data := []*fftypes.Data{{
		Value: fftypes.Byteable(b),
	}}
	operations := []*fftypes.Operation{}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", context.Background(), mock.Anything).Return(operations, nil, nil)
	mdi.On("InsertEvent", context.Background(), mock.MatchedBy(func(event *fftypes.Event) bool {
		return *event.Reference == *pool.ID && event.Namespace == pool.Namespace && event.Type == fftypes.EventTypePoolRejected
	})).Return(nil)

	mam := sh.assets.(*assetmocks.Manager)
	mam.On("ValidateTokenPoolTx", context.Background(), mock.MatchedBy(func(p *fftypes.TokenPool) bool {
		return *p.ID == *pool.ID
	}), "tx123").Return(nil)

	valid, err := sh.HandleSystemBroadcast(context.Background(), msg, data)
	assert.False(t, valid)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mam.AssertExpectations(t)
}

func TestHandleSystemBroadcastTokenPoolIDMismatch(t *testing.T) {
	sh := newTestSystemHandlers(t)

	pool := &fftypes.TokenPool{
		ID:         fftypes.NewUUID(),
		Namespace:  "ns1",
		Name:       "name1",
		Type:       fftypes.TokenTypeFungible,
		ProtocolID: "12345",
		Symbol:     "COIN",
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   fftypes.NewUUID(),
		},
	}
	announce := &fftypes.TokenPoolAnnouncement{
		Pool:         pool,
		ProtocolTxID: "tx123",
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: string(fftypes.SystemTagDefinePool),
		},
	}
	b, err := json.Marshal(&announce)
	assert.NoError(t, err)
	data := []*fftypes.Data{{
		Value: fftypes.Byteable(b),
	}}
	operations := []*fftypes.Operation{}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", context.Background(), mock.Anything).Return(operations, nil, nil)
	mdi.On("GetTransactionByID", context.Background(), pool.TX.ID).Return(nil, nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(t *fftypes.Transaction) bool {
		return t.Subject.Type == fftypes.TransactionTypeTokenPool && *t.Subject.Reference == *pool.ID
	}), false).Return(nil)
	mdi.On("UpsertTokenPool", context.Background(), mock.MatchedBy(func(p *fftypes.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(database.IDMismatch)
	mdi.On("InsertEvent", context.Background(), mock.MatchedBy(func(event *fftypes.Event) bool {
		return *event.Reference == *pool.ID && event.Namespace == pool.Namespace && event.Type == fftypes.EventTypePoolRejected
	})).Return(nil)

	mam := sh.assets.(*assetmocks.Manager)
	mam.On("ValidateTokenPoolTx", context.Background(), mock.MatchedBy(func(p *fftypes.TokenPool) bool {
		return *p.ID == *pool.ID
	}), "tx123").Return(nil)

	valid, err := sh.HandleSystemBroadcast(context.Background(), msg, data)
	assert.False(t, valid)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mam.AssertExpectations(t)
}

func TestHandleSystemBroadcastTokenPoolFailUpsert(t *testing.T) {
	sh := newTestSystemHandlers(t)

	pool := &fftypes.TokenPool{
		ID:         fftypes.NewUUID(),
		Namespace:  "ns1",
		Name:       "name1",
		Type:       fftypes.TokenTypeFungible,
		ProtocolID: "12345",
		Symbol:     "COIN",
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   fftypes.NewUUID(),
		},
	}
	announce := &fftypes.TokenPoolAnnouncement{
		Pool:         pool,
		ProtocolTxID: "tx123",
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: string(fftypes.SystemTagDefinePool),
		},
	}
	b, err := json.Marshal(&announce)
	assert.NoError(t, err)
	data := []*fftypes.Data{{
		Value: fftypes.Byteable(b),
	}}
	operations := []*fftypes.Operation{}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", context.Background(), mock.Anything).Return(operations, nil, nil)
	mdi.On("GetTransactionByID", context.Background(), pool.TX.ID).Return(nil, nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(t *fftypes.Transaction) bool {
		return t.Subject.Type == fftypes.TransactionTypeTokenPool && *t.Subject.Reference == *pool.ID
	}), false).Return(nil)
	mdi.On("UpsertTokenPool", context.Background(), mock.MatchedBy(func(p *fftypes.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(fmt.Errorf("pop"))

	mam := sh.assets.(*assetmocks.Manager)
	mam.On("ValidateTokenPoolTx", context.Background(), mock.MatchedBy(func(p *fftypes.TokenPool) bool {
		return *p.ID == *pool.ID
	}), "tx123").Return(nil)

	valid, err := sh.HandleSystemBroadcast(context.Background(), msg, data)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mam.AssertExpectations(t)
}

func TestHandleSystemBroadcastTokenPoolOpsFail(t *testing.T) {
	sh := newTestSystemHandlers(t)

	announce := &fftypes.TokenPoolAnnouncement{
		Pool: &fftypes.TokenPool{
			ID:         fftypes.NewUUID(),
			Namespace:  "ns1",
			Name:       "name1",
			Type:       fftypes.TokenTypeFungible,
			ProtocolID: "12345",
			Symbol:     "COIN",
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeTokenPool,
				ID:   fftypes.NewUUID(),
			},
		},
		ProtocolTxID: "tx123",
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: string(fftypes.SystemTagDefinePool),
		},
	}
	b, err := json.Marshal(&announce)
	assert.NoError(t, err)
	data := []*fftypes.Data{{
		Value: fftypes.Byteable(b),
	}}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", context.Background(), mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	valid, err := sh.HandleSystemBroadcast(context.Background(), msg, data)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastTokenPoolValidateFail(t *testing.T) {
	sh := newTestSystemHandlers(t)

	announce := &fftypes.TokenPoolAnnouncement{
		Pool:         &fftypes.TokenPool{},
		ProtocolTxID: "tx123",
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: string(fftypes.SystemTagDefinePool),
		},
	}
	b, err := json.Marshal(&announce)
	assert.NoError(t, err)
	data := []*fftypes.Data{{
		Value: fftypes.Byteable(b),
	}}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("InsertEvent", context.Background(), mock.MatchedBy(func(event *fftypes.Event) bool {
		return event.Type == fftypes.EventTypePoolRejected
	})).Return(nil)

	valid, err := sh.HandleSystemBroadcast(context.Background(), msg, data)
	assert.False(t, valid)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}
