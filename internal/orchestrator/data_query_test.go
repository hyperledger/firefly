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

	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
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

func TestGetTransactionById(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetTransactionById", mock.Anything, u).Return(nil, nil)
	_, err := o.GetTransactionById(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetTransactionByIdBadId(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.GetTransactionById(context.Background(), "", "")
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

func TestGetMessageById(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetMessageById", mock.Anything, u).Return(nil, nil)
	_, err := o.GetMessageById(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetMessageByIdBadId(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.GetMessageById(context.Background(), "", "")
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

func TestGetMessagesForDataBadId(t *testing.T) {
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
	mp.On("GetMessageById", mock.Anything, mock.Anything).Return(msg, nil)
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
	mp.On("GetMessageById", mock.Anything, mock.Anything).Return(nil, nil)
	ev, err := o.GetMessageEvents(context.Background(), "ns1", fftypes.NewUUID().String(), f)
	assert.NoError(t, err)
	assert.Nil(t, ev)
}

func TestGetBatchById(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetBatchById", mock.Anything, u).Return(nil, nil)
	_, err := o.GetBatchById(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetBatchByIdBadId(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.GetBatchById(context.Background(), "", "")
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

func TestGetDataById(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetDataById", mock.Anything, u).Return(nil, nil)
	_, err := o.GetDataById(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetDataByIdBadId(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.GetDataById(context.Background(), "", "")
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

func TestGetDataDefsById(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetDataDefinitionById", mock.Anything, u).Return(nil, nil)
	_, err := o.GetDataDefinitionById(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetDataDefsByIdBadId(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.GetDataDefinitionById(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetOperationById(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetOperationById", mock.Anything, u).Return(nil, nil)
	_, err := o.GetOperationById(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetOperationIdBadId(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.GetOperationById(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetEventById(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp
	u := fftypes.NewUUID()
	mp.On("GetEventById", mock.Anything, u).Return(nil, nil)
	_, err := o.GetEventById(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetEventIdBadId(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.GetEventById(context.Background(), "", "")
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
