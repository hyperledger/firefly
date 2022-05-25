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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetNamespace(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	or.mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	_, err := or.GetNamespace(context.Background(), "ns1")
	assert.NoError(t, err)
}

func TestGetTransactionByID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetTransactionByID", mock.Anything, u).Return(nil, nil)
	_, err := or.GetTransactionByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetTransactionByIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	_, err := or.GetTransactionByID(context.Background(), "", "")
	assert.Regexp(t, "FF00138", err)
}

func TestGetTransactionOperationsOk(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*core.Operation{}, nil, nil)
	_, _, err := or.GetTransactionOperations(context.Background(), "ns1", fftypes.NewUUID().String())
	assert.NoError(t, err)
}

func TestGetTransactionOperationBadID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	_, _, err := or.GetTransactionOperations(context.Background(), "ns1", "")
	assert.Regexp(t, "FF00138", err)
}

func TestGetNamespaces(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	or.mdi.On("GetNamespaces", mock.Anything, mock.Anything).Return([]*core.Namespace{}, nil, nil)
	fb := database.NamespaceQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("name", "ns1"))
	_, _, err := or.GetNamespaces(context.Background(), f)
	assert.NoError(t, err)
}

func TestGetTransactions(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetTransactions", mock.Anything, mock.Anything).Return([]*core.Transaction{}, nil, nil)
	fb := database.TransactionQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetTransactions(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetMessageByIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	_, err := or.GetMessageByID(context.Background(), "", "")
	assert.Regexp(t, "FF00138", err)
}

func TestGetMessageByIDWrongNSReturned(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	msgID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(&core.Message{
		Header: core.MessageHeader{
			Namespace: "ns2",
		},
	}, nil)
	_, err := or.GetMessageByID(context.Background(), "ns1", msgID.String())
	assert.Regexp(t, "FF10109", err)
}

func TestGetMessageByIDNoValuesOk(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	msgID := fftypes.NewUUID()
	msg := &core.Message{
		Header: core.MessageHeader{
			Namespace: "ns1",
			ID:        msgID,
		},
		Data: core.DataRefs{
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
	or.databases["database_0"] = or.mdi
	msg := &core.Message{
		Header: core.MessageHeader{
			Namespace: "ns1",
			ID:        msgID,
		},
		Data: core.DataRefs{
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
		},
	}
	or.mdi.On("GetMessageByID", mock.Anything, mock.MatchedBy(func(u *fftypes.UUID) bool { return u.Equals(msgID) })).Return(msg, nil)
	or.mdm.On("GetMessageDataCached", mock.Anything, mock.Anything).Return(core.DataArray{
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
	or.databases["database_0"] = or.mdi
	msgID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, err := or.GetMessageByIDWithData(context.Background(), "ns1", msgID.String())
	assert.EqualError(t, err, "pop")
}

func TestGetMessageByIDWithDataFail(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	msgID := fftypes.NewUUID()
	msg := &core.Message{
		Header: core.MessageHeader{
			Namespace: "ns1",
			ID:        msgID,
		},
		Data: core.DataRefs{
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
		},
	}
	or.mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(msg, nil)
	or.mdm.On("GetMessageDataCached", mock.Anything, mock.Anything).Return(nil, false, fmt.Errorf("pop"))

	_, err := or.GetMessageByIDWithData(context.Background(), "ns1", msgID.String())
	assert.EqualError(t, err, "pop")
}

func TestGetMessages(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetMessages", mock.Anything, mock.Anything).Return([]*core.Message{}, nil, nil)
	fb := database.MessageQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetMessages(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetMessagesWithDataFailMsg(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	or.mdi.On("GetMessages", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))
	fb := database.MessageQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", fftypes.NewUUID()))
	_, _, err := or.GetMessagesWithData(context.Background(), "ns1", f)
	assert.EqualError(t, err, "pop")
}

func TestGetMessagesWithDataOk(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	msgID := fftypes.NewUUID()
	msg := &core.Message{
		Header: core.MessageHeader{
			ID: msgID,
		},
		Data: core.DataRefs{},
	}
	or.mdi.On("GetMessages", mock.Anything, mock.Anything).Return([]*core.Message{msg}, nil, nil)
	fb := database.MessageQueryFactory.NewFilter(context.Background())
	or.mdm.On("GetMessageDataCached", mock.Anything, mock.Anything).Return(core.DataArray{}, true, nil)
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetMessagesWithData(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetMessagesWithDataFail(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	msgID := fftypes.NewUUID()
	msg := &core.Message{
		Header: core.MessageHeader{
			ID: msgID,
		},
		Data: core.DataRefs{},
	}
	or.mdi.On("GetMessages", mock.Anything, mock.Anything).Return([]*core.Message{msg}, nil, nil)
	fb := database.MessageQueryFactory.NewFilter(context.Background())
	or.mdm.On("GetMessageDataCached", mock.Anything, mock.Anything).Return(nil, true, fmt.Errorf("pop"))
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetMessagesWithData(context.Background(), "ns1", f)
	assert.EqualError(t, err, "pop")
}

func TestGetMessagesForData(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetMessagesForData", mock.Anything, u, mock.Anything).Return([]*core.Message{}, nil, nil)
	fb := database.MessageQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetMessagesForData(context.Background(), "ns1", u.String(), f)
	assert.NoError(t, err)
}

func TestGetMessagesForDataBadID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	f := database.MessageQueryFactory.NewFilter(context.Background()).And()
	_, _, err := or.GetMessagesForData(context.Background(), "!wrong", "!bad", f)
	assert.Regexp(t, "FF00138", err)
}

func TestGetMessageTransactionOk(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	msgID := fftypes.NewUUID()
	batchID := fftypes.NewUUID()
	txID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&core.Message{
		BatchID: batchID,
		Header: core.MessageHeader{
			Namespace: "ns1",
			TxType:    core.TransactionTypeBatchPin,
		},
	}, nil)
	or.mdi.On("GetBatchByID", mock.Anything, batchID).Return(&core.BatchPersisted{
		TX: core.TransactionRef{
			Type: core.TransactionTypeBatchPin,
			ID:   txID,
		},
	}, nil)
	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(&core.Transaction{
		ID: txID,
	}, nil)
	tx, err := or.GetMessageTransaction(context.Background(), "ns1", msgID.String())
	assert.NoError(t, err)
	assert.Equal(t, *txID, *tx.ID)
	or.mdi.AssertExpectations(t)
}

func TestGetMessageTransactionOperations(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	msgID := fftypes.NewUUID()
	batchID := fftypes.NewUUID()
	txID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&core.Message{
		BatchID: batchID,
		Header: core.MessageHeader{
			Namespace: "ns1",
			TxType:    core.TransactionTypeBatchPin,
		},
	}, nil)
	or.mdi.On("GetBatchByID", mock.Anything, batchID).Return(&core.BatchPersisted{
		TX: core.TransactionRef{
			Type: core.TransactionTypeBatchPin,
			ID:   txID,
		},
	}, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*core.Operation{}, nil, nil)
	ops, _, err := or.GetMessageOperations(context.Background(), "ns1", msgID.String())
	assert.NoError(t, err)
	assert.Len(t, ops, 0)
	or.mdi.AssertExpectations(t)
}

func TestGetMessageTransactionOperationsNoTX(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	msgID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&core.Message{
		Header: core.MessageHeader{
			Namespace: "ns1",
		},
	}, nil, nil)
	_, _, err := or.GetMessageOperations(context.Background(), "ns1", msgID.String())
	assert.Regexp(t, "FF10207", err)
	or.mdi.AssertExpectations(t)
}

func TestGetMessageTransactionNoBatchTX(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	msgID := fftypes.NewUUID()
	batchID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&core.Message{
		BatchID: batchID,
		Header: core.MessageHeader{
			Namespace: "ns1",
			TxType:    core.TransactionTypeBatchPin,
		},
	}, nil)
	or.mdi.On("GetBatchByID", mock.Anything, batchID).Return(&core.BatchPersisted{}, nil)
	_, err := or.GetMessageTransaction(context.Background(), "ns1", msgID.String())
	assert.Regexp(t, "FF10210", err)
}

func TestGetMessageTransactionNoBatch(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	msgID := fftypes.NewUUID()
	batchID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&core.Message{
		BatchID: batchID,
		Header: core.MessageHeader{
			Namespace: "ns1",
			TxType:    core.TransactionTypeBatchPin,
		},
	}, nil)
	or.mdi.On("GetBatchByID", mock.Anything, batchID).Return(nil, nil)
	_, err := or.GetMessageTransaction(context.Background(), "ns1", msgID.String())
	assert.Regexp(t, "FF10209", err)
}

func TestGetMessageTransactionBatchLookupErr(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	msgID := fftypes.NewUUID()
	batchID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&core.Message{
		BatchID: batchID,
		Header: core.MessageHeader{
			Namespace: "ns1",
			TxType:    core.TransactionTypeBatchPin,
		},
	}, nil)
	or.mdi.On("GetBatchByID", mock.Anything, batchID).Return(nil, fmt.Errorf("pop"))
	_, err := or.GetMessageTransaction(context.Background(), "ns1", msgID.String())
	assert.Regexp(t, "pop", err)
}

func TestGetMessageTransactionNoBatchID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	msgID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&core.Message{
		Header: core.MessageHeader{
			Namespace: "ns1",
			TxType:    core.TransactionTypeBatchPin,
		},
	}, nil)
	_, err := or.GetMessageTransaction(context.Background(), "ns1", msgID.String())
	assert.Regexp(t, "FF10208", err)
}

func TestGetMessageTransactionNoTx(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	msgID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(&core.Message{
		Header: core.MessageHeader{
			Namespace: "ns1",
		},
	}, nil)
	_, err := or.GetMessageTransaction(context.Background(), "ns1", msgID.String())
	assert.Regexp(t, "FF10207", err)
}

func TestGetMessageTransactionMessageNotFound(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	msgID := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, msgID).Return(nil, nil)
	_, err := or.GetMessageTransaction(context.Background(), "ns1", msgID.String())
	assert.Regexp(t, "FF10109", err)
}

func TestGetMessageData(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	msg := &core.Message{
		Header: core.MessageHeader{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
		},
		Data: core.DataRefs{
			{ID: fftypes.NewUUID()},
			{ID: fftypes.NewUUID()},
		},
	}
	or.mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(msg, nil)
	or.mdm.On("GetMessageDataCached", mock.Anything, mock.Anything).Return(core.DataArray{}, true, nil)
	_, err := or.GetMessageData(context.Background(), "ns1", fftypes.NewUUID().String())
	assert.NoError(t, err)
}

func TestGetMessageDataBadMsg(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	or.mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(nil, nil)
	_, err := or.GetMessageData(context.Background(), "ns1", fftypes.NewUUID().String())
	assert.Regexp(t, "FF10109", err)
}

func TestGetMessageEventsOk(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	msg := &core.Message{
		Header: core.MessageHeader{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
		},
		Data: core.DataRefs{
			{ID: fftypes.NewUUID()},
			{ID: fftypes.NewUUID()},
		},
	}
	or.mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(msg, nil)
	or.mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*core.Event{}, nil, nil)
	fb := database.EventQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("type", core.EventTypeMessageConfirmed))
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
	or.databases["database_0"] = or.mdi
	fb := database.EventQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("type", core.EventTypeMessageConfirmed))
	or.mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(nil, nil)
	ev, _, err := or.GetMessageEvents(context.Background(), "ns1", fftypes.NewUUID().String(), f)
	assert.Regexp(t, "FF10109", err)
	assert.Nil(t, ev)
}

func TestGetBatchByID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetBatchByID", mock.Anything, u).Return(&core.BatchPersisted{
		BatchHeader: core.BatchHeader{
			Namespace: "ns1",
		},
	}, nil)
	_, err := or.GetBatchByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetBatchByIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	_, err := or.GetBatchByID(context.Background(), "", "")
	assert.Regexp(t, "FF00138", err)
}

func TestGetBatches(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetBatches", mock.Anything, mock.Anything).Return([]*core.BatchPersisted{}, nil, nil)
	fb := database.BatchQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetBatches(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetDataByID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetDataByID", mock.Anything, u, true).Return(&core.Data{
		Namespace: "ns1",
	}, nil)
	_, err := or.GetDataByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetDataByIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	_, err := or.GetDataByID(context.Background(), "", "")
	assert.Regexp(t, "FF00138", err)
}

func TestGetData(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetData", mock.Anything, mock.Anything).Return(core.DataArray{}, nil, nil)
	fb := database.DataQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetData(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetDatatypeByID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetDatatypeByID", mock.Anything, u).Return(&core.Datatype{
		Namespace: "ns1",
	}, nil)
	_, err := or.GetDatatypeByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetDatatypeByIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	_, err := or.GetDatatypeByID(context.Background(), "", "")
	assert.Regexp(t, "FF00138", err)
}

func TestGetDatatypeByName(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	or.mdi.On("GetDatatypeByName", context.Background(), "ns1", "dt", "1").Return(&core.Datatype{
		Namespace: "ns1",
	}, nil)
	_, err := or.GetDatatypeByName(context.Background(), "ns1", "dt", "1")
	assert.NoError(t, err)
}

func TestGetDatatypeByNameBadNamespace(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	_, err := or.GetDatatypeByName(context.Background(), "", "", "")
	assert.Regexp(t, "FF00140", err)
}

func TestGetDatatypeByNameBadName(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	_, err := or.GetDatatypeByName(context.Background(), "ns1", "", "")
	assert.Regexp(t, "FF00140", err)
}

func TestGetOperationByID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetOperationByID", mock.Anything, u).Return(nil, nil)
	_, err := or.GetOperationByID(context.Background(), u.String())
	assert.NoError(t, err)
}

func TestGetOperationByIDNamespaced(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetOperationByID", mock.Anything, u).Return(&core.Operation{
		Namespace: "ns1",
	}, nil)
	_, err := or.GetOperationByIDNamespaced(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetOperationIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	_, err := or.GetOperationByID(context.Background(), "")
	assert.Regexp(t, "FF00138", err)
}

func TestGetOperationIDNamespacedBadID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	_, err := or.GetOperationByIDNamespaced(context.Background(), "", "")
	assert.Regexp(t, "FF00138", err)
}

func TestGetEventByID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetEventByID", mock.Anything, u).Return(&core.Event{
		Namespace: "ns1",
	}, nil)
	_, err := or.GetEventByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetEventIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	_, err := or.GetEventByID(context.Background(), "", "")
	assert.Regexp(t, "FF00138", err)
}

func TestGetDatatypes(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetDatatypes", mock.Anything, mock.Anything).Return([]*core.Datatype{}, nil, nil)
	fb := database.DatatypeQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetDatatypes(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetOperations(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*core.Operation{}, nil, nil)
	fb := database.OperationQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetOperations(context.Background(), f)
	assert.NoError(t, err)
}

func TestGetOperationsNamespaced(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*core.Operation{}, nil, nil)
	fb := database.OperationQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetOperationsNamespaced(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetEvents(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*core.Event{}, nil, nil)
	fb := database.EventQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetEvents(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetEventsWithReferencesFail(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetEvents", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))
	fb := database.EventQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetEventsWithReferences(context.Background(), "ns1", f)
	assert.EqualError(t, err, "pop")
}

func TestGetEventsWithReferences(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()
	ref2 := fftypes.NewUUID()
	ev2 := fftypes.NewUUID()
	ref3 := fftypes.NewUUID()
	ev3 := fftypes.NewUUID()

	blockchainEvent := &core.Event{
		ID:        ev1,
		Sequence:  10000001,
		Reference: ref1,
		Type:      core.EventTypeBlockchainEventReceived,
	}

	txEvent := &core.Event{
		ID:        ev2,
		Sequence:  10000002,
		Reference: ref2,
		Type:      core.EventTypeTransactionSubmitted,
	}

	msgEvent := &core.Event{
		ID:        ev3,
		Sequence:  10000003,
		Reference: ref3,
		Type:      core.EventTypeMessageConfirmed,
	}

	or.mth.On("EnrichEvent", mock.Anything, blockchainEvent).Return(&core.EnrichedEvent{
		Event: *blockchainEvent,
		BlockchainEvent: &core.BlockchainEvent{
			ID: ref1,
		},
	}, nil)

	or.mth.On("EnrichEvent", mock.Anything, txEvent).Return(&core.EnrichedEvent{
		Event: *txEvent,
		Transaction: &core.Transaction{
			ID: ref2,
		},
	}, nil)

	or.mth.On("EnrichEvent", mock.Anything, msgEvent).Return(&core.EnrichedEvent{
		Event: *msgEvent,
		Message: &core.Message{
			Header: core.MessageHeader{
				ID: ref3,
			},
		},
	}, nil)

	or.mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*core.Event{
		blockchainEvent,
		txEvent,
		msgEvent,
	}, nil, nil)
	fb := database.EventQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetEventsWithReferences(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetEventsWithReferencesEnrichFail(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()

	or.mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*core.Event{{ID: fftypes.NewUUID()}}, nil, nil)
	or.mth.On("EnrichEvent", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	fb := database.EventQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := or.GetEventsWithReferences(context.Background(), "ns1", f)
	assert.EqualError(t, err, "pop")
}

func TestGetBlockchainEventByID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi

	id := fftypes.NewUUID()
	or.mdi.On("GetBlockchainEventByID", context.Background(), id).Return(&core.BlockchainEvent{
		Namespace: "ns1",
	}, nil)

	_, err := or.GetBlockchainEventByID(context.Background(), "ns1", id.String())
	assert.NoError(t, err)
}

func TestGetBlockchainEventByIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	_, err := or.GetBlockchainEventByID(context.Background(), "ns1", "")
	assert.Regexp(t, "FF00138", err)
}

func TestGetBlockchainEvents(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi

	or.mdi.On("GetBlockchainEvents", context.Background(), mock.Anything).Return(nil, nil, nil)

	f := database.ContractListenerQueryFactory.NewFilter(context.Background())
	_, _, err := or.GetBlockchainEvents(context.Background(), "ns", f.And())
	assert.NoError(t, err)
}

func TestGetTransactionBlockchainEventsOk(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return([]*core.BlockchainEvent{}, nil, nil)
	_, _, err := or.GetTransactionBlockchainEvents(context.Background(), "ns1", fftypes.NewUUID().String())
	assert.NoError(t, err)
}

func TestGetTransactionBlockchainEventsBadID(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	_, _, err := or.GetTransactionBlockchainEvents(context.Background(), "ns1", "")
	assert.Regexp(t, "FF00138", err)
}

func TestGetPins(t *testing.T) {
	or := newTestOrchestrator()
	or.databases["database_0"] = or.mdi
	u := fftypes.NewUUID()
	or.mdi.On("GetPins", mock.Anything, mock.Anything).Return([]*core.Pin{}, nil, nil)
	fb := database.PinQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("hash", u))
	_, _, err := or.GetPins(context.Background(), f)
	assert.NoError(t, err)
}
