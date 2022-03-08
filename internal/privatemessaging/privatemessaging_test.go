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

	"github.com/hyperledger/firefly/internal/batch"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/mocks/batchmocks"
	"github.com/hyperledger/firefly/mocks/batchpinmocks"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestPrivateMessagingCommon(t *testing.T, metricsEnabled bool) (*privateMessaging, func()) {
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
	mmi := &metricsmocks.Manager{}
	mom := &operationmocks.Manager{}

	mba.On("RegisterDispatcher",
		pinnedPrivateDispatcherName,
		fftypes.TransactionTypeBatchPin,
		[]fftypes.MessageType{
			fftypes.MessageTypeGroupInit,
			fftypes.MessageTypePrivate,
			fftypes.MessageTypeTransferPrivate,
		}, mock.Anything, mock.Anything).Return()

	mba.On("RegisterDispatcher",
		unpinnedPrivateDispatcherName,
		fftypes.TransactionTypeUnpinned,
		[]fftypes.MessageType{
			fftypes.MessageTypePrivate,
		}, mock.Anything, mock.Anything).Return()
	mmi.On("IsMetricsEnabled").Return(metricsEnabled)
	mom.On("RegisterHandler", mock.Anything, mock.Anything, mock.Anything)

	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	pm, err := NewPrivateMessaging(ctx, mdi, mim, mdx, mbi, mba, mdm, msa, mbp, mmi, mom)
	assert.NoError(t, err)

	// Default mocks to save boilerplate in the tests
	mdx.On("Name").Return("utdx").Maybe()
	mbi.On("Name").Return("utblk").Maybe()

	return pm.(*privateMessaging), cancel
}

func newTestPrivateMessaging(t *testing.T) (*privateMessaging, func()) {
	return newTestPrivateMessagingCommon(t, false)
}

func newTestPrivateMessagingWithMetrics(t *testing.T) (*privateMessaging, func()) {
	pm, cancel := newTestPrivateMessagingCommon(t, true)
	mmi := pm.metrics.(*metricsmocks.Manager)
	mmi.On("MessageSubmitted", mock.Anything).Return()
	return pm, cancel
}

func TestName(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()
	assert.Equal(t, "PrivateMessaging", pm.Name())
}

func TestDispatchBatchWithBlobs(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	localOrg := newTestOrg("localorg")
	batchID := fftypes.NewUUID()
	groupID := fftypes.NewRandB32()
	pin1 := fftypes.NewRandB32()
	pin2 := fftypes.NewRandB32()
	node1 := newTestNode("node1", localOrg)
	node2 := newTestNode("node2", newTestOrg("remoteorg"))
	txID := fftypes.NewUUID()
	batchHash := fftypes.NewRandB32()
	dataID1 := fftypes.NewUUID()
	blob1 := fftypes.NewRandB32()

	mdi := pm.database.(*databasemocks.Plugin)
	mbp := pm.batchpin.(*batchpinmocks.Submitter)
	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mim := pm.identity.(*identitymanagermocks.Manager)
	mom := pm.operations.(*operationmocks.Manager)

	mim.On("GetNodeOwnerOrg", pm.ctx).Return(localOrg, nil)
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(&fftypes.Group{
		Hash: fftypes.NewRandB32(),
		GroupIdentity: fftypes.GroupIdentity{
			Name: "group1",
			Members: fftypes.Members{
				{Identity: "org1", Node: node1.ID},
				{Identity: "org2", Node: node2.ID},
			},
		},
	}, nil)
	mdi.On("GetIdentityByID", pm.ctx, node1.ID).Return(node1, nil).Once()
	mdi.On("GetIdentityByID", pm.ctx, node2.ID).Return(node2, nil).Once()
	mdi.On("GetBlobMatchingHash", pm.ctx, blob1).Return(&fftypes.Blob{
		Hash:       blob1,
		PayloadRef: "/blob/1",
	}, nil)
	mom.On("AddOrReuseOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Type == fftypes.OpTypeDataExchangeBlobSend
	})).Return(nil, nil)
	mom.On("AddOrReuseOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Type == fftypes.OpTypeDataExchangeBlobSend
	})).Return(nil, nil)
	mom.On("AddOrReuseOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Type == fftypes.OpTypeDataExchangeBatchSend
	})).Return(nil, nil)
	mom.On("AddOrReuseOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Type == fftypes.OpTypeDataExchangeBatchSend
	})).Return(nil, nil)
	mom.On("RunOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		if op.Type != fftypes.OpTypeDataExchangeBlobSend {
			return false
		}
		data := op.Data.(transferBlobData)
		return *data.Node.ID == *node2.ID
	})).Return(nil)
	mom.On("RunOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		if op.Type != fftypes.OpTypeDataExchangeBatchSend {
			return false
		}
		data := op.Data.(batchSendData)
		return *data.Node.ID == *node2.ID
	})).Return(nil)

	mbp.On("SubmitPinnedBatch", pm.ctx, mock.Anything, mock.Anything).Return(nil)

	err := pm.dispatchPinnedBatch(pm.ctx, &batch.DispatchState{
		Persisted: fftypes.BatchPersisted{
			BatchHeader: fftypes.BatchHeader{
				ID: batchID,
				SignerRef: fftypes.SignerRef{
					Author: "org1",
				},
				Group:     groupID,
				Namespace: "ns1",
				Hash:      batchHash,
			},
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: txID,
			},
			Data: fftypes.DataArray{
				{ID: dataID1, Blob: &fftypes.BlobRef{Hash: blob1}},
			},
		},
		Pins: []*fftypes.Bytes32{pin1, pin2},
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mbp.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestNewPrivateMessagingMissingDeps(t *testing.T) {
	_, err := NewPrivateMessaging(context.Background(), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestDispatchErrorFindingGroup(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := pm.dispatchPinnedBatch(pm.ctx, &batch.DispatchState{})
	assert.Regexp(t, "pop", err)
}

func TestSendAndSubmitBatchBadID(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	mbp := pm.batchpin.(*batchpinmocks.Submitter)
	mbp.On("SubmitPinnedBatch", pm.ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := pm.dispatchPinnedBatch(pm.ctx, &batch.DispatchState{
		Persisted: fftypes.BatchPersisted{
			BatchHeader: fftypes.BatchHeader{
				SignerRef: fftypes.SignerRef{
					Author: "badauthor",
				},
			},
		},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestSendAndSubmitBatchUnregisteredNode(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	groupID := fftypes.NewRandB32()
	node1 := newTestNode("node1", newTestOrg("localorg"))
	node2 := newTestNode("node2", newTestOrg("remoteorg"))

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByID", pm.ctx, node1.ID).Return(node1, nil).Once()
	mdi.On("GetIdentityByID", pm.ctx, node2.ID).Return(node2, nil).Once()
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(&fftypes.Group{
		Hash: fftypes.NewRandB32(),
		GroupIdentity: fftypes.GroupIdentity{
			Name: "group1",
			Members: fftypes.Members{
				{Identity: "org1", Node: node1.ID},
				{Identity: "org2", Node: node2.ID},
			},
		},
	}, nil)

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", pm.ctx).Return(nil, fmt.Errorf("pop"))

	err := pm.dispatchPinnedBatch(pm.ctx, &batch.DispatchState{
		Persisted: fftypes.BatchPersisted{
			BatchHeader: fftypes.BatchHeader{
				Group: groupID,
				SignerRef: fftypes.SignerRef{
					Author: "badauthor",
				},
			},
		},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestSendImmediateFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := pm.dispatchPinnedBatch(pm.ctx, &batch.DispatchState{
		Persisted: fftypes.BatchPersisted{
			BatchHeader: fftypes.BatchHeader{
				SignerRef: fftypes.SignerRef{
					Author: "org1",
				},
			},
		},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestSendSubmitInsertOperationFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	localOrg := newTestOrg("localorg")
	groupID := fftypes.NewRandB32()
	node1 := newTestNode("node1", localOrg)
	node2 := newTestNode("node2", newTestOrg("remoteorg"))

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", pm.ctx).Return(localOrg, nil)

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByID", pm.ctx, node1.ID).Return(node1, nil).Once()
	mdi.On("GetIdentityByID", pm.ctx, node2.ID).Return(node2, nil).Once()
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(&fftypes.Group{
		Hash: fftypes.NewRandB32(),
		GroupIdentity: fftypes.GroupIdentity{
			Name: "group1",
			Members: fftypes.Members{
				{Identity: "org1", Node: node1.ID},
				{Identity: "org2", Node: node2.ID},
			},
		},
	}, nil)
	mom := pm.operations.(*operationmocks.Manager)
	mom.On("AddOrReuseOperation", pm.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	err := pm.dispatchPinnedBatch(pm.ctx, &batch.DispatchState{
		Persisted: fftypes.BatchPersisted{
			BatchHeader: fftypes.BatchHeader{
				Group: groupID,
				SignerRef: fftypes.SignerRef{
					Author: "org1",
				},
			},
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: fftypes.NewUUID(),
			},
		},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestSendSubmitBlobTransferFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	localOrg := newTestOrg("localorg")
	groupID := fftypes.NewRandB32()
	node1 := newTestNode("node1", localOrg)
	node2 := newTestNode("node2", newTestOrg("remoteorg"))
	blob1 := fftypes.NewRandB32()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", pm.ctx).Return(localOrg, nil)

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByID", pm.ctx, node1.ID).Return(node1, nil).Once()
	mdi.On("GetIdentityByID", pm.ctx, node2.ID).Return(node2, nil).Once()
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(&fftypes.Group{
		Hash: fftypes.NewRandB32(),
		GroupIdentity: fftypes.GroupIdentity{
			Name: "group1",
			Members: fftypes.Members{
				{Identity: "org1", Node: node1.ID},
				{Identity: "org2", Node: node2.ID},
			},
		},
	}, nil)

	mom := pm.operations.(*operationmocks.Manager)
	mom.On("AddOrReuseOperation", pm.ctx, mock.Anything).Return(nil)
	mdi.On("GetBlobMatchingHash", pm.ctx, blob1).Return(&fftypes.Blob{
		Hash:       blob1,
		PayloadRef: "/blob/1",
	}, nil)

	mom.On("RunOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(transferBlobData)
		return op.Type == fftypes.OpTypeDataExchangeBlobSend && *data.Node.ID == *node2.ID
	})).Return(fmt.Errorf("pop"))

	err := pm.dispatchPinnedBatch(pm.ctx, &batch.DispatchState{
		Persisted: fftypes.BatchPersisted{
			BatchHeader: fftypes.BatchHeader{
				Group: groupID,
				SignerRef: fftypes.SignerRef{
					Author: "org1",
				},
			},
		},
		Payload: fftypes.BatchPayload{
			Data: fftypes.DataArray{
				{ID: fftypes.NewUUID(), Blob: &fftypes.BlobRef{Hash: blob1}},
			},
		},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestWriteTransactionSubmitBatchPinFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	localOrg := newTestOrg("localorg")
	groupID := fftypes.NewRandB32()
	node1 := newTestNode("node1", localOrg)
	node2 := newTestNode("node2", newTestOrg("remoteorg"))
	blob1 := fftypes.NewRandB32()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", pm.ctx).Return(localOrg, nil)

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByID", pm.ctx, node1.ID).Return(node1, nil).Once()
	mdi.On("GetIdentityByID", pm.ctx, node2.ID).Return(node2, nil).Once()
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(&fftypes.Group{
		Hash: fftypes.NewRandB32(),
		GroupIdentity: fftypes.GroupIdentity{
			Name: "group1",
			Members: fftypes.Members{
				{Identity: "org1", Node: node1.ID},
				{Identity: "org2", Node: node2.ID},
			},
		},
	}, nil)

	mom := pm.operations.(*operationmocks.Manager)
	mom.On("AddOrReuseOperation", pm.ctx, mock.Anything).Return(nil)
	mom.On("RunOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		if op.Type != fftypes.OpTypeDataExchangeBlobSend {
			return false
		}
		data := op.Data.(transferBlobData)
		return *data.Node.ID == *node2.ID
	})).Return(nil)
	mom.On("RunOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		if op.Type != fftypes.OpTypeDataExchangeBatchSend {
			return false
		}
		data := op.Data.(batchSendData)
		return *data.Node.ID == *node2.ID
	})).Return(nil)

	mdi.On("GetBlobMatchingHash", pm.ctx, blob1).Return(&fftypes.Blob{
		Hash:       blob1,
		PayloadRef: "/blob/1",
	}, nil)

	mbp := pm.batchpin.(*batchpinmocks.Submitter)
	mbp.On("SubmitPinnedBatch", pm.ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := pm.dispatchPinnedBatch(pm.ctx, &batch.DispatchState{
		Persisted: fftypes.BatchPersisted{
			BatchHeader: fftypes.BatchHeader{
				Group: groupID,
				SignerRef: fftypes.SignerRef{
					Author: "org1",
				},
			},
		},
		Payload: fftypes.BatchPayload{
			Data: fftypes.DataArray{
				{ID: fftypes.NewUUID(), Blob: &fftypes.BlobRef{Hash: blob1}},
			},
		},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mbp.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestTransferBlobsNotFound(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", pm.ctx, mock.Anything).Return(nil, nil)

	err := pm.transferBlobs(pm.ctx, fftypes.DataArray{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32(), Blob: &fftypes.BlobRef{Hash: fftypes.NewRandB32()}},
	}, fftypes.NewUUID(), newTestNode("node1", newTestOrg("org1")))
	assert.Regexp(t, "FF10239", err)

	mdi.AssertExpectations(t)
}

func TestTransferBlobsOpInsertFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mom := pm.operations.(*operationmocks.Manager)

	mdi.On("GetBlobMatchingHash", pm.ctx, mock.Anything).Return(&fftypes.Blob{PayloadRef: "blob/1"}, nil)
	mdx.On("TransferBLOB", pm.ctx, mock.Anything, "peer1", "blob/1").Return(nil)
	mom.On("AddOrReuseOperation", pm.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	err := pm.transferBlobs(pm.ctx, fftypes.DataArray{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32(), Blob: &fftypes.BlobRef{Hash: fftypes.NewRandB32()}},
	}, fftypes.NewUUID(), newTestNode("node1", newTestOrg("org1")))
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestStart(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("Start").Return(nil)

	err := pm.Start()
	assert.NoError(t, err)
}
