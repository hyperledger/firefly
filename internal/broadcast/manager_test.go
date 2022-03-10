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

	"github.com/hyperledger/firefly/internal/batch"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/mocks/batchmocks"
	"github.com/hyperledger/firefly/mocks/batchpinmocks"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestBroadcastCommon(t *testing.T, metricsEnabled bool) (*broadcastManager, func()) {
	config.Reset()
	mdi := &databasemocks.Plugin{}
	mim := &identitymanagermocks.Manager{}
	mdm := &datamocks.Manager{}
	mbi := &blockchainmocks.Plugin{}
	mpi := &sharedstoragemocks.Plugin{}
	mba := &batchmocks.Manager{}
	mdx := &dataexchangemocks.Plugin{}
	msa := &syncasyncmocks.Bridge{}
	mbp := &batchpinmocks.Submitter{}
	mmi := &metricsmocks.Manager{}
	mom := &operationmocks.Manager{}
	mmi.On("IsMetricsEnabled").Return(metricsEnabled)
	mbi.On("Name").Return("ut_blockchain").Maybe()
	mpi.On("Name").Return("ut_sharedstorage").Maybe()
	mba.On("RegisterDispatcher",
		broadcastDispatcherName,
		fftypes.TransactionTypeBatchPin,
		[]fftypes.MessageType{
			fftypes.MessageTypeBroadcast,
			fftypes.MessageTypeDefinition,
			fftypes.MessageTypeTransferBroadcast,
		}, mock.Anything, mock.Anything).Return()
	mom.On("RegisterHandler", mock.Anything, mock.Anything, mock.Anything)

	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	b, err := NewBroadcastManager(ctx, mdi, mim, mdm, mbi, mdx, mpi, mba, msa, mbp, mmi, mom)
	assert.NoError(t, err)
	return b.(*broadcastManager), cancel
}

func newTestBroadcast(t *testing.T) (*broadcastManager, func()) {
	return newTestBroadcastCommon(t, false)
}

func newTestBroadcastWithMetrics(t *testing.T) (*broadcastManager, func()) {
	bm, cancel := newTestBroadcastCommon(t, true)
	mmi := bm.metrics.(*metricsmocks.Manager)
	mmi.On("MessageSubmitted", mock.Anything).Return()
	return bm, cancel
}

func TestInitFail(t *testing.T) {
	_, err := NewBroadcastManager(context.Background(), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestName(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	assert.Equal(t, "BroadcastManager", bm.Name())
}

func TestBroadcastMessageGood(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	dataID := fftypes.NewUUID()
	dataHash := fftypes.NewRandB32()
	newMsg := &data.NewMessage{
		Message: &fftypes.MessageInOut{
			Message: fftypes.Message{
				Header: fftypes.MessageHeader{
					ID: fftypes.NewUUID(),
				},
				Data: fftypes.DataRefs{
					{ID: dataID, Hash: dataHash},
				},
			},
		},
		ResolvedData: data.Resolved{
			AllData: fftypes.DataArray{
				{ID: dataID, Hash: dataHash},
			},
		},
	}

	mdm := bm.data.(*datamocks.Manager)
	mdm.On("WriteNewMessage", mock.Anything, newMsg).Return(nil)

	broadcast := broadcastSender{
		mgr: bm,
		msg: newMsg,
	}
	err := broadcast.sendInternal(context.Background(), methodSend)
	assert.NoError(t, err)

	bm.Start()
	bm.WaitStop()

	mdm.AssertExpectations(t)
}

func TestBroadcastMessageBad(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	newMsg := &data.NewMessage{
		Message: &fftypes.MessageInOut{
			Message: fftypes.Message{
				Header: fftypes.MessageHeader{
					ID: fftypes.NewUUID(),
				},
				Data: fftypes.DataRefs{
					{ID: fftypes.NewUUID(), Hash: nil},
				},
			},
		},
	}

	broadcast := broadcastSender{
		mgr: bm,
		msg: newMsg,
	}
	err := broadcast.sendInternal(context.Background(), methodSend)
	assert.Regexp(t, "FF10144", err)

}

func TestDispatchBatchInsertOpFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	state := &batch.DispatchState{
		Pins: []*fftypes.Bytes32{fftypes.NewRandB32()},
	}

	mom := bm.operations.(*operationmocks.Manager)
	mom.On("AddOrReuseOperation", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := bm.dispatchBatch(context.Background(), state)
	assert.EqualError(t, err, "pop")

	mom.AssertExpectations(t)
}

func TestDispatchBatchUploadFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	state := &batch.DispatchState{
		Persisted: fftypes.BatchPersisted{
			BatchHeader: fftypes.BatchHeader{
				ID: fftypes.NewUUID(),
			},
		},
		Pins: []*fftypes.Bytes32{fftypes.NewRandB32()},
	}

	mom := bm.operations.(*operationmocks.Manager)
	mom.On("AddOrReuseOperation", mock.Anything, mock.Anything).Return(nil)
	mom.On("RunOperation", mock.Anything, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(batchBroadcastData)
		return op.Type == fftypes.OpTypeSharedStorageBatchBroadcast && data.Batch.ID.Equals(state.Persisted.ID)
	})).Return(fmt.Errorf("pop"))

	err := bm.dispatchBatch(context.Background(), state)
	assert.EqualError(t, err, "pop")

	mom.AssertExpectations(t)
}

func TestDispatchBatchSubmitBatchPinSucceed(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	state := &batch.DispatchState{
		Persisted: fftypes.BatchPersisted{
			BatchHeader: fftypes.BatchHeader{
				ID: fftypes.NewUUID(),
			},
		},
		Pins: []*fftypes.Bytes32{fftypes.NewRandB32()},
	}

	mdi := bm.database.(*databasemocks.Plugin)
	mbp := bm.batchpin.(*batchpinmocks.Submitter)
	mom := bm.operations.(*operationmocks.Manager)
	mom.On("AddOrReuseOperation", mock.Anything, mock.Anything).Return(nil)
	mbp.On("SubmitPinnedBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mom.On("RunOperation", mock.Anything, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(batchBroadcastData)
		return op.Type == fftypes.OpTypeSharedStorageBatchBroadcast && data.Batch.ID.Equals(state.Persisted.ID)
	})).Return(nil)

	err := bm.dispatchBatch(context.Background(), state)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mbp.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestDispatchBatchSubmitBroadcastFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	state := &batch.DispatchState{
		Persisted: fftypes.BatchPersisted{
			BatchHeader: fftypes.BatchHeader{
				ID:        fftypes.NewUUID(),
				SignerRef: fftypes.SignerRef{Author: "wrong", Key: "wrong"},
			},
		},
		Pins: []*fftypes.Bytes32{fftypes.NewRandB32()},
	}

	mdi := bm.database.(*databasemocks.Plugin)
	mbp := bm.batchpin.(*batchpinmocks.Submitter)
	mom := bm.operations.(*operationmocks.Manager)
	mom.On("AddOrReuseOperation", mock.Anything, mock.Anything).Return(nil)
	mbp.On("SubmitPinnedBatch", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mom.On("RunOperation", mock.Anything, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(batchBroadcastData)
		return op.Type == fftypes.OpTypeSharedStorageBatchBroadcast && data.Batch.ID.Equals(state.Persisted.ID)
	})).Return(nil)

	err := bm.dispatchBatch(context.Background(), state)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mbp.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestPublishBlobsPublishFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdx := bm.exchange.(*dataexchangemocks.Plugin)
	mps := bm.sharedstorage.(*sharedstoragemocks.Plugin)
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

	err := bm.publishBlobs(ctx, &data.NewMessage{
		ResolvedData: data.Resolved{
			DataToPublish: []*fftypes.DataAndBlob{
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
			},
		},
	})
	assert.EqualError(t, err, "pop")

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

	err := bm.publishBlobs(ctx, &data.NewMessage{
		ResolvedData: data.Resolved{
			DataToPublish: []*fftypes.DataAndBlob{
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
			},
		},
	})
	assert.Regexp(t, "FF10240", err)

	mdi.AssertExpectations(t)
}
