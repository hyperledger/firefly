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
)

var (
	schemaColumns = []string{
		"id",
		"stype",
		"namespace",
		"entity",
		"version",
		"hash",
		"created",
		"value",
	}
)

func (s *SQLCommon) UpsertSchema(ctx context.Context, schema *fftypes.Schema) (err error) {
	ctx, tx, err := s.beginTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx)

	// Do a select within the transaction to detemine if the UUID already exists
	schemaRows, err := s.queryTx(ctx, tx,
		sq.Select("id").
			From("schemas").
			Where(sq.Eq{"id": schema.ID}),
	)
	if err != nil {
		return err
	}

	if schemaRows.Next() {
		schemaRows.Close()

		// Update the schema
		if _, err = s.updateTx(ctx, tx,
			sq.Update("schemas").
				Set("stype", string(schema.Type)).
				Set("namespace", schema.Namespace).
				Set("entity", schema.Entity).
				Set("version", schema.Version).
				Set("hash", schema.Hash).
				Set("created", schema.Created).
				Set("value", schema.Value).
				Where(sq.Eq{"id": schema.ID}),
		); err != nil {
			return err
		}
	} else {
		schemaRows.Close()

		if _, err = s.insertTx(ctx, tx,
			sq.Insert("schemas").
				Columns(schemaColumns...).
				Values(
					schema.ID,
					string(schema.Type),
					schema.Namespace,
					schema.Entity,
					schema.Version,
					schema.Hash,
					schema.Created,
					schema.Value,
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

func (s *SQLCommon) schemaResult(ctx context.Context, row *sql.Rows) (*fftypes.Schema, error) {
	var schema fftypes.Schema
	err := row.Scan(
		&schema.ID,
		&schema.Type,
		&schema.Namespace,
		&schema.Entity,
		&schema.Version,
		&schema.Hash,
		&schema.Created,
		&schema.Value,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "schemas")
	}
	return &schema, nil
}

func (s *SQLCommon) GetSchemaById(ctx context.Context, id *uuid.UUID) (message *fftypes.Schema, err error) {

	rows, err := s.query(ctx,
		sq.Select(schemaColumns...).
			From("schemas").
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Schema '%s' not found", id)
		return nil, nil
	}

	schema, err := s.schemaResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return schema, nil
}
