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

	"github.com/kaleido-io/firefly/mocks/blockchainmocks"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/mocks/publicstoragemocks"
	"github.com/kaleido-io/firefly/pkg/blockchain"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSequencedBroadcastBatchOk(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &blockchain.BroadcastBatch{
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchPaylodRef: fftypes.NewRandB32(),
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
		MatchedBy(func(pr *fftypes.Bytes32) bool { return *pr == *batch.BatchPaylodRef })).
		Return(batchReadCloser, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("RunAsGroup", mock.Anything, mock.Anything).Return(nil)
	mbi := &blockchainmocks.Plugin{}

	err = em.SequencedBroadcastBatch(mbi, batch, "0x12345", "tx1", nil)
	assert.NoError(t, err)

	// Call through to persistBatch - the hash of our batch will be invalid,
	// which is swallowed without error as we cannot retry (it is logged of course)
	fn := mdi.Calls[0].Arguments[1].(func(ctx context.Context) error)
	err = fn(context.Background())
	assert.NoError(t, err)
}

func TestSequencedBroadcastRetrieveIPFSFail(t *testing.T) {
	em, cancel := newTestEventManager(t)

	batch := &blockchain.BroadcastBatch{
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchPaylodRef: fftypes.NewRandB32(),
	}

	cancel() // to avoid retry
	mpi := em.publicstorage.(*publicstoragemocks.Plugin)
	mpi.On("RetrieveData", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	mbi := &blockchainmocks.Plugin{}

	err := em.SequencedBroadcastBatch(mbi, batch, "0x12345", "tx1", nil)
	mpi.AssertExpectations(t)
	assert.Regexp(t, "FF10158", err)
}

func TestSequencedBroadcastBatchBadData(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &blockchain.BroadcastBatch{
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchPaylodRef: fftypes.NewRandB32(),
	}
	batchReadCloser := ioutil.NopCloser(bytes.NewReader([]byte(`!json`)))

	mpi := em.publicstorage.(*publicstoragemocks.Plugin)
	mpi.On("RetrieveData", mock.Anything, mock.Anything).Return(batchReadCloser, nil)
	mbi := &blockchainmocks.Plugin{}

	err := em.SequencedBroadcastBatch(mbi, batch, "0x12345", "tx1", nil)
	assert.NoError(t, err) // We do not return a blocking error in the case of bad data stored in IPFS
}

func TestPersistBatchMissingID(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	err := em.persistBatch(context.Background(), &fftypes.Batch{}, "0x12345", "tx1", nil)
	assert.NoError(t, err)
}

func TestPersistBatchBadAuthor(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &fftypes.Batch{
		ID:     fftypes.NewUUID(),
		Author: "0x23456",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
	}
	batch.Hash = batch.Payload.Hash()
	err := em.persistBatch(context.Background(), batch, "0x12345", "tx1", nil)
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

	err := em.persistBatch(context.Background(), batch, "0x12345", "tx1", nil)
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

	err := em.persistBatch(context.Background(), batch, "0x12345", "tx1", nil)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatchGetTransactionFail(t *testing.T) {
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
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, true, false).Return(nil)
	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := em.persistBatch(context.Background(), batch, "0x12345", "tx1", nil)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatchGetTransactionInvalidMatch(t *testing.T) {
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
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, true, false).Return(nil)
	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(&fftypes.Transaction{
		ID: fftypes.NewUUID(), // wrong
	}, nil)

	err := em.persistBatch(context.Background(), batch, "0x12345", "tx1", nil)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistBatcNewTXUpsertFail(t *testing.T) {
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
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, true, false).Return(nil)
	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(nil, nil)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))

	err := em.persistBatch(context.Background(), batch, "0x12345", "tx1", nil)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatcExistingTXHashMismatch(t *testing.T) {
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
		},
	}
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, true, false).Return(nil)
	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(&fftypes.Transaction{
		Subject: fftypes.TransactionSubject{
			Type:      fftypes.TransactionTypeBatchPin,
			Namespace: "ns1",
			Author:    "0x12345",
			Reference: batch.ID,
		},
	}, nil)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything, true, false).Return(database.HashMismatch)

	err := em.persistBatch(context.Background(), batch, "0x12345", "tx1", nil)
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
	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(&fftypes.Transaction{
		Subject: fftypes.TransactionSubject{
			Type:      fftypes.TransactionTypeBatchPin,
			Namespace: "ns1",
			Author:    "0x12345",
			Reference: batch.ID,
		},
	}, nil)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything, true, false).Return(nil)

	err := em.persistBatch(context.Background(), batch, "0x12345", "tx1", nil)
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
	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(&fftypes.Transaction{
		Subject: fftypes.TransactionSubject{
			Type:      fftypes.TransactionTypeBatchPin,
			Namespace: "ns1",
			Author:    "0x12345",
			Reference: batch.ID,
		},
	}, nil)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything, true, false).Return(nil)
	mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))

	err := em.persistBatch(context.Background(), batch, "0x12345", "tx1", nil)
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
				{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}},
			},
		},
	}
	batch.Payload.Messages[0].Header.DataHash = batch.Payload.Messages[0].Data.Hash()
	batch.Payload.Messages[0].Hash = batch.Payload.Messages[0].Header.Hash()
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, true, false).Return(nil)
	mdi.On("GetTransactionByID", mock.Anything, mock.Anything).Return(&fftypes.Transaction{
		Subject: fftypes.TransactionSubject{
			Type:      fftypes.TransactionTypeBatchPin,
			Namespace: "ns1",
			Author:    "0x12345",
			Reference: batch.ID,
		},
	}, nil)
	mdi.On("UpsertTransaction", mock.Anything, mock.Anything, true, false).Return(nil)
	mdi.On("UpsertMessage", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))

	err := em.persistBatch(context.Background(), batch, "0x12345", "tx1", nil)
	assert.EqualError(t, err, "pop")
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

func TestPersistBatchDataUpsertEventError(t *testing.T) {
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
	mdi.On("UpsertEvent", mock.Anything, mock.Anything, false).Return(fmt.Errorf("pop"))

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
	mdi.On("UpsertEvent", mock.Anything, mock.Anything, false).Return(nil)

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

func TestPersistBatchMessageUpsertEventFail(t *testing.T) {
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
	mdi.On("UpsertEvent", mock.Anything, mock.Anything, false).Return(fmt.Errorf("pop"))

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
	mdi.On("UpsertEvent", mock.Anything, mock.Anything, false).Return(nil)

	err := em.persistBatchMessage(context.Background(), batch, 0, msg)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}
