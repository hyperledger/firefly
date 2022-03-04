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

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type transferBlobData struct {
	Node *fftypes.Identity `json:"node"`
	Blob *fftypes.Blob     `json:"blob"`
}

type batchSendData struct {
	Node      *fftypes.Identity         `json:"node"`
	Transport *fftypes.TransportWrapper `json:"transport"`
}

func addTransferBlobInputs(op *fftypes.Operation, nodeID *fftypes.UUID, blobHash *fftypes.Bytes32) {
	op.Input = fftypes.JSONObject{
		"node": nodeID.String(),
		"hash": blobHash.String(),
	}
}

func retrieveTransferBlobInputs(ctx context.Context, op *fftypes.Operation) (nodeID *fftypes.UUID, blobHash *fftypes.Bytes32, err error) {
	nodeID, err = fftypes.ParseUUID(ctx, op.Input.GetString("node"))
	if err == nil {
		blobHash, err = fftypes.ParseBytes32(ctx, op.Input.GetString("hash"))
	}
	return nodeID, blobHash, err
}

func addBatchSendInputs(op *fftypes.Operation, nodeID *fftypes.UUID, groupHash *fftypes.Bytes32, batchID *fftypes.UUID, manifest string) {
	op.Input = fftypes.JSONObject{
		"node":     nodeID.String(),
		"group":    groupHash.String(),
		"batch":    batchID.String(),
		"manifest": manifest,
	}
}

func retrieveBatchSendInputs(ctx context.Context, op *fftypes.Operation) (nodeID *fftypes.UUID, groupHash *fftypes.Bytes32, batchID *fftypes.UUID, manifest string, err error) {
	nodeID, err = fftypes.ParseUUID(ctx, op.Input.GetString("node"))
	if err == nil {
		groupHash, err = fftypes.ParseBytes32(ctx, op.Input.GetString("group"))
	}
	if err == nil {
		batchID, err = fftypes.ParseUUID(ctx, op.Input.GetString("batch"))
	}
	if err == nil {
		manifest = op.Input.GetString("manifest")
	}
	return nodeID, groupHash, batchID, manifest, err
}

func (pm *privateMessaging) PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error) {
	switch op.Type {
	case fftypes.OpTypeDataExchangeBlobSend:
		nodeID, blobHash, err := retrieveTransferBlobInputs(ctx, op)
		if err != nil {
			return nil, err
		}
		node, err := pm.database.GetIdentityByID(ctx, nodeID)
		if err != nil {
			return nil, err
		} else if node == nil {
			return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
		}
		blob, err := pm.database.GetBlobMatchingHash(ctx, blobHash)
		if err != nil {
			return nil, err
		} else if blob == nil {
			return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
		}
		return opTransferBlob(op, node, blob), nil

	case fftypes.OpTypeDataExchangeBatchSend:
		nodeID, groupHash, batchID, _, err := retrieveBatchSendInputs(ctx, op)
		if err != nil {
			return nil, err
		}
		node, err := pm.database.GetIdentityByID(ctx, nodeID)
		if err != nil {
			return nil, err
		} else if node == nil {
			return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
		}
		group, err := pm.database.GetGroupByHash(ctx, groupHash)
		if err != nil {
			return nil, err
		} else if group == nil {
			return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
		}
		batch, err := pm.database.GetBatchByID(ctx, batchID)
		if err != nil {
			return nil, err
		} else if batch == nil {
			return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
		}
		transport := &fftypes.TransportWrapper{Group: group, Batch: batch}
		return opBatchSend(op, node, transport), nil

	default:
		return nil, i18n.NewError(ctx, i18n.MsgOperationNotSupported)
	}
}

func (pm *privateMessaging) RunOperation(ctx context.Context, op *fftypes.PreparedOperation) (complete bool, err error) {
	switch data := op.Data.(type) {
	case transferBlobData:
		return false, pm.exchange.TransferBLOB(ctx, op.ID, data.Node.Profile.GetString("id"), data.Blob.PayloadRef)

	case batchSendData:
		payload, err := json.Marshal(data.Transport)
		if err != nil {
			return false, i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
		}
		return false, pm.exchange.SendMessage(ctx, op.ID, data.Node.Profile.GetString("id"), payload)

	default:
		return false, i18n.NewError(ctx, i18n.MsgOperationNotSupported)
	}
}

func opTransferBlob(op *fftypes.Operation, node *fftypes.Identity, blob *fftypes.Blob) *fftypes.PreparedOperation {
	return &fftypes.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
		Data: transferBlobData{Node: node, Blob: blob},
	}
}

func opBatchSend(op *fftypes.Operation, node *fftypes.Identity, transport *fftypes.TransportWrapper) *fftypes.PreparedOperation {
	return &fftypes.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
		Data: batchSendData{Node: node, Transport: transport},
	}
}
