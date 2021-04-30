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
	"sort"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/persistence"
	"github.com/stretchr/testify/assert"
)

func TestUpsertE2EWithDB(t *testing.T) {

	s := &SQLCommon{}
	ctx := context.Background()
	InitSQLCommon(ctx, s, ensureTestDB(t), nil)

	// Create a new message
	msgId := uuid.New()
	dataId1 := uuid.New()
	dataId2 := uuid.New()
	randB32 := fftypes.NewRandB32()
	msg := &fftypes.MessageRefsOnly{
		MessageBase: fftypes.MessageBase{
			ID:        &msgId,
			CID:       nil,
			Type:      fftypes.MessageTypeBroadcast,
			Author:    "0x12345",
			Created:   time.Now().UnixNano(),
			Namespace: "ns12345",
			Topic:     "topic1",
			Context:   "context1",
			Group:     nil,
			DataHash:  &randB32,
			Hash:      &randB32,
			Confirmed: 0,
		},
		TX: nil,
		Data: []fftypes.DataRef{
			{ID: &dataId1},
			{ID: &dataId2},
		},
	}
	err := s.UpsertMessage(ctx, msg)
	assert.NoError(t, err)

	// Check we get the exact same message back - note data gets sorted automatically on retrieve
	sort.Sort(msg.Data)
	msgRead, err := s.GetMessageById(ctx, &msgId)
	assert.NoError(t, err)
	msgJson, _ := json.Marshal(&msg)
	msgReadJson, _ := json.Marshal(&msgRead)
	assert.Equal(t, string(msgJson), string(msgReadJson))

	// Update the message (this is testing what's possible at the persistence layer,
	// and does not account for the verification that happens at the higher level)
	dataId3 := uuid.New()
	cid := uuid.New()
	gid := uuid.New()
	msgUpdated := &fftypes.MessageRefsOnly{
		MessageBase: fftypes.MessageBase{
			ID:        &msgId,
			CID:       &cid,
			Type:      fftypes.MessageTypeBroadcast,
			Author:    "0x12345",
			Created:   time.Now().UnixNano(),
			Namespace: "ns12345",
			Topic:     "topic1",
			Context:   "context1",
			Group:     &gid,
			DataHash:  &randB32,
			Hash:      &randB32,
			Confirmed: time.Now().UnixNano(),
		},
		TX: nil,
		Data: []fftypes.DataRef{
			{ID: &dataId1},
			{ID: &dataId3},
		},
	}
	err = s.UpsertMessage(context.Background(), msgUpdated)
	assert.NoError(t, err)

	// Check we get the exact same message back - note the removal of one of the data elements
	sort.Sort(msgUpdated.Data)
	msgRead, err = s.GetMessageById(ctx, &msgId)
	assert.NoError(t, err)
	msgJson, _ = json.Marshal(&msgUpdated)
	msgReadJson, _ = json.Marshal(&msgRead)
	assert.Equal(t, string(msgJson), string(msgReadJson))

	// Query back the message
	msgs, err := s.GetMessages(ctx, 0, 1, &persistence.MessageFilter{
		ConfrimedOnly:   true,
		IDEquals:        msgUpdated.ID,
		NamespaceEquals: msgUpdated.Namespace,
		TypeEquals:      string(msgUpdated.Type),
		AuthorEquals:    msgUpdated.Author,
		TopicEquals:     msgUpdated.Topic,
		ContextEquals:   msgUpdated.Context,
		GroupEquals:     msgUpdated.Group,
		CIDEquals:       msgUpdated.CID,
		CreatedAfter:    1,
		ConfirmedAfter:  1,
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(msgs))
	msgReadJson, _ = json.Marshal(msgs[0])
	assert.Equal(t, string(msgJson), string(msgReadJson))

}

func TestUpsertMessageFailBegin(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertMessage(context.Background(), &fftypes.MessageRefsOnly{})
	assert.Regexp(t, "FF10114", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertMessageFailSelect(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	msgId := uuid.New()
	err := s.UpsertMessage(context.Background(), &fftypes.MessageRefsOnly{MessageBase: fftypes.MessageBase{ID: &msgId}})
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertMessageFailInsert(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	msgId := uuid.New()
	err := s.UpsertMessage(context.Background(), &fftypes.MessageRefsOnly{MessageBase: fftypes.MessageBase{ID: &msgId}})
	assert.Regexp(t, "FF10116", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertMessageFailUpdate(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(msgId.String()))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertMessage(context.Background(), &fftypes.MessageRefsOnly{MessageBase: fftypes.MessageBase{ID: &msgId}})
	assert.Regexp(t, "FF10117", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertMessageFailLoadRefs(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertMessage(context.Background(), &fftypes.MessageRefsOnly{MessageBase: fftypes.MessageBase{ID: &msgId}})
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertMessageFailCommit(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id"}))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertMessage(context.Background(), &fftypes.MessageRefsOnly{MessageBase: fftypes.MessageBase{ID: &msgId}})
	assert.Regexp(t, "FF10119", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageDataRefsScanFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id"}).AddRow("not the uuid you are looking for"))
	_, err := s.getMessageDataRefs(context.Background(), tx, &msgId)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMessageDataRefsNilID(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	dataId := uuid.New()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id"}).AddRow(dataId.String()))
	err := s.updateMessageDataRefs(context.Background(), tx, &fftypes.MessageRefsOnly{
		MessageBase: fftypes.MessageBase{ID: &msgId},
		Data:        []fftypes.DataRef{{ID: nil}},
	})
	assert.Regexp(t, "FF10123", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMessageDataDeleteFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	dataId := uuid.New()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id"}).AddRow(dataId.String()))
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	err := s.updateMessageDataRefs(context.Background(), tx, &fftypes.MessageRefsOnly{
		MessageBase: fftypes.MessageBase{ID: &msgId},
	})
	assert.Regexp(t, "FF10118", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMessageDataAddFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	dataId := uuid.New()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id"}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.updateMessageDataRefs(context.Background(), tx, &fftypes.MessageRefsOnly{
		MessageBase: fftypes.MessageBase{ID: &msgId},
		Data:        []fftypes.DataRef{{ID: &dataId}},
	})
	assert.Regexp(t, "FF10116", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadMessageDataRefsQueryFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.loadDataRefs(context.Background(), []*fftypes.MessageRefsOnly{
		{
			MessageBase: fftypes.MessageBase{ID: &msgId},
		},
	})
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadMessageDataRefsScanFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id"}).AddRow("only one"))
	err := s.loadDataRefs(context.Background(), []*fftypes.MessageRefsOnly{
		{
			MessageBase: fftypes.MessageBase{ID: &msgId},
		},
	})
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadMessageDataRefsEmpty(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	msg := &fftypes.MessageRefsOnly{MessageBase: fftypes.MessageBase{ID: &msgId}}
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id"}))
	err := s.loadDataRefs(context.Background(), []*fftypes.MessageRefsOnly{msg})
	assert.NoError(t, err)
	assert.Equal(t, fftypes.DataRefSortable{}, msg.Data)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageByIdSelectFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetMessageById(context.Background(), &msgId)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageByIdNotFound(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	// msg := &fftypes.MessageRefsOnly{MessageBase: fftypes.MessageBase{ID: &msgId}}
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetMessageById(context.Background(), &msgId)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageByIdScanFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetMessageById(context.Background(), &msgId)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageByIdLoadRefsFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	b32 := fftypes.NewRandB32()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows(msgColumns).
		AddRow(msgId.String(), nil, fftypes.MessageTypeBroadcast, "0x12345", 0, "ns1", "t1", "c1", nil, b32.String(), b32.String(), 0))
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetMessageById(context.Background(), &msgId)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetMessages(context.Background(), 0, 1, &persistence.MessageFilter{})
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesReadMessageFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetMessages(context.Background(), 0, 1, &persistence.MessageFilter{})
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesLoadRefsFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	b32 := fftypes.NewRandB32()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows(msgColumns).
		AddRow(msgId.String(), nil, fftypes.MessageTypeBroadcast, "0x12345", 0, "ns1", "t1", "c1", nil, b32.String(), b32.String(), 0))
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetMessages(context.Background(), 0, 1, &persistence.MessageFilter{ConfrimedOnly: true})
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}
