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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/persistence"
	"github.com/stretchr/testify/assert"
)

func TestTransaction2EWithDB(t *testing.T) {

	log.SetLevel("debug")
	s := &SQLCommon{}
	ctx := context.Background()
	InitSQLCommon(ctx, s, ensureTestDB(t), testSQLOptions())

	// Create a new transaction entry
	transactionId := uuid.New()
	transaction := &fftypes.Transaction{
		ID:     &transactionId,
		Type:   fftypes.TransactionTypePin,
		Author: "0x12345",
	}
	err := s.UpsertTransaction(ctx, transaction)
	assert.NoError(t, err)

	// Check we get the exact same transaction back
	transactionRead, err := s.GetTransactionById(ctx, "ns1", &transactionId)
	assert.NoError(t, err)
	assert.NotNil(t, transactionRead)
	transactionJson, _ := json.Marshal(&transaction)
	transactionReadJson, _ := json.Marshal(&transactionRead)
	assert.Equal(t, string(transactionJson), string(transactionReadJson))

	// Update the transaction (this is testing what's possible at the persistence layer,
	// and does not account for the verification that happens at the higher level)
	transactionUpdated := &fftypes.Transaction{
		ID:         &transactionId,
		Type:       fftypes.TransactionTypePin,
		Namespace:  "ns2",
		Author:     "0x12345",
		Created:    fftypes.NowMillis(),
		TrackingID: "0x22222",
		ProtocolID: "0x33333",
		Info: fftypes.JSONData{
			"some": "data",
		},
		Confirmed: fftypes.NowMillis(),
	}
	err = s.UpsertTransaction(context.Background(), transactionUpdated)
	assert.NoError(t, err)

	// Check we get the exact same message back - note the removal of one of the transaction elements
	transactionRead, err = s.GetTransactionById(ctx, "ns1", &transactionId)
	assert.NoError(t, err)
	transactionJson, _ = json.Marshal(&transactionUpdated)
	transactionReadJson, _ = json.Marshal(&transactionRead)
	assert.Equal(t, string(transactionJson), string(transactionReadJson))

	// Query back the transaction
	fb := persistence.TransactionFilterBuilder.New(ctx, 0)
	filter := fb.And(
		fb.Eq("id", transactionUpdated.ID.String()),
		fb.Eq("trackingid", transactionUpdated.TrackingID),
		fb.Eq("protocolid", transactionUpdated.ProtocolID),
		fb.Eq("author", transactionUpdated.Author),
		fb.Gt("created", "0"),
		fb.Gt("confirmed", "0"),
	)
	transactions, err := s.GetTransactions(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(transactions))
	transactionReadJson, _ = json.Marshal(transactions[0])
	assert.Equal(t, string(transactionJson), string(transactionReadJson))

	// Negative test on filter
	filter = fb.And(
		fb.Eq("id", transactionUpdated.ID.String()),
		fb.Eq("confirmed", "0"),
	)
	transactions, err = s.GetTransactions(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(transactions))
}

func TestUpsertTransactionFailBegin(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTransaction(context.Background(), &fftypes.Transaction{})
	assert.Regexp(t, "FF10114", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTransactionFailSelect(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	transactionId := uuid.New()
	err := s.UpsertTransaction(context.Background(), &fftypes.Transaction{ID: &transactionId})
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTransactionFailInsert(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	transactionId := uuid.New()
	err := s.UpsertTransaction(context.Background(), &fftypes.Transaction{ID: &transactionId})
	assert.Regexp(t, "FF10116", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTransactionFailUpdate(t *testing.T) {
	s, mock := getMockDB()
	transactionId := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(transactionId.String()))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertTransaction(context.Background(), &fftypes.Transaction{ID: &transactionId})
	assert.Regexp(t, "FF10117", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTransactionFailCommit(t *testing.T) {
	s, mock := getMockDB()
	transactionId := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTransaction(context.Background(), &fftypes.Transaction{ID: &transactionId})
	assert.Regexp(t, "FF10119", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransactionByIdSelectFail(t *testing.T) {
	s, mock := getMockDB()
	transactionId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetTransactionById(context.Background(), "ns1", &transactionId)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransactionByIdNotFound(t *testing.T) {
	s, mock := getMockDB()
	transactionId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetTransactionById(context.Background(), "ns1", &transactionId)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransactionByIdScanFail(t *testing.T) {
	s, mock := getMockDB()
	transactionId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetTransactionById(context.Background(), "ns1", &transactionId)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransactionsQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := persistence.TransactionFilterBuilder.New(context.Background(), 0).Eq("id", "")
	_, err := s.GetTransactions(context.Background(), f)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransactionsBuildQueryFail(t *testing.T) {
	s, _ := getMockDB()
	f := persistence.TransactionFilterBuilder.New(context.Background(), 0).Eq("id", map[bool]bool{true: false})
	_, err := s.GetTransactions(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestGettTransactionsReadMessageFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := persistence.TransactionFilterBuilder.New(context.Background(), 0).Eq("id", "")
	_, err := s.GetTransactions(context.Background(), f)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}
