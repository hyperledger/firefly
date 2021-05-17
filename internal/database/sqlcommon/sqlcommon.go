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
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
)

type SQLCommon struct {
	db           *sql.DB
	capabilities *database.Capabilities
	events       database.Events
	options      *SQLCommonOptions
}

type txContextKey struct{}

type SQLCommonOptions struct {
	// PlaceholderFormat as supported by the SQL sequence of the plugin
	PlaceholderFormat sq.PlaceholderFormat
	// SequenceField must be auto added by the database to each table, via appropriate DDL in the migrations
	SequenceField string
}

func InitSQLCommon(ctx context.Context, s *SQLCommon, db *sql.DB, events database.Events, capabilities *database.Capabilities, options *SQLCommonOptions) error {
	s.db = db
	s.capabilities = capabilities
	s.events = events
	if options == nil || options.PlaceholderFormat == nil || options.SequenceField == "" {
		log.L(ctx).Errorf("Invalid SQL options from plugin: %+v", options)
		return i18n.NewError(ctx, i18n.MsgDBInitFailed)
	}
	s.options = options
	return nil
}

func (s *SQLCommon) Capabilities() *database.Capabilities { return s.capabilities }

func (s *SQLCommon) RunAsGroup(ctx context.Context, fn func(ctx context.Context) error) error {
	ctx, tx, _, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, false /* we _are_ the auto-committer */)

	if err = fn(ctx); err != nil {
		return err
	}

	return s.commitTx(ctx, tx, false /* we _are_ the auto-committer */)
}

func getTXFromContext(ctx context.Context) *sql.Tx {
	ctxKey := txContextKey{}
	txi := ctx.Value(ctxKey)
	if txi != nil {
		if tx, ok := txi.(*sql.Tx); ok {
			return tx
		}
	}
	return nil
}

func (s *SQLCommon) beginOrUseTx(ctx context.Context) (ctx1 context.Context, tx *sql.Tx, autoCommit bool, err error) {

	tx = getTXFromContext(ctx)
	if tx != nil {
		// There is s transaction on the context already.
		// return existing with auto-commit flag, to prevent early commit
		return ctx, tx, true, nil
	}

	l := log.L(ctx).WithField("dbtx", fftypes.ShortID())
	ctx1 = log.WithLogger(ctx, l)
	l.Debugf("SQL-> begin")
	tx, err = s.db.Begin()
	if err != nil {
		return ctx1, nil, false, i18n.WrapError(ctx1, err, i18n.MsgDBBeginFailed)
	}
	ctx1 = context.WithValue(ctx, txContextKey{}, tx)
	l.Debugf("SQL<- begin")
	return ctx1, tx, false, err
}

func (s *SQLCommon) queryTx(ctx context.Context, tx *sql.Tx, q sq.SelectBuilder) (*sql.Rows, error) {
	if tx == nil {
		// If there is a transaction in the context, we should use it to provide consistency
		// in the read operations (read after insert for example).
		tx = getTXFromContext(ctx)
	}

	l := log.L(ctx)
	sqlQuery, args, err := q.PlaceholderFormat(s.options.PlaceholderFormat).ToSql()
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> query: %s`, sqlQuery)
	l.Tracef(`SQL-> query args: %+v`, args)
	var rows *sql.Rows
	if tx != nil {
		rows, err = tx.QueryContext(ctx, sqlQuery, args...)
	} else {
		rows, err = s.db.QueryContext(ctx, sqlQuery, args...)
	}
	if err != nil {
		l.Errorf(`SQL query failed: %s sql=[ %s ]`, err, sqlQuery)
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBQueryFailed)
	}
	l.Debugf(`SQL<- query`)
	return rows, nil
}

func (s *SQLCommon) query(ctx context.Context, q sq.SelectBuilder) (*sql.Rows, error) {
	return s.queryTx(ctx, nil, q)
}

func (s *SQLCommon) insertTx(ctx context.Context, tx *sql.Tx, q sq.InsertBuilder) (sql.Result, error) {
	l := log.L(ctx)
	sqlQuery, args, err := q.PlaceholderFormat(s.options.PlaceholderFormat).ToSql()
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> insert: %s`, sqlQuery)
	l.Tracef(`SQL-> insert args: %+v`, args)
	res, err := tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		l.Errorf(`SQL insert failed: %s sql=[ %s ]: %s`, err, sqlQuery, err)
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBInsertFailed)
	}
	ra, _ := res.RowsAffected() // currently only used for debugging
	l.Debugf(`SQL<- insert affected=%d`, ra)
	return res, nil
}

func (s *SQLCommon) deleteTx(ctx context.Context, tx *sql.Tx, q sq.DeleteBuilder) (sql.Result, error) {
	l := log.L(ctx)
	sqlQuery, args, err := q.PlaceholderFormat(s.options.PlaceholderFormat).ToSql()
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> delete: %s`, sqlQuery)
	l.Tracef(`SQL-> delete args: %+v`, args)
	res, err := tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		l.Errorf(`SQL delete failed: %s sql=[ %s ]: %s`, err, sqlQuery, err)
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBDeleteFailed)
	}
	ra, _ := res.RowsAffected() // currently only used for debugging
	l.Debugf(`SQL<- delete affected=%d`, ra)
	return res, nil
}

func (s *SQLCommon) updateTx(ctx context.Context, tx *sql.Tx, q sq.UpdateBuilder) (sql.Result, error) {
	l := log.L(ctx)
	sqlQuery, args, err := q.PlaceholderFormat(s.options.PlaceholderFormat).ToSql()
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> update: %s`, sqlQuery)
	l.Tracef(`SQL-> update args: %+v`, args)
	res, err := tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		l.Errorf(`SQL update failed: %s sql=[ %s ]`, err, sqlQuery)
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBUpdateFailed)
	}
	ra, _ := res.RowsAffected() // currently only used for debugging
	l.Debugf(`SQL<- update affected=%d`, ra)
	return res, nil
}

// rollbackTx be safely called as a defer, as it is a cheap no-op if the transaction is complete
func (s *SQLCommon) rollbackTx(ctx context.Context, tx *sql.Tx, autoCommit bool) {
	if autoCommit {
		// We're inside of a wide transaction boundary with an auto-commit
		return
	}

	err := tx.Rollback()
	if err == nil {
		log.L(ctx).Warnf("SQL! transaction rollback")
	}
	if err != nil && err != sql.ErrTxDone {
		log.L(ctx).Errorf(`SQL rollback failed: %s`, err)
	}
}

func (s *SQLCommon) commitTx(ctx context.Context, tx *sql.Tx, autoCommit bool) error {
	if autoCommit {
		// We're inside of a wide transaction boundary with an auto-commit
		return nil
	}

	l := log.L(ctx)
	l.Debugf(`SQL-> commit`)
	err := tx.Commit()
	if err != nil {
		l.Errorf(`SQL commit failed: %s`, err)
		return i18n.WrapError(ctx, err, i18n.MsgDBCommitFailed)
	}
	l.Debugf(`SQL<- commit`)
	return nil
}
