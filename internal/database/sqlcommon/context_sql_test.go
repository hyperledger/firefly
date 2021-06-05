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

func TestContextsE2EWithDB(t *testing.T) {
	log.SetLevel("trace")

	s := newQLTestProvider(t)
	defer s.Close()
	ctx := context.Background()

	// Create a new context entry
	contextZero := &fftypes.Context{
		Hash:  fftypes.NewRandB32(),
		Group: fftypes.NewUUID(),
		Topic: "topic12345",
	}
	err := s.UpsertContextNextNonce(ctx, contextZero)
	assert.NoError(t, err)

	// Check we get the exact same context back
	contextRead, err := s.GetContext(ctx, contextZero.Hash)
	assert.NoError(t, err)
	assert.NotNil(t, contextRead)
	contextJson, _ := json.Marshal(&contextZero)
	contextReadJson, _ := json.Marshal(&contextRead)
	assert.Equal(t, string(contextJson), string(contextReadJson))

	// Update the context (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	var contextUpdated fftypes.Context
	contextUpdated = *contextZero

	// Increment a couple of times
	err = s.UpsertContextNextNonce(context.Background(), &contextUpdated)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), contextUpdated.Nonce)
	err = s.UpsertContextNextNonce(context.Background(), &contextUpdated)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), contextUpdated.Nonce)

	// Check we get the exact same data back
	contextRead, err = s.GetContext(ctx, contextUpdated.Hash)
	assert.NoError(t, err)
	contextJson, _ = json.Marshal(&contextUpdated)
	contextReadJson, _ = json.Marshal(&contextRead)
	assert.Equal(t, string(contextJson), string(contextReadJson))

	// Query back the context
	fb := database.ContextQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("hash", contextUpdated.Hash),
		fb.Eq("nonce", contextUpdated.Nonce),
		fb.Eq("group", contextUpdated.Group),
		fb.Eq("topic", contextUpdated.Topic),
	)
	contextRes, err := s.GetContexts(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(contextRes))
	contextReadJson, _ = json.Marshal(contextRes[0])
	assert.Equal(t, string(contextJson), string(contextReadJson))

	// Test delete
	err = s.DeleteContext(ctx, contextUpdated.Hash)
	assert.NoError(t, err)
	contexts, err := s.GetContexts(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(contexts))

}

func TestUpsertContextFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertContextNextNonce(context.Background(), &fftypes.Context{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertContextFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertContextNextNonce(context.Background(), &fftypes.Context{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertContextFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertContextNextNonce(context.Background(), &fftypes.Context{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertContextFailScan(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}).AddRow())
	mock.ExpectRollback()
	err := s.UpsertContextNextNonce(context.Background(), &fftypes.Context{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertContextFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"nonce", "sequence"}).AddRow(int64(12345), int64(11111)))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertContextNextNonce(context.Background(), &fftypes.Context{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContextSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetContext(context.Background(), fftypes.NewRandB32())
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContextNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash", "nonce", "group_id", "topic"}))
	msg, err := s.GetContext(context.Background(), fftypes.NewRandB32())
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContextScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash"}).AddRow("only one"))
	_, err := s.GetContext(context.Background(), fftypes.NewRandB32())
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContextQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.ContextQueryFactory.NewFilter(context.Background()).Eq("hash", "")
	_, err := s.GetContexts(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContextBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.ContextQueryFactory.NewFilter(context.Background()).Eq("hash", map[bool]bool{true: false})
	_, err := s.GetContexts(context.Background(), f)
	assert.Regexp(t, "FF10149.*hash", err)
}

func TestGetContextReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash"}).AddRow("only one"))
	f := database.ContextQueryFactory.NewFilter(context.Background()).Eq("topic", "")
	_, err := s.GetContexts(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContextDeleteBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteContext(context.Background(), fftypes.NewRandB32())
	assert.Regexp(t, "FF10114", err)
}

func TestContextDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.DeleteContext(context.Background(), fftypes.NewRandB32())
	assert.Regexp(t, "FF10118", err)
}
