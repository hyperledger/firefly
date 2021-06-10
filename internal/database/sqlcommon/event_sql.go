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
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

var (
	eventColumns = []string{
		"id",
		"etype",
		"namespace",
		"ref",
		"group_hash",
		"created",
	}
	eventFilterTypeMap = map[string]string{
		"type":      "etype",
		"reference": "ref",
		"group":     "group_hash",
	}
)

func (s *SQLCommon) UpsertEvent(ctx context.Context, event *fftypes.Event, allowExisting bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		// Do a select within the event to detemine if the UUID already exists
		eventRows, err := s.queryTx(ctx, tx,
			sq.Select("id").
				From("events").
				Where(sq.Eq{"id": event.ID}),
		)
		if err != nil {
			return err
		}
		existing = eventRows.Next()
		eventRows.Close()
	}

	if existing {
		// Update the event
		if err = s.updateTx(ctx, tx,
			sq.Update("events").
				Set("etype", string(event.Type)).
				Set("namespace", event.Namespace).
				Set("ref", event.Reference).
				Set("group_hash", event.Group).
				Set("created", event.Created).
				Where(sq.Eq{"id": event.ID}),
		); err != nil {
			return err
		}
	} else {
		sequence, err := s.insertTx(ctx, tx,
			sq.Insert("events").
				Columns(eventColumns...).
				Values(
					event.ID,
					string(event.Type),
					event.Namespace,
					event.Reference,
					event.Group,
					event.Created,
				),
		)
		if err != nil {
			return err
		}

		s.postCommitEvent(tx, func() {
			s.callbacks.EventCreated(sequence)
		})

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
		&event.Group,
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
	cols = append(cols, s.provider.SequenceField(""))
	rows, err := s.query(ctx,
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

func (s *SQLCommon) GetEvents(ctx context.Context, filter database.Filter) (message []*fftypes.Event, err error) {

	cols := append([]string{}, eventColumns...)
	cols = append(cols, s.provider.SequenceField(""))
	query, err := s.filterSelect(ctx, "", sq.Select(cols...).From("events"), filter, eventFilterTypeMap)
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []*fftypes.Event{}
	for rows.Next() {
		event, err := s.eventResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, err

}

func (s *SQLCommon) UpdateEvent(ctx context.Context, id *fftypes.UUID, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update("events"), update, eventFilterTypeMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"id": id})

	err = s.updateTx(ctx, tx, query)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
