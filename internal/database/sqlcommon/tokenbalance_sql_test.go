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
	balance := &fftypes.TokenBalance{
		PoolProtocolID: "F1",
		TokenIndex:     "1",
		Connector:      "erc1155",
		Namespace:      "ns1",
		Key:            "0x0",
		Balance:        *fftypes.NewBigInt(10),
	}
	balanceJson, _ := json.Marshal(&balance)

	err := s.UpdateTokenBalances(ctx, transfer)
	assert.NoError(t, err)

	// Query back the token balance (by pool ID and identity)
	balanceRead, err := s.GetTokenBalance(ctx, "F1", "1", "0x0")
	assert.NoError(t, err)
	assert.NotNil(t, balanceRead)
	assert.Greater(t, balanceRead.Updated.UnixNano(), int64(0))
	balanceRead.Updated = nil
	balanceReadJson, _ := json.Marshal(&balanceRead)
	assert.Equal(t, string(balanceJson), string(balanceReadJson))

	// Query back the token balance (by query filter)
	fb := database.TokenBalanceQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("poolprotocolid", balance.PoolProtocolID),
		fb.Eq("tokenindex", balance.TokenIndex),
		fb.Eq("key", balance.Key),
	)
	balances, res, err := s.GetTokenBalances(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(balances))
	assert.Equal(t, int64(1), *res.TotalCount)
	assert.Greater(t, balances[0].Updated.UnixNano(), int64(0))
	balances[0].Updated = nil
	balanceReadJson, _ = json.Marshal(balances[0])
	assert.Equal(t, string(balanceJson), string(balanceReadJson))

	// Transfer half to a different address
	transfer.From = "0x0"
	transfer.To = "0x1"
	transfer.Amount = *fftypes.NewBigInt(5)
	err = s.UpdateTokenBalances(ctx, transfer)
	assert.NoError(t, err)

	// Query back the token balance (by pool ID and identity)
	balanceRead, err = s.GetTokenBalance(ctx, "F1", "1", "0x0")
	assert.NoError(t, err)
	assert.NotNil(t, balanceRead)
	assert.Greater(t, balanceRead.Updated.UnixNano(), int64(0))
	balanceRead.Updated = nil
	balanceReadJson, _ = json.Marshal(&balanceRead)
	balance.Balance = *fftypes.NewBigInt(5)
	balanceJson, _ = json.Marshal(&balance)
	assert.Equal(t, string(balanceJson), string(balanceReadJson))

	// Query back the other token balance (by pool ID and identity)
	balanceRead, err = s.GetTokenBalance(ctx, "F1", "1", "0x1")
	assert.NoError(t, err)
	assert.NotNil(t, balanceRead)
	assert.Greater(t, balanceRead.Updated.UnixNano(), int64(0))
	balanceRead.Updated = nil
	balanceReadJson, _ = json.Marshal(&balanceRead)
	balance.Key = "0x1"
	balance.Balance = *fftypes.NewBigInt(5)
	balanceJson, _ = json.Marshal(&balance)
	assert.Equal(t, string(balanceJson), string(balanceReadJson))

	// Query the list of unique accounts
	fb2 := database.TokenBalanceQueryFactory.NewFilter(ctx)
	accounts, fr, err := s.GetTokenAccounts(ctx, fb2.And().Count(true))
	assert.NoError(t, err)
	assert.Equal(t, int64(2), *fr.TotalCount)
	assert.Equal(t, 2, len(accounts))
	assert.Equal(t, "0x1", accounts[0].Key)
	assert.Equal(t, "0x0", accounts[1].Key)
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

func TestGetTokenAccountsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.TokenBalanceQueryFactory.NewFilter(context.Background()).And()
	_, _, err := s.GetTokenAccounts(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTokenAccountsBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.TokenBalanceQueryFactory.NewFilter(context.Background()).Eq("poolprotocolid", map[bool]bool{true: false})
	_, _, err := s.GetTokenAccounts(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestGetTokenAccountsScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"key", "bad"}).AddRow("too many", "columns"))
	f := database.TokenBalanceQueryFactory.NewFilter(context.Background()).And()
	_, _, err := s.GetTokenAccounts(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
