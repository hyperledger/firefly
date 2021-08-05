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
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDataE2EWithDB(t *testing.T) {
	log.SetLevel("trace")

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new data entry
	dataID := fftypes.NewUUID()
	val := fftypes.JSONObject{
		"some": "data",
		"with": map[string]interface{}{
			"nesting": 12345,
		},
	}
	data := &fftypes.Data{
		ID:        dataID,
		Validator: fftypes.ValidatorTypeSystemDefinition,
		Namespace: "ns1",
		Hash:      fftypes.NewRandB32(),
		Created:   fftypes.Now(),
		Value:     []byte(val.String()),
	}

	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionData, fftypes.ChangeEventTypeCreated, "ns1", dataID, mock.Anything).Return()
	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionData, fftypes.ChangeEventTypeUpdated, "ns1", dataID, mock.Anything).Return()

	err := s.UpsertData(ctx, data, true, false)
	assert.NoError(t, err)

	// Check we get the exact same data back - we should not to return the value first
	dataRead, err := s.GetDataByID(ctx, dataID, false)
	assert.NoError(t, err)
	assert.Equal(t, *dataID, *dataRead.ID)
	assert.Nil(t, dataRead.Value)

	// Now with value
	dataRead, err = s.GetDataByID(ctx, dataID, true)
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
			"and":  "stuff",
		},
	}
	dataUpdated := &fftypes.Data{
		ID:        dataID,
		Validator: fftypes.ValidatorTypeJSON,
		Namespace: "ns1",
		Datatype: &fftypes.DatatypeRef{
			Name:    "customer",
			Version: "0.0.1",
		},
		Hash:    fftypes.NewRandB32(),
		Created: fftypes.Now(),
		Value:   []byte(val2.String()),
		Blob: &fftypes.BlobRef{
			Hash:   fftypes.NewRandB32(),
			Public: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		},
	}

	// Check disallows hash update
	err = s.UpsertData(context.Background(), dataUpdated, true, false)
	assert.Equal(t, database.HashMismatch, err)

	err = s.UpsertData(context.Background(), dataUpdated, true, true)
	assert.NoError(t, err)

	// Check we get the exact same message back - note the removal of one of the data elements
	dataRead, err = s.GetDataByID(ctx, dataID, true)
	assert.NoError(t, err)
	dataJson, _ = json.Marshal(&dataUpdated)
	dataReadJson, _ = json.Marshal(&dataRead)
	assert.Equal(t, string(dataJson), string(dataReadJson))

	valRestored, ok := dataRead.Value.JSONObjectOk()
	assert.True(t, ok)
	assert.Equal(t, "stuff", valRestored.GetObject("of").GetString("and"))

	// Query back the data
	fb := database.DataQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("id", dataUpdated.ID.String()),
		fb.Eq("namespace", dataUpdated.Namespace),
		fb.Eq("validator", string(dataUpdated.Validator)),
		fb.Eq("datatype.name", dataUpdated.Datatype.Name),
		fb.Eq("datatype.version", dataUpdated.Datatype.Version),
		fb.Eq("hash", dataUpdated.Hash),
		fb.Gt("created", 0),
	)
	dataRes, _, err := s.GetData(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(dataRes))
	dataReadJson, _ = json.Marshal(dataRes[0])
	assert.Equal(t, string(dataJson), string(dataReadJson))

	dataRefRes, _, err := s.GetDataRefs(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(dataRefRes))
	assert.Equal(t, *dataUpdated.ID, *dataRefRes[0].ID)
	assert.Equal(t, dataUpdated.Hash, dataRefRes[0].Hash)

	// Update
	v2 := "2.0.0"
	up := database.DataQueryFactory.NewUpdate(ctx).Set("datatype.version", v2)
	err = s.UpdateData(ctx, dataID, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("id", dataUpdated.ID.String()),
		fb.Eq("datatype.version", v2),
	)
	dataRes, res, err := s.GetData(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(dataRes))
	assert.Equal(t, int64(1), *res.Count)

	s.callbacks.AssertExpectations(t)
}

func TestUpsertDataFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertData(context.Background(), &fftypes.Data{}, true, true)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDataFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	dataID := fftypes.NewUUID()
	err := s.UpsertData(context.Background(), &fftypes.Data{ID: dataID}, true, true)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDataFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	dataID := fftypes.NewUUID()
	err := s.UpsertData(context.Background(), &fftypes.Data{ID: dataID}, true, true)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDataFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	dataID := fftypes.NewUUID()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(dataID.String()))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertData(context.Background(), &fftypes.Data{ID: dataID}, true, true)
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertDataFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	dataID := fftypes.NewUUID()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertData(context.Background(), &fftypes.Data{ID: dataID}, true, true)
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	dataID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetDataByID(context.Background(), dataID, false)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	dataID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetDataByID(context.Background(), dataID, true)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	dataID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetDataByID(context.Background(), dataID, true)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.DataQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetData(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.DataQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	_, _, err := s.GetData(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestGetDataReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.DataQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetData(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataRefsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.DataQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetDataRefs(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataRefsBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.DataQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	_, _, err := s.GetDataRefs(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestGetDataRefsReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.DataQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetDataRefs(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDataUpdateBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.DataQueryFactory.NewUpdate(context.Background()).Set("id", "anything")
	err := s.UpdateData(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err)
}

func TestDataUpdateBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	u := database.DataQueryFactory.NewUpdate(context.Background()).Set("id", map[bool]bool{true: false})
	err := s.UpdateData(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestDataUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.DataQueryFactory.NewUpdate(context.Background()).Set("id", fftypes.NewUUID())
	err := s.UpdateData(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err)
}
