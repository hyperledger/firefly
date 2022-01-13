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

package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/publicstoragemocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBatchPinCompleteOkBroadcast(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &blockchain.BatchPin{
		Namespace:       "ns1",
		TransactionID:   fftypes.NewUUID(),
		BatchID:         fftypes.NewUUID(),
		BatchPayloadRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts:        []*fftypes.Bytes32{fftypes.NewRandB32()},
		Event: blockchain.Event{
			Name:       "BatchPin",
			ProtocolID: "tx1",
		},
	}
	batchData := &fftypes.Batch{
		ID:        batch.BatchID,
		Namespace: "ns1",
		Identity: fftypes.Identity{
			Author: "author1",
			Key:    "0x12345",
		},
		PayloadRef: batch.BatchPayloadRef,
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   batch.TransactionID,
			},
			Messages: []*fftypes.Message{},
			Data:     []*fftypes.Data{},
		},
	}
	batchData.Hash = batchData.Payload.Hash()
	batch.BatchHash = batchData.Hash
	batchDataBytes, err := json.Marshal(&batchData)
	assert.NoError(t, err)
	batchReadCloser := ioutil.NopCloser(bytes.NewReader(batchDataBytes))

	mpi := em.publicstorage.(*publicstoragemocks.Plugin)
	mpi.On("RetrieveData", mock.Anything, mock.
		MatchedBy(func(pr string) bool { return pr == batch.BatchPayloadRef })).
		Return(batchReadCloser, nil)

	mdi := em.database.(*databasemocks.Plugin)
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Return(nil)
	rag.RunFn = func(a mock.Arguments) {
		// Call through to persistBatch - the hash of our batch will be invalid,
		// which is swallowed without error as we cannot retry (it is logged of course)
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(ctx context.Context) error)(a[0].(context.Context)),
		}
	}

	mdi.On("InsertBlockchainEvent", mock.Anything, mock.MatchedBy(func(e *fftypes.BlockchainEvent) bool {
		return e.Name == batch.Event.Name
	})).Return(fmt.Errorf("pop")).Once()
	mdi.On("InsertBlockchainEvent", mock.Anything, mock.MatchedBy(func(e *fftypes.BlockchainEvent) bool {
		return e.Name == batch.Event.Name
	})).Return(nil).Times(2)
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeBlockchainEvent
	})).Return(nil).Times(2)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Once()
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything).Return(nil).Once()
	mdi.On("UpsertPin", mock.Anything, mock.Anything).Return(nil).Once()
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, false).Return(nil).Once()
	mbi := &blockchainmocks.Plugin{}

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveSigningKeyIdentity", mock.Anything, "0x12345").Return("author1", nil)

	err = em.BatchPinComplete(mbi, batch, "0x12345")
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestBatchPinCompleteOkPrivate(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &blockchain.BatchPin{
		Namespace:     "ns1",
		TransactionID: fftypes.NewUUID(),
		BatchID:       fftypes.NewUUID(),
		Contexts:      []*fftypes.Bytes32{fftypes.NewRandB32()},
	}
	batchData := &fftypes.Batch{
		ID:         batch.BatchID,
		Namespace:  "ns1",
		PayloadRef: batch.BatchPayloadRef,
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   batch.TransactionID,
			},
			Messages: []*fftypes.Message{},
			Data:     []*fftypes.Data{},
		},
	}
	batchDataBytes, err := json.Marshal(&batchData)
	assert.NoError(t, err)
	batchReadCloser := ioutil.NopCloser(bytes.NewReader(batchDataBytes))

	mpi := em.publicstorage.(*publicstoragemocks.Plugin)
	mpi.On("RetrieveData", mock.Anything, mock.
		MatchedBy(func(pr string) bool { return pr == batch.BatchPayloadRef })).
		Return(batchReadCloser, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("RunAsGroup", mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertPin", mock.Anything, mock.Anything).Return(nil)
	mbi := &blockchainmocks.Plugin{}

	err = em.BatchPinComplete(mbi, batch, "0x12345")
	assert.NoError(t, err)

	// Call through to persistBatch - the hash of our batch will be invalid,
	// which is swallowed without error as we cannot retry (it is logged of course)
	fn := mdi.Calls[0].Arguments[1].(func(ctx context.Context) error)
	err = fn(context.Background())
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestSequencedBroadcastRetrieveIPFSFail(t *testing.T) {
	em, cancel := newTestEventManager(t)

	batch := &blockchain.BatchPin{
		Namespace:       "ns",
		TransactionID:   fftypes.NewUUID(),
		BatchID:         fftypes.NewUUID(),
		BatchPayloadRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts:        []*fftypes.Bytes32{fftypes.NewRandB32()},
	}

	cancel() // to avoid retry
	mpi := em.publicstorage.(*publicstoragemocks.Plugin)
	mpi.On("RetrieveData", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	mbi := &blockchainmocks.Plugin{}

	err := em.BatchPinComplete(mbi, batch, "0x12345")
	mpi.AssertExpectations(t)
	assert.Regexp(t, "FF10158", err)
}

func TestBatchPinCompleteBadData(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &blockchain.BatchPin{
		Namespace:       "ns",
		TransactionID:   fftypes.NewUUID(),
		BatchID:         fftypes.NewUUID(),
		BatchPayloadRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts:        []*fftypes.Bytes32{fftypes.NewRandB32()},
	}
	batchReadCloser := ioutil.NopCloser(bytes.NewReader([]byte(`!json`)))

	mpi := em.publicstorage.(*publicstoragemocks.Plugin)
	mpi.On("RetrieveData", mock.Anything, mock.Anything).Return(batchReadCloser, nil)
	mbi := &blockchainmocks.Plugin{}

	err := em.BatchPinComplete(mbi, batch, "0x12345")
	assert.NoError(t, err) // We do not return a blocking error in the case of bad data stored in IPFS
}

func TestBatchPinCompleteNoTX(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &blockchain.BatchPin{}
	mbi := &blockchainmocks.Plugin{}

	err := em.BatchPinComplete(mbi, batch, "0x12345")
	assert.NoError(t, err)
}

func TestBatchPinCompleteBadNamespace(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &blockchain.BatchPin{
		Namespace:     "!bad",
		TransactionID: fftypes.NewUUID(),
	}
	mbi := &blockchainmocks.Plugin{}

	err := em.BatchPinComplete(mbi, batch, "0x12345")
	assert.NoError(t, err)
}

func TestPersistBatchMissingID(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	valid, err := em.persistBatch(context.Background(), &fftypes.Batch{})
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestPersistBatchAuthorResolveFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batchHash := fftypes.NewRandB32()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Identity: fftypes.Identity{
			Author: "author1",
			Key:    "0x12345",
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
		Hash: batchHash,
	}
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveSigningKeyIdentity", mock.Anything, mock.Anything).Return("", fmt.Errorf("pop"))
	batch.Hash = batch.Payload.Hash()
	valid, err := em.persistBatchFromBroadcast(context.Background(), batch, batchHash, "0x12345")
	assert.NoError(t, err) // retryable
	assert.False(t, valid)
}

func TestPersistBatchBadAuthor(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batchHash := fftypes.NewRandB32()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Identity: fftypes.Identity{
			Author: "author1",
			Key:    "0x12345",
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
		Hash: batchHash,
	}
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveSigningKeyIdentity", mock.Anything, mock.Anything).Return("author2", nil)
	batch.Hash = batch.Payload.Hash()
	valid, err := em.persistBatchFromBroadcast(context.Background(), batch, batchHash, "0x12345")
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestPersistBatchMismatchChainHash(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Identity: fftypes.Identity{
			Author: "author1",
			Key:    "0x12345",
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
		Hash: fftypes.NewRandB32(),
	}
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveSigningKeyIdentity", mock.Anything, mock.Anything).Return("author1", nil)
	batch.Hash = batch.Payload.Hash()
	valid, err := em.persistBatchFromBroadcast(context.Background(), batch, fftypes.NewRandB32(), "0x12345")
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestPersistBatchUpsertBatchMismatchHash(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Identity: fftypes.Identity{
			Author: "author1",
			Key:    "0x12345",
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
	}
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, false).Return(database.HashMismatch)

	valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistBatchBadHash(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Identity: fftypes.Identity{
			Author: "author1",
			Key:    "0x12345",
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
	}
	batch.Hash = fftypes.NewRandB32()

	valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestPersistBatchUpsertBatchFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Identity: fftypes.Identity{
			Author: "author1",
			Key:    "0x12345",
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
	}
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, false).Return(fmt.Errorf("pop"))

	valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatchSwallowBadData(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Identity: fftypes.Identity{
			Author: "author1",
			Key:    "0x12345",
		},
		Namespace: "ns1",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
			Messages: []*fftypes.Message{nil},
			Data:     []*fftypes.Data{nil},
		},
	}
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, false).Return(nil)

	valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistBatchGoodDataUpsertOptimizeExistingFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID:   fftypes.NewUUID(),
		Node: testNodeID,
		Identity: fftypes.Identity{
			Author: "author1",
			Key:    "0x12345",
		},
		Namespace: "ns1",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
			Data: []*fftypes.Data{
				{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)},
			},
		},
	}
	batch.Payload.Data[0].Hash = batch.Payload.Data[0].Value.Hash()
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationExisting).Return(fmt.Errorf("pop"))

	valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatchGoodDataUpsertOptimizeNewFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID:   fftypes.NewUUID(),
		Node: fftypes.NewUUID(),
		Identity: fftypes.Identity{
			Author: "author1",
			Key:    "0x12345",
		},
		Namespace: "ns1",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
			Data: []*fftypes.Data{
				{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)},
			},
		},
	}
	batch.Payload.Data[0].Hash = batch.Payload.Data[0].Value.Hash()
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationNew).Return(fmt.Errorf("pop"))

	valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatchGoodDataMessageFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Identity: fftypes.Identity{
			Author: "author1",
			Key:    "0x12345",
		},
		Namespace: "ns1",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{
					ID: fftypes.NewUUID(),
					Identity: fftypes.Identity{
						Author: "author1",
						Key:    "0x12345",
					},
				}},
			},
		},
	}
	batch.Payload.Messages[0].Header.DataHash = batch.Payload.Messages[0].Data.Hash()
	batch.Payload.Messages[0].Hash = batch.Payload.Messages[0].Header.Hash()
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("UpsertMessage", mock.Anything, mock.Anything, database.UpsertOptimizationSkip).Return(fmt.Errorf("pop"))

	valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatchGoodMessageAuthorMismatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Identity: fftypes.Identity{
			Author: "author1",
			Key:    "0x12345",
		},
		Namespace: "ns1",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{
					ID: fftypes.NewUUID(),
					Identity: fftypes.Identity{
						Author: "author1",
						Key:    "0x9999999",
					},
				}},
			},
		},
	}
	batch.Payload.Messages[0].Header.DataHash = batch.Payload.Messages[0].Data.Hash()
	batch.Payload.Messages[0].Hash = batch.Payload.Messages[0].Header.Hash()
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, false).Return(nil)

	valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestPersistBatchDataNilData(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
	}
	data := &fftypes.Data{
		ID: fftypes.NewUUID(),
	}
	err := em.persistBatchData(context.Background(), batch, 0, data, database.UpsertOptimizationSkip)
	assert.NoError(t, err)
}

func TestPersistBatchDataBadHash(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
	}
	data := &fftypes.Data{
		ID:    fftypes.NewUUID(),
		Value: fftypes.JSONAnyPtr(`"test"`),
		Hash:  fftypes.NewRandB32(),
	}
	err := em.persistBatchData(context.Background(), batch, 0, data, database.UpsertOptimizationSkip)
	assert.NoError(t, err)
}

func TestPersistBatchDataUpsertHashMismatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
	}

	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	data.Hash = data.Value.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationSkip).Return(database.HashMismatch)

	err := em.persistBatchData(context.Background(), batch, 0, data, database.UpsertOptimizationSkip)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistBatchDataUpsertDataError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
	}

	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	data.Hash = data.Value.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationSkip).Return(fmt.Errorf("pop"))

	err := em.persistBatchData(context.Background(), batch, 0, data, database.UpsertOptimizationSkip)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatchDataOk(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
	}

	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	data.Hash = data.Value.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationSkip).Return(nil)

	err := em.persistBatchData(context.Background(), batch, 0, data, database.UpsertOptimizationSkip)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistBatchMessageNilData(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	valid, err := em.persistBatchMessage(context.Background(), batch, 0, msg, database.UpsertOptimizationSkip)
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestPersistBatchMessageUpsertHashMismatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	msg.Header.DataHash = msg.Data.Hash()
	msg.Hash = msg.Header.Hash()
	assert.NoError(t, msg.Verify(context.Background()))

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertMessage", mock.Anything, mock.Anything, database.UpsertOptimizationSkip).Return(database.HashMismatch)

	valid, err := em.persistBatchMessage(context.Background(), batch, 0, msg, database.UpsertOptimizationSkip)
	assert.False(t, valid)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistBatchMessageUpsertMessageFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	msg.Header.DataHash = msg.Data.Hash()
	msg.Hash = msg.Header.Hash()
	assert.NoError(t, msg.Verify(context.Background()))

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertMessage", mock.Anything, mock.Anything, database.UpsertOptimizationSkip).Return(fmt.Errorf("pop"))

	valid, err := em.persistBatchMessage(context.Background(), batch, 0, msg, database.UpsertOptimizationSkip)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatchMessageOK(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	msg.Header.DataHash = msg.Data.Hash()
	msg.Hash = msg.Header.Hash()
	assert.NoError(t, msg.Verify(context.Background()))

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertMessage", mock.Anything, mock.Anything, database.UpsertOptimizationSkip).Return(nil)

	valid, err := em.persistBatchMessage(context.Background(), batch, 0, msg, database.UpsertOptimizationSkip)
	assert.True(t, valid)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistContextsFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertPin", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := em.persistContexts(em.ctx, &blockchain.BatchPin{
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
		},
	}, false)
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}
