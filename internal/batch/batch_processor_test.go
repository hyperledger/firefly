// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in comdiliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or imdilied.
// See the License for the specific language governing permissions and
// limitations under the License.

package batch

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/sysmessagingmocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestBatchProcessor(dispatch DispatchHandler) (*databasemocks.Plugin, *batchProcessor) {
	mdi := &databasemocks.Plugin{}
	mni := &sysmessagingmocks.LocalNodeInfo{}
	mni.On("GetNodeUUID", mock.Anything).Return(fftypes.NewUUID()).Maybe()
	bp := newBatchProcessor(context.Background(), mni, mdi, &batchProcessorConf{
		namespace: "ns1",
		txType:    fftypes.TransactionTypeBatchPin,
		signer:    fftypes.SignerRef{Author: "did:firefly:org/abcd", Key: "0x12345"},
		dispatch:  dispatch,
		DispatcherOptions: DispatcherOptions{
			BatchMaxSize:   10,
			BatchMaxBytes:  1024 * 1024,
			BatchTimeout:   100 * time.Millisecond,
			DisposeTimeout: 200 * time.Millisecond,
		},
	}, &retry.Retry{
		InitialDelay: 1 * time.Microsecond,
		MaximumDelay: 1 * time.Microsecond,
	})
	bp.txHelper = &txcommonmocks.Helper{}
	return mdi, bp
}

func mockRunAsGroupPassthrough(mdi *databasemocks.Plugin) {
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		fn := a[1].(func(context.Context) error)
		rag.ReturnArguments = mock.Arguments{fn(a[0].(context.Context))}
	}
}

func TestUnfilledBatch(t *testing.T) {
	log.SetLevel("debug")
	config.Reset()

	dispatched := make(chan *DispatchState)
	mdi, bp := newTestBatchProcessor(func(c context.Context, state *DispatchState) error {
		dispatched <- state
		return nil
	})

	mockRunAsGroupPassthrough(mdi)
	mdi.On("UpdateMessages", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpdateBatch", mock.Anything, mock.Anything).Return(nil)

	mth := bp.txHelper.(*txcommonmocks.Helper)
	mth.On("SubmitNewTransaction", mock.Anything, "ns1", fftypes.TransactionTypeBatchPin).Return(fftypes.NewUUID(), nil)

	// Dispatch the work
	go func() {
		for i := 0; i < 5; i++ {
			msgid := fftypes.NewUUID()
			bp.newWork <- &batchWork{
				msg: &fftypes.Message{Header: fftypes.MessageHeader{ID: msgid}, Sequence: int64(1000 + i)},
			}
		}
	}()

	// Wait for the confirmations, and the dispatch
	batch := <-dispatched

	// Check we got all the messages in a single batch
	assert.Equal(t, 5, len(batch.Payload.Messages))

	bp.cancelCtx()
	<-bp.done
}

func TestBatchSizeOverflow(t *testing.T) {
	log.SetLevel("debug")
	config.Reset()

	dispatched := make(chan *DispatchState)
	mdi, bp := newTestBatchProcessor(func(c context.Context, state *DispatchState) error {
		dispatched <- state
		return nil
	})
	bp.conf.BatchMaxBytes = batchSizeEstimateBase + (&fftypes.Message{}).EstimateSize(false) + 100
	mockRunAsGroupPassthrough(mdi)
	mdi.On("UpdateMessages", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpdateBatch", mock.Anything, mock.Anything).Return(nil)

	mth := bp.txHelper.(*txcommonmocks.Helper)
	mth.On("SubmitNewTransaction", mock.Anything, "ns1", fftypes.TransactionTypeBatchPin).Return(fftypes.NewUUID(), nil)

	// Dispatch the work
	msgIDs := []*fftypes.UUID{fftypes.NewUUID(), fftypes.NewUUID()}
	go func() {
		for i := 0; i < 2; i++ {
			bp.newWork <- &batchWork{
				msg: &fftypes.Message{Header: fftypes.MessageHeader{ID: msgIDs[i]}, Sequence: int64(1000 + i)},
			}
		}
	}()

	// Wait for the confirmations, and the dispatch
	batch1 := <-dispatched
	batch2 := <-dispatched

	// Check we got all messages across two batches
	assert.Equal(t, 1, len(batch1.Payload.Messages))
	assert.Equal(t, msgIDs[0], batch1.Payload.Messages[0].Header.ID)
	assert.Equal(t, 1, len(batch2.Payload.Messages))
	assert.Equal(t, msgIDs[1], batch2.Payload.Messages[0].Header.ID)

	bp.cancelCtx()
	<-bp.done
}

func TestCloseToUnblockDispatch(t *testing.T) {
	_, bp := newTestBatchProcessor(func(c context.Context, state *DispatchState) error {
		return fmt.Errorf("pop")
	})
	bp.cancelCtx()
	bp.dispatchBatch(&DispatchState{})
	<-bp.done
}

func TestCloseToUnblockUpsertBatch(t *testing.T) {

	mdi, bp := newTestBatchProcessor(func(c context.Context, state *DispatchState) error {
		return nil
	})
	bp.retry.MaximumDelay = 1 * time.Microsecond
	bp.conf.BatchMaxSize = 1
	bp.conf.BatchTimeout = 100 * time.Second
	mockRunAsGroupPassthrough(mdi)
	waitForCall := make(chan bool)
	mth := bp.txHelper.(*txcommonmocks.Helper)
	mth.On("SubmitNewTransaction", mock.Anything, "ns1", fftypes.TransactionTypeBatchPin).
		Run(func(a mock.Arguments) {
			waitForCall <- true
			<-waitForCall
		}).
		Return(nil, fmt.Errorf("pop"))

	// Generate the work
	msgid := fftypes.NewUUID()
	go func() {
		bp.newWork <- &batchWork{
			msg: &fftypes.Message{Header: fftypes.MessageHeader{ID: msgid}, Sequence: int64(1000)},
		}
	}()

	// Ensure the mock has been run
	<-waitForCall
	close(waitForCall)

	// Close to unblock
	bp.cancelCtx()
	<-bp.done
}

func TestCalcPinsFail(t *testing.T) {
	_, bp := newTestBatchProcessor(func(c context.Context, state *DispatchState) error {
		return nil
	})
	bp.cancelCtx()
	mdi := bp.database.(*databasemocks.Plugin)
	mdi.On("UpsertNonceNext", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mockRunAsGroupPassthrough(mdi)

	gid := fftypes.NewRandB32()
	err := bp.sealBatch(&DispatchState{
		Persisted: fftypes.BatchPersisted{
			BatchHeader: fftypes.BatchHeader{
				Group: gid,
			},
		},
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{
					Group:  gid,
					Topics: fftypes.FFStringArray{"topic1"},
				}},
			},
		},
	})
	assert.Regexp(t, "FF10158", err)

	<-bp.done

	mdi.AssertExpectations(t)
}

func TestAddWorkInRecentlyFlushed(t *testing.T) {
	_, bp := newTestBatchProcessor(func(c context.Context, state *DispatchState) error {
		return nil
	})
	bp.flushedSequences = []int64{100, 500, 400, 900, 200, 700}
	_, _ = bp.addWork(&batchWork{
		msg: &fftypes.Message{
			Sequence: 200,
		},
	})
	assert.Empty(t, bp.assemblyQueue)

}

func TestAddWorkInSortDeDup(t *testing.T) {
	_, bp := newTestBatchProcessor(func(c context.Context, state *DispatchState) error {
		return nil
	})
	bp.assemblyQueue = []*batchWork{
		{msg: &fftypes.Message{Sequence: 200}},
		{msg: &fftypes.Message{Sequence: 201}},
		{msg: &fftypes.Message{Sequence: 202}},
		{msg: &fftypes.Message{Sequence: 204}},
	}
	_, _ = bp.addWork(&batchWork{
		msg: &fftypes.Message{Sequence: 200},
	})
	_, _ = bp.addWork(&batchWork{
		msg: &fftypes.Message{Sequence: 203},
	})
	assert.Equal(t, []*batchWork{
		{msg: &fftypes.Message{Sequence: 200}},
		{msg: &fftypes.Message{Sequence: 201}},
		{msg: &fftypes.Message{Sequence: 202}},
		{msg: &fftypes.Message{Sequence: 203}},
		{msg: &fftypes.Message{Sequence: 204}},
	}, bp.assemblyQueue)
}

func TestStartFlushOverflow(t *testing.T) {
	_, bp := newTestBatchProcessor(func(c context.Context, state *DispatchState) error {
		return nil
	})
	batchID := fftypes.NewUUID()
	bp.assemblyID = batchID
	bp.flushedSequences = []int64{100, 101, 102, 103, 104}
	bp.assemblyQueue = []*batchWork{
		{msg: &fftypes.Message{Sequence: 200}},
		{msg: &fftypes.Message{Sequence: 201}},
		{msg: &fftypes.Message{Sequence: 202}},
		{msg: &fftypes.Message{Sequence: 203}},
	}
	bp.conf.BatchMaxSize = 3

	flushBatchID, flushAssembly, _ := bp.startFlush(true)
	assert.Equal(t, batchID, flushBatchID)
	assert.Equal(t, []int64{102, 103, 104, 200, 201, 202}, bp.flushedSequences)
	assert.Equal(t, []*batchWork{
		{msg: &fftypes.Message{Sequence: 200}},
		{msg: &fftypes.Message{Sequence: 201}},
		{msg: &fftypes.Message{Sequence: 202}},
	}, flushAssembly)
	assert.Equal(t, []*batchWork{
		{msg: &fftypes.Message{Sequence: 203}},
	}, bp.assemblyQueue)
	assert.NotEqual(t, batchID, bp.assemblyID)
}

func TestStartQuiesceNonBlocking(t *testing.T) {
	_, bp := newTestBatchProcessor(func(c context.Context, state *DispatchState) error {
		return nil
	})
	bp.startQuiesce()
	bp.startQuiesce() // we're just checking this doesn't hang
}

func TestMarkMessageDispatchedUnpinnedOK(t *testing.T) {
	log.SetLevel("debug")
	config.Reset()

	dispatched := make(chan *DispatchState)
	mdi, bp := newTestBatchProcessor(func(c context.Context, state *DispatchState) error {
		dispatched <- state
		return nil
	})
	bp.conf.txType = fftypes.TransactionTypeUnpinned

	mockRunAsGroupPassthrough(mdi)
	mdi.On("UpdateMessages", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Once()
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)

	mth := bp.txHelper.(*txcommonmocks.Helper)
	mth.On("SubmitNewTransaction", mock.Anything, "ns1", fftypes.TransactionTypeUnpinned).Return(fftypes.NewUUID(), nil)

	// Dispatch the work
	go func() {
		for i := 0; i < 5; i++ {
			msgid := fftypes.NewUUID()
			bp.newWork <- &batchWork{
				msg: &fftypes.Message{Header: fftypes.MessageHeader{ID: msgid}, Sequence: int64(1000 + i)},
			}
		}
	}()

	// Wait for the confirmations, and the dispatch
	batch := <-dispatched

	// Check we got all the messages in a single batch
	assert.Equal(t, 5, len(batch.Payload.Messages))

	bp.cancelCtx()
	<-bp.done

	mdi.AssertExpectations(t)
}

func TestMaskContextsDuplicate(t *testing.T) {
	log.SetLevel("debug")
	config.Reset()

	dispatched := make(chan *DispatchState)
	mdi, bp := newTestBatchProcessor(func(c context.Context, state *DispatchState) error {
		dispatched <- state
		return nil
	})

	mdi.On("UpsertNonceNext", mock.Anything, mock.Anything).Return(nil).Once()
	mdi.On("UpdateMessage", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	payload := &fftypes.BatchPayload{
		Messages: []*fftypes.Message{
			{
				Header: fftypes.MessageHeader{
					ID:     fftypes.NewUUID(),
					Type:   fftypes.MessageTypePrivate,
					Group:  fftypes.NewRandB32(),
					Topics: fftypes.FFStringArray{"topic1"},
				},
			},
		},
	}

	_, err := bp.maskContexts(bp.ctx, payload)
	assert.NoError(t, err)

	// 2nd time no DB ops
	_, err = bp.maskContexts(bp.ctx, payload)
	assert.NoError(t, err)

	bp.cancelCtx()
	<-bp.done

	mdi.AssertExpectations(t)
}

func TestMaskContextsUpdataMessageFail(t *testing.T) {
	log.SetLevel("debug")
	config.Reset()

	dispatched := make(chan *DispatchState)
	mdi, bp := newTestBatchProcessor(func(c context.Context, state *DispatchState) error {
		dispatched <- state
		return nil
	})

	mdi.On("UpsertNonceNext", mock.Anything, mock.Anything).Return(nil).Once()
	mdi.On("UpdateMessage", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Once()

	payload := &fftypes.BatchPayload{
		Messages: []*fftypes.Message{
			{
				Header: fftypes.MessageHeader{
					ID:     fftypes.NewUUID(),
					Type:   fftypes.MessageTypePrivate,
					Group:  fftypes.NewRandB32(),
					Topics: fftypes.FFStringArray{"topic1"},
				},
			},
		},
	}

	_, err := bp.maskContexts(bp.ctx, payload)
	assert.Regexp(t, "pop", err)

	bp.cancelCtx()
	<-bp.done

	mdi.AssertExpectations(t)
}
