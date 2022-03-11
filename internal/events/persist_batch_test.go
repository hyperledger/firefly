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
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPersistBatchFromBroadcast(t *testing.T) {

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
	batch.Hash = fftypes.HashString(batch.Manifest().String())

	_, err = em.persistBatchFromBroadcast(em.ctx, batch, batch.Hash)
	assert.EqualError(t, err, "pop") // Confirms we got to upserting the batch

}

func TestPersistBatchFromBroadcastNoCacheDataNotInBatch(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil)
	mdi.On("UpsertMessage", em.ctx, mock.Anything, database.UpsertOptimizationSkip).Return(nil)

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
						TxType: fftypes.TransactionTypeBatchPin,
					},
					Data: fftypes.DataRefs{
						{
							ID:   fftypes.NewUUID(),
							Hash: fftypes.NewRandB32(),
						},
					},
				},
			},
			Data: nil,
		},
	}
	batch.Payload.Messages[0].Seal(em.ctx)
	batch.Hash = fftypes.HashString(batch.Manifest().String())

	valid, err := em.persistBatchFromBroadcast(em.ctx, batch, batch.Hash)
	assert.True(t, valid)
	assert.NoError(t, err)

}

func TestPersistBatchNilMessageEntryop(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil)

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
			Messages: []*fftypes.Message{nil},
			Data:     nil,
		},
	}
	batch.Hash = fftypes.HashString(batch.Manifest().String())

	valid, err := em.persistBatchFromBroadcast(em.ctx, batch, batch.Hash)
	assert.False(t, valid)
	assert.NoError(t, err)

}

func TestPersistBatchFromBroadcastBadHash(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(fmt.Errorf(("pop")))

	batch := &fftypes.Batch{}
	batch.Hash = fftypes.NewRandB32()

	ok, err := em.persistBatchFromBroadcast(em.ctx, batch, fftypes.NewRandB32())
	assert.NoError(t, err)
	assert.False(t, ok)

}
