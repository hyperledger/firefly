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
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
)

var (
	offsetColumns = []string{
		"otype",
		"namespace",
		"name",
		"current",
	}
	offsetFilterTypeMap = map[string]string{
		"type": "otype",
	}
)

func (s *SQLCommon) UpsertOffset(ctx context.Context, offset *fftypes.Offset, allowExisting bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		// Do a select within the transaction to detemine if the UUID already exists
		offsetRows, err := s.queryTx(ctx, tx,
			sq.Select("otype", "namespace", "name").
				From("offsets").
				Where(
					sq.Eq{"otype": offset.Type,
						"namespace": offset.Namespace,
						"name":      offset.Name}),
		)
		if err != nil {
			return err
		}
		existing = offsetRows.Next()
		offsetRows.Close()
	}

	if existing {

		// Update the offset
		if _, err = s.updateTx(ctx, tx,
			sq.Update("offsets").
				Set("otype", string(offset.Type)).
				Set("namespace", offset.Namespace).
				Set("name", offset.Name).
				Set("current", offset.Current).
				Where(sq.Eq{"otype": offset.Type,
					"namespace": offset.Namespace,
					"name":      offset.Name}),
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("offsets").
				Columns(offsetColumns...).
				Values(
					string(offset.Type),
					offset.Namespace,
					offset.Name,
					offset.Current,
				),
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) offsetResult(ctx context.Context, row *sql.Rows) (*fftypes.Offset, error) {
	offset := fftypes.Offset{}
	err := row.Scan(
		&offset.Type,
		&offset.Namespace,
		&offset.Name,
		&offset.Current,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "offsets")
	}
	return &offset, nil
}

func (s *SQLCommon) GetOffset(ctx context.Context, t fftypes.OffsetType, ns, name string) (message *fftypes.Offset, err error) {

	rows, err := s.query(ctx,
		sq.Select(offsetColumns...).
			From("offsets").
			Where(sq.Eq{"otype": t,
				"namespace": ns,
				"name":      name}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Offset '%s:%s:%s' not found", t, ns, name)
		return nil, nil
	}

	offset, err := s.offsetResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return offset, nil
}

func (s *SQLCommon) GetOffsets(ctx context.Context, filter database.Filter) (message []*fftypes.Offset, err error) {

	query, err := s.filterSelect(ctx, "", sq.Select(offsetColumns...).From("offsets"), filter, offsetFilterTypeMap)
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	offset := []*fftypes.Offset{}
	for rows.Next() {
		d, err := s.offsetResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		offset = append(offset, d)
	}

	return offset, err

}

func (s *SQLCommon) UpdateOffset(ctx context.Context, t fftypes.OffsetType, ns, name string, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(ctx, "", sq.Update("offsets"), update, offsetFilterTypeMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"otype": t,
		"namespace": ns,
		"name":      name})

	_, err = s.updateTx(ctx, tx, query)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
