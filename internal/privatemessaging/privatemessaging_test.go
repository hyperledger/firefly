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

package privatemessaging

import (
	"context"
	"fmt"
	"testing"

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/mocks/batchmocks"
	"github.com/kaleido-io/firefly/mocks/blockchainmocks"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/mocks/dataexchangemocks"
	"github.com/kaleido-io/firefly/mocks/datamocks"
	"github.com/kaleido-io/firefly/mocks/identitymocks"
	"github.com/kaleido-io/firefly/pkg/blockchain"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestPrivateMessaging(t *testing.T) (*privateMessaging, func()) {
	config.Reset()
	config.Set(config.NodeIdentity, "localnodeid")
	config.Set(config.OrgIdentity, "localorg")
	config.Set(config.GroupCacheTTL, "1m")
	config.Set(config.GroupCacheSize, "1m")

	mdi := &databasemocks.Plugin{}
	mii := &identitymocks.Plugin{}
	mdx := &dataexchangemocks.Plugin{}
	mbi := &blockchainmocks.Plugin{}
	mba := &batchmocks.Manager{}
	mdm := &datamocks.Manager{}

	mba.On("RegisterDispatcher", []fftypes.MessageType{fftypes.MessageTypeGroupInit, fftypes.MessageTypePrivate}, mock.Anything, mock.Anything).Return()

	ctx, cancel := context.WithCancel(context.Background())
	pm, err := NewPrivateMessaging(ctx, mdi, mii, mdx, mbi, mba, mdm)
	assert.NoError(t, err)

	// Default mocks to save boilerplate in the tests
	mdx.On("Name").Return("utdx").Maybe()
	mbi.On("Name").Return("utblk").Maybe()
	mii.On("Resolve", ctx, "org1").Return(&fftypes.Identity{
		Identifier: "org1", OnChain: "0x12345",
	}, nil).Maybe()
	mbi.On("VerifyIdentitySyntax", ctx, mock.MatchedBy(func(i *fftypes.Identity) bool { return i.OnChain == "0x12345" })).Return(nil).Maybe()
	mii.On("Resolve", ctx, "org1").Return(&fftypes.Identity{
		Identifier: "org1", OnChain: "0x23456",
	}, nil).Maybe()
	mbi.On("VerifyIdentitySyntax", ctx, mock.MatchedBy(func(i *fftypes.Identity) bool { return i.OnChain == "0x23456" })).Return(nil).Maybe()

	return pm.(*privateMessaging), cancel
}

func uuidMatches(id1 *fftypes.UUID) interface{} {
	return mock.MatchedBy(func(id2 *fftypes.UUID) bool { return id1.Equals(id2) })
}

func TestDispatchBatch(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	batchID := fftypes.NewUUID()
	groupID := fftypes.NewUUID()
	pin1 := fftypes.NewRandB32()
	pin2 := fftypes.NewRandB32()
	node1 := fftypes.NewUUID()
	node2 := fftypes.NewUUID()
	txID := fftypes.NewUUID()
	batchHash := fftypes.NewRandB32()

	mdi := pm.database.(*databasemocks.Plugin)
	mbi := pm.blockchain.(*blockchainmocks.Plugin)
	mdx := pm.exchange.(*dataexchangemocks.Plugin)

	rag := mdi.On("RunAsGroup", pm.ctx, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}

	mdi.On("GetGroupByID", pm.ctx, groupID).Return(&fftypes.Group{
		ID: groupID,
		Members: fftypes.Members{
			{Identity: "org1", Node: node1},
			{Identity: "org2", Node: node2},
		},
	}, nil)
	mdi.On("GetNodeByID", pm.ctx, uuidMatches(node1)).Return(&fftypes.Node{
		ID: node1,
		DX: fftypes.DXInfo{
			Peer:     "node1",
			Endpoint: fftypes.JSONObject{"url": "https://node1.example.com"},
		},
	}, nil).Once()
	mdi.On("GetNodeByID", pm.ctx, uuidMatches(node2)).Return(&fftypes.Node{
		ID: node2,
		DX: fftypes.DXInfo{
			Peer:     "node2",
			Endpoint: fftypes.JSONObject{"url": "https://node2.example.com"},
		},
	}, nil).Once()

	mdx.On("SendMessage", pm.ctx, mock.Anything, mock.Anything).Return("tracking1", nil).Once()
	mdi.On("UpsertOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.BackendID == "tracking1" && op.Type == fftypes.OpTypeDataExchangeBatchSend
	}), false).Return(nil, nil)
	mdx.On("SendMessage", pm.ctx, mock.Anything, mock.Anything).Return("tracking2", nil).Once()
	mdi.On("UpsertOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.BackendID == "tracking2" && op.Type == fftypes.OpTypeDataExchangeBatchSend
	}), false).Return(nil, nil)

	mdi.On("UpsertTransaction", pm.ctx, mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeBatchPin && tx.ID.Equals(txID)
	}), true, false).Return(nil, nil)
	mbi.On("SubmitBatchPin", pm.ctx, mock.Anything, mock.Anything, mock.MatchedBy(func(bp *blockchain.BatchPin) bool {
		assert.Equal(t, txID, bp.TransactionID)
		assert.Equal(t, batchID, bp.BatchID)
		assert.Equal(t, batchHash, bp.BatchHash)
		assert.Equal(t, "ns1", bp.Namespace)
		assert.Equal(t, []*fftypes.Bytes32{pin1, pin2}, bp.Contexts)
		return true
	})).Return("tracking3", nil)
	mdi.On("UpsertOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.BackendID == "tracking3" && op.Type == fftypes.OpTypeBlockchainBatchPin
	}), false).Return(nil, nil)

	err := pm.dispatchBatch(pm.ctx, &fftypes.Batch{
		ID:        batchID,
		Author:    "org1",
		Group:     groupID,
		Namespace: "ns1",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: txID,
			},
		},
		Hash: batchHash,
	}, []*fftypes.Bytes32{pin1, pin2})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestNewPrivateMessagingMissingDeps(t *testing.T) {
	_, err := NewPrivateMessaging(context.Background(), nil, nil, nil, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestDispatchBatchBadData(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	err := pm.dispatchBatch(pm.ctx, &fftypes.Batch{
		Payload: fftypes.BatchPayload{
			Data: []*fftypes.Data{
				{Value: fftypes.Byteable(`{!json}`)},
			},
		},
	}, []*fftypes.Bytes32{})
	assert.Regexp(t, "FF10137", err)
}

func TestDispatchErrorFindingGroup(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByID", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := pm.dispatchBatch(pm.ctx, &fftypes.Batch{}, []*fftypes.Bytes32{})
	assert.Regexp(t, "pop", err)
}

func TestSendAndSubmitBatchBadID(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByID", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	mii := pm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", pm.ctx, "badauthor").Return(&fftypes.Identity{OnChain: "!badaddress"}, nil)

	mbi := pm.blockchain.(*blockchainmocks.Plugin)
	mbi.On("VerifyIdentitySyntax", pm.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	err := pm.sendAndSubmitBatch(pm.ctx, &fftypes.Batch{
		Author: "badauthor",
	}, []*fftypes.Node{}, fftypes.Byteable(`{}`), []*fftypes.Bytes32{})
	assert.Regexp(t, "pop", err)
}

func TestSendImmediateFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("SendMessage", pm.ctx, mock.Anything, mock.Anything).Return("", fmt.Errorf("pop"))

	err := pm.sendAndSubmitBatch(pm.ctx, &fftypes.Batch{
		Author: "org1",
	}, []*fftypes.Node{
		{
			DX: fftypes.DXInfo{
				Peer:     "node1",
				Endpoint: fftypes.JSONObject{"url": "https://node1.example.com"},
			},
		},
	}, fftypes.Byteable(`{}`), []*fftypes.Bytes32{})
	assert.Regexp(t, "pop", err)
}

func TestSendSubmitUpsertOperationFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("SendMessage", pm.ctx, mock.Anything, mock.Anything).Return("tracking1", nil)

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertOperation", pm.ctx, mock.Anything, false).Return(fmt.Errorf("pop"))

	err := pm.sendAndSubmitBatch(pm.ctx, &fftypes.Batch{
		Author: "org1",
	}, []*fftypes.Node{
		{
			DX: fftypes.DXInfo{
				Peer:     "node1",
				Endpoint: fftypes.JSONObject{"url": "https://node1.example.com"},
			},
		},
	}, fftypes.Byteable(`{}`), []*fftypes.Bytes32{})
	assert.Regexp(t, "pop", err)
}

func TestWriteTransactionUpsertFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertTransaction", pm.ctx, mock.Anything, true, false).Return(fmt.Errorf("pop"))

	err := pm.writeTransaction(pm.ctx, &fftypes.Identity{OnChain: "0x12345"}, &fftypes.Batch{}, []*fftypes.Bytes32{})
	assert.Regexp(t, "pop", err)
}

func TestWriteTransactionSubmitBatchPinFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertTransaction", pm.ctx, mock.Anything, true, false).Return(nil)

	mbi := pm.blockchain.(*blockchainmocks.Plugin)
	mbi.On("SubmitBatchPin", pm.ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", fmt.Errorf("pop"))

	err := pm.writeTransaction(pm.ctx, &fftypes.Identity{OnChain: "0x12345"}, &fftypes.Batch{}, []*fftypes.Bytes32{})
	assert.Regexp(t, "pop", err)
}

func TestWriteTransactionUpsertOpFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertTransaction", pm.ctx, mock.Anything, true, false).Return(nil)

	mbi := pm.blockchain.(*blockchainmocks.Plugin)
	mbi.On("SubmitBatchPin", pm.ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("tracking1", nil)

	mdi.On("UpsertOperation", pm.ctx, mock.Anything, false).Return(fmt.Errorf("pop"))

	err := pm.writeTransaction(pm.ctx, &fftypes.Identity{OnChain: "0x12345"}, &fftypes.Batch{}, []*fftypes.Bytes32{})
	assert.Regexp(t, "pop", err)
}
