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
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	sq "github.com/Masterminds/squirrel"
	"github.com/golang-migrate/migrate/v4"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func uuidMatches(id1 *fftypes.UUID) interface{} {
	return mock.MatchedBy(func(id2 *fftypes.UUID) bool {
		return id1.Equals(id2)
	})
}

func TestInitSQLCommon(t *testing.T) {
	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	assert.NotNil(t, s.Capabilities())
	assert.NotNil(t, s.DB())
}

func TestInitSQLCommonMissingOptions(t *testing.T) {
	s := &SQLCommon{}
	err := s.Init(context.Background(), nil, nil, nil, nil)
	assert.Regexp(t, "FF10112", err)
}

func TestInitSQLCommonOpenFailed(t *testing.T) {
	mp := newMockProvider()
	mp.openError = fmt.Errorf("pop")
	err := mp.SQLCommon.Init(context.Background(), mp, mp.prefix, mp.callbacks, mp.capabilities)
	assert.Regexp(t, "FF10112.*pop", err)
}

func TestInitSQLCommonMigrationOpenFailed(t *testing.T) {
	mp := newMockProvider()
	mp.prefix.Set(SQLConfMigrationsAuto, true)
	mp.getMigrationDriverError = fmt.Errorf("pop")
	err := mp.SQLCommon.Init(context.Background(), mp, mp.prefix, mp.callbacks, mp.capabilities)
	assert.Regexp(t, "FF10163.*pop", err)
}

func TestMigrationUpDown(t *testing.T) {
	tp, cleanup := newSQLiteTestProvider(t)
	defer cleanup()

	driver, err := tp.GetMigrationDriver(tp.db)
	assert.NoError(t, err)
	var m *migrate.Migrate
	m, err = migrate.NewWithDatabaseInstance(
		"file://../../../db/migrations/sqlite",
		tp.MigrationsDir(), driver)
	assert.NoError(t, err)
	err = m.Down()
	assert.NoError(t, err)
}

func TestQueryTxBadSQL(t *testing.T) {
	tp, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	_, _, err := tp.queryTx(context.Background(), nil, sq.SelectBuilder{})
	assert.Regexp(t, "FF10113", err)
}

func TestInsertTxPostgreSQLReturnedSyntax(t *testing.T) {
	s, mdb := newMockProvider().init()
	mdb.ExpectBegin()
	mdb.ExpectQuery("INSERT.*").WillReturnRows(sqlmock.NewRows([]string{sequenceColumn}).AddRow(12345))
	ctx, tx, _, err := s.beginOrUseTx(context.Background())
	assert.NoError(t, err)
	s.fakePSQLInsert = true
	sb := sq.Insert("table").Columns("col1").Values(("val1"))
	sequence, err := s.insertTx(ctx, tx, sb, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), sequence)
}

func TestInsertTxPostgreSQLReturnedSyntaxFail(t *testing.T) {
	s, mdb := newMockProvider().init()
	mdb.ExpectBegin()
	mdb.ExpectQuery("INSERT.*").WillReturnError(fmt.Errorf("pop"))
	ctx, tx, _, err := s.beginOrUseTx(context.Background())
	assert.NoError(t, err)
	s.fakePSQLInsert = true
	sb := sq.Insert("table").Columns("col1").Values(("val1"))
	_, err = s.insertTx(ctx, tx, sb, nil)
	assert.Regexp(t, "FF10116", err)
}

func TestInsertTxBadSQL(t *testing.T) {
	s, _ := newMockProvider().init()
	_, err := s.insertTx(context.Background(), nil, sq.InsertBuilder{}, nil)
	assert.Regexp(t, "FF10113", err)
}

func TestUpdateTxBadSQL(t *testing.T) {
	s, _ := newMockProvider().init()
	err := s.updateTx(context.Background(), nil, sq.UpdateBuilder{}, nil)
	assert.Regexp(t, "FF10113", err)
}

func TestDeleteTxBadSQL(t *testing.T) {
	s, _ := newMockProvider().init()
	err := s.deleteTx(context.Background(), nil, sq.DeleteBuilder{}, nil)
	assert.Regexp(t, "FF10113", err)
}

func TestDeleteTxZeroRowsAffected(t *testing.T) {
	s, mdb := newMockProvider().init()
	mdb.ExpectBegin()
	mdb.ExpectExec("DELETE.*").WillReturnResult(driver.ResultNoRows)
	ctx, tx, _, err := s.beginOrUseTx(context.Background())
	assert.NoError(t, err)
	s.fakePSQLInsert = true
	sb := sq.Delete("table")
	err = s.deleteTx(ctx, tx, sb, nil)
	assert.Regexp(t, "FF10109", err)
}

func TestRunAsGroup(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT.*").WillReturnResult(driver.ResultNoRows)
	mock.ExpectExec("INSERT.*").WillReturnResult(driver.ResultNoRows)
	mock.ExpectQuery("SELECT.*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectCommit()

	err := s.RunAsGroup(context.Background(), func(ctx context.Context) (err error) {
		// First insert
		ctx, tx, ac, err := s.beginOrUseTx(ctx)
		assert.NoError(t, err)
		_, err = s.insertTx(ctx, tx, sq.Insert("test").Columns("test").Values("test"), nil)
		assert.NoError(t, err)
		err = s.commitTx(ctx, tx, ac)
		assert.NoError(t, err)

		// Second insert
		ctx, tx, ac, err = s.beginOrUseTx(ctx)
		assert.NoError(t, err)
		_, err = s.insertTx(ctx, tx, sq.Insert("test").Columns("test").Values("test"), nil)
		assert.NoError(t, err)
		err = s.commitTx(ctx, tx, ac)
		assert.NoError(t, err)

		// Query, not specifying a transaction
		_, _, err = s.query(ctx, sq.Select("test").From("test"))
		assert.NoError(t, err)
		return
	})

	assert.NoError(t, mock.ExpectationsWereMet())
	assert.NoError(t, err)
}

func TestRunAsGroupBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.RunAsGroup(context.Background(), func(ctx context.Context) (err error) {
		return
	})
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Regexp(t, "FF10114", err)
}

func TestRunAsGroupFunctionFails(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT.*").WillReturnResult(driver.ResultNoRows)
	mock.ExpectRollback()
	err := s.RunAsGroup(context.Background(), func(ctx context.Context) (err error) {
		ctx, tx, ac, err := s.beginOrUseTx(ctx)
		assert.NoError(t, err)
		_, err = s.insertTx(ctx, tx, sq.Insert("test").Columns("test").Values("test"), nil)
		assert.NoError(t, err)
		s.rollbackTx(ctx, tx, ac) // won't actually rollback
		assert.NoError(t, err)

		return fmt.Errorf("pop")
	})
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Regexp(t, "pop", err)
}

func TestRunAsGroupCommitFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.RunAsGroup(context.Background(), func(ctx context.Context) (err error) {
		return
	})
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Regexp(t, "FF10119", err)
}

func TestRollbackFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectRollback().WillReturnError(fmt.Errorf("pop"))
	s.rollbackTx(context.Background(), &txWrapper{sqlTX: tx}, false)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTXConcurrency(t *testing.T) {
	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()

	// This test exercise our transaction begin/use/end code for parallel execution.
	// It was originally written to validate the pure Go implementation of SQLite:
	// https://gitlab.com/cznic/sqlite
	// Sadly we found problems with that implementation, and are now only using
	// the well adopted CGO implementation.
	// When the e2e DB tests move to being able to be run against any database, this
	// test should be included.
	// (additional refactor required - see https://github.com/hyperledger-labs/firefly/issues/119)

	_, err := s.db.Exec(`
		CREATE TABLE testconc ( seq INTEGER PRIMARY KEY AUTOINCREMENT, val VARCHAR(256) );
	`)
	assert.NoError(t, err)

	racer := func(done chan struct{}, name string) func() {
		return func() {
			defer close(done)
			for i := 0; i < 5; i++ {
				ctx, tx, ac, err := s.beginOrUseTx(context.Background())
				assert.NoError(t, err)
				val := fmt.Sprintf("%s/%d", name, i)
				sequence, err := s.insertTx(ctx, tx, sq.Insert("testconc").Columns("val").Values(val), nil)
				assert.NoError(t, err)
				t.Logf("%s = %d", val, sequence)
				err = s.commitTx(ctx, tx, ac)
				assert.NoError(t, err)
			}
		}
	}
	flags := make([]chan struct{}, 5)
	for i := 0; i < len(flags); i++ {
		flags[i] = make(chan struct{})
		go racer(flags[i], fmt.Sprintf("racer_%d", i))()
	}
	for i := 0; i < len(flags); i++ {
		<-flags[i]
		t.Logf("Racer %d complete", i)
	}
}

func TestCountQueryBadSQL(t *testing.T) {
	s, _ := newMockProvider().init()
	_, err := s.countQuery(context.Background(), nil, "", sq.Insert("wrong"))
	assert.Regexp(t, "FF10113", err)
}

func TestCountQueryQueryFailed(t *testing.T) {
	s, mdb := newMockProvider().init()
	mdb.ExpectQuery("SELECT COUNT.*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.countQuery(context.Background(), nil, "table1", sq.Eq{"col1": "val1"})
	assert.Regexp(t, "FF10115.*pop", err)
}

func TestCountQueryScanFailTx(t *testing.T) {
	s, mdb := newMockProvider().init()
	mdb.ExpectBegin()
	mdb.ExpectQuery("SELECT COUNT.*").WillReturnRows(sqlmock.NewRows([]string{"col1"}).AddRow("not a number"))
	ctx, tx, _, err := s.beginOrUseTx(context.Background())
	assert.NoError(t, err)
	_, err = s.countQuery(ctx, tx, "table1", sq.Eq{"col1": "val1"})
	assert.Regexp(t, "FF10121", err)
}

func TestQueryResSwallowError(t *testing.T) {
	s, _ := newMockProvider().init()
	res := s.queryRes(context.Background(), nil, "", sq.Insert("wrong"), &database.FilterInfo{
		Count: true,
	})
	assert.Equal(t, int64(-1), *res.TotalCount)
}
