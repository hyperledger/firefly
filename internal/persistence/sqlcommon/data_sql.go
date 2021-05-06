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
	dataColumns = []string{
		"id",
		"dtype",
		"namespace",
		"schema_entity",
		"schema_version",
		"hash",
		"created",
		"value",
	}
	dataFilterTypeMap = map[string]string{
		"type":           "dtype",
		"schema.entity":  "schema_entity",
		"schema.version": "schema_version",
	}
)

func (s *SQLCommon) UpsertData(ctx context.Context, data *fftypes.Data) (err error) {
	ctx, tx, err := s.beginTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx)

	// Do a select within the transaction to detemine if the UUID already exists
	dataRows, err := s.queryTx(ctx, tx,
		sq.Select("id").
			From("data").
			Where(sq.Eq{"id": data.ID}),
	)
	if err != nil {
		return err
	}

	schema := data.Schema
	if schema == nil {
		schema = &fftypes.SchemaRef{}
	}

	if dataRows.Next() {
		dataRows.Close()

		// Update the data
		if _, err = s.updateTx(ctx, tx,
			sq.Update("data").
				Set("dtype", string(data.Type)).
				Set("namespace", data.Namespace).
				Set("schema_entity", schema.Entity).
				Set("schema_version", schema.Version).
				Set("hash", data.Hash).
				Set("created", data.Created).
				Set("value", data.Value).
				Where(sq.Eq{"id": data.ID}),
		); err != nil {
			return err
		}
	} else {
		dataRows.Close()

		if _, err = s.insertTx(ctx, tx,
			sq.Insert("data").
				Columns(dataColumns...).
				Values(
					data.ID,
					string(data.Type),
					data.Namespace,
					schema.Entity,
					schema.Version,
					data.Hash,
					data.Created,
					data.Value,
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

func (s *SQLCommon) dataResult(ctx context.Context, row *sql.Rows) (*fftypes.Data, error) {
	data := fftypes.Data{
		Schema: &fftypes.SchemaRef{},
	}
	err := row.Scan(
		&data.ID,
		&data.Type,
		&data.Namespace,
		&data.Schema.Entity,
		&data.Schema.Version,
		&data.Hash,
		&data.Created,
		&data.Value,
	)
	if data.Schema.Entity == "" && data.Schema.Version == "" {
		data.Schema = nil
	}
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "data")
	}
	return &data, nil
}

func (s *SQLCommon) GetDataById(ctx context.Context, ns string, id *uuid.UUID) (message *fftypes.Data, err error) {

	rows, err := s.query(ctx,
		sq.Select(dataColumns...).
			From("data").
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Data '%s' not found", id)
		return nil, nil
	}

	data, err := s.dataResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *SQLCommon) GetData(ctx context.Context, filter persistence.Filter) (message []*fftypes.Data, err error) {

	query, err := s.filterSelect(ctx, sq.Select(dataColumns...).From("data"), filter, dataFilterTypeMap)
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := []*fftypes.Data{}
	for rows.Next() {
		d, err := s.dataResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		data = append(data, d)
	}

	return data, err

}
