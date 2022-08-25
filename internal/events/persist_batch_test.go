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
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPersistBatch(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(fmt.Errorf(("pop")))

	org := newTestOrg("org1")
	orgBytes, err := json.Marshal(&org)
	assert.NoError(t, err)
	data := &core.Data{
		ID:        fftypes.NewUUID(),
		Value:     fftypes.JSONAnyPtrBytes(orgBytes),
		Validator: core.MessageTypeDefinition,
	}

	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: fftypes.NewUUID(),
			SignerRef: core.SignerRef{
				Author: "did:firefly:org/12345",
				Key:    "0x12345",
			},
		},
		Payload: core.BatchPayload{
			TX: core.TransactionRef{
				ID:   fftypes.NewUUID(),
				Type: core.TransactionTypeBatchPin,
			},
			Messages: []*core.Message{
				{
					Header: core.MessageHeader{
						ID:   fftypes.NewUUID(),
						Type: core.MessageTypeDefinition,
						SignerRef: core.SignerRef{
							Author: "did:firefly:org/12345",
							Key:    "0x12345",
						},
					},
					Data: core.DataRefs{
						{
							ID:   data.ID,
							Hash: data.Hash,
						},
					},
				},
			},
			Data: core.DataArray{
				data,
			},
		},
	}
	bp, _ := batch.Confirmed("peer1")
	batch.Hash = fftypes.HashString(bp.Manifest.String())

	_, _, err = em.persistBatch(em.ctx, "peer1", batch)
	assert.EqualError(t, err, "pop") // Confirms we got to upserting the batch

}

func TestPersistBatchNoCacheDataNotInBatch(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil)
	mdi.On("UpsertMessage", em.ctx, mock.Anything, database.UpsertOptimizationSkip, mock.AnythingOfType("database.PostCompletionHook")).Return(nil)

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	data.ID = fftypes.NewUUID()
	_ = data.Seal(em.ctx, nil)
	bp, _ := batch.Confirmed("peer1")
	batch.Hash = fftypes.HashString(bp.Manifest.String())

	_, valid, err := em.persistBatch(em.ctx, "peer1", batch)
	assert.False(t, valid)
	assert.NoError(t, err)

}

func TestPersistBatchExtraDataInBatch(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil)
	mdi.On("UpsertMessage", em.ctx, mock.Anything, database.UpsertOptimizationSkip, mock.AnythingOfType("database.PostCompletionHook")).Return(nil)

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	data2 := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test2"`)}
	_ = data2.Seal(em.ctx, nil)
	batch.Payload.Data = append(batch.Payload.Data, data2)
	bp, _ := batch.Confirmed("peer1")
	batch.Hash = fftypes.HashString(bp.Manifest.String())

	_, valid, err := em.persistBatch(em.ctx, "peer1", batch)
	assert.False(t, valid)
	assert.NoError(t, err)

}

func TestPersistBatchNilMessageEntryop(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	valid := em.validateBatchMessage(em.ctx, &core.Batch{}, 0, nil)
	assert.False(t, valid)

}

func TestPersistBatchContentSendByUsOK(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	batch.Node = testNodeID

	mdm := em.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", em.ctx, batch.Payload.Messages[0].Header.ID).Return(batch.Payload.Messages[0], batch.Payload.Data, true, nil)

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalNode", mock.Anything).Return(testNode, nil)

	ok, err := em.persistBatchContent(em.ctx, batch, []*messageAndData{})
	assert.NoError(t, err)
	assert.True(t, ok)

	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestPersistBatchContentSentByNil(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	batch.Node = nil

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("InsertDataArray", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything, mock.AnythingOfType("database.PostCompletionHook")).Return(nil, nil).Run(func(args mock.Arguments) {
		args[2].(database.PostCompletionHook)()
	})

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalNode", mock.Anything).Return(nil, nil)

	ok, err := em.persistBatchContent(em.ctx, batch, []*messageAndData{})
	assert.NoError(t, err)
	assert.True(t, ok)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)

}

func TestPersistBatchContentSentByUsNotFoundFallback(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	batch.Node = testNodeID

	mdm := em.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", em.ctx, batch.Payload.Messages[0].Header.ID).Return(nil, nil, false, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("InsertDataArray", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything, mock.AnythingOfType("database.PostCompletionHook")).Return(nil, nil).Run(func(args mock.Arguments) {
		args[2].(database.PostCompletionHook)()
	})

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalNode", mock.Anything).Return(testNode, nil)

	ok, err := em.persistBatchContent(em.ctx, batch, []*messageAndData{})
	assert.NoError(t, err)
	assert.True(t, ok)

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)

}

func TestPersistBatchContentSentByUsFoundMismatch(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	batch.Node = testNodeID

	mdm := em.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", em.ctx, batch.Payload.Messages[0].Header.ID).Return(&core.Message{
		Header: core.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}, nil, true, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("InsertDataArray", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertMessages", mock.Anything, mock.Anything, mock.AnythingOfType("database.PostCompletionHook")).Return(fmt.Errorf("optimization miss"))
	mdi.On("UpsertMessage", mock.Anything, mock.Anything, database.UpsertOptimizationExisting, mock.AnythingOfType("database.PostCompletionHook")).Return(database.HashMismatch)

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalNode", mock.Anything).Return(testNode, nil)

	ok, err := em.persistBatchContent(em.ctx, batch, []*messageAndData{})
	assert.NoError(t, err)
	assert.False(t, ok)

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)

}

func TestPersistBatchContentInsertMessagesFail(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("InsertDataArray", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertMessages", mock.Anything, mock.Anything, mock.AnythingOfType("database.PostCompletionHook")).Return(fmt.Errorf("optimization miss"))
	mdi.On("UpsertMessage", mock.Anything, mock.Anything, database.UpsertOptimizationExisting, mock.AnythingOfType("database.PostCompletionHook")).Return(nil).Run(func(args mock.Arguments) {
		args[3].(database.PostCompletionHook)()
	})

	mdm := em.data.(*datamocks.Manager)
	msgData := &messageAndData{
		message: batch.Payload.Messages[0],
		data:    batch.Payload.Data,
	}
	mdm.On("UpdateMessageCache", msgData.message, msgData.data).Return()

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalNode", mock.Anything).Return(testNode, nil)

	ok, err := em.persistBatchContent(em.ctx, batch, []*messageAndData{msgData})
	assert.NoError(t, err)
	assert.True(t, ok)

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestPersistBatchContentSentByUsFoundError(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	batch.Node = testNodeID

	mdm := em.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", em.ctx, batch.Payload.Messages[0].Header.ID).Return(nil, nil, false, fmt.Errorf("pop"))

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalNode", mock.Anything).Return(testNode, nil)

	ok, err := em.persistBatchContent(em.ctx, batch, []*messageAndData{})
	assert.Regexp(t, "pop", err)
	assert.False(t, ok)

	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)

}

func TestPersistBatchContentDataHashMismatch(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("InsertDataArray", mock.Anything, mock.Anything).Return(fmt.Errorf("optimization miss"))
	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationExisting).Return(database.HashMismatch)

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalNode", mock.Anything).Return(testNode, nil)

	ok, err := em.persistBatchContent(em.ctx, batch, []*messageAndData{})
	assert.NoError(t, err)
	assert.False(t, ok)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestPersistBatchContentDataMissingBlobRef(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	blob := &core.Blob{
		Hash: fftypes.NewRandB32(),
	}
	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`), Blob: &core.BlobRef{
		Hash: blob.Hash,
	}}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data}, blob)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", mock.Anything, mock.Anything).Return(nil, nil)

	ok, err := em.validateAndPersistBatchContent(em.ctx, batch)
	assert.NoError(t, err)
	assert.False(t, ok)

	mdi.AssertExpectations(t)

}
