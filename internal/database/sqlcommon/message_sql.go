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
	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
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
	msgFilterTypeMap = map[string]string{
		"type":    "mtype",
		"tx.type": "tx_type",
		"tx.id":   "tx_id",
		"batchid": "batch_id",
		"group":   "group_id",
	}
)

func (s *SQLCommon) UpsertMessage(ctx context.Context, message *fftypes.Message, allowHashUpdate bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	// Do a select within the transaction to detemine if the UUID already exists
	msgRows, err := s.queryTx(ctx, tx,
		sq.Select("hash").
			From("messages").
			Where(sq.Eq{"id": message.Header.ID}),
	)
	if err != nil {
		return err
	}

	exists := msgRows.Next()
	if exists {
		if !allowHashUpdate {
			var hash *fftypes.Bytes32
			_ = msgRows.Scan(&hash)
			if !fftypes.SafeHashCompare(hash, message.Hash) {
				msgRows.Close()
				log.L(ctx).Errorf("Existing=%s New=%s", hash, message.Hash)
				return database.HashMismatch
			}
		}
		msgRows.Close()

		// Update the message
		if _, err = s.updateTx(ctx, tx,
			sq.Update("messages").
				Set("cid", message.Header.CID).
				Set("mtype", string(message.Header.Type)).
				Set("author", message.Header.Author).
				Set("created", message.Header.Created).
				Set("namespace", message.Header.Namespace).
				Set("topic", message.Header.Topic).
				Set("context", message.Header.Context).
				Set("group_id", message.Header.Group).
				Set("datahash", message.Header.DataHash).
				Set("hash", message.Hash).
				Set("confirmed", message.Confirmed).
				Set("tx_type", message.Header.TX.Type).
				Set("tx_id", message.Header.TX.ID).
				Set("batch_id", message.BatchID).
				Where(sq.Eq{"id": message.Header.ID}),
		); err != nil {
			return err
		}
	} else {
		msgRows.Close()
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("messages").
				Columns(msgColumns...).
				Values(
					message.Header.ID,
					message.Header.CID,
					string(message.Header.Type),
					message.Header.Author,
					message.Header.Created,
					message.Header.Namespace,
					message.Header.Topic,
					message.Header.Context,
					message.Header.Group,
					message.Header.DataHash,
					message.Hash,
					message.Confirmed,
					message.Header.TX.Type,
					message.Header.TX.ID,
					message.BatchID,
				),
		); err != nil {
			return err
		}

		s.postCommitEvent(ctx, tx, func() {
			s.events.MessageCreated(message.Header.ID)
		})

	}

	if err = s.updateMessageDataRefs(ctx, tx, message); err != nil {
		return err
	}

	if err = s.commitTx(ctx, tx, autoCommit); err != nil {
		return err
	}

	return nil
}

func (s *SQLCommon) getMessageDataRefs(ctx context.Context, tx *txWrapper, msgId *uuid.UUID) (fftypes.DataRefs, error) {
	existingRefs, err := s.queryTx(ctx, tx,
		sq.Select(
			"data_id",
			"data_hash",
			"data_idx",
		).
			From("messages_data").
			Where(sq.Eq{"message_id": msgId}).
			OrderBy("data_idx"),
	)
	if err != nil {
		return nil, err
	}
	defer existingRefs.Close()

	var dataIDs fftypes.DataRefs
	for existingRefs.Next() {
		var dataID uuid.UUID
		var dataHash fftypes.Bytes32
		var dataIdx int
		if err = existingRefs.Scan(&dataID, &dataHash, &dataIdx); err != nil {
			return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "messages_data")
		}
		dataIDs = append(dataIDs, fftypes.DataRef{
			ID:   &dataID,
			Hash: &dataHash,
		})
	}
	return dataIDs, nil
}

func (s *SQLCommon) updateMessageDataRefs(ctx context.Context, tx *txWrapper, message *fftypes.Message) error {

	dataIDs, err := s.getMessageDataRefs(ctx, tx, message.Header.ID)
	if err != nil {
		return err
	}

	// Run through the ones in the message, finding ones that already exist, and ones that need to be created
	for msgDataRefIdx, msgDataRef := range message.Data {
		if msgDataRef.ID == nil {
			return i18n.NewError(ctx, i18n.MsgNullDataReferenceID, msgDataRefIdx)
		}
		if msgDataRef.Hash == nil {
			return i18n.NewError(ctx, i18n.MsgMissingDataHashIndex, msgDataRefIdx)
		}
		var found = false
		for dataRefIdx, dataID := range dataIDs {
			if *dataID.ID == *msgDataRef.ID {
				found = true
				// Check the index is correct per the new list
				if msgDataRefIdx != dataRefIdx {
					if _, err = s.updateTx(ctx, tx,
						sq.Update("messages_data").
							Set("data_idx", msgDataRefIdx).
							Where(sq.And{
								sq.Eq{"message_id": message.Header.ID},
								sq.Eq{"data_id": msgDataRef.ID},
							}),
					); err != nil {
						return err
					}
				}
				// Remove it from the list, so we can use this list as ones we need to delete
				copy(dataIDs[dataRefIdx:], dataIDs[dataRefIdx+1:])
				dataIDs = dataIDs[:len(dataIDs)-1]
				break
			}
		}
		if !found {
			// Add the linkage
			if _, err = s.insertTx(ctx, tx,
				sq.Insert("messages_data").
					Columns(
						"message_id",
						"data_id",
						"data_hash",
						"data_idx",
					).
					Values(
						message.Header.ID,
						msgDataRef.ID,
						msgDataRef.Hash,
						msgDataRefIdx,
					),
			); err != nil {
				return err
			}
		}
	}

	// Fun through the extra IDs that are no longer needed
	for _, idToDelete := range dataIDs {
		if _, err = s.deleteTx(ctx, tx,
			sq.Delete("messages_data").
				Where(sq.And{
					sq.Eq{"message_id": message.Header.ID},
					sq.Eq{"data_id": idToDelete.ID},
				}),
		); err != nil {
			return err
		}
	}

	return nil

}

func (s *SQLCommon) loadDataRefs(ctx context.Context, msgs []*fftypes.Message) error {

	msgIds := make([]string, len(msgs))
	for i, m := range msgs {
		if m != nil {
			msgIds[i] = m.Header.ID.String()
		}
	}

	existingRefs, err := s.query(ctx,
		sq.Select(
			"message_id",
			"data_id",
			"data_hash",
			"data_idx",
		).
			From("messages_data").
			Where(sq.Eq{"message_id": msgIds}).
			OrderBy("data_idx"),
	)
	if err != nil {
		return err
	}
	defer existingRefs.Close()

	for existingRefs.Next() {
		var msgID uuid.UUID
		var dataID uuid.UUID
		var dataHash fftypes.Bytes32
		var dataIdx int
		if err = existingRefs.Scan(&msgID, &dataID, &dataHash, &dataIdx); err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "messages_data")
		}
		for _, m := range msgs {
			if *m.Header.ID == msgID {
				m.Data = append(m.Data, fftypes.DataRef{
					ID:   &dataID,
					Hash: &dataHash,
				})
			}
		}
	}
	// Ensure we return an empty array if no entries, and a consistent order for the data
	for _, m := range msgs {
		if m.Data == nil {
			m.Data = fftypes.DataRefs{}
		}
	}

	return nil
}

func (s *SQLCommon) msgResult(ctx context.Context, row *sql.Rows) (*fftypes.Message, error) {
	var msg fftypes.Message
	err := row.Scan(
		&msg.Header.ID,
		&msg.Header.CID,
		&msg.Header.Type,
		&msg.Header.Author,
		&msg.Header.Created,
		&msg.Header.Namespace,
		&msg.Header.Topic,
		&msg.Header.Context,
		&msg.Header.Group,
		&msg.Header.DataHash,
		&msg.Hash,
		&msg.Confirmed,
		&msg.Header.TX.Type,
		&msg.Header.TX.ID,
		&msg.BatchID,
		// Must be added to the list of columns in all selects
		&msg.Sequence,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "messages")
	}
	return &msg, nil
}

func (s *SQLCommon) GetMessageById(ctx context.Context, id *uuid.UUID) (message *fftypes.Message, err error) {

	cols := append([]string{}, msgColumns...)
	cols = append(cols, s.options.SequenceField)
	rows, err := s.query(ctx,
		sq.Select(cols...).
			From("messages").
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Message '%s' not found", id)
		return nil, nil
	}

	msg, err := s.msgResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	if err = s.loadDataRefs(ctx, []*fftypes.Message{msg}); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *SQLCommon) GetMessages(ctx context.Context, filter database.Filter) (message []*fftypes.Message, err error) {

	cols := append([]string{}, msgColumns...)
	cols = append(cols, s.options.SequenceField)
	query, err := s.filterSelect(ctx, sq.Select(cols...).From("messages"), filter, msgFilterTypeMap)
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	msgs := []*fftypes.Message{}
	for rows.Next() {
		msg, err := s.msgResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}

	if len(msgs) > 0 {
		if err = s.loadDataRefs(ctx, msgs); err != nil {
			return nil, err
		}
	}

	return msgs, err

}

func (s *SQLCommon) UpdateMessage(ctx context.Context, msgid *uuid.UUID, update database.Update) (err error) {
	return s.UpdateMessages(ctx, database.MessageQueryFactory.NewFilter(ctx, 0).Eq("id", msgid), update)
}

func (s *SQLCommon) UpdateMessages(ctx context.Context, filter database.Filter, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(ctx, sq.Update("messages"), update, msgFilterTypeMap)
	if err != nil {
		return err
	}

	query, err = s.filterUpdate(ctx, query, filter, opFilterTypeMap)
	if err != nil {
		return err
	}

	_, err = s.updateTx(ctx, tx, query)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
