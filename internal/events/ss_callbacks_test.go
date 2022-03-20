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

package events

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSharedStorageBatchDownloadedOk(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, fftypes.BatchTypeBroadcast, fftypes.TransactionTypeBatchPin, fftypes.DataArray{data})
	b, _ := json.Marshal(&batch)

	mdi := em.database.(*databasemocks.Plugin)
	mss := em.sharedstorage.(*sharedstoragemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything).Return(nil, nil)
	mss.On("Name").Return("utdx").Maybe()
	mdm := em.data.(*datamocks.Manager)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	bid, err := em.SharedStorageBatchDownloaded(mss, batch.Namespace, "payload1", b)
	assert.NoError(t, err)
	assert.Equal(t, batch.ID, bid)

	assert.Equal(t, *batch.ID, <-em.aggregator.rewindBatches)

	mdi.AssertExpectations(t)
	mss.AssertExpectations(t)
	mdm.AssertExpectations(t)

}

func TestSharedStorageBatchDownloadedPersistFail(t *testing.T) {

	em, cancel := newTestEventManager(t)
	cancel()

	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, fftypes.BatchTypeBroadcast, fftypes.TransactionTypeBatchPin, fftypes.DataArray{data})
	b, _ := json.Marshal(&batch)

	mdi := em.database.(*databasemocks.Plugin)
	mss := em.sharedstorage.(*sharedstoragemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(fmt.Errorf("pop"))
	mss.On("Name").Return("utdx").Maybe()

	_, err := em.SharedStorageBatchDownloaded(mss, batch.Namespace, "payload1", b)
	assert.Regexp(t, "FF10158", err)

	mdi.AssertExpectations(t)
	mss.AssertExpectations(t)

}

func TestSharedStorageBatchDownloadedNSMismatch(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, fftypes.BatchTypeBroadcast, fftypes.TransactionTypeBatchPin, fftypes.DataArray{data})
	b, _ := json.Marshal(&batch)

	mss := em.sharedstorage.(*sharedstoragemocks.Plugin)
	mss.On("Name").Return("utdx").Maybe()

	_, err := em.SharedStorageBatchDownloaded(mss, "srong", "payload1", b)
	assert.NoError(t, err)

	mss.AssertExpectations(t)

}

func TestSharedStorageBatchDownloadedBadData(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	mss := em.sharedstorage.(*sharedstoragemocks.Plugin)
	mss.On("Name").Return("utdx").Maybe()

	_, err := em.SharedStorageBatchDownloaded(mss, "srong", "payload1", []byte("!json"))
	assert.NoError(t, err)

	mss.AssertExpectations(t)

}

func TestSharedStorageBLOBDownloadedOk(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	dataID := fftypes.NewUUID()
	batchID := fftypes.NewUUID()

	mdi := em.database.(*databasemocks.Plugin)
	mss := em.sharedstorage.(*sharedstoragemocks.Plugin)
	mss.On("Name").Return("utsd")
	mdi.On("InsertBlob", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("GetDataRefs", em.ctx, mock.Anything).Return(fftypes.DataRefs{
		{ID: dataID},
	}, nil, nil)
	mdi.On("GetMessagesForData", em.ctx, dataID, mock.Anything).Return([]*fftypes.Message{
		{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}, BatchID: batchID},
	}, nil, nil)

	err := em.SharedStorageBLOBDownloaded(mss, *fftypes.NewRandB32(), 12345, "payload1")
	assert.NoError(t, err)

	assert.Equal(t, *batchID, <-em.aggregator.rewindBatches)

	mdi.AssertExpectations(t)
	mss.AssertExpectations(t)

}
