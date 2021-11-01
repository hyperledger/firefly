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

	transfer := &fftypes.TokenTransfer{
		Type:           fftypes.TokenTransferTypeTransfer,
		PoolProtocolID: "F1",
		TokenIndex:     "0",
		Connector:      "erc1155",
		Key:            "0x12345",
		From:           "0x1",
		To:             "0x2",
		Amount:         *fftypes.NewBigInt(1),
	}
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "F1").Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetTokenPoolByProtocolID", em.ctx, "F1").Return(pool, nil).Times(3)
	mdi.On("UpsertTokenTransfer", em.ctx, transfer).Return(fmt.Errorf("pop")).Once()
	mdi.On("UpsertTokenTransfer", em.ctx, transfer).Return(nil).Times(2)
	mdi.On("UpdateTokenAccountBalances", em.ctx, transfer).Return(fmt.Errorf("pop")).Once()
	mdi.On("UpdateTokenAccountBalances", em.ctx, transfer).Return(nil).Once()
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeTransferConfirmed && ev.Reference == transfer.LocalID && ev.Namespace == pool.Namespace
	})).Return(nil).Once()

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokensTransferred(mti, transfer, "tx1", info)
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
		Type:           fftypes.TokenTransferTypeTransfer,
		PoolProtocolID: "F1",
		TokenIndex:     "0",
		Key:            "0x12345",
		From:           "0x1",
		To:             "0x2",
		Amount:         *fftypes.NewBigInt(1),
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

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "F1").Return(pool, nil).Times(3)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop")).Once()
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(operationsBad, nil, nil).Once()
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(operationsGood, nil, nil).Once()
	mdi.On("GetTransactionByID", em.ctx, transfer.TX.ID).Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetTransactionByID", em.ctx, transfer.TX.ID).Return(nil, nil).Once()
	mdi.On("UpsertTransaction", em.ctx, mock.MatchedBy(func(t *fftypes.Transaction) bool {
		return *t.ID == *transfer.TX.ID && t.Subject.Type == fftypes.TransactionTypeTokenTransfer && t.ProtocolID == "tx1"
	}), false).Return(database.HashMismatch).Once()

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokensTransferred(mti, transfer, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTokensTransferredAddBalanceIgnore(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	transfer := &fftypes.TokenTransfer{
		Type:           fftypes.TokenTransferTypeTransfer,
		PoolProtocolID: "F1",
		TokenIndex:     "0",
		Key:            "0x12345",
		From:           "0x1",
		To:             "0x2",
		Amount:         *fftypes.NewBigInt(1),
	}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "F1").Return(nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokensTransferred(mti, transfer, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTokensTransferredWithMessageReceived(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	transfer := &fftypes.TokenTransfer{
		Type:           fftypes.TokenTransferTypeTransfer,
		PoolProtocolID: "F1",
		TokenIndex:     "0",
		Connector:      "erc1155",
		Key:            "0x12345",
		From:           "0x1",
		To:             "0x2",
		MessageHash:    fftypes.NewRandB32(),
		Amount:         *fftypes.NewBigInt(1),
	}
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}
	messages := []*fftypes.Message{{
		BatchID: fftypes.NewUUID(),
	}}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "F1").Return(pool, nil).Times(2)
	mdi.On("UpsertTokenTransfer", em.ctx, transfer).Return(nil).Times(2)
	mdi.On("UpdateTokenAccountBalances", em.ctx, transfer).Return(nil).Times(2)
	mdi.On("GetMessages", em.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop")).Once()
	mdi.On("GetMessages", em.ctx, mock.Anything).Return(messages, nil, nil).Once()
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeTransferConfirmed && ev.Reference == transfer.LocalID && ev.Namespace == pool.Namespace
	})).Return(nil).Once()

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokensTransferred(mti, transfer, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTokensTransferredWithMessageSend(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	transfer := &fftypes.TokenTransfer{
		Type:           fftypes.TokenTransferTypeTransfer,
		PoolProtocolID: "F1",
		TokenIndex:     "0",
		Connector:      "erc1155",
		Key:            "0x12345",
		From:           "0x1",
		To:             "0x2",
		MessageHash:    fftypes.NewRandB32(),
		Amount:         *fftypes.NewBigInt(1),
	}
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}
	messages := []*fftypes.Message{{
		BatchID: fftypes.NewUUID(),
		State:   fftypes.MessageStateStaged,
	}}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "F1").Return(pool, nil).Times(2)
	mdi.On("UpsertTokenTransfer", em.ctx, transfer).Return(nil).Times(2)
	mdi.On("UpdateTokenAccountBalances", em.ctx, transfer).Return(nil).Times(2)
	mdi.On("GetMessages", em.ctx, mock.Anything).Return(messages, nil, nil).Times(2)
	mdi.On("UpsertMessage", em.ctx, mock.Anything, true, false).Return(fmt.Errorf("pop"))
	mdi.On("UpsertMessage", em.ctx, mock.MatchedBy(func(msg *fftypes.Message) bool {
		return msg.State == fftypes.MessageStateReady
	}), true, false).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeTransferConfirmed && ev.Reference == transfer.LocalID && ev.Namespace == pool.Namespace
	})).Return(nil).Once()

	info := fftypes.JSONObject{"some": "info"}
	err := em.TokensTransferred(mti, transfer, "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}
