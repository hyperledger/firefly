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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newApproval() *tokens.TokenApproval {
	return &tokens.TokenApproval{
		PoolLocator: "F1",
		TokenApproval: core.TokenApproval{
			LocalID:    fftypes.NewUUID(),
			Pool:       fftypes.NewUUID(),
			Connector:  "erc1155",
			Namespace:  "ns1",
			Key:        "0x01",
			Operator:   "0x02",
			Approved:   true,
			ProtocolID: "0001/01/01",
			Subject:    "123",
			TX: core.TransactionRef{
				Type: core.TransactionTypeTokenApproval,
				ID:   fftypes.NewUUID(),
			},
		},
		Event: &blockchain.Event{
			BlockchainTXID: "0xffffeeee",
			Name:           "TokenApproval",
			ProtocolID:     "0000/0000/0000",
			Info:           fftypes.JSONObject{"some": "info"},
		},
	}
}

func TestTokensApprovedSucceedWithRetries(t *testing.T) {
	em := newTestEventManagerWithMetrics(t)
	defer em.cleanup(t)

	mti := &tokenmocks.Plugin{}

	approval := newApproval()
	approval.TX = core.TransactionRef{}
	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}

	em.mdi.On("GetTokenPoolByLocator", em.ctx, "ns1", "erc1155", "F1").Return(nil, fmt.Errorf("pop")).Once()
	em.mdi.On("GetTokenPoolByLocator", em.ctx, "ns1", "erc1155", "F1").Return(pool, nil).Times(4)
	em.mdi.On("GetTokenApprovalByProtocolID", em.ctx, "ns1", approval.Connector, approval.ProtocolID).Return(nil, fmt.Errorf("pop")).Once()
	em.mdi.On("GetTokenApprovalByProtocolID", em.ctx, "ns1", approval.Connector, approval.ProtocolID).Return(nil, nil).Times(3)
	em.mth.On("InsertOrGetBlockchainEvent", em.ctx, mock.MatchedBy(func(e *core.BlockchainEvent) bool {
		return e.Namespace == pool.Namespace && e.Name == approval.Event.Name
	})).Return(nil, nil).Times(3)
	em.mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *core.Event) bool {
		return ev.Type == core.EventTypeBlockchainEventReceived && ev.Namespace == pool.Namespace
	})).Return(nil).Times(3)
	em.mdi.On("UpdateTokenApprovals", em.ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Once()
	em.mdi.On("UpdateTokenApprovals", em.ctx, mock.Anything, mock.Anything).Return(nil).Times(2)
	em.mdi.On("UpsertTokenApproval", em.ctx, &approval.TokenApproval).Return(fmt.Errorf("pop")).Once()
	em.mdi.On("UpsertTokenApproval", em.ctx, &approval.TokenApproval).Return(nil).Times(1)
	em.mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *core.Event) bool {
		return ev.Type == core.EventTypeApprovalConfirmed && ev.Reference == approval.LocalID && ev.Namespace == pool.Namespace
	})).Return(nil).Once()

	err := em.TokensApproved(mti, approval)
	assert.NoError(t, err)

	mti.AssertExpectations(t)
}

func TestPersistApprovalDuplicate(t *testing.T) {
	em := newTestEventManager(t)
	defer em.cleanup(t)

	approval := newApproval()
	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}

	em.mdi.On("GetTokenPoolByLocator", em.ctx, "ns1", "erc1155", "F1").Return(pool, nil)
	em.mdi.On("GetTokenApprovalByProtocolID", em.ctx, "ns1", approval.Connector, approval.ProtocolID).Return(&core.TokenApproval{}, nil)

	valid, err := em.persistTokenApproval(em.ctx, approval)
	assert.False(t, valid)
	assert.NoError(t, err)

}

func TestPersistApprovalOpFail(t *testing.T) {
	em := newTestEventManager(t)
	defer em.cleanup(t)

	approval := newApproval()
	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}

	em.mdi.On("GetTokenPoolByLocator", em.ctx, "ns1", "erc1155", "F1").Return(pool, nil)
	em.mdi.On("GetTokenApprovalByProtocolID", em.ctx, "ns1", approval.Connector, approval.ProtocolID).Return(nil, nil)
	em.mdi.On("GetOperations", em.ctx, "ns1", mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	valid, err := em.persistTokenApproval(em.ctx, approval)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

}

func TestPersistApprovalBadOp(t *testing.T) {
	em := newTestEventManager(t)
	defer em.cleanup(t)

	approval := newApproval()
	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}
	ops := []*core.Operation{{
		Input: fftypes.JSONObject{
			"localId": "realbad",
		},
		Transaction: fftypes.NewUUID(),
	}}

	em.mdi.On("GetTokenPoolByLocator", em.ctx, "ns1", "erc1155", "F1").Return(pool, nil)
	em.mdi.On("GetTokenApprovalByProtocolID", em.ctx, "ns1", approval.Connector, approval.ProtocolID).Return(nil, nil)
	em.mdi.On("GetOperations", em.ctx, "ns1", mock.Anything).Return(ops, nil, nil)
	em.mth.On("PersistTransaction", mock.Anything, approval.TX.ID, core.TransactionTypeTokenApproval, "0xffffeeee").Return(false, fmt.Errorf("pop"))

	valid, err := em.persistTokenApproval(em.ctx, approval)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

}

func TestPersistApprovalTxFail(t *testing.T) {
	em := newTestEventManager(t)
	defer em.cleanup(t)

	approval := newApproval()
	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}
	localID := fftypes.NewUUID()
	ops := []*core.Operation{{
		Input: fftypes.JSONObject{
			"localId":   localID.String(),
			"connector": approval.Connector,
			"pool":      pool.ID.String(),
		},
	}}

	em.mdi.On("GetTokenPoolByLocator", em.ctx, "ns1", "erc1155", "F1").Return(pool, nil)
	em.mdi.On("GetTokenApprovalByProtocolID", em.ctx, "ns1", approval.Connector, approval.ProtocolID).Return(nil, nil)
	em.mdi.On("GetOperations", em.ctx, "ns1", mock.Anything).Return(ops, nil, nil)
	em.mdi.On("GetTokenApprovalByID", em.ctx, "ns1", localID).Return(nil, nil)
	em.mth.On("PersistTransaction", mock.Anything, approval.TX.ID, core.TransactionTypeTokenApproval, "0xffffeeee").Return(false, fmt.Errorf("pop"))

	valid, err := em.persistTokenApproval(em.ctx, approval)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

}

func TestPersistApprovalGetApprovalFail(t *testing.T) {
	em := newTestEventManager(t)
	defer em.cleanup(t)

	approval := newApproval()
	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}
	localID := fftypes.NewUUID()
	ops := []*core.Operation{{
		Input: fftypes.JSONObject{
			"localId":   localID.String(),
			"connector": approval.Connector,
			"pool":      pool.ID.String(),
		},
	}}

	em.mdi.On("GetTokenPoolByLocator", em.ctx, "ns1", "erc1155", "F1").Return(pool, nil)
	em.mdi.On("GetTokenApprovalByProtocolID", em.ctx, "ns1", approval.Connector, approval.ProtocolID).Return(nil, nil)
	em.mdi.On("GetOperations", em.ctx, "ns1", mock.Anything).Return(ops, nil, nil)
	em.mdi.On("GetTokenApprovalByID", em.ctx, "ns1", localID).Return(nil, fmt.Errorf("pop"))

	valid, err := em.persistTokenApproval(em.ctx, approval)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

}

func TestApprovedBadPool(t *testing.T) {
	em := newTestEventManager(t)
	defer em.cleanup(t)

	mti := &tokenmocks.Plugin{}

	approval := newApproval()
	em.mdi.On("GetTokenPoolByLocator", em.ctx, "ns1", "erc1155", "F1").Return(nil, nil)

	err := em.TokensApproved(mti, approval)
	assert.NoError(t, err)

	mti.AssertExpectations(t)
}

func TestApprovedWithTransactionRegenerateLocalID(t *testing.T) {
	em := newTestEventManager(t)
	defer em.cleanup(t)

	mti := &tokenmocks.Plugin{}

	approval := newApproval()
	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}
	localID := fftypes.NewUUID()
	ops := []*core.Operation{{
		Input: fftypes.JSONObject{
			"localId":   localID.String(),
			"connector": approval.Connector,
			"pool":      pool.ID.String(),
		},
	}}

	em.mdi.On("GetTokenPoolByLocator", em.ctx, "ns1", "erc1155", "F1").Return(pool, nil)
	em.mdi.On("GetTokenApprovalByProtocolID", em.ctx, "ns1", approval.Connector, approval.ProtocolID).Return(nil, nil)
	em.mdi.On("GetOperations", em.ctx, "ns1", mock.Anything).Return(ops, nil, nil)
	em.mth.On("PersistTransaction", mock.Anything, approval.TX.ID, core.TransactionTypeTokenApproval, "0xffffeeee").Return(true, nil)
	em.mdi.On("GetTokenApprovalByID", em.ctx, "ns1", localID).Return(&core.TokenApproval{}, nil)
	em.mth.On("InsertOrGetBlockchainEvent", em.ctx, mock.MatchedBy(func(e *core.BlockchainEvent) bool {
		return e.Namespace == pool.Namespace && e.Name == approval.Event.Name
	})).Return(nil, nil)
	em.mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(ev *core.Event) bool {
		return ev.Type == core.EventTypeBlockchainEventReceived && ev.Namespace == pool.Namespace
	})).Return(nil)
	em.mdi.On("UpdateTokenApprovals", em.ctx, mock.Anything, mock.Anything).Return(nil)
	em.mdi.On("UpsertTokenApproval", em.ctx, &approval.TokenApproval).Return(nil)

	valid, err := em.persistTokenApproval(em.ctx, approval)
	assert.True(t, valid)
	assert.NoError(t, err)

	assert.NotEqual(t, *localID, *approval.LocalID)

	mti.AssertExpectations(t)
}

func TestApprovedBlockchainEventFail(t *testing.T) {
	em := newTestEventManager(t)
	defer em.cleanup(t)

	approval := newApproval()
	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}
	localID := fftypes.NewUUID()
	ops := []*core.Operation{{
		Input: fftypes.JSONObject{
			"localId":   localID.String(),
			"connector": approval.Connector,
			"pool":      pool.ID.String(),
		},
	}}

	em.mdi.On("GetTokenPoolByLocator", em.ctx, "ns1", "erc1155", "F1").Return(pool, nil)
	em.mdi.On("GetTokenApprovalByProtocolID", em.ctx, "ns1", approval.Connector, approval.ProtocolID).Return(nil, nil)
	em.mdi.On("GetOperations", em.ctx, "ns1", mock.Anything).Return(ops, nil, nil)
	em.mth.On("PersistTransaction", mock.Anything, approval.TX.ID, core.TransactionTypeTokenApproval, "0xffffeeee").Return(true, nil)
	em.mdi.On("GetTokenApprovalByID", em.ctx, "ns1", localID).Return(&core.TokenApproval{}, nil)
	em.mth.On("InsertOrGetBlockchainEvent", em.ctx, mock.MatchedBy(func(e *core.BlockchainEvent) bool {
		return e.Namespace == pool.Namespace && e.Name == approval.Event.Name
	})).Return(nil, fmt.Errorf("pop"))

	valid, err := em.persistTokenApproval(em.ctx, approval)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

}
