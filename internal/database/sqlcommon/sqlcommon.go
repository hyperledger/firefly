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
	"github.com/golang-migrate/migrate/v4"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"

	// Import migrate file source
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type SQLCommon struct {
	db           *sql.DB
	capabilities *database.Capabilities
	callbacks    database.Callbacks
	provider     Provider
}

type txContextKey struct{}

type txWrapper struct {
	sqlTX      *sql.Tx
	postCommit []func()
}

func (s *SQLCommon) Init(ctx context.Context, provider Provider, prefix config.Prefix, callbacks database.Callbacks, capabilities *database.Capabilities) (err error) {
	s.capabilities = capabilities
	s.callbacks = callbacks
	s.provider = provider
	if s.provider == nil || s.provider.PlaceholderFormat() == nil || sequenceColumn == "" {
		log.L(ctx).Errorf("Invalid SQL options from provider '%T'", s.provider)
		return i18n.NewError(ctx, i18n.MsgDBInitFailed)
	}

	if s.db, err = provider.Open(prefix.GetString(SQLConfDatasourceURL)); err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgDBInitFailed)
	}
	connLimit := prefix.GetInt(SQLConfMaxConnections)
	if connLimit > 0 {
		s.db.SetMaxOpenConns(connLimit)
	}

	if prefix.GetBool(SQLConfMigrationsAuto) {
		if err = s.applyDBMigrations(ctx, prefix, provider); err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgDBMigrationFailed)
		}
	}

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

func (s *SQLCommon) applyDBMigrations(ctx context.Context, prefix config.Prefix, provider Provider) error {
	driver, err := provider.GetMigrationDriver(s.db)
	if err == nil {
		var m *migrate.Migrate
		m, err = migrate.NewWithDatabaseInstance(
			"file://"+prefix.GetString(SQLConfMigrationsDirectory),
			provider.MigrationsDir(), driver)
		if err == nil {
			err = m.Up()
		}
	}
	if err != nil && err != migrate.ErrNoChange {
		return i18n.WrapError(ctx, err, i18n.MsgDBMigrationFailed)
	}
	return nil
}

func getTXFromContext(ctx context.Context) *txWrapper {
	ctxKey := txContextKey{}
	txi := ctx.Value(ctxKey)
	if txi != nil {
		if tx, ok := txi.(*txWrapper); ok {
			return tx
		}
	}
	return nil
}

func (s *SQLCommon) beginOrUseTx(ctx context.Context) (ctx1 context.Context, tx *txWrapper, autoCommit bool, err error) {

	tx = getTXFromContext(ctx)
	if tx != nil {
		// There is s transaction on the context already.
		// return existing with auto-commit flag, to prevent early commit
		return ctx, tx, true, nil
	}

	l := log.L(ctx).WithField("dbtx", fftypes.ShortID())
	ctx1 = log.WithLogger(ctx, l)
	l.Debugf("SQL-> begin")
	sqlTX, err := s.db.Begin()
	if err != nil {
		return ctx1, nil, false, i18n.WrapError(ctx1, err, i18n.MsgDBBeginFailed)
	}
	tx = &txWrapper{
		sqlTX: sqlTX,
	}
	ctx1 = context.WithValue(ctx1, txContextKey{}, tx)
	l.Debugf("SQL<- begin")
	return ctx1, tx, false, err
}

func (s *SQLCommon) queryTx(ctx context.Context, tx *txWrapper, q sq.SelectBuilder) (*sql.Rows, error) {
	if tx == nil {
		// If there is a transaction in the context, we should use it to provide consistency
		// in the read operations (read after insert for example).
		tx = getTXFromContext(ctx)
	}

	l := log.L(ctx)
	sqlQuery, args, err := q.PlaceholderFormat(s.provider.PlaceholderFormat()).ToSql()
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> query: %s`, sqlQuery)
	l.Tracef(`SQL-> query args: %+v`, args)
	var rows *sql.Rows
	if tx != nil {
		rows, err = tx.sqlTX.QueryContext(ctx, sqlQuery, args...)
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

func (s *SQLCommon) insertTx(ctx context.Context, tx *txWrapper, q sq.InsertBuilder, postCommit func()) (int64, error) {
	l := log.L(ctx)
	q, useQuery := s.provider.UpdateInsertForSequenceReturn(q)

	sqlQuery, args, err := q.PlaceholderFormat(s.provider.PlaceholderFormat()).ToSql()
	if err != nil {
		return -1, i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> insert: %s`, sqlQuery)
	l.Tracef(`SQL-> insert args: %+v`, args)
	var sequence int64
	if useQuery {
		err := tx.sqlTX.QueryRowContext(ctx, sqlQuery, args...).Scan(&sequence)
		if err != nil {
			l.Errorf(`SQL insert failed: %s sql=[ %s ]: %s`, err, sqlQuery, err)
			return -1, i18n.WrapError(ctx, err, i18n.MsgDBInsertFailed)
		}
	} else {
		res, err := tx.sqlTX.ExecContext(ctx, sqlQuery, args...)
		if err != nil {
			l.Errorf(`SQL insert failed: %s sql=[ %s ]: %s`, err, sqlQuery, err)
			return -1, i18n.WrapError(ctx, err, i18n.MsgDBInsertFailed)
		}
		sequence, _ = res.LastInsertId()
	}
	l.Debugf(`SQL<- inserted sequence=%d`, sequence)

	if postCommit != nil {
		s.postCommitEvent(tx, postCommit)
	}
	return sequence, nil
}

func (s *SQLCommon) deleteTx(ctx context.Context, tx *txWrapper, q sq.DeleteBuilder, postCommit func()) error {
	l := log.L(ctx)
	sqlQuery, args, err := q.PlaceholderFormat(s.provider.PlaceholderFormat()).ToSql()
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> delete: %s`, sqlQuery)
	l.Tracef(`SQL-> delete args: %+v`, args)
	res, err := tx.sqlTX.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		l.Errorf(`SQL delete failed: %s sql=[ %s ]: %s`, err, sqlQuery, err)
		return i18n.WrapError(ctx, err, i18n.MsgDBDeleteFailed)
	}
	ra, _ := res.RowsAffected()
	l.Debugf(`SQL<- delete affected=%d`, ra)
	if ra < 1 {
		return database.DeleteRecordNotFound
	}

	if postCommit != nil {
		s.postCommitEvent(tx, postCommit)
	}
	return nil
}

func (s *SQLCommon) updateTx(ctx context.Context, tx *txWrapper, q sq.UpdateBuilder, postCommit func()) error {
	l := log.L(ctx)
	sqlQuery, args, err := q.PlaceholderFormat(s.provider.PlaceholderFormat()).ToSql()
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> update: %s`, sqlQuery)
	l.Tracef(`SQL-> update args: %+v`, args)
	res, err := tx.sqlTX.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		l.Errorf(`SQL update failed: %s sql=[ %s ]`, err, sqlQuery)
		return i18n.WrapError(ctx, err, i18n.MsgDBUpdateFailed)
	}
	ra, _ := res.RowsAffected() // currently only used for debugging
	l.Debugf(`SQL<- update affected=%d`, ra)

	if postCommit != nil {
		s.postCommitEvent(tx, postCommit)
	}
	return nil
}

func (s *SQLCommon) postCommitEvent(tx *txWrapper, fn func()) {
	tx.postCommit = append(tx.postCommit, fn)
}

// rollbackTx be safely called as a defer, as it is a cheap no-op if the transaction is complete
func (s *SQLCommon) rollbackTx(ctx context.Context, tx *txWrapper, autoCommit bool) {
	if autoCommit {
		// We're inside of a wide transaction boundary with an auto-commit
		return
	}

	err := tx.sqlTX.Rollback()
	if err == nil {
		log.L(ctx).Warnf("SQL! transaction rollback")
	}
	if err != nil && err != sql.ErrTxDone {
		log.L(ctx).Errorf(`SQL rollback failed: %s`, err)
	}
}

func (s *SQLCommon) commitTx(ctx context.Context, tx *txWrapper, autoCommit bool) error {
	if autoCommit {
		// We're inside of a wide transaction boundary with an auto-commit
		return nil
	}

	l := log.L(ctx)
	l.Debugf(`SQL-> commit`)
	err := tx.sqlTX.Commit()
	if err != nil {
		l.Errorf(`SQL commit failed: %s`, err)
		return i18n.WrapError(ctx, err, i18n.MsgDBCommitFailed)
	}
	l.Debugf(`SQL<- commit`)

	// Emit any post commit events (these aren't currently allowed to cause errors)
	for i, pce := range tx.postCommit {
		l.Tracef(`-> post commit event %d`, i)
		pce()
		l.Tracef(`<- post commit event %d`, i)
	}

	return nil
}

func (s *SQLCommon) DB() *sql.DB {
	return s.db
}

func (s *SQLCommon) Close() {
	if s.db != nil {
		err := s.db.Close()
		log.L(context.Background()).Debugf("Database closed (err=%v)", err)
	}
}
