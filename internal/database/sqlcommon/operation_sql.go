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
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

var (
	opColumns = []string{
		"id",
		"tx_id",
		"optype",
		"opstatus",
		"recipient",
		"plugin",
		"backend_id",
		"created",
		"updated",
		"error",
		"info",
	}
	opFilterTypeMap = map[string]string{
		"tx":        "tx_id",
		"type":      "optype",
		"status":    "opstatus",
		"backendid": "backend_id",
	}
)

func (s *SQLCommon) UpsertOperation(ctx context.Context, operation *fftypes.Operation, allowExisting bool) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		// Do a select within the transaction to detemine if the UUID already exists
		opRows, err := s.queryTx(ctx, tx,
			sq.Select("id").
				From("operations").
				Where(sq.Eq{"id": operation.ID}),
		)
		if err != nil {
			return err
		}

		existing = opRows.Next()
		opRows.Close()
	}

	if existing {
		// Update the operation
		if err = s.updateTx(ctx, tx,
			sq.Update("operations").
				Set("tx_id", operation.Transaction).
				Set("optype", operation.Type).
				Set("opstatus", operation.Status).
				Set("recipient", operation.Recipient).
				Set("plugin", operation.Plugin).
				Set("backend_id", operation.BackendID).
				Set("created", operation.Created).
				Set("updated", operation.Updated).
				Set("error", operation.Error).
				Set("info", operation.Info).
				Where(sq.Eq{"id": operation.ID}),
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("operations").
				Columns(opColumns...).
				Values(
					operation.ID,
					operation.Transaction,
					string(operation.Type),
					string(operation.Status),
					operation.Recipient,
					operation.Plugin,
					operation.BackendID,
					operation.Created,
					operation.Updated,
					operation.Error,
					operation.Info,
				),
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) opResult(ctx context.Context, row *sql.Rows) (*fftypes.Operation, error) {
	var op fftypes.Operation
	err := row.Scan(
		&op.ID,
		&op.Transaction,
		&op.Type,
		&op.Status,
		&op.Recipient,
		&op.Plugin,
		&op.BackendID,
		&op.Created,
		&op.Updated,
		&op.Error,
		&op.Info,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "operations")
	}
	return &op, nil
}

func (s *SQLCommon) GetOperationByID(ctx context.Context, id *fftypes.UUID) (operation *fftypes.Operation, err error) {

	rows, err := s.query(ctx,
		sq.Select(opColumns...).
			From("operations").
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Operation '%s' not found", id)
		return nil, nil
	}

	op, err := s.opResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return op, nil
}

func (s *SQLCommon) GetOperations(ctx context.Context, filter database.Filter) (operation []*fftypes.Operation, err error) {

	query, err := s.filterSelect(ctx, "", sq.Select(opColumns...).From("operations"), filter, opFilterTypeMap)
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ops := []*fftypes.Operation{}
	for rows.Next() {
		op, err := s.opResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}

	return ops, err
}

func (s *SQLCommon) UpdateOperations(ctx context.Context, filter database.Filter, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update("operations"), update, opFilterTypeMap)
	if err != nil {
		return err
	}

	query, err = s.filterUpdate(ctx, "", query, filter, opFilterTypeMap)
	if err != nil {
		return err
	}

	err = s.updateTx(ctx, tx, query)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
