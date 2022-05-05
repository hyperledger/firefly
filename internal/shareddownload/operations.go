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

package shareddownload

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/docker/go-units"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/pkg/core"
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

func addDownloadBatchInputs(op *core.Operation, ns, payloadRef string) {
	op.Input = fftypes.JSONObject{
		"namespace":  ns,
		"payloadRef": payloadRef,
	}
}

func getDownloadBatchOutputs(batchID *fftypes.UUID) fftypes.JSONObject {
	return fftypes.JSONObject{
		"batch": batchID,
	}
}

func addDownloadBlobInputs(op *core.Operation, ns string, dataID *fftypes.UUID, payloadRef string) {
	op.Input = fftypes.JSONObject{
		"namespace":  ns,
		"dataId":     dataID.String(),
		"payloadRef": payloadRef,
	}
}

func getDownloadBlobOutputs(hash *fftypes.Bytes32, size int64, dxPayloadRef string) fftypes.JSONObject {
	return fftypes.JSONObject{
		"hash":         hash,
		"size":         size,
		"dxPayloadRef": dxPayloadRef,
	}
}

func retrieveDownloadBatchInputs(op *core.Operation) (string, string) {
	return op.Input.GetString("namespace"),
		op.Input.GetString("payloadRef")
}

func retrieveDownloadBlobInputs(ctx context.Context, op *core.Operation) (namespace string, dataID *fftypes.UUID, payloadRef string, err error) {
	namespace = op.Input.GetString("namespace")
	dataID, err = fftypes.ParseUUID(ctx, op.Input.GetString("dataId"))
	if err != nil {
		return "", nil, "", err
	}
	payloadRef = op.Input.GetString("payloadRef")
	return
}

func (dm *downloadManager) PrepareOperation(ctx context.Context, op *core.Operation) (*core.PreparedOperation, error) {
	switch op.Type {

	case core.OpTypeSharedStorageDownloadBatch:
		namespace, payloadRef := retrieveDownloadBatchInputs(op)
		return opDownloadBatch(op, namespace, payloadRef), nil

	case core.OpTypeSharedStorageDownloadBlob:
		namespace, dataID, payloadRef, err := retrieveDownloadBlobInputs(ctx, op)
		if err != nil {
			return nil, err
		}
		return opDownloadBlob(op, namespace, dataID, payloadRef), nil

	default:
		return nil, i18n.NewError(ctx, coremsgs.MsgOperationNotSupported, op.Type)
	}
}

func (dm *downloadManager) RunOperation(ctx context.Context, op *core.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error) {
	switch data := op.Data.(type) {
	case downloadBatchData:
		return dm.downloadBatch(ctx, data)
	case downloadBlobData:
		return dm.downloadBlob(ctx, data)
	default:
		return nil, false, i18n.NewError(ctx, coremsgs.MsgOperationDataIncorrect, op.Data)
	}
}

// downloadBatch retrieves a serialized batch from public storage, then persists it and drives a rewind
// on the messages included (just like the event driven when we receive data over DX).
func (dm *downloadManager) downloadBatch(ctx context.Context, data downloadBatchData) (outputs fftypes.JSONObject, complete bool, err error) {

	// Download into memory for batches
	reader, err := dm.sharedstorage.DownloadData(ctx, data.PayloadRef)
	if err != nil {
		return nil, false, i18n.WrapError(ctx, err, coremsgs.MsgDownloadSharedFailed, data.PayloadRef)
	}
	defer reader.Close()

	// Read from the stream up to the limit
	maxReadLimit := dm.broadcastBatchPayloadLimit + 1024
	limitedReader := io.LimitReader(reader, maxReadLimit)
	batchBytes, err := ioutil.ReadAll(limitedReader)
	if err != nil {
		return nil, false, i18n.WrapError(ctx, err, coremsgs.MsgDownloadSharedFailed, data.PayloadRef)
	}
	if len(batchBytes) == int(maxReadLimit) {
		return nil, false, i18n.WrapError(ctx, err, coremsgs.MsgDownloadBatchMaxBytes, data.PayloadRef)
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
	dxPayloadRef, hash, blobSize, err := dm.dataexchange.UploadBlob(ctx, data.Namespace, *data.DataID, reader)
	if err != nil {
		return nil, false, i18n.WrapError(ctx, err, coremsgs.MsgDownloadSharedFailed, data.PayloadRef)
	}
	log.L(ctx).Infof("Transferred blob '%s' (%s) from shared storage '%s' to local data exchange '%s'", hash, units.HumanSizeWithPrecision(float64(blobSize), 2), data.PayloadRef, dxPayloadRef)

	// then callback to store metadata
	dm.callbacks.SharedStorageBlobDownloaded(*hash, blobSize, dxPayloadRef)

	return getDownloadBlobOutputs(hash, blobSize, dxPayloadRef), true, nil
}

func (dm *downloadManager) OnOperationUpdate(ctx context.Context, op *core.Operation, update *operations.OperationUpdate) error {
	return nil
}

func opDownloadBatch(op *core.Operation, ns string, payloadRef string) *core.PreparedOperation {
	return &core.PreparedOperation{
		ID:        op.ID,
		Namespace: op.Namespace,
		Type:      op.Type,
		Data: downloadBatchData{
			Namespace:  ns,
			PayloadRef: payloadRef,
		},
	}
}

func opDownloadBlob(op *core.Operation, ns string, dataID *fftypes.UUID, payloadRef string) *core.PreparedOperation {
	return &core.PreparedOperation{
		ID:        op.ID,
		Namespace: op.Namespace,
		Type:      op.Type,
		Data: downloadBlobData{
			Namespace:  ns,
			DataID:     dataID,
			PayloadRef: payloadRef,
		},
	}
}
