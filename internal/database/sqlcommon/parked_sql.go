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
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

var (
	parkedColumns = []string{
		"pin",
		"ledger_id",
		"batch_id",
		"created",
	}
	parkedFilterTypeMap = map[string]string{
		"ledger": "ledger_id",
		"batch":  "batch_id",
	}
)

func (s *SQLCommon) InsertParked(ctx context.Context, parked *fftypes.Parked) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	sequence, err := s.insertTx(ctx, tx,
		sq.Insert("parked").
			Columns(parkedColumns...).
			Values(
				parked.Pin,
				parked.Ledger,
				parked.Batch,
				parked.Created,
			),
	)
	if err != nil {
		return err
	}
	parked.Sequence = sequence

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) parkedResult(ctx context.Context, row *sql.Rows) (*fftypes.Parked, error) {
	parked := fftypes.Parked{}
	err := row.Scan(
		&parked.Pin,
		&parked.Ledger,
		&parked.Batch,
		&parked.Created,
		&parked.Sequence,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "parked")
	}
	return &parked, nil
}

func (s *SQLCommon) GetParked(ctx context.Context, filter database.Filter) (message []*fftypes.Parked, err error) {

	cols := append([]string{}, parkedColumns...)
	cols = append(cols, s.provider.SequenceField(""))
	query, err := s.filterSelect(ctx, "", sq.Select(cols...).From("parked"), filter, parkedFilterTypeMap)
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	parked := []*fftypes.Parked{}
	for rows.Next() {
		d, err := s.parkedResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		parked = append(parked, d)
	}

	return parked, err

}

func (s *SQLCommon) DeleteParked(ctx context.Context, sequence int64) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	err = s.deleteTx(ctx, tx, sq.Delete("parked").Where(sq.Eq{
		s.provider.SequenceField(""): sequence,
	}))
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
