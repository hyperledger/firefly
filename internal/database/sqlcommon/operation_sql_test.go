// Copyright © 2022 Kaleido, Inc.
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
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
)

func TestOperationE2EWithDB(t *testing.T) {

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new operation entry
	operationID := fftypes.NewUUID()
	operation := &core.Operation{
		ID:          operationID,
		Namespace:   "ns1",
		Type:        core.OpTypeBlockchainPinBatch,
		Transaction: fftypes.NewUUID(),
		Status:      core.OpStatusFailed,
		Plugin:      "ethereum",
		Error:       "pop",
		Input:       fftypes.JSONObject{"some": "input-info"},
		Output:      fftypes.JSONObject{"some": "output-info"},
		Created:     fftypes.Now(),
		Updated:     fftypes.Now(),
	}
	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionOperations, core.ChangeEventTypeCreated, "ns1", operationID).Return()
	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionOperations, core.ChangeEventTypeUpdated, "ns1", operationID).Return()
	hookCalled := false
	err := s.InsertOperation(ctx, operation, func() {
		hookCalled = true
	})
	assert.NoError(t, err)
	assert.True(t, hookCalled)

	// Query back the operation (by ID)
	operationRead, err := s.GetOperationByID(ctx, "ns1", operationID)
	assert.NoError(t, err)
	operationJson, _ := json.Marshal(operation)
	operationReadJson, _ := json.Marshal(operationRead)
	assert.Equal(t, string(operationJson), string(operationReadJson))

	// Query back the operation (by query filter)
	fb := database.OperationQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("id", operation.ID.String()),
		fb.Eq("tx", operation.Transaction),
		fb.Eq("type", operation.Type),
		fb.Eq("status", operation.Status),
		fb.Eq("error", operation.Error),
		fb.Eq("plugin", operation.Plugin),
		fb.Gt("created", 0),
		fb.Gt("updated", 0),
	)
	operations, res, err := s.GetOperations(ctx, "ns1", filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(operations))
	assert.Equal(t, int64(1), *res.TotalCount)
	operationJson, _ = json.Marshal(operation)
	operationReadJson, _ = json.Marshal(operations[0])
	assert.Equal(t, string(operationJson), string(operationReadJson))

	// Negative test on filter
	filter = fb.And(
		fb.Eq("id", operation.ID.String()),
		fb.Eq("updated", "0"),
	)
	operations, _, err = s.GetOperations(ctx, "ns1", filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(operations))

	// Update
	errMsg := "FF10143"
	update := database.OperationQueryFactory.NewUpdate(ctx).S()
	update.Set("status", core.OpStatusFailed)
	update.Set("error", errMsg)
	err = s.UpdateOperation(ctx, operation.Namespace, operation.ID, update)
	assert.NoError(t, err)

	// Update not found is 404
	err = s.UpdateOperation(ctx, operation.Namespace, fftypes.NewUUID(), update)
	assert.Regexp(t, "FF10143", err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("id", operation.ID.String()),
		fb.Eq("status", core.OpStatusFailed),
		fb.Eq("error", "FF10143"),
	)
	operations, _, err = s.GetOperations(ctx, "ns1", filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(operations))

	s.callbacks.AssertExpectations(t)
}

func TestInsertOperationFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertOperation(context.Background(), &core.Operation{})
	assert.Regexp(t, "FF00175", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertOperationFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	operationID := fftypes.NewUUID()
	err := s.InsertOperation(context.Background(), &core.Operation{ID: operationID})
	assert.Regexp(t, "FF00177", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertOperationFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	operationID := fftypes.NewUUID()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertOperation(context.Background(), &core.Operation{ID: operationID})
	assert.Regexp(t, "FF00180", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOperationByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	operationID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetOperationByID(context.Background(), "ns1", operationID)
	assert.Regexp(t, "FF00176", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOperationByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	operationID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetOperationByID(context.Background(), "ns1", operationID)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOperationByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	operationID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetOperationByID(context.Background(), "ns1", operationID)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOperationsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.OperationQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetOperations(context.Background(), "ns1", f)
	assert.Regexp(t, "FF00176", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOperationsBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.OperationQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	_, _, err := s.GetOperations(context.Background(), "ns1", f)
	assert.Regexp(t, "FF00143.*id", err)
}

func TestGettOperationsReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.OperationQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetOperations(context.Background(), "ns1", f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOperationUpdateBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.OperationQueryFactory.NewUpdate(context.Background()).Set("id", fftypes.NewUUID())
	err := s.UpdateOperation(context.Background(), "ns1", fftypes.NewUUID(), u)
	assert.Regexp(t, "FF00175", err)
}

func TestOperationUpdateBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	u := database.OperationQueryFactory.NewUpdate(context.Background()).Set("id", map[bool]bool{true: false})
	err := s.UpdateOperation(context.Background(), "ns1", fftypes.NewUUID(), u)
	assert.Regexp(t, "FF00143.*id", err)
}

func TestOperationUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.OperationQueryFactory.NewUpdate(context.Background()).Set("id", fftypes.NewUUID())
	err := s.UpdateOperation(context.Background(), "ns1", fftypes.NewUUID(), u)
	assert.Regexp(t, "FF00178", err)
}
