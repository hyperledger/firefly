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
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

var (
	dataColumns = []string{
		"id",
		"validator",
		"namespace",
		"def_name",
		"def_version",
		"hash",
		"created",
		"value",
	}
	dataFilterTypeMap = map[string]string{
		"validator":          "validator",
		"definition.name":    "def_name",
		"definition.version": "def_version",
	}
)

func (s *SQLCommon) UpsertData(ctx context.Context, data *fftypes.Data, allowExisting, allowHashUpdate bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		// Do a select within the transaction to detemine if the UUID already exists
		dataRows, err := s.queryTx(ctx, tx,
			sq.Select("hash").
				From("data").
				Where(sq.Eq{"id": data.ID}),
		)
		if err != nil {
			return err
		}

		existing = dataRows.Next()
		if existing && !allowHashUpdate {
			var hash *fftypes.Bytes32
			_ = dataRows.Scan(&hash)
			if !fftypes.SafeHashCompare(hash, data.Hash) {
				dataRows.Close()
				log.L(ctx).Errorf("Existing=%s New=%s", hash, data.Hash)
				return database.HashMismatch
			}
		}
		dataRows.Close()
	}

	dataDef := data.Definition
	if dataDef == nil {
		dataDef = &fftypes.DataDefinitionRef{}
	}

	if existing {
		// Update the data
		if _, err = s.updateTx(ctx, tx,
			sq.Update("data").
				Set("validator", string(data.Validator)).
				Set("namespace", data.Namespace).
				Set("def_name", dataDef.Name).
				Set("def_version", dataDef.Version).
				Set("hash", data.Hash).
				Set("created", data.Created).
				Set("value", data.Value).
				Where(sq.Eq{"id": data.ID}),
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("data").
				Columns(dataColumns...).
				Values(
					data.ID,
					string(data.Validator),
					data.Namespace,
					dataDef.Name,
					dataDef.Version,
					data.Hash,
					data.Created,
					data.Value,
				),
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) dataResult(ctx context.Context, row *sql.Rows) (*fftypes.Data, error) {
	data := fftypes.Data{
		Definition: &fftypes.DataDefinitionRef{},
	}
	err := row.Scan(
		&data.ID,
		&data.Validator,
		&data.Namespace,
		&data.Definition.Name,
		&data.Definition.Version,
		&data.Hash,
		&data.Created,
		&data.Value,
	)
	if data.Definition.Name == "" && data.Definition.Version == "" {
		data.Definition = nil
	}
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "data")
	}
	return &data, nil
}

func (s *SQLCommon) GetDataById(ctx context.Context, id *fftypes.UUID) (message *fftypes.Data, err error) {

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

func (s *SQLCommon) GetData(ctx context.Context, filter database.Filter) (message []*fftypes.Data, err error) {

	query, err := s.filterSelect(ctx, "", sq.Select(dataColumns...).From("data"), filter, dataFilterTypeMap)
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

func (s *SQLCommon) GetDataRefs(ctx context.Context, filter database.Filter) (message fftypes.DataRefs, err error) {

	query, err := s.filterSelect(ctx, "", sq.Select("id", "hash").From("data"), filter, dataFilterTypeMap)
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	refs := fftypes.DataRefs{}
	for rows.Next() {
		ref := fftypes.DataRef{}
		err := rows.Scan(
			&ref.ID,
			&ref.Hash,
		)
		if err != nil {
			return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "data")
		}
		refs = append(refs, ref)
	}

	return refs, err

}

func (s *SQLCommon) UpdateData(ctx context.Context, id *fftypes.UUID, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(ctx, "", sq.Update("data"), update, dataFilterTypeMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"id": id})

	_, err = s.updateTx(ctx, tx, query)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
