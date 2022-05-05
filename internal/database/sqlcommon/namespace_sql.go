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
	namespaceColumns = []string{
		"id",
		"message_id",
		"ntype",
		"name",
		"description",
		"created",
	}
	namespaceFilterFieldMap = map[string]string{
		"message": "message_id",
		"type":    "ntype",
	}
)

const namespacesTable = "namespaces"

func (s *SQLCommon) UpsertNamespace(ctx context.Context, namespace *core.Namespace, allowExisting bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		// Do a select within the transaction to determine if the UUID already exists
		namespaceRows, _, err := s.queryTx(ctx, namespacesTable, tx,
			sq.Select("id").
				From(namespacesTable).
				Where(sq.Eq{"name": namespace.Name}),
		)
		if err != nil {
			return err
		}
		existing = namespaceRows.Next()

		if existing {
			var id fftypes.UUID
			_ = namespaceRows.Scan(&id)
			if namespace.ID != nil {
				if *namespace.ID != id {
					namespaceRows.Close()
					return database.IDMismatch
				}
			}
			namespace.ID = &id // Update on returned object
		}
		namespaceRows.Close()
	}

	if existing {
		// Update the namespace
		if _, err = s.updateTx(ctx, namespacesTable, tx,
			sq.Update(namespacesTable).
				// Note we do not update ID
				Set("message_id", namespace.Message).
				Set("ntype", string(namespace.Type)).
				Set("name", namespace.Name).
				Set("description", namespace.Description).
				Set("created", namespace.Created).
				Where(sq.Eq{"name": namespace.Name}),
			func() {
				s.callbacks.UUIDCollectionEvent(database.CollectionNamespaces, core.ChangeEventTypeUpdated, namespace.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if namespace.ID == nil {
			namespace.ID = fftypes.NewUUID()
		}

		if _, err = s.insertTx(ctx, namespacesTable, tx,
			sq.Insert(namespacesTable).
				Columns(namespaceColumns...).
				Values(
					namespace.ID,
					namespace.Message,
					string(namespace.Type),
					namespace.Name,
					namespace.Description,
					namespace.Created,
				),
			func() {
				s.callbacks.UUIDCollectionEvent(database.CollectionNamespaces, core.ChangeEventTypeCreated, namespace.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) namespaceResult(ctx context.Context, row *sql.Rows) (*core.Namespace, error) {
	namespace := core.Namespace{}
	err := row.Scan(
		&namespace.ID,
		&namespace.Message,
		&namespace.Type,
		&namespace.Name,
		&namespace.Description,
		&namespace.Created,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, namespacesTable)
	}
	return &namespace, nil
}

func (s *SQLCommon) getNamespaceEq(ctx context.Context, eq sq.Eq, textName string) (message *core.Namespace, err error) {
	rows, _, err := s.query(ctx, namespacesTable,
		sq.Select(namespaceColumns...).
			From(namespacesTable).
			Where(eq),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Namespace '%s' not found", textName)
		return nil, nil
	}

	namespace, err := s.namespaceResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return namespace, nil
}

func (s *SQLCommon) GetNamespace(ctx context.Context, name string) (message *core.Namespace, err error) {
	return s.getNamespaceEq(ctx, sq.Eq{"name": name}, name)
}

func (s *SQLCommon) GetNamespaceByID(ctx context.Context, id *fftypes.UUID) (ns *core.Namespace, err error) {
	return s.getNamespaceEq(ctx, sq.Eq{"id": id}, id.String())
}

func (s *SQLCommon) GetNamespaces(ctx context.Context, filter database.Filter) (message []*core.Namespace, fr *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(namespaceColumns...).From(namespacesTable), filter, namespaceFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, namespacesTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	namespace := []*core.Namespace{}
	for rows.Next() {
		d, err := s.namespaceResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		namespace = append(namespace, d)
	}

	return namespace, s.queryRes(ctx, namespacesTable, tx, fop, fi), err

}

func (s *SQLCommon) DeleteNamespace(ctx context.Context, id *fftypes.UUID) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	err = s.deleteTx(ctx, namespacesTable, tx, sq.Delete(namespacesTable).Where(sq.Eq{
		"id": id,
	}),
		func() {
			s.callbacks.UUIDCollectionEvent(database.CollectionNamespaces, core.ChangeEventTypeDeleted, id)
		})
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
