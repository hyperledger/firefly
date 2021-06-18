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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/hyperledger-labs/firefly/mocks/blockchainmocks"
	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/mocks/identitymocks"
	"github.com/hyperledger-labs/firefly/mocks/publicstoragemocks"
	"github.com/hyperledger-labs/firefly/pkg/blockchain"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBatchPinCompleteOkBroadcast(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &blockchain.BatchPin{
		Namespace:      "ns1",
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchPaylodRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts:       []*fftypes.Bytes32{fftypes.NewRandB32()},
	}
	batchData := &fftypes.Batch{
		ID:         batch.BatchID,
		Namespace:  "ns1",
		Author:     "0x12345",
		PayloadRef: batch.BatchPaylodRef,
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
		MatchedBy(func(pr string) bool { return pr == batch.BatchPaylodRef })).
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
	mdi.On("GetTransactionByID", mock.Anything, uuidMatches(batchData.Payload.TX.ID)).Return(nil, nil)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything, true, false).Return(nil)
	mdi.On("UpsertPin", mock.Anything, mock.Anything).Return(nil)
	mbi := &blockchainmocks.Plugin{}

	mii := em.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x12345").Return(&fftypes.Identity{OnChain: "0x12345"}, nil)

	err = em.BatchPinComplete(mbi, batch, "0x12345", "tx1", nil)
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
		PayloadRef: batch.BatchPaylodRef,
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
		MatchedBy(func(pr string) bool { return pr == batch.BatchPaylodRef })).
		Return(batchReadCloser, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("RunAsGroup", mock.Anything, mock.Anything).Return(nil)
	mdi.On("GetTransactionByID", mock.Anything, uuidMatches(batchData.Payload.TX.ID)).Return(nil, nil)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything, true, false).Return(nil)
	mdi.On("UpsertPin", mock.Anything, mock.Anything).Return(nil)
	mbi := &blockchainmocks.Plugin{}

	err = em.BatchPinComplete(mbi, batch, "0x12345", "tx1", nil)
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
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchPaylodRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts:       []*fftypes.Bytes32{fftypes.NewRandB32()},
	}

	cancel() // to avoid retry
	mpi := em.publicstorage.(*publicstoragemocks.Plugin)
	mpi.On("RetrieveData", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	mbi := &blockchainmocks.Plugin{}

	err := em.BatchPinComplete(mbi, batch, "0x12345", "tx1", nil)
	mpi.AssertExpectations(t)
	assert.Regexp(t, "FF10158", err)
}

func TestBatchPinCompleteBadData(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &blockchain.BatchPin{
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchPaylodRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts:       []*fftypes.Bytes32{fftypes.NewRandB32()},
	}
	batchReadCloser := ioutil.NopCloser(bytes.NewReader([]byte(`!json`)))

	mpi := em.publicstorage.(*publicstoragemocks.Plugin)
	mpi.On("RetrieveData", mock.Anything, mock.Anything).Return(batchReadCloser, nil)
	mbi := &blockchainmocks.Plugin{}

	err := em.BatchPinComplete(mbi, batch, "0x12345", "tx1", nil)
	assert.NoError(t, err) // We do not return a blocking error in the case of bad data stored in IPFS
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
		ID:     fftypes.NewUUID(),
		Author: "0x23456",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
		Hash: batchHash,
	}
	mii := em.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x23456").Return(nil, fmt.Errorf("pop"))
	batch.Hash = batch.Payload.Hash()
	err := em.persistBatchFromBroadcast(context.Background(), batch, batchHash, "0x12345")
	assert.NoError(t, err)
}

func TestPersistBatchBadAuthor(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batchHash := fftypes.NewRandB32()
	batch := &fftypes.Batch{
		ID:     fftypes.NewUUID(),
		Author: "0x23456",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
		Hash: batchHash,
	}
	mii := em.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x23456").Return(&fftypes.Identity{OnChain: "0x23456"}, nil)
	batch.Hash = batch.Payload.Hash()
	err := em.persistBatchFromBroadcast(context.Background(), batch, batchHash, "0x12345")
	assert.NoError(t, err)
}

func TestPersistBatchMismatchChainHash(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID:     fftypes.NewUUID(),
		Author: "0x12345",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
		Hash: fftypes.NewRandB32(),
	}
	mii := em.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x12345").Return(&fftypes.Identity{OnChain: "0x12345"}, nil)
	batch.Hash = batch.Payload.Hash()
	err := em.persistBatchFromBroadcast(context.Background(), batch, fftypes.NewRandB32(), "0x12345")
	assert.NoError(t, err)
}

func TestPersistBatchUpsertBatchMismatchHash(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID:     fftypes.NewUUID(),
		Author: "0x12345",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
	}
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, true, false).Return(database.HashMismatch)

	valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistBatchUpsertBatchFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID:     fftypes.NewUUID(),
		Author: "0x12345",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
	}
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))

	valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatchGetTransactionBadNamespace(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batchPin := &blockchain.BatchPin{
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchHash:      fftypes.NewRandB32(),
		BatchPaylodRef: "",
		Contexts:       []*fftypes.Bytes32{fftypes.NewRandB32()},
	}

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(nil, nil)

	err := em.persistBatchTransaction(context.Background(), batchPin, "0x12345", "txid1", fftypes.JSONObject{})
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistBatchGetTransactionFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batchPin := &blockchain.BatchPin{
		Namespace:      "ns1",
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchHash:      fftypes.NewRandB32(),
		BatchPaylodRef: "",
		Contexts:       []*fftypes.Bytes32{fftypes.NewRandB32()},
	}

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := em.persistBatchTransaction(context.Background(), batchPin, "0x12345", "txid1", fftypes.JSONObject{})
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestPersistBatchGetTransactionInvalidMatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batchPin := &blockchain.BatchPin{
		Namespace:      "ns1",
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchHash:      fftypes.NewRandB32(),
		BatchPaylodRef: "",
		Contexts:       []*fftypes.Bytes32{fftypes.NewRandB32()},
	}

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(&fftypes.Transaction{
		ID: fftypes.NewUUID(), // wrong
	}, nil)

	err := em.persistBatchTransaction(context.Background(), batchPin, "0x12345", "txid1", fftypes.JSONObject{})
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistBatcNewTXUpsertFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batchPin := &blockchain.BatchPin{
		Namespace:      "ns1",
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchHash:      fftypes.NewRandB32(),
		BatchPaylodRef: "",
		Contexts:       []*fftypes.Bytes32{fftypes.NewRandB32()},
	}

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(nil, nil)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))

	err := em.persistBatchTransaction(context.Background(), batchPin, "0x12345", "txid1", fftypes.JSONObject{})
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestPersistBatcExistingTXHashMismatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batchPin := &blockchain.BatchPin{
		Namespace:      "ns1",
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchHash:      fftypes.NewRandB32(),
		BatchPaylodRef: "",
		Contexts:       []*fftypes.Bytes32{fftypes.NewRandB32()},
	}

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(&fftypes.Transaction{
		Subject: fftypes.TransactionSubject{
			Type:      fftypes.TransactionTypeBatchPin,
			Namespace: "ns1",
			Signer:    "0x12345",
			Reference: batchPin.BatchID,
		},
	}, nil)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything, true, false).Return(database.HashMismatch)

	err := em.persistBatchTransaction(context.Background(), batchPin, "0x12345", "txid1", fftypes.JSONObject{})
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistBatchSwallowBadData(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID:        fftypes.NewUUID(),
		Author:    "0x12345",
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
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, true, false).Return(nil)

	valid, err := em.persistBatch(context.Background(), batch)
	assert.True(t, valid)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistBatchGoodDataUpsertFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID:        fftypes.NewUUID(),
		Author:    "0x12345",
		Namespace: "ns1",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
			Data: []*fftypes.Data{
				{ID: fftypes.NewUUID()},
			},
		},
	}
	batch.Payload.Data[0].Hash = batch.Payload.Data[0].Value.Hash()
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, true, false).Return(nil)
	mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))

	valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatchGoodDataMessageFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID:        fftypes.NewUUID(),
		Author:    "0x12345",
		Namespace: "ns1",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{
					ID:     fftypes.NewUUID(),
					Author: "0x12345",
				}},
			},
		},
	}
	batch.Payload.Messages[0].Header.DataHash = batch.Payload.Messages[0].Data.Hash()
	batch.Payload.Messages[0].Hash = batch.Payload.Messages[0].Header.Hash()
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, true, false).Return(nil)
	mdi.On("UpsertMessage", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))

	valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatchGoodMessageAuthorMismatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID:        fftypes.NewUUID(),
		Author:    "0x12345",
		Namespace: "ns1",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
			Messages: []*fftypes.Message{
				{Header: fftypes.MessageHeader{
					ID:     fftypes.NewUUID(),
					Author: "0x9999999",
				}},
			},
		},
	}
	batch.Payload.Messages[0].Header.DataHash = batch.Payload.Messages[0].Data.Hash()
	batch.Payload.Messages[0].Hash = batch.Payload.Messages[0].Header.Hash()
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, true, false).Return(nil)

	valid, err := em.persistBatch(context.Background(), batch)
	assert.True(t, valid)
	assert.NoError(t, err)
}

func TestPersistBatchDataBadHash(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
	}
	data := &fftypes.Data{
		ID: fftypes.NewUUID(),
	}
	err := em.persistBatchData(context.Background(), batch, 0, data)
	assert.NoError(t, err)
}

func TestPersistBatchDataUpsertHashMismatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
	}

	data := &fftypes.Data{
		ID: fftypes.NewUUID(),
	}
	data.Hash = data.Value.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(database.HashMismatch)

	err := em.persistBatchData(context.Background(), batch, 0, data)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistBatchDataUpsertDataError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
	}

	data := &fftypes.Data{
		ID: fftypes.NewUUID(),
	}
	data.Hash = data.Value.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))

	err := em.persistBatchData(context.Background(), batch, 0, data)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatchDataOk(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
	}

	data := &fftypes.Data{
		ID: fftypes.NewUUID(),
	}
	data.Hash = data.Value.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(nil)

	err := em.persistBatchData(context.Background(), batch, 0, data)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistBatchMessageBadHash(t *testing.T) {
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
	err := em.persistBatchMessage(context.Background(), batch, 0, msg)
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
	mdi.On("UpsertMessage", mock.Anything, mock.Anything, true, false).Return(database.HashMismatch)

	err := em.persistBatchMessage(context.Background(), batch, 0, msg)
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
	mdi.On("UpsertMessage", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))

	err := em.persistBatchMessage(context.Background(), batch, 0, msg)
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
	mdi.On("UpsertMessage", mock.Anything, mock.Anything, true, false).Return(nil)

	err := em.persistBatchMessage(context.Background(), batch, 0, msg)
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
