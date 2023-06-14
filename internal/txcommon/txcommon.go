// Copyright © 2023 Kaleido, Inc.
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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

type Helper interface {
	SubmitNewTransaction(ctx context.Context, txType core.TransactionType, idempotencyKey core.IdempotencyKey) (*fftypes.UUID, error)
	PersistTransaction(ctx context.Context, id *fftypes.UUID, txType core.TransactionType, blockchainTXID string) (valid bool, err error)
	AddBlockchainTX(ctx context.Context, tx *core.Transaction, blockchainTXID string) error
	InsertNewBlockchainEvents(ctx context.Context, events []*core.BlockchainEvent) (inserted []*core.BlockchainEvent, err error)
	GetTransactionByIDCached(ctx context.Context, id *fftypes.UUID) (*core.Transaction, error)
	GetBlockchainEventByIDCached(ctx context.Context, id *fftypes.UUID) (*core.BlockchainEvent, error)
	FindOperationInTransaction(ctx context.Context, tx *fftypes.UUID, opType core.OpType) (*core.Operation, error)
}

type transactionHelper struct {
	namespace            string
	database             database.Plugin
	data                 data.Manager
	transactionCache     cache.CInterface
	blockchainEventCache cache.CInterface
}

func NewTransactionHelper(ctx context.Context, ns string, di database.Plugin, dm data.Manager, cacheManager cache.Manager) (Helper, error) {
	t := &transactionHelper{
		namespace: ns,
		database:  di,
		data:      dm,
	}

	transactionCache, err := cacheManager.GetCache(
		cache.NewCacheConfig(
			ctx,
			coreconfig.CacheTransactionSize,
			coreconfig.CacheTransactionTTL,
			ns,
		),
	)

	if err != nil {
		return nil, err
	}

	t.transactionCache = transactionCache

	blockchainEventCache, err := cacheManager.GetCache(
		cache.NewCacheConfig(
			ctx,
			coreconfig.CacheBlockchainEventLimit,
			coreconfig.CacheBlockchainEventTTL,
			ns,
		),
	)

	if err != nil {
		return nil, err
	}
	t.blockchainEventCache = blockchainEventCache
	return t, nil
}

func (t *transactionHelper) updateTransactionsCache(tx *core.Transaction) {
	t.transactionCache.Set(tx.ID.String(), tx)
}

func (t *transactionHelper) GetTransactionByIDCached(ctx context.Context, id *fftypes.UUID) (*core.Transaction, error) {

	if cachedValue := t.transactionCache.Get(id.String()); cachedValue != nil {
		return cachedValue.(*core.Transaction), nil
	}
	tx, err := t.database.GetTransactionByID(ctx, t.namespace, id)
	if err != nil || tx == nil {
		return tx, err
	}
	t.updateTransactionsCache(tx)
	return tx, nil
}

// SubmitNewTransaction is called when there is a new transaction being submitted by the local node
func (t *transactionHelper) SubmitNewTransaction(ctx context.Context, txType core.TransactionType, idempotencyKey core.IdempotencyKey) (*fftypes.UUID, error) {

	tx := &core.Transaction{
		ID:             fftypes.NewUUID(),
		Namespace:      t.namespace,
		Type:           txType,
		IdempotencyKey: idempotencyKey,
	}

	// Note that InsertTransaction is responsible for idempotency key duplicate detection and helpful error creation.
	// (In cases where one or more operations have not yet left 'initialized' state then we need to resubmit them even if
	// we've seen this idempotency key before.)
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

	tx, err := t.GetTransactionByIDCached(ctx, id)
	if err != nil {
		return false, err
	}

	if tx != nil {
		if tx.Type != txType {
			log.L(ctx).Errorf("Type mismatch for transaction '%s' existing=%s new=%s", tx.ID, tx.Type, txType)
			return false, nil
		}

		var changed bool
		tx.BlockchainIDs, changed = tx.BlockchainIDs.AddToSortedSet(blockchainTXID)
		if !changed {
			return true, nil
		}

		if err = t.database.UpdateTransaction(ctx, t.namespace, tx.ID, database.TransactionQueryFactory.NewUpdate(ctx).Set("blockchainids", tx.BlockchainIDs)); err != nil {
			return false, err
		}
	} else {
		tx = &core.Transaction{
			ID:            id,
			Namespace:     t.namespace,
			Type:          txType,
			BlockchainIDs: fftypes.NewFFStringArray(strings.ToLower(blockchainTXID)),
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
	t.blockchainEventCache.Set(chainEvent.ID.String(), chainEvent)
}

func (t *transactionHelper) GetBlockchainEventByIDCached(ctx context.Context, id *fftypes.UUID) (*core.BlockchainEvent, error) {

	if cachedValue := t.blockchainEventCache.Get(id.String()); cachedValue != nil {
		return cachedValue.(*core.BlockchainEvent), nil
	}
	chainEvent, err := t.database.GetBlockchainEventByID(ctx, t.namespace, id)
	if err != nil || chainEvent == nil {
		return chainEvent, err
	}
	t.addBlockchainEventToCache(chainEvent)
	return chainEvent, nil
}

func (t *transactionHelper) InsertNewBlockchainEvents(ctx context.Context, events []*core.BlockchainEvent) (inserted []*core.BlockchainEvent, err error) {
	// First we try and insert the whole bundle using batch insert
	err = t.database.InsertBlockchainEvents(ctx, events, func() {
		for _, event := range events {
			t.addBlockchainEventToCache(event)
		}
	})
	if err == nil {
		// happy path worked - all new events
		return events, nil
	}
	// Fall back to insert-or-get
	log.L(ctx).Warnf("Blockchain event insert-many optimization failed: %s", err)
	inserted = make([]*core.BlockchainEvent, 0, len(events))
	for _, event := range events {
		existing, err := t.database.InsertOrGetBlockchainEvent(ctx, event)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			log.L(ctx).Debugf("Ignoring duplicate blockchain event %s", existing.ProtocolID)
			t.addBlockchainEventToCache(existing)
		} else {
			inserted = append(inserted, event)
			t.addBlockchainEventToCache(event)
		}
	}
	return inserted, nil
}

func (t *transactionHelper) FindOperationInTransaction(ctx context.Context, tx *fftypes.UUID, opType core.OpType) (*core.Operation, error) {
	fb := database.OperationQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("tx", tx),
		fb.Eq("type", opType),
	)
	ops, _, err := t.database.GetOperations(ctx, t.namespace, filter)
	if err != nil || len(ops) == 0 {
		return nil, err
	}
	return ops[0], nil
}
