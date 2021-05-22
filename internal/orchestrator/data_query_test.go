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

	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetNamespace(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	mp.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	_, err := o.GetNamespace(context.Background(), "ns1")
	assert.NoError(t, err)
}

func TestGetTransactionByID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetTransactionByID", mock.Anything, u).Return(nil, nil)
	_, err := o.GetTransactionByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetTransactionByIDBadID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.GetTransactionByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetNamespaces(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	mp.On("GetNamespaces", mock.Anything, mock.Anything).Return([]*fftypes.Namespace{}, nil)
	fb := database.NamespaceQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("name", "ns1"))
	_, err := o.GetNamespaces(context.Background(), f)
	assert.NoError(t, err)
}

func TestGetTransactions(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetTransactions", mock.Anything, mock.Anything).Return([]*fftypes.Transaction{}, nil)
	fb := database.TransactionQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := o.GetTransactions(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetMessageByID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetMessageByID", mock.Anything, u).Return(nil, nil)
	_, err := o.GetMessageByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetMessageByIDBadID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.GetMessageByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetMessages(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{}, nil)
	fb := database.MessageQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := o.GetMessages(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetMessagesForData(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetMessagesForData", mock.Anything, u, mock.Anything).Return([]*fftypes.Message{}, nil)
	fb := database.MessageQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := o.GetMessagesForData(context.Background(), "ns1", u.String(), f)
	assert.NoError(t, err)
}

func TestGetMessagesForDataBadID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	f := database.MessageQueryFactory.NewFilter(context.Background()).And()
	_, err := o.GetMessagesForData(context.Background(), "!wrong", "!bad", f)
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetMessageOperations(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	mp.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{}, nil)
	fb := database.OperationQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("type", fftypes.OpTypeBlockchainBatchPin))
	_, err := o.GetMessageOperations(context.Background(), "ns1", fftypes.NewUUID().String(), f)
	assert.NoError(t, err)
}

func TestGetMessageEventsOk(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID()},
			{ID: fftypes.NewUUID()},
		},
	}
	mp.On("GetMessageByID", mock.Anything, mock.Anything).Return(msg, nil)
	mp.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	fb := database.EventQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("type", fftypes.EventTypeDataArrivedBroadcast))
	_, err := o.GetMessageEvents(context.Background(), "ns1", fftypes.NewUUID().String(), f)
	assert.NoError(t, err)
	calculatedFilter, err := mp.Calls[1].Arguments[1].(database.Filter).Finalize()
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(
		`( type == 'DataArrivedBroadcast' ) && ( reference IN ['%s','%s','%s'] )`,
		msg.Header.ID, msg.Data[0].ID, msg.Data[1].ID,
	), calculatedFilter.String())
	assert.NoError(t, err)
}

func TestGetMessageEventsBadMsgID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	fb := database.EventQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("type", fftypes.EventTypeDataArrivedBroadcast))
	mp.On("GetMessageByID", mock.Anything, mock.Anything).Return(nil, nil)
	ev, err := o.GetMessageEvents(context.Background(), "ns1", fftypes.NewUUID().String(), f)
	assert.NoError(t, err)
	assert.Nil(t, ev)
}

func TestGetBatchByID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetBatchByID", mock.Anything, u).Return(nil, nil)
	_, err := o.GetBatchByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetBatchByIDBadID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.GetBatchByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetBatches(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetBatches", mock.Anything, mock.Anything).Return([]*fftypes.Batch{}, nil)
	fb := database.BatchQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := o.GetBatches(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetDataByID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetDataByID", mock.Anything, u).Return(nil, nil)
	_, err := o.GetDataByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetDataByIDBadID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.GetDataByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetData(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetData", mock.Anything, mock.Anything).Return([]*fftypes.Data{}, nil)
	fb := database.DataQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := o.GetData(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetDataDefsByID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetDataDefinitionByID", mock.Anything, u).Return(nil, nil)
	_, err := o.GetDataDefinitionByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetDataDefsByIDBadID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.GetDataDefinitionByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetOperationByID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetOperationByID", mock.Anything, u).Return(nil, nil)
	_, err := o.GetOperationByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetOperationIDBadID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.GetOperationByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetEventByID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetEventByID", mock.Anything, u).Return(nil, nil)
	_, err := o.GetEventByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetEventIDBadID(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.GetEventByID(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetDataDefinitions(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetDataDefinitions", mock.Anything, mock.Anything).Return([]*fftypes.DataDefinition{}, nil)
	fb := database.DataDefinitionQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := o.GetDataDefinitions(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetOperations(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{}, nil)
	fb := database.OperationQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := o.GetOperations(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetEvents(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	fb := database.EventQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, err := o.GetEvents(context.Background(), "ns1", f)
	assert.NoError(t, err)
}
