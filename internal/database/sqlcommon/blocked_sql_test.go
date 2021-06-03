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

func TestBlockedsE2EWithDB(t *testing.T) {
	log.SetLevel("trace")

	s := newQLTestProvider(t)
	defer s.Close()
	ctx := context.Background()

	// Create a new blocked entry
	groupID := fftypes.NewUUID()
	blocked := &fftypes.Blocked{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Context:   "context1",
		Group:     groupID,
		Message:   fftypes.NewUUID(),
		Created:   fftypes.Now(),
	}
	err := s.UpsertBlocked(ctx, blocked, true)
	assert.NoError(t, err)

	// Check we get the exact same blocked back
	blockedRead, err := s.GetBlockedByContext(ctx, blocked.Namespace, blocked.Context, blocked.Group)
	assert.NoError(t, err)
	assert.NotNil(t, blockedRead)
	blockedJson, _ := json.Marshal(&blocked)
	blockedReadJson, _ := json.Marshal(&blockedRead)
	assert.Equal(t, string(blockedJson), string(blockedReadJson))

	// Update the blocked (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	blockedUpdated := &fftypes.Blocked{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Context:   "context1",
		Group:     groupID,
		Message:   fftypes.NewUUID(),
		Created:   fftypes.Now(),
	}

	// Attempt with wrong ID
	err = s.UpsertBlocked(context.Background(), blockedUpdated, true)
	assert.Equal(t, err, database.IDMismatch)

	// Remove ID for upsert and retry
	blockedUpdated.ID = nil
	err = s.UpsertBlocked(context.Background(), blockedUpdated, true)
	assert.NoError(t, err)

	// Check we get the exact same data back - note the removal of one of the blocked elements
	blockedRead, err = s.GetBlockedByContext(ctx, blocked.Namespace, blocked.Context, blocked.Group)
	assert.NoError(t, err)
	blockedJson, _ = json.Marshal(&blockedUpdated)
	blockedReadJson, _ = json.Marshal(&blockedRead)
	assert.Equal(t, string(blockedJson), string(blockedReadJson))

	// Query back the blocked
	fb := database.BlockedQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("context", string(blockedUpdated.Context)),
		fb.Eq("namespace", blockedUpdated.Namespace),
		fb.Gt("created", 0),
	)
	blockedRes, err := s.GetBlocked(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(blockedRes))
	blockedReadJson, _ = json.Marshal(blockedRes[0])
	assert.Equal(t, string(blockedJson), string(blockedReadJson))

	// Update
	newUUID := fftypes.NewUUID()
	up := database.BlockedQueryFactory.NewUpdate(ctx).Set("message", newUUID)
	err = s.UpdateBlocked(ctx, blockedUpdated.ID, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("context", blockedUpdated.Context),
	)
	blockeds, err := s.GetBlocked(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(blockeds))

	// Test delete
	err = s.DeleteBlocked(ctx, blockedUpdated.ID)
	assert.NoError(t, err)
	blockeds, err = s.GetBlocked(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(blockeds))

}

func TestUpsertBlockedFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertBlocked(context.Background(), &fftypes.Blocked{}, true)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBlockedFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertBlocked(context.Background(), &fftypes.Blocked{Context: "context1"}, true)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBlockedFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertBlocked(context.Background(), &fftypes.Blocked{Context: "context1"}, true)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBlockedFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"context", "namespace", "context"}).
		AddRow("ns1", "context1", nil))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertBlocked(context.Background(), &fftypes.Blocked{Context: "context1"}, true)
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBlockedFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"context", "namespace", "context"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertBlocked(context.Background(), &fftypes.Blocked{Context: "context1"}, true)
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlockedByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetBlockedByContext(context.Background(), "ns1", "context1", nil)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlockedByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"context", "namespace", "context"}))
	msg, err := s.GetBlockedByContext(context.Background(), "ns1", "context1", nil)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlockedByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"context"}).AddRow("only one"))
	_, err := s.GetBlockedByContext(context.Background(), "ns1", "context1", nil)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlockedQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.BlockedQueryFactory.NewFilter(context.Background()).Eq("context", "")
	_, err := s.GetBlocked(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlockedBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.BlockedQueryFactory.NewFilter(context.Background()).Eq("context", map[bool]bool{true: false})
	_, err := s.GetBlocked(context.Background(), f)
	assert.Regexp(t, "FF10149.*type", err)
}

func TestGetBlockedReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"context"}).AddRow("only one"))
	f := database.BlockedQueryFactory.NewFilter(context.Background()).Eq("context", "")
	_, err := s.GetBlocked(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBlockedUpdateBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.BlockedQueryFactory.NewUpdate(context.Background()).Set("context", "anything")
	err := s.UpdateBlocked(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err)
}

func TestBlockedUpdateBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	u := database.BlockedQueryFactory.NewUpdate(context.Background()).Set("context", map[bool]bool{true: false})
	err := s.UpdateBlocked(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10149.*context", err)
}

func TestBlockedUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.BlockedQueryFactory.NewUpdate(context.Background()).Set("context", fftypes.NewUUID())
	err := s.UpdateBlocked(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err)
}

func TestBlockedDeleteBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteBlocked(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10114", err)
}

func TestBlockedDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.DeleteBlocked(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10118", err)
}
