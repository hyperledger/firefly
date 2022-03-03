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
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestContractListenerE2EWithDB(t *testing.T) {
	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new contract listener entry
	location := fftypes.JSONObject{"path": "my-api"}
	locationJson, _ := json.Marshal(location)
	sub := &fftypes.ContractListener{
		ID: fftypes.NewUUID(),
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
		Event: &fftypes.FFISerializedEvent{
			FFIEventDefinition: fftypes.FFIEventDefinition{
				Name: "event1",
			},
		},
		Namespace:  "ns",
		Name:       "sub1",
		ProtocolID: "sb-123",
		Location:   fftypes.JSONAnyPtrBytes(locationJson),
	}

	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionContractListeners, fftypes.ChangeEventTypeCreated, "ns", sub.ID).Return()
	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionContractListeners, fftypes.ChangeEventTypeUpdated, "ns", sub.ID).Return()
	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionContractListeners, fftypes.ChangeEventTypeDeleted, "ns", sub.ID).Return()

	err := s.UpsertContractListener(ctx, sub)
	assert.NotNil(t, sub.Created)
	assert.NoError(t, err)
	subJson, _ := json.Marshal(&sub)

	// Query back the listener (by query filter)
	fb := database.ContractListenerQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("protocolid", sub.ProtocolID),
	)
	subs, res, err := s.GetContractListeners(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(subs))
	assert.Equal(t, int64(1), *res.TotalCount)
	subReadJson, _ := json.Marshal(subs[0])
	assert.Equal(t, string(subJson), string(subReadJson))

	// Query back the listener (by name)
	subRead, err := s.GetContractListener(ctx, "ns", "sub1")
	assert.NoError(t, err)
	subReadJson, _ = json.Marshal(subRead)
	assert.Equal(t, string(subJson), string(subReadJson))

	// Query back the listener (by ID)
	subRead, err = s.GetContractListenerByID(ctx, sub.ID)
	assert.NoError(t, err)
	subReadJson, _ = json.Marshal(subRead)
	assert.Equal(t, string(subJson), string(subReadJson))

	// Query back the listener (by protocol ID)
	subRead, err = s.GetContractListenerByProtocolID(ctx, sub.ProtocolID)
	assert.NoError(t, err)
	subReadJson, _ = json.Marshal(subRead)
	assert.Equal(t, string(subJson), string(subReadJson))

	// Update the listener
	sub.Location = fftypes.JSONAnyPtr("{}")
	subJson, _ = json.Marshal(&sub)
	err = s.UpsertContractListener(ctx, sub)
	assert.NoError(t, err)

	// Query back the listener (by query filter)
	filter = fb.And(
		fb.Eq("protocolid", sub.ProtocolID),
	)
	subs, res, err = s.GetContractListeners(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(subs))
	assert.Equal(t, int64(1), *res.TotalCount)
	subReadJson, _ = json.Marshal(subs[0])
	assert.Equal(t, string(subJson), string(subReadJson))

	// Test delete, and refind no return
	err = s.DeleteContractListenerByID(ctx, sub.ID)
	assert.NoError(t, err)
	subs, _, err = s.GetContractListeners(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(subs))
}

func TestUpsertContractListenerFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertContractListener(context.Background(), &fftypes.ContractListener{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertContractListenerFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertContractListener(context.Background(), &fftypes.ContractListener{})
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertContractListenerFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertContractListener(context.Background(), &fftypes.ContractListener{})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertContractListenerFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"protocolid"}).AddRow("1"))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertContractListener(context.Background(), &fftypes.ContractListener{})
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertContractListenerFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"protocolid"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertContractListener(context.Background(), &fftypes.ContractListener{})
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractListenerByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetContractListenerByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractListenerByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"protocolid"}))
	msg, err := s.GetContractListenerByID(context.Background(), fftypes.NewUUID())
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractListenerByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"protocolid"}).AddRow("only one"))
	_, err := s.GetContractListenerByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractListenersQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.ContractListenerQueryFactory.NewFilter(context.Background()).Eq("protocolid", "")
	_, _, err := s.GetContractListeners(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractListenersBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.ContractListenerQueryFactory.NewFilter(context.Background()).Eq("protocolid", map[bool]bool{true: false})
	_, _, err := s.GetContractListeners(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestGetContractListenersScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"protocolid"}).AddRow("only one"))
	f := database.ContractListenerQueryFactory.NewFilter(context.Background()).Eq("protocolid", "")
	_, _, err := s.GetContractListeners(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContractListenerDeleteBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteContractListenerByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10114", err)
}

func TestContractListenerDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows(contractListenerColumns).AddRow(
		fftypes.NewUUID(), nil, []byte("{}"), "ns1", "sub1", "123", "{}", fftypes.Now()),
	)
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteContractListenerByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10118", err)
}
