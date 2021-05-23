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

package orchestrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
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
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetNamespaces(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetNamespaces", mock.Anything, mock.Anything).Return([]*fftypes.Namespace{}, nil)
	fb := database.NamespaceQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("name", "ns1"))
	_, err := or.GetNamespaces(context.Background(), f)
	assert.NoError(t, err)
}

func TestGetTransactions(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetTransactions", mock.Anything, mock.Anything).Return([]*fftypes.Transaction{}, nil)
	fb := database.TransactionQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := or.GetTransactions(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetMessageByID(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetMessageByID", mock.Anything, u).Return(nil, nil)
	_, err := or.GetMessageByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetMessageByIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	_, err := or.GetMessageByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetMessages(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{}, nil)
	fb := database.MessageQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := or.GetMessages(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetMessagesForData(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetMessagesForData", mock.Anything, u, mock.Anything).Return([]*fftypes.Message{}, nil)
	fb := database.MessageQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := or.GetMessagesForData(context.Background(), "ns1", u.String(), f)
	assert.NoError(t, err)
}

func TestGetMessagesForDataBadID(t *testing.T) {
	or := newTestOrchestrator()
	f := database.MessageQueryFactory.NewFilter(context.Background()).And()
	_, err := or.GetMessagesForData(context.Background(), "!wrong", "!bad", f)
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetMessageOperations(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{}, nil)
	fb := database.OperationQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("type", fftypes.OpTypeBlockchainBatchPin))
	_, err := or.GetMessageOperations(context.Background(), "ns1", fftypes.NewUUID().String(), f)
	assert.NoError(t, err)
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
	or.mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	fb := database.EventQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("type", fftypes.EventTypeDataArrivedBroadcast))
	_, err := or.GetMessageEvents(context.Background(), "ns1", fftypes.NewUUID().String(), f)
	assert.NoError(t, err)
	calculatedFilter, err := or.mdi.Calls[1].Arguments[1].(database.Filter).Finalize()
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(
		`( type == 'DataArrivedBroadcast' ) && ( reference IN ['%s','%s','%s'] )`,
		msg.Header.ID, msg.Data[0].ID, msg.Data[1].ID,
	), calculatedFilter.String())
	assert.NoError(t, err)
}

func TestGetMessageEventsBadMsgID(t *testing.T) {
	or := newTestOrchestrator()
	fb := database.EventQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("type", fftypes.EventTypeDataArrivedBroadcast))
	or.mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(nil, nil)
	ev, err := or.GetMessageEvents(context.Background(), "ns1", fftypes.NewUUID().String(), f)
	assert.NoError(t, err)
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
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetBatches(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetBatches", mock.Anything, mock.Anything).Return([]*fftypes.Batch{}, nil)
	fb := database.BatchQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := or.GetBatches(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetDataByID(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetDataByID", mock.Anything, u).Return(nil, nil)
	_, err := or.GetDataByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetDataByIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	_, err := or.GetDataByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetData(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetData", mock.Anything, mock.Anything).Return([]*fftypes.Data{}, nil)
	fb := database.DataQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := or.GetData(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetDataDefsByID(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetDataDefinitionByID", mock.Anything, u).Return(nil, nil)
	_, err := or.GetDataDefinitionByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetDataDefsByIDBadID(t *testing.T) {
	or := newTestOrchestrator()
	_, err := or.GetDataDefinitionByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
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
	assert.Regexp(t, "FF10142", err.Error())
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
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetDataDefinitions(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetDataDefinitions", mock.Anything, mock.Anything).Return([]*fftypes.DataDefinition{}, nil)
	fb := database.DataDefinitionQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := or.GetDataDefinitions(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetOperations(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{}, nil)
	fb := database.OperationQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := or.GetOperations(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetEvents(t *testing.T) {
	or := newTestOrchestrator()
	u := fftypes.NewUUID()
	or.mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	fb := database.EventQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := or.GetEvents(context.Background(), "ns1", f)
	assert.NoError(t, err)
}
