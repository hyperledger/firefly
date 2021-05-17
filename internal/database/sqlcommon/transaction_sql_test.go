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
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/stretchr/testify/assert"
)

func TestTransactionE2EWithDB(t *testing.T) {

	log.SetLevel("debug")
	s := &SQLCommon{}
	ctx := context.Background()
	InitSQLCommon(ctx, s, ensureTestDB(t), nil, &database.Capabilities{}, testSQLOptions())

	// Create a new transaction entry
	transactionId := uuid.New()
	transaction := &fftypes.Transaction{
		ID:   &transactionId,
		Hash: fftypes.NewRandB32(),
		Subject: fftypes.TransactionSubject{
			Type:   fftypes.TransactionTypePin,
			Author: "0x12345",
		},
		Created: fftypes.Now(),
		Status:  fftypes.TransactionStatusPending,
	}
	err := s.UpsertTransaction(ctx, transaction, false)
	assert.NoError(t, err)

	// Check we get the exact same transaction back
	transactionRead, err := s.GetTransactionById(ctx, &transactionId)
	// The generated sequence will have been added
	transaction.Sequence = transactionRead.Sequence
	assert.NoError(t, err)
	assert.NotNil(t, transactionRead)
	transactionJson, _ := json.Marshal(&transaction)
	transactionReadJson, _ := json.Marshal(&transactionRead)
	assert.Equal(t, string(transactionJson), string(transactionReadJson))

	// Update the transaction (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	transactionUpdated := &fftypes.Transaction{
		ID:   &transactionId,
		Hash: fftypes.NewRandB32(),
		Subject: fftypes.TransactionSubject{
			Type:      fftypes.TransactionTypePin,
			Namespace: "ns2",
			Author:    "0x12345",
			Message:   fftypes.NewUUID(),
			Batch:     fftypes.NewUUID(),
		},
		Created:    fftypes.Now(),
		ProtocolID: "0x33333",
		Status:     fftypes.TransactionStatusFailed,
		Info: fftypes.JSONObject{
			"some": "data",
		},
		Confirmed: fftypes.Now(),
	}

	// Check reject hash update
	err = s.UpsertTransaction(context.Background(), transactionUpdated, false)
	assert.Equal(t, database.HashMismatch, err)

	err = s.UpsertTransaction(context.Background(), transactionUpdated, true)
	assert.NoError(t, err)

	// Check we get the exact same message back - note the removal of one of the transaction elements
	transactionRead, err = s.GetTransactionById(ctx, &transactionId)
	assert.NoError(t, err)
	// The generated sequence will have been added
	transactionUpdated.Sequence = transaction.Sequence
	transactionJson, _ = json.Marshal(&transactionUpdated)
	transactionReadJson, _ = json.Marshal(&transactionRead)
	assert.Equal(t, string(transactionJson), string(transactionReadJson))

	// Query back the transaction
	fb := database.TransactionQueryFactory.NewFilter(ctx, 0)
	filter := fb.And(
		fb.Eq("id", transactionUpdated.ID.String()),
		fb.Eq("protocolid", transactionUpdated.ProtocolID),
		fb.Eq("author", transactionUpdated.Subject.Author),
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

	// Update
	up := database.TransactionQueryFactory.NewUpdate(ctx).
		Set("status", fftypes.TransactionStatusConfirmed)
	err = s.UpdateTransaction(ctx, transactionUpdated.ID, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("id", transactionUpdated.ID.String()),
		fb.Eq("status", fftypes.TransactionStatusConfirmed),
	)
	transactions, err = s.GetTransactions(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(transactions))
}

func TestUpsertTransactionFailBegin(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTransaction(context.Background(), &fftypes.Transaction{}, true)
	assert.Regexp(t, "FF10114", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTransactionFailSelect(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	transactionId := uuid.New()
	err := s.UpsertTransaction(context.Background(), &fftypes.Transaction{ID: &transactionId}, true)
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
	err := s.UpsertTransaction(context.Background(), &fftypes.Transaction{ID: &transactionId}, true)
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
	err := s.UpsertTransaction(context.Background(), &fftypes.Transaction{ID: &transactionId}, true)
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
	err := s.UpsertTransaction(context.Background(), &fftypes.Transaction{ID: &transactionId}, true)
	assert.Regexp(t, "FF10119", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransactionByIdSelectFail(t *testing.T) {
	s, mock := getMockDB()
	transactionId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetTransactionById(context.Background(), &transactionId)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransactionByIdNotFound(t *testing.T) {
	s, mock := getMockDB()
	transactionId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetTransactionById(context.Background(), &transactionId)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransactionByIdScanFail(t *testing.T) {
	s, mock := getMockDB()
	transactionId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetTransactionById(context.Background(), &transactionId)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransactionsQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.TransactionQueryFactory.NewFilter(context.Background(), 0).Eq("id", "")
	_, err := s.GetTransactions(context.Background(), f)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransactionsBuildQueryFail(t *testing.T) {
	s, _ := getMockDB()
	f := database.TransactionQueryFactory.NewFilter(context.Background(), 0).Eq("id", map[bool]bool{true: false})
	_, err := s.GetTransactions(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestGettTransactionsReadMessageFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.TransactionQueryFactory.NewFilter(context.Background(), 0).Eq("id", "")
	_, err := s.GetTransactions(context.Background(), f)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionUpdateBeginFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.TransactionQueryFactory.NewUpdate(context.Background()).Set("id", "anything")
	err := s.UpdateTransaction(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err.Error())
}

func TestTransactionUpdateBuildQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	u := database.TransactionQueryFactory.NewUpdate(context.Background()).Set("id", map[bool]bool{true: false})
	err := s.UpdateTransaction(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestTransactionUpdateFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.TransactionQueryFactory.NewUpdate(context.Background()).Set("id", fftypes.NewUUID())
	err := s.UpdateTransaction(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err.Error())
}
