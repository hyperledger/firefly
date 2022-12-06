// Copyright © 2022 Kaleido, Inc.
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
package batch

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/retry"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/mocks/cachemocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestBatchProcessor(t *testing.T, dispatch DispatchHandler) (func(), *databasemocks.Plugin, *batchProcessor) {
	bm, cancel := newTestBatchManager(t)
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := txcommon.NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)
	bp := newBatchProcessor(bm, &batchProcessorConf{
		txType:   core.TransactionTypeBatchPin,
		signer:   core.SignerRef{Author: "did:firefly:org/abcd", Key: "0x12345"},
		dispatch: dispatch,
		DispatcherOptions: DispatcherOptions{
			BatchMaxSize:   10,
			BatchMaxBytes:  1024 * 1024,
			BatchTimeout:   100 * time.Millisecond,
			DisposeTimeout: 200 * time.Millisecond,
		},
	}, &retry.Retry{
		InitialDelay: 1 * time.Microsecond,
		MaximumDelay: 1 * time.Microsecond,
	}, txHelper)
	bp.txHelper = &txcommonmocks.Helper{}
	return cancel, mdi, bp
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
	coreconfig.Reset()

	dispatched := make(chan *DispatchState)
	cancel, mdi, bp := newTestBatchProcessor(t, func(c context.Context, state *DispatchState) error {
		dispatched <- state
		return nil
	})
	defer cancel()

	mockRunAsGroupPassthrough(mdi)
	mdi.On("UpdateMessages", mock.Anything, "ns1", mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mth := bp.txHelper.(*txcommonmocks.Helper)
	mth.On("SubmitNewTransaction", mock.Anything, core.TransactionTypeBatchPin, core.IdempotencyKey("")).Return(fftypes.NewUUID(), nil)

	mdm := bp.data.(*datamocks.Manager)
	mdm.On("UpdateMessageIfCached", mock.Anything, mock.Anything).Return()

	mim := bp.bm.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalNode", mock.Anything).Return(&core.Identity{}, nil)

	// Dispatch the work
	go func() {
		for i := 0; i < 5; i++ {
			msgid := fftypes.NewUUID()
			bp.newWork <- &batchWork{
				msg: &core.Message{Header: core.MessageHeader{ID: msgid}, Sequence: int64(1000 + i)},
			}
		}
	}()

	// Wait for the confirmations, and the dispatch
	batch := <-dispatched

	// Check we got all the messages in a single batch
	assert.Equal(t, 5, len(batch.Messages))

	bp.cancelCtx()
	<-bp.done

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestBatchSizeOverflow(t *testing.T) {
	log.SetLevel("debug")
	coreconfig.Reset()

	dispatched := make(chan *DispatchState)
	cancel, mdi, bp := newTestBatchProcessor(t, func(c context.Context, state *DispatchState) error {
		dispatched <- state
		return nil
	})
	defer cancel()
	bp.conf.BatchMaxBytes = batchSizeEstimateBase + (&core.Message{}).EstimateSize(false) + 100
	mockRunAsGroupPassthrough(mdi)
	mdi.On("UpdateMessages", mock.Anything, "ns1", mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mth := bp.txHelper.(*txcommonmocks.Helper)
	mth.On("SubmitNewTransaction", mock.Anything, core.TransactionTypeBatchPin, core.IdempotencyKey("")).Return(fftypes.NewUUID(), nil)

	mdm := bp.data.(*datamocks.Manager)
	mdm.On("UpdateMessageIfCached", mock.Anything, mock.Anything).Return()

	mim := bp.bm.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalNode", mock.Anything).Return(&core.Identity{}, nil)

	// Dispatch the work
	msgIDs := []*fftypes.UUID{fftypes.NewUUID(), fftypes.NewUUID()}
	go func() {
		for i := 0; i < 2; i++ {
			bp.newWork <- &batchWork{
				msg: &core.Message{Header: core.MessageHeader{ID: msgIDs[i]}, Sequence: int64(1000 + i)},
			}
		}
	}()

	// Wait for the confirmations, and the dispatch
	batch1 := <-dispatched
	batch2 := <-dispatched

	// Check we got all messages across two batches
	assert.Equal(t, 1, len(batch1.Messages))
	assert.Equal(t, msgIDs[0], batch1.Messages[0].Header.ID)
	assert.Equal(t, 1, len(batch2.Messages))
	assert.Equal(t, msgIDs[1], batch2.Messages[0].Header.ID)

	bp.cancelCtx()
	<-bp.done

	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestCloseToUnblockDispatch(t *testing.T) {
	cancel, _, bp := newTestBatchProcessor(t, func(c context.Context, state *DispatchState) error {
		return fmt.Errorf("pop")
	})
	defer cancel()
	bp.cancelCtx()
	bp.dispatchBatch(&DispatchState{})
	<-bp.done
}

func TestCloseToUnblockUpsertBatch(t *testing.T) {

	cancel, mdi, bp := newTestBatchProcessor(t, func(c context.Context, state *DispatchState) error {
		return nil
	})
	defer cancel()
	bp.retry.MaximumDelay = 1 * time.Microsecond
	bp.conf.BatchMaxSize = 1
	bp.conf.BatchTimeout = 100 * time.Second
	mockRunAsGroupPassthrough(mdi)
	waitForCall := make(chan bool)
	mth := bp.txHelper.(*txcommonmocks.Helper)
	mth.On("SubmitNewTransaction", mock.Anything, core.TransactionTypeBatchPin, core.IdempotencyKey("")).
		Run(func(a mock.Arguments) {
			waitForCall <- true
			<-waitForCall
		}).
		Return(nil, fmt.Errorf("pop"))

	mim := bp.bm.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalNode", mock.Anything).Return(&core.Identity{}, nil)

	// Generate the work
	msgid := fftypes.NewUUID()
	go func() {
		bp.newWork <- &batchWork{
			msg: &core.Message{Header: core.MessageHeader{ID: msgid}, Sequence: int64(1000)},
		}
	}()

	// Ensure the mock has been run
	<-waitForCall
	close(waitForCall)

	// Close to unblock
	bp.cancelCtx()
	<-bp.done

	mth.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestInsertNewNonceFail(t *testing.T) {
	cancel, _, bp := newTestBatchProcessor(t, func(c context.Context, state *DispatchState) error {
		return nil
	})
	defer cancel()
	bp.cancelCtx()
	mdi := bp.database.(*databasemocks.Plugin)
	mdi.On("GetNonce", mock.Anything, mock.Anything).Return(nil, nil)
	mdi.On("InsertNonce", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mockRunAsGroupPassthrough(mdi)

	mdm := bp.data.(*datamocks.Manager)
	mdm.On("UpdateMessageIfCached", mock.Anything, mock.Anything).Return()

	gid := fftypes.NewRandB32()
	err := bp.sealBatch(&DispatchState{
		Persisted: core.BatchPersisted{
			BatchHeader: core.BatchHeader{
				Group: gid,
			},
		},
		Messages: []*core.Message{
			{Header: core.MessageHeader{
				ID:     fftypes.NewUUID(),
				Group:  gid,
				Topics: fftypes.FFStringArray{"topic1"},
			}},
		},
	})
	assert.Regexp(t, "FF00154", err)

	<-bp.done

	mdi.AssertExpectations(t)
}

func TestUpdateExistingNonceFail(t *testing.T) {
	cancel, _, bp := newTestBatchProcessor(t, func(c context.Context, state *DispatchState) error {
		return nil
	})
	defer cancel()
	bp.cancelCtx()
	mdi := bp.database.(*databasemocks.Plugin)
	mdi.On("GetNonce", mock.Anything, mock.Anything).Return(&core.Nonce{
		Nonce: 12345,
	}, nil)
	mdi.On("UpdateNonce", mock.Anything, mock.MatchedBy(func(dbNonce *core.Nonce) bool {
		return dbNonce.Nonce == 12346
	})).Return(fmt.Errorf("pop"))
	mockRunAsGroupPassthrough(mdi)

	mdm := bp.data.(*datamocks.Manager)
	mdm.On("UpdateMessageIfCached", mock.Anything, mock.Anything).Return()

	gid := fftypes.NewRandB32()
	err := bp.sealBatch(&DispatchState{
		Persisted: core.BatchPersisted{
			BatchHeader: core.BatchHeader{
				Group: gid,
			},
		},
		Messages: []*core.Message{
			{Header: core.MessageHeader{
				ID:     fftypes.NewUUID(),
				Group:  gid,
				Topics: fftypes.FFStringArray{"topic1"},
			}},
		},
	})
	assert.Regexp(t, "FF00154", err)

	<-bp.done

	mdi.AssertExpectations(t)
}

func TestGetNonceFail(t *testing.T) {
	cancel, _, bp := newTestBatchProcessor(t, func(c context.Context, state *DispatchState) error {
		return nil
	})
	defer cancel()
	bp.cancelCtx()
	mdi := bp.database.(*databasemocks.Plugin)
	mdi.On("GetNonce", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	mockRunAsGroupPassthrough(mdi)

	mdm := bp.data.(*datamocks.Manager)
	mdm.On("UpdateMessageIfCached", mock.Anything, mock.Anything).Return()

	gid := fftypes.NewRandB32()
	err := bp.sealBatch(&DispatchState{
		Persisted: core.BatchPersisted{
			BatchHeader: core.BatchHeader{
				Group: gid,
			},
		},
		Messages: []*core.Message{
			{Header: core.MessageHeader{
				ID:     fftypes.NewUUID(),
				Group:  gid,
				Topics: fftypes.FFStringArray{"topic1"},
			}},
		},
	})
	assert.Regexp(t, "FF00154", err)

	<-bp.done

	mdi.AssertExpectations(t)
}

func TestGetNonceMigrationFail(t *testing.T) {
	cancel, _, bp := newTestBatchProcessor(t, func(c context.Context, state *DispatchState) error {
		return nil
	})
	defer cancel()
	bp.cancelCtx()
	mdi := bp.database.(*databasemocks.Plugin)
	mdi.On("GetNonce", mock.Anything, mock.Anything).Return(nil, nil).Once()
	mdi.On("GetNonce", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	mockRunAsGroupPassthrough(mdi)

	mdm := bp.data.(*datamocks.Manager)
	mdm.On("UpdateMessageIfCached", mock.Anything, mock.Anything).Return()

	gid := fftypes.NewRandB32()
	err := bp.sealBatch(&DispatchState{
		Persisted: core.BatchPersisted{
			BatchHeader: core.BatchHeader{
				Group: gid,
			},
		},
		Messages: []*core.Message{
			{Header: core.MessageHeader{
				ID:     fftypes.NewUUID(),
				Group:  gid,
				Topics: fftypes.FFStringArray{"topic1"},
			}},
		},
	})
	assert.Regexp(t, "FF00154", err)

	<-bp.done

	mdi.AssertExpectations(t)
}

func TestAddWorkInSort(t *testing.T) {
	cancel, _, bp := newTestBatchProcessor(t, func(c context.Context, state *DispatchState) error {
		return nil
	})
	defer cancel()
	bp.assemblyQueue = []*batchWork{
		{msg: &core.Message{Sequence: 200}},
		{msg: &core.Message{Sequence: 201}},
		{msg: &core.Message{Sequence: 202}},
		{msg: &core.Message{Sequence: 204}},
	}
	_, _ = bp.addWork(&batchWork{
		msg: &core.Message{Sequence: 203},
	})
	assert.Equal(t, []*batchWork{
		{msg: &core.Message{Sequence: 200}},
		{msg: &core.Message{Sequence: 201}},
		{msg: &core.Message{Sequence: 202}},
		{msg: &core.Message{Sequence: 203}},
		{msg: &core.Message{Sequence: 204}},
	}, bp.assemblyQueue)
}

func TestStartQuiesceNonBlocking(t *testing.T) {
	cancel, _, bp := newTestBatchProcessor(t, func(c context.Context, state *DispatchState) error {
		return nil
	})
	defer cancel()
	bp.startQuiesce()
	bp.startQuiesce() // we're just checking this doesn't hang
}

func TestMarkMessageDispatchedUnpinnedOK(t *testing.T) {
	log.SetLevel("debug")
	coreconfig.Reset()

	dispatched := make(chan *DispatchState)
	cancel, mdi, bp := newTestBatchProcessor(t, func(c context.Context, state *DispatchState) error {
		dispatched <- state
		return nil
	})
	defer cancel()
	bp.conf.txType = core.TransactionTypeUnpinned

	mockRunAsGroupPassthrough(mdi)
	mdi.On("UpdateMessages", mock.Anything, "ns1", mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Once()
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)

	mth := bp.txHelper.(*txcommonmocks.Helper)
	mth.On("SubmitNewTransaction", mock.Anything, core.TransactionTypeUnpinned, core.IdempotencyKey("")).Return(fftypes.NewUUID(), nil)

	mdm := bp.data.(*datamocks.Manager)
	mdm.On("UpdateMessageIfCached", mock.Anything, mock.Anything).Return()

	mim := bp.bm.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalNode", mock.Anything).Return(&core.Identity{}, nil)

	// Dispatch the work
	go func() {
		for i := 0; i < 5; i++ {
			msgid := fftypes.NewUUID()
			bp.newWork <- &batchWork{
				msg: &core.Message{Header: core.MessageHeader{ID: msgid, Topics: fftypes.FFStringArray{"topic1"}}, Sequence: int64(1000 + i)},
			}
		}
	}()

	// Wait for the confirmations, and the dispatch
	batch := <-dispatched

	// Check we got all the messages in a single batch
	assert.Equal(t, 5, len(batch.Messages))

	bp.cancelCtx()
	<-bp.done

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mth.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestMaskContextsRetryAfterPinsAssigned(t *testing.T) {
	log.SetLevel("debug")
	coreconfig.Reset()

	dispatched := make(chan *DispatchState)
	cancel, mdi, bp := newTestBatchProcessor(t, func(c context.Context, state *DispatchState) error {
		dispatched <- state
		return nil
	})
	defer cancel()

	mockRunAsGroupPassthrough(mdi)
	mdi.On("GetNonce", mock.Anything, mock.Anything).Return(&core.Nonce{
		Nonce: 12345,
	}, nil).Once()
	mdi.On("UpdateNonce", mock.Anything, mock.MatchedBy(func(dbNonce *core.Nonce) bool {
		return dbNonce.Nonce == 12347 // twice incremented
	})).Return(nil).Once()
	mdi.On("UpdateMessage", mock.Anything, "ns1", mock.Anything, mock.Anything).Return(nil).Twice()
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mdm := bp.data.(*datamocks.Manager)
	mdm.On("UpdateMessageIfCached", mock.Anything, mock.Anything).Return()

	mth := bp.txHelper.(*txcommonmocks.Helper)
	mth.On("SubmitNewTransaction", mock.Anything, core.TransactionTypeBatchPin, core.IdempotencyKey("")).Return(fftypes.NewUUID(), nil)

	mim := bp.bm.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalNode", mock.Anything).Return(&core.Identity{}, nil)

	groupID := fftypes.NewRandB32()
	msg1 := &core.Message{
		Header: core.MessageHeader{
			ID:     fftypes.NewUUID(),
			Type:   core.MessageTypePrivate,
			Group:  groupID,
			Topics: fftypes.FFStringArray{"topic1"},
		},
	}
	msg2 := &core.Message{
		Header: core.MessageHeader{
			ID:     fftypes.NewUUID(),
			Type:   core.MessageTypePrivate,
			Group:  groupID,
			Topics: fftypes.FFStringArray{"topic1"},
		},
	}

	state := bp.initFlushState(fftypes.NewUUID(), []*batchWork{{msg: msg1}, {msg: msg2}})
	err := bp.sealBatch(state)
	assert.NoError(t, err)

	// Second time there should be no additional calls, because now the messages
	// have pins in there that have been written to the database.
	err = bp.sealBatch(state)
	assert.NoError(t, err)

	bp.cancelCtx()
	<-bp.done

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestMaskContextsUpdateMessageFail(t *testing.T) {
	log.SetLevel("debug")
	coreconfig.Reset()

	dispatched := make(chan *DispatchState)
	cancel, mdi, bp := newTestBatchProcessor(t, func(c context.Context, state *DispatchState) error {
		dispatched <- state
		return nil
	})
	cancel()

	mockRunAsGroupPassthrough(mdi)
	mdi.On("GetNonce", mock.Anything, mock.Anything).Return(nil, nil)
	mdi.On("InsertNonce", mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpdateMessage", mock.Anything, "ns1", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Once()

	mim := bp.bm.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalNode", mock.Anything).Return(&core.Identity{}, nil)

	msg := &core.Message{
		Header: core.MessageHeader{
			ID:     fftypes.NewUUID(),
			Type:   core.MessageTypePrivate,
			Group:  fftypes.NewRandB32(),
			Topics: fftypes.FFStringArray{"topic1"},
		},
	}

	state := bp.initFlushState(fftypes.NewUUID(), []*batchWork{{msg: msg}})
	err := bp.sealBatch(state)
	assert.Regexp(t, "FF00154", err)

	bp.cancelCtx()
	<-bp.done

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestBigBatchEstimate(t *testing.T) {
	log.SetLevel("debug")
	coreconfig.Reset()

	bd := []byte(`{
		"id": "37ba893b-fcfa-4cf9-8ce8-34cd8bc9bc72",
		"type": "broadcast",
		"namespace": "default",
		"node": "248ba775-f595-40a6-a989-c2f2faae2dea",
		"author": "did:firefly:org/org_0",
		"key": "0x7e3bb2198959d3a1c3ede9db1587560320ce8998",
		"Group": null,
		"created": "2022-03-18T14:57:33.228374398Z",
		"hash": "7c620c12207ec153afea75d958de3edf601beced2570c798ebc246c2c44a5f66",
		"payload": {
		  "tx": {
			"type": "batch_pin",
			"id": "8d3f06b8-adb5-4745-a536-a9e262fd2e9f"
		  },
		  "messages": [
			{
			  "header": {
				"id": "2b393190-28e7-4b86-8af6-00906e94989b",
				"type": "broadcast",
				"txtype": "batch_pin",
				"author": "did:firefly:org/org_0",
				"key": "0x7e3bb2198959d3a1c3ede9db1587560320ce8998",
				"created": "2022-03-18T14:57:32.209734225Z",
				"namespace": "default",
				"topics": [
				  "default"
				],
				"tag": "perf_02e01e12-b918-4982-8407-2f9a08d673f3_740",
				"datahash": "b5b0c398450707b885f5973248ffa9a542f4c2f54860eba6c2d7aee48d0f9109"
			  },
			  "hash": "5fc430f1c8134c6c32c4e34ef65984843bb77bb19e73c862d464669537d96dbd",
			  "data": [
				{
				  "id": "147743b4-bd23-4da1-bd21-90c4ad9f1650",
				  "hash": "8ed265110f60711f79de1bc87b476e00bd8f8be436cdda3cf27fbf886d5e6ce6"
				}
			  ]
			}
		  ],
		  "data": [
			{
			  "id": "147743b4-bd23-4da1-bd21-90c4ad9f1650",
			  "validator": "json",
			  "namespace": "default",
			  "hash": "8ed265110f60711f79de1bc87b476e00bd8f8be436cdda3cf27fbf886d5e6ce6",
			  "created": "2022-03-18T14:57:32.209705277Z",
			  "value": {
				"broadcastID": "740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740740"
			  }
			}
		  ]
		}
	  }`)
	var batch core.Batch
	err := json.Unmarshal(bd, &batch)
	assert.NoError(t, err)

	sizeEstimate := batchSizeEstimateBase
	for i, m := range batch.Payload.Messages {
		dataJSONSize := 0
		bw := &batchWork{
			msg: m,
		}
		for _, dr := range m.Data {
			for _, d := range batch.Payload.Data {
				if d.ID.Equals(dr.ID) {
					bw.data = append(bw.data, d)
					break
				}
			}
			bd, err := json.Marshal(&bw.data)
			assert.NoError(t, err)
			dataJSONSize += len(bd)
		}
		md, err := json.Marshal(&bw.msg)
		assert.NoError(t, err)
		msgJSONSize := len(md)
		t.Logf("Msg=%.3d/%s Estimate=%d JSON - Msg=%d Data=%d Total=%d", i, m.Header.ID, bw.estimateSize(), msgJSONSize, dataJSONSize, msgJSONSize+dataJSONSize)
		sizeEstimate += bw.estimateSize()
	}

	assert.Greater(t, sizeEstimate, int64(len(bd)))
}
