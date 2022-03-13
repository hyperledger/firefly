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
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang-migrate/migrate/v4"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/sirupsen/logrus"

	// Import migrate file source
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type SQLCommon struct {
	db           *sql.DB
	capabilities *database.Capabilities
	callbacks    database.Callbacks
	provider     Provider
	features     SQLFeatures
}

type txContextKey struct{}

type txWrapper struct {
	sqlTX           *sql.Tx
	preCommitEvents []*fftypes.Event
	postCommit      []func()
	tableLocks      []string
}

func shortenSQL(sqlString string) string {
	buff := strings.Builder{}
	spaceCount := 0
	for _, c := range sqlString {
		if c == ' ' {
			spaceCount++
			if spaceCount >= 3 {
				break
			}
		}
		buff.WriteRune(c)
	}
	return buff.String()
}

func (s *SQLCommon) Init(ctx context.Context, provider Provider, prefix config.Prefix, callbacks database.Callbacks, capabilities *database.Capabilities) (err error) {
	s.capabilities = capabilities
	s.callbacks = callbacks
	s.provider = provider
	if s.provider != nil {
		s.features = s.provider.Features()
	}
	if s.provider == nil || s.features.PlaceholderFormat == nil {
		log.L(ctx).Errorf("Invalid SQL options from provider '%T'", s.provider)
		return i18n.NewError(ctx, i18n.MsgDBInitFailed)
	}

	if s.db, err = provider.Open(prefix.GetString(SQLConfDatasourceURL)); err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgDBInitFailed)
	}
	connLimit := prefix.GetInt(SQLConfMaxConnections)
	if connLimit > 0 {
		s.db.SetMaxOpenConns(connLimit)
		s.db.SetConnMaxIdleTime(prefix.GetDuration(SQLConfMaxConnIdleTime))
		maxIdleConns := prefix.GetInt(SQLConfMaxIdleConns)
		if maxIdleConns <= 0 {
			// By default we rely on the idle time, rather than a maximum number of conns to leave open
			maxIdleConns = connLimit
		}
		s.db.SetMaxIdleConns(maxIdleConns)
		s.db.SetConnMaxLifetime(prefix.GetDuration(SQLConfMaxConnLifetime))
	}
	if connLimit > 1 {
		capabilities.Concurrency = true
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
	if tx := getTXFromContext(ctx); tx != nil {
		// transaction already exists - just continue using it
		return fn(ctx)
	}

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

func (s *SQLCommon) queryTx(ctx context.Context, tx *txWrapper, q sq.SelectBuilder) (*sql.Rows, *txWrapper, error) {
	if tx == nil {
		// If there is a transaction in the context, we should use it to provide consistency
		// in the read operations (read after insert for example).
		tx = getTXFromContext(ctx)
	}

	l := log.L(ctx)
	sqlQuery, args, err := q.PlaceholderFormat(s.features.PlaceholderFormat).ToSql()
	if err != nil {
		return nil, tx, i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> query: %s`, shortenSQL(sqlQuery))
	l.Tracef(`SQL-> query args: %+v`, args)
	var rows *sql.Rows
	if tx != nil {
		rows, err = tx.sqlTX.QueryContext(ctx, sqlQuery, args...)
	} else {
		rows, err = s.db.QueryContext(ctx, sqlQuery, args...)
	}
	if err != nil {
		l.Errorf(`SQL query failed: %s sql=[ %s ]`, err, sqlQuery)
		return nil, tx, i18n.WrapError(ctx, err, i18n.MsgDBQueryFailed)
	}
	l.Debugf(`SQL<- query`)
	return rows, tx, nil
}

func (s *SQLCommon) query(ctx context.Context, q sq.SelectBuilder) (*sql.Rows, *txWrapper, error) {
	return s.queryTx(ctx, nil, q)
}

func (s *SQLCommon) countQuery(ctx context.Context, tx *txWrapper, tableName string, fop sq.Sqlizer, countExpr string) (count int64, err error) {
	count = -1
	l := log.L(ctx)
	if tx == nil {
		// If there is a transaction in the context, we should use it to provide consistency
		// in the read operations (read after insert for example).
		tx = getTXFromContext(ctx)
	}
	if countExpr == "" {
		countExpr = "*"
	}
	q := sq.Select(fmt.Sprintf("COUNT(%s)", countExpr)).From(tableName).Where(fop)
	sqlQuery, args, err := q.PlaceholderFormat(s.features.PlaceholderFormat).ToSql()
	if err != nil {
		return count, i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> count query: %s`, shortenSQL(sqlQuery))
	l.Tracef(`SQL-> count query args: %+v`, args)
	var rows *sql.Rows
	if tx != nil {
		rows, err = tx.sqlTX.QueryContext(ctx, sqlQuery, args...)
	} else {
		rows, err = s.db.QueryContext(ctx, sqlQuery, args...)
	}
	if err != nil {
		l.Errorf(`SQL count query failed: %s sql=[ %s ]`, err, sqlQuery)
		return count, i18n.WrapError(ctx, err, i18n.MsgDBQueryFailed)
	}
	defer rows.Close()
	if rows.Next() {
		if err = rows.Scan(&count); err != nil {
			return count, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, tableName)
		}
	}
	l.Debugf(`SQL<- count query: %d`, count)
	return count, nil
}

func (s *SQLCommon) queryRes(ctx context.Context, tx *txWrapper, tableName string, fop sq.Sqlizer, fi *database.FilterInfo) *database.FilterResult {
	fr := &database.FilterResult{}
	if fi.Count {
		count, err := s.countQuery(ctx, tx, tableName, fop, fi.CountExpr)
		if err != nil {
			// Log, but continue
			log.L(ctx).Warnf("Unable to return count for query: %s", err)
		}
		fr.TotalCount = &count // could be -1 if the count extract fails - we still return the result
	}
	return fr
}

func (s *SQLCommon) insertTx(ctx context.Context, tx *txWrapper, q sq.InsertBuilder, postCommit func()) (int64, error) {
	return s.insertTxExt(ctx, tx, q, postCommit, false)
}

func (s *SQLCommon) insertTxExt(ctx context.Context, tx *txWrapper, q sq.InsertBuilder, postCommit func(), requestConflictEmptyResult bool) (int64, error) {
	sequences := []int64{-1}
	err := s.insertTxRows(ctx, tx, q, postCommit, sequences, requestConflictEmptyResult)
	return sequences[0], err
}

func (s *SQLCommon) insertTxRows(ctx context.Context, tx *txWrapper, q sq.InsertBuilder, postCommit func(), sequences []int64, requestConflictEmptyResult bool) error {
	l := log.L(ctx)
	q, useQuery := s.provider.ApplyInsertQueryCustomizations(q, requestConflictEmptyResult)

	sqlQuery, args, err := q.PlaceholderFormat(s.features.PlaceholderFormat).ToSql()
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> insert %s`, shortenSQL(sqlQuery))
	l.Tracef(`SQL-> insert query: %s (args: %+v)`, sqlQuery, args)
	if useQuery {
		result, err := tx.sqlTX.QueryContext(ctx, sqlQuery, args...)
		for i := 0; i < len(sequences) && err == nil; i++ {
			if result.Next() {
				err = result.Scan(&sequences[i])
			} else {
				err = i18n.NewError(ctx, i18n.MsgDBNoSequence, i+1)
			}
		}
		if result != nil {
			result.Close()
		}
		if err != nil {
			level := logrus.DebugLevel
			if !requestConflictEmptyResult {
				level = logrus.ErrorLevel
			}
			l.Logf(level, `SQL insert failed (conflictEmptyRequested=%t): %s sql=[ %s ]: %s`, requestConflictEmptyResult, err, sqlQuery, err)
			return i18n.WrapError(ctx, err, i18n.MsgDBInsertFailed)
		}
	} else {
		if len(sequences) > 1 {
			return i18n.WrapError(ctx, err, i18n.MsgDBMultiRowConfigError)
		}
		res, err := tx.sqlTX.ExecContext(ctx, sqlQuery, args...)
		if err != nil {
			l.Errorf(`SQL insert failed: %s sql=[ %s ]: %s`, err, sqlQuery, err)
			return i18n.WrapError(ctx, err, i18n.MsgDBInsertFailed)
		}
		sequences[0], _ = res.LastInsertId()
	}
	l.Debugf(`SQL<- inserted sequences=%v`, sequences)

	if postCommit != nil {
		s.postCommitEvent(tx, postCommit)
	}
	return nil
}

func (s *SQLCommon) deleteTx(ctx context.Context, tx *txWrapper, q sq.DeleteBuilder, postCommit func()) error {
	l := log.L(ctx)
	sqlQuery, args, err := q.PlaceholderFormat(s.features.PlaceholderFormat).ToSql()
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> delete: %s`, shortenSQL(sqlQuery))
	l.Debugf(`SQL-> delete args: %+v`, args)
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

func (s *SQLCommon) updateTx(ctx context.Context, tx *txWrapper, q sq.UpdateBuilder, postCommit func()) (int64, error) {
	l := log.L(ctx)
	sqlQuery, args, err := q.PlaceholderFormat(s.features.PlaceholderFormat).ToSql()
	if err != nil {
		return -1, i18n.WrapError(ctx, err, i18n.MsgDBQueryBuildFailed)
	}
	l.Debugf(`SQL-> update: %s`, shortenSQL(sqlQuery))
	l.Debugf(`SQL-> update query: %s (args: %+v)`, sqlQuery, args)
	res, err := tx.sqlTX.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		l.Errorf(`SQL update failed: %s sql=[ %s ]`, err, sqlQuery)
		return -1, i18n.WrapError(ctx, err, i18n.MsgDBUpdateFailed)
	}
	ra, _ := res.RowsAffected()
	l.Debugf(`SQL<- update affected=%d`, ra)

	if postCommit != nil {
		s.postCommitEvent(tx, postCommit)
	}
	return ra, nil
}

func (s *SQLCommon) postCommitEvent(tx *txWrapper, fn func()) {
	tx.postCommit = append(tx.postCommit, fn)
}

func (s *SQLCommon) addPreCommitEvent(tx *txWrapper, event *fftypes.Event) {
	tx.preCommitEvents = append(tx.preCommitEvents, event)
}

func (tx *txWrapper) tableIsLocked(table string) bool {
	for _, t := range tx.tableLocks {
		if t == table {
			return true
		}
	}
	return false
}

func (s *SQLCommon) lockTableExclusiveTx(ctx context.Context, tx *txWrapper, table string) error {
	l := log.L(ctx)
	if s.features.ExclusiveTableLockSQL != nil && !tx.tableIsLocked(table) {
		sqlQuery := s.features.ExclusiveTableLockSQL(table)

		l.Debugf(`SQL-> lock: %s`, shortenSQL(sqlQuery))
		_, err := tx.sqlTX.ExecContext(ctx, sqlQuery)
		if err != nil {
			l.Errorf(`SQL lock failed: %s sql=[ %s ]`, err, sqlQuery)
			return i18n.WrapError(ctx, err, i18n.MsgDBLockFailed)
		}
		tx.tableLocks = append(tx.tableLocks, table)
		l.Debugf(`SQL<- lock %s`, table)
	}
	return nil
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

	// Only at this stage do we write to the special events Database table, so we know
	// regardless of the higher level logic, the events are always written at this point
	// at the end of the transaction
	if len(tx.preCommitEvents) > 0 {
		if err := s.insertEventsPreCommit(ctx, tx, tx.preCommitEvents); err != nil {
			s.rollbackTx(ctx, tx, false)
			return err
		}
	}

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
