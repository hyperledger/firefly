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
		identity:  fftypes.Identity{Author: "did:firefly:org/abcd", Key: "0x12345"},
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

	dispatched := make(chan *fftypes.Batch)
	mdi, bp := newTestBatchProcessor(func(c context.Context, b *fftypes.Batch, s []*fftypes.Bytes32) error {
		dispatched <- b
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

	dispatched := make(chan *fftypes.Batch)
	mdi, bp := newTestBatchProcessor(func(c context.Context, b *fftypes.Batch, s []*fftypes.Bytes32) error {
		dispatched <- b
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
	go func() {
		for i := 0; i < 2; i++ {
			msgid := fftypes.NewUUID()
			bp.newWork <- &batchWork{
				msg: &fftypes.Message{Header: fftypes.MessageHeader{ID: msgid}, Sequence: int64(1000 + i)},
			}
		}
	}()

	// Wait for the confirmations, and the dispatch
	batch1 := <-dispatched
	batch2 := <-dispatched

	// Check we got all messages across two batches
	assert.Equal(t, 1, len(batch1.Payload.Messages))
	assert.Equal(t, int64(1000), batch1.Payload.Messages[0].Sequence)
	assert.Equal(t, 1, len(batch2.Payload.Messages))
	assert.Equal(t, int64(1001), batch2.Payload.Messages[0].Sequence)

	bp.cancelCtx()
	<-bp.done
}

func TestCloseToUnblockDispatch(t *testing.T) {
	_, bp := newTestBatchProcessor(func(c context.Context, b *fftypes.Batch, s []*fftypes.Bytes32) error {
		return fmt.Errorf("pop")
	})
	bp.cancelCtx()
	bp.dispatchBatch(&fftypes.Batch{}, []*fftypes.Bytes32{})
	<-bp.done
}

func TestCloseToUnblockUpsertBatch(t *testing.T) {

	mdi, bp := newTestBatchProcessor(func(c context.Context, b *fftypes.Batch, s []*fftypes.Bytes32) error {
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
	_, bp := newTestBatchProcessor(func(c context.Context, b *fftypes.Batch, s []*fftypes.Bytes32) error {
		return nil
	})
	mdi := bp.database.(*databasemocks.Plugin)
	mdi.On("UpsertNonceNext", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	gid := fftypes.NewRandB32()
	_, err := bp.maskContexts(bp.ctx, &fftypes.Batch{
		Group: gid,
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{
					Group:  gid,
					Topics: fftypes.FFStringArray{"topic1"},
				}},
			},
		},
	})
	assert.Regexp(t, "pop", err)

	bp.cancelCtx()
	<-bp.done
}
