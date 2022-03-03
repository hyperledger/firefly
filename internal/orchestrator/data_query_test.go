// Copyright © 2022 Kaleido, Inc.
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

package orchestrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetNamespace(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	_, err := or.GetNamespace(context.Background(), "ns1")
	assert.NoError(t, err)
}

func TestGetTransactionByID(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetTransactionByID", mock.Anything, u).Return(nil, nil)
	_, err := or.GetTransactionByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetTransactionByIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	_, err := or.GetTransactionByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err)
}

func TestGetTransactionOperationsOk(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{}, nil, nil)
	_, _, err := or.GetTransactionOperations(context.Background(), "ns1", fftypes.NewUUID().String())
	assert.NoError(t, err)
}

func TestGetTransactionOperationBadID(t *testing.T) {
	or := newTestOrchestrator()
	_, _, err := or.GetTransactionOperations(context.Background(), "ns1", "")
	assert.Regexp(t, "FF10142", err)
}

func TestGetNamespaces(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetNamespaces", mock.Anything, mock.Anything).Return([]*fftypes.Namespace{}, nil, nil)
	fb := database.NamespaceQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("name", "ns1"))
	_, _, err := or.GetNamespaces(context.Background(), f)
	assert.NoError(t, err)
}

func TestGetTransactions(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetTransactions", mock.Anything, mock.Anything).Return([]*fftypes.Transaction{}, nil, nil)
	fb := database.TransactionQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetTransactions(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetMessageByIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	_, err := or.GetMessageByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err)
}

func TestGetMessageByIDNoValuesOk(t *testing.T) {
	or := newTestOrchestrator()
	msgID := fftypes.NewUUID()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: msgID,
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
		},
	}
	or.mdi.On("GetMessageByID", mock.Anything, mock.MatchedBy(func(u *fftypes.UUID) bool { return u.Equals(msgID) })).Return(msg, nil)

	msgI, err := or.GetMessageByID(context.Background(), "ns1", msgID.String())
	assert.NoError(t, err)
	assert.NotNil(t, msgI.Data[0].ID)
	assert.NotNil(t, msgI.Data[0].Hash)
	assert.NotNil(t, msgI.Data[1].ID)
	assert.NotNil(t, msgI.Data[1].Hash)
}

func TestGetMessageByIDWithDataOk(t *testing.T) {
	or := newTestOrchestrator()
	msgID := fftypes.NewUUID()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: msgID,
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
		},
	}
	or.mdi.On("GetMessageByID", mock.Anything, mock.MatchedBy(func(u *fftypes.UUID) bool { return u.Equals(msgID) })).Return(msg, nil)
	or.mdm.On("GetMessageData", mock.Anything, mock.Anything, true).Return([]*fftypes.Data{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32(), Value: fftypes.JSONAnyPtr("{}")},
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32(), Value: fftypes.JSONAnyPtr("{}")},
	}, true, nil)

	msgI, err := or.GetMessageByIDWithData(context.Background(), "ns1", msgID.String())
	assert.NoError(t, err)
	assert.NotNil(t, msgI.InlineData[0].ID)
	assert.NotNil(t, msgI.InlineData[0].Hash)
	assert.NotNil(t, msgI.InlineData[0].Value)
	assert.NotNil(t, msgI.InlineData[1].ID)
	assert.NotNil(t, msgI.InlineData[1].Hash)
	assert.NotNil(t, msgI.InlineData[1].Value)
}

func TestGetMessageByIDWithDataMsgFail(t *testing.T) {
	or := newTestOrchestrator()
	msgID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, err := or.GetMessageByIDWithData(context.Background(), "ns1", msgID.String())
	assert.EqualError(t, err, "pop")
}

func TestGetMessageByIDWithDataFail(t *testing.T) {
	or := newTestOrchestrator()
	msgID := fftypes.NewUUID()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: msgID,
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
		},
	}
	or.mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(msg, nil)
	or.mdm.On("GetMessageData", mock.Anything, mock.Anything, true).Return(nil, false, fmt.Errorf("pop"))

	_, err := or.GetMessageByIDWithData(context.Background(), "ns1", msgID.String())
	assert.EqualError(t, err, "pop")
}

func TestGetMessages(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{}, nil, nil)
	fb := database.MessageQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetMessages(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetMessagesWithDataFailMsg(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetMessages", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))
	fb := database.MessageQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", fftypes.NewUUID()))
	_, _, err := or.GetMessagesWithData(context.Background(), "ns1", f)
	assert.EqualError(t, err, "pop")
}

func TestGetMessagesWithDataOk(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	msgID := fftypes.NewUUID()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: msgID,
		},
		Data: fftypes.DataRefs{},
	}
	or.mdi.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{msg}, nil, nil)
	fb := database.MessageQueryFactory.NewFilter(context.Background())
	or.mdm.On("GetMessageData", mock.Anything, mock.Anything, true).Return([]*fftypes.Data{}, true, nil)
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetMessagesWithData(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetMessagesWithDataFail(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	msgID := fftypes.NewUUID()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: msgID,
		},
		Data: fftypes.DataRefs{},
	}
	or.mdi.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{msg}, nil, nil)
	fb := database.MessageQueryFactory.NewFilter(context.Background())
	or.mdm.On("GetMessageData", mock.Anything, mock.Anything, true).Return(nil, true, fmt.Errorf("pop"))
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetMessagesWithData(context.Background(), "ns1", f)
	assert.EqualError(t, err, "pop")
}

func TestGetMessagesForData(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetMessagesForData", mock.Anything, u, mock.Anything).Return([]*fftypes.Message{}, nil, nil)
	fb := database.MessageQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetMessagesForData(context.Background(), "ns1", u.String(), f)
	assert.NoError(t, err)
}

func TestGetMessagesForDataBadID(t *testing.T) {
	or := newTestOrchestrator()
	f := database.MessageQueryFactory.NewFilter(context.Background()).And()
	_, _, err := or.GetMessagesForData(context.Background(), "!wrong", "!bad", f)
	assert.Regexp(t, "FF10142", err)
}

func TestGetMessageTransactionOk(t *testing.T) {
	or := newTestOrchestrator()
	msgID := fftypes.NewUUID()
	batchID := fftypes.NewUUID()
	txID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&fftypes.Message{
		BatchID: batchID,
		Header: fftypes.MessageHeader{
			TxType: fftypes.TransactionTypeBatchPin,
		},
	}, nil)
	or.mdi.On("GetBatchByID", mock.Anything, batchID).Return(&fftypes.Batch{
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   txID,
			},
		},
	}, nil)
	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(&fftypes.Transaction{
		ID: txID,
	}, nil)
	tx, err := or.GetMessageTransaction(context.Background(), "ns1", msgID.String())
	assert.NoError(t, err)
	assert.Equal(t, *txID, *tx.ID)
	or.mdi.AssertExpectations(t)
}

func TestGetMessageTransactionOperations(t *testing.T) {
	or := newTestOrchestrator()
	msgID := fftypes.NewUUID()
	batchID := fftypes.NewUUID()
	txID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&fftypes.Message{
		BatchID: batchID,
		Header: fftypes.MessageHeader{
			TxType: fftypes.TransactionTypeBatchPin,
		},
	}, nil)
	or.mdi.On("GetBatchByID", mock.Anything, batchID).Return(&fftypes.Batch{
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   txID,
			},
		},
	}, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{}, nil, nil)
	ops, _, err := or.GetMessageOperations(context.Background(), "ns1", msgID.String())
	assert.NoError(t, err)
	assert.Len(t, ops, 0)
	or.mdi.AssertExpectations(t)
}

func TestGetMessageTransactionOperationsNoTX(t *testing.T) {
	or := newTestOrchestrator()
	msgID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&fftypes.Message{}, nil, nil)
	_, _, err := or.GetMessageOperations(context.Background(), "ns1", msgID.String())
	assert.Regexp(t, "FF10207", err)
	or.mdi.AssertExpectations(t)
}

func TestGetMessageTransactionNoBatchTX(t *testing.T) {
	or := newTestOrchestrator()
	msgID := fftypes.NewUUID()
	batchID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&fftypes.Message{
		BatchID: batchID,
		Header: fftypes.MessageHeader{
			TxType: fftypes.TransactionTypeBatchPin,
		},
	}, nil)
	or.mdi.On("GetBatchByID", mock.Anything, batchID).Return(&fftypes.Batch{}, nil)
	_, err := or.GetMessageTransaction(context.Background(), "ns1", msgID.String())
	assert.Regexp(t, "FF10210", err)
}

func TestGetMessageTransactionNoBatch(t *testing.T) {
	or := newTestOrchestrator()
	msgID := fftypes.NewUUID()
	batchID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&fftypes.Message{
		BatchID: batchID,
		Header: fftypes.MessageHeader{
			TxType: fftypes.TransactionTypeBatchPin,
		},
	}, nil)
	or.mdi.On("GetBatchByID", mock.Anything, batchID).Return(nil, nil)
	_, err := or.GetMessageTransaction(context.Background(), "ns1", msgID.String())
	assert.Regexp(t, "FF10209", err)
}

func TestGetMessageTransactionBatchLookupErr(t *testing.T) {
	or := newTestOrchestrator()
	msgID := fftypes.NewUUID()
	batchID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&fftypes.Message{
		BatchID: batchID,
		Header: fftypes.MessageHeader{
			TxType: fftypes.TransactionTypeBatchPin,
		},
	}, nil)
	or.mdi.On("GetBatchByID", mock.Anything, batchID).Return(nil, fmt.Errorf("pop"))
	_, err := or.GetMessageTransaction(context.Background(), "ns1", msgID.String())
	assert.Regexp(t, "pop", err)
}

func TestGetMessageTransactionNoBatchID(t *testing.T) {
	or := newTestOrchestrator()
	msgID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&fftypes.Message{
		Header: fftypes.MessageHeader{
			TxType: fftypes.TransactionTypeBatchPin,
		},
	}, nil)
	_, err := or.GetMessageTransaction(context.Background(), "ns1", msgID.String())
	assert.Regexp(t, "FF10208", err)
}

func TestGetMessageTransactionNoTx(t *testing.T) {
	or := newTestOrchestrator()
	msgID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&fftypes.Message{}, nil)
	_, err := or.GetMessageTransaction(context.Background(), "ns1", msgID.String())
	assert.Regexp(t, "FF10207", err)
}

func TestGetMessageTransactionMessageNotFound(t *testing.T) {
	or := newTestOrchestrator()
	msgID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(nil, nil)
	_, err := or.GetMessageTransaction(context.Background(), "ns1", msgID.String())
	assert.Regexp(t, "FF10109", err)
}

func TestGetMessageData(t *testing.T) {
	or := newTestOrchestrator()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID()},
			{ID: fftypes.NewUUID()},
		},
	}
	or.mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(msg, nil)
	or.mdm.On("GetMessageData", mock.Anything, msg, true).Return([]*fftypes.Data{}, true, nil)
	_, err := or.GetMessageData(context.Background(), "ns1", fftypes.NewUUID().String())
	assert.NoError(t, err)
}

func TestGetMessageDataBadMsg(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(nil, nil)
	_, err := or.GetMessageData(context.Background(), "ns1", fftypes.NewUUID().String())
	assert.Regexp(t, "FF10109", err)
}

func TestGetMessageEventsOk(t *testing.T) {
	or := newTestOrchestrator()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID()},
			{ID: fftypes.NewUUID()},
		},
	}
	or.mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(msg, nil)
	or.mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil, nil)
	fb := database.EventQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("type", fftypes.EventTypeMessageConfirmed))
	_, _, err := or.GetMessageEvents(context.Background(), "ns1", fftypes.NewUUID().String(), f)
	assert.NoError(t, err)
	calculatedFilter, err := or.mdi.Calls[1].Arguments[1].(database.Filter).Finalize()
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(
		`( type == 'message_confirmed' ) && ( reference IN ['%s','%s','%s'] )`,
		msg.Header.ID, msg.Data[0].ID, msg.Data[1].ID,
	), calculatedFilter.String())
	assert.NoError(t, err)
}

func TestGetMessageEventsBadMsgID(t *testing.T) {
	or := newTestOrchestrator()
	fb := database.EventQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("type", fftypes.EventTypeMessageConfirmed))
	or.mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(nil, nil)
	ev, _, err := or.GetMessageEvents(context.Background(), "ns1", fftypes.NewUUID().String(), f)
	assert.Regexp(t, "FF10109", err)
	assert.Nil(t, ev)
}

func TestGetBatchByID(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetBatchByID", mock.Anything, u).Return(nil, nil)
	_, err := or.GetBatchByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetBatchByIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	_, err := or.GetBatchByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err)
}

func TestGetBatches(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetBatches", mock.Anything, mock.Anything).Return([]*fftypes.Batch{}, nil, nil)
	fb := database.BatchQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetBatches(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetDataByID(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetDataByID", mock.Anything, u, true).Return(nil, nil)
	_, err := or.GetDataByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetDataByIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	_, err := or.GetDataByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err)
}

func TestGetData(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetData", mock.Anything, mock.Anything).Return([]*fftypes.Data{}, nil, nil)
	fb := database.DataQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetData(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetDatatypeByID(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetDatatypeByID", mock.Anything, u).Return(nil, nil)
	_, err := or.GetDatatypeByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetDatatypeByIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	_, err := or.GetDatatypeByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err)
}

func TestGetDatatypeByName(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetDatatypeByName", context.Background(), "ns1", "dt", "1").Return(nil, nil)
	_, err := or.GetDatatypeByName(context.Background(), "ns1", "dt", "1")
	assert.NoError(t, err)
}

func TestGetDatatypeByNameBadNamespace(t *testing.T) {
	or := newTestOrchestrator()
	_, err := or.GetDatatypeByName(context.Background(), "", "", "")
	assert.Regexp(t, "FF10131", err)
}

func TestGetDatatypeByNameBadName(t *testing.T) {
	or := newTestOrchestrator()
	_, err := or.GetDatatypeByName(context.Background(), "ns1", "", "")
	assert.Regexp(t, "FF10131", err)
}

func TestGetOperationByID(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetOperationByID", mock.Anything, u).Return(nil, nil)
	_, err := or.GetOperationByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetOperationIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	_, err := or.GetOperationByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err)
}

func TestGetEventByID(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetEventByID", mock.Anything, u).Return(nil, nil)
	_, err := or.GetEventByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetEventIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	_, err := or.GetEventByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err)
}

func TestGetDatatypes(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetDatatypes", mock.Anything, mock.Anything).Return([]*fftypes.Datatype{}, nil, nil)
	fb := database.DatatypeQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetDatatypes(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetOperations(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{}, nil, nil)
	fb := database.OperationQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetOperations(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetEvents(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil, nil)
	fb := database.EventQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetEvents(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetBlockchainEventByID(t *testing.T) {
	or := newTestOrchestrator()

	id := fftypes.NewUUID()
	or.mdi.On("GetBlockchainEventByID", context.Background(), id).Return(&fftypes.BlockchainEvent{}, nil)

	_, err := or.GetBlockchainEventByID(context.Background(), id)
	assert.NoError(t, err)
}

func TestGetBlockchainEvents(t *testing.T) {
	or := newTestOrchestrator()

	or.mdi.On("GetBlockchainEvents", context.Background(), mock.Anything).Return(nil, nil, nil)

	f := database.ContractListenerQueryFactory.NewFilter(context.Background())
	_, _, err := or.GetBlockchainEvents(context.Background(), "ns", f.And())
	assert.NoError(t, err)
}

func TestGetTransactionBlockchainEventsOk(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return([]*fftypes.BlockchainEvent{}, nil, nil)
	_, _, err := or.GetTransactionBlockchainEvents(context.Background(), "ns1", fftypes.NewUUID().String())
	assert.NoError(t, err)
}

func TestGetTransactionBlockchainEventsBadID(t *testing.T) {
	or := newTestOrchestrator()
	_, _, err := or.GetTransactionBlockchainEvents(context.Background(), "ns1", "")
	assert.Regexp(t, "FF10142", err)
}
