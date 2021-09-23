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
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUpsertE2EWithDB(t *testing.T) {
	log.SetLevel("trace")

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new message
	msgID := fftypes.NewUUID()
	dataID1 := fftypes.NewUUID()
	dataID2 := fftypes.NewUUID()
	rand1 := fftypes.NewRandB32()
	rand2 := fftypes.NewRandB32()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        msgID,
			CID:       nil,
			Type:      fftypes.MessageTypeBroadcast,
			Author:    "0x12345",
			Created:   fftypes.Now(),
			Namespace: "ns12345",
			Topics:    []string{"test1"},
			Group:     nil,
			DataHash:  fftypes.NewRandB32(),
			TxType:    fftypes.TransactionTypeNone,
		},
		Hash:      fftypes.NewRandB32(),
		Pending:   true,
		Confirmed: nil,
		Data: []*fftypes.DataRef{
			{ID: dataID1, Hash: rand1},
			{ID: dataID2, Hash: rand2},
		},
	}

	s.callbacks.On("OrderedUUIDCollectionNSEvent", database.CollectionMessages, fftypes.ChangeEventTypeCreated, "ns12345", msgID, mock.Anything).Return()
	s.callbacks.On("OrderedUUIDCollectionNSEvent", database.CollectionMessages, fftypes.ChangeEventTypeUpdated, "ns12345", msgID, mock.Anything).Return()

	err := s.InsertMessageLocal(ctx, msg)
	assert.NoError(t, err)

	// Check we get the exact same message back
	msgRead, err := s.GetMessageByID(ctx, msgID)
	assert.NoError(t, err)
	assert.True(t, msgRead.Local)
	// The generated sequence will have been added
	msg.Sequence = msgRead.Sequence
	assert.NoError(t, err)
	msgJson, _ := json.Marshal(&msg)
	msgReadJson, _ := json.Marshal(&msgRead)
	assert.Equal(t, string(msgJson), string(msgReadJson))

	// Update the message (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	dataID3 := fftypes.NewUUID()
	rand3 := fftypes.NewRandB32()
	cid := fftypes.NewUUID()
	gid := fftypes.NewRandB32()
	bid := fftypes.NewUUID()
	msgUpdated := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        msgID,
			CID:       cid,
			Type:      fftypes.MessageTypeBroadcast,
			Author:    "0x12345",
			Created:   fftypes.Now(),
			Namespace: "ns12345",
			Topics:    []string{"topic1", "topic2"},
			Tag:       "tag1",
			Group:     gid,
			DataHash:  fftypes.NewRandB32(),
			TxType:    fftypes.TransactionTypeBatchPin,
		},
		Hash:      fftypes.NewRandB32(),
		Pins:      []string{fftypes.NewRandB32().String(), fftypes.NewRandB32().String()},
		Rejected:  true,
		Pending:   false,
		Confirmed: fftypes.Now(),
		BatchID:   bid,
		Data: []*fftypes.DataRef{
			{ID: dataID2, Hash: rand2},
			{ID: dataID3, Hash: rand3},
		},
		Local: false, // must be ignored
	}

	// Ensure hash change rejected
	err = s.UpsertMessage(context.Background(), msgUpdated, true, false)
	assert.Equal(t, database.HashMismatch, err)

	err = s.UpsertMessage(context.Background(), msgUpdated, true, true)
	assert.NoError(t, err)

	// Check we get the exact same message back - note the removal of one of the data elements
	msgRead, err = s.GetMessageByID(ctx, msgID)
	assert.True(t, msgRead.Local) // Must not have been overridden with the update
	// The generated sequence will have been added
	msgUpdated.Sequence = msgRead.Sequence
	msgUpdated.Local = true // retained
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
		fb.Eq("topics", msgUpdated.Header.Topics),
		fb.Eq("group", msgUpdated.Header.Group),
		fb.Eq("cid", msgUpdated.Header.CID),
		fb.Eq("local", true),
		fb.Gt("created", "0"),
		fb.Gt("confirmed", "0"),
	)
	msgs, res, err := s.GetMessages(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(msgs))
	assert.Equal(t, int64(1), *res.TotalCount)
	msgReadJson, _ = json.Marshal(msgs[0])
	assert.Equal(t, string(msgJson), string(msgReadJson))

	// Check just getting hte refs
	msgRefs, res, err := s.GetMessageRefs(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(msgs))
	assert.Equal(t, int64(1), *res.TotalCount)
	assert.Equal(t, msgUpdated.Header.ID, msgRefs[0].ID)
	assert.Equal(t, msgUpdated.Hash, msgRefs[0].Hash)
	assert.Equal(t, msgUpdated.Sequence, msgRefs[0].Sequence)

	// Check we can get it with a filter on only mesasges with a particular data ref
	msgs, _, err = s.GetMessagesForData(ctx, dataID3, filter.Count(true))
	assert.Regexp(t, "FF10267", err) // The left join means it will take non-trivial extra work to support this. So not supported for now
	msgs, _, err = s.GetMessagesForData(ctx, dataID3, filter.Count(false))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(msgs))
	msgReadJson, _ = json.Marshal(msgs[0])
	assert.Equal(t, string(msgJson), string(msgReadJson))

	// Negative test on filter
	filter = fb.And(
		fb.Eq("id", msgUpdated.Header.ID.String()),
		fb.Eq("created", "0"),
	)
	msgs, _, err = s.GetMessages(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(msgs))

	// Update
	gid2 := fftypes.NewRandB32()
	bid2 := fftypes.NewUUID()
	up := database.MessageQueryFactory.NewUpdate(ctx).
		Set("group", gid2).
		Set("batch", bid2)
	err = s.UpdateMessage(ctx, msgID, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("id", msgUpdated.Header.ID.String()),
		fb.Eq("group", gid2),
	)
	msgs, _, err = s.GetMessages(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(msgs))
	assert.Equal(t, *bid2, *msgs[0].BatchID)

	s.callbacks.AssertExpectations(t)
}

func TestUpsertMessageFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertMessage(context.Background(), &fftypes.Message{}, true, true)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertMessageFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	msgID := fftypes.NewUUID()
	err := s.UpsertMessage(context.Background(), &fftypes.Message{Header: fftypes.MessageHeader{ID: msgID}}, true, true)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertMessageFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	msgID := fftypes.NewUUID()
	err := s.UpsertMessage(context.Background(), &fftypes.Message{Header: fftypes.MessageHeader{ID: msgID}}, true, true)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertMessageFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	msgID := fftypes.NewUUID()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(msgID.String()))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertMessage(context.Background(), &fftypes.Message{Header: fftypes.MessageHeader{ID: msgID}}, true, true)
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertMessageFailUpdateRefs(t *testing.T) {
	s, mock := newMockProvider().init()
	msgID := fftypes.NewUUID()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(msgID.String()))
	mock.ExpectExec("UPDATE .*").WillReturnResult(driver.ResultNoRows)
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertMessage(context.Background(), &fftypes.Message{Header: fftypes.MessageHeader{ID: msgID}}, true, true)
	assert.Regexp(t, "FF10118", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertMessageFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	msgID := fftypes.NewUUID()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertMessage(context.Background(), &fftypes.Message{Header: fftypes.MessageHeader{ID: msgID}}, true, true)
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMessageDataRefsNilID(t *testing.T) {
	s, mock := newMockProvider().init()
	msgID := fftypes.NewUUID()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	err := s.updateMessageDataRefs(context.Background(), &txWrapper{sqlTX: tx}, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: msgID},
		Data:   []*fftypes.DataRef{{ID: nil}},
	}, false)
	assert.Regexp(t, "FF10123", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMessageDataRefsNilHash(t *testing.T) {
	s, mock := newMockProvider().init()
	msgID := fftypes.NewUUID()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	err := s.updateMessageDataRefs(context.Background(), &txWrapper{sqlTX: tx}, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: msgID},
		Data:   []*fftypes.DataRef{{ID: fftypes.NewUUID()}},
	}, false)
	assert.Regexp(t, "FF10139", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMessageDataDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	msgID := fftypes.NewUUID()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	err := s.updateMessageDataRefs(context.Background(), &txWrapper{sqlTX: tx}, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: msgID},
	}, true)
	assert.Regexp(t, "FF10118", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMessageDataAddFail(t *testing.T) {
	s, mock := newMockProvider().init()
	msgID := fftypes.NewUUID()
	dataID := fftypes.NewUUID()
	dataHash := fftypes.NewRandB32()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.updateMessageDataRefs(context.Background(), &txWrapper{sqlTX: tx}, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: msgID},
		Data:   []*fftypes.DataRef{{ID: dataID, Hash: dataHash}},
	}, false)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadMessageDataRefsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	msgID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.loadDataRefs(context.Background(), []*fftypes.Message{
		{
			Header: fftypes.MessageHeader{ID: msgID},
		},
	})
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadMessageDataRefsScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	msgID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id"}).AddRow("only one"))
	err := s.loadDataRefs(context.Background(), []*fftypes.Message{
		{
			Header: fftypes.MessageHeader{ID: msgID},
		},
	})
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadMessageDataRefsEmpty(t *testing.T) {
	s, mock := newMockProvider().init()
	msgID := fftypes.NewUUID()
	msg := &fftypes.Message{Header: fftypes.MessageHeader{ID: msgID}}
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"data_id", "data_hash"}))
	err := s.loadDataRefs(context.Background(), []*fftypes.Message{msg})
	assert.NoError(t, err)
	assert.Equal(t, fftypes.DataRefs{}, msg.Data)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	msgID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetMessageByID(context.Background(), msgID)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	msgID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	msg, err := s.GetMessageByID(context.Background(), msgID)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	msgID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetMessageByID(context.Background(), msgID)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageByIDLoadRefsFail(t *testing.T) {
	s, mock := newMockProvider().init()
	msgID := fftypes.NewUUID()
	b32 := fftypes.NewRandB32()
	cols := append([]string{}, msgColumns...)
	cols = append(cols, "id()")
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows(cols).
		AddRow(msgID.String(), nil, fftypes.MessageTypeBroadcast, "0x12345", 0, "ns1", "t1", "c1", nil, b32.String(), b32.String(), b32.String(), true, true, 0, "pin", nil, false, 0))
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetMessageByID(context.Background(), msgID)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	_, _, err := s.GetMessages(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestGetMessagesQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetMessages(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesForDataBadQuery(t *testing.T) {
	s, mock := newMockProvider().init()
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("!wrong", "")
	_, _, err := s.GetMessagesForData(context.Background(), fftypes.NewUUID(), f)
	assert.Regexp(t, "FF10148", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetMessages(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesLoadRefsFail(t *testing.T) {
	s, mock := newMockProvider().init()
	msgID := fftypes.NewUUID()
	b32 := fftypes.NewRandB32()
	cols := append([]string{}, msgColumns...)
	cols = append(cols, "id()")
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows(cols).
		AddRow(msgID.String(), nil, fftypes.MessageTypeBroadcast, "0x12345", 0, "ns1", "t1", "c1", nil, b32.String(), b32.String(), b32.String(), true, true, 0, "pin", nil, false, 0))
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.MessageQueryFactory.NewFilter(context.Background()).Gt("confirmed", "0")
	_, _, err := s.GetMessages(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageRefsBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	_, _, err := s.GetMessageRefs(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestGetMessageRefsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetMessageRefs(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageRefsReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("id", "")
	_, _, err := s.GetMessageRefs(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageUpdateBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.MessageQueryFactory.NewUpdate(context.Background()).Set("id", "anything")
	err := s.UpdateMessage(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err)
}

func TestMessageUpdateBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	u := database.MessageQueryFactory.NewUpdate(context.Background()).Set("id", map[bool]bool{true: false})
	err := s.UpdateMessage(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestMessagesUpdateBuildFilterFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	f := database.MessageQueryFactory.NewFilter(context.Background()).Eq("id", map[bool]bool{true: false})
	u := database.MessageQueryFactory.NewUpdate(context.Background()).Set("type", fftypes.MessageTypeBroadcast)
	err := s.UpdateMessages(context.Background(), f, u)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestMessageUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.MessageQueryFactory.NewUpdate(context.Background()).Set("group", fftypes.NewRandB32())
	err := s.UpdateMessage(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err)
}
