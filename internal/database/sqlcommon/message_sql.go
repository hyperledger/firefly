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
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

var (
	msgColumns = []string{
		"id",
		"cid",
		"mtype",
		"author",
		"created",
		"namespace",
		"topics",
		"tag",
		"group_hash",
		"datahash",
		"hash",
		"pins",
		"rejected",
		"pending",
		"confirmed",
		"tx_type",
		"batch_id",
		"local",
	}
	msgFilterFieldMap = map[string]string{
		"type":   "mtype",
		"txtype": "tx_type",
		"batch":  "batch_id",
		"group":  "group_hash",
	}
)

func (s *SQLCommon) InsertMessageLocal(ctx context.Context, message *fftypes.Message) (err error) {
	message.Local = true
	return s.upsertMessageCommon(ctx, message, false, false, true /* local insert */)
}

func (s *SQLCommon) UpsertMessage(ctx context.Context, message *fftypes.Message, allowExisting, allowHashUpdate bool) (err error) {
	return s.upsertMessageCommon(ctx, message, allowExisting, allowHashUpdate, false /* not local */)
}

func (s *SQLCommon) upsertMessageCommon(ctx context.Context, message *fftypes.Message, allowExisting, allowHashUpdate, isLocal bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		// Do a select within the transaction to detemine if the UUID already exists
		msgRows, err := s.queryTx(ctx, tx,
			sq.Select("hash", sequenceColumn).
				From("messages").
				Where(sq.Eq{"id": message.Header.ID}),
		)
		if err != nil {
			return err
		}

		existing = msgRows.Next()
		if existing && !allowHashUpdate {
			var hash *fftypes.Bytes32
			_ = msgRows.Scan(&hash, &message.Sequence)
			if !fftypes.SafeHashCompare(hash, message.Hash) {
				msgRows.Close()
				log.L(ctx).Errorf("Existing=%s New=%s", hash, message.Hash)
				return database.HashMismatch
			}
		}
		msgRows.Close()
	}

	if existing {

		// Update the message
		if err = s.updateTx(ctx, tx,
			sq.Update("messages").
				Set("cid", message.Header.CID).
				Set("mtype", string(message.Header.Type)).
				Set("author", message.Header.Author).
				Set("created", message.Header.Created).
				Set("namespace", message.Header.Namespace).
				Set("topics", message.Header.Topics).
				Set("tag", message.Header.Tag).
				Set("group_hash", message.Header.Group).
				Set("datahash", message.Header.DataHash).
				Set("hash", message.Hash).
				Set("pins", message.Pins).
				Set("rejected", message.Rejected).
				Set("pending", message.Pending).
				Set("confirmed", message.Confirmed).
				Set("tx_type", message.Header.TxType).
				Set("batch_id", message.BatchID).
				// Intentionally does NOT include the "local" column
				Where(sq.Eq{"id": message.Header.ID}),
			func() {
				s.callbacks.OrderedUUIDCollectionNSEvent(database.CollectionMessages, fftypes.ChangeEventTypeUpdated, message.Header.Namespace, message.Header.ID, message.Sequence)
			},
		); err != nil {
			return err
		}
	} else {
		message.Sequence, err = s.insertTx(ctx, tx,
			sq.Insert("messages").
				Columns(msgColumns...).
				Values(
					message.Header.ID,
					message.Header.CID,
					string(message.Header.Type),
					message.Header.Author,
					message.Header.Created,
					message.Header.Namespace,
					message.Header.Topics,
					message.Header.Tag,
					message.Header.Group,
					message.Header.DataHash,
					message.Hash,
					message.Pins,
					message.Rejected,
					message.Pending,
					message.Confirmed,
					message.Header.TxType,
					message.BatchID,
					isLocal,
				),
			func() {
				s.callbacks.OrderedUUIDCollectionNSEvent(database.CollectionMessages, fftypes.ChangeEventTypeCreated, message.Header.Namespace, message.Header.ID, message.Sequence)
			},
		)
		if err != nil {
			return err
		}
	}

	if err = s.updateMessageDataRefs(ctx, tx, message, existing); err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) updateMessageDataRefs(ctx context.Context, tx *txWrapper, message *fftypes.Message, existing bool) error {

	if existing {
		if err := s.deleteTx(ctx, tx,
			sq.Delete("messages_data").
				Where(sq.And{
					sq.Eq{"message_id": message.Header.ID},
				}),
			nil, // no change event
		); err != nil {
			return err
		}
	}

	// Run through the ones in the message, finding ones that already exist, and ones that need to be created
	for msgDataRefIDx, msgDataRef := range message.Data {
		if msgDataRef.ID == nil {
			return i18n.NewError(ctx, i18n.MsgNullDataReferenceID, msgDataRefIDx)
		}
		if msgDataRef.Hash == nil {
			return i18n.NewError(ctx, i18n.MsgMissingDataHashIndex, msgDataRefIDx)
		}
		// Add the linkage
		if _, err := s.insertTx(ctx, tx,
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
					msgDataRefIDx,
				),
			nil, // no change event
		); err != nil {
			return err
		}
	}

	return nil

}

// Why not a LEFT JOIN you ask? ... well we need to be able to reliably perform a LIMIT on
// the number of messages, and it seems there isn't a clean and cross-database
// way for a single-query option. So a two-query option ended up being simplest.
// See commit e304161a30b8044a42b5bac3fcfca7e7bd8f8ab7 for the abandoned changeset
// that implemented LEFT JOIN
func (s *SQLCommon) loadDataRefs(ctx context.Context, msgs []*fftypes.Message) error {

	msgIDs := make([]string, len(msgs))
	for i, m := range msgs {
		if m != nil {
			msgIDs[i] = m.Header.ID.String()
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
			Where(sq.Eq{"message_id": msgIDs}).
			OrderBy("data_idx"),
	)
	if err != nil {
		return err
	}
	defer existingRefs.Close()

	for existingRefs.Next() {
		var msgID fftypes.UUID
		var dataID fftypes.UUID
		var dataHash fftypes.Bytes32
		var dataIDx int
		if err = existingRefs.Scan(&msgID, &dataID, &dataHash, &dataIDx); err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "messages_data")
		}
		for _, m := range msgs {
			if *m.Header.ID == msgID {
				m.Data = append(m.Data, &fftypes.DataRef{
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
		&msg.Header.Topics,
		&msg.Header.Tag,
		&msg.Header.Group,
		&msg.Header.DataHash,
		&msg.Hash,
		&msg.Pins,
		&msg.Rejected,
		&msg.Pending,
		&msg.Confirmed,
		&msg.Header.TxType,
		&msg.BatchID,
		&msg.Local,
		// Must be added to the list of columns in all selects
		&msg.Sequence,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "messages")
	}
	return &msg, nil
}

func (s *SQLCommon) GetMessageByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.Message, err error) {

	cols := append([]string{}, msgColumns...)
	cols = append(cols, sequenceColumn)
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

	rows.Close()
	if err = s.loadDataRefs(ctx, []*fftypes.Message{msg}); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *SQLCommon) getMessagesQuery(ctx context.Context, query sq.SelectBuilder) (message []*fftypes.Message, err error) {
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

	rows.Close()
	if len(msgs) > 0 {
		if err = s.loadDataRefs(ctx, msgs); err != nil {
			return nil, err
		}
	}

	return msgs, err
}

func (s *SQLCommon) GetMessages(ctx context.Context, filter database.Filter) (message []*fftypes.Message, err error) {
	cols := append([]string{}, msgColumns...)
	cols = append(cols, sequenceColumn)
	query, err := s.filterSelect(ctx, "", sq.Select(cols...).From("messages"), filter, msgFilterFieldMap,
		[]string{"pending", "confirmed", "created"}) // put unconfirmed messages first, then order by confirmed
	if err != nil {
		return nil, err
	}
	return s.getMessagesQuery(ctx, query)
}

func (s *SQLCommon) GetMessagesForData(ctx context.Context, dataID *fftypes.UUID, filter database.Filter) (message []*fftypes.Message, err error) {
	cols := make([]string, len(msgColumns)+1)
	for i, col := range msgColumns {
		cols[i] = fmt.Sprintf("m.%s", col)
	}
	cols[len(msgColumns)] = "m.seq"
	query, err := s.filterSelect(ctx, "m", sq.Select(cols...).From("messages_data AS md"), filter, msgFilterFieldMap, []string{"sequence"},
		sq.Eq{"md.data_id": dataID})
	if err != nil {
		return nil, err
	}

	query = query.LeftJoin("messages AS m ON m.id = md.message_id")
	return s.getMessagesQuery(ctx, query)
}

func (s *SQLCommon) GetMessageRefs(ctx context.Context, filter database.Filter) ([]*fftypes.MessageRef, error) {
	query, err := s.filterSelect(ctx, "", sq.Select("id", sequenceColumn, "hash").From("messages"), filter, msgFilterFieldMap, []string{"sequence"})
	if err != nil {
		return nil, err
	}
	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	msgRefs := []*fftypes.MessageRef{}
	for rows.Next() {
		var msgRef fftypes.MessageRef
		if err = rows.Scan(&msgRef.ID, &msgRef.Sequence, &msgRef.Hash); err != nil {
			return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "messages")
		}
		msgRefs = append(msgRefs, &msgRef)
	}
	return msgRefs, nil
}

func (s *SQLCommon) UpdateMessage(ctx context.Context, msgid *fftypes.UUID, update database.Update) (err error) {
	return s.UpdateMessages(ctx, database.MessageQueryFactory.NewFilter(ctx).Eq("id", msgid), update)
}

func (s *SQLCommon) UpdateMessages(ctx context.Context, filter database.Filter, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update("messages"), update, msgFilterFieldMap)
	if err != nil {
		return err
	}

	query, err = s.filterUpdate(ctx, "", query, filter, opFilterFieldMap)
	if err != nil {
		return err
	}

	err = s.updateTx(ctx, tx, query, nil /* no change events filter based update */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
