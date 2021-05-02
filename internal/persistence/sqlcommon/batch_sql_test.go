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
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/persistence"
	"github.com/stretchr/testify/assert"
)

func TestBatch2EWithDB(t *testing.T) {

	s := &SQLCommon{}
	ctx := context.Background()
	InitSQLCommon(ctx, s, ensureTestDB(t), nil)

	// Create a new batch entry
	batchId := uuid.New()
	msgId1 := uuid.New()
	randB32 := fftypes.NewRandB32()
	batch := &fftypes.Batch{
		ID:     &batchId,
		Type:   fftypes.BatchTypeBroadcast,
		Author: "0x12345",
		Hash:   &randB32,
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.MessageRefsOnly{
				{MessageBase: fftypes.MessageBase{ID: &msgId1}},
			},
		},
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeNone,
		},
	}
	err := s.UpsertBatch(ctx, batch)
	assert.NoError(t, err)

	// Check we get the exact same batch back
	batchRead, err := s.GetBatchById(ctx, &batchId)
	assert.NoError(t, err)
	assert.NotNil(t, batchRead)
	batchJson, _ := json.Marshal(&batch)
	batchReadJson, _ := json.Marshal(&batchRead)
	assert.Equal(t, string(batchJson), string(batchReadJson))

	// Update the batch (this is testing what's possible at the persistence layer,
	// and does not account for the verification that happens at the higher level)
	txid := uuid.New()
	msgId2 := uuid.New()
	batchUpdated := &fftypes.Batch{
		ID:        &batchId,
		Type:      fftypes.BatchTypeBroadcast,
		Author:    "0x12345",
		Namespace: "ns1",
		Hash:      &randB32,
		Created:   time.Now().UnixNano(),
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.MessageRefsOnly{
				{MessageBase: fftypes.MessageBase{ID: &msgId1}},
				{MessageBase: fftypes.MessageBase{ID: &msgId2}},
			},
		},
		TX: fftypes.TransactionRef{
			ID:   &txid,
			Type: fftypes.TransactionTypePin,
		},
		Confirmed: time.Now().UnixNano(),
	}
	err = s.UpsertBatch(context.Background(), batchUpdated)
	assert.NoError(t, err)

	// Check we get the exact same message back - note the removal of one of the batch elements
	batchRead, err = s.GetBatchById(ctx, &batchId)
	assert.NoError(t, err)
	batchJson, _ = json.Marshal(&batchUpdated)
	batchReadJson, _ = json.Marshal(&batchRead)
	assert.Equal(t, string(batchJson), string(batchReadJson))

	// Query back the batch
	filter := &persistence.BatchFilter{
		IDEquals:        batchUpdated.ID,
		NamespaceEquals: batchUpdated.Namespace,
		AuthorEquals:    batchUpdated.Author,
		CreatedAfter:    1,
		ConfirmedAfter:  1,
	}
	batches, err := s.GetBatches(ctx, 0, 1, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(batches))
	batchReadJson, _ = json.Marshal(batches[0])
	assert.Equal(t, string(batchJson), string(batchReadJson))

	// Negative test on filter
	filter.ConfrimedOnly = false
	filter.UnconfrimedOnly = true
	filter.ConfirmedAfter = 0
	batches, err = s.GetBatches(ctx, 0, 1, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(batches))
}

func TestUpsertBatchFailBegin(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertBatch(context.Background(), &fftypes.Batch{})
	assert.Regexp(t, "FF10114", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBatchFailSelect(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	batchId := uuid.New()
	err := s.UpsertBatch(context.Background(), &fftypes.Batch{ID: &batchId})
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBatchFailInsert(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	batchId := uuid.New()
	err := s.UpsertBatch(context.Background(), &fftypes.Batch{ID: &batchId})
	assert.Regexp(t, "FF10116", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBatchFailUpdate(t *testing.T) {
	s, mock := getMockDB()
	batchId := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(batchId.String()))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertBatch(context.Background(), &fftypes.Batch{ID: &batchId})
	assert.Regexp(t, "FF10117", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBatchFailCommit(t *testing.T) {
	s, mock := getMockDB()
	batchId := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertBatch(context.Background(), &fftypes.Batch{ID: &batchId})
	assert.Regexp(t, "FF10119", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBatchByIdSelectFail(t *testing.T) {
	s, mock := getMockDB()
	batchId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetBatchById(context.Background(), &batchId)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBatchByIdNotFound(t *testing.T) {
	s, mock := getMockDB()
	batchId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetBatchById(context.Background(), &batchId)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBatchByIdScanFail(t *testing.T) {
	s, mock := getMockDB()
	batchId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetBatchById(context.Background(), &batchId)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBatchesQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetBatches(context.Background(), 0, 1, &persistence.BatchFilter{ConfrimedOnly: true})
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGettBatchesReadMessageFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetBatches(context.Background(), 0, 1, &persistence.BatchFilter{})
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}
