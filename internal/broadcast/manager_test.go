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

package broadcast

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/mocks/batchmocks"
	"github.com/hyperledger/firefly/mocks/batchpinmocks"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/publicstoragemocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestBroadcast(t *testing.T) (*broadcastManager, func()) {
	config.Reset()
	mdi := &databasemocks.Plugin{}
	mim := &identitymanagermocks.Manager{}
	mdm := &datamocks.Manager{}
	mbi := &blockchainmocks.Plugin{}
	mpi := &publicstoragemocks.Plugin{}
	mba := &batchmocks.Manager{}
	mdx := &dataexchangemocks.Plugin{}
	msa := &syncasyncmocks.Bridge{}
	mbp := &batchpinmocks.Submitter{}
	mmi := &metricsmocks.Manager{}
	mmi.On("IsMetricsEnabled").Return(false)
	mbi.On("Name").Return("ut_blockchain").Maybe()
	mpi.On("Name").Return("ut_publicstorage").Maybe()
	mba.On("RegisterDispatcher", []fftypes.MessageType{
		fftypes.MessageTypeBroadcast,
		fftypes.MessageTypeDefinition,
		fftypes.MessageTypeTransferBroadcast,
	}, mock.Anything, mock.Anything).Return()

	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	b, err := NewBroadcastManager(ctx, mdi, mim, mdm, mbi, mdx, mpi, mba, msa, mbp, mmi)
	assert.NoError(t, err)
	return b.(*broadcastManager), cancel
}

func newTestBroadcastWithMetrics(t *testing.T) (*broadcastManager, func()) {
	config.Reset()
	mdi := &databasemocks.Plugin{}
	mim := &identitymanagermocks.Manager{}
	mdm := &datamocks.Manager{}
	mbi := &blockchainmocks.Plugin{}
	mpi := &publicstoragemocks.Plugin{}
	mba := &batchmocks.Manager{}
	mdx := &dataexchangemocks.Plugin{}
	msa := &syncasyncmocks.Bridge{}
	mbp := &batchpinmocks.Submitter{}
	mmi := &metricsmocks.Manager{}
	mmi.On("MessageSubmitted", mock.Anything).Return()
	mmi.On("IsMetricsEnabled").Return(true)
	mbi.On("Name").Return("ut_blockchain").Maybe()
	mpi.On("Name").Return("ut_publicstorage").Maybe()
	mba.On("RegisterDispatcher", []fftypes.MessageType{
		fftypes.MessageTypeBroadcast,
		fftypes.MessageTypeDefinition,
		fftypes.MessageTypeTransferBroadcast,
	}, mock.Anything, mock.Anything).Return()

	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	b, err := NewBroadcastManager(ctx, mdi, mim, mdm, mbi, mdx, mpi, mba, msa, mbp, mmi)
	assert.NoError(t, err)
	return b.(*broadcastManager), cancel
}

func TestInitFail(t *testing.T) {
	_, err := NewBroadcastManager(context.Background(), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestBroadcastMessageGood(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	msg := &fftypes.MessageInOut{}
	bm.database.(*databasemocks.Plugin).On("UpsertMessage", mock.Anything, &msg.Message, database.UpsertOptimizationNew).Return(nil)

	broadcast := broadcastSender{
		mgr: bm,
		msg: msg,
	}
	err := broadcast.sendInternal(context.Background(), methodSend)
	assert.NoError(t, err)

	bm.Start()
	bm.WaitStop()
}

func TestBroadcastMessageBad(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	dupID := fftypes.NewUUID()
	msg := &fftypes.MessageInOut{
		Message: fftypes.Message{
			Data: fftypes.DataRefs{
				{ID: dupID /* missing hash */},
			},
		},
	}
	bm.database.(*databasemocks.Plugin).On("UpsertMessage", mock.Anything, msg, false).Return(nil)

	broadcast := broadcastSender{
		mgr: bm,
		msg: msg,
	}
	err := broadcast.sendInternal(context.Background(), methodSend)
	assert.Regexp(t, "FF10144", err)

}

func TestDispatchBatchInvalidData(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	err := bm.dispatchBatch(context.Background(), &fftypes.Batch{
		Payload: fftypes.BatchPayload{
			Data: []*fftypes.Data{
				{Value: fftypes.JSONAnyPtr(`!json`)},
			},
		},
	}, []*fftypes.Bytes32{fftypes.NewRandB32()})
	assert.Regexp(t, "FF10137", err)
}

func TestDispatchBatchUploadFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	bm.publicstorage.(*publicstoragemocks.Plugin).On("PublishData", mock.Anything, mock.Anything).Return("", fmt.Errorf("pop"))

	err := bm.dispatchBatch(context.Background(), &fftypes.Batch{}, []*fftypes.Bytes32{fftypes.NewRandB32()})
	assert.EqualError(t, err, "pop")
}

func TestDispatchBatchSubmitBatchPinSucceed(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
	}

	mdi := bm.database.(*databasemocks.Plugin)
	mps := bm.publicstorage.(*publicstoragemocks.Plugin)
	mbp := bm.batchpin.(*batchpinmocks.Submitter)
	mps.On("PublishData", mock.Anything, mock.Anything).Return("id1", nil)
	mdi.On("UpdateBatch", mock.Anything, batch.ID, mock.Anything).Return(nil)
	mdi.On("InsertOperation", mock.Anything, mock.Anything).Return(nil)
	mbp.On("SubmitPinnedBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := bm.dispatchBatch(context.Background(), batch, []*fftypes.Bytes32{fftypes.NewRandB32()})
	assert.NoError(t, err)
}

func TestDispatchBatchSubmitBroadcastFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	mdi := bm.database.(*databasemocks.Plugin)
	mps := bm.publicstorage.(*publicstoragemocks.Plugin)
	mbp := bm.batchpin.(*batchpinmocks.Submitter)
	mps.On("PublishData", mock.Anything, mock.Anything).Return("id1", nil)
	mdi.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertOperation", mock.Anything, mock.Anything).Return(nil)
	mbp.On("SubmitPinnedBatch", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := bm.dispatchBatch(context.Background(), &fftypes.Batch{Identity: fftypes.Identity{Author: "wrong", Key: "wrong"}}, []*fftypes.Bytes32{fftypes.NewRandB32()})
	assert.EqualError(t, err, "pop")
}

func TestSubmitTXAndUpdateDBUpdateBatchFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	bm.blockchain.(*blockchainmocks.Plugin).On("SubmitBatchPin", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", fmt.Errorf("pop"))

	err := bm.submitTXAndUpdateDB(context.Background(), &fftypes.Batch{Identity: fftypes.Identity{Author: "org1", Key: "0x12345"}}, []*fftypes.Bytes32{fftypes.NewRandB32()})
	assert.Regexp(t, "pop", err)
}

func TestSubmitTXAndUpdateDBAddOp1Fail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	mdi := bm.database.(*databasemocks.Plugin)
	mbi := bm.blockchain.(*blockchainmocks.Plugin)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertOperation", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mbi.On("SubmitBatchPin", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("txid", nil)
	mbi.On("Name").Return("unittest")

	batch := &fftypes.Batch{
		Identity: fftypes.Identity{Author: "org1", Key: "0x12345"},
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{
					ID: fftypes.NewUUID(),
				}},
			},
		},
	}

	err := bm.submitTXAndUpdateDB(context.Background(), batch, []*fftypes.Bytes32{fftypes.NewRandB32()})
	assert.Regexp(t, "pop", err)
}

func TestSubmitTXAndUpdateDBSucceed(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	mdi := bm.database.(*databasemocks.Plugin)
	mbi := bm.blockchain.(*blockchainmocks.Plugin)
	mbp := bm.batchpin.(*batchpinmocks.Submitter)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertOperation", mock.Anything, mock.Anything).Return(nil)
	mbi.On("SubmitBatchPin", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mbp.On("SubmitPinnedBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	msgID := fftypes.NewUUID()
	batch := &fftypes.Batch{
		Identity: fftypes.Identity{Author: "org1", Key: "0x12345"},
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
		PayloadRef: "ipfs_id",
	}

	err := bm.submitTXAndUpdateDB(context.Background(), batch, []*fftypes.Bytes32{fftypes.NewRandB32()})
	assert.NoError(t, err)

	op := mdi.Calls[1].Arguments[1].(*fftypes.Operation)
	assert.Equal(t, *batch.Payload.TX.ID, *op.Transaction)
	assert.Equal(t, "ut_publicstorage", op.Plugin)
	assert.Equal(t, fftypes.OpTypePublicStorageBatchBroadcast, op.Type)

}

func TestPublishBlobsUpdateDataFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdx := bm.exchange.(*dataexchangemocks.Plugin)
	mps := bm.publicstorage.(*publicstoragemocks.Plugin)
	mim := bm.identity.(*identitymanagermocks.Manager)

	blobHash := fftypes.NewRandB32()
	dataID := fftypes.NewUUID()

	ctx := context.Background()
	mdx.On("DownloadBLOB", ctx, "blob/1").Return(ioutil.NopCloser(bytes.NewReader([]byte(`some data`))), nil)
	mps.On("PublishData", ctx, mock.MatchedBy(func(reader io.ReadCloser) bool {
		b, err := ioutil.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, "some data", string(b))
		return true
	})).Return("payload-ref", nil)
	mdi.On("UpdateData", ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mim.On("ResolveInputIdentity", ctx, mock.Anything).Return(nil)

	err := bm.publishBlobs(ctx, []*fftypes.DataAndBlob{
		{
			Data: &fftypes.Data{
				ID: dataID,
				Blob: &fftypes.BlobRef{
					Hash: blobHash,
				},
			},
			Blob: &fftypes.Blob{
				Hash:       blobHash,
				PayloadRef: "blob/1",
			},
		},
	})
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestPublishBlobsPublishFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdx := bm.exchange.(*dataexchangemocks.Plugin)
	mps := bm.publicstorage.(*publicstoragemocks.Plugin)
	mim := bm.identity.(*identitymanagermocks.Manager)

	blobHash := fftypes.NewRandB32()
	dataID := fftypes.NewUUID()

	ctx := context.Background()
	mdx.On("DownloadBLOB", ctx, "blob/1").Return(ioutil.NopCloser(bytes.NewReader([]byte(`some data`))), nil)
	mps.On("PublishData", ctx, mock.MatchedBy(func(reader io.ReadCloser) bool {
		b, err := ioutil.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, "some data", string(b))
		return true
	})).Return("", fmt.Errorf("pop"))
	mim.On("ResolveInputIdentity", ctx, mock.Anything).Return(nil)

	err := bm.publishBlobs(ctx, []*fftypes.DataAndBlob{
		{
			Data: &fftypes.Data{
				ID: dataID,
				Blob: &fftypes.BlobRef{
					Hash: blobHash,
				},
			},
			Blob: &fftypes.Blob{
				Hash:       blobHash,
				PayloadRef: "blob/1",
			},
		},
	})
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestPublishBlobsDownloadFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdx := bm.exchange.(*dataexchangemocks.Plugin)
	mim := bm.identity.(*identitymanagermocks.Manager)

	blobHash := fftypes.NewRandB32()
	dataID := fftypes.NewUUID()

	ctx := context.Background()
	mdx.On("DownloadBLOB", ctx, "blob/1").Return(nil, fmt.Errorf("pop"))
	mim.On("ResolveInputIdentity", ctx, mock.Anything).Return(nil)

	err := bm.publishBlobs(ctx, []*fftypes.DataAndBlob{
		{
			Data: &fftypes.Data{
				ID: dataID,
				Blob: &fftypes.BlobRef{
					Hash: blobHash,
				},
			},
			Blob: &fftypes.Blob{
				Hash:       blobHash,
				PayloadRef: "blob/1",
			},
		},
	})
	assert.Regexp(t, "FF10240", err)

	mdi.AssertExpectations(t)
}
