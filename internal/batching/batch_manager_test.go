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

package batching

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestE2EDispatch(t *testing.T) {

	mp := &databasemocks.Plugin{}
	mp.On("GetOffset", mock.Anything, fftypes.OffsetTypeBatch, fftypes.SystemNamespace, msgBatchOffsetName).Return(nil, nil)
	mp.On("UpsertOffset", mock.Anything, mock.Anything).Return(nil)
	waitForDispatch := make(chan *fftypes.Batch)
	handler := func(ctx context.Context, b *fftypes.Batch) error {
		waitForDispatch <- b
		return nil
	}
	bmi, _ := NewBatchManager(context.Background(), mp)
	bm := bmi.(*batchManager)

	bm.RegisterDispatcher(fftypes.MessageTypeBroadcast, handler, BatchOptions{
		BatchMaxSize:   2,
		BatchTimeout:   0,
		DisposeTimeout: 120 * time.Second,
	})

	dataId1 := fftypes.NewUUID()
	dataHash := fftypes.NewRandB32()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:      fftypes.MessageTypeBroadcast,
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Author:    "0x12345",
		},
		Data: fftypes.DataRefs{
			{ID: dataId1, Hash: dataHash},
		},
	}
	data := &fftypes.Data{
		ID:   dataId1,
		Hash: dataHash,
	}
	mp.On("GetDataById", mock.Anything, mock.MatchedBy(func(i interface{}) bool {
		return *(i.(*uuid.UUID)) == *dataId1
	})).Return(data, nil)
	mp.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{msg}, nil).Once()
	mp.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{}, nil)
	mp.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mp.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	rag := mp.On("RunAsGroup", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	rag.RunFn = func(a mock.Arguments) {
		ctx := a.Get(0).(context.Context)
		fn := a.Get(1).(func(context.Context) error)
		fn(ctx)
	}
	mp.On("UpdateMessages", mock.Anything, mock.MatchedBy(func(f database.Filter) bool {
		fi, err := f.Finalize()
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("id IN ['%s']", msg.Header.ID.String()), fi.String())
		return true
	}), mock.Anything).Return(nil)

	err := bm.Start()
	assert.NoError(t, err)

	bm.NewMessages() <- msg.Header.ID

	b := <-waitForDispatch
	assert.Equal(t, *msg.Header.ID, *b.Payload.Messages[0].Header.ID)
	assert.Equal(t, *data.ID, *b.Payload.Data[0].ID)

	// Wait until everything closes
	bm.Close()
	for len(bm.dispatchers[fftypes.MessageTypeBroadcast].processors) > 0 {
		time.Sleep(1 * time.Microsecond)
	}

}

func TestInitFailNoPersistence(t *testing.T) {
	_, err := NewBatchManager(context.Background(), nil)
	assert.Error(t, err)
}

func TestInitRestoreExistingOffset(t *testing.T) {
	mp := &databasemocks.Plugin{}
	mp.On("GetOffset", mock.Anything, fftypes.OffsetTypeBatch, fftypes.SystemNamespace, msgBatchOffsetName).Return(&fftypes.Offset{
		Type:      fftypes.OffsetTypeBatch,
		Namespace: fftypes.SystemNamespace,
		Name:      msgBatchOffsetName,
		Current:   12345,
	}, nil)
	bm, err := NewBatchManager(context.Background(), mp)
	assert.NoError(t, err)
	defer bm.Close()
	err = bm.Start()
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), bm.(*batchManager).offset)
}

func TestInitFailCannotRestoreOffset(t *testing.T) {
	mp := &databasemocks.Plugin{}
	mp.On("GetOffset", mock.Anything, fftypes.OffsetTypeBatch, fftypes.SystemNamespace, msgBatchOffsetName).Return(nil, fmt.Errorf("pop"))
	bm, err := NewBatchManager(context.Background(), mp)
	assert.NoError(t, err)
	defer bm.Close()
	bm.(*batchManager).retry.MaximumDelay = 1 * time.Microsecond
	err = bm.Start()
	assert.Regexp(t, "pop", err.Error())
}

func TestInitFailCannotCreateOffset(t *testing.T) {
	mp := &databasemocks.Plugin{}
	mp.On("GetOffset", mock.Anything, fftypes.OffsetTypeBatch, fftypes.SystemNamespace, msgBatchOffsetName).Return(nil, nil)
	mp.On("UpsertOffset", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	bm, err := NewBatchManager(context.Background(), mp)
	assert.NoError(t, err)
	defer bm.Close()
	bm.(*batchManager).retry.MaximumDelay = 1 * time.Microsecond
	err = bm.Start()
	assert.Regexp(t, "pop", err.Error())
}

func TestGetInvalidBatchTypeMsg(t *testing.T) {

	mp := &databasemocks.Plugin{}
	mp.On("GetOffset", mock.Anything, fftypes.OffsetTypeBatch, fftypes.SystemNamespace, msgBatchOffsetName).Return(&fftypes.Offset{
		Current: 12345,
	}, nil)
	bm, _ := NewBatchManager(context.Background(), mp)
	defer bm.Close()
	msg := &fftypes.Message{Header: fftypes.MessageHeader{}}
	err := bm.(*batchManager).dispatchMessage(nil, msg)
	assert.Regexp(t, "FF10126", err.Error())
}

func TestMessageSequencerCancelledContext(t *testing.T) {
	mp := &databasemocks.Plugin{}
	mp.On("GetMessages", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	bm, _ := NewBatchManager(context.Background(), mp)
	defer bm.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	bm.(*batchManager).ctx = ctx
	bm.(*batchManager).messageSequencer()
	assert.Equal(t, 1, len(mp.Calls))
}

func TestMessageSequencerMissingMessageData(t *testing.T) {
	mp := &databasemocks.Plugin{}
	bm, _ := NewBatchManager(context.Background(), mp)

	dataId := fftypes.NewUUID()
	gmMock := mp.On("GetMessages", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Message{
		{
			Header: fftypes.MessageHeader{
				ID:        fftypes.NewUUID(),
				Namespace: "ns1",
			},
			Data: []fftypes.DataRef{
				{ID: dataId},
			}},
	}, nil)
	gmMock.RunFn = func(a mock.Arguments) {
		bm.Close() // so we only go round once
	}
	mp.On("GetDataById", mock.Anything, dataId).Return(nil, nil)

	bm.(*batchManager).messageSequencer()
	assert.Equal(t, 2, len(mp.Calls))
}

func TestMessageSequencerDispatchFail(t *testing.T) {
	mp := &databasemocks.Plugin{}
	bm, _ := NewBatchManager(context.Background(), mp)

	dataId := fftypes.NewUUID()
	gmMock := mp.On("GetMessages", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Message{
		{
			Header: fftypes.MessageHeader{
				ID:        fftypes.NewUUID(),
				Type:      fftypes.MessageTypePrivate,
				Namespace: "ns1",
			},
			Data: []fftypes.DataRef{
				{ID: dataId},
			}},
	}, nil)
	gmMock.RunFn = func(a mock.Arguments) {
		bm.Close() // so we only go round once
	}
	mp.On("GetDataById", mock.Anything, dataId).Return(&fftypes.Data{ID: dataId}, nil)

	bm.(*batchManager).messageSequencer()
	assert.Equal(t, 2, len(mp.Calls))
}

func TestMessageSequencerUpdateMessagesClosed(t *testing.T) {
	mp := &databasemocks.Plugin{}
	bm, _ := NewBatchManager(context.Background(), mp)
	bm.RegisterDispatcher(fftypes.MessageTypeBroadcast, func(c context.Context, b *fftypes.Batch) error {
		return nil
	}, BatchOptions{BatchMaxSize: 1, DisposeTimeout: 0})

	dataId := fftypes.NewUUID()
	gmMock := mp.On("GetMessages", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Message{
		{
			Header: fftypes.MessageHeader{
				ID:        fftypes.NewUUID(),
				Type:      fftypes.MessageTypeBroadcast,
				Namespace: "ns1",
			},
			Data: []fftypes.DataRef{
				{ID: dataId},
			}},
	}, nil)
	gmMock.RunFn = func(a mock.Arguments) {
		bm.Close() // so we only go round once
	}
	mp.On("GetDataById", mock.Anything, dataId).Return(&fftypes.Data{ID: dataId}, nil)
	mp.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mp.On("UpdateMessages", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("fizzle"))
	rag := mp.On("RunAsGroup", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	rag.RunFn = func(a mock.Arguments) {
		ctx := a.Get(0).(context.Context)
		fn := a.Get(1).(func(context.Context) error)
		assert.Regexp(t, "fizzle", fn(ctx).Error())
	}

	bm.(*batchManager).messageSequencer()
	mp.AssertExpectations(t)
}

func TestWaitForPollTimeout(t *testing.T) {
	mp := &databasemocks.Plugin{}
	bm, _ := NewBatchManager(context.Background(), mp)
	bm.(*batchManager).messagePollTimeout = 1 * time.Microsecond
	bm.(*batchManager).waitForShoulderTapOrPollTimeout()
}

func TestWaitConsumesAllShoulderTaps(t *testing.T) {
	config.Reset()
	mp := &databasemocks.Plugin{}
	bm, _ := NewBatchManager(context.Background(), mp)
	for i := 0; i < int(bm.(*batchManager).readPageSize); i++ {
		bm.NewMessages() <- fftypes.NewUUID()
	}
	bm.(*batchManager).messagePollTimeout = 1 * time.Second
	bm.(*batchManager).waitForShoulderTapOrPollTimeout()
	select {
	case <-bm.(*batchManager).newMessages:
		assert.Fail(t, "should have been drained")
	default:
	}
}

func TestAssembleMessageDataNilData(t *testing.T) {
	mp := &databasemocks.Plugin{}
	bm, _ := NewBatchManager(context.Background(), mp)
	bm.Close()
	_, err := bm.(*batchManager).assembleMessageData(&fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
		Data: fftypes.DataRefs{{ID: nil}},
	})
	assert.NoError(t, err)
}

func TestAssembleMessageDataClosed(t *testing.T) {
	mp := &databasemocks.Plugin{}
	bm, _ := NewBatchManager(context.Background(), mp)
	mp.On("GetDataById", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	bm.Close()
	_, err := bm.(*batchManager).assembleMessageData(&fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
		},
	})
	assert.EqualError(t, err, "pop")
}
