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
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
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

func (s *SQLCommon) UpsertDatatype(ctx context.Context, datatype *fftypes.Datatype, allowExisting bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		// Do a select within the transaction to detemine if the UUID already exists
		datatypeRows, _, err := s.queryTx(ctx, tx,
			sq.Select("id").
				From("datatypes").
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
		if err = s.updateTx(ctx, tx,
			sq.Update("datatypes").
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
				s.callbacks.UUIDCollectionNSEvent(database.CollectionDataTypes, fftypes.ChangeEventTypeUpdated, datatype.Namespace, datatype.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("datatypes").
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
				s.callbacks.UUIDCollectionNSEvent(database.CollectionDataTypes, fftypes.ChangeEventTypeCreated, datatype.Namespace, datatype.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) datatypeResult(ctx context.Context, row *sql.Rows) (*fftypes.Datatype, error) {
	var datatype fftypes.Datatype
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
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "datatypes")
	}
	return &datatype, nil
}

func (s *SQLCommon) getDatatypeEq(ctx context.Context, eq sq.Eq, textName string) (message *fftypes.Datatype, err error) {

	rows, _, err := s.query(ctx,
		sq.Select(datatypeColumns...).
			From("datatypes").
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

func (s *SQLCommon) GetDatatypeByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.Datatype, err error) {
	return s.getDatatypeEq(ctx, sq.Eq{"id": id}, id.String())
}

func (s *SQLCommon) GetDatatypeByName(ctx context.Context, ns, name, version string) (message *fftypes.Datatype, err error) {
	return s.getDatatypeEq(ctx, sq.Eq{"namespace": ns, "name": name, "version": version}, fmt.Sprintf("%s:%s", ns, name))
}

func (s *SQLCommon) GetDatatypes(ctx context.Context, filter database.Filter) (message []*fftypes.Datatype, res *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(datatypeColumns...).From("datatypes"), filter, datatypeFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	datatypes := []*fftypes.Datatype{}
	for rows.Next() {
		datatype, err := s.datatypeResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		datatypes = append(datatypes, datatype)
	}

	return datatypes, s.queryRes(ctx, tx, "datatypes", fop, fi), err

}

func (s *SQLCommon) UpdateDatatype(ctx context.Context, id *fftypes.UUID, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update("datatypes"), update, datatypeFilterFieldMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"id": id})

	err = s.updateTx(ctx, tx, query, nil /* no change events for filter based updates */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
