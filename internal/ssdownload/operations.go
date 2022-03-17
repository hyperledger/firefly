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

package ssdownload

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/docker/go-units"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type downloadBatchData struct {
	Namespace  string `json:"namespace"`
	PayloadRef string `json:"payloadRef"`
}

type downloadBlobData struct {
	Namespace  string        `json:"namespace"`
	DataID     *fftypes.UUID `json:"dataId"`
	PayloadRef string        `json:"payloadRef"`
}

func addDownloadBatchInputs(op *fftypes.Operation, payloadRef string) {
	op.Input = fftypes.JSONObject{
		"payloadRef": payloadRef,
	}
}

func getDownloadBatchOutputs(batchID *fftypes.UUID) fftypes.JSONObject {
	return fftypes.JSONObject{
		"batch": batchID,
	}
}

func addDownloadBlobInputs(op *fftypes.Operation, payloadRef string) {
	op.Input = fftypes.JSONObject{
		"payloadRef": payloadRef,
	}
}

func getDownloadBlobOutputs(hash *fftypes.Bytes32, size int64, dxPaylodRef string) fftypes.JSONObject {
	return fftypes.JSONObject{
		"hash":         hash,
		"size":         size,
		"dxPayloadRef": dxPaylodRef,
	}
}

func retrieveDownloadBatchInputs(op *fftypes.Operation) (string, string) {
	return op.Input.GetString("namespace"),
		op.Input.GetString("payloadRef")
}

func retrieveDownloadBlobInputs(ctx context.Context, op *fftypes.Operation) (namespace string, dataID *fftypes.UUID, payloadRef string, err error) {
	namespace = op.Input.GetString("namespace")
	dataID, err = fftypes.ParseUUID(ctx, op.Input.GetString("dataId"))
	if err != nil {
		return "", nil, "", err
	}
	payloadRef = op.Input.GetString("payloadRef")
	return
}

func (dm *downloadManager) PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error) {
	switch op.Type {

	case fftypes.OpTypeSharedStorageDownloadBatch:
		namespace, payloadRef := retrieveDownloadBatchInputs(op)
		return opDownloadBatch(op, namespace, payloadRef), nil

	case fftypes.OpTypeSharedStorageDownloadBlob:
		namespace, dataID, payloadRef, err := retrieveDownloadBlobInputs(ctx, op)
		if err != nil {
			return nil, err
		}
		return opDownloadBlob(op, namespace, dataID, payloadRef), nil

	default:
		return nil, i18n.NewError(ctx, i18n.MsgOperationNotSupported, op.Type)
	}
}

func (dm *downloadManager) RunOperation(ctx context.Context, op *fftypes.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error) {
	switch data := op.Data.(type) {
	case downloadBatchData:
		return dm.downloadBatch(ctx, data)
	case downloadBlobData:
		return dm.downloadBlob(ctx, data)
	default:
		return nil, false, i18n.NewError(ctx, i18n.MsgOperationDataIncorrect, op.Data)
	}
}

// downloadBatch retrieves a serialized batch from public storage, then persists it and drives a rewind
// on the messages included (just like the event driven when we receive data over DX).
func (dm *downloadManager) downloadBatch(ctx context.Context, data downloadBatchData) (outputs fftypes.JSONObject, complete bool, err error) {

	// Download into memory for batches
	reader, err := dm.sharedstorage.DownloadData(ctx, data.PayloadRef)
	if err != nil {
		return nil, false, i18n.WrapError(ctx, err, i18n.MsgDownloadSharedFailed, data.PayloadRef)
	}
	defer reader.Close()

	// Read from the stream up to the limit
	maxReadLimit := dm.broadcastBatchPayloadLimit + 1024
	limitedReader := io.LimitReader(reader, maxReadLimit)
	batchBytes, err := ioutil.ReadAll(limitedReader)
	if err != nil {
		return nil, false, i18n.WrapError(ctx, err, i18n.MsgDownloadSharedFailed, data.PayloadRef)
	}
	if len(batchBytes) == int(maxReadLimit) {
		return nil, false, i18n.WrapError(ctx, err, i18n.MsgDownloadBatchMaxBytes, data.PayloadRef)
	}

	// Parse and store the batch
	batchID, err := dm.callbacks.SharedStorageBatchDownloaded(data.Namespace, data.PayloadRef, batchBytes)
	if err != nil {
		return nil, false, err
	}
	return getDownloadBatchOutputs(batchID), true, nil
}

func (dm *downloadManager) downloadBlob(ctx context.Context, data downloadBlobData) (outputs fftypes.JSONObject, complete bool, err error) {

	// Stream from shared storage ...
	reader, err := dm.sharedstorage.DownloadData(ctx, data.PayloadRef)
	if err != nil {
		return nil, false, err
	}
	defer reader.Close()

	// ... to data exchange
	dxPayloadRef, hash, blobSize, err := dm.dataexchange.UploadBLOB(ctx, data.Namespace, *data.DataID, reader)
	if err != nil {
		return nil, false, i18n.WrapError(ctx, err, i18n.MsgDownloadSharedFailed, data.PayloadRef)
	}
	log.L(ctx).Infof("Transferred blob '%s' (%s) from shared storage '%s' to local data exchange '%s'", hash, units.HumanSizeWithPrecision(float64(blobSize), 2), data.PayloadRef, dxPayloadRef)

	// then callback to store metadata
	err = dm.callbacks.SharedStorageBLOBDownloaded(*hash, blobSize, dxPayloadRef)
	if err != nil {
		return nil, false, err
	}

	return getDownloadBlobOutputs(hash, blobSize, dxPayloadRef), true, nil
}

func opDownloadBatch(op *fftypes.Operation, ns string, payloadRef string) *fftypes.PreparedOperation {
	return &fftypes.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
		Data: downloadBatchData{
			Namespace:  ns,
			PayloadRef: payloadRef,
		},
	}
}

func opDownloadBlob(op *fftypes.Operation, ns string, dataID *fftypes.UUID, payloadRef string) *fftypes.PreparedOperation {
	return &fftypes.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
		Data: downloadBlobData{
			Namespace:  ns,
			DataID:     dataID,
			PayloadRef: payloadRef,
		},
	}
}
