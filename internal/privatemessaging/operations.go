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

package privatemessaging

import (
	"context"
	"encoding/json"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
)

type transferBlobData struct {
	Node *core.Identity `json:"node"`
	Blob *core.Blob     `json:"blob"`
}

type batchSendData struct {
	Node      *core.Identity         `json:"node"`
	Transport *core.TransportWrapper `json:"transport"`
}

func addTransferBlobInputs(op *core.Operation, nodeID *fftypes.UUID, blobHash *fftypes.Bytes32) {
	op.Input = fftypes.JSONObject{
		"node": nodeID.String(),
		"hash": blobHash.String(),
	}
}

func retrieveSendBlobInputs(ctx context.Context, op *core.Operation) (nodeID *fftypes.UUID, blobHash *fftypes.Bytes32, err error) {
	nodeID, err = fftypes.ParseUUID(ctx, op.Input.GetString("node"))
	if err == nil {
		blobHash, err = fftypes.ParseBytes32(ctx, op.Input.GetString("hash"))
	}
	return nodeID, blobHash, err
}

func addBatchSendInputs(op *core.Operation, nodeID *fftypes.UUID, groupHash *fftypes.Bytes32, batchID *fftypes.UUID) {
	op.Input = fftypes.JSONObject{
		"node":  nodeID.String(),
		"group": groupHash.String(),
		"batch": batchID.String(),
	}
}

func retrieveBatchSendInputs(ctx context.Context, op *core.Operation) (nodeID *fftypes.UUID, groupHash *fftypes.Bytes32, batchID *fftypes.UUID, err error) {
	nodeID, err = fftypes.ParseUUID(ctx, op.Input.GetString("node"))
	if err == nil {
		groupHash, err = fftypes.ParseBytes32(ctx, op.Input.GetString("group"))
	}
	if err == nil {
		batchID, err = fftypes.ParseUUID(ctx, op.Input.GetString("batch"))
	}
	return nodeID, groupHash, batchID, err
}

func (pm *privateMessaging) PrepareOperation(ctx context.Context, op *core.Operation) (*core.PreparedOperation, error) {
	switch op.Type {
	case core.OpTypeDataExchangeSendBlob:
		nodeID, blobHash, err := retrieveSendBlobInputs(ctx, op)
		if err != nil {
			return nil, err
		}
		node, err := pm.identity.CachedIdentityLookupByID(ctx, nodeID)
		if err != nil {
			return nil, err
		} else if node == nil {
			return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
		}
		blob, err := pm.database.GetBlobMatchingHash(ctx, blobHash)
		if err != nil {
			return nil, err
		} else if blob == nil {
			return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
		}
		return opSendBlob(op, node, blob), nil

	case core.OpTypeDataExchangeSendBatch:
		nodeID, groupHash, batchID, err := retrieveBatchSendInputs(ctx, op)
		if err != nil {
			return nil, err
		}
		node, err := pm.identity.CachedIdentityLookupByID(ctx, nodeID)
		if err != nil {
			return nil, err
		} else if node == nil {
			return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
		}
		group, err := pm.database.GetGroupByHash(ctx, pm.namespace.Name, groupHash)
		if err != nil {
			return nil, err
		} else if group == nil {
			return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
		}
		bp, err := pm.database.GetBatchByID(ctx, pm.namespace.Name, batchID)
		if err != nil {
			return nil, err
		} else if bp == nil {
			return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
		}
		batch, err := pm.data.HydrateBatch(ctx, bp)
		if err != nil {
			return nil, err
		}
		transport := &core.TransportWrapper{Group: group, Batch: batch}
		return opSendBatch(op, node, transport), nil

	default:
		return nil, i18n.NewError(ctx, coremsgs.MsgOperationNotSupported, op.Type)
	}
}

func (pm *privateMessaging) RunOperation(ctx context.Context, op *core.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error) {
	switch data := op.Data.(type) {
	case transferBlobData:
		localNode, err := pm.identity.GetLocalNode(ctx)
		if err != nil {
			return nil, false, err
		}
		return nil, false, pm.exchange.TransferBlob(ctx, op.NamespacedIDString(), data.Node.Profile, localNode.Profile, data.Blob.PayloadRef)

	case batchSendData:
		localNode, err := pm.identity.GetLocalNode(ctx)
		if err != nil {
			return nil, false, err
		}

		payload, err := json.Marshal(data.Transport)
		if err != nil {
			return nil, false, i18n.WrapError(ctx, err, coremsgs.MsgSerializationFailed)
		}
		return nil, false, pm.exchange.SendMessage(ctx, op.NamespacedIDString(), data.Node.Profile, localNode.Profile, payload)

	default:
		return nil, false, i18n.NewError(ctx, coremsgs.MsgOperationDataIncorrect, op.Data)
	}
}

func (pm *privateMessaging) OnOperationUpdate(ctx context.Context, op *core.Operation, update *core.OperationUpdate) error {
	return nil
}

func opSendBlob(op *core.Operation, node *core.Identity, blob *core.Blob) *core.PreparedOperation {
	return &core.PreparedOperation{
		ID:        op.ID,
		Namespace: op.Namespace,
		Plugin:    op.Plugin,
		Type:      op.Type,
		Data:      transferBlobData{Node: node, Blob: blob},
	}
}

func opSendBatch(op *core.Operation, node *core.Identity, transport *core.TransportWrapper) *core.PreparedOperation {
	return &core.PreparedOperation{
		ID:        op.ID,
		Namespace: op.Namespace,
		Plugin:    op.Plugin,
		Type:      op.Type,
		Data:      batchSendData{Node: node, Transport: transport},
	}
}
