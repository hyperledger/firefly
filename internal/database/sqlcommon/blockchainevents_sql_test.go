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

func TestBlockchainEventsE2EWithDB(t *testing.T) {
	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new contract event entry
	event := &fftypes.BlockchainEvent{
		ID:         fftypes.NewUUID(),
		Namespace:  "ns",
		Listener:   fftypes.NewUUID(),
		Name:       "Changed",
		ProtocolID: "tx1",
		Output:     fftypes.JSONObject{"value": 1},
		Info:       fftypes.JSONObject{"blockNumber": 1},
		Timestamp:  fftypes.Now(),
	}

	s.callbacks.On("OrderedUUIDCollectionNSEvent", database.CollectionBlockchainEvents, fftypes.ChangeEventTypeCreated, "ns", event.ID, int64(1)).Return()

	err := s.InsertBlockchainEvent(ctx, event)
	assert.NotNil(t, event.Timestamp)
	assert.NoError(t, err)
	eventJson, _ := json.Marshal(&event)

	// Query back the event (by query filter)
	fb := database.BlockchainEventQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("name", "Changed"),
		fb.Eq("listener", event.Listener),
	)
	events, res, err := s.GetBlockchainEvents(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, int64(1), *res.TotalCount)
	eventReadJson, _ := json.Marshal(events[0])
	assert.Equal(t, string(eventJson), string(eventReadJson))

	// Query back the event (by ID)
	eventRead, err := s.GetBlockchainEventByID(ctx, event.ID)
	assert.NoError(t, err)
	eventReadJson, _ = json.Marshal(eventRead)
	assert.Equal(t, string(eventJson), string(eventReadJson))
}

func TestInsertBlockchainEventFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertBlockchainEvent(context.Background(), &fftypes.BlockchainEvent{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertBlockchainEventFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.InsertBlockchainEvent(context.Background(), &fftypes.BlockchainEvent{})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertBlockchainEventFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertBlockchainEvent(context.Background(), &fftypes.BlockchainEvent{})
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlockchainEventByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetBlockchainEventByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlockchainEventByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetBlockchainEventByID(context.Background(), fftypes.NewUUID())
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlockchainEventByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetBlockchainEventByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlockchainEventsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.BlockchainEventQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetBlockchainEvents(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlockchainEventsBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.BlockchainEventQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	_, _, err := s.GetBlockchainEvents(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestGetBlockchainEventsScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.BlockchainEventQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetBlockchainEvents(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
