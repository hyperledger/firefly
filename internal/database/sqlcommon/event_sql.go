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
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var (
	eventColumns = []string{
		"id",
		"etype",
		"namespace",
		"ref",
		"created",
	}
	eventFilterFieldMap = map[string]string{
		"type":      "etype",
		"reference": "ref",
		"group":     "group_hash",
	}
)

func (s *SQLCommon) InsertEvent(ctx context.Context, event *fftypes.Event) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	// This is special to events - we take the cost of a full table lock on the events table, so
	// that nobody can add rows that increment the sequence, until our transaction has committed.
	// This allows us to rely on the sequence to always be increasing, even when writing events
	// concurrently (it does not guarantee we won't get a gap in the sequences).
	if err = s.lockTableExclusiveTx(ctx, tx, "events"); err != nil {
		return err
	}

	event.Sequence, err = s.insertTx(ctx, tx,
		sq.Insert("events").
			Columns(eventColumns...).
			Values(
				event.ID,
				string(event.Type),
				event.Namespace,
				event.Reference,
				event.Created,
			),
		func() {
			s.callbacks.OrderedUUIDCollectionNSEvent(database.CollectionEvents, fftypes.ChangeEventTypeCreated, event.Namespace, event.ID, event.Sequence)
		},
	)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) eventResult(ctx context.Context, row *sql.Rows) (*fftypes.Event, error) {
	var event fftypes.Event
	err := row.Scan(
		&event.ID,
		&event.Type,
		&event.Namespace,
		&event.Reference,
		&event.Created,
		// Must be added to the list of columns in all selects
		&event.Sequence,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "events")
	}
	return &event, nil
}

func (s *SQLCommon) GetEventByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.Event, err error) {

	cols := append([]string{}, eventColumns...)
	cols = append(cols, sequenceColumn)
	rows, _, err := s.query(ctx,
		sq.Select(cols...).
			From("events").
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Event '%s' not found", id)
		return nil, nil
	}

	event, err := s.eventResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return event, nil
}

func (s *SQLCommon) GetEvents(ctx context.Context, filter database.Filter) (message []*fftypes.Event, res *database.FilterResult, err error) {

	cols := append([]string{}, eventColumns...)
	cols = append(cols, sequenceColumn)
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(cols...).From("events"), filter, eventFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	events := []*fftypes.Event{}
	for rows.Next() {
		event, err := s.eventResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		events = append(events, event)
	}

	return events, s.queryRes(ctx, tx, "events", fop, fi), err

}

func (s *SQLCommon) UpdateEvent(ctx context.Context, id *fftypes.UUID, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update("events"), update, eventFilterFieldMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"id": id})

	_, err = s.updateTx(ctx, tx, query, nil /* no change events on filter based update */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
