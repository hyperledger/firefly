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
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTransactionE2EWithDB(t *testing.T) {

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new transaction entry
	transactionID := fftypes.NewUUID()
	transaction := &fftypes.Transaction{
		ID:   transactionID,
		Hash: fftypes.NewRandB32(),
		Subject: fftypes.TransactionSubject{
			Type:      fftypes.TransactionTypeBatchPin,
			Namespace: "ns1",
			Signer:    "0x12345",
			Reference: fftypes.NewUUID(),
		},
		Created: fftypes.Now(),
		Status:  fftypes.OpStatusPending,
	}

	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionTransactions, fftypes.ChangeEventTypeCreated, "ns1", transactionID, mock.Anything).Return()
	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionTransactions, fftypes.ChangeEventTypeUpdated, "ns1", transactionID, mock.Anything).Return()

	err := s.UpsertTransaction(ctx, transaction, true, false)
	assert.NoError(t, err)

	// Check we get the exact same transaction back
	transactionRead, err := s.GetTransactionByID(ctx, transactionID)
	assert.NoError(t, err)
	assert.NotNil(t, transactionRead)
	transactionJson, _ := json.Marshal(&transaction)
	transactionReadJson, _ := json.Marshal(&transactionRead)
	assert.Equal(t, string(transactionJson), string(transactionReadJson))

	// Update the transaction (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	transactionUpdated := &fftypes.Transaction{
		ID:   transactionID,
		Hash: fftypes.NewRandB32(),
		Subject: fftypes.TransactionSubject{
			Type:      fftypes.TransactionTypeBatchPin,
			Namespace: "ns1",
			Signer:    "0x12345",
			Reference: fftypes.NewUUID(),
		},
		Created:    fftypes.Now(),
		ProtocolID: "0x33333",
		Status:     fftypes.OpStatusFailed,
		Info: fftypes.JSONObject{
			"some": "data",
		},
	}

	// Check reject hash update
	err = s.UpsertTransaction(context.Background(), transactionUpdated, true, false)
	assert.Equal(t, database.HashMismatch, err)

	err = s.UpsertTransaction(context.Background(), transactionUpdated, true, true)
	assert.NoError(t, err)

	// Check we get the exact same message back - note the removal of one of the transaction elements
	transactionRead, err = s.GetTransactionByID(ctx, transactionID)
	assert.NoError(t, err)
	transactionJson, _ = json.Marshal(&transactionUpdated)
	transactionReadJson, _ = json.Marshal(&transactionRead)
	assert.Equal(t, string(transactionJson), string(transactionReadJson))

	// Query back the transaction
	fb := database.TransactionQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("id", transactionUpdated.ID.String()),
		fb.Eq("protocolid", transactionUpdated.ProtocolID),
		fb.Eq("signer", transactionUpdated.Subject.Signer),
		fb.Gt("created", "0"),
	)
	transactions, err := s.GetTransactions(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(transactions))
	transactionReadJson, _ = json.Marshal(transactions[0])
	assert.Equal(t, string(transactionJson), string(transactionReadJson))

	// Negative test on filter
	filter = fb.And(
		fb.Eq("id", transactionUpdated.ID.String()),
		fb.Eq("created", "0"),
	)
	transactions, err = s.GetTransactions(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(transactions))

	// Update
	up := database.TransactionQueryFactory.NewUpdate(ctx).
		Set("status", fftypes.OpStatusSucceeded)
	err = s.UpdateTransaction(ctx, transactionUpdated.ID, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("id", transactionUpdated.ID.String()),
		fb.Eq("status", fftypes.OpStatusSucceeded),
	)
	transactions, err = s.GetTransactions(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(transactions))
}

func TestUpsertTransactionFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTransaction(context.Background(), &fftypes.Transaction{}, true, true)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTransactionFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	transactionID := fftypes.NewUUID()
	err := s.UpsertTransaction(context.Background(), &fftypes.Transaction{ID: transactionID}, true, true)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTransactionFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	transactionID := fftypes.NewUUID()
	err := s.UpsertTransaction(context.Background(), &fftypes.Transaction{ID: transactionID}, true, true)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTransactionFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	transactionID := fftypes.NewUUID()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(transactionID.String()))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertTransaction(context.Background(), &fftypes.Transaction{ID: transactionID}, true, true)
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTransactionFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	transactionID := fftypes.NewUUID()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTransaction(context.Background(), &fftypes.Transaction{ID: transactionID}, true, true)
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransactionByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	transactionID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetTransactionByID(context.Background(), transactionID)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransactionByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	transactionID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetTransactionByID(context.Background(), transactionID)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransactionByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	transactionID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetTransactionByID(context.Background(), transactionID)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransactionsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.TransactionQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, err := s.GetTransactions(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTransactionsBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.TransactionQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	_, err := s.GetTransactions(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestGettTransactionsReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.TransactionQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, err := s.GetTransactions(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTxSubmissionUpdateBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.TransactionQueryFactory.NewUpdate(context.Background()).Set("id", "anything")
	err := s.UpdateTransaction(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err)
}

func TestTxSubmissionUpdateBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	u := database.TransactionQueryFactory.NewUpdate(context.Background()).Set("id", map[bool]bool{true: false})
	err := s.UpdateTransaction(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestTxSubmissionUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.TransactionQueryFactory.NewUpdate(context.Background()).Set("id", fftypes.NewUUID())
	err := s.UpdateTransaction(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err)
}
