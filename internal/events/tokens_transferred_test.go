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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTokensTransferredSucceedWithRetries(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	uri := "firefly://token/1"
	transfer := &fftypes.TokenTransfer{
		Type:       fftypes.TokenTransferTypeTransfer,
		TokenIndex: "0",
		URI:        uri,
		Connector:  "erc1155",
		Key:        "0x12345",
		From:       "0x1",
		To:         "0x2",
		ProtocolID: "123",
		Amount:     *fftypes.NewBigInt(1),
	}
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}

	mdi.On("GetTokenTransferByProtocolID", em.ctx, "erc1155", "123").Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetTokenTransferByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil).Times(4)
	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(pool, nil).Times(3)
	mdi.On("UpsertTokenTransfer", em.ctx, transfer).Return(fmt.Errorf("pop")).Once()
	mdi.On("UpsertTokenTransfer", em.ctx, transfer).Return(nil).Times(2)
	mdi.On("UpdateTokenBalances", em.ctx, transfer).Return(fmt.Errorf("pop")).Once()
	mdi.On("UpdateTokenBalances", em.ctx, transfer).Return(nil).Once()
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeTransferConfirmed && ev.Reference == transfer.LocalID && ev.Namespace == pool.Namespace
	})).Return(nil).Once()

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokensTransferred(mti, "F1", transfer, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTokensTransferredIgnoreExisting(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	transfer := &fftypes.TokenTransfer{
		Type:       fftypes.TokenTransferTypeTransfer,
		TokenIndex: "0",
		Connector:  "erc1155",
		Key:        "0x12345",
		From:       "0x1",
		To:         "0x2",
		ProtocolID: "123",
		Amount:     *fftypes.NewBigInt(1),
	}

	mdi.On("GetTokenTransferByProtocolID", em.ctx, "erc1155", "123").Return(&fftypes.TokenTransfer{}, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokensTransferred(mti, "F1", transfer, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTokensTransferredWithTransactionRetries(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	transfer := &fftypes.TokenTransfer{
		Type:       fftypes.TokenTransferTypeTransfer,
		TokenIndex: "0",
		Connector:  "erc1155",
		Key:        "0x12345",
		From:       "0x1",
		To:         "0x2",
		ProtocolID: "123",
		Amount:     *fftypes.NewBigInt(1),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenTransfer,
		},
	}
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}
	operationsBad := []*fftypes.Operation{{
		Input: fftypes.JSONObject{
			"id": "bad",
		},
	}}
	operationsGood := []*fftypes.Operation{{
		Input: fftypes.JSONObject{
			"id": fftypes.NewUUID().String(),
		},
	}}

	mdi.On("GetTokenTransferByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil).Times(3)
	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(pool, nil).Times(3)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop")).Once()
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(operationsBad, nil, nil).Once()
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(operationsGood, nil, nil).Once()
	mdi.On("GetTransactionByID", em.ctx, transfer.TX.ID).Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetTransactionByID", em.ctx, transfer.TX.ID).Return(nil, nil).Once()
	mdi.On("UpsertTransaction", em.ctx, mock.MatchedBy(func(t *fftypes.Transaction) bool {
		return *t.ID == *transfer.TX.ID && t.Subject.Type == fftypes.TransactionTypeTokenTransfer && t.ProtocolID == "tx1"
	}), false).Return(database.HashMismatch).Once()

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokensTransferred(mti, "F1", transfer, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTokensTransferredWithTransactionLoadLocalID(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	transfer := &fftypes.TokenTransfer{
		Type:       fftypes.TokenTransferTypeTransfer,
		TokenIndex: "0",
		Connector:  "erc1155",
		Key:        "0x12345",
		From:       "0x1",
		To:         "0x2",
		ProtocolID: "123",
		Amount:     *fftypes.NewBigInt(1),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenTransfer,
		},
	}
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}
	localID := fftypes.NewUUID()
	operations := []*fftypes.Operation{{
		Input: fftypes.JSONObject{
			"id": localID.String(),
		},
	}}

	mdi.On("GetTokenTransferByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil).Times(2)
	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(pool, nil).Times(2)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(operations, nil, nil).Times(2)
	mdi.On("GetTransactionByID", em.ctx, transfer.TX.ID).Return(nil, nil).Times(2)
	mdi.On("UpsertTransaction", em.ctx, mock.MatchedBy(func(t *fftypes.Transaction) bool {
		return *t.ID == *transfer.TX.ID && t.Subject.Type == fftypes.TransactionTypeTokenTransfer && t.ProtocolID == "tx1"
	}), false).Return(nil).Times(2)
	mdi.On("GetTokenTransfer", em.ctx, localID).Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetTokenTransfer", em.ctx, localID).Return(nil, nil).Once()
	mdi.On("UpsertTokenTransfer", em.ctx, transfer).Return(nil).Once()
	mdi.On("UpdateTokenBalances", em.ctx, transfer).Return(nil).Once()
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeTransferConfirmed && ev.Reference == transfer.LocalID && ev.Namespace == pool.Namespace
	})).Return(nil).Once()

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokensTransferred(mti, "F1", transfer, "tx1", info)
	assert.NoError(t, err)

	assert.Equal(t, *localID, *transfer.LocalID)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTokensTransferredWithTransactionRegenerateLocalID(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	transfer := &fftypes.TokenTransfer{
		Type:       fftypes.TokenTransferTypeTransfer,
		TokenIndex: "0",
		Connector:  "erc1155",
		Key:        "0x12345",
		From:       "0x1",
		To:         "0x2",
		ProtocolID: "123",
		Amount:     *fftypes.NewBigInt(1),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenTransfer,
		},
	}
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}
	localID := fftypes.NewUUID()
	operations := []*fftypes.Operation{{
		Input: fftypes.JSONObject{
			"id": localID.String(),
		},
	}}

	mdi.On("GetTokenTransferByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil).Once()
	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(pool, nil).Once()
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(operations, nil, nil).Once()
	mdi.On("GetTransactionByID", em.ctx, transfer.TX.ID).Return(nil, nil).Once()
	mdi.On("UpsertTransaction", em.ctx, mock.MatchedBy(func(t *fftypes.Transaction) bool {
		return *t.ID == *transfer.TX.ID && t.Subject.Type == fftypes.TransactionTypeTokenTransfer && t.ProtocolID == "tx1"
	}), false).Return(nil).Once()
	mdi.On("GetTokenTransfer", em.ctx, localID).Return(&fftypes.TokenTransfer{}, nil).Once()
	mdi.On("UpsertTokenTransfer", em.ctx, transfer).Return(nil).Once()
	mdi.On("UpdateTokenBalances", em.ctx, transfer).Return(nil).Once()
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeTransferConfirmed && ev.Reference == transfer.LocalID && ev.Namespace == pool.Namespace
	})).Return(nil).Once()

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokensTransferred(mti, "F1", transfer, "tx1", info)
	assert.NoError(t, err)

	assert.NotEqual(t, *localID, *transfer.LocalID)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTokensTransferredBadPool(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	transfer := &fftypes.TokenTransfer{
		Type:       fftypes.TokenTransferTypeTransfer,
		TokenIndex: "0",
		Connector:  "erc1155",
		Key:        "0x12345",
		From:       "0x1",
		To:         "0x2",
		ProtocolID: "123",
		Amount:     *fftypes.NewBigInt(1),
	}

	mdi.On("GetTokenTransferByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil)
	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokensTransferred(mti, "F1", transfer, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTokensTransferredWithMessageReceived(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	uri := "firefly://token/1"
	transfer := &fftypes.TokenTransfer{
		Type:       fftypes.TokenTransferTypeTransfer,
		TokenIndex: "0",
		URI:        uri,
		Connector:  "erc1155",
		Key:        "0x12345",
		From:       "0x1",
		To:         "0x2",
		ProtocolID: "123",
		Message:    fftypes.NewUUID(),
		Amount:     *fftypes.NewBigInt(1),
	}
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}
	message := &fftypes.Message{
		BatchID: fftypes.NewUUID(),
	}

	mdi.On("GetTokenTransferByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil).Times(2)
	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(pool, nil).Times(2)
	mdi.On("UpsertTokenTransfer", em.ctx, transfer).Return(nil).Times(2)
	mdi.On("UpdateTokenBalances", em.ctx, transfer).Return(nil).Times(2)
	mdi.On("GetMessageByID", em.ctx, transfer.Message).Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetMessageByID", em.ctx, transfer.Message).Return(message, nil).Once()
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeTransferConfirmed && ev.Reference == transfer.LocalID && ev.Namespace == pool.Namespace
	})).Return(nil).Once()

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokensTransferred(mti, "F1", transfer, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTokensTransferredWithMessageSend(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	uri := "firefly://token/1"
	transfer := &fftypes.TokenTransfer{
		Type:       fftypes.TokenTransferTypeTransfer,
		TokenIndex: "0",
		URI:        uri,
		Connector:  "erc1155",
		Key:        "0x12345",
		From:       "0x1",
		To:         "0x2",
		ProtocolID: "123",
		Message:    fftypes.NewUUID(),
		Amount:     *fftypes.NewBigInt(1),
	}
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}
	message := &fftypes.Message{
		BatchID: fftypes.NewUUID(),
		State:   fftypes.MessageStateStaged,
	}

	mdi.On("GetTokenTransferByProtocolID", em.ctx, "erc1155", "123").Return(nil, nil).Times(2)
	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(pool, nil).Times(2)
	mdi.On("UpsertTokenTransfer", em.ctx, transfer).Return(nil).Times(2)
	mdi.On("UpdateTokenBalances", em.ctx, transfer).Return(nil).Times(2)
	mdi.On("GetMessageByID", em.ctx, mock.Anything).Return(message, nil).Times(2)
	mdi.On("UpsertMessage", em.ctx, mock.Anything, database.UpsertOptimizationExisting).Return(fmt.Errorf("pop"))
	mdi.On("UpsertMessage", em.ctx, mock.MatchedBy(func(msg *fftypes.Message) bool {
		return msg.State == fftypes.MessageStateReady
	}), database.UpsertOptimizationExisting).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeTransferConfirmed && ev.Reference == transfer.LocalID && ev.Namespace == pool.Namespace
	})).Return(nil).Once()

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokensTransferred(mti, "F1", transfer, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}
