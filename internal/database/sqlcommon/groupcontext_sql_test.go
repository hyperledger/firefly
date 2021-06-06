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

func TestGroupContextsE2EWithDB(t *testing.T) {
	log.SetLevel("trace")

	s := newQLTestProvider(t)
	defer s.Close()
	ctx := context.Background()

	// Create a new groupcontext entry
	groupcontextZero := &fftypes.GroupContext{
		Hash:  fftypes.NewRandB32(),
		Group: fftypes.NewUUID(),
		Topic: "topic12345",
	}
	err := s.UpsertGroupContextNextNonce(ctx, groupcontextZero)
	assert.NoError(t, err)

	// Check we get the exact same groupcontext back
	groupcontextRead, err := s.GetGroupContext(ctx, groupcontextZero.Hash)
	assert.NoError(t, err)
	assert.NotNil(t, groupcontextRead)
	groupcontextJson, _ := json.Marshal(&groupcontextZero)
	groupcontextReadJson, _ := json.Marshal(&groupcontextRead)
	assert.Equal(t, string(groupcontextJson), string(groupcontextReadJson))

	// Update the groupcontext (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	var groupcontextUpdated fftypes.GroupContext
	groupcontextUpdated = *groupcontextZero

	// Increment a couple of times
	err = s.UpsertGroupContextNextNonce(context.Background(), &groupcontextUpdated)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), groupcontextUpdated.Nonce)
	err = s.UpsertGroupContextNextNonce(context.Background(), &groupcontextUpdated)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), groupcontextUpdated.Nonce)

	// Check we get the exact same data back
	groupcontextRead, err = s.GetGroupContext(ctx, groupcontextUpdated.Hash)
	assert.NoError(t, err)
	groupcontextJson, _ = json.Marshal(&groupcontextUpdated)
	groupcontextReadJson, _ = json.Marshal(&groupcontextRead)
	assert.Equal(t, string(groupcontextJson), string(groupcontextReadJson))

	// Query back the groupcontext
	fb := database.GroupContextQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("hash", groupcontextUpdated.Hash),
		fb.Eq("nonce", groupcontextUpdated.Nonce),
		fb.Eq("group", groupcontextUpdated.Group),
		fb.Eq("topic", groupcontextUpdated.Topic),
	)
	groupcontextRes, err := s.GetGroupContexts(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(groupcontextRes))
	groupcontextReadJson, _ = json.Marshal(groupcontextRes[0])
	assert.Equal(t, string(groupcontextJson), string(groupcontextReadJson))

	// Test delete
	err = s.DeleteGroupContext(ctx, groupcontextUpdated.Hash)
	assert.NoError(t, err)
	groupcontexts, err := s.GetGroupContexts(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(groupcontexts))

}

func TestUpsertGroupContextFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertGroupContextNextNonce(context.Background(), &fftypes.GroupContext{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertGroupContextFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertGroupContextNextNonce(context.Background(), &fftypes.GroupContext{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertGroupContextFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertGroupContextNextNonce(context.Background(), &fftypes.GroupContext{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertGroupContextFailScan(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}).AddRow())
	mock.ExpectRollback()
	err := s.UpsertGroupContextNextNonce(context.Background(), &fftypes.GroupContext{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertGroupContextFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"nonce", "sequence"}).AddRow(int64(12345), int64(11111)))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertGroupContextNextNonce(context.Background(), &fftypes.GroupContext{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGroupContextSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetGroupContext(context.Background(), fftypes.NewRandB32())
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGroupContextNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash", "nonce", "group_id", "topic"}))
	msg, err := s.GetGroupContext(context.Background(), fftypes.NewRandB32())
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGroupContextScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash"}).AddRow("only one"))
	_, err := s.GetGroupContext(context.Background(), fftypes.NewRandB32())
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGroupContextQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.GroupContextQueryFactory.NewFilter(context.Background()).Eq("hash", "")
	_, err := s.GetGroupContexts(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGroupContextBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.GroupContextQueryFactory.NewFilter(context.Background()).Eq("hash", map[bool]bool{true: false})
	_, err := s.GetGroupContexts(context.Background(), f)
	assert.Regexp(t, "FF10149.*hash", err)
}

func TestGetGroupContextReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash"}).AddRow("only one"))
	f := database.GroupContextQueryFactory.NewFilter(context.Background()).Eq("topic", "")
	_, err := s.GetGroupContexts(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupContextDeleteBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteGroupContext(context.Background(), fftypes.NewRandB32())
	assert.Regexp(t, "FF10114", err)
}

func TestGroupContextDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.DeleteGroupContext(context.Background(), fftypes.NewRandB32())
	assert.Regexp(t, "FF10118", err)
}
