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
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var (
	pinColumns = []string{
		"masked",
		"hash",
		"batch_id",
		"idx",
		"dispatched",
		"created",
	}
	pinFilterFieldMap = map[string]string{
		"batch": "batch_id",
		"index": "idx",
	}
)

func (s *SQLCommon) UpsertPin(ctx context.Context, pin *fftypes.Pin) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	// Do a select within the transaction to detemine if the UUID already exists
	pinRows, tx, err := s.queryTx(ctx, tx,
		sq.Select(sequenceColumn, "masked", "dispatched").
			From("pins").
			Where(sq.Eq{
				"hash":     pin.Hash,
				"batch_id": pin.Batch,
				"idx":      pin.Index,
			}))
	if err != nil {
		return err
	}
	existing := pinRows.Next()

	if existing {
		err := pinRows.Scan(&pin.Sequence, &pin.Masked, &pin.Dispatched)
		pinRows.Close()
		if err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "pins")
		}
		// Pin's can only go from undispatched, to dispatched - so no update here.
		log.L(ctx).Debugf("Existing pin returned at sequence %d", pin.Sequence)
	} else {
		pinRows.Close()
		if pin.Sequence, err = s.insertTx(ctx, tx,
			sq.Insert("pins").
				Columns(pinColumns...).
				Values(
					pin.Masked,
					pin.Hash,
					pin.Batch,
					pin.Index,
					pin.Dispatched,
					pin.Created,
				),
			func() {
				s.callbacks.OrderedCollectionEvent(database.CollectionPins, fftypes.ChangeEventTypeCreated, pin.Sequence)
			},
		); err != nil {
			return err
		}

	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) pinResult(ctx context.Context, row *sql.Rows) (*fftypes.Pin, error) {
	pin := fftypes.Pin{}
	err := row.Scan(
		&pin.Masked,
		&pin.Hash,
		&pin.Batch,
		&pin.Index,
		&pin.Dispatched,
		&pin.Created,
		&pin.Sequence,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "pins")
	}
	return &pin, nil
}

func (s *SQLCommon) GetPins(ctx context.Context, filter database.Filter) (message []*fftypes.Pin, fr *database.FilterResult, err error) {

	cols := append([]string{}, pinColumns...)
	cols = append(cols, sequenceColumn)
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(cols...).From("pins"), filter, pinFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	pin := []*fftypes.Pin{}
	for rows.Next() {
		d, err := s.pinResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		pin = append(pin, d)
	}

	return pin, s.queryRes(ctx, tx, "pins", fop, fi), err

}

func (s *SQLCommon) SetPinDispatched(ctx context.Context, sequence int64) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	err = s.updateTx(ctx, tx, sq.
		Update("pins").
		Set("dispatched", true).
		Where(sq.Eq{
			sequenceColumn: sequence,
		}),
		func() {
			s.callbacks.OrderedCollectionEvent(database.CollectionPins, fftypes.ChangeEventTypeUpdated, sequence)
		},
	)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) DeletePin(ctx context.Context, sequence int64) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	err = s.deleteTx(ctx, tx, sq.Delete("pins").Where(sq.Eq{
		sequenceColumn: sequence,
	}),
		func() {
			s.callbacks.OrderedCollectionEvent(database.CollectionPins, fftypes.ChangeEventTypeDeleted, sequence)
		})
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
