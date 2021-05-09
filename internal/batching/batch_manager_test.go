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
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/mocks/persistencemocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestE2EDispatch(t *testing.T) {

	mp := &persistencemocks.Plugin{}
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
			{ID: dataId1, Hash: &dataHash},
		},
	}
	data := &fftypes.Data{
		ID:   dataId1,
		Hash: &dataHash,
	}
	mp.On("GetDataById", mock.Anything, "ns1", mock.MatchedBy(func(i interface{}) bool {
		return *(i.(*uuid.UUID)) == *dataId1
	})).Return(data, nil)
	mp.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{msg}, nil).Once()
	mp.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{}, nil)
	mp.On("UpsertBatch", mock.Anything, mock.Anything).Return(nil)
	mp.On("UpdateMessage", mock.Anything, mock.MatchedBy(func(i interface{}) bool {
		return *(i.(*uuid.UUID)) == *msg.Header.ID
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

func TestInitFailCannotRestoreOffset(t *testing.T) {
	mp := &persistencemocks.Plugin{}
	mp.On("GetOffset", mock.Anything, fftypes.OffsetTypeBatch, fftypes.SystemNamespace, msgBatchOffsetName).Return(nil, fmt.Errorf("pop"))
	bm, err := NewBatchManager(context.Background(), mp)
	defer bm.Close()
	assert.NoError(t, err)
	bm.(*batchManager).retry.MaximumDelay = 1 * time.Microsecond
	err = bm.Start()
	assert.Regexp(t, "pop", err.Error())
}

func TestInitFailCannotCreateOffset(t *testing.T) {
	mp := &persistencemocks.Plugin{}
	mp.On("GetOffset", mock.Anything, fftypes.OffsetTypeBatch, fftypes.SystemNamespace, msgBatchOffsetName).Return(nil, nil)
	mp.On("UpsertOffset", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	bm, err := NewBatchManager(context.Background(), mp)
	defer bm.Close()
	assert.NoError(t, err)
	bm.(*batchManager).retry.MaximumDelay = 1 * time.Microsecond
	err = bm.Start()
	assert.Regexp(t, "pop", err.Error())
}

func TestGetInvalidBatchTypeMsg(t *testing.T) {

	mp := &persistencemocks.Plugin{}
	mp.On("GetOffset", mock.Anything, fftypes.OffsetTypeBatch, fftypes.SystemNamespace, msgBatchOffsetName).Return(&fftypes.Offset{
		Current: 12345,
	}, nil)
	bm, _ := NewBatchManager(context.Background(), mp)
	defer bm.Close()
	msg := &fftypes.Message{Header: fftypes.MessageHeader{}}
	err := bm.(*batchManager).dispatchMessage(context.Background(), msg)
	assert.Regexp(t, "FF10126", err.Error())
}
