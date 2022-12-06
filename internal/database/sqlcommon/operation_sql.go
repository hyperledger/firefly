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
	"github.com/hyperledger/firefly-common/pkg/ffapi"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

var (
	opColumns = []string{
		"id",
		"namespace",
		"tx_id",
		"optype",
		"opstatus",
		"plugin",
		"created",
		"updated",
		"error",
		"input",
		"output",
		"retry_id",
	}
	opFilterFieldMap = map[string]string{
		"tx":     "tx_id",
		"type":   "optype",
		"status": "opstatus",
		"retry":  "retry_id",
	}
)

const operationsTable = "operations"

func (s *SQLCommon) InsertOperation(ctx context.Context, operation *core.Operation, hooks ...database.PostCompletionHook) (err error) {
	ctx, tx, autoCommit, err := s.BeginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.RollbackTx(ctx, tx, autoCommit)

	if _, err = s.InsertTx(ctx, operationsTable, tx,
		sq.Insert(operationsTable).
			Columns(opColumns...).
			Values(
				operation.ID,
				operation.Namespace,
				operation.Transaction,
				string(operation.Type),
				string(operation.Status),
				operation.Plugin,
				operation.Created,
				operation.Updated,
				operation.Error,
				operation.Input,
				operation.Output,
				operation.Retry,
			),
		func() {
			s.callbacks.UUIDCollectionNSEvent(database.CollectionOperations, core.ChangeEventTypeCreated, operation.Namespace, operation.ID)
			for _, hook := range hooks {
				hook()
			}
		},
	); err != nil {
		return err
	}

	return s.CommitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) opResult(ctx context.Context, row *sql.Rows) (*core.Operation, error) {
	var op core.Operation
	err := row.Scan(
		&op.ID,
		&op.Namespace,
		&op.Transaction,
		&op.Type,
		&op.Status,
		&op.Plugin,
		&op.Created,
		&op.Updated,
		&op.Error,
		&op.Input,
		&op.Output,
		&op.Retry,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, operationsTable)
	}
	return &op, nil
}

func (s *SQLCommon) GetOperationByID(ctx context.Context, namespace string, id *fftypes.UUID) (operation *core.Operation, err error) {

	rows, _, err := s.Query(ctx, operationsTable,
		sq.Select(opColumns...).
			From(operationsTable).
			Where(sq.Eq{"id": id, "namespace": namespace}),
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

func (s *SQLCommon) GetOperations(ctx context.Context, namespace string, filter ffapi.Filter) (operation []*core.Operation, fr *ffapi.FilterResult, err error) {

	query, fop, fi, err := s.FilterSelect(ctx, "", sq.Select(opColumns...).From(operationsTable), filter, opFilterFieldMap, []interface{}{"sequence"}, sq.Eq{"namespace": namespace})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.Query(ctx, operationsTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	ops := []*core.Operation{}
	for rows.Next() {
		op, err := s.opResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		ops = append(ops, op)
	}

	return ops, s.QueryRes(ctx, operationsTable, tx, fop, fi), err
}

func (s *SQLCommon) UpdateOperation(ctx context.Context, ns string, id *fftypes.UUID, update ffapi.Update) (err error) {

	ctx, tx, autoCommit, err := s.BeginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.RollbackTx(ctx, tx, autoCommit)

	query, err := s.BuildUpdate(sq.Update(operationsTable), update, opFilterFieldMap)
	if err != nil {
		return err
	}
	query = query.Set("updated", fftypes.Now())
	query = query.Where(sq.And{
		sq.Eq{"id": id},
		sq.Eq{"namespace": ns},
	})

	ra, err := s.UpdateTx(ctx, operationsTable, tx, query, func() {
		s.callbacks.UUIDCollectionNSEvent(database.CollectionOperations, core.ChangeEventTypeUpdated, ns, id)
	})
	if err != nil {
		return err
	}
	if ra < 1 {
		return i18n.NewError(ctx, coremsgs.Msg404NoResult)
	}

	return s.CommitTx(ctx, tx, autoCommit)
}
