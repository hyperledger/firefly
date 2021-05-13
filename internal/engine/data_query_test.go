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

package engine

import (
	"context"
	"testing"

	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetTransactionById(t *testing.T) {
	e := NewEngine().(*engine)
	mp := &databasemocks.Plugin{}
	e.database = mp
	u := fftypes.NewUUID()
	mp.On("GetTransactionById", mock.Anything, "ns1", u).Return(nil, nil)
	_, err := e.GetTransactionById(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetTransactionByIdBadId(t *testing.T) {
	e := NewEngine().(*engine)
	_, err := e.GetTransactionById(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetTransactions(t *testing.T) {
	e := NewEngine().(*engine)
	mp := &databasemocks.Plugin{}
	e.database = mp
	u := fftypes.NewUUID()
	mp.On("GetTransactions", mock.Anything, mock.Anything).Return([]*fftypes.Transaction{}, nil)
	fb := database.TransactionQueryFactory.NewFilter(context.Background(), 0)
	f := fb.And(fb.Eq("id", u))
	_, err := e.GetTransactions(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetMessageById(t *testing.T) {
	e := NewEngine().(*engine)
	mp := &databasemocks.Plugin{}
	e.database = mp
	u := fftypes.NewUUID()
	mp.On("GetMessageById", mock.Anything, "ns1", u).Return(nil, nil)
	_, err := e.GetMessageById(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetMessageByIdBadId(t *testing.T) {
	e := NewEngine().(*engine)
	_, err := e.GetMessageById(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetMessages(t *testing.T) {
	e := NewEngine().(*engine)
	mp := &databasemocks.Plugin{}
	e.database = mp
	u := fftypes.NewUUID()
	mp.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{}, nil)
	fb := database.MessageQueryFactory.NewFilter(context.Background(), 0)
	f := fb.And(fb.Eq("id", u))
	_, err := e.GetMessages(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetBatchById(t *testing.T) {
	e := NewEngine().(*engine)
	mp := &databasemocks.Plugin{}
	e.database = mp
	u := fftypes.NewUUID()
	mp.On("GetBatchById", mock.Anything, "ns1", u).Return(nil, nil)
	_, err := e.GetBatchById(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetBatchByIdBadId(t *testing.T) {
	e := NewEngine().(*engine)
	_, err := e.GetBatchById(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetBatches(t *testing.T) {
	e := NewEngine().(*engine)
	mp := &databasemocks.Plugin{}
	e.database = mp
	u := fftypes.NewUUID()
	mp.On("GetBatches", mock.Anything, mock.Anything).Return([]*fftypes.Batch{}, nil)
	fb := database.BatchQueryFactory.NewFilter(context.Background(), 0)
	f := fb.And(fb.Eq("id", u))
	_, err := e.GetBatches(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetDataById(t *testing.T) {
	e := NewEngine().(*engine)
	mp := &databasemocks.Plugin{}
	e.database = mp
	u := fftypes.NewUUID()
	mp.On("GetDataById", mock.Anything, "ns1", u).Return(nil, nil)
	_, err := e.GetDataById(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetDataByIdBadId(t *testing.T) {
	e := NewEngine().(*engine)
	_, err := e.GetDataById(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetData(t *testing.T) {
	e := NewEngine().(*engine)
	mp := &databasemocks.Plugin{}
	e.database = mp
	u := fftypes.NewUUID()
	mp.On("GetData", mock.Anything, mock.Anything).Return([]*fftypes.Data{}, nil)
	fb := database.DataQueryFactory.NewFilter(context.Background(), 0)
	f := fb.And(fb.Eq("id", u))
	_, err := e.GetData(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetDataDefsById(t *testing.T) {
	e := NewEngine().(*engine)
	mp := &databasemocks.Plugin{}
	e.database = mp
	u := fftypes.NewUUID()
	mp.On("GetDataDefinitionById", mock.Anything, "ns1", u).Return(nil, nil)
	_, err := e.GetDataDefinitionById(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetDataDefsByIdBadId(t *testing.T) {
	e := NewEngine().(*engine)
	_, err := e.GetDataDefinitionById(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetDataDefinitions(t *testing.T) {
	e := NewEngine().(*engine)
	mp := &databasemocks.Plugin{}
	e.database = mp
	u := fftypes.NewUUID()
	mp.On("GetDataDefinitions", mock.Anything, mock.Anything).Return([]*fftypes.DataDefinition{}, nil)
	fb := database.DataDefinitionQueryFactory.NewFilter(context.Background(), 0)
	f := fb.And(fb.Eq("id", u))
	_, err := e.GetDataDefinitions(context.Background(), "ns1", f)
	assert.NoError(t, err)
}
