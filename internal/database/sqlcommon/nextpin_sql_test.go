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
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestNextPinsE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s := newQLTestProvider(t)
	defer s.Close()
	ctx := context.Background()

	// Create a new nextpin entry
	nextpin := &fftypes.NextPin{
		Context:  fftypes.NewRandB32(),
		Identity: "0x12345",
		Hash:     fftypes.NewRandB32(),
		Nonce:    int64(12345),
	}
	err := s.InsertNextPin(ctx, nextpin)
	assert.NoError(t, err)

	// Check we get the exact same nextpin back
	nextpinRead, err := s.GetNextPinByContextAndIdentity(ctx, nextpin.Context, nextpin.Identity)
	assert.NoError(t, err)
	assert.NotNil(t, nextpinRead)
	nextpinJson, _ := json.Marshal(&nextpin)
	nextpinReadJson, _ := json.Marshal(&nextpinRead)
	assert.Equal(t, string(nextpinJson), string(nextpinReadJson))

	// Attempt with wrong ID
	var nextpinUpdated fftypes.NextPin
	nextpinUpdated = *nextpin
	nextpinUpdated.Nonce = 1111111
	nextpinUpdated.Hash = fftypes.NewRandB32()
	err = s.UpdateNextPin(context.Background(), nextpin.Sequence, database.NextPinQueryFactory.NewUpdate(ctx).
		Set("hash", nextpinUpdated.Hash).
		Set("nonce", nextpinUpdated.Nonce),
	)

	// Check we get the exact same data back - note the removal of one of the nextpin elements
	nextpinRead, err = s.GetNextPinByHash(ctx, nextpinUpdated.Hash)
	assert.NoError(t, err)
	nextpinJson, _ = json.Marshal(&nextpinUpdated)
	nextpinReadJson, _ = json.Marshal(&nextpinRead)
	assert.Equal(t, string(nextpinJson), string(nextpinReadJson))

	// Query back the nextpin
	fb := database.NextPinQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("context", nextpinUpdated.Context),
		fb.Eq("hash", nextpinUpdated.Hash),
		fb.Eq("identity", nextpinUpdated.Identity),
		fb.Eq("nonce", nextpinUpdated.Nonce),
	)
	nextpinRes, err := s.GetNextPins(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(nextpinRes))
	nextpinReadJson, _ = json.Marshal(nextpinRes[0])
	assert.Equal(t, string(nextpinJson), string(nextpinReadJson))

	// Test delete
	err = s.DeleteNextPin(ctx, nextpinUpdated.Sequence)
	assert.NoError(t, err)
	nextpins, err := s.GetNextPins(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(nextpins))

}

func TestUpsertNextPinFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertNextPin(context.Background(), &fftypes.NextPin{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNextPinFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.InsertNextPin(context.Background(), &fftypes.NextPin{Context: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNextPinFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertNextPin(context.Background(), &fftypes.NextPin{Context: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNextPinByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetNextPinByContextAndIdentity(context.Background(), fftypes.NewRandB32(), "0x12345")
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNextPinByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	msg, err := s.GetNextPinByContextAndIdentity(context.Background(), fftypes.NewRandB32(), "0x12345")
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNextPinByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"context"}).AddRow("only one"))
	_, err := s.GetNextPinByContextAndIdentity(context.Background(), fftypes.NewRandB32(), "0x12345")
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNextPinQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.NextPinQueryFactory.NewFilter(context.Background()).Eq("context", "")
	_, err := s.GetNextPins(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNextPinBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.NextPinQueryFactory.NewFilter(context.Background()).Eq("context", map[bool]bool{true: false})
	_, err := s.GetNextPins(context.Background(), f)
	assert.Regexp(t, "FF10149.*type", err)
}

func TestGetNextPinReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"context"}).AddRow("only one"))
	f := database.NextPinQueryFactory.NewFilter(context.Background()).Eq("context", "")
	_, err := s.GetNextPins(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNextPinUpdateBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.NextPinQueryFactory.NewUpdate(context.Background()).Set("context", "anything")
	err := s.UpdateNextPin(context.Background(), 12345, u)
	assert.Regexp(t, "FF10114", err)
}

func TestNextPinUpdateBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	u := database.NextPinQueryFactory.NewUpdate(context.Background()).Set("context", map[bool]bool{true: false})
	err := s.UpdateNextPin(context.Background(), 12345, u)
	assert.Regexp(t, "FF10149.*context", err)
}

func TestNextPinUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.NextPinQueryFactory.NewUpdate(context.Background()).Set("context", fftypes.NewRandB32())
	err := s.UpdateNextPin(context.Background(), 12345, u)
	assert.Regexp(t, "FF10117", err)
}

func TestNextPinDeleteBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteNextPin(context.Background(), 12345)
	assert.Regexp(t, "FF10114", err)
}

func TestNextPinDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.DeleteNextPin(context.Background(), 12345)
	assert.Regexp(t, "FF10118", err)
}
