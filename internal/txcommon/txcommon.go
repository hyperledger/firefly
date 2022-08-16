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

package txcommon

import (
	"context"
	"strings"
	"time"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/karlseguin/ccache"
)

type Helper interface {
	SubmitNewTransaction(ctx context.Context, txType core.TransactionType) (*fftypes.UUID, error)
	PersistTransaction(ctx context.Context, id *fftypes.UUID, txType core.TransactionType, blockchainTXID string) (valid bool, err error)
	AddBlockchainTX(ctx context.Context, tx *core.Transaction, blockchainTXID string) error
	InsertBlockchainEvent(ctx context.Context, chainEvent *core.BlockchainEvent) error
	EnrichEvent(ctx context.Context, event *core.Event) (*core.EnrichedEvent, error)
	GetTransactionByIDCached(ctx context.Context, id *fftypes.UUID) (*core.Transaction, error)
	GetBlockchainEventByIDCached(ctx context.Context, id *fftypes.UUID) (*core.BlockchainEvent, error)
}

type transactionHelper struct {
	namespace            string
	database             database.Plugin
	data                 data.Manager
	transactionCache     *ccache.Cache
	transactionCacheTTL  time.Duration
	blockchainEventCache *ccache.Cache
	blockchainEventTTL   time.Duration
}

func NewTransactionHelper(ns string, di database.Plugin, dm data.Manager) Helper {
	t := &transactionHelper{
		namespace: ns,
		database:  di,
		data:      dm,
	}
	t.transactionCache = ccache.New(
		// We use a LRU cache with a size-aware max
		ccache.Configure().
			MaxSize(config.GetByteSize(coreconfig.TransactionCacheSize)),
	)
	t.blockchainEventCache = ccache.New(
		// We use a LRU cache with a size-aware max
		ccache.Configure().
			MaxSize(config.GetByteSize(coreconfig.BlockchainEventCacheSize)),
	)
	return t
}

func (t *transactionHelper) updateTransactionsCache(tx *core.Transaction) {
	t.transactionCache.Set(tx.ID.String(), tx, t.transactionCacheTTL)
}

func (t *transactionHelper) GetTransactionByIDCached(ctx context.Context, id *fftypes.UUID) (*core.Transaction, error) {
	cached := t.transactionCache.Get(id.String())
	if cached != nil {
		cached.Extend(t.transactionCacheTTL)
		return cached.Value().(*core.Transaction), nil
	}
	tx, err := t.database.GetTransactionByID(ctx, t.namespace, id)
	if err != nil || tx == nil {
		return tx, err
	}
	t.updateTransactionsCache(tx)
	return tx, nil
}

// SubmitNewTransaction is called when there is a new transaction being submitted by the local node
func (t *transactionHelper) SubmitNewTransaction(ctx context.Context, txType core.TransactionType) (*fftypes.UUID, error) {

	tx := &core.Transaction{
		ID:        fftypes.NewUUID(),
		Namespace: t.namespace,
		Type:      txType,
	}

	if err := t.database.InsertTransaction(ctx, tx); err != nil {
		return nil, err
	}

	if err := t.database.InsertEvent(ctx, core.NewEvent(core.EventTypeTransactionSubmitted, tx.Namespace, tx.ID, tx.ID, tx.Type.String())); err != nil {
		return nil, err
	}

	t.updateTransactionsCache(tx)
	return tx.ID, nil
}

// PersistTransaction is called when we need to ensure a transaction exists in the DB, and optionally associate a new BlockchainTXID to it
func (t *transactionHelper) PersistTransaction(ctx context.Context, id *fftypes.UUID, txType core.TransactionType, blockchainTXID string) (valid bool, err error) {

	// TODO: Consider if this can exploit caching
	tx, err := t.database.GetTransactionByID(ctx, t.namespace, id)
	if err != nil {
		return false, err
	}

	if tx != nil {
		if tx.Type != txType {
			log.L(ctx).Errorf("Type mismatch for transaction '%s' existing=%s new=%s", tx.ID, tx.Type, txType)
			return false, nil
		}

		newBlockchainIDs, changed := tx.BlockchainIDs.AddToSortedSet(blockchainTXID)
		if !changed {
			return true, nil
		}

		if err = t.database.UpdateTransaction(ctx, t.namespace, tx.ID, database.TransactionQueryFactory.NewUpdate(ctx).Set("blockchainids", newBlockchainIDs)); err != nil {
			return false, err
		}
	} else {
		tx = &core.Transaction{
			ID:            id,
			Namespace:     t.namespace,
			Type:          txType,
			BlockchainIDs: core.NewFFStringArray(strings.ToLower(blockchainTXID)),
		}
		if err = t.database.InsertTransaction(ctx, tx); err != nil {
			return false, err
		}
	}

	t.updateTransactionsCache(tx)
	return true, nil
}

// AddBlockchainTX is called when we know the transaction should exist, and we don't need any validation
// but just want to bolt on an extra blockchain TXID (if it's not there already).
func (t *transactionHelper) AddBlockchainTX(ctx context.Context, tx *core.Transaction, blockchainTXID string) error {

	var changed bool
	tx.BlockchainIDs, changed = tx.BlockchainIDs.AddToSortedSet(blockchainTXID)
	if !changed {
		return nil
	}

	err := t.database.UpdateTransaction(ctx, t.namespace, tx.ID, database.TransactionQueryFactory.NewUpdate(ctx).Set("blockchainids", tx.BlockchainIDs))
	if err != nil {
		return err
	}

	t.updateTransactionsCache(tx)
	return nil
}

func (t *transactionHelper) addBlockchainEventToCache(chainEvent *core.BlockchainEvent) {
	t.blockchainEventCache.Set(chainEvent.ID.String(), chainEvent, t.blockchainEventTTL)
}

func (t *transactionHelper) GetBlockchainEventByIDCached(ctx context.Context, id *fftypes.UUID) (*core.BlockchainEvent, error) {
	cached := t.blockchainEventCache.Get(id.String())
	if cached != nil {
		cached.Extend(t.blockchainEventTTL)
		return cached.Value().(*core.BlockchainEvent), nil
	}
	chainEvent, err := t.database.GetBlockchainEventByID(ctx, t.namespace, id)
	if err != nil || chainEvent == nil {
		return chainEvent, err
	}
	t.addBlockchainEventToCache(chainEvent)
	return chainEvent, nil
}

func (t *transactionHelper) InsertBlockchainEvent(ctx context.Context, chainEvent *core.BlockchainEvent) error {
	err := t.database.InsertBlockchainEvent(ctx, chainEvent)
	if err != nil {
		return err
	}
	t.addBlockchainEventToCache(chainEvent)
	return nil
}
