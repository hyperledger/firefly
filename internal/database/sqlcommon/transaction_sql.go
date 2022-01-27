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

package sqlcommon

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var (
	transactionColumns = []string{
		"id",
		"ttype",
		"namespace",
		"created",
		"status",
		"blockchain_ids",
	}
	transactionFilterFieldMap = map[string]string{
		"type":          "ttype",
		"blockchainids": "blockchain_ids",
	}
)

func (s *SQLCommon) UpsertTransaction(ctx context.Context, transaction *fftypes.Transaction) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	// Do a select within the transaction to determine if the UUID already exists
	transactionRows, _, err := s.queryTx(ctx, tx,
		sq.Select("blockchain_ids").
			From("transactions").
			Where(sq.Eq{"id": transaction.ID}),
	)
	if err != nil {
		return err
	}
	existing := transactionRows.Next()

	if existing {
		var existingBlockchainIDs fftypes.FFStringArray
		_ = transactionRows.Scan(&existingBlockchainIDs)
		transaction.BlockchainIDs = transaction.BlockchainIDs.MergeLower(existingBlockchainIDs)
		transactionRows.Close()
		// Update the transaction
		if _, err = s.updateTx(ctx, tx,
			sq.Update("transactions").
				Set("ttype", string(transaction.Type)).
				Set("namespace", transaction.Namespace).
				Set("status", transaction.Status).
				Set("blockchain_ids", transaction.BlockchainIDs).
				Where(sq.Eq{"id": transaction.ID}),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionTransactions, fftypes.ChangeEventTypeUpdated, transaction.Namespace, transaction.ID)
			},
		); err != nil {
			return err
		}
	} else {
		transactionRows.Close()
		// Insert a transaction
		transaction.Created = fftypes.Now()
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("transactions").
				Columns(transactionColumns...).
				Values(
					transaction.ID,
					string(transaction.Type),
					transaction.Namespace,
					transaction.Created,
					transaction.Status,
					transaction.BlockchainIDs,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionTransactions, fftypes.ChangeEventTypeCreated, transaction.Namespace, transaction.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) transactionResult(ctx context.Context, row *sql.Rows) (*fftypes.Transaction, error) {
	var transaction fftypes.Transaction
	err := row.Scan(
		&transaction.ID,
		&transaction.Type,
		&transaction.Namespace,
		&transaction.Created,
		&transaction.Status,
		&transaction.BlockchainIDs,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "transactions")
	}
	return &transaction, nil
}

func (s *SQLCommon) GetTransactionByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.Transaction, err error) {

	rows, _, err := s.query(ctx,
		sq.Select(transactionColumns...).
			From("transactions").
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Transaction '%s' not found", id)
		return nil, nil
	}

	transaction, err := s.transactionResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return transaction, nil
}

func (s *SQLCommon) GetTransactions(ctx context.Context, filter database.Filter) (message []*fftypes.Transaction, fr *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(transactionColumns...).From("transactions"), filter, transactionFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	transactions := []*fftypes.Transaction{}
	for rows.Next() {
		transaction, err := s.transactionResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		transactions = append(transactions, transaction)
	}

	return transactions, s.queryRes(ctx, tx, "transactions", fop, fi), err

}

func (s *SQLCommon) UpdateTransaction(ctx context.Context, id *fftypes.UUID, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update("transactions"), update, transactionFilterFieldMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"id": id})

	_, err = s.updateTx(ctx, tx, query, nil /* no change evnents for filter based updates */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
