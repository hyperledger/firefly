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
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

var (
	pinColumns = []string{
		"masked",
		"hash",
		"batch_id",
		"batch_hash",
		"idx",
		"signer",
		"dispatched",
		"created",
	}
	pinFilterFieldMap = map[string]string{
		"batch":     "batch_id",
		"batchhash": "batch_hash",
		"index":     "idx",
	}
)

const pinsTable = "pins"

func (s *SQLCommon) UpsertPin(ctx context.Context, pin *core.Pin) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	// Do a select within the transaction to detemine if the UUID already exists
	pinRows, tx, err := s.queryTx(ctx, pinsTable, tx,
		sq.Select(sequenceColumn, "masked", "dispatched").
			From(pinsTable).
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
			return i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, pinsTable)
		}
		// Pin's can only go from undispatched, to dispatched - so no update here.
		log.L(ctx).Debugf("Existing pin returned at sequence %d", pin.Sequence)
	} else {
		pinRows.Close()
		if err = s.attemptPinInsert(ctx, tx, pin); err != nil {
			return err
		}

	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) attemptPinInsert(ctx context.Context, tx *txWrapper, pin *core.Pin) (err error) {
	pin.Sequence, err = s.insertTx(ctx, pinsTable, tx,
		s.setPinInsertValues(sq.Insert(pinsTable).Columns(pinColumns...), pin),
		func() {
			log.L(ctx).Debugf("Triggering creation event for pin %d", pin.Sequence)
			s.callbacks.OrderedCollectionEvent(database.CollectionPins, core.ChangeEventTypeCreated, pin.Sequence)
		},
	)
	return err
}

func (s *SQLCommon) setPinInsertValues(query sq.InsertBuilder, pin *core.Pin) sq.InsertBuilder {
	return query.Values(
		pin.Masked,
		pin.Hash,
		pin.Batch,
		pin.BatchHash,
		pin.Index,
		pin.Signer,
		pin.Dispatched,
		pin.Created,
	)
}

func (s *SQLCommon) InsertPins(ctx context.Context, pins []*core.Pin) error {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	if s.features.MultiRowInsert {
		query := sq.Insert(pinsTable).Columns(pinColumns...)
		for _, pin := range pins {
			query = s.setPinInsertValues(query, pin)
		}
		sequences := make([]int64, len(pins))
		err := s.insertTxRows(ctx, pinsTable, tx, query, func() {
			for i, pin := range pins {
				pin.Sequence = sequences[i]
				s.callbacks.OrderedCollectionEvent(database.CollectionPins, core.ChangeEventTypeCreated, pin.Sequence)
			}
		}, sequences, true /* we want the caller to be able to retry with individual upserts */)
		if err != nil {
			return err
		}
	} else {
		// Fall back to individual inserts grouped in a TX
		for _, pin := range pins {
			if err := s.attemptPinInsert(ctx, tx, pin); err != nil {
				return err
			}
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) pinResult(ctx context.Context, row *sql.Rows) (*core.Pin, error) {
	pin := core.Pin{}
	err := row.Scan(
		&pin.Masked,
		&pin.Hash,
		&pin.Batch,
		&pin.BatchHash,
		&pin.Index,
		&pin.Signer,
		&pin.Dispatched,
		&pin.Created,
		&pin.Sequence,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, pinsTable)
	}
	return &pin, nil
}

func (s *SQLCommon) GetPins(ctx context.Context, filter database.Filter) (message []*core.Pin, fr *database.FilterResult, err error) {

	cols := append([]string{}, pinColumns...)
	cols = append(cols, sequenceColumn)
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(cols...).From(pinsTable), filter, pinFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, pinsTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	pin := []*core.Pin{}
	for rows.Next() {
		d, err := s.pinResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		pin = append(pin, d)
	}

	return pin, s.queryRes(ctx, pinsTable, tx, fop, fi), err

}

func (s *SQLCommon) UpdatePins(ctx context.Context, filter database.Filter, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update(pinsTable), update, pinFilterFieldMap)
	if err != nil {
		return err
	}

	query, err = s.filterUpdate(ctx, query, filter, pinFilterFieldMap)
	if err != nil {
		return err
	}

	_, err = s.updateTx(ctx, pinsTable, tx, query, nil /* no change events filter based update */)
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

	err = s.deleteTx(ctx, pinsTable, tx, sq.Delete(pinsTable).Where(sq.Eq{
		sequenceColumn: sequence,
	}),
		func() {
			s.callbacks.OrderedCollectionEvent(database.CollectionPins, core.ChangeEventTypeDeleted, sequence)
		})
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
