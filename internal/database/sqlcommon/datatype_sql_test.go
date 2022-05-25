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
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDatatypeE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new datatype entry
	datatypeID := fftypes.NewUUID()
	randB32 := fftypes.NewRandB32()
	val := fftypes.JSONObject{
		"some": "datatype",
		"with": map[string]interface{}{
			"nesting": 12345,
		},
	}
	datatype := &core.Datatype{
		ID:        datatypeID,
		Message:   fftypes.NewUUID(),
		Validator: core.ValidatorTypeJSON,
		Namespace: "ns1",
		Hash:      randB32,
		Created:   fftypes.Now(),
		Value:     fftypes.JSONAnyPtr(val.String()),
	}

	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionDataTypes, core.ChangeEventTypeCreated, "ns1", datatypeID, mock.Anything).Return()
	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionDataTypes, core.ChangeEventTypeUpdated, "ns1", datatypeID, mock.Anything).Return()

	err := s.UpsertDatatype(ctx, datatype, true)
	assert.NoError(t, err)

	// Check we get the exact same datatype back
	datatypeRead, err := s.GetDatatypeByID(ctx, datatypeID)
	assert.NoError(t, err)
	assert.NotNil(t, datatypeRead)
	datatypeJson, _ := json.Marshal(&datatype)
	datatypeReadJson, _ := json.Marshal(&datatypeRead)
	assert.Equal(t, string(datatypeJson), string(datatypeReadJson))

	// Update the datatype (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	val2 := fftypes.JSONObject{
		"another": "set",
		"of": map[string]interface{}{
			"datatype": 12345,
		},
	}
	datatypeUpdated := &core.Datatype{
		ID:        datatypeID,
		Message:   fftypes.NewUUID(),
		Validator: core.ValidatorTypeJSON,
		Namespace: "ns1",
		Name:      "customer",
		Version:   "0.0.1",
		Hash:      randB32,
		Created:   fftypes.Now(),
		Value:     fftypes.JSONAnyPtr(val2.String()),
	}
	err = s.UpsertDatatype(context.Background(), datatypeUpdated, true)
	assert.NoError(t, err)

	// Check we get the exact same data back - note the removal of one of the datatype elements
	datatypeRead, err = s.GetDatatypeByID(ctx, datatypeID)
	assert.NoError(t, err)
	datatypeJson, _ = json.Marshal(&datatypeUpdated)
	datatypeReadJson, _ = json.Marshal(&datatypeRead)
	assert.Equal(t, string(datatypeJson), string(datatypeReadJson))

	// Query back the data
	fb := database.DatatypeQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("id", datatypeUpdated.ID.String()),
		fb.Eq("namespace", datatypeUpdated.Namespace),
		fb.Eq("validator", string(datatypeUpdated.Validator)),
		fb.Eq("name", datatypeUpdated.Name),
		fb.Eq("version", datatypeUpdated.Version),
		fb.Gt("created", "0"),
	)
	datatypes, res, err := s.GetDatatypes(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(datatypes))
	assert.Equal(t, int64(1), *res.TotalCount)
	datatypeReadJson, _ = json.Marshal(datatypes[0])
	assert.Equal(t, string(datatypeJson), string(datatypeReadJson))

	// Update
	v2 := "2.0.0"
	up := database.DatatypeQueryFactory.NewUpdate(ctx).Set("version", v2)
	err = s.UpdateDatatype(ctx, datatypeUpdated.ID, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("id", datatypeUpdated.ID.String()),
		fb.Eq("version", v2),
	)
	datatypes, _, err = s.GetDatatypes(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(datatypes))

	s.callbacks.AssertExpectations(t)
}

func TestUpsertDatatypeFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertDatatype(context.Background(), &core.Datatype{}, true)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDatatypeFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	datatypeID := fftypes.NewUUID()
	err := s.UpsertDatatype(context.Background(), &core.Datatype{ID: datatypeID}, true)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDatatypeFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	datatypeID := fftypes.NewUUID()
	err := s.UpsertDatatype(context.Background(), &core.Datatype{ID: datatypeID}, true)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDatatypeFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	datatypeID := fftypes.NewUUID()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(datatypeID.String()))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertDatatype(context.Background(), &core.Datatype{ID: datatypeID}, true)
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDatatypeFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	datatypeID := fftypes.NewUUID()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertDatatype(context.Background(), &core.Datatype{ID: datatypeID}, true)
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDatatypeByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	datatypeID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetDatatypeByID(context.Background(), datatypeID)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDatatypeByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	datatypeID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetDatatypeByID(context.Background(), datatypeID)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDatatypeByNameNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetDatatypeByName(context.Background(), "ns1", "name1", "0.0.1")
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestGetDatatypeByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	datatypeID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetDatatypeByID(context.Background(), datatypeID)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDatatypesQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.DatatypeQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetDatatypes(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDatatypesBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.DatatypeQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	_, _, err := s.GetDatatypes(context.Background(), f)
	assert.Regexp(t, "FF00143.*id", err)
}

func TestGetDatatypesReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.DatatypeQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetDatatypes(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDatatypeUpdateBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.DatatypeQueryFactory.NewUpdate(context.Background()).Set("id", "anything")
	err := s.UpdateDatatype(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err)
}

func TestDatatypeUpdateBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	u := database.DatatypeQueryFactory.NewUpdate(context.Background()).Set("id", map[bool]bool{true: false})
	err := s.UpdateDatatype(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF00143.*id", err)
}

func TestDatatypeUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.DatatypeQueryFactory.NewUpdate(context.Background()).Set("id", fftypes.NewUUID())
	err := s.UpdateDatatype(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err)
}
