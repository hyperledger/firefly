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

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/mocks/batchmocks"
	"github.com/hyperledger/firefly/mocks/batchpinmocks"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestPrivateMessaging(t *testing.T) (*privateMessaging, func()) {
	config.Reset()
	config.Set(config.NodeName, "node1")
	config.Set(config.GroupCacheTTL, "1m")
	config.Set(config.GroupCacheSize, "1m")

	mdi := &databasemocks.Plugin{}
	mim := &identitymanagermocks.Manager{}
	mdx := &dataexchangemocks.Plugin{}
	mbi := &blockchainmocks.Plugin{}
	mba := &batchmocks.Manager{}
	mdm := &datamocks.Manager{}
	msa := &syncasyncmocks.Bridge{}
	mbp := &batchpinmocks.Submitter{}

	mba.On("RegisterDispatcher", []fftypes.MessageType{
		fftypes.MessageTypeGroupInit,
		fftypes.MessageTypePrivate,
		fftypes.MessageTypeTransferPrivate,
	}, mock.Anything, mock.Anything).Return()

	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	pm, err := NewPrivateMessaging(ctx, mdi, mim, mdx, mbi, mba, mdm, msa, mbp)
	assert.NoError(t, err)

	// Default mocks to save boilerplate in the tests
	mdx.On("Name").Return("utdx").Maybe()
	mbi.On("Name").Return("utblk").Maybe()

	return pm.(*privateMessaging), cancel
}

func TestDispatchBatchWithBlobs(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	batchID := fftypes.NewUUID()
	groupID := fftypes.NewRandB32()
	pin1 := fftypes.NewRandB32()
	pin2 := fftypes.NewRandB32()
	node1 := fftypes.NewUUID()
	node2 := fftypes.NewUUID()
	txID := fftypes.NewUUID()
	batchHash := fftypes.NewRandB32()
	dataID1 := fftypes.NewUUID()
	blob1 := fftypes.NewRandB32()

	mdi := pm.database.(*databasemocks.Plugin)
	mbp := pm.batchpin.(*batchpinmocks.Submitter)
	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mim := pm.identity.(*identitymanagermocks.Manager)

	mim.On("ResolveInputIdentity", pm.ctx, mock.Anything).Run(func(args mock.Arguments) {
		identity := args[1].(*fftypes.Identity)
		assert.Equal(t, "org1", identity.Author)
		identity.Key = "0x12345"
	}).Return(nil)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("localorg", nil)
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(&fftypes.Group{
		Hash: fftypes.NewRandB32(),
		GroupIdentity: fftypes.GroupIdentity{
			Name: "group1",
			Members: fftypes.Members{
				{Identity: "org1", Node: node1},
				{Identity: "org2", Node: node2},
			},
		},
	}, nil)
	mdi.On("GetNodeByID", pm.ctx, node1).Return(&fftypes.Node{
		ID: node1,
		DX: fftypes.DXInfo{
			Peer:     "node1",
			Endpoint: fftypes.JSONObject{"url": "https://node1.example.com"},
		},
	}, nil).Once()
	mdi.On("GetNodeByID", pm.ctx, node2).Return(&fftypes.Node{
		ID: node2,
		DX: fftypes.DXInfo{
			Peer:     "node2",
			Endpoint: fftypes.JSONObject{"url": "https://node2.example.com"},
		},
	}, nil).Once()
	mdi.On("GetBlobMatchingHash", pm.ctx, blob1).Return(&fftypes.Blob{
		Hash:       blob1,
		PayloadRef: "/blob/1",
	}, nil)
	mdx.On("TransferBLOB", pm.ctx, "node1", "/blob/1").Return("tracking1", nil)
	mdi.On("InsertOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.BackendID == "tracking1" && op.Type == fftypes.OpTypeDataExchangeBlobSend
	})).Return(nil, nil)
	mdx.On("TransferBLOB", pm.ctx, "node2", "/blob/1").Return("tracking2", nil)
	mdi.On("InsertOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.BackendID == "tracking2" && op.Type == fftypes.OpTypeDataExchangeBlobSend
	})).Return(nil, nil)

	mdx.On("SendMessage", pm.ctx, mock.Anything, mock.Anything).Return("tracking3", nil).Once()
	mdi.On("InsertOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.BackendID == "tracking3" && op.Type == fftypes.OpTypeDataExchangeBatchSend
	})).Return(nil, nil)
	mdx.On("SendMessage", pm.ctx, mock.Anything, mock.Anything).Return("tracking4", nil).Once()
	mdi.On("InsertOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.BackendID == "tracking4" && op.Type == fftypes.OpTypeDataExchangeBatchSend
	})).Return(nil, nil)

	mbp.On("SubmitPinnedBatch", pm.ctx, mock.Anything, mock.Anything).Return(nil)

	err := pm.dispatchBatch(pm.ctx, &fftypes.Batch{
		ID: batchID,
		Identity: fftypes.Identity{
			Author: "org1",
		},
		Group:     groupID,
		Namespace: "ns1",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: txID,
			},
			Data: []*fftypes.Data{
				{ID: dataID1, Blob: &fftypes.BlobRef{Hash: blob1}},
			},
		},
		Hash: batchHash,
	}, []*fftypes.Bytes32{pin1, pin2})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestNewPrivateMessagingMissingDeps(t *testing.T) {
	_, err := NewPrivateMessaging(context.Background(), nil, nil, nil, nil, nil, nil, nil, nil)
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
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := pm.dispatchBatch(pm.ctx, &fftypes.Batch{}, []*fftypes.Bytes32{})
	assert.Regexp(t, "pop", err)
}

func TestSendAndSubmitBatchBadID(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("localorg", nil)
	mim.On("ResolveInputIdentity", pm.ctx, mock.MatchedBy(func(identity *fftypes.Identity) bool {
		assert.Equal(t, "badauthor", identity.Author)
		return true
	})).Return(fmt.Errorf("pop"))

	mbp := pm.batchpin.(*batchpinmocks.Submitter)
	mbp.On("SubmitPinnedBatch", pm.ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := pm.sendAndSubmitBatch(pm.ctx, &fftypes.Batch{
		Identity: fftypes.Identity{
			Author: "badauthor",
		},
	}, []*fftypes.Node{}, fftypes.Byteable(`{}`), []*fftypes.Bytes32{})
	assert.Regexp(t, "pop", err)
}

func TestSendAndSubmitBatchUnregisteredNode(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("", fmt.Errorf("pop"))

	err := pm.sendAndSubmitBatch(pm.ctx, &fftypes.Batch{
		Identity: fftypes.Identity{
			Author: "badauthor",
		},
	}, []*fftypes.Node{}, fftypes.Byteable(`{}`), []*fftypes.Bytes32{})
	assert.Regexp(t, "pop", err)
}

func TestSendImmediateFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("localorg", nil)

	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("SendMessage", pm.ctx, mock.Anything, mock.Anything).Return("", fmt.Errorf("pop"))

	err := pm.sendAndSubmitBatch(pm.ctx, &fftypes.Batch{
		Identity: fftypes.Identity{
			Author: "org1",
		},
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

func TestSendSubmitInsertOperationFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("localorg", nil)

	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("SendMessage", pm.ctx, mock.Anything, mock.Anything).Return("tracking1", nil)

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("InsertOperation", pm.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	err := pm.sendAndSubmitBatch(pm.ctx, &fftypes.Batch{
		Identity: fftypes.Identity{
			Author: "org1",
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: fftypes.NewUUID(),
			},
		},
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

func TestSendSubmitBlobTransferFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("localorg", nil)

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := pm.sendAndSubmitBatch(pm.ctx, &fftypes.Batch{
		Identity: fftypes.Identity{
			Author: "org1",
		},
		Payload: fftypes.BatchPayload{
			Data: []*fftypes.Data{
				{ID: fftypes.NewUUID(), Blob: &fftypes.BlobRef{Hash: fftypes.NewRandB32()}},
			},
		},
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

func TestWriteTransactionSubmitBatchPinFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertTransaction", pm.ctx, mock.Anything, true, false).Return(nil)
	mdi.On("InsertOperation", pm.ctx, mock.Anything).Return(nil)

	mbp := pm.batchpin.(*batchpinmocks.Submitter)
	mbp.On("SubmitPinnedBatch", pm.ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := pm.writeTransaction(pm.ctx, &fftypes.Batch{
		Identity: fftypes.Identity{
			Author: "org1",
		}}, []*fftypes.Bytes32{})
	assert.Regexp(t, "pop", err)
}

func TestTransferBlobsNotFound(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", pm.ctx, mock.Anything).Return(nil, nil)

	err := pm.transferBlobs(pm.ctx, []*fftypes.Data{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32(), Blob: &fftypes.BlobRef{Hash: fftypes.NewRandB32()}},
	}, fftypes.NewUUID(), &fftypes.Node{ID: fftypes.NewUUID(), DX: fftypes.DXInfo{Peer: "peer1"}})
	assert.Regexp(t, "FF10239", err)
}

func TestTransferBlobsFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", pm.ctx, mock.Anything).Return(&fftypes.Blob{PayloadRef: "blob/1"}, nil)
	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("TransferBLOB", pm.ctx, "peer1", "blob/1").Return("", fmt.Errorf("pop"))

	err := pm.transferBlobs(pm.ctx, []*fftypes.Data{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32(), Blob: &fftypes.BlobRef{Hash: fftypes.NewRandB32()}},
	}, fftypes.NewUUID(), &fftypes.Node{ID: fftypes.NewUUID(), DX: fftypes.DXInfo{Peer: "peer1"}})
	assert.Regexp(t, "pop", err)
}

func TestTransferBlobsOpInsertFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdx := pm.exchange.(*dataexchangemocks.Plugin)

	mdi.On("GetBlobMatchingHash", pm.ctx, mock.Anything).Return(&fftypes.Blob{PayloadRef: "blob/1"}, nil)
	mdx.On("TransferBLOB", pm.ctx, "peer1", "blob/1").Return("tracking1", nil)
	mdi.On("InsertOperation", pm.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	err := pm.transferBlobs(pm.ctx, []*fftypes.Data{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32(), Blob: &fftypes.BlobRef{Hash: fftypes.NewRandB32()}},
	}, fftypes.NewUUID(), &fftypes.Node{ID: fftypes.NewUUID(), DX: fftypes.DXInfo{Peer: "peer1"}})
	assert.Regexp(t, "pop", err)
}

func TestStart(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("Start").Return(nil)

	err := pm.Start()
	assert.NoError(t, err)
}
