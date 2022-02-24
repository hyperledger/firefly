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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newApproval() *tokens.TokenApproval {
	return &tokens.TokenApproval{
		PoolProtocolID: "F1",
		TokenApproval: fftypes.TokenApproval{
			LocalID:    fftypes.NewUUID(),
			Pool:       fftypes.NewUUID(),
			Connector:  "erc1155",
			Namespace:  "ns1",
			Key:        "0x01",
			Operator:   "0x02",
			Approved:   true,
			ProtocolID: "123",
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeTokenApproval,
				ID:   fftypes.NewUUID(),
			},
		},
		Event: blockchain.Event{
			BlockchainTXID: "0xffffeeee",
			Name:           "TokenApproval",
			ProtocolID:     "0000/0000/0000",
			Info:           fftypes.JSONObject{"some": "info"},
		},
	}
}

func TestTokensApprovedSucceedWithRetries(t *testing.T) {
	em, cancel := newTestEventManagerWithMetrics(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	approval := newApproval()
	approval.TX = fftypes.TransactionRef{}
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(pool, nil).Times(2)
	mdi.On("InsertBlockchainEvent", em.ctx, mock.MatchedBy(func(e *fftypes.BlockchainEvent) bool {
		return e.Namespace == pool.Namespace && e.Name == approval.Event.Name
	})).Return(nil).Times(2)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeBlockchainEvent && ev.Namespace == pool.Namespace
	})).Return(nil).Times(2)
	mdi.On("UpsertTokenApproval", em.ctx, &approval.TokenApproval).Return(fmt.Errorf("pop")).Once()
	mdi.On("UpsertTokenApproval", em.ctx, &approval.TokenApproval).Return(nil).Times(1)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeApprovalConfirmed && ev.Reference == approval.LocalID && ev.Namespace == pool.Namespace
	})).Return(nil).Once()

	err := em.TokensApproved(mti, approval)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestPersistApprovalOpFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)

	approval := newApproval()
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(pool, nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	valid, err := em.persistTokenApproval(em.ctx, approval)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestPersistApprovalBadOp(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mth := em.txHelper.(*txcommonmocks.Helper)

	approval := newApproval()
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}
	ops := []*fftypes.Operation{{
		Input: fftypes.JSONObject{
			"localid": "realbad",
		},
		Transaction: fftypes.NewUUID(),
	}}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(pool, nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(ops, nil, nil)
	mth.On("PersistTransaction", mock.Anything, "ns1", approval.TX.ID, fftypes.TransactionTypeTokenApproval, "0xffffeeee").Return(false, fmt.Errorf("pop"))

	valid, err := em.persistTokenApproval(em.ctx, approval)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestPersistApprovalTxFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mth := em.txHelper.(*txcommonmocks.Helper)

	approval := newApproval()
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}
	localID := fftypes.NewUUID()
	ops := []*fftypes.Operation{{
		Input: fftypes.JSONObject{
			"localid": localID.String(),
		},
	}}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(pool, nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(ops, nil, nil)
	mth.On("PersistTransaction", mock.Anything, "ns1", approval.TX.ID, fftypes.TransactionTypeTokenApproval, "0xffffeeee").Return(false, fmt.Errorf("pop"))

	valid, err := em.persistTokenApproval(em.ctx, approval)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestPersistApprovalGetApprovalFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mth := em.txHelper.(*txcommonmocks.Helper)

	approval := newApproval()
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}
	localID := fftypes.NewUUID()
	ops := []*fftypes.Operation{{
		Input: fftypes.JSONObject{
			"localid": localID.String(),
		},
	}}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(pool, nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(ops, nil, nil)
	mth.On("PersistTransaction", mock.Anything, "ns1", approval.TX.ID, fftypes.TransactionTypeTokenApproval, "0xffffeeee").Return(true, nil)
	mdi.On("GetTokenApproval", em.ctx, localID).Return(nil, fmt.Errorf("pop"))

	valid, err := em.persistTokenApproval(em.ctx, approval)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestApprovedBadPool(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	approval := newApproval()
	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(nil, nil)

	err := em.TokensApproved(mti, approval)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestApprovedWithTransactionRegenerateLocalID(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}
	mth := em.txHelper.(*txcommonmocks.Helper)

	approval := newApproval()
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}
	localID := fftypes.NewUUID()
	ops := []*fftypes.Operation{{
		Input: fftypes.JSONObject{
			"localid": localID.String(),
		},
	}}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(pool, nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(ops, nil, nil)
	mth.On("PersistTransaction", mock.Anything, "ns1", approval.TX.ID, fftypes.TransactionTypeTokenApproval, "0xffffeeee").Return(true, nil)
	mdi.On("GetTokenApproval", em.ctx, localID).Return(&fftypes.TokenApproval{}, nil)
	mdi.On("InsertBlockchainEvent", em.ctx, mock.MatchedBy(func(e *fftypes.BlockchainEvent) bool {
		return e.Namespace == pool.Namespace && e.Name == approval.Event.Name
	})).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeBlockchainEvent && ev.Namespace == pool.Namespace
	})).Return(nil)
	mdi.On("UpsertTokenApproval", em.ctx, &approval.TokenApproval).Return(nil)

	valid, err := em.persistTokenApproval(em.ctx, approval)
	assert.True(t, valid)
	assert.NoError(t, err)

	assert.NotEqual(t, *localID, *approval.LocalID)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestApprovedBlockchainEventFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mth := em.txHelper.(*txcommonmocks.Helper)

	approval := newApproval()
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}
	localID := fftypes.NewUUID()
	ops := []*fftypes.Operation{{
		Input: fftypes.JSONObject{
			"localid": localID.String(),
		},
	}}

	mdi.On("GetTokenPoolByProtocolID", em.ctx, "erc1155", "F1").Return(pool, nil)
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(ops, nil, nil)
	mth.On("PersistTransaction", mock.Anything, "ns1", approval.TX.ID, fftypes.TransactionTypeTokenApproval, "0xffffeeee").Return(true, nil)
	mdi.On("GetTokenApproval", em.ctx, localID).Return(&fftypes.TokenApproval{}, nil)
	mdi.On("InsertBlockchainEvent", em.ctx, mock.MatchedBy(func(e *fftypes.BlockchainEvent) bool {
		return e.Namespace == pool.Namespace && e.Name == approval.Event.Name
	})).Return(fmt.Errorf("pop"))

	valid, err := em.persistTokenApproval(em.ctx, approval)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}
