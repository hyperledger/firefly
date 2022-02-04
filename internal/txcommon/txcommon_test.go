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

package txcommon

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSubmitNewTransactionOK(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	var txidInserted *fftypes.UUID
	mdi.On("InsertTransaction", ctx, mock.MatchedBy(func(transaction *fftypes.Transaction) bool {
		txidInserted = transaction.ID
		assert.NotNil(t, transaction.ID)
		assert.Equal(t, "ns1", transaction.Namespace)
		assert.Equal(t, fftypes.TransactionTypeBatchPin, transaction.Type)
		assert.Empty(t, transaction.BlockchainIDs)
		return true
	})).Return(nil)
	mdi.On("InsertEvent", ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeTransactionSubmitted && e.Reference.Equals(txidInserted)
	})).Return(nil)

	txidReturned, err := txHelper.SubmitNewTransaction(ctx, "ns1", fftypes.TransactionTypeBatchPin)
	assert.NoError(t, err)
	assert.Equal(t, *txidInserted, *txidReturned)

	mdi.AssertExpectations(t)

}

func TestSubmitNewTransactionFail(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	mdi.On("InsertTransaction", ctx, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := txHelper.SubmitNewTransaction(ctx, "ns1", fftypes.TransactionTypeBatchPin)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)

}

func TestSubmitNewTransactionEventFail(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	mdi.On("InsertTransaction", ctx, mock.Anything).Return(nil)
	mdi.On("InsertEvent", ctx, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := txHelper.SubmitNewTransaction(ctx, "ns1", fftypes.TransactionTypeBatchPin)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionNew(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, txid).Return(nil, nil)
	mdi.On("InsertTransaction", ctx, mock.MatchedBy(func(transaction *fftypes.Transaction) bool {
		assert.Equal(t, txid, transaction.ID)
		assert.Equal(t, "ns1", transaction.Namespace)
		assert.Equal(t, fftypes.TransactionTypeBatchPin, transaction.Type)
		assert.Equal(t, fftypes.FFStringArray{"0x222222"}, transaction.BlockchainIDs)
		return true
	})).Return(nil)

	valid, err := txHelper.PersistTransaction(ctx, "ns1", txid, fftypes.TransactionTypeBatchPin, "0x222222")
	assert.NoError(t, err)
	assert.True(t, valid)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionNewInserTFail(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, txid).Return(nil, nil)
	mdi.On("InsertTransaction", ctx, mock.Anything).Return(fmt.Errorf("pop"))

	valid, err := txHelper.PersistTransaction(ctx, "ns1", txid, fftypes.TransactionTypeBatchPin, "0x222222")
	assert.Regexp(t, "pop", err)
	assert.False(t, valid)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionExistingAddBlockchainID(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, txid).Return(&fftypes.Transaction{
		ID:            txid,
		Namespace:     "ns1",
		Type:          fftypes.TransactionTypeBatchPin,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}, nil)
	mdi.On("UpdateTransaction", ctx, txid, mock.Anything).Return(nil)

	valid, err := txHelper.PersistTransaction(ctx, "ns1", txid, fftypes.TransactionTypeBatchPin, "0x222222")
	assert.NoError(t, err)
	assert.True(t, valid)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionExistingUpdateFail(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, txid).Return(&fftypes.Transaction{
		ID:            txid,
		Namespace:     "ns1",
		Type:          fftypes.TransactionTypeBatchPin,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}, nil)
	mdi.On("UpdateTransaction", ctx, txid, mock.Anything).Return(fmt.Errorf("pop"))

	valid, err := txHelper.PersistTransaction(ctx, "ns1", txid, fftypes.TransactionTypeBatchPin, "0x222222")
	assert.Regexp(t, "pop", err)
	assert.False(t, valid)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionExistingNoChange(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, txid).Return(&fftypes.Transaction{
		ID:            txid,
		Namespace:     "ns1",
		Type:          fftypes.TransactionTypeBatchPin,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}, nil)

	valid, err := txHelper.PersistTransaction(ctx, "ns1", txid, fftypes.TransactionTypeBatchPin, "0x111111")
	assert.NoError(t, err)
	assert.True(t, valid)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionExistingNoBlockchainID(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, txid).Return(&fftypes.Transaction{
		ID:            txid,
		Namespace:     "ns1",
		Type:          fftypes.TransactionTypeBatchPin,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}, nil)

	valid, err := txHelper.PersistTransaction(ctx, "ns1", txid, fftypes.TransactionTypeBatchPin, "")
	assert.NoError(t, err)
	assert.True(t, valid)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionExistingLookupFail(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, txid).Return(nil, fmt.Errorf("pop"))

	valid, err := txHelper.PersistTransaction(ctx, "ns1", txid, fftypes.TransactionTypeBatchPin, "")
	assert.Regexp(t, "pop", err)
	assert.False(t, valid)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionExistingMismatchNS(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, txid).Return(&fftypes.Transaction{
		ID:            txid,
		Namespace:     "ns2",
		Type:          fftypes.TransactionTypeBatchPin,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}, nil)

	valid, err := txHelper.PersistTransaction(ctx, "ns1", txid, fftypes.TransactionTypeBatchPin, "")
	assert.NoError(t, err)
	assert.False(t, valid)

	mdi.AssertExpectations(t)

}

func TestPersistTransactionExistingMismatchType(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, txid).Return(&fftypes.Transaction{
		ID:            txid,
		Namespace:     "ns1",
		Type:          fftypes.TransactionTypeContractInvoke,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}, nil)

	valid, err := txHelper.PersistTransaction(ctx, "ns1", txid, fftypes.TransactionTypeBatchPin, "")
	assert.NoError(t, err)
	assert.False(t, valid)

	mdi.AssertExpectations(t)

}

func TestAddBlockchainTX(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, txid).Return(&fftypes.Transaction{
		ID:            txid,
		Namespace:     "ns1",
		Type:          fftypes.TransactionTypeContractInvoke,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}, nil)
	mdi.On("UpdateTransaction", ctx, txid, mock.MatchedBy(func(u database.Update) bool {
		info, _ := u.Finalize()
		assert.Equal(t, 1, len(info.SetOperations))
		assert.Equal(t, "blockchainids", info.SetOperations[0].Field)
		val, _ := info.SetOperations[0].Value.Value()
		assert.Equal(t, "0x111111,abc", val)
		return true
	})).Return(nil)

	err := txHelper.AddBlockchainTX(ctx, txid, "abc")
	assert.NoError(t, err)

	mdi.AssertExpectations(t)

}

func TestAddBlockchainTXGetFail(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, txid).Return(nil, fmt.Errorf("pop"))

	err := txHelper.AddBlockchainTX(ctx, txid, "abc")
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)

}

func TestAddBlockchainTXUpdateFail(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, txid).Return(&fftypes.Transaction{
		ID:            txid,
		Namespace:     "ns1",
		Type:          fftypes.TransactionTypeContractInvoke,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}, nil)
	mdi.On("UpdateTransaction", ctx, txid, mock.Anything).Return(fmt.Errorf("pop"))

	err := txHelper.AddBlockchainTX(ctx, txid, "abc")
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)

}

func TestAddBlockchainTXUnchanged(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	txHelper := NewTransactionHelper(mdi)
	ctx := context.Background()

	txid := fftypes.NewUUID()
	mdi.On("GetTransactionByID", ctx, txid).Return(&fftypes.Transaction{
		ID:            txid,
		Namespace:     "ns1",
		Type:          fftypes.TransactionTypeContractInvoke,
		Created:       fftypes.Now(),
		BlockchainIDs: fftypes.FFStringArray{"0x111111"},
	}, nil)

	err := txHelper.AddBlockchainTX(ctx, txid, "0x111111")
	assert.NoError(t, err)

	mdi.AssertExpectations(t)

}
