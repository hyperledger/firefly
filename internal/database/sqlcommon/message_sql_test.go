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

func TestUpsertE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s := &SQLCommon{}
	ctx := context.Background()
	me := databasemocks.Events{}
	InitSQLCommon(ctx, s, ensureTestDB(t), &me, &database.Capabilities{}, testSQLOptions())

	me.On("MessageCreated", mock.Anything).Return()

	// Create a new message
	msgId := uuid.New()
	dataId1 := uuid.New()
	dataId2 := uuid.New()
	rand1 := fftypes.NewRandB32()
	rand2 := fftypes.NewRandB32()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        &msgId,
			CID:       nil,
			Type:      fftypes.MessageTypeBroadcast,
			Author:    "0x12345",
			Created:   fftypes.Now(),
			Namespace: "ns12345",
			Topic:     "topic1",
			Context:   "context1",
			Group:     nil,
			DataHash:  fftypes.NewRandB32(),
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeNone,
			},
		},
		Hash:      fftypes.NewRandB32(),
		Confirmed: nil,
		Data: []fftypes.DataRef{
			{ID: &dataId1, Hash: rand1},
			{ID: &dataId2, Hash: rand2},
		},
	}
	err := s.UpsertMessage(ctx, msg, true)
	assert.NoError(t, err)

	// Check we get the exact same message back
	msgRead, err := s.GetMessageById(ctx, &msgId)
	assert.NoError(t, err)
	// The generated sequence will have been added
	msg.Sequence = msgRead.Sequence
	assert.NoError(t, err)
	msgJson, _ := json.Marshal(&msg)
	msgReadJson, _ := json.Marshal(&msgRead)
	assert.Equal(t, string(msgJson), string(msgReadJson))

	// Update the message (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	dataId3 := uuid.New()
	rand3 := fftypes.NewRandB32()
	cid := uuid.New()
	gid := uuid.New()
	bid := uuid.New()
	txid := uuid.New()
	msgUpdated := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        &msgId,
			CID:       &cid,
			Type:      fftypes.MessageTypeBroadcast,
			Author:    "0x12345",
			Created:   fftypes.Now(),
			Namespace: "ns12345",
			Topic:     "topic1",
			Context:   "context1",
			Group:     &gid,
			DataHash:  fftypes.NewRandB32(),
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypePin,
				ID:   &txid,
			},
		},
		Hash:      fftypes.NewRandB32(),
		Confirmed: fftypes.Now(),
		BatchID:   &bid,
		Data: []fftypes.DataRef{
			{ID: &dataId2, Hash: rand2},
			{ID: &dataId3, Hash: rand3},
		},
	}

	// Ensure hash change rejected
	err = s.UpsertMessage(context.Background(), msgUpdated, false)
	assert.Equal(t, database.HashMismatch, err)

	err = s.UpsertMessage(context.Background(), msgUpdated, true)
	assert.NoError(t, err)

	// Check we get the exact same message back - note the removal of one of the data elements
	msgRead, err = s.GetMessageById(ctx, &msgId)
	// The generated sequence will have been added
	msgUpdated.Sequence = msgRead.Sequence
	assert.NoError(t, err)
	msgJson, _ = json.Marshal(&msgUpdated)
	msgReadJson, _ = json.Marshal(&msgRead)
	assert.Equal(t, string(msgJson), string(msgReadJson))

	// Query back the message
	fb := database.MessageQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("id", msgUpdated.Header.ID.String()),
		fb.Eq("namespace", msgUpdated.Header.Namespace),
		fb.Eq("type", string(msgUpdated.Header.Type)),
		fb.Eq("author", msgUpdated.Header.Author),
		fb.Eq("topic", msgUpdated.Header.Topic),
		fb.Eq("group", msgUpdated.Header.Group),
		fb.Eq("cid", msgUpdated.Header.CID),
		fb.Gt("created", "0"),
		fb.Gt("confirmed", "0"),
	)
	msgs, err := s.GetMessages(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(msgs))
	msgReadJson, _ = json.Marshal(msgs[0])
	assert.Equal(t, string(msgJson), string(msgReadJson))

	// Check just getting hte refs
	msgRefs, err := s.GetMessageRefs(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(msgs))
	assert.Equal(t, msgUpdated.Header.ID, msgRefs[0].ID)
	assert.Equal(t, msgUpdated.Hash, msgRefs[0].Hash)
	assert.Equal(t, msgUpdated.Sequence, msgRefs[0].Sequence)

	// The data will not be available though, confirm we report as such
	dataAvailable, err := s.CheckDataAvailable(ctx, msgUpdated)
	assert.NoError(t, err)
	assert.False(t, dataAvailable)

	// Check we can get it with a filter on only mesasges with a particular data ref
	msgs, err = s.GetMessagesForData(ctx, &dataId3, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(msgs))
	msgReadJson, _ = json.Marshal(msgs[0])
	assert.Equal(t, string(msgJson), string(msgReadJson))

	// Negative test on filter
	filter = fb.And(
		fb.Eq("id", msgUpdated.Header.ID.String()),
		fb.Eq("created", "0"),
	)
	msgs, err = s.GetMessages(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(msgs))

	// Update
	gid2 := uuid.New()
	up := database.MessageQueryFactory.NewUpdate(ctx).Set("group", &gid2)
	err = s.UpdateMessage(ctx, &msgId, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("id", msgUpdated.Header.ID.String()),
		fb.Eq("group", gid2),
	)
	msgs, err = s.GetMessages(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(msgs))

}

func TestUpsertMessageFailBegin(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertMessage(context.Background(), &fftypes.Message{}, true)
	assert.Regexp(t, "FF10114", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertMessageFailSelect(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	msgId := uuid.New()
	err := s.UpsertMessage(context.Background(), &fftypes.Message{Header: fftypes.MessageHeader{ID: &msgId}}, true)
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
	err := s.UpsertMessage(context.Background(), &fftypes.Message{Header: fftypes.MessageHeader{ID: &msgId}}, true)
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
	err := s.UpsertMessage(context.Background(), &fftypes.Message{Header: fftypes.MessageHeader{ID: &msgId}}, true)
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
	err := s.UpsertMessage(context.Background(), &fftypes.Message{Header: fftypes.MessageHeader{ID: &msgId}}, true)
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
	err := s.UpsertMessage(context.Background(), &fftypes.Message{Header: fftypes.MessageHeader{ID: &msgId}}, true)
	assert.Regexp(t, "FF10119", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageDataRefsScanFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id"}).AddRow("not the uuid you are looking for"))
	_, err := s.getMessageDataRefs(context.Background(), &txWrapper{sqlTX: tx}, &msgId)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMessageDataRefsNilID(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	dataId := uuid.New()
	dataHash := fftypes.NewRandB32()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id", "data_hash", "data_idx"}).AddRow(dataId.String(), dataHash.String(), 0))
	err := s.updateMessageDataRefs(context.Background(), &txWrapper{sqlTX: tx}, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: &msgId},
		Data:   []fftypes.DataRef{{ID: nil}},
	})
	assert.Regexp(t, "FF10123", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMessageDataRefsNilHash(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	dataId := uuid.New()
	dataHash := fftypes.NewRandB32()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id", "data_hash", "dataIdx"}).AddRow(dataId.String(), dataHash.String(), 0))
	err := s.updateMessageDataRefs(context.Background(), &txWrapper{sqlTX: tx}, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: &msgId},
		Data:   []fftypes.DataRef{{ID: fftypes.NewUUID()}},
	})
	assert.Regexp(t, "FF10139", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMessageDataDeleteFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	dataId := uuid.New()
	dataHash := fftypes.NewRandB32()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id", "data_hash", "dataIdx"}).AddRow(dataId.String(), dataHash.String(), 0))
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	err := s.updateMessageDataRefs(context.Background(), &txWrapper{sqlTX: tx}, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: &msgId},
	})
	assert.Regexp(t, "FF10118", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMessageDataAddFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	dataId := uuid.New()
	dataHash := fftypes.NewRandB32()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id", "data_hash", "data_idx"}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.updateMessageDataRefs(context.Background(), &txWrapper{sqlTX: tx}, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: &msgId},
		Data:   []fftypes.DataRef{{ID: &dataId, Hash: dataHash}},
	})
	assert.Regexp(t, "FF10116", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMessageDataSwitchIdxFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	dataId1 := uuid.New()
	dataHash1 := fftypes.NewRandB32()
	dataId2 := uuid.New()
	dataHash2 := fftypes.NewRandB32()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id", "data_hash", "data_idx"})).
		WillReturnRows(sqlmock.NewRows(
			[]string{"data_id", "data_hash", "data_idx"},
		).AddRow(
			dataId1, dataHash1, 0,
		))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	err := s.updateMessageDataRefs(context.Background(), &txWrapper{sqlTX: tx}, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: &msgId},
		Data:   []fftypes.DataRef{{ID: &dataId2, Hash: dataHash2}, {ID: &dataId1, Hash: dataHash1}},
	})
	assert.Regexp(t, "FF10117", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadMessageDataRefsQueryFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.loadDataRefs(context.Background(), []*fftypes.Message{
		{
			Header: fftypes.MessageHeader{ID: &msgId},
		},
	})
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadMessageDataRefsScanFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id"}).AddRow("only one"))
	err := s.loadDataRefs(context.Background(), []*fftypes.Message{
		{
			Header: fftypes.MessageHeader{ID: &msgId},
		},
	})
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadMessageDataRefsEmpty(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	msg := &fftypes.Message{Header: fftypes.MessageHeader{ID: &msgId}}
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id", "data_hash"}))
	err := s.loadDataRefs(context.Background(), []*fftypes.Message{msg})
	assert.NoError(t, err)
	assert.Equal(t, fftypes.DataRefs{}, msg.Data)
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
	cols := append([]string{}, msgColumns...)
	cols = append(cols, "id()")
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows(cols).
		AddRow(msgId.String(), nil, fftypes.MessageTypeBroadcast, "0x12345", 0, "ns1", "t1", "c1", nil, b32.String(), b32.String(), 0, "pin", nil, nil, 0))
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetMessageById(context.Background(), &msgId)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesBuildQueryFail(t *testing.T) {
	s, _ := getMockDB()
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	_, err := s.GetMessages(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestGetMessagesQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, err := s.GetMessages(context.Background(), f)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesForDataBadQuery(t *testing.T) {
	s, mock := getMockDB()
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("!wrong", "")
	_, err := s.GetMessagesForData(context.Background(), fftypes.NewUUID(), f)
	assert.Regexp(t, "FF10148", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesReadMessageFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, err := s.GetMessages(context.Background(), f)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesLoadRefsFail(t *testing.T) {
	s, mock := getMockDB()
	msgId := uuid.New()
	b32 := fftypes.NewRandB32()
	cols := append([]string{}, msgColumns...)
	cols = append(cols, "id()")
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows(cols).
		AddRow(msgId.String(), nil, fftypes.MessageTypeBroadcast, "0x12345", 0, "ns1", "t1", "c1", nil, b32.String(), b32.String(), 0, "pin", nil, nil, 0))
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.MessageQueryFactory.NewFilter(context.Background()).Gt("confirmed", "0")
	_, err := s.GetMessages(context.Background(), f)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageRefsBuildQueryFail(t *testing.T) {
	s, _ := getMockDB()
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	_, err := s.GetMessageRefs(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestGetMessageRefsQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, err := s.GetMessageRefs(context.Background(), f)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageRefsReadMessageFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, err := s.GetMessageRefs(context.Background(), f)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageUpdateBeginFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.MessageQueryFactory.NewUpdate(context.Background()).Set("id", "anything")
	err := s.UpdateMessage(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err.Error())
}

func TestMessageUpdateBuildQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	u := database.MessageQueryFactory.NewUpdate(context.Background()).Set("id", map[bool]bool{true: false})
	err := s.UpdateMessage(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestMessagesUpdateBuildFilterFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	u := database.MessageQueryFactory.NewUpdate(context.Background()).Set("type", fftypes.MessageTypeBroadcast)
	err := s.UpdateMessages(context.Background(), f, u)
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestMessageUpdateFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.MessageQueryFactory.NewUpdate(context.Background()).Set("group", fftypes.NewUUID())
	err := s.UpdateMessage(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err.Error())
}

func TestCheckDataAvailableFalseBadMessage(t *testing.T) {
	s, _ := getMockDB()
	ok, err := s.CheckDataAvailable(context.Background(), &fftypes.Message{})
	assert.False(t, ok)
	assert.NoError(t, err) // function just returns false and logs if the data cannot be checked for input error
}

func TestCheckDataAvailableFalseBadMessageRefs(t *testing.T) {
	s, _ := getMockDB()
	ok, err := s.CheckDataAvailable(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID(), Hash: nil},
		},
	})
	assert.False(t, ok)
	assert.NoError(t, err) // function just returns false and logs if the data cannot be checked for input error
}

func TestCheckDataAvailableTrueNoData(t *testing.T) {
	s, _ := getMockDB()
	ok, err := s.CheckDataAvailable(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
		},
	})
	assert.True(t, ok)
	assert.NoError(t, err)
}

func TestCheckDataAvailableDatabaseError(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	ok, err := s.CheckDataAvailable(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
		},
	})
	assert.Regexp(t, "FF10115", err.Error())
	assert.False(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckDataAvailableScanError(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	ok, err := s.CheckDataAvailable(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
		},
	})
	assert.Regexp(t, "FF10121", err.Error())
	assert.False(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}
