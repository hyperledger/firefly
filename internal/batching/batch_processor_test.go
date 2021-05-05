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
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/mocks/persistencemocks"
	"github.com/likexian/gokit/assert"
	"github.com/stretchr/testify/mock"
)

func newTestBatchProcessor(dispatch DispatchHandler) (*persistencemocks.Plugin, *batchProcessor) {
	mp := &persistencemocks.Plugin{}
	bp := newBatchProcessor(context.Background(), &batchProcessorConf{
		namespace:          "ns1",
		author:             "0x12345",
		persitence:         mp,
		dispatch:           dispatch,
		processorQuiescing: func() {},
		BatchOptions: BatchOptions{
			BatchMaxSize:   10,
			BatchTimeout:   10 * time.Millisecond,
			DisposeTimeout: 20 * time.Millisecond,
		},
	})
	return mp, bp
}

func TestUnfilledBatch(t *testing.T) {
	log.SetLevel("debug")

	wg := sync.WaitGroup{}
	wg.Add(2)

	dispatched := []*fftypes.Batch{}
	mp, bp := newTestBatchProcessor(func(c context.Context, b *fftypes.Batch) error {
		dispatched = append(dispatched, b)
		wg.Done()
		return nil
	})
	mp.On("UpsertBatch", mock.Anything, mock.Anything).Return(nil)

	// Generate the work the work
	work := make([]*batchWork, 5)
	for i := 0; i < 5; i++ {
		msgid := uuid.New()
		work[i] = &batchWork{
			msg:        &fftypes.Message{Header: fftypes.MessageHeader{ID: &msgid}},
			dispatched: make(chan *uuid.UUID),
		}
	}

	// Kick off a go routine to consume the confirmations
	go func() {
		for i := 0; i < 5; i++ {
			<-work[i].dispatched
		}
		wg.Done()
	}()

	// Dispatch the work
	for i := 0; i < 5; i++ {
		bp.newWork <- work[i]
	}

	// Wait for the confirmations, and the dispatch
	wg.Wait()

	// Check we got all the messages in a single batch
	assert.Equal(t, len(dispatched[0].Payload.Messages), 5)

	bp.close()

}

func TestFilledBatchSlowPersistence(t *testing.T) {
	log.SetLevel("debug")

	wg := sync.WaitGroup{}
	wg.Add(2)

	dispatched := []*fftypes.Batch{}
	mp, bp := newTestBatchProcessor(func(c context.Context, b *fftypes.Batch) error {
		dispatched = append(dispatched, b)
		wg.Done()
		return nil
	})
	bp.conf.BatchTimeout = 1 * time.Hour // Must fill the batch
	mockUpsert := mp.On("UpsertBatch", mock.Anything, mock.Anything)
	mockUpsert.ReturnArguments = mock.Arguments{nil}
	unblockPersistence := make(chan time.Time)
	mockUpsert.WaitFor = unblockPersistence

	// Generate the work the work
	work := make([]*batchWork, 10)
	for i := 0; i < 10; i++ {
		msgid := uuid.New()
		if i%2 == 0 {
			work[i] = &batchWork{
				msg:        &fftypes.Message{Header: fftypes.MessageHeader{ID: &msgid}},
				dispatched: make(chan *uuid.UUID),
			}
		} else {
			work[i] = &batchWork{
				data:       []*fftypes.Data{{ID: &msgid}},
				dispatched: make(chan *uuid.UUID),
			}
		}
	}

	// Kick off a go routine to consume the confirmations
	go func() {
		for i := 0; i < 10; i++ {
			<-work[i].dispatched
		}
		wg.Done()
	}()

	// Dispatch the work
	for i := 0; i < 10; i++ {
		bp.newWork <- work[i]
	}

	// Unblock the dispatch
	time.Sleep(10 * time.Millisecond)
	mockUpsert.WaitFor = nil
	unblockPersistence <- time.Now() // First call to write the first entry in the batch

	// Wait for completion
	wg.Wait()

	// Check we got all the messages in a single batch
	assert.Equal(t, len(dispatched[0].Payload.Messages), 5)
	assert.Equal(t, len(dispatched[0].Payload.Data), 5)

	bp.close()

}

func TestCloseToUnblockDispatch(t *testing.T) {
	_, bp := newTestBatchProcessor(func(c context.Context, b *fftypes.Batch) error {
		return fmt.Errorf("pop")
	})
	bp.close()
	bp.dispatchBatch(&fftypes.Batch{})
}

func TestCloseToUnblockUpsertBatch(t *testing.T) {

	wg := sync.WaitGroup{}
	wg.Add(1)

	mp, bp := newTestBatchProcessor(func(c context.Context, b *fftypes.Batch) error {
		return nil
	})
	bp.retry.MaximumDelay = 1 * time.Microsecond
	bp.conf.BatchTimeout = 100 * time.Second
	mup := mp.On("UpsertBatch", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	waitForCall := make(chan bool)
	mup.RunFn = func(a mock.Arguments) {
		waitForCall <- true
		<-waitForCall
	}

	// Generate the work the work
	msgid := uuid.New()
	work := &batchWork{
		msg:        &fftypes.Message{Header: fftypes.MessageHeader{ID: &msgid}},
		dispatched: make(chan *uuid.UUID),
	}

	// Dispatch the work
	bp.newWork <- work

	// Ensure the mock has been run
	<-waitForCall
	close(waitForCall)

	// Close to unblock
	bp.close()

	// Wait for the persistence loop to terminate
	<-bp.batchSealed

}
