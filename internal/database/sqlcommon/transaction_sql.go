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

package sqlcommon

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

var (
	transactionColumns = []string{
		"id",
		"ttype",
		"namespace",
		"ref",
		"signer",
		"hash",
		"created",
		"protocol_id",
		"status",
		"info",
	}
	transactionFilterFieldMap = map[string]string{
		"type":       "ttype",
		"protocolid": "protocol_id",
		"reference":  "ref",
	}
)

func (s *SQLCommon) UpsertTransaction(ctx context.Context, transaction *fftypes.Transaction, allowExisting, allowHashUpdate bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		// Do a select within the transaction to detemine if the UUID already exists
		transactionRows, err := s.queryTx(ctx, tx,
			sq.Select("hash").
				From("transactions").
				Where(sq.Eq{"id": transaction.ID}),
		)
		if err != nil {
			return err
		}
		existing = transactionRows.Next()

		if existing && !allowHashUpdate {
			var hash *fftypes.Bytes32
			_ = transactionRows.Scan(&hash)
			if !fftypes.SafeHashCompare(hash, transaction.Hash) {
				transactionRows.Close()
				log.L(ctx).Errorf("Existing=%s New=%s", hash, transaction.Hash)
				return database.HashMismatch
			}
		}
		transactionRows.Close()
	}

	if existing {

		// Update the transaction
		if err = s.updateTx(ctx, tx,
			sq.Update("transactions").
				Set("ttype", string(transaction.Subject.Type)).
				Set("namespace", transaction.Subject.Namespace).
				Set("ref", transaction.Subject.Reference).
				Set("signer", transaction.Subject.Signer).
				Set("hash", transaction.Hash).
				Set("created", transaction.Created).
				Set("protocol_id", transaction.ProtocolID).
				Set("status", transaction.Status).
				Set("info", transaction.Info).
				Where(sq.Eq{"id": transaction.ID}),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionTransactions, fftypes.ChangeEventTypeUpdated, transaction.Subject.Namespace, transaction.ID)
			},
		); err != nil {
			return err
		}
	} else {

		if _, err = s.insertTx(ctx, tx,
			sq.Insert("transactions").
				Columns(transactionColumns...).
				Values(
					transaction.ID,
					string(transaction.Subject.Type),
					transaction.Subject.Namespace,
					transaction.Subject.Reference,
					transaction.Subject.Signer,
					transaction.Hash,
					transaction.Created,
					transaction.ProtocolID,
					transaction.Status,
					transaction.Info,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionTransactions, fftypes.ChangeEventTypeCreated, transaction.Subject.Namespace, transaction.ID)
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
		&transaction.Subject.Type,
		&transaction.Subject.Namespace,
		&transaction.Subject.Reference,
		&transaction.Subject.Signer,
		&transaction.Hash,
		&transaction.Created,
		&transaction.ProtocolID,
		&transaction.Status,
		&transaction.Info,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "transactions")
	}
	return &transaction, nil
}

func (s *SQLCommon) GetTransactionByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.Transaction, err error) {

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

func (s *SQLCommon) GetTransactions(ctx context.Context, filter database.Filter) (message []*fftypes.Transaction, err error) {

	query, err := s.filterSelect(ctx, "", sq.Select(transactionColumns...).From("transactions"), filter, transactionFilterFieldMap, []string{"sequence"})
	if err != nil {
		return nil, err
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

	err = s.updateTx(ctx, tx, query, nil /* no change evnents for filter based updates */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
