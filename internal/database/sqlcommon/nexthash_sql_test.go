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
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestNextHashesE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s := newQLTestProvider(t)
	defer s.Close()
	ctx := context.Background()

	// Create a new nexthash entry
	nexthash := &fftypes.NextHash{
		Context:  fftypes.NewRandB32(),
		Identity: "0x12345",
		Hash:     fftypes.NewRandB32(),
		Nonce:    int64(12345),
	}
	err := s.InsertNextHash(ctx, nexthash)
	assert.NoError(t, err)

	// Check we get the exact same nexthash back
	nexthashRead, err := s.GetNextHashByContextAndIdentity(ctx, nexthash.Context, nexthash.Identity)
	assert.NoError(t, err)
	assert.NotNil(t, nexthashRead)
	nexthashJson, _ := json.Marshal(&nexthash)
	nexthashReadJson, _ := json.Marshal(&nexthashRead)
	assert.Equal(t, string(nexthashJson), string(nexthashReadJson))

	// Attempt with wrong ID
	var nexthashUpdated fftypes.NextHash
	nexthashUpdated = *nexthash
	nexthashUpdated.Nonce = 1111111
	nexthashUpdated.Hash = fftypes.NewRandB32()
	err = s.UpdateNextHash(context.Background(), nexthash.Sequence, database.NextHashQueryFactory.NewUpdate(ctx).
		Set("hash", nexthashUpdated.Hash).
		Set("nonce", nexthashUpdated.Nonce),
	)

	// Check we get the exact same data back - note the removal of one of the nexthash elements
	nexthashRead, err = s.GetNextHashByHash(ctx, nexthashUpdated.Hash)
	assert.NoError(t, err)
	nexthashJson, _ = json.Marshal(&nexthashUpdated)
	nexthashReadJson, _ = json.Marshal(&nexthashRead)
	assert.Equal(t, string(nexthashJson), string(nexthashReadJson))

	// Query back the nexthash
	fb := database.NextHashQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("context", nexthashUpdated.Context),
		fb.Eq("hash", nexthashUpdated.Hash),
		fb.Eq("identity", nexthashUpdated.Identity),
		fb.Eq("nonce", nexthashUpdated.Nonce),
	)
	nexthashRes, err := s.GetNextHashes(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(nexthashRes))
	nexthashReadJson, _ = json.Marshal(nexthashRes[0])
	assert.Equal(t, string(nexthashJson), string(nexthashReadJson))

	// Test delete
	err = s.DeleteNextHash(ctx, nexthashUpdated.Sequence)
	assert.NoError(t, err)
	nexthashes, err := s.GetNextHashes(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(nexthashes))

}

func TestUpsertNextHashFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertNextHash(context.Background(), &fftypes.NextHash{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNextHashFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.InsertNextHash(context.Background(), &fftypes.NextHash{Context: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNextHashFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertNextHash(context.Background(), &fftypes.NextHash{Context: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNextHashByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetNextHashByContextAndIdentity(context.Background(), fftypes.NewRandB32(), "0x12345")
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNextHashByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	msg, err := s.GetNextHashByContextAndIdentity(context.Background(), fftypes.NewRandB32(), "0x12345")
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNextHashByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"context"}).AddRow("only one"))
	_, err := s.GetNextHashByContextAndIdentity(context.Background(), fftypes.NewRandB32(), "0x12345")
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNextHashQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.NextHashQueryFactory.NewFilter(context.Background()).Eq("context", "")
	_, err := s.GetNextHashes(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNextHashBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.NextHashQueryFactory.NewFilter(context.Background()).Eq("context", map[bool]bool{true: false})
	_, err := s.GetNextHashes(context.Background(), f)
	assert.Regexp(t, "FF10149.*type", err)
}

func TestGetNextHashReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"context"}).AddRow("only one"))
	f := database.NextHashQueryFactory.NewFilter(context.Background()).Eq("context", "")
	_, err := s.GetNextHashes(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNextHashUpdateBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.NextHashQueryFactory.NewUpdate(context.Background()).Set("context", "anything")
	err := s.UpdateNextHash(context.Background(), 12345, u)
	assert.Regexp(t, "FF10114", err)
}

func TestNextHashUpdateBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	u := database.NextHashQueryFactory.NewUpdate(context.Background()).Set("context", map[bool]bool{true: false})
	err := s.UpdateNextHash(context.Background(), 12345, u)
	assert.Regexp(t, "FF10149.*context", err)
}

func TestNextHashUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.NextHashQueryFactory.NewUpdate(context.Background()).Set("context", fftypes.NewUUID())
	err := s.UpdateNextHash(context.Background(), 12345, u)
	assert.Regexp(t, "FF10117", err)
}

func TestNextHashDeleteBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteNextHash(context.Background(), 12345)
	assert.Regexp(t, "FF10114", err)
}

func TestNextHashDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.DeleteNextHash(context.Background(), 12345)
	assert.Regexp(t, "FF10118", err)
}
