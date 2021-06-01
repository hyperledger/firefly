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

package broadcast

import (
	"context"
	"fmt"
	"testing"

	"github.com/kaleido-io/firefly/mocks/batchmocks"
	"github.com/kaleido-io/firefly/mocks/blockchainmocks"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/mocks/datamocks"
	"github.com/kaleido-io/firefly/mocks/publicstoragemocks"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestBroadcast(t *testing.T) (*broadcastManager, func()) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mbi := &blockchainmocks.Plugin{}
	mpi := &publicstoragemocks.Plugin{}
	mb := &batchmocks.Manager{}
	mb.On("RegisterDispatcher", fftypes.MessageTypeBroadcast, mock.Anything, mock.Anything).Return()
	mb.On("RegisterDispatcher", fftypes.MessageTypeDefinition, mock.Anything, mock.Anything).Return()
	ctx, cancel := context.WithCancel(context.Background())
	b, err := NewBroadcastManager(ctx, "0x12345", mdi, mdm, mbi, mpi, mb)
	assert.NoError(t, err)
	return b.(*broadcastManager), cancel
}

func TestInitFail(t *testing.T) {
	_, err := NewBroadcastManager(context.Background(), "0x12345", nil, nil, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestBroadcastMessageGood(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	msg := &fftypes.Message{}
	bm.database.(*databasemocks.Plugin).On("UpsertMessage", mock.Anything, msg, false, false).Return(nil)

	err := bm.broadcastMessageCommon(context.Background(), msg)
	assert.NoError(t, err)

	bm.Start()
	bm.WaitStop()
}

func TestBroadcastMessageBad(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	dupID := fftypes.NewUUID()
	msg := &fftypes.Message{
		Data: fftypes.DataRefs{
			{ID: dupID /* missing hash */},
		},
	}
	bm.database.(*databasemocks.Plugin).On("UpsertMessage", mock.Anything, msg, false).Return(nil)

	err := bm.broadcastMessageCommon(context.Background(), msg)
	assert.Regexp(t, "FF10144", err)

}

func TestDispatchBatchInvalidData(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	err := bm.dispatchBatch(context.Background(), &fftypes.Batch{
		Payload: fftypes.BatchPayload{
			Data: []*fftypes.Data{
				{Value: fftypes.Byteable(`!json`)},
			},
		},
	})
	assert.Regexp(t, "FF10137", err)
}

func TestDispatchBatchUploadFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	bm.publicstorage.(*publicstoragemocks.Plugin).On("PublishData", mock.Anything, mock.Anything).Return(nil, "", fmt.Errorf("pop"))

	err := bm.dispatchBatch(context.Background(), &fftypes.Batch{})
	assert.EqualError(t, err, "pop")
}

func TestDispatchBatchSubmitBroadcastBatchSucceed(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	dbMocks := bm.database.(*databasemocks.Plugin)
	dbMocks.On("RunAsGroup", mock.Anything, mock.Anything).Return(nil)

	bm.publicstorage.(*publicstoragemocks.Plugin).On("PublishData", mock.Anything, mock.Anything).Return(fftypes.NewRandB32(), "id1", nil)

	err := bm.dispatchBatch(context.Background(), &fftypes.Batch{})
	assert.NoError(t, err)
}

func TestDispatchBatchSubmitBroadcastBatchFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	dbMocks := bm.database.(*databasemocks.Plugin)
	dbMocks.On("RunAsGroup", mock.Anything, mock.Anything).Return(nil)

	bm.publicstorage.(*publicstoragemocks.Plugin).On("PublishData", mock.Anything, mock.Anything).Return(fftypes.NewRandB32(), "id1", nil)

	err := bm.dispatchBatch(context.Background(), &fftypes.Batch{})
	assert.NoError(t, err)

	dbMocks.On("UpsertTransaction", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))
	fn := dbMocks.Calls[0].Arguments[1].(func(ctx context.Context) error)
	err = fn(context.Background())
	assert.Regexp(t, "pop", err)
}

func TestSubmitTXAndUpdateDBUpdateBatchFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	dbMocks := bm.database.(*databasemocks.Plugin)
	dbMocks.On("UpsertTransaction", mock.Anything, mock.Anything, true, false).Return(nil)
	dbMocks.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	bm.blockchain.(*blockchainmocks.Plugin).On("SubmitBroadcastBatch", mock.Anything, mock.Anything, mock.Anything).Return("", fmt.Errorf("pop"))

	err := bm.submitTXAndUpdateDB(context.Background(), &fftypes.Batch{}, fftypes.NewRandB32(), "id1")
	assert.Regexp(t, "pop", err)
}

func TestSubmitTXAndUpdateDBSubmitFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	dbMocks := bm.database.(*databasemocks.Plugin)
	dbMocks.On("UpsertTransaction", mock.Anything, mock.Anything, true, false).Return(nil)
	dbMocks.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	bm.blockchain.(*blockchainmocks.Plugin).On("SubmitBroadcastBatch", mock.Anything, mock.Anything, mock.Anything).Return("", fmt.Errorf("pop"))

	err := bm.submitTXAndUpdateDB(context.Background(), &fftypes.Batch{}, fftypes.NewRandB32(), "id1")
	assert.Regexp(t, "pop", err)
}

func TestSubmitTXAndUpdateDBAddOp1Fail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	dbMocks := bm.database.(*databasemocks.Plugin)
	dbMocks.On("UpsertTransaction", mock.Anything, mock.Anything, true, false).Return(nil)
	dbMocks.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	dbMocks.On("UpsertOperation", mock.Anything, mock.Anything, false).Return(fmt.Errorf("pop"))

	blkMocks := bm.blockchain.(*blockchainmocks.Plugin)
	blkMocks.On("SubmitBroadcastBatch", mock.Anything, mock.Anything, mock.Anything).Return("txid", nil)
	blkMocks.On("Name").Return("unittest")

	batch := &fftypes.Batch{
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{
					ID: fftypes.NewUUID(),
				}},
			},
		},
	}

	err := bm.submitTXAndUpdateDB(context.Background(), batch, fftypes.NewRandB32(), "id1")
	assert.Regexp(t, "pop", err)
}

func TestSubmitTXAndUpdateDBAddOp2Fail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	dbMocks := bm.database.(*databasemocks.Plugin)
	dbMocks.On("UpsertTransaction", mock.Anything, mock.Anything, true, false).Return(nil)
	dbMocks.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	dbMocks.On("UpsertOperation", mock.Anything, mock.Anything, false).Once().Return(nil)
	dbMocks.On("UpsertOperation", mock.Anything, mock.Anything, false).Once().Return(fmt.Errorf("pop"))

	blkMocks := bm.blockchain.(*blockchainmocks.Plugin)
	blkMocks.On("SubmitBroadcastBatch", mock.Anything, mock.Anything, mock.Anything).Return("txid", nil)
	blkMocks.On("Name").Return("ut_blockchain")

	bm.publicstorage.(*publicstoragemocks.Plugin).On("Name").Return("ut_publicstorage")

	batch := &fftypes.Batch{
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{
					ID: fftypes.NewUUID(),
				}},
			},
		},
	}

	err := bm.submitTXAndUpdateDB(context.Background(), batch, fftypes.NewRandB32(), "id1")
	assert.Regexp(t, "pop", err)
}

func TestSubmitTXAndUpdateDBSucceed(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	dbMocks := bm.database.(*databasemocks.Plugin)
	dbMocks.On("UpsertTransaction", mock.Anything, mock.Anything, true, false).Return(nil)
	dbMocks.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	dbMocks.On("UpsertOperation", mock.Anything, mock.Anything, false).Once().Return(nil)
	dbMocks.On("UpsertOperation", mock.Anything, mock.Anything, false).Once().Return(nil)

	blkMocks := bm.blockchain.(*blockchainmocks.Plugin)
	blkMocks.On("SubmitBroadcastBatch", mock.Anything, mock.Anything, mock.Anything).Return("blockchain_id", nil)
	blkMocks.On("Name").Return("ut_blockchain")

	bm.publicstorage.(*publicstoragemocks.Plugin).On("Name").Return("ut_publicstorage")

	msgID := fftypes.NewUUID()
	batch := &fftypes.Batch{
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{
					ID: msgID,
				}},
			},
		},
	}

	err := bm.submitTXAndUpdateDB(context.Background(), batch, fftypes.NewRandB32(), "ipfs_id")
	assert.NoError(t, err)

	op1 := dbMocks.Calls[2].Arguments[1].(*fftypes.Operation)
	assert.Equal(t, *batch.Payload.TX.ID, *op1.Transaction)
	assert.Equal(t, "ut_blockchain", op1.Plugin)
	assert.Equal(t, "blockchain_id", op1.BackendID)
	assert.Equal(t, fftypes.OpTypeBlockchainBatchPin, op1.Type)

	op2 := dbMocks.Calls[3].Arguments[1].(*fftypes.Operation)
	assert.Equal(t, *batch.Payload.TX.ID, *op2.Transaction)
	assert.Equal(t, "ut_publicstorage", op2.Plugin)
	assert.Equal(t, "ipfs_id", op2.BackendID)
	assert.Equal(t, fftypes.OpTypePublicStorageBatchBroadcast, op2.Type)
}
