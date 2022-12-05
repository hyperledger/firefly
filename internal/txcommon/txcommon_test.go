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

package txcommon

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffapi"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/mocks/cachemocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func NewTestTransactionHelper(di database.Plugin, dm data.Manager) (Helper, cache.CInterface, cache.CInterface) {
	t := &transactionHelper{
		namespace: "ns1",
		database:  di,
		data:      dm,
	}
	t.transactionCache = cache.NewUmanagedCache(context.Background(), config.GetByteSize(coreconfig.CacheTransactionSize), config.GetDuration(coreconfig.CacheTransactionTTL))
	t.blockchainEventCache = cache.NewUmanagedCache(context.Background(), config.GetByteSize(coreconfig.CacheBlockchainEventLimit), config.GetDuration(coreconfig.CacheBlockchainEventTTL))
	return t, t.transactionCache, t.blockchainEventCache
}

func TestSubmitNewTransactionOK(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, err := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	var txidInserted *fftypes.UUID
	mdi.On("InsertTransaction", ctx, mock.MatchedBy(func(transaction *core.Transaction) bool {
		txidInserted = transaction.ID
		assert.NotNil(t, transaction.ID)
		assert.Equal(t, "ns1", transaction.Namespace)
		assert.Equal(t, core.TransactionTypeBatchPin, transaction.Type)
		assert.Empty(t, transaction.BlockchainIDs)
		return true
	})).Return(nil)
	mdi.On("InsertEvent", ctx, mock.MatchedBy(func(e *core.Event) bool {
		return e.Type == core.EventTypeTransactionSubmitted && e.Reference.Equals(txidInserted)
	})).Return(nil)

	txidReturned, err := txHelper.SubmitNewTransaction(ctx, core.TransactionTypeBatchPin, "idem1")
	assert.NoError(t, err)
	assert.Equal(t, *txidInserted, *txidReturned)

	mdi.AssertExpectations(t)

}

func TestSubmitNewTransactionFail(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	mdi.On("InsertTransaction", ctx, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := txHelper.SubmitNewTransaction(ctx, core.TransactionTypeBatchPin, "idem1")
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)

}

func TestSubmitNewTransactionEventFail(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	mdi.On("InsertTransaction", ctx, mock.Anything).Return(nil)
	mdi.On("InsertEvent", ctx, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := txHelper.SubmitNewTransaction(ctx, core.TransactionTypeBatchPin, "idem1")
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)

}

func TestCacheInitFail(t *testing.T) {
	cacheInitErr := errors.New("Initialization error.")
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	ns := "ns1"
	tErrcmi := &cachemocks.Manager{}
	tErrcmi.On("GetCache", cache.NewCacheConfig(
		ctx,
		coreconfig.CacheTransactionSize,
		coreconfig.CacheTransactionTTL,
		ns,
	)).Return(nil, cacheInitErr).Once()
	defer tErrcmi.AssertExpectations(t)
	_, err := NewTransactionHelper(ctx, ns, mdi, mdm, tErrcmi)
	assert.Equal(t, cacheInitErr, err)

	bErrcmi := &cachemocks.Manager{}
	bErrcmi.On("GetCache", cache.NewCacheConfig(
		ctx,
		coreconfig.CacheTransactionSize,
		coreconfig.CacheTransactionTTL,
		ns,
	)).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil).Once()
	bErrcmi.On("GetCache", cache.NewCacheConfig(
		ctx,
		coreconfig.CacheBlockchainEventLimit,
		coreconfig.CacheBlockchainEventTTL,
		ns,
	)).Return(nil, cacheInitErr).Once()
	defer bErrcmi.AssertExpectations(t)
	_, err = NewTransactionHelper(ctx, ns, mdi, mdm, bErrcmi)
	assert.Equal(t, cacheInitErr, err)
}

func TestPersistTransactionNew(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, "ns1", txid).Return(nil, nil)
	mdi.On("InsertTransaction", ctx, mock.MatchedBy(func(transaction *core.Transaction) bool {
		assert.Equal(t, txid, transaction.ID)
		assert.Equal(t, "ns1", transaction.Namespace)
		assert.Equal(t, core.TransactionTypeBatchPin, transaction.Type)
		assert.Equal(t, fftypes.FFStringArray{"0x222222"}, transaction.BlockchainIDs)
		return true
	})).Return(nil)

	valid, err := txHelper.PersistTransaction(ctx, txid, core.TransactionTypeBatchPin, "0x222222")
	assert.NoError(t, err)
	assert.True(t, valid)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionNewInserTFail(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, "ns1", txid).Return(nil, nil)
	mdi.On("InsertTransaction", ctx, mock.Anything).Return(fmt.Errorf("pop"))

	valid, err := txHelper.PersistTransaction(ctx, txid, core.TransactionTypeBatchPin, "0x222222")
	assert.Regexp(t, "pop", err)
	assert.False(t, valid)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionExistingAddBlockchainID(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, "ns1", txid).Return(&core.Transaction{
		ID:            txid,
		Namespace:     "ns1",
		Type:          core.TransactionTypeBatchPin,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}, nil)
	mdi.On("UpdateTransaction", ctx, "ns1", txid, mock.Anything).Return(nil)

	valid, err := txHelper.PersistTransaction(ctx, txid, core.TransactionTypeBatchPin, "0x222222")
	assert.NoError(t, err)
	assert.True(t, valid)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionExistingUpdateFail(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, "ns1", txid).Return(&core.Transaction{
		ID:            txid,
		Namespace:     "ns1",
		Type:          core.TransactionTypeBatchPin,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}, nil)
	mdi.On("UpdateTransaction", ctx, "ns1", txid, mock.Anything).Return(fmt.Errorf("pop"))

	valid, err := txHelper.PersistTransaction(ctx, txid, core.TransactionTypeBatchPin, "0x222222")
	assert.Regexp(t, "pop", err)
	assert.False(t, valid)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionExistingNoChange(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	ns := "ns1"
	txHelper, _ := NewTransactionHelper(ctx, ns, mdi, mdm, cmi)
	cmi.AssertCalled(t, "GetCache", cache.NewCacheConfig(
		ctx,
		coreconfig.CacheTransactionSize,
		coreconfig.CacheTransactionTTL,
		ns,
	))
	cmi.AssertCalled(t, "GetCache", cache.NewCacheConfig(
		ctx,
		coreconfig.CacheBlockchainEventLimit,
		coreconfig.CacheBlockchainEventTTL,
		ns,
	))

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, "ns1", txid).Return(&core.Transaction{
		ID:            txid,
		Namespace:     "ns1",
		Type:          core.TransactionTypeBatchPin,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}, nil)

	valid, err := txHelper.PersistTransaction(ctx, txid, core.TransactionTypeBatchPin, "0x111111")
	assert.NoError(t, err)
	assert.True(t, valid)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionExistingNoBlockchainID(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, "ns1", txid).Return(&core.Transaction{
		ID:            txid,
		Namespace:     "ns1",
		Type:          core.TransactionTypeBatchPin,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}, nil)

	valid, err := txHelper.PersistTransaction(ctx, txid, core.TransactionTypeBatchPin, "")
	assert.NoError(t, err)
	assert.True(t, valid)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionExistingLookupFail(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, "ns1", txid).Return(nil, fmt.Errorf("pop"))

	valid, err := txHelper.PersistTransaction(ctx, txid, core.TransactionTypeBatchPin, "")
	assert.Regexp(t, "pop", err)
	assert.False(t, valid)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionExistingMismatchType(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, "ns1", txid).Return(&core.Transaction{
		ID:            txid,
		Namespace:     "ns1",
		Type:          core.TransactionTypeContractInvoke,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}, nil)

	valid, err := txHelper.PersistTransaction(ctx, txid, core.TransactionTypeBatchPin, "")
	assert.NoError(t, err)
	assert.False(t, valid)

	mdi.AssertExpectations(t)

}

func TestAddBlockchainTX(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	tx := &core.Transaction{
		ID:            fftypes.NewUUID(),
		Namespace:     "ns1",
		Type:          core.TransactionTypeContractInvoke,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}
	mdi.On("UpdateTransaction", ctx, "ns1", tx.ID, mock.MatchedBy(func(u ffapi.Update) bool {
		info, _ := u.Finalize()
		assert.Equal(t, 1, len(info.SetOperations))
		assert.Equal(t, "blockchainids", info.SetOperations[0].Field)
		val, _ := info.SetOperations[0].Value.Value()
		assert.Equal(t, "0x111111,abc", val)
		return true
	})).Return(nil)

	err := txHelper.AddBlockchainTX(ctx, tx, "abc")
	assert.NoError(t, err)

	mdi.AssertExpectations(t)

}

func TestAddBlockchainTXUpdateFail(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	tx := &core.Transaction{
		ID:            fftypes.NewUUID(),
		Namespace:     "ns1",
		Type:          core.TransactionTypeContractInvoke,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}
	mdi.On("UpdateTransaction", ctx, "ns1", tx.ID, mock.Anything).Return(fmt.Errorf("pop"))

	err := txHelper.AddBlockchainTX(ctx, tx, "abc")
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)

}

func TestAddBlockchainTXUnchanged(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	tx := &core.Transaction{
		ID:            fftypes.NewUUID(),
		Namespace:     "ns1",
		Type:          core.TransactionTypeContractInvoke,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}

	err := txHelper.AddBlockchainTX(ctx, tx, "0x111111")
	assert.NoError(t, err)

}

func TestGetTransactionByIDCached(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper, _, _ := NewTestTransactionHelper(mdi, mdm)
	ctx := context.Background()

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, "ns1", txid).Return(&core.Transaction{
		ID:            txid,
		Namespace:     "ns1",
		Type:          core.TransactionTypeContractInvoke,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}, nil).Once()

	tx, err := txHelper.GetTransactionByIDCached(ctx, txid)
	assert.NoError(t, err)
	assert.Equal(t, txid, tx.ID)

	// second attempt is cached
	tx, err = txHelper.GetTransactionByIDCached(ctx, txid)
	assert.NoError(t, err)
	assert.Equal(t, txid, tx.ID)

	mdi.AssertExpectations(t)

}

func TestGetTransactionByIDCachedFail(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper, _, _ := NewTestTransactionHelper(mdi, mdm)
	ctx := context.Background()

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, "ns1", txid).Return(nil, fmt.Errorf("pop"))

	_, err := txHelper.GetTransactionByIDCached(ctx, txid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)

}

func TestGetBlockchainEventByIDCached(t *testing.T) {
	config.Set(coreconfig.CacheEnabled, true)
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	evID := fftypes.NewUUID()
	mdi.On("GetBlockchainEventByID", ctx, "ns1", evID).Return(&core.BlockchainEvent{
		ID:        evID,
		Namespace: "ns1",
	}, nil).Once()

	chainEvent, err := txHelper.GetBlockchainEventByIDCached(ctx, evID)
	assert.NoError(t, err)
	assert.Equal(t, evID, chainEvent.ID)

	chainEvent, err = txHelper.GetBlockchainEventByIDCached(ctx, evID)
	assert.NoError(t, err)
	assert.Equal(t, evID, chainEvent.ID)

	mdi.AssertExpectations(t)

}

func TestGetBlockchainEventByIDNil(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	evID := fftypes.NewUUID()
	mdi.On("GetBlockchainEventByID", ctx, "ns1", evID).Return(nil, nil)

	chainEvent, err := txHelper.GetBlockchainEventByIDCached(ctx, evID)
	assert.NoError(t, err)
	assert.Nil(t, chainEvent)

	mdi.AssertExpectations(t)

}

func TestGetBlockchainEventByIDErr(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	evID := fftypes.NewUUID()
	mdi.On("GetBlockchainEventByID", ctx, "ns1", evID).Return(nil, fmt.Errorf("pop"))

	_, err := txHelper.GetBlockchainEventByIDCached(ctx, evID)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)

}

func TestInsertGetBlockchainEventCached(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	evID := fftypes.NewUUID()
	chainEvent := &core.BlockchainEvent{
		ID:        evID,
		Namespace: "ns1",
	}
	mdi.On("InsertOrGetBlockchainEvent", ctx, chainEvent).Return(nil, nil)

	_, err := txHelper.InsertOrGetBlockchainEvent(ctx, chainEvent)
	assert.NoError(t, err)

	cached, err := txHelper.GetBlockchainEventByIDCached(ctx, evID)
	assert.NoError(t, err)
	assert.Equal(t, chainEvent, cached)

	mdi.AssertExpectations(t)

}

func TestInsertBlockchainEventDuplicate(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	evID := fftypes.NewUUID()
	chainEvent := &core.BlockchainEvent{
		ID:        evID,
		Namespace: "ns1",
	}
	existingEvent := &core.BlockchainEvent{}
	mdi.On("InsertOrGetBlockchainEvent", ctx, chainEvent).Return(existingEvent, nil)

	result, err := txHelper.InsertOrGetBlockchainEvent(ctx, chainEvent)
	assert.NoError(t, err)
	assert.Equal(t, existingEvent, result)

	mdi.AssertExpectations(t)

}

func TestInsertBlockchainEventErr(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	evID := fftypes.NewUUID()
	chainEvent := &core.BlockchainEvent{
		ID:        evID,
		Namespace: "ns1",
	}
	mdi.On("InsertOrGetBlockchainEvent", ctx, chainEvent).Return(nil, fmt.Errorf("pop"))

	_, err := txHelper.InsertOrGetBlockchainEvent(ctx, chainEvent)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)

}
