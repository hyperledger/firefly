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
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/persistence"
	"github.com/stretchr/testify/assert"
)

func TestDataDefinitionE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s := &SQLCommon{}
	ctx := context.Background()
	InitSQLCommon(ctx, s, ensureTestDB(t), nil, &persistence.Capabilities{}, testSQLOptions())

	// Create a new data definition entry
	dataDefId := uuid.New()
	randB32 := fftypes.NewRandB32()
	val := fftypes.JSONData{
		"some": "dataDef",
		"with": map[string]interface{}{
			"nesting": 12345,
		},
	}
	dataDef := &fftypes.DataDefinition{
		ID:        &dataDefId,
		Validator: fftypes.ValidatorTypeJSON,
		Namespace: "ns1",
		Hash:      randB32,
		Created:   fftypes.NowMillis(),
		Value:     val,
	}
	err := s.UpsertDataDefinition(ctx, dataDef)
	assert.NoError(t, err)

	// Check we get the exact same data definition back
	dataDefRead, err := s.GetDataDefinitionById(ctx, "ns1", &dataDefId)
	assert.NoError(t, err)
	assert.NotNil(t, dataDefRead)
	dataDefJson, _ := json.Marshal(&dataDef)
	dataDefReadJson, _ := json.Marshal(&dataDefRead)
	assert.Equal(t, string(dataDefJson), string(dataDefReadJson))

	// Update the data definition (this is testing what's possible at the persistence layer,
	// and does not account for the verification that happens at the higher level)
	val2 := fftypes.JSONData{
		"another": "set",
		"of": map[string]interface{}{
			"dataDef": 12345,
		},
	}
	dataDefUpdated := &fftypes.DataDefinition{
		ID:        &dataDefId,
		Validator: fftypes.ValidatorTypeJSON,
		Namespace: "ns2",
		Name:      "customer",
		Version:   "0.0.1",
		Hash:      randB32,
		Created:   fftypes.NowMillis(),
		Value:     val2,
	}
	err = s.UpsertDataDefinition(context.Background(), dataDefUpdated)
	assert.NoError(t, err)

	// Check we get the exact same data back - note the removal of one of the dataDef elements
	dataDefRead, err = s.GetDataDefinitionById(ctx, "ns1", &dataDefId)
	assert.NoError(t, err)
	dataDefJson, _ = json.Marshal(&dataDefUpdated)
	dataDefReadJson, _ = json.Marshal(&dataDefRead)
	assert.Equal(t, string(dataDefJson), string(dataDefReadJson))

	// Query back the data
	fb := persistence.DataDefinitionQueryFactory.NewFilter(ctx, 0)
	filter := fb.And(
		fb.Eq("id", dataDefUpdated.ID.String()),
		fb.Eq("namespace", dataDefUpdated.Namespace),
		fb.Eq("validator", string(dataDefUpdated.Validator)),
		fb.Eq("name", dataDefUpdated.Name),
		fb.Eq("version", dataDefUpdated.Version),
		fb.Gt("created", "0"),
	)
	dataDefs, err := s.GetDataDefinitions(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(dataDefs))
	dataDefReadJson, _ = json.Marshal(dataDefs[0])
	assert.Equal(t, string(dataDefJson), string(dataDefReadJson))

	// Update
	v2 := "2.0.0"
	up := persistence.DataDefinitionQueryFactory.NewUpdate(ctx).Set("version", v2)
	err = s.UpdateDataDefinition(ctx, dataDefUpdated.ID, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("id", dataDefUpdated.ID.String()),
		fb.Eq("version", v2),
	)
	dataDefs, err = s.GetDataDefinitions(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(dataDefs))
}

func TestUpsertDataDefinitionFailBegin(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertDataDefinition(context.Background(), &fftypes.DataDefinition{})
	assert.Regexp(t, "FF10114", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDataDefinitionFailSelect(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	dataDefId := uuid.New()
	err := s.UpsertDataDefinition(context.Background(), &fftypes.DataDefinition{ID: &dataDefId})
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDataDefinitionFailInsert(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	dataDefId := uuid.New()
	err := s.UpsertDataDefinition(context.Background(), &fftypes.DataDefinition{ID: &dataDefId})
	assert.Regexp(t, "FF10116", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDataDefinitionFailUpdate(t *testing.T) {
	s, mock := getMockDB()
	dataDefId := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(dataDefId.String()))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertDataDefinition(context.Background(), &fftypes.DataDefinition{ID: &dataDefId})
	assert.Regexp(t, "FF10117", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDataDefinitionFailCommit(t *testing.T) {
	s, mock := getMockDB()
	dataDefId := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertDataDefinition(context.Background(), &fftypes.DataDefinition{ID: &dataDefId})
	assert.Regexp(t, "FF10119", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataDefinitionByIdSelectFail(t *testing.T) {
	s, mock := getMockDB()
	dataDefId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetDataDefinitionById(context.Background(), "ns1", &dataDefId)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataDefinitionByIdNotFound(t *testing.T) {
	s, mock := getMockDB()
	dataDefId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetDataDefinitionById(context.Background(), "ns1", &dataDefId)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataDefinitionByIdScanFail(t *testing.T) {
	s, mock := getMockDB()
	dataDefId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetDataDefinitionById(context.Background(), "ns1", &dataDefId)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataDefinitionsQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := persistence.DataDefinitionQueryFactory.NewFilter(context.Background(), 0).Eq("id", "")
	_, err := s.GetDataDefinitions(context.Background(), f)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataDefinitionsBuildQueryFail(t *testing.T) {
	s, _ := getMockDB()
	f := persistence.DataDefinitionQueryFactory.NewFilter(context.Background(), 0).Eq("id", map[bool]bool{true: false})
	_, err := s.GetDataDefinitions(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestGetDataDefinitionsReadMessageFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := persistence.DataDefinitionQueryFactory.NewFilter(context.Background(), 0).Eq("id", "")
	_, err := s.GetDataDefinitions(context.Background(), f)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDataDefinitionUpdateBeginFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := persistence.DataDefinitionQueryFactory.NewUpdate(context.Background()).Set("id", "anything")
	err := s.UpdateDataDefinition(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err.Error())
}

func TestDataDefinitionUpdateBuildQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	u := persistence.DataDefinitionQueryFactory.NewUpdate(context.Background()).Set("id", map[bool]bool{true: false})
	err := s.UpdateDataDefinition(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestDataDefinitionUpdateFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := persistence.DataDefinitionQueryFactory.NewUpdate(context.Background()).Set("id", fftypes.NewUUID())
	err := s.UpdateDataDefinition(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err.Error())
}
