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

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/karlseguin/ccache"
)

type Helper interface {
	SubmitNewTransaction(ctx context.Context, ns string, txType fftypes.TransactionType) (*fftypes.UUID, error)
	PersistTransaction(ctx context.Context, ns string, id *fftypes.UUID, txType fftypes.TransactionType, blockchainTXID string) (valid bool, err error)
	AddBlockchainTX(ctx context.Context, id *fftypes.UUID, blockchainTXID string) error
	EnrichEvent(ctx context.Context, event *fftypes.Event) (*fftypes.EnrichedEvent, error)
	GetTransactionByIDCached(ctx context.Context, id *fftypes.UUID) (*fftypes.Transaction, error)
}

type transactionHelper struct {
	database            database.Plugin
	data                data.Manager
	transactionCache    *ccache.Cache
	transactionCacheTTL time.Duration
}

func NewTransactionHelper(di database.Plugin, dm data.Manager) Helper {
	t := &transactionHelper{
		database: di,
		data:     dm,
	}
	t.transactionCache = ccache.New(
		// We use a LRU cache with a size-aware max
		ccache.Configure().
			MaxSize(config.GetByteSize(config.MessageCacheTTL)),
	)
	return t
}

func (t *transactionHelper) updateTransactionsCache(tx *fftypes.Transaction) {
	t.transactionCache.Set(tx.ID.String(), tx, t.transactionCacheTTL)
}

func (t *transactionHelper) GetTransactionByIDCached(ctx context.Context, id *fftypes.UUID) (*fftypes.Transaction, error) {
	cached := t.transactionCache.Get(id.String())
	if cached != nil {
		cached.Extend(t.transactionCacheTTL)
		return cached.Value().(*fftypes.Transaction), nil
	}
	tx, err := t.database.GetTransactionByID(ctx, id)
	if err != nil || tx == nil {
		return tx, err
	}
	t.updateTransactionsCache(tx)
	return tx, nil
}

// SubmitNewTransaction is called when there is a new transaction being submitted by the local node
func (t *transactionHelper) SubmitNewTransaction(ctx context.Context, ns string, txType fftypes.TransactionType) (*fftypes.UUID, error) {

	tx := &fftypes.Transaction{
		ID:        fftypes.NewUUID(),
		Namespace: ns,
		Type:      txType,
	}

	if err := t.database.InsertTransaction(ctx, tx); err != nil {
		return nil, err
	}

	if err := t.database.InsertEvent(ctx, fftypes.NewEvent(fftypes.EventTypeTransactionSubmitted, tx.Namespace, tx.ID, tx.ID)); err != nil {
		return nil, err
	}

	return tx.ID, nil
}

// PersistTransaction is called when we need to ensure a transaction exists in the DB, and optionally associate a new BlockchainTXID to it
func (t *transactionHelper) PersistTransaction(ctx context.Context, ns string, id *fftypes.UUID, txType fftypes.TransactionType, blockchainTXID string) (valid bool, err error) {

	// TODO: Consider if this can exploit caching
	tx, err := t.database.GetTransactionByID(ctx, id)
	if err != nil {
		return false, err
	}

	if tx != nil {

		if tx.Namespace != ns {
			log.L(ctx).Errorf("Namespace mismatch for transaction '%s' existing=%s new=%s", tx.ID, tx.Namespace, ns)
			return false, nil
		}

		if tx.Type != txType {
			log.L(ctx).Errorf("Type mismatch for transaction '%s' existing=%s new=%s", tx.ID, tx.Type, txType)
			return false, nil
		}

		newBlockchainIDs, changed := tx.BlockchainIDs.AddToSortedSet(blockchainTXID)
		if !changed {
			return true, nil
		}

		if err = t.database.UpdateTransaction(ctx, tx.ID, database.TransactionQueryFactory.NewUpdate(ctx).Set("blockchainids", newBlockchainIDs)); err != nil {
			return false, err
		}

	} else {
		tx = &fftypes.Transaction{
			ID:            id,
			Namespace:     ns,
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
func (t *transactionHelper) AddBlockchainTX(ctx context.Context, id *fftypes.UUID, blockchainTXID string) error {

	tx, err := t.database.GetTransactionByID(ctx, id)
	if err != nil {
		return err
	}

	if tx != nil {

		newBlockchainIDs, changed := tx.BlockchainIDs.AddToSortedSet(blockchainTXID)
		if !changed {
			return nil
		}

		if err = t.database.UpdateTransaction(ctx, tx.ID, database.TransactionQueryFactory.NewUpdate(ctx).Set("blockchainids", newBlockchainIDs)); err != nil {
			return err
		}

	}

	return nil
}
