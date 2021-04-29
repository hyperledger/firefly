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
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/persistence"
)

type SQLCommon struct {
	db      *sql.DB
	options *SQLCommonOptions
}

type SQLCommonOptions struct {
	PlaceholderFormat sq.PlaceholderFormat
}

func InitSQLCommon(ctx context.Context, s *SQLCommon, db *sql.DB, options *SQLCommonOptions) (*persistence.Capabilities, error) {
	s.db = db
	if options == nil {
		options = &SQLCommonOptions{
			PlaceholderFormat: sq.Dollar,
		}
	}
	s.options = options
	return &persistence.Capabilities{}, nil
}

func (s *SQLCommon) beginTx(ctx context.Context) (context.Context, *sql.Tx, error) {
	l := log.L(ctx).WithField("dbtx", fftypes.ShortID())
	ctx = log.WithLogger(ctx, l)
	l.Debugf("SQL-> begin")
	tx, err := s.db.Begin()
	if err != nil {
		return ctx, nil, i18n.WrapError(ctx, err, i18n.MsgDBBeginFailed)
	}
	l.Debugf("SQL<- begin ok=%t", err == nil)
	return ctx, tx, err
}

func (s *SQLCommon) queryTx(ctx context.Context, tx *sql.Tx, q sq.SelectBuilder) (*sql.Rows, error) {
	l := log.L(ctx)
	sqlQuery, args, err := q.PlaceholderFormat(s.options.PlaceholderFormat).ToSql()
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> query sql=[ %s ] args=%+v`, sqlQuery, args)
	row, err := tx.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		l.Errorf(`SQL query failed: %s sql=[ %s ]`, err, sqlQuery)
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBQueryFailed)
	}
	l.Debugf(`SQL<- query ok=%t`, err == nil)
	return row, nil
}

func (s *SQLCommon) insertTx(ctx context.Context, tx *sql.Tx, q sq.InsertBuilder) (sql.Result, error) {
	l := log.L(ctx)
	sqlQuery, args, err := q.PlaceholderFormat(s.options.PlaceholderFormat).ToSql()
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> insert sql=[ %s ] args=%+v`, sqlQuery, args)
	res, err := tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		l.Errorf(`SQL insert failed: %s sql=[ %s ]: %s`, err, sqlQuery, err)
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBUpdateFailed)
	}
	ra, _ := res.RowsAffected() // currently only used for debugging
	l.Debugf(`SQL<- insert ok=%t affected=%d`, err == nil, ra)
	return res, nil
}

func (s *SQLCommon) updateTx(ctx context.Context, tx *sql.Tx, q sq.UpdateBuilder) (sql.Result, error) {
	l := log.L(ctx)
	sqlQuery, args, err := q.PlaceholderFormat(s.options.PlaceholderFormat).ToSql()
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> update sql=[ %s ] args=%+v`, sqlQuery, args)
	res, err := tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		l.Errorf(`SQL update failed: %s sql=[ %s ]`, err, sqlQuery)
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBUpdateFailed)
	}
	ra, _ := res.RowsAffected() // currently only used for debugging
	l.Debugf(`SQL<- update ok=%t affected=%d`, err == nil, ra)
	return res, nil
}

func (s *SQLCommon) rollbackTx(ctx context.Context, tx *sql.Tx) {
	err := tx.Rollback()
	if err == nil || err != sql.ErrTxDone {
		log.L(ctx).Warnf("SQL! transaction rollback")
	}
}

func sqlUUID(u *uuid.UUID) interface{} {
	if u == nil {
		return nil
	}
	return u.String()
}

func sqlB32(b *fftypes.Bytes32) interface{} {
	if b == nil {
		return nil
	}
	return b.String()
}

func sqlInt64(t int64) interface{} {
	if t == 0 {
		return nil
	}
	return t
}

func sqlStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (s *SQLCommon) UpsertMessage(ctx context.Context, message *fftypes.MessageBase) (err error) {
	ctx, tx, err := s.beginTx(ctx)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgDBBeginFailed)
	}
	defer s.rollbackTx(ctx, tx)

	// Do a select within the transaction to detemine if the UUID already exists
	msgRows, err := s.queryTx(ctx, tx,
		sq.Select("id").
			From("messages").
			Where(sq.Eq{"id": sqlUUID(message.ID)}),
	)
	if err != nil {
		return err
	}
	defer msgRows.Close()

	var joinId string
	if msgRows.Next() {
		// Update the message
		if _, err = s.updateTx(ctx, tx,
			sq.Update("messages").
				Set("cid", sqlUUID(message.CID)).
				Set("type", sqlStr(string(message.Type))).
				Set("author", sqlStr(message.Author)).
				Set("created", sqlInt64(message.Created)).
				Set("namespace", sqlStr(message.Namespace)).
				Set("topic", sqlStr(message.Topic)).
				Set("context", sqlStr(message.Context)).
				Set("group_id", sqlUUID(message.Group)).
				Set("datahash", sqlB32(message.DataHash)).
				Set("hash", sqlB32(message.Hash)).
				Set("confirmed", sqlInt64(message.Confirmed)).
				Where(sq.Eq{"id": sqlUUID(message.ID)}),
		); err != nil {
			return err
		}

		// There should already be a join entry in the message_data join table
		joinRows, err := s.queryTx(ctx, tx,
			sq.Select("data_id").
				From("messages_data").
				Where(sq.Eq{"message_id": sqlUUID(message.ID)}),
		)
		if err != nil {
			return err
		}
		if !joinRows.Next() {
			return i18n.NewError(ctx, i18n.MsgDBMissingJoin, "messages_data", message.ID)
		}
		if err = joinRows.Scan(&joinId); err != nil {
			return i18n.NewError(ctx, i18n.MsgDBReadErr, "messages_data")
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("messages").
				Columns(
					"id",
					"cid",
					"type",
					"author",
					"created",
					"namespace",
					"context",
					"topic",
					"group_id",
					"datahash",
					"hash",
					"confirmed",
				).
				Values(
					sqlUUID(message.ID),
					sqlUUID(message.CID),
					sqlStr(string(message.Type)),
					sqlStr(message.Author),
					sqlInt64(message.Created),
					sqlStr(message.Namespace),
					sqlStr(message.Topic),
					sqlStr(message.Context),
					sqlUUID(message.Group),
					sqlB32(message.DataHash),
					sqlB32(message.Hash),
					sqlInt64(message.Confirmed)),
		); err != nil {
			return err
		}

	}

	return nil
}

func (s *SQLCommon) GetMessageById(ctx context.Context, id uuid.UUID) (message *fftypes.MessageBase, err error) {
	return nil, err
}

func (s *SQLCommon) GetMessages(ctx context.Context, filter *persistence.MessageFilter, skip, limit uint) (message *fftypes.MessageBase, err error) {
	return nil, err
}
