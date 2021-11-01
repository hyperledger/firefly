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
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestTokenBalanceE2EWithDB(t *testing.T) {

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new token account
	transfer := &fftypes.TokenTransfer{
		PoolProtocolID: "F1",
		TokenIndex:     "1",
		Connector:      "erc1155",
		Namespace:      "ns1",
		To:             "0x0",
		Amount:         *fftypes.NewBigInt(10),
	}
	account := &fftypes.TokenBalance{
		PoolProtocolID: "F1",
		TokenIndex:     "1",
		Connector:      "erc1155",
		Namespace:      "ns1",
		Key:            "0x0",
		Balance:        *fftypes.NewBigInt(10),
	}
	accountJson, _ := json.Marshal(&account)

	err := s.UpdateTokenBalances(ctx, transfer)
	assert.NoError(t, err)

	// Query back the token account (by pool ID and identity)
	accountRead, err := s.GetTokenBalance(ctx, "F1", "1", "0x0")
	assert.NoError(t, err)
	assert.NotNil(t, accountRead)
	assert.Greater(t, accountRead.Updated.UnixNano(), int64(0))
	accountRead.Updated = nil
	accountReadJson, _ := json.Marshal(&accountRead)
	assert.Equal(t, string(accountJson), string(accountReadJson))

	// Query back the token account (by query filter)
	fb := database.TokenBalanceQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("poolprotocolid", account.PoolProtocolID),
		fb.Eq("tokenindex", account.TokenIndex),
		fb.Eq("key", account.Key),
	)
	accounts, res, err := s.GetTokenBalances(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(accounts))
	assert.Equal(t, int64(1), *res.TotalCount)
	assert.Greater(t, accounts[0].Updated.UnixNano(), int64(0))
	accounts[0].Updated = nil
	accountReadJson, _ = json.Marshal(accounts[0])
	assert.Equal(t, string(accountJson), string(accountReadJson))

	// Transfer half to a different account
	transfer.From = "0x0"
	transfer.To = "0x1"
	transfer.Amount = *fftypes.NewBigInt(5)
	err = s.UpdateTokenBalances(ctx, transfer)
	assert.NoError(t, err)

	// Query back the token account (by pool ID and identity)
	accountRead, err = s.GetTokenBalance(ctx, "F1", "1", "0x0")
	assert.NoError(t, err)
	assert.NotNil(t, accountRead)
	assert.Greater(t, accountRead.Updated.UnixNano(), int64(0))
	accountRead.Updated = nil
	accountReadJson, _ = json.Marshal(&accountRead)
	account.Balance = *fftypes.NewBigInt(5)
	accountJson, _ = json.Marshal(&account)
	assert.Equal(t, string(accountJson), string(accountReadJson))

	// Query back the other token account (by pool ID and identity)
	accountRead, err = s.GetTokenBalance(ctx, "F1", "1", "0x1")
	assert.NoError(t, err)
	assert.NotNil(t, accountRead)
	assert.Greater(t, accountRead.Updated.UnixNano(), int64(0))
	accountRead.Updated = nil
	accountReadJson, _ = json.Marshal(&accountRead)
	account.Key = "0x1"
	account.Balance = *fftypes.NewBigInt(5)
	accountJson, _ = json.Marshal(&account)
	assert.Equal(t, string(accountJson), string(accountReadJson))
}

func TestUpdateTokenBalancesFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpdateTokenBalances(context.Background(), &fftypes.TokenTransfer{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTokenBalancesFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.UpdateTokenBalances(context.Background(), &fftypes.TokenTransfer{To: "0x0"})
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTokenBalancesFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpdateTokenBalances(context.Background(), &fftypes.TokenTransfer{From: "0x0"})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTokenBalancesFailInsert2(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpdateTokenBalances(context.Background(), &fftypes.TokenTransfer{To: "0x0"})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTokenBalancesFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows(tokenBalanceColumns).AddRow("F1", "1", "", "", "0x0", "0", 0))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpdateTokenBalances(context.Background(), &fftypes.TokenTransfer{To: "0x0"})
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTokenBalancesFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpdateTokenBalances(context.Background(), &fftypes.TokenTransfer{To: "0x0"})
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTokenBalanceNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetTokenBalance(context.Background(), "F1", "1", "0x0")
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTokenBalanceScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetTokenBalance(context.Background(), "F1", "1", "0x0")
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTokenBalancesQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.TokenBalanceQueryFactory.NewFilter(context.Background()).Eq("poolprotocolid", "")
	_, _, err := s.GetTokenBalances(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTokenBalancesBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.TokenBalanceQueryFactory.NewFilter(context.Background()).Eq("poolprotocolid", map[bool]bool{true: false})
	_, _, err := s.GetTokenBalances(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestGetTokenBalancesScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"poolprotocolid"}).AddRow("only one"))
	f := database.TokenBalanceQueryFactory.NewFilter(context.Background()).Eq("poolprotocolid", "")
	_, _, err := s.GetTokenBalances(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
