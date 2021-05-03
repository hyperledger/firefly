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
	bm, _ := NewBatchManager(context.Background(), mp)
	defer bm.Close()

	mp.On("UpsertBatch", mock.Anything, mock.Anything).Return(nil)
	waitForDispatch := make(chan *fftypes.Batch)
	handler := func(ctx context.Context, b *fftypes.Batch) error {
		waitForDispatch <- b
		return nil
	}

	bm.RegisterDispatcher(fftypes.BatchTypeBroadcast, handler, BatchOptions{
		BatchMaxSize:   1,
		BatchTimeout:   0,
		DisposeTimeout: 120 * time.Second,
	})

	msgid := uuid.New()
	msg := &fftypes.MessageRefsOnly{MessageBase: fftypes.MessageBase{
		ID:        &msgid,
		Namespace: "ns1",
		Author:    "0x12345",
	}}

	id, err := bm.DispatchMessage(context.Background(), fftypes.BatchTypeBroadcast, msg)
	assert.NoError(t, err)
	assert.NotNil(t, id)

	b := <-waitForDispatch
	assert.Equal(t, msgid, *b.Payload.Messages[0].ID)

}

func TestInitFail(t *testing.T) {
	_, err := NewBatchManager(context.Background(), nil)
	assert.Error(t, err)
}

func TestGetInvalidBatchType(t *testing.T) {

	mp := &persistencemocks.Plugin{}
	bm, _ := NewBatchManager(context.Background(), mp)
	defer bm.Close()

	msg := &fftypes.MessageRefsOnly{MessageBase: fftypes.MessageBase{}}
	_, err := bm.DispatchMessage(context.Background(), fftypes.BatchTypeBroadcast, msg)
	assert.Regexp(t, "FF10126", err.Error())

}

func TestTimeout(t *testing.T) {

	mp := &persistencemocks.Plugin{}
	bm, _ := NewBatchManager(context.Background(), mp)
	defer bm.Close()

	blocker := make(chan time.Time)
	mup := mp.On("UpsertBatch", mock.Anything, mock.Anything)
	mup.WaitFor = blocker

	handler := func(ctx context.Context, b *fftypes.Batch) error {
		return nil
	}

	bm.RegisterDispatcher(fftypes.BatchTypeBroadcast, handler, BatchOptions{
		BatchMaxSize:   1,
		BatchTimeout:   0,
		DisposeTimeout: 120 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	msgid := uuid.New()
	msg := &fftypes.MessageRefsOnly{MessageBase: fftypes.MessageBase{
		ID:        &msgid,
		Namespace: "ns1",
		Author:    "0x12345",
	}}
	_, err := bm.DispatchMessage(ctx, fftypes.BatchTypeBroadcast, msg)

	assert.Regexp(t, "FF10127", err)

}
