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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSharedStorageBatchDownloadedOk(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	b, _ := json.Marshal(&batch)

	mdi := em.database.(*databasemocks.Plugin)
	mss := em.sharedstorage.(*sharedstoragemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything, mock.AnythingOfType("database.PostCompletionHook")).Return(nil, nil).Run(func(args mock.Arguments) {
		args[2].(database.PostCompletionHook)()
	})
	mss.On("Name").Return("utdx").Maybe()
	mdm := em.data.(*datamocks.Manager)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	bid, err := em.SharedStorageBatchDownloaded(mss, "payload1", b)
	assert.NoError(t, err)
	assert.Equal(t, batch.ID, bid)

	brw := <-em.aggregator.rewinder.rewindRequests
	assert.Equal(t, *batch.ID, brw.uuid)

	mdi.AssertExpectations(t)
	mss.AssertExpectations(t)
	mdm.AssertExpectations(t)

}

func TestSharedStorageBatchDownloadedPersistFail(t *testing.T) {

	em, cancel := newTestEventManager(t)
	cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	b, _ := json.Marshal(&batch)

	mdi := em.database.(*databasemocks.Plugin)
	mss := em.sharedstorage.(*sharedstoragemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(fmt.Errorf("pop"))
	mss.On("Name").Return("utdx").Maybe()

	_, err := em.SharedStorageBatchDownloaded(mss, "payload1", b)
	assert.Regexp(t, "FF00154", err)

	mdi.AssertExpectations(t)
	mss.AssertExpectations(t)

}

func TestSharedStorageBatchDownloadedNSMismatch(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	b, _ := json.Marshal(&batch)

	mss := em.sharedstorage.(*sharedstoragemocks.Plugin)
	mss.On("Name").Return("utdx").Maybe()

	em.namespace = "ns2"
	_, err := em.SharedStorageBatchDownloaded(mss, "payload1", b)
	assert.NoError(t, err)

	mss.AssertExpectations(t)

}

func TestSharedStorageBatchDownloadedBadData(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	mss := em.sharedstorage.(*sharedstoragemocks.Plugin)
	mss.On("Name").Return("utdx").Maybe()

	_, err := em.SharedStorageBatchDownloaded(mss, "payload1", []byte("!json"))
	assert.NoError(t, err)

	mss.AssertExpectations(t)

}

func TestSharedStorageBlobDownloadedOk(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mss := em.sharedstorage.(*sharedstoragemocks.Plugin)
	mss.On("Name").Return("utsd")
	mdi.On("GetBlobs", em.ctx, mock.Anything).Return([]*core.Blob{}, nil, nil)
	mdi.On("InsertBlobs", em.ctx, mock.Anything).Return(nil, nil)

	hash := fftypes.NewRandB32()
	em.SharedStorageBlobDownloaded(mss, *hash, 12345, "payload1")

	brw := <-em.aggregator.rewinder.rewindRequests
	assert.Equal(t, rewind{hash: *hash, rewindType: rewindBlob}, brw)

	mdi.AssertExpectations(t)
	mss.AssertExpectations(t)

}
