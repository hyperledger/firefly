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

func TestBatch2EWithDB(t *testing.T) {

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new batch entry
	batchID := fftypes.NewUUID()
	msgID1 := fftypes.NewUUID()
	batch := &fftypes.Batch{
		ID:   batchID,
		Type: fftypes.MessageTypeBroadcast,
		Identity: fftypes.Identity{
			Key:    "0x12345",
			Author: "did:firefly:org/abcd",
		},
		Namespace: "ns1",
		Hash:      fftypes.NewRandB32(),
		Created:   fftypes.Now(),
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{ID: msgID1}},
			},
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeNone,
			},
		},
	}

	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionBatches, fftypes.ChangeEventTypeCreated, "ns1", batchID, mock.Anything).Return()
	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionBatches, fftypes.ChangeEventTypeUpdated, "ns1", batchID, mock.Anything).Return()

	err := s.UpsertBatch(ctx, batch, true)
	assert.NoError(t, err)

	// Check we get the exact same batch back
	batchRead, err := s.GetBatchByID(ctx, batchID)
	assert.NoError(t, err)
	assert.NotNil(t, batchRead)
	batchJson, _ := json.Marshal(&batch)
	batchReadJson, _ := json.Marshal(&batchRead)
	assert.Equal(t, string(batchJson), string(batchReadJson))

	// Update the batch (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	txid := fftypes.NewUUID()
	msgID2 := fftypes.NewUUID()
	payloadRef := ""
	batchUpdated := &fftypes.Batch{
		ID:   batchID,
		Type: fftypes.MessageTypeBroadcast,
		Identity: fftypes.Identity{
			Key:    "0x12345",
			Author: "did:firefly:org/abcd",
		},
		Namespace: "ns1",
		Hash:      fftypes.NewRandB32(),
		Created:   fftypes.Now(),
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID:   txid,
				Type: fftypes.TransactionTypeBatchPin,
			},
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{ID: msgID1}},
				{Header: fftypes.MessageHeader{ID: msgID2}},
			},
		},
		PayloadRef: payloadRef,
		Confirmed:  fftypes.Now(),
	}

	// Rejects hash change
	err = s.UpsertBatch(context.Background(), batchUpdated, false)
	assert.Equal(t, database.HashMismatch, err)

	err = s.UpsertBatch(context.Background(), batchUpdated, true)
	assert.NoError(t, err)

	// Check we get the exact same message back - note the removal of one of the batch elements
	batchRead, err = s.GetBatchByID(ctx, batchID)
	assert.NoError(t, err)
	batchJson, _ = json.Marshal(&batchUpdated)
	batchReadJson, _ = json.Marshal(&batchRead)
	assert.Equal(t, string(batchJson), string(batchReadJson))

	// Query back the batch
	fb := database.BatchQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("id", batchUpdated.ID.String()),
		fb.Eq("namespace", batchUpdated.Namespace),
		fb.Eq("author", batchUpdated.Author),
		fb.Gt("created", "0"),
		fb.Gt("confirmed", "0"),
	)
	batches, _, err := s.GetBatches(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(batches))
	batchReadJson, _ = json.Marshal(batches[0])
	assert.Equal(t, string(batchJson), string(batchReadJson))

	// Negative test on filter
	filter = fb.And(
		fb.Eq("id", batchUpdated.ID.String()),
		fb.Eq("created", "0"),
	)
	batches, _, err = s.GetBatches(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(batches))

	// Update
	author2 := "0x222222"
	up := database.BatchQueryFactory.NewUpdate(ctx).Set("author", author2)
	err = s.UpdateBatch(ctx, batchID, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("id", batchUpdated.ID.String()),
		fb.Eq("author", author2),
	)
	batches, res, err := s.GetBatches(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(batches))
	assert.Equal(t, int64(1), *res.TotalCount)

	s.callbacks.AssertExpectations(t)
}

func TestUpsertBatchFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertBatch(context.Background(), &fftypes.Batch{}, true)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBatchFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	batchID := fftypes.NewUUID()
	err := s.UpsertBatch(context.Background(), &fftypes.Batch{ID: batchID}, true)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBatchFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	batchID := fftypes.NewUUID()
	err := s.UpsertBatch(context.Background(), &fftypes.Batch{ID: batchID}, true)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBatchFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	batchID := fftypes.NewUUID()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(batchID.String()))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertBatch(context.Background(), &fftypes.Batch{ID: batchID}, true)
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBatchFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	batchID := fftypes.NewUUID()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertBatch(context.Background(), &fftypes.Batch{ID: batchID}, true)
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBatchByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	batchID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetBatchByID(context.Background(), batchID)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBatchByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	batchID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetBatchByID(context.Background(), batchID)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBatchByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	batchID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetBatchByID(context.Background(), batchID)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBatchesQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.BatchQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetBatches(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBatchesBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.BatchQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	_, _, err := s.GetBatches(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestGetBatchesReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.BatchQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetBatches(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchUpdateBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.BatchQueryFactory.NewUpdate(context.Background()).Set("id", "anything")
	err := s.UpdateBatch(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err)
}

func TestBatchUpdateBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	u := database.BatchQueryFactory.NewUpdate(context.Background()).Set("id", map[bool]bool{true: false})
	err := s.UpdateBatch(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestBatchUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.BatchQueryFactory.NewUpdate(context.Background()).Set("id", fftypes.NewUUID())
	err := s.UpdateBatch(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err)
}
