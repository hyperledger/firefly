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

func TestContractEventsE2EWithDB(t *testing.T) {
	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new contract event entry
	event := &fftypes.ContractEvent{
		ID:           fftypes.NewUUID(),
		Namespace:    "ns",
		Subscription: fftypes.NewUUID(),
		Name:         "Changed",
		Outputs:      fftypes.JSONObject{"value": 1},
		Info:         fftypes.JSONObject{"blockNumber": 1},
		Timestamp:    fftypes.Now(),
	}

	s.callbacks.On("OrderedUUIDCollectionNSEvent", database.CollectionContractEvents, fftypes.ChangeEventTypeCreated, "ns", event.ID, int64(1)).Return()

	err := s.InsertContractEvent(ctx, event)
	assert.NotNil(t, event.Timestamp)
	assert.NoError(t, err)
	eventJson, _ := json.Marshal(&event)

	// Query back the event (by query filter)
	fb := database.ContractEventQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("name", "Changed"),
		fb.Eq("subscription", event.Subscription),
	)
	events, res, err := s.GetContractEvents(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, int64(1), *res.TotalCount)
	eventReadJson, _ := json.Marshal(events[0])
	assert.Equal(t, string(eventJson), string(eventReadJson))

	// Query back the event (by ID)
	eventRead, err := s.GetContractEventByID(ctx, event.ID)
	assert.NoError(t, err)
	eventReadJson, _ = json.Marshal(eventRead)
	assert.Equal(t, string(eventJson), string(eventReadJson))
}

func TestInsertContractEventFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertContractEvent(context.Background(), &fftypes.ContractEvent{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertContractEventFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.InsertContractEvent(context.Background(), &fftypes.ContractEvent{})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertContractEventFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertContractEvent(context.Background(), &fftypes.ContractEvent{})
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractEventByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetContractEventByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractEventByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetContractEventByID(context.Background(), fftypes.NewUUID())
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractEventByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetContractEventByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractEventsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.ContractEventQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetContractEvents(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractEventsBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.ContractEventQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	_, _, err := s.GetContractEvents(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestGetContractEventsScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.ContractEventQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetContractEvents(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
