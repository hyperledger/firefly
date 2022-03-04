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

package broadcast

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type batchBroadcastData struct {
	Batch *fftypes.Batch `json:"batch"`
}

func addBatchBroadcastInputs(op *fftypes.Operation, batchID *fftypes.UUID) {
	op.Input = fftypes.JSONObject{
		"id": batchID.String(),
	}
}

func retrieveBatchBroadcastInputs(ctx context.Context, op *fftypes.Operation) (batchID *fftypes.UUID, err error) {
	return fftypes.ParseUUID(ctx, op.Input.GetString("id"))
}

func (bm *broadcastManager) PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error) {
	switch op.Type {
	case fftypes.OpTypeSharedStorageBatchBroadcast:
		id, err := retrieveBatchBroadcastInputs(ctx, op)
		if err != nil {
			return nil, err
		}
		bp, err := bm.database.GetBatchByID(ctx, id)
		if err != nil {
			return nil, err
		} else if bp == nil {
			return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
		}
		batch, err := bm.data.HydrateBatch(ctx, bp, true)
		if err != nil {
			return nil, err
		}
		return opBatchBroadcast(op, batch), nil

	default:
		return nil, i18n.NewError(ctx, i18n.MsgOperationNotSupported)
	}
}

func (bm *broadcastManager) RunOperation(ctx context.Context, op *fftypes.PreparedOperation) (complete bool, err error) {
	switch data := op.Data.(type) {
	case batchBroadcastData:
		// Serialize the full payload, which has already been sealed for us by the BatchManager
		payload, err := json.Marshal(data.Batch)
		if err != nil {
			return false, i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
		}

		// Write it to IPFS to get a payload reference
		payloadRef, err := bm.sharedstorage.PublishData(ctx, bytes.NewReader(payload))
		if err != nil {
			return false, err
		}

		// Update the batch to store the payloadRef
		data.Batch.PayloadRef = payloadRef
		update := database.BatchQueryFactory.NewUpdate(ctx).Set("payloadref", payloadRef)
		return true, bm.database.UpdateBatch(ctx, data.Batch.ID, update)

	default:
		return false, i18n.NewError(ctx, i18n.MsgOperationNotSupported)
	}
}

func opBatchBroadcast(op *fftypes.Operation, batch *fftypes.Batch) *fftypes.PreparedOperation {
	return &fftypes.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
		Data: batchBroadcastData{Batch: batch},
	}
}
