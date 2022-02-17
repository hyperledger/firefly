// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in comdiliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or imdilied.
// See the License for the specific language governing permissions and
// limitations under the License.

package privatemessaging

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPrepareAndRunTransferBlob(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	op := &fftypes.Operation{
		Type: fftypes.OpTypeDataExchangeBlobSend,
		ID:   fftypes.NewUUID(),
	}
	node := &fftypes.Node{
		ID: fftypes.NewUUID(),
		DX: fftypes.DXInfo{
			Peer: "peer1",
		},
	}
	blob := &fftypes.Blob{
		Hash:       fftypes.NewRandB32(),
		PayloadRef: "payload",
	}
	addTransferBlobInputs(op, node.ID, blob.Hash)

	mdi := pm.database.(*databasemocks.Plugin)
	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mdi.On("GetNodeByID", context.Background(), node.ID).Return(node, nil)
	mdi.On("GetBlobMatchingHash", context.Background(), blob.Hash).Return(blob, nil)
	mdx.On("TransferBLOB", context.Background(), op.ID, "peer1", "payload").Return(nil)

	po, err := pm.PrepareOperation(context.Background(), op)
	assert.NoError(t, err)
	assert.Equal(t, node, po.Data.(transferBlobData).Node)
	assert.Equal(t, blob, po.Data.(transferBlobData).Blob)

	complete, err := pm.RunOperation(context.Background(), po)

	assert.False(t, complete)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestPrepareAndRunBatchSend(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	op := &fftypes.Operation{
		Type: fftypes.OpTypeDataExchangeBatchSend,
		ID:   fftypes.NewUUID(),
	}
	node := &fftypes.Node{
		ID: fftypes.NewUUID(),
		DX: fftypes.DXInfo{
			Peer: "peer1",
		},
	}
	group := &fftypes.Group{
		Hash: fftypes.NewRandB32(),
	}
	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
	}
	addBatchSendInputs(op, node.ID, group.Hash, batch.ID, "manifest-info")

	mdi := pm.database.(*databasemocks.Plugin)
	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mdi.On("GetNodeByID", context.Background(), node.ID).Return(node, nil)
	mdi.On("GetGroupByHash", context.Background(), group.Hash).Return(group, nil)
	mdi.On("GetBatchByID", context.Background(), batch.ID).Return(batch, nil)
	mdx.On("SendMessage", context.Background(), op.ID, "peer1", mock.Anything).Return(nil)

	po, err := pm.PrepareOperation(context.Background(), op)
	assert.NoError(t, err)
	assert.Equal(t, node, po.Data.(batchSendData).Node)
	assert.Equal(t, group, po.Data.(batchSendData).Transport.Group)
	assert.Equal(t, batch, po.Data.(batchSendData).Transport.Batch)

	complete, err := pm.RunOperation(context.Background(), po)

	assert.False(t, complete)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestPrepareOperationNotSupported(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	po, err := pm.PrepareOperation(context.Background(), &fftypes.Operation{})

	assert.Nil(t, po)
	assert.Regexp(t, "FF10348", err)
}

func TestPrepareOperationBlobSendBadInput(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	op := &fftypes.Operation{
		Type:  fftypes.OpTypeDataExchangeBlobSend,
		Input: fftypes.JSONObject{"node": "bad"},
	}

	_, err := pm.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10142", err)
}

func TestPrepareOperationBlobSendNodeFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	nodeID := fftypes.NewUUID()
	blobHash := fftypes.NewRandB32()
	op := &fftypes.Operation{
		Type: fftypes.OpTypeDataExchangeBlobSend,
		Input: fftypes.JSONObject{
			"node": nodeID.String(),
			"blob": blobHash.String(),
		},
	}

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNodeByID", context.Background(), nodeID).Return(nil, fmt.Errorf("pop"))

	_, err := pm.PrepareOperation(context.Background(), op)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestPrepareOperationBlobSendNodeNotFound(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	nodeID := fftypes.NewUUID()
	blobHash := fftypes.NewRandB32()
	op := &fftypes.Operation{
		Type: fftypes.OpTypeDataExchangeBlobSend,
		Input: fftypes.JSONObject{
			"node": nodeID.String(),
			"blob": blobHash.String(),
		},
	}

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNodeByID", context.Background(), nodeID).Return(nil, nil)

	_, err := pm.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10109", err)

	mdi.AssertExpectations(t)
}

func TestPrepareOperationBlobSendBlobFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	blobHash := fftypes.NewRandB32()
	node := &fftypes.Node{
		ID: fftypes.NewUUID(),
		DX: fftypes.DXInfo{
			Peer: "peer1",
		},
	}
	op := &fftypes.Operation{
		Type: fftypes.OpTypeDataExchangeBlobSend,
		Input: fftypes.JSONObject{
			"node": node.ID.String(),
			"blob": blobHash.String(),
		},
	}

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNodeByID", context.Background(), node.ID).Return(node, nil)
	mdi.On("GetBlobMatchingHash", context.Background(), blobHash).Return(nil, fmt.Errorf("pop"))

	_, err := pm.PrepareOperation(context.Background(), op)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestPrepareOperationBlobSendBlobNotFound(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	blobHash := fftypes.NewRandB32()
	node := &fftypes.Node{
		ID: fftypes.NewUUID(),
		DX: fftypes.DXInfo{
			Peer: "peer1",
		},
	}
	op := &fftypes.Operation{
		Type: fftypes.OpTypeDataExchangeBlobSend,
		Input: fftypes.JSONObject{
			"node": node.ID.String(),
			"blob": blobHash.String(),
		},
	}

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNodeByID", context.Background(), node.ID).Return(node, nil)
	mdi.On("GetBlobMatchingHash", context.Background(), blobHash).Return(nil, nil)

	_, err := pm.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10109", err)

	mdi.AssertExpectations(t)
}

func TestPrepareOperationBatchSendBadInput(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	op := &fftypes.Operation{
		Type:  fftypes.OpTypeDataExchangeBatchSend,
		Input: fftypes.JSONObject{"node": "bad"},
	}

	_, err := pm.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10142", err)
}

func TestPrepareOperationBatchSendNodeFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	nodeID := fftypes.NewUUID()
	groupHash := fftypes.NewRandB32()
	batchID := fftypes.NewUUID()
	op := &fftypes.Operation{
		Type: fftypes.OpTypeDataExchangeBatchSend,
		Input: fftypes.JSONObject{
			"node":  nodeID.String(),
			"group": groupHash.String(),
			"batch": batchID.String(),
		},
	}

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNodeByID", context.Background(), nodeID).Return(nil, fmt.Errorf("pop"))

	_, err := pm.PrepareOperation(context.Background(), op)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestPrepareOperationBatchSendNodeNotFound(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	nodeID := fftypes.NewUUID()
	groupHash := fftypes.NewRandB32()
	batchID := fftypes.NewUUID()
	op := &fftypes.Operation{
		Type: fftypes.OpTypeDataExchangeBatchSend,
		Input: fftypes.JSONObject{
			"node":  nodeID.String(),
			"group": groupHash.String(),
			"batch": batchID.String(),
		},
	}

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNodeByID", context.Background(), nodeID).Return(nil, nil)

	_, err := pm.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10109", err)

	mdi.AssertExpectations(t)
}

func TestPrepareOperationBatchSendGroupFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	groupHash := fftypes.NewRandB32()
	batchID := fftypes.NewUUID()
	node := &fftypes.Node{
		ID: fftypes.NewUUID(),
	}
	op := &fftypes.Operation{
		Type: fftypes.OpTypeDataExchangeBatchSend,
		Input: fftypes.JSONObject{
			"node":  node.ID.String(),
			"group": groupHash.String(),
			"batch": batchID.String(),
		},
	}

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNodeByID", context.Background(), node.ID).Return(node, nil)
	mdi.On("GetGroupByHash", context.Background(), groupHash).Return(nil, fmt.Errorf("pop"))

	_, err := pm.PrepareOperation(context.Background(), op)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestPrepareOperationBatchSendGroupNotFound(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	groupHash := fftypes.NewRandB32()
	batchID := fftypes.NewUUID()
	node := &fftypes.Node{
		ID: fftypes.NewUUID(),
	}
	op := &fftypes.Operation{
		Type: fftypes.OpTypeDataExchangeBatchSend,
		Input: fftypes.JSONObject{
			"node":  node.ID.String(),
			"group": groupHash.String(),
			"batch": batchID.String(),
		},
	}

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNodeByID", context.Background(), node.ID).Return(node, nil)
	mdi.On("GetGroupByHash", context.Background(), groupHash).Return(nil, nil)

	_, err := pm.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10109", err)

	mdi.AssertExpectations(t)
}

func TestPrepareOperationBatchSendBatchFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	batchID := fftypes.NewUUID()
	node := &fftypes.Node{
		ID: fftypes.NewUUID(),
	}
	group := &fftypes.Group{
		Hash: fftypes.NewRandB32(),
	}
	op := &fftypes.Operation{
		Type: fftypes.OpTypeDataExchangeBatchSend,
		Input: fftypes.JSONObject{
			"node":  node.ID.String(),
			"group": group.Hash.String(),
			"batch": batchID.String(),
		},
	}

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNodeByID", context.Background(), node.ID).Return(node, nil)
	mdi.On("GetGroupByHash", context.Background(), group.Hash).Return(group, nil)
	mdi.On("GetBatchByID", context.Background(), batchID).Return(nil, fmt.Errorf("pop"))

	_, err := pm.PrepareOperation(context.Background(), op)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestPrepareOperationBatchSendBatchNotFound(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	batchID := fftypes.NewUUID()
	node := &fftypes.Node{
		ID: fftypes.NewUUID(),
	}
	group := &fftypes.Group{
		Hash: fftypes.NewRandB32(),
	}
	op := &fftypes.Operation{
		Type: fftypes.OpTypeDataExchangeBatchSend,
		Input: fftypes.JSONObject{
			"node":  node.ID.String(),
			"group": group.Hash.String(),
			"batch": batchID.String(),
		},
	}

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNodeByID", context.Background(), node.ID).Return(node, nil)
	mdi.On("GetGroupByHash", context.Background(), group.Hash).Return(group, nil)
	mdi.On("GetBatchByID", context.Background(), batchID).Return(nil, nil)

	_, err := pm.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10109", err)

	mdi.AssertExpectations(t)
}

func TestRunOperationNotSupported(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	complete, err := pm.RunOperation(context.Background(), &fftypes.PreparedOperation{})

	assert.False(t, complete)
	assert.Regexp(t, "FF10348", err)
}

func TestRunOperationBatchSendInvalidData(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	op := &fftypes.Operation{}
	node := &fftypes.Node{
		ID: fftypes.NewUUID(),
	}
	transport := &fftypes.TransportWrapper{
		Group: &fftypes.Group{},
		Batch: &fftypes.Batch{
			Payload: fftypes.BatchPayload{
				Data: []*fftypes.Data{
					{Value: fftypes.JSONAnyPtr(`!json`)},
				},
			},
		},
	}

	complete, err := pm.RunOperation(context.Background(), opBatchSend(op, node, transport))

	assert.False(t, complete)
	assert.Regexp(t, "FF10137", err)
}
