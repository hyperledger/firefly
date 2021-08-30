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

	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/mocks/tokenmocks"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTokenPoolCreatedSuccess(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	pool := &fftypes.TokenPool{
		ID: fftypes.NewUUID(),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenPool,
		},
		Namespace: "test-ns",
		Name:      "my-pool",
	}

	mdi.On("GetTransactionByID", mock.Anything, uuidMatches(pool.TX.ID)).Return(nil, nil)
	mdi.On("UpsertTransaction", em.ctx, mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenPool
	}), false).Return(nil)
	mdi.On("UpsertTokenPool", em.ctx, pool).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypePoolConfirmed && ev.Reference == pool.ID && ev.Namespace == pool.Namespace
	})).Return(nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokenPoolCreated(mti, pool, "0x12345", "tx1", info)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestTokenPoolMissingID(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	pool := &fftypes.TokenPool{}

	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypePoolRejected && ev.Reference == pool.ID && ev.Namespace == pool.Namespace
	})).Return(nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokenPoolCreated(mti, pool, "0x12345", "tx1", info)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestTokenPoolBadNamespace(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	pool := &fftypes.TokenPool{
		ID: fftypes.NewUUID(),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenPool,
		},
	}

	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypePoolRejected && ev.Reference == pool.ID && ev.Namespace == pool.Namespace
	})).Return(nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokenPoolCreated(mti, pool, "0x12345", "tx1", info)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestTokenPoolBadName(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	pool := &fftypes.TokenPool{
		ID: fftypes.NewUUID(),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenPool,
		},
		Namespace: "test-ns",
	}

	mdi.On("GetTransactionByID", mock.Anything, uuidMatches(pool.TX.ID)).Return(nil, nil)
	mdi.On("UpsertTransaction", em.ctx, mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenPool
	}), false).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypePoolRejected && ev.Reference == pool.ID && ev.Namespace == pool.Namespace
	})).Return(nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokenPoolCreated(mti, pool, "0x12345", "tx1", info)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestTokenPoolGetTransactionFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)

	pool := &fftypes.TokenPool{
		ID: fftypes.NewUUID(),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenPool,
		},
		Namespace: "test-ns",
		Name:      "my-pool",
	}

	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))

	info := fftypes.JSONObject{"some": "info"}
	valid, err := em.persistTokenPoolTransaction(em.ctx, pool, "0x12345", "tx1", info)
	assert.EqualError(t, err, "pop")
	assert.False(t, valid)
	mdi.AssertExpectations(t)
}

func TestTokenPoolGetTransactionInvalidMatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)

	pool := &fftypes.TokenPool{
		ID: fftypes.NewUUID(),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenPool,
		},
		Namespace: "test-ns",
		Name:      "my-pool",
	}

	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(&fftypes.Transaction{
		ID: fftypes.NewUUID(), // wrong
	}, nil)

	info := fftypes.JSONObject{"some": "info"}
	valid, err := em.persistTokenPoolTransaction(em.ctx, pool, "0x12345", "tx1", info)
	assert.NoError(t, err)
	assert.False(t, valid)
	mdi.AssertExpectations(t)
}

func TestTokenPoolNewTXUpsertFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)

	pool := &fftypes.TokenPool{
		ID: fftypes.NewUUID(),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenPool,
		},
		Namespace: "test-ns",
		Name:      "my-pool",
	}

	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(nil, nil)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything, false).Return(fmt.Errorf("pop"))

	info := fftypes.JSONObject{"some": "info"}
	valid, err := em.persistTokenPoolTransaction(em.ctx, pool, "0x12345", "tx1", info)
	assert.EqualError(t, err, "pop")
	assert.False(t, valid)
	mdi.AssertExpectations(t)
}

func TestTokenPoolExistingTXHashMismatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)

	pool := &fftypes.TokenPool{
		ID: fftypes.NewUUID(),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenPool,
		},
		Namespace: "test-ns",
		Name:      "my-pool",
	}

	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(&fftypes.Transaction{
		Subject: fftypes.TransactionSubject{
			Type:      fftypes.TransactionTypeTokenPool,
			Namespace: pool.Namespace,
			Signer:    "0x12345",
			Reference: pool.ID,
		},
	}, nil)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything, false).Return(database.HashMismatch)

	info := fftypes.JSONObject{"some": "info"}
	valid, err := em.persistTokenPoolTransaction(em.ctx, pool, "0x12345", "tx1", info)
	assert.NoError(t, err)
	assert.False(t, valid)
	mdi.AssertExpectations(t)
}

func TestTokenPoolIDMismatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	pool := &fftypes.TokenPool{
		ID: fftypes.NewUUID(),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenPool,
		},
		Namespace: "test-ns",
		Name:      "my-pool",
	}

	mdi.On("GetTransactionByID", mock.Anything, uuidMatches(pool.TX.ID)).Return(nil, nil)
	mdi.On("UpsertTransaction", em.ctx, mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenPool
	}), false).Return(nil)
	mdi.On("UpsertTokenPool", em.ctx, pool).Return(database.IDMismatch)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypePoolRejected && ev.Reference == pool.ID && ev.Namespace == pool.Namespace
	})).Return(nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokenPoolCreated(mti, pool, "0x12345", "tx1", info)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestTokenPoolUpsertFailAndRetry(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	pool := &fftypes.TokenPool{
		ID: fftypes.NewUUID(),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenPool,
		},
		Namespace: "test-ns",
		Name:      "my-pool",
	}

	mdi.On("GetTransactionByID", mock.Anything, uuidMatches(pool.TX.ID)).Return(nil, nil)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("UpsertTokenPool", em.ctx, pool).Return(fmt.Errorf("pop")).Once()
	mdi.On("UpsertTokenPool", em.ctx, pool).Return(nil).Once()
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypePoolConfirmed && ev.Reference == pool.ID && ev.Namespace == pool.Namespace
	})).Return(nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokenPoolCreated(mti, pool, "0x12345", "tx1", info)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}
