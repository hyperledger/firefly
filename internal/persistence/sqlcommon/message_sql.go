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
	"sort"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/persistence"
)

var (
	msgColumns = []string{
		"id",
		"cid",
		"mtype",
		"author",
		"created",
		"namespace",
		"topic",
		"context",
		"group_id",
		"datahash",
		"hash",
		"confirmed",
		"tx_type",
		"tx_id",
		"batch_id",
	}
)

func (s *SQLCommon) UpsertMessage(ctx context.Context, message *fftypes.MessageRefsOnly) (err error) {
	ctx, tx, err := s.beginTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx)

	// Do a select within the transaction to detemine if the UUID already exists
	msgRows, err := s.queryTx(ctx, tx,
		sq.Select("id").
			From("messages").
			Where(sq.Eq{"id": message.ID}),
	)
	if err != nil {
		return err
	}
	defer msgRows.Close()

	if msgRows.Next() {
		// Update the message
		if _, err = s.updateTx(ctx, tx,
			sq.Update("messages").
				Set("cid", message.CID).
				Set("mtype", string(message.Type)).
				Set("author", message.Author).
				Set("created", message.Created).
				Set("namespace", message.Namespace).
				Set("topic", message.Topic).
				Set("context", message.Context).
				Set("group_id", message.Group).
				Set("datahash", message.DataHash).
				Set("hash", message.Hash).
				Set("confirmed", message.Confirmed).
				Set("tx_type", message.TX.Type).
				Set("tx_id", message.TX.ID).
				Set("batch_id", message.TX.BatchID).
				Where(sq.Eq{"id": message.ID}),
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("messages").
				Columns(msgColumns...).
				Values(
					message.ID,
					message.CID,
					string(message.Type),
					message.Author,
					message.Created,
					message.Namespace,
					message.Topic,
					message.Context,
					message.Group,
					message.DataHash,
					message.Hash,
					message.Confirmed,
					message.TX.Type,
					message.TX.ID,
					message.TX.BatchID,
				),
		); err != nil {
			return err
		}
	}

	if err = s.updateMessageDataRefs(ctx, tx, message); err != nil {
		return err
	}

	if err = s.commitTx(ctx, tx); err != nil {
		return err
	}

	return nil
}

func (s *SQLCommon) getMessageDataRefs(ctx context.Context, tx *sql.Tx, msgId *uuid.UUID) ([]*uuid.UUID, error) {
	existingRefs, err := s.queryTx(ctx, tx,
		sq.Select("data_id").
			From("messages_data").
			Where(sq.Eq{"message_id": msgId}),
	)
	if err != nil {
		return nil, err
	}

	var dataIDs []*uuid.UUID
	for existingRefs.Next() {
		var dataID uuid.UUID
		if err = existingRefs.Scan(&dataID); err != nil {
			return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "message_data")
		}
		dataIDs = append(dataIDs, &dataID)
	}

	return dataIDs, nil
}

func (s *SQLCommon) updateMessageDataRefs(ctx context.Context, tx *sql.Tx, message *fftypes.MessageRefsOnly) error {

	dataIDs, err := s.getMessageDataRefs(ctx, tx, message.ID)
	if err != nil {
		return err
	}

	// Run through the ones in the message, finding ones that already exist, and ones that need to be created
	missingRefs := make([]*uuid.UUID, 0, len(dataIDs))
	for msgDataRefIdx, msgDataRef := range message.Data {
		if msgDataRef.ID == nil {
			return i18n.NewError(ctx, i18n.MsgNullDataReferenceID, msgDataRefIdx)
		}
		var found = false
		for dataRefIdx, dataID := range dataIDs {
			if *dataID == *msgDataRef.ID {
				found = true
				// Remove it from the list, so we can use this list as ones we need to delete
				copy(dataIDs[dataRefIdx:], dataIDs[dataRefIdx+1:])
				dataIDs = dataIDs[:len(dataIDs)-1]
				break
			}
		}
		if !found {
			missingRefs = append(missingRefs, msgDataRef.ID)
		}
	}

	// Fun through the extra IDs that are no longer needed
	for _, idToDelete := range dataIDs {
		if _, err = s.deleteTx(ctx, tx,
			sq.Delete("messages_data").
				Where(sq.And{
					sq.Eq{"message_id": message.ID},
					sq.Eq{"data_id": idToDelete},
				}),
		); err != nil {
			return err
		}
	}

	// Run through the ones we need to create
	for _, newID := range missingRefs {
		// Add the linkage
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("messages_data").
				Columns(
					"message_id",
					"data_id",
				).
				Values(
					message.ID,
					newID,
				),
		); err != nil {
			return err
		}
	}

	return nil

}

func (s *SQLCommon) loadDataRefs(ctx context.Context, msgs []*fftypes.MessageRefsOnly) error {

	msgIds := make([]string, len(msgs))
	for i, m := range msgs {
		if m != nil {
			msgIds[i] = m.ID.String()
		}
	}

	existingRefs, err := s.query(ctx,
		sq.Select(
			"message_id",
			"data_id",
		).
			From("messages_data").
			Where(sq.Eq{"message_id": msgIds}),
	)
	if err != nil {
		return err
	}

	for existingRefs.Next() {
		var msgID uuid.UUID
		var dataID uuid.UUID
		if err = existingRefs.Scan(&msgID, &dataID); err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "messages_data")
		}
		for _, m := range msgs {
			if *m.ID == msgID {
				m.Data = append(m.Data, fftypes.DataRef{
					ID: &dataID,
				})
			}
		}
	}
	// Ensure we return an empty array if no entries, and a consistent order for the data
	for _, m := range msgs {
		if m.Data == nil {
			m.Data = fftypes.DataRefSortable{}
		} else {
			sort.Sort(m.Data)
		}
	}

	return nil
}

func (s *SQLCommon) msgResult(ctx context.Context, row *sql.Rows) (*fftypes.MessageRefsOnly, error) {
	var msg fftypes.MessageRefsOnly
	err := row.Scan(
		&msg.ID,
		&msg.CID,
		&msg.Type,
		&msg.Author,
		&msg.Created,
		&msg.Namespace,
		&msg.Topic,
		&msg.Context,
		&msg.Group,
		&msg.DataHash,
		&msg.Hash,
		&msg.Confirmed,
		&msg.TX.Type,
		&msg.TX.ID,
		&msg.TX.BatchID,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "messages")
	}
	return &msg, nil
}

func (s *SQLCommon) GetMessageById(ctx context.Context, id *uuid.UUID) (message *fftypes.MessageRefsOnly, err error) {

	rows, err := s.query(ctx,
		sq.Select(msgColumns...).
			From("messages").
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return nil, err
	}

	if !rows.Next() {
		log.L(ctx).Debugf("Message '%s' not found", id)
		return nil, nil
	}

	msg, err := s.msgResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	if err = s.loadDataRefs(ctx, []*fftypes.MessageRefsOnly{msg}); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *SQLCommon) GetMessages(ctx context.Context, skip, limit uint64, filter *persistence.MessageFilter) (message []*fftypes.MessageRefsOnly, err error) {

	query := sq.Select(msgColumns...).From("messages")

	if filter.ConfirmedAfter > 0 {
		query = query.Where(sq.Gt{"confirmed": filter.ConfirmedAfter})
	} else if filter.ConfrimedOnly {
		query = query.Where(sq.Gt{"confirmed": 0})
	} else if filter.UnconfrimedOnly {
		query = query.Where(sq.Eq{"confirmed": 0})
	}

	if filter.NamespaceEquals != "" {
		query = query.Where(sq.Eq{"namespace": filter.NamespaceEquals})
	}
	if filter.TypeEquals != "" {
		query = query.Where(sq.Eq{"mtype": filter.TypeEquals})
	}
	if filter.AuthorEquals != "" {
		query = query.Where(sq.Eq{"author": filter.AuthorEquals})
	}
	if filter.TopicEquals != "" {
		query = query.Where(sq.Eq{"topic": filter.TopicEquals})
	}
	if filter.ContextEquals != "" {
		query = query.Where(sq.Eq{"context": filter.ContextEquals})
	}
	if filter.GroupEquals != nil {
		query = query.Where(sq.Eq{"group_id": filter.GroupEquals})
	}
	if filter.CIDEquals != nil {
		query = query.Where(sq.Eq{"cid": filter.CIDEquals})
	}
	if filter.CreatedAfter > 0 {
		query = query.Where(sq.Gt{"created": filter.CreatedAfter})
	}
	query = query.OrderBy("confirmed,created DESC")
	if limit > 0 {
		query = query.Offset(skip).Limit(limit)
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}

	msgs := []*fftypes.MessageRefsOnly{}
	for rows.Next() {
		msg, err := s.msgResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}

	if err = s.loadDataRefs(ctx, msgs); err != nil {
		return nil, err
	}

	return msgs, err

}
