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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestTokenAccountE2EWithDB(t *testing.T) {

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new token account
	account := &fftypes.TokenAccount{
		PoolProtocolID: "F1",
		TokenIndex:     "1",
		Identity:       "0x0",
		Balance:        10,
	}
	accountJson, _ := json.Marshal(&account)

	err := s.UpsertTokenAccount(ctx, account)
	assert.NoError(t, err)

	// Query back the token account (by pool ID and identity)
	accountRead, err := s.GetTokenAccount(ctx, "F1", "1", "0x0")
	assert.NoError(t, err)
	assert.NotNil(t, accountRead)
	accountReadJson, _ := json.Marshal(&accountRead)
	assert.Equal(t, string(accountJson), string(accountReadJson))

	// Query back the token account (by query filter)
	fb := database.TokenAccountQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("poolprotocolid", account.PoolProtocolID),
		fb.Eq("tokenindex", account.TokenIndex),
		fb.Eq("identity", account.Identity),
	)
	accounts, res, err := s.GetTokenAccounts(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(accounts))
	assert.Equal(t, int64(1), *res.TotalCount)
	accountReadJson, _ = json.Marshal(accounts[0])
	assert.Equal(t, string(accountJson), string(accountReadJson))
}

func TestUpsertTokenAccountFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTokenAccount(context.Background(), &fftypes.TokenAccount{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTokenAccountFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTokenAccount(context.Background(), &fftypes.TokenAccount{})
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTokenAccountFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertTokenAccount(context.Background(), &fftypes.TokenAccount{})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTokenAccountFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertTokenAccount(context.Background(), &fftypes.TokenAccount{})
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTokenAccountFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTokenAccount(context.Background(), &fftypes.TokenAccount{})
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTokenAccountInsertSuccess(t *testing.T) {
	s, db := newMockProvider().init()
	callbacks := &databasemocks.Callbacks{}
	s.SQLCommon.callbacks = callbacks
	account := &fftypes.TokenAccount{
		PoolProtocolID: "F1",
		TokenIndex:     "1",
		Identity:       "0x0",
		Balance:        10,
	}

	db.ExpectBegin()
	db.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	db.ExpectExec("INSERT .*").
		WithArgs("F1", "1", "0x0", 10).
		WillReturnResult(sqlmock.NewResult(1, 1))
	db.ExpectCommit()
	err := s.UpsertTokenAccount(context.Background(), account)
	assert.NoError(t, err)
	assert.NoError(t, db.ExpectationsWereMet())
}

func TestUpsertTokenAccountUpdateSuccess(t *testing.T) {
	s, db := newMockProvider().init()
	callbacks := &databasemocks.Callbacks{}
	s.SQLCommon.callbacks = callbacks
	account := &fftypes.TokenAccount{
		PoolProtocolID: "F1",
		TokenIndex:     "1",
		Identity:       "0x0",
		Balance:        10,
	}

	db.ExpectBegin()
	db.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"seq"}).AddRow("1"))
	db.ExpectExec("UPDATE .*").WillReturnResult(sqlmock.NewResult(1, 1))
	db.ExpectCommit()
	err := s.UpsertTokenAccount(context.Background(), account)
	assert.NoError(t, err)
	assert.NoError(t, db.ExpectationsWereMet())
}

func TestGetTokenAccountSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetTokenAccount(context.Background(), "F1", "1", "0x0")
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTokenAccountNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetTokenAccount(context.Background(), "F1", "1", "0x0")
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTokenAccountScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetTokenAccount(context.Background(), "F1", "1", "0x0")
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTokenAccountsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.TokenAccountQueryFactory.NewFilter(context.Background()).Eq("poolprotocolid", "")
	_, _, err := s.GetTokenAccounts(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTokenAccountsBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.TokenAccountQueryFactory.NewFilter(context.Background()).Eq("poolprotocolid", map[bool]bool{true: false})
	_, _, err := s.GetTokenAccounts(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestGetTokenAccountsScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"poolprotocolid"}).AddRow("only one"))
	f := database.TokenAccountQueryFactory.NewFilter(context.Background()).Eq("poolprotocolid", "")
	_, _, err := s.GetTokenAccounts(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
