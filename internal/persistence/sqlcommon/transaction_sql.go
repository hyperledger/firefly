// Copyright © 2021 Kaleido, Inc.
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
	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/persistence"
)

var (
	transactionColumns = []string{
		"id",
		"ttype",
		"author",
		"created",
		"tracking_id",
		"protocol_id",
		"confirmed",
		"info",
	}
)

func (s *SQLCommon) UpsertTransaction(ctx context.Context, transaction *fftypes.Transaction) (err error) {
	ctx, tx, err := s.beginTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx)

	// Do a select within the transaction to detemine if the UUID already exists
	transactionRows, err := s.queryTx(ctx, tx,
		sq.Select("id").
			From("transactions").
			Where(sq.Eq{"id": transaction.ID}),
	)
	if err != nil {
		return err
	}

	if transactionRows.Next() {
		transactionRows.Close()

		// Update the transaction
		if _, err = s.updateTx(ctx, tx,
			sq.Update("transactions").
				Set("ttype", string(transaction.Type)).
				Set("author", transaction.Author).
				Set("created", transaction.Created).
				Set("tracking_id", transaction.TrackingID).
				Set("protocol_id", transaction.ProtocolID).
				Set("confirmed", transaction.Confirmed).
				Set("info", transaction.Info).
				Where(sq.Eq{"id": transaction.ID}),
		); err != nil {
			return err
		}
	} else {
		transactionRows.Close()

		if _, err = s.insertTx(ctx, tx,
			sq.Insert("transactions").
				Columns(transactionColumns...).
				Values(
					transaction.ID,
					string(transaction.Type),
					transaction.Author,
					transaction.Created,
					transaction.TrackingID,
					transaction.ProtocolID,
					transaction.Confirmed,
					transaction.Info,
				),
		); err != nil {
			return err
		}
	}

	if err = s.commitTx(ctx, tx); err != nil {
		return err
	}

	return nil
}

func (s *SQLCommon) transactionResult(ctx context.Context, row *sql.Rows) (*fftypes.Transaction, error) {
	var transaction fftypes.Transaction
	err := row.Scan(
		&transaction.ID,
		&transaction.Type,
		&transaction.Author,
		&transaction.Created,
		&transaction.TrackingID,
		&transaction.ProtocolID,
		&transaction.Confirmed,
		&transaction.Info,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "transactions")
	}
	return &transaction, nil
}

func (s *SQLCommon) GetTransactionById(ctx context.Context, ns string, id *uuid.UUID) (message *fftypes.Transaction, err error) {

	rows, err := s.query(ctx,
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

func (s *SQLCommon) GetTransactions(ctx context.Context, skip, limit uint64, filter *persistence.TransactionFilter) (message []*fftypes.Transaction, err error) {

	query := sq.Select(transactionColumns...).From("transactions")

	if filter.ConfirmedAfter > 0 {
		query = query.Where(sq.Gt{"confirmed": filter.ConfirmedAfter})
	} else if filter.ConfrimedOnly {
		query = query.Where(sq.Gt{"confirmed": 0})
	} else if filter.UnconfrimedOnly {
		query = query.Where(sq.Eq{"confirmed": 0})
	}

	if filter.AuthorEquals != "" {
		query = query.Where(sq.Eq{"author": filter.AuthorEquals})
	}
	if filter.TrackingIDEquals != "" {
		query = query.Where(sq.Eq{"tracking_id": filter.TrackingIDEquals})
	}
	if filter.ProtocolIDEquals != "" {
		query = query.Where(sq.Eq{"protocol_id": filter.ProtocolIDEquals})
	}
	if filter.CreatedAfter > 0 {
		query = query.Where(sq.Gt{"created": filter.CreatedAfter})
	}
	query = query.OrderBy("confirmed,created DESC")
	if limit > 0 {
		query = query.Offset(skip).Limit(limit)
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := []*fftypes.Transaction{}
	for rows.Next() {
		transaction, err := s.transactionResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}

	return transactions, err

}
