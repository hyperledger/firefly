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
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEventE2EWithDB(t *testing.T) {

	log.SetLevel("trace")
	s := &SQLCommon{}
	ctx := context.Background()
	me := databasemocks.Events{}
	InitSQLCommon(ctx, s, ensureTestDB(t), &me, &database.Capabilities{}, testSQLOptions())

	me.On("EventCreated", mock.Anything).Return()

	// Create a new event entry
	eventId := uuid.New()
	event := &fftypes.Event{
		ID:        &eventId,
		Namespace: "ns1",
		Type:      fftypes.EventTypeMessageConfirmed,
		Reference: fftypes.NewUUID(),
		Created:   fftypes.Now(),
	}
	err := s.UpsertEvent(ctx, event)
	assert.NoError(t, err)

	// Check we get the exact same event back
	eventRead, err := s.GetEventById(ctx, &eventId)
	// The generated sequence will have been added
	event.Sequence = eventRead.Sequence
	assert.NoError(t, err)
	assert.NotNil(t, eventRead)
	eventJson, _ := json.Marshal(&event)
	eventReadJson, _ := json.Marshal(&eventRead)
	assert.Equal(t, string(eventJson), string(eventReadJson))

	// Update the event (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	eventUpdated := &fftypes.Event{
		ID:        &eventId,
		Namespace: "ns1",
		Type:      fftypes.EventTypeMessageConfirmed,
		Reference: fftypes.NewUUID(),
		Created:   fftypes.Now(),
	}
	err = s.UpsertEvent(context.Background(), eventUpdated)
	assert.NoError(t, err)

	// Check we get the exact same message back - note the removal of one of the event elements
	eventRead, err = s.GetEventById(ctx, &eventId)
	assert.NoError(t, err)
	// The generated sequence will have been added
	eventUpdated.Sequence = event.Sequence
	eventJson, _ = json.Marshal(&eventUpdated)
	eventReadJson, _ = json.Marshal(&eventRead)
	assert.Equal(t, string(eventJson), string(eventReadJson))

	// Query back the event
	fb := database.EventQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("id", eventUpdated.ID.String()),
		fb.Eq("reference", eventUpdated.Reference.String()),
	)
	events, err := s.GetEvents(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	eventReadJson, _ = json.Marshal(events[0])
	assert.Equal(t, string(eventJson), string(eventReadJson))

	// Negative test on filter
	filter = fb.And(
		fb.Eq("id", eventUpdated.ID.String()),
		fb.Eq("reference", fftypes.NewUUID().String()),
	)
	events, err = s.GetEvents(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(events))

	// Update
	newUUID := fftypes.NewUUID()
	up := database.EventQueryFactory.NewUpdate(ctx).
		Set("reference", newUUID)
	err = s.UpdateEvent(ctx, eventUpdated.ID, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("id", eventUpdated.ID.String()),
		fb.Eq("reference", newUUID),
	)
	events, err = s.GetEvents(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
}

func TestUpsertEventFailBegin(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertEvent(context.Background(), &fftypes.Event{})
	assert.Regexp(t, "FF10114", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertEventFailSelect(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	eventId := uuid.New()
	err := s.UpsertEvent(context.Background(), &fftypes.Event{ID: &eventId})
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertEventFailInsert(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	eventId := uuid.New()
	err := s.UpsertEvent(context.Background(), &fftypes.Event{ID: &eventId})
	assert.Regexp(t, "FF10116", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertEventFailUpdate(t *testing.T) {
	s, mock := getMockDB()
	eventId := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(eventId.String()))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertEvent(context.Background(), &fftypes.Event{ID: &eventId})
	assert.Regexp(t, "FF10117", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertEventFailCommit(t *testing.T) {
	s, mock := getMockDB()
	eventId := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertEvent(context.Background(), &fftypes.Event{ID: &eventId})
	assert.Regexp(t, "FF10119", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetEventByIdSelectFail(t *testing.T) {
	s, mock := getMockDB()
	eventId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetEventById(context.Background(), &eventId)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetEventByIdNotFound(t *testing.T) {
	s, mock := getMockDB()
	eventId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetEventById(context.Background(), &eventId)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetEventByIdScanFail(t *testing.T) {
	s, mock := getMockDB()
	eventId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetEventById(context.Background(), &eventId)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetEventsQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.EventQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, err := s.GetEvents(context.Background(), f)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetEventsBuildQueryFail(t *testing.T) {
	s, _ := getMockDB()
	f := database.EventQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	_, err := s.GetEvents(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestGettEventsReadMessageFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.EventQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, err := s.GetEvents(context.Background(), f)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEventUpdateBeginFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.EventQueryFactory.NewUpdate(context.Background()).Set("id", "anything")
	err := s.UpdateEvent(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err.Error())
}

func TestEventUpdateBuildQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	u := database.EventQueryFactory.NewUpdate(context.Background()).Set("id", map[bool]bool{true: false})
	err := s.UpdateEvent(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestEventUpdateFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.EventQueryFactory.NewUpdate(context.Background()).Set("id", fftypes.NewUUID())
	err := s.UpdateEvent(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err.Error())
}
