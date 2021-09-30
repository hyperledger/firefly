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
	"github.com/stretchr/testify/mock"
)

func TestEventE2EWithDB(t *testing.T) {

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new event entry
	eventID := fftypes.NewUUID()
	event := &fftypes.Event{
		ID:        eventID,
		Namespace: "ns1",
		Type:      fftypes.EventTypeMessageConfirmed,
		Reference: fftypes.NewUUID(),
		Created:   fftypes.Now(),
	}

	s.callbacks.On("OrderedUUIDCollectionNSEvent", database.CollectionEvents, fftypes.ChangeEventTypeCreated, "ns1", eventID, mock.Anything).Return()

	err := s.InsertEvent(ctx, event)
	assert.NoError(t, err)

	// Check we get the exact same event back
	eventRead, err := s.GetEventByID(ctx, eventID)
	// The generated sequence will have been added
	event.Sequence = eventRead.Sequence
	assert.NoError(t, err)
	assert.NotNil(t, eventRead)
	eventJson, _ := json.Marshal(&event)
	eventReadJson, _ := json.Marshal(&eventRead)
	assert.Equal(t, string(eventJson), string(eventReadJson))

	// Query back the event
	fb := database.EventQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("id", eventRead.ID.String()),
		fb.Eq("reference", eventRead.Reference.String()),
	)
	events, res, err := s.GetEvents(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, int64(1), *res.TotalCount)
	eventReadJson, _ = json.Marshal(events[0])
	assert.Equal(t, string(eventJson), string(eventReadJson))

	// Negative test on filter
	filter = fb.And(
		fb.Eq("id", eventRead.ID.String()),
		fb.Eq("reference", fftypes.NewUUID().String()),
	)
	events, _, err = s.GetEvents(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(events))

	// Update
	newUUID := fftypes.NewUUID()
	up := database.EventQueryFactory.NewUpdate(ctx).
		Set("reference", newUUID)
	err = s.UpdateEvent(ctx, eventRead.ID, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("id", eventRead.ID.String()),
		fb.Eq("reference", newUUID),
	)
	events, _, err = s.GetEvents(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))

	s.callbacks.AssertExpectations(t)
}

func TestInsertEventFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertEvent(context.Background(), &fftypes.Event{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertEventFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	eventID := fftypes.NewUUID()
	err := s.InsertEvent(context.Background(), &fftypes.Event{ID: eventID})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertEventFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	eventID := fftypes.NewUUID()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertEvent(context.Background(), &fftypes.Event{ID: eventID})
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetEventByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	eventID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetEventByID(context.Background(), eventID)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetEventByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	eventID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetEventByID(context.Background(), eventID)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetEventByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	eventID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetEventByID(context.Background(), eventID)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetEventsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.EventQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetEvents(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetEventsBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.EventQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	_, _, err := s.GetEvents(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestGettEventsReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.EventQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetEvents(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEventUpdateBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.EventQueryFactory.NewUpdate(context.Background()).Set("id", "anything")
	err := s.UpdateEvent(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err)
}

func TestEventUpdateBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	u := database.EventQueryFactory.NewUpdate(context.Background()).Set("id", map[bool]bool{true: false})
	err := s.UpdateEvent(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestEventUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.EventQueryFactory.NewUpdate(context.Background()).Set("id", fftypes.NewUUID())
	err := s.UpdateEvent(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err)
}
