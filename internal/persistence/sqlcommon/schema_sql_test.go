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
	"github.com/stretchr/testify/assert"
)

func TestSchemaE2EWithDB(t *testing.T) {

	s := &SQLCommon{}
	ctx := context.Background()
	InitSQLCommon(ctx, s, ensureTestDB(t), nil)

	// Create a new schema entry
	schemaId := uuid.New()
	randB32 := fftypes.NewRandB32()
	val := fftypes.JSONData{
		"some": "schema",
		"with": map[string]interface{}{
			"nesting": 12345,
		},
	}
	schema := &fftypes.Schema{
		ID:        &schemaId,
		Type:      fftypes.SchemaTypeJSONSchema,
		Namespace: "ns1",
		Hash:      &randB32,
		Created:   fftypes.NowMillis(),
		Value:     val,
	}
	err := s.UpsertSchema(ctx, schema)
	assert.NoError(t, err)

	// Check we get the exact same schema back
	schemaRead, err := s.GetSchemaById(ctx, "ns1", &schemaId)
	assert.NoError(t, err)
	assert.NotNil(t, schemaRead)
	schemaJson, _ := json.Marshal(&schema)
	schemaReadJson, _ := json.Marshal(&schemaRead)
	assert.Equal(t, string(schemaJson), string(schemaReadJson))

	// Update the schema (this is testing what's possible at the persistence layer,
	// and does not account for the verification that happens at the higher level)
	val2 := fftypes.JSONData{
		"another": "set",
		"of": map[string]interface{}{
			"schema": 12345,
		},
	}
	schemaUpdated := &fftypes.Schema{
		ID:        &schemaId,
		Type:      fftypes.SchemaTypeJSONSchema,
		Namespace: "ns2",
		Entity:    "customer",
		Version:   "0.0.1",
		Hash:      &randB32,
		Created:   fftypes.NowMillis(),
		Value:     val2,
	}
	err = s.UpsertSchema(context.Background(), schemaUpdated)
	assert.NoError(t, err)

	// Check we get the exact same message back - note the removal of one of the schema elements
	schemaRead, err = s.GetSchemaById(ctx, "ns1", &schemaId)
	assert.NoError(t, err)
	schemaJson, _ = json.Marshal(&schemaUpdated)
	schemaReadJson, _ = json.Marshal(&schemaRead)
	assert.Equal(t, string(schemaJson), string(schemaReadJson))

}

func TestUpsertSchemaFailBegin(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertSchema(context.Background(), &fftypes.Schema{})
	assert.Regexp(t, "FF10114", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertSchemaFailSelect(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	schemaId := uuid.New()
	err := s.UpsertSchema(context.Background(), &fftypes.Schema{ID: &schemaId})
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertSchemaFailInsert(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	schemaId := uuid.New()
	err := s.UpsertSchema(context.Background(), &fftypes.Schema{ID: &schemaId})
	assert.Regexp(t, "FF10116", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertSchemaFailUpdate(t *testing.T) {
	s, mock := getMockDB()
	schemaId := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(schemaId.String()))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertSchema(context.Background(), &fftypes.Schema{ID: &schemaId})
	assert.Regexp(t, "FF10117", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertSchemaFailCommit(t *testing.T) {
	s, mock := getMockDB()
	schemaId := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertSchema(context.Background(), &fftypes.Schema{ID: &schemaId})
	assert.Regexp(t, "FF10119", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSchemaByIdSelectFail(t *testing.T) {
	s, mock := getMockDB()
	schemaId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetSchemaById(context.Background(), "ns1", &schemaId)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSchemaByIdNotFound(t *testing.T) {
	s, mock := getMockDB()
	schemaId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetSchemaById(context.Background(), "ns1", &schemaId)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSchemaByIdScanFail(t *testing.T) {
	s, mock := getMockDB()
	schemaId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetSchemaById(context.Background(), "ns1", &schemaId)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}
