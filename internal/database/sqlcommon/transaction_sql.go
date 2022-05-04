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
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

var (
	transactionColumns = []string{
		"id",
		"ttype",
		"namespace",
		"created",
		"blockchain_ids",
	}
	transactionFilterFieldMap = map[string]string{
		"type":          "ttype",
		"blockchainids": "blockchain_ids",
	}
)

const transactionsTable = "transactions"

func (s *SQLCommon) InsertTransaction(ctx context.Context, transaction *core.Transaction) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	transaction.Created = fftypes.Now()
	if _, err = s.insertTx(ctx, transactionsTable, tx,
		sq.Insert(transactionsTable).
			Columns(transactionColumns...).
			Values(
				transaction.ID,
				string(transaction.Type),
				transaction.Namespace,
				transaction.Created,
				transaction.BlockchainIDs,
			),
		func() {
			s.callbacks.UUIDCollectionNSEvent(database.CollectionTransactions, core.ChangeEventTypeCreated, transaction.Namespace, transaction.ID)
		},
	); err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) transactionResult(ctx context.Context, row *sql.Rows) (*core.Transaction, error) {
	var transaction core.Transaction
	err := row.Scan(
		&transaction.ID,
		&transaction.Type,
		&transaction.Namespace,
		&transaction.Created,
		&transaction.BlockchainIDs,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, transactionsTable)
	}
	return &transaction, nil
}

func (s *SQLCommon) GetTransactionByID(ctx context.Context, id *fftypes.UUID) (message *core.Transaction, err error) {

	rows, _, err := s.query(ctx, transactionsTable,
		sq.Select(transactionColumns...).
			From(transactionsTable).
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

func (s *SQLCommon) GetTransactions(ctx context.Context, filter database.Filter) (message []*core.Transaction, fr *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(transactionColumns...).From(transactionsTable), filter, transactionFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, transactionsTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	transactions := []*core.Transaction{}
	for rows.Next() {
		transaction, err := s.transactionResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		transactions = append(transactions, transaction)
	}

	return transactions, s.queryRes(ctx, transactionsTable, tx, fop, fi), err

}

func (s *SQLCommon) UpdateTransaction(ctx context.Context, id *fftypes.UUID, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update(transactionsTable), update, transactionFilterFieldMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"id": id})

	_, err = s.updateTx(ctx, transactionsTable, tx, query, nil /* no change evnents for filter based updates */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
