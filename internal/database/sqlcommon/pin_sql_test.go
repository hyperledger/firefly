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
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPinsE2EWithDB(t *testing.T) {
	log.SetLevel("trace")

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new pin entry
	pin := &fftypes.Pin{
		Masked:     true,
		Hash:       fftypes.NewRandB32(),
		Batch:      fftypes.NewUUID(),
		Index:      10,
		Created:    fftypes.Now(),
		Signer:     "0x12345",
		Dispatched: false,
	}

	s.callbacks.On("OrderedCollectionEvent", database.CollectionPins, fftypes.ChangeEventTypeCreated, mock.Anything).Return()
	s.callbacks.On("OrderedCollectionEvent", database.CollectionPins, fftypes.ChangeEventTypeDeleted, mock.Anything).Return()

	err := s.UpsertPin(ctx, pin)
	assert.NoError(t, err)

	// Query back the pin
	fb := database.PinQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("masked", pin.Masked),
		fb.Eq("hash", pin.Hash),
		fb.Eq("batch", pin.Batch),
		fb.Gt("created", 0),
	)
	pinRes, res, err := s.GetPins(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pinRes))
	assert.Equal(t, int64(1), *res.TotalCount)

	// Set it dispatched
	err = s.UpdatePins(ctx, database.PinQueryFactory.NewFilter(ctx).Eq("sequence", pin.Sequence), database.PinQueryFactory.NewUpdate(ctx).Set("dispatched", true))
	assert.NoError(t, err)

	// Double insert, checking no error and we keep the dispatched flag
	existingSequence := pin.Sequence
	pin.Sequence = 99999
	err = s.UpsertPin(ctx, pin)
	assert.NoError(t, err)
	pinRes, _, err = s.GetPins(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pinRes)) // we didn't add twice
	assert.Equal(t, existingSequence, pin.Sequence)
	assert.True(t, pin.Dispatched)

	// Test delete
	err = s.DeletePin(ctx, pin.Sequence)
	assert.NoError(t, err)
	p, _, err := s.GetPins(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(p))

	s.callbacks.AssertExpectations(t)
}

func TestUpsertPinFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertPin(context.Background(), &fftypes.Pin{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertPinFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"sequence", "masked", "dispatched"}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertPin(context.Background(), &fftypes.Pin{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertPinFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertPin(context.Background(), &fftypes.Pin{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertPinFailExistingSequenceScan(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(mock.NewRows([]string{"only one"}).AddRow(true))
	mock.ExpectRollback()
	err := s.UpsertPin(context.Background(), &fftypes.Pin{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertPinFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"sequence", "masked", "dispatched"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertPin(context.Background(), &fftypes.Pin{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertPinsBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertPins(context.Background(), []*fftypes.Pin{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
	s.callbacks.AssertExpectations(t)
}

func TestInsertPinsMultiRowOK(t *testing.T) {
	s, mock := newMockProvider().init()
	s.features.MultiRowInsert = true
	s.fakePSQLInsert = true

	pin1 := &fftypes.Pin{Hash: fftypes.NewRandB32()}
	pin2 := &fftypes.Pin{Hash: fftypes.NewRandB32()}
	s.callbacks.On("OrderedCollectionEvent", database.CollectionPins, fftypes.ChangeEventTypeCreated, int64(1001))
	s.callbacks.On("OrderedCollectionEvent", database.CollectionPins, fftypes.ChangeEventTypeCreated, int64(1002))

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT.*").WillReturnRows(sqlmock.NewRows([]string{sequenceColumn}).
		AddRow(int64(1001)).
		AddRow(int64(1002)),
	)
	mock.ExpectCommit()
	err := s.InsertPins(context.Background(), []*fftypes.Pin{pin1, pin2})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
	s.callbacks.AssertExpectations(t)
}

func TestInsertPinsMultiRowFail(t *testing.T) {
	s, mock := newMockProvider().init()
	s.features.MultiRowInsert = true
	s.fakePSQLInsert = true
	pin1 := &fftypes.Pin{Hash: fftypes.NewRandB32()}
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT.*").WillReturnError(fmt.Errorf("pop"))
	err := s.InsertPins(context.Background(), []*fftypes.Pin{pin1})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
	s.callbacks.AssertExpectations(t)
}

func TestInsertPinsSingleRowFail(t *testing.T) {
	s, mock := newMockProvider().init()
	pin1 := &fftypes.Pin{Hash: fftypes.NewRandB32()}
	mock.ExpectBegin()
	mock.ExpectExec("INSERT.*").WillReturnError(fmt.Errorf("pop"))
	err := s.InsertPins(context.Background(), []*fftypes.Pin{pin1})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
	s.callbacks.AssertExpectations(t)
}

func TestGetPinQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.PinQueryFactory.NewFilter(context.Background()).Eq("hash", "")
	_, _, err := s.GetPins(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPinBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.PinQueryFactory.NewFilter(context.Background()).Eq("hash", map[bool]bool{true: false})
	_, _, err := s.GetPins(context.Background(), f)
	assert.Regexp(t, "FF10149.*type", err)
}

func TestGetPinReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"pin"}).AddRow("only one"))
	f := database.PinQueryFactory.NewFilter(context.Background()).Eq("hash", "")
	_, _, err := s.GetPins(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdatePinsBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	ctx := context.Background()
	err := s.UpdatePins(ctx, database.PinQueryFactory.NewFilter(ctx).Eq("sequence", 1), database.PinQueryFactory.NewUpdate(ctx).Set("dispatched", true))
	assert.Regexp(t, "FF10114", err)
}

func TestUpdatePinsUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	ctx := context.Background()
	err := s.UpdatePins(ctx, database.PinQueryFactory.NewFilter(ctx).Eq("sequence", 1), database.PinQueryFactory.NewUpdate(ctx).Set("dispatched", true))
	assert.Regexp(t, "FF10117", err)
}

func TestUpdatePinsBadFilter(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectRollback()
	ctx := context.Background()
	err := s.UpdatePins(ctx, database.PinQueryFactory.NewFilter(ctx).Eq("sequence", 1), database.PinQueryFactory.NewUpdate(ctx).Set("bad", true))
	assert.Regexp(t, "FF10148", err)
}

func TestUpdatePinsBadUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectRollback()
	ctx := context.Background()
	err := s.UpdatePins(ctx, database.PinQueryFactory.NewFilter(ctx).Eq("bad", 1), database.PinQueryFactory.NewUpdate(ctx).Set("dispatched", true))
	assert.Regexp(t, "FF10148", err)
}

func TestPinDeleteBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.DeletePin(context.Background(), 12345)
	assert.Regexp(t, "FF10114", err)
}

func TestPinDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.DeletePin(context.Background(), 12345)
	assert.Regexp(t, "FF10118", err)
}
