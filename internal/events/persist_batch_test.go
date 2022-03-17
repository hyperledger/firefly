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
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
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
	data := &fftypes.Data{
		ID:        fftypes.NewUUID(),
		Value:     fftypes.JSONAnyPtrBytes(orgBytes),
		Validator: fftypes.MessageTypeDefinition,
	}

	batch := &fftypes.Batch{
		BatchHeader: fftypes.BatchHeader{
			ID: fftypes.NewUUID(),
			SignerRef: fftypes.SignerRef{
				Author: "did:firefly:org/12345",
				Key:    "0x12345",
			},
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID:   fftypes.NewUUID(),
				Type: fftypes.TransactionTypeBatchPin,
			},
			Messages: []*fftypes.Message{
				{
					Header: fftypes.MessageHeader{
						ID:   fftypes.NewUUID(),
						Type: fftypes.MessageTypeDefinition,
						SignerRef: fftypes.SignerRef{
							Author: "did:firefly:org/12345",
							Key:    "0x12345",
						},
					},
					Data: fftypes.DataRefs{
						{
							ID:   data.ID,
							Hash: data.Hash,
						},
					},
				},
			},
			Data: fftypes.DataArray{
				data,
			},
		},
	}
	bp, _ := batch.Confirmed()
	batch.Hash = fftypes.HashString(bp.Manifest.String())

	_, _, err = em.persistBatch(em.ctx, batch)
	assert.EqualError(t, err, "pop") // Confirms we got to upserting the batch

}

func TestPersistBatchNoCacheDataNotInBatch(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil)
	mdi.On("UpsertMessage", em.ctx, mock.Anything, database.UpsertOptimizationSkip).Return(nil)

	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, fftypes.BatchTypeBroadcast, fftypes.TransactionTypeBatchPin, fftypes.DataArray{data})
	data.ID = fftypes.NewUUID()
	_ = data.Seal(em.ctx, nil)
	bp, _ := batch.Confirmed()
	batch.Hash = fftypes.HashString(bp.Manifest.String())

	_, valid, err := em.persistBatch(em.ctx, batch)
	assert.False(t, valid)
	assert.NoError(t, err)

}

func TestPersistBatchExtraDataInBatch(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil)
	mdi.On("UpsertMessage", em.ctx, mock.Anything, database.UpsertOptimizationSkip).Return(nil)

	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, fftypes.BatchTypeBroadcast, fftypes.TransactionTypeBatchPin, fftypes.DataArray{data})
	data2 := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test2"`)}
	_ = data2.Seal(em.ctx, nil)
	batch.Payload.Data = append(batch.Payload.Data, data2)
	bp, _ := batch.Confirmed()
	batch.Hash = fftypes.HashString(bp.Manifest.String())

	_, valid, err := em.persistBatch(em.ctx, batch)
	assert.False(t, valid)
	assert.NoError(t, err)

}

func TestPersistBatchNilMessageEntryop(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	valid := em.validateBatchMessage(em.ctx, &fftypes.Batch{}, 0, nil)
	assert.False(t, valid)

}

func TestPersistBatchContentSendByUsOK(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, fftypes.BatchTypeBroadcast, fftypes.TransactionTypeBatchPin, fftypes.DataArray{data})
	batch.Node = testNodeID

	mdm := em.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", em.ctx, batch.Payload.Messages[0].Header.ID).Return(batch.Payload.Messages[0], batch.Payload.Data, true, nil)

	ok, err := em.persistBatchContent(em.ctx, batch, []*messageAndData{})
	assert.NoError(t, err)
	assert.True(t, ok)

	mdm.AssertExpectations(t)
}

func TestPersistBatchContentSentByNil(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, fftypes.BatchTypeBroadcast, fftypes.TransactionTypeBatchPin, fftypes.DataArray{data})
	batch.Node = nil

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("InsertDataArray", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertMessages", mock.Anything, mock.Anything).Return(nil)

	ok, err := em.persistBatchContent(em.ctx, batch, []*messageAndData{})
	assert.NoError(t, err)
	assert.True(t, ok)

	mdi.AssertExpectations(t)

}

func TestPersistBatchContentSentByUsNotFoundFallback(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, fftypes.BatchTypeBroadcast, fftypes.TransactionTypeBatchPin, fftypes.DataArray{data})
	batch.Node = testNodeID

	mdm := em.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", em.ctx, batch.Payload.Messages[0].Header.ID).Return(nil, nil, false, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("InsertDataArray", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertMessages", mock.Anything, mock.Anything).Return(nil)

	ok, err := em.persistBatchContent(em.ctx, batch, []*messageAndData{})
	assert.NoError(t, err)
	assert.True(t, ok)

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestPersistBatchContentSentByUsFoundMismatch(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, fftypes.BatchTypeBroadcast, fftypes.TransactionTypeBatchPin, fftypes.DataArray{data})
	batch.Node = testNodeID

	mdm := em.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", em.ctx, batch.Payload.Messages[0].Header.ID).Return(&fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}, nil, true, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("InsertDataArray", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertMessages", mock.Anything, mock.Anything).Return(fmt.Errorf("optimization miss"))
	mdi.On("UpsertMessage", mock.Anything, mock.Anything, database.UpsertOptimizationExisting).Return(database.HashMismatch)

	ok, err := em.persistBatchContent(em.ctx, batch, []*messageAndData{})
	assert.NoError(t, err)
	assert.False(t, ok)

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestPersistBatchContentSentByUsFoundError(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, fftypes.BatchTypeBroadcast, fftypes.TransactionTypeBatchPin, fftypes.DataArray{data})
	batch.Node = testNodeID

	mdm := em.data.(*datamocks.Manager)
	mdm.On("GetMessageWithDataCached", em.ctx, batch.Payload.Messages[0].Header.ID).Return(nil, nil, false, fmt.Errorf("pop"))

	ok, err := em.persistBatchContent(em.ctx, batch, []*messageAndData{})
	assert.Regexp(t, "pop", err)
	assert.False(t, ok)

	mdm.AssertExpectations(t)

}

func TestPersistBatchContentDataHashMismatch(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, fftypes.BatchTypeBroadcast, fftypes.TransactionTypeBatchPin, fftypes.DataArray{data})

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("InsertDataArray", mock.Anything, mock.Anything).Return(fmt.Errorf("optimization miss"))
	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationExisting).Return(database.HashMismatch)

	ok, err := em.persistBatchContent(em.ctx, batch, []*messageAndData{})
	assert.NoError(t, err)
	assert.False(t, ok)

	mdi.AssertExpectations(t)

}
