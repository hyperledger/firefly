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
)

func TestConfigRecordE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new namespace entry
	configRecord := &fftypes.ConfigRecord{
		Key:   "foo",
		Value: fftypes.Byteable(`{"foo":"bar"}`),
	}
	err := s.UpsertConfigRecord(ctx, configRecord, true)
	assert.NoError(t, err)

	// Check we get the exact same config record back
	configRecordRead, err := s.GetConfigRecord(ctx, configRecord.Key)
	assert.NoError(t, err)
	assert.NotNil(t, configRecordRead)
	configRecordJson, _ := json.Marshal(&configRecord)
	configRecordReadJson, _ := json.Marshal(&configRecordRead)
	assert.Equal(t, string(configRecordJson), string(configRecordReadJson))

	// Update the config record (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	configRecordUpdated := &fftypes.ConfigRecord{
		Key:   "foo",
		Value: fftypes.Byteable(`{"fiz":"buzz"}`),
	}
	err = s.UpsertConfigRecord(context.Background(), configRecordUpdated, true)
	assert.NoError(t, err)

	// Check we get the exact same data back
	configRecordRead, err = s.GetConfigRecord(ctx, configRecord.Key)
	assert.NoError(t, err)
	configRecordJson, _ = json.Marshal(&configRecordUpdated)
	configRecordReadJson, _ = json.Marshal(&configRecordRead)
	assert.Equal(t, string(configRecordJson), string(configRecordReadJson))

	// Query back the config record
	fb := database.ConfigRecordQueryFactory.NewFilter(ctx)
	filter := fb.And()
	configRecordRes, res, err := s.GetConfigRecords(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configRecordRes))
	assert.Equal(t, int64(1), res.Count)
	configRecordReadJson, _ = json.Marshal(configRecordRes[0])
	assert.Equal(t, string(configRecordJson), string(configRecordReadJson))

	// Delete the record
	err = s.DeleteConfigRecord(ctx, "foo")
	assert.NoError(t, err)
	configRecordRes, _, err = s.GetConfigRecords(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(configRecordRes))
}

func TestUpsertConfigRecordFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertConfigRecord(context.Background(), &fftypes.ConfigRecord{}, true)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertConfigRecordFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertConfigRecord(context.Background(), &fftypes.ConfigRecord{Key: "key1"}, true)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertConfigRecordFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertConfigRecord(context.Background(), &fftypes.ConfigRecord{Key: "key1"}, true)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertConfigRecordFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	configID := fftypes.NewUUID()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"key"}).AddRow(configID.String()))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertConfigRecord(context.Background(), &fftypes.ConfigRecord{Key: "key1"}, true)
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertConfigRecordFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"key"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertConfigRecord(context.Background(), &fftypes.ConfigRecord{Key: "key1"}, true)
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetConfigRecordByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetConfigRecord(context.Background(), "key1")
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetConfigRecordByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"key"}))
	msg, err := s.GetConfigRecord(context.Background(), "key1")
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetConfigRecordByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"key"}).AddRow("only one"))
	_, err := s.GetConfigRecord(context.Background(), "key1")
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetConfigRecordsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.ConfigRecordQueryFactory.NewFilter(context.Background()).Eq("key", "")
	_, _, err := s.GetConfigRecords(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetConfigRecordsBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.ConfigRecordQueryFactory.NewFilter(context.Background()).Eq("key", map[bool]bool{true: false})
	_, _, err := s.GetConfigRecords(context.Background(), f)
	assert.Regexp(t, "FF10149.*key", err)
}

func TestGettConfigRecordsReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"key"}).AddRow("only one"))
	f := database.ConfigRecordQueryFactory.NewFilter(context.Background()).Eq("key", "")
	_, _, err := s.GetConfigRecords(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConfigRecordDeleteBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteConfigRecord(context.Background(), "key1")
	assert.Regexp(t, "FF10114", err)
}

func TestConfigRecordDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteConfigRecord(context.Background(), "key1")
	assert.Regexp(t, "FF10118", err)
}
