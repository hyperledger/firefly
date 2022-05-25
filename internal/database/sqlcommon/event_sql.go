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
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

var (
	eventColumns = []string{
		"id",
		"etype",
		"namespace",
		"ref",
		"cid",
		"tx_id",
		"topic",
		"created",
	}
	eventFilterFieldMap = map[string]string{
		"type":       "etype",
		"reference":  "ref",
		"correlator": "cid",
		"tx":         "tx_id",
	}
)

// Events are special.
//
// They are an ordered sequence of recorded state, that must be detected and processed in order.
//
// We choose (today) to coordinate the emission of these, into a DB transaction where the other
// state changes happen - so the event is assured atomically to happen "after" the other state
// changes, but also not to be lost. Downstream fan-out of those events occurs via
// Webhook/WebSocket (.../NATS/Kafka) pluggable pub/sub interfaces.
//
// Implementing this single stream of incrementing (note not guaranteed to be gapless) ordered
// items on top of a SQL database, means taking a lock (see below).
// This is not safe to do unless you are really sure what other locks will be taken after
// that in the transaction. So we defer the emission of the events to a pre-commit capture.
func (s *SQLCommon) InsertEvent(ctx context.Context, event *core.Event) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	event.Sequence = -1 // the sequence is not allocated until the post-commit callback
	s.addPreCommitEvent(tx, event)
	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) setEventInsertValues(query sq.InsertBuilder, event *core.Event) sq.InsertBuilder {
	return query.Values(
		event.ID,
		string(event.Type),
		event.Namespace,
		event.Reference,
		event.Correlator,
		event.Transaction,
		event.Topic,
		event.Created,
	)
}

const eventsTable = "events"

func (s *SQLCommon) eventInserted(ctx context.Context, event *core.Event) {
	s.callbacks.OrderedUUIDCollectionNSEvent(database.CollectionEvents, core.ChangeEventTypeCreated, event.Namespace, event.ID, event.Sequence)
	log.L(ctx).Infof("Emitted %s event %s for %s:%s (correlator=%v,topic=%s)", event.Type, event.ID, event.Namespace, event.Reference, event.Correlator, event.Topic)
}

func (s *SQLCommon) insertEventsPreCommit(ctx context.Context, tx *txWrapper, events []*core.Event) (err error) {

	// We take the cost of a full table lock on the events table.
	// This allows us to rely on the sequence to always be increasing, even when writing events
	// concurrently (it does not guarantee we won't get a gap in the sequences).
	if err = s.lockTableExclusiveTx(ctx, eventsTable, tx); err != nil {
		return err
	}

	if s.features.MultiRowInsert {
		query := sq.Insert(eventsTable).Columns(eventColumns...)
		for _, event := range events {
			query = s.setEventInsertValues(query, event)
		}
		sequences := make([]int64, len(events))
		err := s.insertTxRows(ctx, eventsTable, tx, query, func() {
			for i, event := range events {
				event.Sequence = sequences[i]
				s.eventInserted(ctx, event)
			}
		}, sequences, true /* we want the caller to be able to retry with individual upserts */)
		if err != nil {
			return err
		}
	} else {
		// Fall back to individual inserts grouped in a TX
		for _, event := range events {
			query := s.setEventInsertValues(sq.Insert(eventsTable).Columns(eventColumns...), event)
			event.Sequence, err = s.insertTx(ctx, eventsTable, tx, query, func() {
				s.eventInserted(ctx, event)
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *SQLCommon) eventResult(ctx context.Context, row *sql.Rows) (*core.Event, error) {
	var event core.Event
	err := row.Scan(
		&event.ID,
		&event.Type,
		&event.Namespace,
		&event.Reference,
		&event.Correlator,
		&event.Transaction,
		&event.Topic,
		&event.Created,
		// Must be added to the list of columns in all selects
		&event.Sequence,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, eventsTable)
	}
	return &event, nil
}

func (s *SQLCommon) GetEventByID(ctx context.Context, id *fftypes.UUID) (message *core.Event, err error) {

	cols := append([]string{}, eventColumns...)
	cols = append(cols, sequenceColumn)
	rows, _, err := s.query(ctx, eventsTable,
		sq.Select(cols...).
			From(eventsTable).
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

func (s *SQLCommon) GetEvents(ctx context.Context, filter database.Filter) (message []*core.Event, res *database.FilterResult, err error) {

	cols := append([]string{}, eventColumns...)
	cols = append(cols, sequenceColumn)
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(cols...).From(eventsTable), filter, eventFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, eventsTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	events := []*core.Event{}
	for rows.Next() {
		event, err := s.eventResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		events = append(events, event)
	}

	return events, s.queryRes(ctx, eventsTable, tx, fop, fi), err

}

func (s *SQLCommon) UpdateEvent(ctx context.Context, id *fftypes.UUID, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update(eventsTable), update, eventFilterFieldMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"id": id})

	_, err = s.updateTx(ctx, eventsTable, tx, query, nil /* no change events on filter based update */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
