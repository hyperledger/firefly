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
	pinColumns = []string{
		"masked",
		"hash",
		"batch_id",
		"created",
	}
	pinFilterTypeMap = map[string]string{
		"batch": "batch_id",
	}
)

func (s *SQLCommon) UpsertPin(ctx context.Context, pin *fftypes.Pin) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	if _, err = s.insertTx(ctx, tx,
		sq.Insert("pins").
			Columns(pinColumns...).
			Values(
				pin.Masked,
				pin.Hash,
				pin.Batch,
				pin.Created,
			),
	); err != nil {
		// Check it's not just that it already exsits (edge case, so we optimize for insert)
		pinRows, queryErr := s.queryTx(ctx, tx,
			sq.Select("masked").
				From("pins").
				Where(sq.Eq{
					"hash":     pin.Hash,
					"batch_id": pin.Batch,
				}),
		)
		existing := false
		if queryErr == nil {
			existing = pinRows.Next()
			pinRows.Close()
		}
		if !existing {
			// Something else went wrong - return the original error
			return err
		}
		log.L(ctx).Debugf("Pin already existed")
		return nil
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) pinResult(ctx context.Context, row *sql.Rows) (*fftypes.Pin, error) {
	pin := fftypes.Pin{}
	err := row.Scan(
		&pin.Masked,
		&pin.Hash,
		&pin.Batch,
		&pin.Created,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "pins")
	}
	return &pin, nil
}

func (s *SQLCommon) GetPins(ctx context.Context, filter database.Filter) (message []*fftypes.Pin, err error) {

	query, err := s.filterSelect(ctx, "", sq.Select(pinColumns...).From("pins"), filter, pinFilterTypeMap)
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pin := []*fftypes.Pin{}
	for rows.Next() {
		d, err := s.pinResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		pin = append(pin, d)
	}

	return pin, err

}

func (s *SQLCommon) DeletePin(ctx context.Context, hash *fftypes.Bytes32, batch *fftypes.UUID) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	err = s.deleteTx(ctx, tx, sq.Delete("pins").Where(sq.Eq{
		"hash":     hash,
		"batch_id": batch,
	}))
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
