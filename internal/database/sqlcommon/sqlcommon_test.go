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
	"github.com/stretchr/testify/assert"
)

func TestInitSQLCommon(t *testing.T) {
	s := newQLTestProvider(t)
	defer s.Close()
	assert.NotNil(t, s.Capabilities())
	assert.NotNil(t, s.DB())
}

func TestInitSQLCommonMissingOptions(t *testing.T) {
	s := &SQLCommon{}
	err := s.Init(context.Background(), nil, nil, nil, nil)
	assert.Regexp(t, "FF10112", err.Error())
}

func TestInitSQLCommonOpenFailed(t *testing.T) {
	mp := newMockProvider()
	mp.openError = fmt.Errorf("pop")
	err := mp.SQLCommon.Init(context.Background(), mp, mp.prefix, mp.callbacks, mp.capabilities)
	assert.Regexp(t, "FF10112.*pop", err.Error())
}

func TestInitSQLCommonMigrationOpenFailed(t *testing.T) {
	mp := newMockProvider()
	mp.prefix.Set(SQLConfMigrationsAuto, true)
	mp.getMigrationDriverError = fmt.Errorf("pop")
	err := mp.SQLCommon.Init(context.Background(), mp, mp.prefix, mp.callbacks, mp.capabilities)
	assert.Regexp(t, "FF10163.*pop", err.Error())
}

func TestQueryTxBadSQL(t *testing.T) {
	tp := newQLTestProvider(t)
	_, err := tp.queryTx(context.Background(), nil, sq.SelectBuilder{})
	assert.Regexp(t, "FF10113", err.Error())
}

func TestInsertTxPostgreSQLReturnedSyntax(t *testing.T) {
	s, mdb := newMockProvider().init()
	mdb.ExpectBegin()
	mdb.ExpectQuery("INSERT.*").WillReturnRows(sqlmock.NewRows([]string{"seq"}).AddRow(12345))
	ctx, tx, _, err := s.beginOrUseTx(context.Background())
	assert.NoError(t, err)
	s.fakePSQLInsert = true
	sb := sq.Insert("table").Columns("col1").Values(("val1"))
	sequence, err := s.insertTx(ctx, tx, sb)
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
	_, err = s.insertTx(ctx, tx, sb)
	assert.Regexp(t, "FF10116", err.Error())
}

func TestInsertTxBadSQL(t *testing.T) {
	s, _ := newMockProvider().init()
	_, err := s.insertTx(context.Background(), nil, sq.InsertBuilder{})
	assert.Regexp(t, "FF10113", err.Error())
}

func TestUpdateTxBadSQL(t *testing.T) {
	s, _ := newMockProvider().init()
	err := s.updateTx(context.Background(), nil, sq.UpdateBuilder{})
	assert.Regexp(t, "FF10113", err.Error())
}

func TestDeleteTxBadSQL(t *testing.T) {
	s, _ := newMockProvider().init()
	err := s.deleteTx(context.Background(), nil, sq.DeleteBuilder{})
	assert.Regexp(t, "FF10113", err.Error())
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
		_, err = s.insertTx(ctx, tx, sq.Insert("test").Columns("test").Values("test"))
		assert.NoError(t, err)
		err = s.commitTx(ctx, tx, ac)
		assert.NoError(t, err)

		// Second insert
		ctx, tx, ac, err = s.beginOrUseTx(ctx)
		assert.NoError(t, err)
		_, err = s.insertTx(ctx, tx, sq.Insert("test").Columns("test").Values("test"))
		assert.NoError(t, err)
		err = s.commitTx(ctx, tx, ac)
		assert.NoError(t, err)

		// Query, not specifying a transaction
		_, err = s.query(ctx, sq.Select("test").From("test"))
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
	assert.Regexp(t, "FF10114", err.Error())
}

func TestRunAsGroupFunctionFails(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT.*").WillReturnResult(driver.ResultNoRows)
	mock.ExpectRollback()
	err := s.RunAsGroup(context.Background(), func(ctx context.Context) (err error) {
		ctx, tx, ac, err := s.beginOrUseTx(ctx)
		assert.NoError(t, err)
		_, err = s.insertTx(ctx, tx, sq.Insert("test").Columns("test").Values("test"))
		assert.NoError(t, err)
		s.rollbackTx(ctx, tx, ac) // won't actually rollback
		assert.NoError(t, err)

		return fmt.Errorf("pop")
	})
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Regexp(t, "pop", err.Error())
}

func TestRunAsGroupCommitFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.RunAsGroup(context.Background(), func(ctx context.Context) (err error) {
		return
	})
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Regexp(t, "FF10119", err.Error())
}

func TestRollbackFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectRollback().WillReturnError(fmt.Errorf("pop"))
	s.rollbackTx(context.Background(), &txWrapper{sqlTX: tx}, false)
	assert.NoError(t, mock.ExpectationsWereMet())
}
