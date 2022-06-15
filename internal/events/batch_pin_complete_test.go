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
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/shareddownloadmocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func sampleBatch(t *testing.T, batchType core.BatchType, txType core.TransactionType, data core.DataArray, blobs ...*core.Blob) *core.Batch {
	identity := core.SignerRef{Author: "signingOrg", Key: "0x12345"}
	msgType := core.MessageTypeBroadcast
	if batchType == core.BatchTypePrivate {
		msgType = core.MessageTypePrivate
	}
	for i, d := range data {
		var blob *core.Blob
		d.Namespace = "ns1"
		if len(blobs) > i {
			blob = blobs[i]
		}
		err := d.Seal(context.Background(), blob)
		assert.NoError(t, err)
	}
	msg := &core.Message{
		Header: core.MessageHeader{
			Namespace: "ns1",
			SignerRef: identity,
			ID:        fftypes.NewUUID(),
			Type:      msgType,
			TxType:    txType,
			Topics:    core.FFStringArray{"topic1"},
		},
		Data: data.Refs(),
	}
	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			Namespace: "ns1",
			SignerRef: identity,
			Type:      batchType,
			ID:        fftypes.NewUUID(),
			Node:      fftypes.NewUUID(),
		},
		Payload: core.BatchPayload{
			TX: core.TransactionRef{
				ID:   fftypes.NewUUID(),
				Type: txType,
			},
			Messages: []*core.Message{msg},
			Data:     data,
		},
	}
	err := msg.Seal(context.Background())
	assert.NoError(t, err)
	bp, _ := batch.Confirmed()
	batch.Hash = fftypes.HashString(bp.Manifest.String())
	return batch
}

func TestBatchPinCompleteOkBroadcast(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	batchPin := &blockchain.BatchPin{
		Namespace:       "ns1",
		TransactionID:   batch.Payload.TX.ID,
		BatchID:         batch.ID,
		BatchPayloadRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts:        []*fftypes.Bytes32{fftypes.NewRandB32()},
		Event: blockchain.Event{
			Name:           "BatchPin",
			BlockchainTXID: "0x12345",
			ProtocolID:     "10/20/30",
		},
	}

	batch.Hash = batch.Payload.Hash()
	batchPin.BatchHash = batch.Hash

	mth := em.txHelper.(*txcommonmocks.Helper)
	mth.On("PersistTransaction", mock.Anything, batchPin.TransactionID, core.TransactionTypeBatchPin, "0x12345").
		Return(false, fmt.Errorf("pop")).Once()
	mth.On("PersistTransaction", mock.Anything, batchPin.TransactionID, core.TransactionTypeBatchPin, "0x12345").
		Return(true, nil)

	mdi := em.database.(*databasemocks.Plugin)
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Return(nil)
	rag.RunFn = func(a mock.Arguments) {
		// Call through to persistBatch - the hash of our batch will be invalid,
		// which is swallowed without error as we cannot retry (it is logged of course)
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(ctx context.Context) error)(a[0].(context.Context)),
		}
	}

	mdi.On("GetBlockchainEventByProtocolID", mock.Anything, "ns1", (*fftypes.UUID)(nil), batchPin.Event.ProtocolID).Return(nil, nil)
	mth.On("InsertBlockchainEvent", mock.Anything, mock.MatchedBy(func(e *core.BlockchainEvent) bool {
		return e.Name == batchPin.Event.Name
	})).Return(fmt.Errorf("pop")).Once()
	mth.On("InsertBlockchainEvent", mock.Anything, mock.MatchedBy(func(e *core.BlockchainEvent) bool {
		return e.Name == batchPin.Event.Name
	})).Return(nil).Times(1)
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(e *core.Event) bool {
		return e.Type == core.EventTypeBlockchainEventReceived
	})).Return(nil).Times(1)
	mdi.On("InsertPins", mock.Anything, mock.Anything).Return(nil).Once()
	msd := em.sharedDownload.(*shareddownloadmocks.Manager)
	mdi.On("GetBatchByID", mock.Anything, mock.Anything).Return(nil, nil)
	msd.On("InitiateDownloadBatch", mock.Anything, "ns1", batchPin.TransactionID, batchPin.BatchPayloadRef).Return(nil)
	mbi := &blockchainmocks.Plugin{}

	err := em.BatchPinComplete(mbi, batchPin, &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x12345",
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestBatchPinCompleteOkBroadcastExistingBatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	batchPin := &blockchain.BatchPin{
		Namespace:       "ns1",
		TransactionID:   batch.Payload.TX.ID,
		BatchID:         batch.ID,
		BatchPayloadRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts:        []*fftypes.Bytes32{fftypes.NewRandB32()},
		Event: blockchain.Event{
			Name:           "BatchPin",
			BlockchainTXID: "0x12345",
			ProtocolID:     "10/20/30",
		},
	}
	payloadBinary, jsonErr := json.Marshal(&batch.Payload)
	assert.NoError(t, jsonErr)
	batchPersisted := &core.BatchPersisted{
		TX:          batch.Payload.TX,
		BatchHeader: batch.BatchHeader,
		Manifest:    fftypes.JSONAnyPtr(string(payloadBinary)),
		Hash:        batch.Payload.Hash(),
	}

	batch.Hash = batch.Payload.Hash()
	batchPin.BatchHash = batch.Hash

	mth := em.txHelper.(*txcommonmocks.Helper)
	mth.On("PersistTransaction", mock.Anything, batchPin.TransactionID, core.TransactionTypeBatchPin, "0x12345").
		Return(false, fmt.Errorf("pop")).Once()
	mth.On("PersistTransaction", mock.Anything, batchPin.TransactionID, core.TransactionTypeBatchPin, "0x12345").
		Return(true, nil)

	mdi := em.database.(*databasemocks.Plugin)
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Return(nil)
	rag.RunFn = func(a mock.Arguments) {
		// Call through to persistBatch - the hash of our batch will be invalid,
		// which is swallowed without error as we cannot retry (it is logged of course)
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(ctx context.Context) error)(a[0].(context.Context)),
		}
	}

	mdi.On("GetBlockchainEventByProtocolID", mock.Anything, "ns1", (*fftypes.UUID)(nil), batchPin.Event.ProtocolID).Return(nil, nil)
	mth.On("InsertBlockchainEvent", mock.Anything, mock.MatchedBy(func(e *core.BlockchainEvent) bool {
		return e.Name == batchPin.Event.Name
	})).Return(fmt.Errorf("pop")).Once()
	mth.On("InsertBlockchainEvent", mock.Anything, mock.MatchedBy(func(e *core.BlockchainEvent) bool {
		return e.Name == batchPin.Event.Name
	})).Return(nil).Times(1)
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(e *core.Event) bool {
		return e.Type == core.EventTypeBlockchainEventReceived
	})).Return(nil).Times(1)
	mdi.On("InsertPins", mock.Anything, mock.Anything).Return(nil).Once()
	mdi.On("GetBatchByID", mock.Anything, mock.Anything).Return(batchPersisted, nil)
	mbi := &blockchainmocks.Plugin{}

	err := em.BatchPinComplete(mbi, batchPin, &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x12345",
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestBatchPinCompleteOkPrivate(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batchPin := &blockchain.BatchPin{
		Namespace:     "ns1",
		TransactionID: fftypes.NewUUID(),
		BatchID:       fftypes.NewUUID(),
		Contexts:      []*fftypes.Bytes32{fftypes.NewRandB32()},
		Event: blockchain.Event{
			BlockchainTXID: "0x12345",
			ProtocolID:     "10/20/30",
		},
	}

	mth := em.txHelper.(*txcommonmocks.Helper)
	mth.On("PersistTransaction", mock.Anything, batchPin.TransactionID, core.TransactionTypeBatchPin, "0x12345").Return(true, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("RunAsGroup", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertPins", mock.Anything, mock.Anything).Return(fmt.Errorf("These pins have been seen before")) // simulate replay fallback
	mdi.On("UpsertPin", mock.Anything, mock.Anything).Return(nil)
	mdi.On("GetBlockchainEventByProtocolID", mock.Anything, "ns1", (*fftypes.UUID)(nil), batchPin.Event.ProtocolID).Return(nil, nil)
	mth.On("InsertBlockchainEvent", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	mdi.On("GetBatchByID", mock.Anything, mock.Anything).Return(nil, nil)

	mbi := &blockchainmocks.Plugin{}

	err := em.BatchPinComplete(mbi, batchPin, &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0xffffeeee",
	})
	assert.NoError(t, err)

	// Call through to persistBatch - the hash of our batch will be invalid,
	// which is swallowed without error as we cannot retry (it is logged of course)
	fn := mdi.Calls[1].Arguments[1].(func(ctx context.Context) error)
	err = fn(context.Background())
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestBatchPinCompleteInsertPinsFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel()

	batchPin := &blockchain.BatchPin{
		Namespace:     "ns1",
		TransactionID: fftypes.NewUUID(),
		BatchID:       fftypes.NewUUID(),
		Contexts:      []*fftypes.Bytes32{fftypes.NewRandB32()},
		Event: blockchain.Event{
			BlockchainTXID: "0x12345",
			ProtocolID:     "10/20/30",
		},
	}

	mth := em.txHelper.(*txcommonmocks.Helper)
	mth.On("PersistTransaction", mock.Anything, batchPin.TransactionID, core.TransactionTypeBatchPin, "0x12345").Return(true, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("RunAsGroup", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertPins", mock.Anything, mock.Anything).Return(fmt.Errorf("optimization miss"))
	mdi.On("UpsertPin", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mdi.On("GetBlockchainEventByProtocolID", mock.Anything, "ns1", (*fftypes.UUID)(nil), batchPin.Event.ProtocolID).Return(nil, nil)
	mth.On("InsertBlockchainEvent", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)

	mbi := &blockchainmocks.Plugin{}

	err := em.BatchPinComplete(mbi, batchPin, &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0xffffeeee",
	})
	assert.Regexp(t, "FF00154", err)

	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestBatchPinCompleteGetBatchByIDFails(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel()

	batchPin := &blockchain.BatchPin{
		Namespace:     "ns1",
		TransactionID: fftypes.NewUUID(),
		BatchID:       fftypes.NewUUID(),
		Contexts:      []*fftypes.Bytes32{fftypes.NewRandB32()},
		Event: blockchain.Event{
			BlockchainTXID: "0x12345",
			ProtocolID:     "10/20/30",
		},
	}

	mth := em.txHelper.(*txcommonmocks.Helper)
	mth.On("PersistTransaction", mock.Anything, batchPin.TransactionID, core.TransactionTypeBatchPin, "0x12345").Return(true, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("RunAsGroup", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertPins", mock.Anything, mock.Anything).Return(nil)
	mdi.On("GetBlockchainEventByProtocolID", mock.Anything, "ns1", (*fftypes.UUID)(nil), batchPin.Event.ProtocolID).Return(nil, nil)
	mth.On("InsertBlockchainEvent", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	mdi.On("GetBatchByID", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("batch lookup failed"))

	mbi := &blockchainmocks.Plugin{}

	err := em.BatchPinComplete(mbi, batchPin, &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0xffffeeee",
	})
	assert.Regexp(t, "FF00154", err)

	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestSequencedBroadcastInitiateDownloadFail(t *testing.T) {
	em, cancel := newTestEventManager(t)

	batchPin := &blockchain.BatchPin{
		Namespace:       "ns1",
		TransactionID:   fftypes.NewUUID(),
		BatchID:         fftypes.NewUUID(),
		BatchPayloadRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts:        []*fftypes.Bytes32{fftypes.NewRandB32()},
		Event: blockchain.Event{
			BlockchainTXID: "0x12345",
			ProtocolID:     "10/20/30",
		},
	}

	cancel() // to avoid retry
	mbi := &blockchainmocks.Plugin{}

	mth := em.txHelper.(*txcommonmocks.Helper)
	mth.On("PersistTransaction", mock.Anything, batchPin.TransactionID, core.TransactionTypeBatchPin, "0x12345").Return(true, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetBlockchainEventByProtocolID", mock.Anything, "ns1", (*fftypes.UUID)(nil), batchPin.Event.ProtocolID).Return(nil, nil)
	mth.On("InsertBlockchainEvent", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertPins", mock.Anything, mock.Anything).Return(nil)
	mdi.On("GetBatchByID", mock.Anything, mock.Anything).Return(nil, nil)
	msd := em.sharedDownload.(*shareddownloadmocks.Manager)
	msd.On("InitiateDownloadBatch", mock.Anything, "ns1", batchPin.TransactionID, batchPin.BatchPayloadRef).Return(fmt.Errorf("pop"))

	err := em.BatchPinComplete(mbi, batchPin, &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0xffffeeee",
	})
	mdi.AssertExpectations(t)
	msd.AssertExpectations(t)
	mth.AssertExpectations(t)
	assert.Regexp(t, "FF00154", err)
}

func TestBatchPinCompleteNoTX(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &blockchain.BatchPin{}
	mbi := &blockchainmocks.Plugin{}

	err := em.BatchPinComplete(mbi, batch, &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x12345",
	})
	assert.NoError(t, err)
}

func TestBatchPinCompleteBadNamespace(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &blockchain.BatchPin{
		Namespace:     "!bad",
		TransactionID: fftypes.NewUUID(),
		Event: blockchain.Event{
			BlockchainTXID: "0x12345",
		},
	}
	mbi := &blockchainmocks.Plugin{}

	err := em.BatchPinComplete(mbi, batch, &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x12345",
	})
	assert.NoError(t, err)
}

func TestBatchPinCompleteWrongNamespace(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &blockchain.BatchPin{
		Namespace:     "ns2",
		TransactionID: fftypes.NewUUID(),
		Event: blockchain.Event{
			BlockchainTXID: "0x12345",
		},
	}
	mbi := &blockchainmocks.Plugin{}

	err := em.BatchPinComplete(mbi, batch, &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x12345",
	})
	assert.NoError(t, err)
}

func TestPersistBatchMissingID(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch, valid, err := em.persistBatch(context.Background(), &core.Batch{})
	assert.False(t, valid)
	assert.Nil(t, batch)
	assert.NoError(t, err)
}

func TestPersistBatchAuthorResolveFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batchHash := fftypes.NewRandB32()
	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: fftypes.NewUUID(),
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
		Hash: batchHash,
		Payload: core.BatchPayload{
			TX: core.TransactionRef{
				Type: core.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
	}
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("NormalizeSigningKeyIdentity", mock.Anything, mock.Anything).Return("", fmt.Errorf("pop"))
	batch.Hash = batch.Payload.Hash()
	_, valid, err := em.persistBatch(context.Background(), batch)
	assert.NoError(t, err) // retryable
	assert.False(t, valid)
}

func TestPersistBatchBadAuthor(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batchHash := fftypes.NewRandB32()
	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: fftypes.NewUUID(),
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
		Hash: batchHash,
		Payload: core.BatchPayload{
			TX: core.TransactionRef{
				Type: core.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
	}
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("NormalizeSigningKeyIdentity", mock.Anything, mock.Anything).Return("author2", nil)
	batch.Hash = batch.Payload.Hash()
	_, valid, err := em.persistBatch(context.Background(), batch)
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestPersistBatchMismatchChainHash(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: fftypes.NewUUID(),
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
		Hash: fftypes.NewRandB32(),
		Payload: core.BatchPayload{
			TX: core.TransactionRef{
				Type: core.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
	}
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("NormalizeSigningKeyIdentity", mock.Anything, mock.Anything).Return("author1", nil)
	batch.Hash = batch.Payload.Hash()
	_, valid, err := em.persistBatch(context.Background(), batch)
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestPersistBatchUpsertBatchMismatchHash(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything).Return(database.HashMismatch)

	bp, valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.Nil(t, bp)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistBatchBadHash(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	batch.Hash = fftypes.NewRandB32()

	bp, valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.Nil(t, bp)
	assert.NoError(t, err)
}

func TestPersistBatchNoData(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: fftypes.NewUUID(),
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
		Payload: core.BatchPayload{
			TX: core.TransactionRef{
				Type: core.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
		},
	}
	batch.Hash = fftypes.NewRandB32()

	bp, valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.Nil(t, bp)
	assert.NoError(t, err)
}

func TestPersistBatchUpsertBatchFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	bp, valid, err := em.persistBatch(context.Background(), batch)
	assert.Nil(t, bp)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatchSwallowBadData(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: fftypes.NewUUID(),
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
			Namespace: "ns1",
		},
		Payload: core.BatchPayload{
			TX: core.TransactionRef{
				Type: core.TransactionTypeBatchPin,
				ID:   fftypes.NewUUID(),
			},
			Messages: []*core.Message{nil},
			Data:     core.DataArray{nil},
		},
	}
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything).Return(nil)

	bp, valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.NoError(t, err)
	assert.Nil(t, bp)
	mdi.AssertExpectations(t)
}

func TestPersistBatchGoodDataUpsertOptimizeFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertDataArray", mock.Anything, mock.Anything).Return(fmt.Errorf("optimzation miss"))
	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationExisting).Return(fmt.Errorf("pop"))

	bp, valid, err := em.persistBatch(context.Background(), batch)
	assert.Nil(t, bp)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatchGoodDataMessageFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertDataArray", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertMessages", mock.Anything, mock.Anything, mock.AnythingOfType("database.PostCompletionHook")).Return(fmt.Errorf("optimzation miss"))
	mdi.On("UpsertMessage", mock.Anything, mock.Anything, database.UpsertOptimizationExisting, mock.AnythingOfType("database.PostCompletionHook")).Return(fmt.Errorf("pop"))

	bp, valid, err := em.persistBatch(context.Background(), batch)
	assert.False(t, valid)
	assert.Nil(t, bp)
	assert.EqualError(t, err, "pop")
}

func TestPersistBatchGoodMessageAuthorMismatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	batch.Payload.Messages[0].Header.Key = "0x9999999"
	batch.Payload.Messages[0].Header.DataHash = batch.Payload.Messages[0].Data.Hash()
	batch.Payload.Messages[0].Hash = batch.Payload.Messages[0].Header.Hash()
	batch.Hash = batch.Payload.Hash()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything).Return(nil)

	bp, valid, err := em.persistBatch(context.Background(), batch)
	assert.Nil(t, bp)
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestPersistBatchDataNilData(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: fftypes.NewUUID(),
		},
	}
	data := &core.Data{
		ID: fftypes.NewUUID(),
	}
	valid := em.validateBatchData(context.Background(), batch, 0, data)
	assert.False(t, valid)
}

func TestPersistBatchDataBadHash(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	data := &core.Data{
		ID:    fftypes.NewUUID(),
		Value: fftypes.JSONAnyPtr(`"test"`),
	}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})
	batch.Payload.Data[0].Hash = fftypes.NewRandB32()
	valid := em.validateBatchData(context.Background(), batch, 0, data)
	assert.False(t, valid)
}

func TestPersistBatchDataOk(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data})

	valid := em.validateBatchData(context.Background(), batch, 0, data)
	assert.True(t, valid)
}

func TestPersistBatchDataWithPublicAlreaydDownloadedOk(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	blob := &core.Blob{
		Hash: fftypes.NewRandB32(),
		Size: 12345,
	}
	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`), Blob: &core.BlobRef{
		Hash:   blob.Hash,
		Size:   12345,
		Name:   "myfile.txt",
		Public: "ref1",
	}}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data}, blob)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", mock.Anything, blob.Hash).Return(blob, nil)

	valid, err := em.checkAndInitiateBlobDownloads(context.Background(), batch, 0, data)
	assert.Nil(t, err)
	assert.True(t, valid)
}

func TestPersistBatchDataWithPublicInitiateDownload(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	blob := &core.Blob{
		Hash: fftypes.NewRandB32(),
		Size: 12345,
	}
	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`), Blob: &core.BlobRef{
		Hash:   blob.Hash,
		Size:   12345,
		Name:   "myfile.txt",
		Public: "ref1",
	}}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data}, blob)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", mock.Anything, blob.Hash).Return(nil, nil)

	msd := em.sharedDownload.(*shareddownloadmocks.Manager)
	msd.On("InitiateDownloadBlob", mock.Anything, batch.Namespace, batch.Payload.TX.ID, data.ID, "ref1").Return(nil)

	valid, err := em.checkAndInitiateBlobDownloads(context.Background(), batch, 0, data)
	assert.Nil(t, err)
	assert.True(t, valid)
}

func TestPersistBatchDataWithPublicInitiateDownloadFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	blob := &core.Blob{
		Hash: fftypes.NewRandB32(),
		Size: 12345,
	}
	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`), Blob: &core.BlobRef{
		Hash:   blob.Hash,
		Size:   12345,
		Name:   "myfile.txt",
		Public: "ref1",
	}}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data}, blob)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", mock.Anything, blob.Hash).Return(nil, nil)

	msd := em.sharedDownload.(*shareddownloadmocks.Manager)
	msd.On("InitiateDownloadBlob", mock.Anything, batch.Namespace, batch.Payload.TX.ID, data.ID, "ref1").Return(fmt.Errorf("pop"))

	valid, err := em.checkAndInitiateBlobDownloads(context.Background(), batch, 0, data)
	assert.Regexp(t, "pop", err)
	assert.False(t, valid)
}

func TestPersistBatchDataWithBlobGetBlobFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	blob := &core.Blob{
		Hash: fftypes.NewRandB32(),
		Size: 12345,
	}
	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`), Blob: &core.BlobRef{
		Hash:   blob.Hash,
		Size:   12345,
		Name:   "myfile.txt",
		Public: "ref1",
	}}
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{data}, blob)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", mock.Anything, blob.Hash).Return(nil, fmt.Errorf("pop"))

	valid, err := em.checkAndInitiateBlobDownloads(context.Background(), batch, 0, data)
	assert.Regexp(t, "pop", err)
	assert.False(t, valid)
}

func TestPersistBatchMessageNilData(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			ID: fftypes.NewUUID(),
		},
	}
	msg := &core.Message{
		Header: core.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	valid := em.validateBatchMessage(context.Background(), batch, 0, msg)
	assert.False(t, valid)
}

func TestPersistBatchMessageOK(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	batch := sampleBatch(t, core.BatchTypeBroadcast, core.TransactionTypeBatchPin, core.DataArray{})

	valid := em.validateBatchMessage(context.Background(), batch, 0, batch.Payload.Messages[0])
	assert.True(t, valid)
}
