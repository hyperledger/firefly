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

func TestSchemaE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s := &SQLCommon{}
	ctx := context.Background()
	InitSQLCommon(ctx, s, ensureTestDB(t), nil, &persistence.Capabilities{}, testSQLOptions())

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

	// Check we get the exact same data back - note the removal of one of the schema elements
	schemaRead, err = s.GetSchemaById(ctx, "ns1", &schemaId)
	assert.NoError(t, err)
	schemaJson, _ = json.Marshal(&schemaUpdated)
	schemaReadJson, _ = json.Marshal(&schemaRead)
	assert.Equal(t, string(schemaJson), string(schemaReadJson))

	// Query back the data
	fb := persistence.SchemaQueryFactory.NewFilter(ctx, 0)
	filter := fb.And(
		fb.Eq("id", schemaUpdated.ID.String()),
		fb.Eq("namespace", schemaUpdated.Namespace),
		fb.Eq("type", string(schemaUpdated.Type)),
		fb.Eq("entity", schemaUpdated.Entity),
		fb.Eq("version", schemaUpdated.Version),
		fb.Gt("created", "0"),
	)
	schemas, err := s.GetSchemas(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(schemas))
	schemaReadJson, _ = json.Marshal(schemas[0])
	assert.Equal(t, string(schemaJson), string(schemaReadJson))

	// Update
	v2 := "2.0.0"
	up := persistence.SchemaQueryFactory.NewUpdate(ctx).Set("version", v2)
	err = s.UpdateSchema(ctx, schemaUpdated.ID, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("id", schemaUpdated.ID.String()),
		fb.Eq("version", v2),
	)
	schemas, err = s.GetSchemas(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(schemas))
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

func TestGetSchemasQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := persistence.SchemaQueryFactory.NewFilter(context.Background(), 0).Eq("id", "")
	_, err := s.GetSchemas(context.Background(), f)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSchemasBuildQueryFail(t *testing.T) {
	s, _ := getMockDB()
	f := persistence.SchemaQueryFactory.NewFilter(context.Background(), 0).Eq("id", map[bool]bool{true: false})
	_, err := s.GetSchemas(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestGetSchemasReadMessageFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := persistence.SchemaQueryFactory.NewFilter(context.Background(), 0).Eq("id", "")
	_, err := s.GetSchemas(context.Background(), f)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSchemaUpdateBeginFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := persistence.SchemaQueryFactory.NewUpdate(context.Background()).Set("id", "anything")
	err := s.UpdateSchema(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err.Error())
}

func TestSchemaUpdateBuildQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	u := persistence.SchemaQueryFactory.NewUpdate(context.Background()).Set("id", map[bool]bool{true: false})
	err := s.UpdateSchema(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestSchemaUpdateFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := persistence.SchemaQueryFactory.NewUpdate(context.Background()).Set("id", fftypes.NewUUID())
	err := s.UpdateSchema(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err.Error())
}
