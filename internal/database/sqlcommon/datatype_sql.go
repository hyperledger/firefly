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
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

var (
	datatypeColumns = []string{
		"id",
		"message_id",
		"validator",
		"namespace",
		"name",
		"version",
		"hash",
		"created",
		"value",
	}
	datatypeFilterFieldMap = map[string]string{
		"message": "message_id",
	}
)

const datatypesTable = "datatypes"

func (s *SQLCommon) UpsertDatatype(ctx context.Context, datatype *core.Datatype, allowExisting bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		// Do a select within the transaction to detemine if the UUID already exists
		datatypeRows, _, err := s.queryTx(ctx, datatypesTable, tx,
			sq.Select("id").
				From(datatypesTable).
				Where(sq.Eq{"id": datatype.ID}),
		)
		if err != nil {
			return err
		}
		existing = datatypeRows.Next()
		datatypeRows.Close()
	}

	if existing {

		// Update the datatype
		if _, err = s.updateTx(ctx, datatypesTable, tx,
			sq.Update(datatypesTable).
				Set("message_id", datatype.Message).
				Set("validator", string(datatype.Validator)).
				Set("namespace", datatype.Namespace).
				Set("name", datatype.Name).
				Set("version", datatype.Version).
				Set("hash", datatype.Hash).
				Set("created", datatype.Created).
				Set("value", datatype.Value).
				Where(sq.Eq{"id": datatype.ID}),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionDataTypes, core.ChangeEventTypeUpdated, datatype.Namespace, datatype.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, datatypesTable, tx,
			sq.Insert(datatypesTable).
				Columns(datatypeColumns...).
				Values(
					datatype.ID,
					datatype.Message,
					string(datatype.Validator),
					datatype.Namespace,
					datatype.Name,
					datatype.Version,
					datatype.Hash,
					datatype.Created,
					datatype.Value,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionDataTypes, core.ChangeEventTypeCreated, datatype.Namespace, datatype.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) datatypeResult(ctx context.Context, row *sql.Rows) (*core.Datatype, error) {
	var datatype core.Datatype
	err := row.Scan(
		&datatype.ID,
		&datatype.Message,
		&datatype.Validator,
		&datatype.Namespace,
		&datatype.Name,
		&datatype.Version,
		&datatype.Hash,
		&datatype.Created,
		&datatype.Value,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, datatypesTable)
	}
	return &datatype, nil
}

func (s *SQLCommon) getDatatypeEq(ctx context.Context, eq sq.Eq, textName string) (message *core.Datatype, err error) {

	rows, _, err := s.query(ctx, datatypesTable,
		sq.Select(datatypeColumns...).
			From(datatypesTable).
			Where(eq),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Datatype '%s' not found", textName)
		return nil, nil
	}

	datatype, err := s.datatypeResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return datatype, nil
}

func (s *SQLCommon) GetDatatypeByID(ctx context.Context, id *fftypes.UUID) (message *core.Datatype, err error) {
	return s.getDatatypeEq(ctx, sq.Eq{"id": id}, id.String())
}

func (s *SQLCommon) GetDatatypeByName(ctx context.Context, ns, name, version string) (message *core.Datatype, err error) {
	return s.getDatatypeEq(ctx, sq.Eq{"namespace": ns, "name": name, "version": version}, fmt.Sprintf("%s:%s", ns, name))
}

func (s *SQLCommon) GetDatatypes(ctx context.Context, filter database.Filter) (message []*core.Datatype, res *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(datatypeColumns...).From(datatypesTable), filter, datatypeFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, datatypesTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	datatypes := []*core.Datatype{}
	for rows.Next() {
		datatype, err := s.datatypeResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		datatypes = append(datatypes, datatype)
	}

	return datatypes, s.queryRes(ctx, datatypesTable, tx, fop, fi), err

}

func (s *SQLCommon) UpdateDatatype(ctx context.Context, id *fftypes.UUID, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update(datatypesTable), update, datatypeFilterFieldMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"id": id})

	_, err = s.updateTx(ctx, datatypesTable, tx, query, nil /* no change events for filter based updates */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
