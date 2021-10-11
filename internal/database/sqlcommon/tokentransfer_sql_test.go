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
	"github.com/stretchr/testify/mock"
)

func TestTokenTransferE2EWithDB(t *testing.T) {
	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new token transfer entry
	transfer := &fftypes.TokenTransfer{
		LocalID:        fftypes.NewUUID(),
		Type:           fftypes.TokenTransferTypeTransfer,
		PoolProtocolID: "F1",
		TokenIndex:     "1",
		From:           "0x01",
		To:             "0x02",
		ProtocolID:     "12345",
		MessageHash:    fftypes.NewRandB32(),
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenTransfer,
			ID:   fftypes.NewUUID(),
		},
	}
	transfer.Amount.Int().SetInt64(10)

	s.callbacks.On("UUIDCollectionEvent", database.CollectionTokenTransfers, fftypes.ChangeEventTypeCreated, transfer.LocalID, mock.Anything).
		Return().Once()
	s.callbacks.On("UUIDCollectionEvent", database.CollectionTokenTransfers, fftypes.ChangeEventTypeUpdated, transfer.LocalID, mock.Anything).
		Return().Once()

	err := s.UpsertTokenTransfer(ctx, transfer)
	assert.NoError(t, err)

	assert.NotNil(t, transfer.Created)
	transferJson, _ := json.Marshal(&transfer)

	// Query back the token transfer (by ID)
	transferRead, err := s.GetTokenTransfer(ctx, transfer.LocalID)
	assert.NoError(t, err)
	assert.NotNil(t, transferRead)
	transferReadJson, _ := json.Marshal(&transferRead)
	assert.Equal(t, string(transferJson), string(transferReadJson))

	// Query back the token transfer (by query filter)
	fb := database.TokenTransferQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("poolprotocolid", transfer.PoolProtocolID),
		fb.Eq("tokenindex", transfer.TokenIndex),
		fb.Eq("from", transfer.From),
		fb.Eq("to", transfer.To),
		fb.Eq("protocolid", transfer.ProtocolID),
		fb.Eq("created", transfer.Created),
	)
	transfers, res, err := s.GetTokenTransfers(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(transfers))
	assert.Equal(t, int64(1), *res.TotalCount)
	transferReadJson, _ = json.Marshal(transfers[0])
	assert.Equal(t, string(transferJson), string(transferReadJson))

	// Update the token transfer
	transfer.Type = fftypes.TokenTransferTypeMint
	transfer.Amount.Int().SetInt64(1)
	transfer.To = "0x03"
	err = s.UpsertTokenTransfer(ctx, transfer)
	assert.NoError(t, err)

	// Query back the token transfer (by ID)
	transferRead, err = s.GetTokenTransfer(ctx, transfer.LocalID)
	assert.NoError(t, err)
	assert.NotNil(t, transferRead)
	transferJson, _ = json.Marshal(&transfer)
	transferReadJson, _ = json.Marshal(&transferRead)
	assert.Equal(t, string(transferJson), string(transferReadJson))
}

func TestUpsertTokenTransferFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTokenTransfer(context.Background(), &fftypes.TokenTransfer{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTokenTransferFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTokenTransfer(context.Background(), &fftypes.TokenTransfer{})
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTokenTransferFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertTokenTransfer(context.Background(), &fftypes.TokenTransfer{})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTokenTransferFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"protocolid"}).AddRow("1"))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertTokenTransfer(context.Background(), &fftypes.TokenTransfer{})
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTokenTransferFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"protocolid"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTokenTransfer(context.Background(), &fftypes.TokenTransfer{})
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTokenTransferByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetTokenTransfer(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTokenTransferByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"protocolid"}))
	msg, err := s.GetTokenTransfer(context.Background(), fftypes.NewUUID())
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTokenTransferByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"protocolid"}).AddRow("only one"))
	_, err := s.GetTokenTransfer(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTokenTransfersQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.TokenTransferQueryFactory.NewFilter(context.Background()).Eq("protocolid", "")
	_, _, err := s.GetTokenTransfers(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTokenTransfersBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.TokenTransferQueryFactory.NewFilter(context.Background()).Eq("protocolid", map[bool]bool{true: false})
	_, _, err := s.GetTokenTransfers(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestGetTokenTransfersScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"protocolid"}).AddRow("only one"))
	f := database.TokenTransferQueryFactory.NewFilter(context.Background()).Eq("protocolid", "")
	_, _, err := s.GetTokenTransfers(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
