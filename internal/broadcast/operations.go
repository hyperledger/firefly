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
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type uploadBatchData struct {
	BatchPersisted *fftypes.BatchPersisted `json:"batchPersisted"`
	Batch          *fftypes.Batch          `json:"batch"`
}

type uploadBlobData struct {
	Data *fftypes.Data `json:"data"`
	Blob *fftypes.Blob `json:"batch"`
}

func addUploadBatchInputs(op *fftypes.Operation, batchID *fftypes.UUID) {
	op.Input = fftypes.JSONObject{
		"id": batchID.String(),
	}
}

func getUploadBatchOutputs(payloadRef string) fftypes.JSONObject {
	return fftypes.JSONObject{
		"payloadRef": payloadRef,
	}
}

func addUploadBlobInputs(op *fftypes.Operation, dataID *fftypes.UUID) {
	op.Input = fftypes.JSONObject{
		"dataId": dataID.String(),
	}
}

func getUploadBlobOutputs(payloadRef string) fftypes.JSONObject {
	return fftypes.JSONObject{
		"payloadRef": payloadRef,
	}
}

func retrieveUploadBatchInputs(ctx context.Context, op *fftypes.Operation) (*fftypes.UUID, error) {
	return fftypes.ParseUUID(ctx, op.Input.GetString("id"))
}

func retrieveUploadBlobInputs(ctx context.Context, op *fftypes.Operation) (*fftypes.UUID, error) {
	return fftypes.ParseUUID(ctx, op.Input.GetString("dataId"))
}

func (bm *broadcastManager) PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error) {
	switch op.Type {
	case fftypes.OpTypeSharedStorageUploadBatch:
		id, err := retrieveUploadBatchInputs(ctx, op)
		if err != nil {
			return nil, err
		}
		bp, err := bm.database.GetBatchByID(ctx, id)
		if err != nil {
			return nil, err
		} else if bp == nil {
			return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
		}
		batch, err := bm.data.HydrateBatch(ctx, bp)
		if err != nil {
			return nil, err
		}
		return opUploadBatch(op, batch, bp), nil

	case fftypes.OpTypeSharedStorageUploadBlob:
		dataID, err := retrieveUploadBlobInputs(ctx, op)
		if err != nil {
			return nil, err
		}
		d, err := bm.database.GetDataByID(ctx, dataID, false)
		if err != nil {
			return nil, err
		} else if d == nil || d.Blob == nil {
			return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
		}
		blob, err := bm.database.GetBlobMatchingHash(ctx, d.Blob.Hash)
		if err != nil {
			return nil, err
		} else if blob == nil {
			return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
		}
		return opUploadBlob(op, d, blob), nil

	default:
		return nil, i18n.NewError(ctx, i18n.MsgOperationNotSupported, op.Type)
	}
}

func (bm *broadcastManager) RunOperation(ctx context.Context, op *fftypes.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error) {
	switch data := op.Data.(type) {
	case uploadBatchData:
		return bm.uploadBatch(ctx, data)
	case uploadBlobData:
		return bm.uploadBlob(ctx, data)
	default:
		return nil, false, i18n.NewError(ctx, i18n.MsgOperationDataIncorrect, op.Data)
	}
}

// uploadBatch uploads the serialized batch to public storage
func (bm *broadcastManager) uploadBatch(ctx context.Context, data uploadBatchData) (outputs fftypes.JSONObject, complete bool, err error) {
	// Serialize the full payload, which has already been sealed for us by the BatchManager
	payload, err := json.Marshal(data.Batch)
	if err != nil {
		return nil, false, i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	// Write it to IPFS to get a payload reference
	payloadRef, err := bm.sharedstorage.UploadData(ctx, bytes.NewReader(payload))
	if err != nil {
		return nil, false, err
	}
	log.L(ctx).Infof("Published batch '%s' to shared storage: '%s'", data.Batch.ID, payloadRef)

	// Update the batch to store the payloadRef
	data.BatchPersisted.PayloadRef = payloadRef
	update := database.BatchQueryFactory.NewUpdate(ctx).Set("payloadref", payloadRef)
	return getUploadBatchOutputs(payloadRef), true, bm.database.UpdateBatch(ctx, data.Batch.ID, update)
}

// uploadBlob streams a blob from the local data exchange, to public storage
func (bm *broadcastManager) uploadBlob(ctx context.Context, data uploadBlobData) (outputs fftypes.JSONObject, complete bool, err error) {

	// Stream from the local data exchange ...
	reader, err := bm.exchange.DownloadBLOB(ctx, data.Blob.PayloadRef)
	if err != nil {
		return nil, false, i18n.WrapError(ctx, err, i18n.MsgDownloadBlobFailed, data.Blob.PayloadRef)
	}
	defer reader.Close()

	// ... to the shared storage
	data.Data.Blob.Public, err = bm.sharedstorage.UploadData(ctx, reader)
	if err != nil {
		return nil, false, err
	}

	// Update the data in the DB
	err = bm.database.UpdateData(ctx, data.Data.ID, database.DataQueryFactory.NewUpdate(ctx).Set("blob.public", data.Data.Blob.Public))
	if err != nil {
		return nil, false, err
	}

	log.L(ctx).Infof("Published blob with hash '%s' for data '%s' to shared storage: '%s'", data.Data.Blob.Hash, data.Data.ID, data.Data.Blob.Public)
	return getUploadBlobOutputs(data.Data.Blob.Public), true, nil
}

func opUploadBatch(op *fftypes.Operation, batch *fftypes.Batch, batchPersisted *fftypes.BatchPersisted) *fftypes.PreparedOperation {
	return &fftypes.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
		Data: uploadBatchData{Batch: batch, BatchPersisted: batchPersisted},
	}
}

func opUploadBlob(op *fftypes.Operation, data *fftypes.Data, blob *fftypes.Blob) *fftypes.PreparedOperation {
	return &fftypes.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
		Data: uploadBlobData{Data: data, Blob: blob},
	}
}
