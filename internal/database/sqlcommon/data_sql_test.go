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

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/stretchr/testify/assert"
)

func TestDataE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s := &SQLCommon{}
	ctx := context.Background()
	InitSQLCommon(ctx, s, ensureTestDB(t), nil, &database.Capabilities{}, testSQLOptions())

	// Create a new data entry
	dataId := uuid.New()
	val := fftypes.JSONObject{
		"some": "data",
		"with": map[string]interface{}{
			"nesting": 12345,
		},
	}
	data := &fftypes.Data{
		ID:        &dataId,
		Validator: fftypes.ValidatorTypeBLOB,
		Namespace: "ns1",
		Hash:      fftypes.NewRandB32(),
		Created:   fftypes.Now(),
		Value:     val,
	}
	err := s.UpsertData(ctx, data, true)
	assert.NoError(t, err)

	// Check we get the exact same data back
	dataRead, err := s.GetDataById(ctx, &dataId)
	assert.NoError(t, err)
	assert.NotNil(t, dataRead)
	dataJson, _ := json.Marshal(&data)
	dataReadJson, _ := json.Marshal(&dataRead)
	assert.Equal(t, string(dataJson), string(dataReadJson))

	// Update the data (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	val2 := fftypes.JSONObject{
		"another": "set",
		"of": map[string]interface{}{
			"data": 12345,
		},
	}
	dataUpdated := &fftypes.Data{
		ID:        &dataId,
		Validator: fftypes.ValidatorTypeJSON,
		Namespace: "ns2",
		Definition: &fftypes.DataDefinitionRef{
			Name:    "customer",
			Version: "0.0.1",
		},
		Hash:    fftypes.NewRandB32(),
		Created: fftypes.Now(),
		Value:   val2,
	}

	// Check disallows hash update
	err = s.UpsertData(context.Background(), dataUpdated, false)
	assert.Equal(t, database.HashMismatch, err)

	err = s.UpsertData(context.Background(), dataUpdated, true)
	assert.NoError(t, err)

	// Check we get the exact same message back - note the removal of one of the data elements
	dataRead, err = s.GetDataById(ctx, &dataId)
	assert.NoError(t, err)
	dataJson, _ = json.Marshal(&dataUpdated)
	dataReadJson, _ = json.Marshal(&dataRead)
	assert.Equal(t, string(dataJson), string(dataReadJson))

	// Query back the data
	fb := database.DataQueryFactory.NewFilter(ctx, 0)
	filter := fb.And(
		fb.Eq("id", dataUpdated.ID.String()),
		fb.Eq("namespace", dataUpdated.Namespace),
		fb.Eq("validator", string(dataUpdated.Validator)),
		fb.Eq("definition.name", dataUpdated.Definition.Name),
		fb.Eq("definition.version", dataUpdated.Definition.Version),
		fb.Eq("hash", dataUpdated.Hash),
		fb.Gt("created", 0),
	)
	dataRes, err := s.GetData(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(dataRes))
	dataReadJson, _ = json.Marshal(dataRes[0])
	assert.Equal(t, string(dataJson), string(dataReadJson))

	// Update
	v2 := "2.0.0"
	up := database.DataQueryFactory.NewUpdate(ctx).Set("definition.version", v2)
	err = s.UpdateData(ctx, &dataId, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("id", dataUpdated.ID.String()),
		fb.Eq("definition.version", v2),
	)
	dataRes, err = s.GetData(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(dataRes))
}

func TestUpsertDataFailBegin(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertData(context.Background(), &fftypes.Data{}, true)
	assert.Regexp(t, "FF10114", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDataFailSelect(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	dataId := uuid.New()
	err := s.UpsertData(context.Background(), &fftypes.Data{ID: &dataId}, true)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDataFailInsert(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	dataId := uuid.New()
	err := s.UpsertData(context.Background(), &fftypes.Data{ID: &dataId}, true)
	assert.Regexp(t, "FF10116", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDataFailUpdate(t *testing.T) {
	s, mock := getMockDB()
	dataId := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(dataId.String()))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertData(context.Background(), &fftypes.Data{ID: &dataId}, true)
	assert.Regexp(t, "FF10117", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDataFailCommit(t *testing.T) {
	s, mock := getMockDB()
	dataId := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertData(context.Background(), &fftypes.Data{ID: &dataId}, true)
	assert.Regexp(t, "FF10119", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataByIdSelectFail(t *testing.T) {
	s, mock := getMockDB()
	dataId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetDataById(context.Background(), &dataId)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataByIdNotFound(t *testing.T) {
	s, mock := getMockDB()
	dataId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetDataById(context.Background(), &dataId)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataByIdScanFail(t *testing.T) {
	s, mock := getMockDB()
	dataId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetDataById(context.Background(), &dataId)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.DataQueryFactory.NewFilter(context.Background(), 0).Eq("id", "")
	_, err := s.GetData(context.Background(), f)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataBuildQueryFail(t *testing.T) {
	s, _ := getMockDB()
	f := database.DataQueryFactory.NewFilter(context.Background(), 0).Eq("id", map[bool]bool{true: false})
	_, err := s.GetData(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestGetDataReadMessageFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.DataQueryFactory.NewFilter(context.Background(), 0).Eq("id", "")
	_, err := s.GetData(context.Background(), f)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDataUpdateBeginFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.DataQueryFactory.NewUpdate(context.Background()).Set("id", "anything")
	err := s.UpdateData(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err.Error())
}

func TestDataUpdateBuildQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	u := database.DataQueryFactory.NewUpdate(context.Background()).Set("id", map[bool]bool{true: false})
	err := s.UpdateData(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestDataUpdateFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.DataQueryFactory.NewUpdate(context.Background()).Set("id", fftypes.NewUUID())
	err := s.UpdateData(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err.Error())
}
