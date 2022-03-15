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

package data

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestMessageWriter(t *testing.T) *messageWriter {
	return newTestMessageWriterConf(t, &messageWriterConf{
		workerCount:  1,
		batchTimeout: 100 * time.Millisecond,
		maxInserts:   200,
	}, &database.Capabilities{Concurrency: true})
}

func newTestMessageWriterNoConcrrency(t *testing.T) *messageWriter {
	return newTestMessageWriterConf(t, &messageWriterConf{workerCount: 1}, &database.Capabilities{Concurrency: false})
}

func newTestMessageWriterConf(t *testing.T, conf *messageWriterConf, dbCapabilities *database.Capabilities) *messageWriter {
	mdi := &databasemocks.Plugin{}
	mdi.On("Capabilities").Return(dbCapabilities)
	return newMessageWriter(context.Background(), mdi, conf)
}

func TestNewMessageWriterNoConcurrency(t *testing.T) {
	mw := newTestMessageWriterNoConcrrency(t)
	assert.Zero(t, mw.conf.workerCount)
}

func TestWriteNewMessageClosed(t *testing.T) {
	mw := newTestMessageWriter(t)
	mw.close()
	err := mw.WriteNewMessage(mw.ctx, &NewMessage{
		Message: &fftypes.MessageInOut{},
	})
	assert.Regexp(t, "FF10158", err)
}

func TestWriteDataClosed(t *testing.T) {
	mw := newTestMessageWriter(t)
	mw.close()
	err := mw.WriteData(mw.ctx, &fftypes.Data{})
	assert.Regexp(t, "FF10158", err)
}

func TestWriteNewMessageSyncFallback(t *testing.T) {
	mw := newTestMessageWriterNoConcrrency(t)
	customCtx := context.WithValue(context.Background(), "dbtx", "on this context")

	msg1 := &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				ID: fftypes.NewUUID(),
			},
		},
	}
	data1 := &fftypes.Data{ID: fftypes.NewUUID()}

	mdi := mw.database.(*databasemocks.Plugin)
	mdi.On("RunAsGroup", customCtx, mock.Anything).Run(func(args mock.Arguments) {
		err := args[1].(func(context.Context) error)(customCtx)
		assert.NoError(t, err)
	}).Return(nil)
	mdi.On("InsertMessages", customCtx, []*fftypes.Message{&msg1.Message}).Return(nil)
	mdi.On("InsertDataArray", customCtx, fftypes.DataArray{data1}).Return(nil)

	err := mw.WriteNewMessage(customCtx, &NewMessage{
		Message: msg1,
		NewData: fftypes.DataArray{data1},
	})

	assert.NoError(t, err)
}

func TestWriteDataSyncFallback(t *testing.T) {
	mw := newTestMessageWriterNoConcrrency(t)
	customCtx := context.WithValue(context.Background(), "dbtx", "on this context")

	data1 := &fftypes.Data{ID: fftypes.NewUUID()}

	mdi := mw.database.(*databasemocks.Plugin)
	mdi.On("UpsertData", customCtx, data1, database.UpsertOptimizationNew).Return(nil)

	err := mw.WriteData(customCtx, data1)

	assert.NoError(t, err)
}

func TestWriteMessagesInsertMessagesFail(t *testing.T) {
	mw := newTestMessageWriterNoConcrrency(t)

	msg1 := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}

	mdi := mw.database.(*databasemocks.Plugin)
	mdi.On("InsertMessages", mw.ctx, []*fftypes.Message{msg1}).Return(fmt.Errorf("pop"))

	err := mw.writeMessages(mw.ctx, []*fftypes.Message{msg1}, fftypes.DataArray{})

	assert.Regexp(t, "pop", err)
}

func TestWriteMessagesInsertDataArrayFail(t *testing.T) {
	mw := newTestMessageWriterNoConcrrency(t)

	data1 := &fftypes.Data{ID: fftypes.NewUUID()}

	mdi := mw.database.(*databasemocks.Plugin)
	mdi.On("InsertDataArray", mw.ctx, fftypes.DataArray{data1}).Return(fmt.Errorf("pop"))

	err := mw.writeMessages(mw.ctx, []*fftypes.Message{}, fftypes.DataArray{data1})

	assert.Regexp(t, "pop", err)
}
