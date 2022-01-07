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

package data

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"

	"github.com/docker/go-units"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/publicstorage"
)

type blobStore struct {
	dm            *dataManager
	publicstorage publicstorage.Plugin
	database      database.Plugin
	exchange      dataexchange.Plugin
}

func (bs *blobStore) uploadVerifyBLOB(ctx context.Context, ns string, id *fftypes.UUID, expectedHash *fftypes.Bytes32, reader io.Reader) (hash *fftypes.Bytes32, written int64, payloadRef string, err error) {
	hashCalc := sha256.New()
	dxReader, dx := io.Pipe()
	storeAndHash := io.MultiWriter(hashCalc, dx)

	copyDone := make(chan error, 1)
	go func() {
		var err error
		written, err = io.Copy(storeAndHash, reader)
		log.L(ctx).Debugf("Upload BLOB streamed %d bytes (err=%v)", written, err)
		_ = dx.Close()
		copyDone <- err
	}()

	payloadRef, uploadHash, uploadSize, dxErr := bs.exchange.UploadBLOB(ctx, ns, *id, dxReader)
	dxReader.Close()
	copyErr := <-copyDone
	if dxErr != nil {
		return nil, -1, "", dxErr
	}
	if copyErr != nil {
		return nil, -1, "", i18n.WrapError(ctx, copyErr, i18n.MsgBlobStreamingFailed)
	}

	hash = fftypes.HashResult(hashCalc)
	log.L(ctx).Debugf("Upload BLOB size=%d hashes: calculated=%s upload=%s (expected=%v) size=%d (expected=%d)", written, hash, uploadHash, expectedHash, uploadSize, written)

	if !uploadHash.Equals(hash) {
		return nil, -1, "", i18n.NewError(ctx, i18n.MsgDXBadHash, uploadHash, hash)
	}

	if expectedHash != nil && !uploadHash.Equals(expectedHash) {
		return nil, -1, "", i18n.NewError(ctx, i18n.MsgDXBadHash, uploadHash, expectedHash)
	}
	if uploadSize > 0 && uploadSize != written {
		return nil, -1, "", i18n.NewError(ctx, i18n.MsgDXBadSize, uploadSize, written)
	}

	return hash, written, payloadRef, nil

}

func (bs *blobStore) UploadBLOB(ctx context.Context, ns string, inData *fftypes.DataRefOrValue, mpart *fftypes.Multipart, autoMeta bool) (*fftypes.Data, error) {

	data := &fftypes.Data{
		ID:        fftypes.NewUUID(),
		Namespace: ns,
		Created:   fftypes.Now(),
		Validator: inData.Validator,
		Datatype:  inData.Datatype,
		Value:     inData.Value,
	}

	data.ID = fftypes.NewUUID()
	data.Namespace = ns
	data.Created = fftypes.Now()

	hash, blobSize, payloadRef, err := bs.uploadVerifyBLOB(ctx, ns, data.ID, nil /* we don't have an expected hash for a new upload */, mpart.Data)
	if err != nil {
		return nil, err
	}
	data.Blob = &fftypes.BlobRef{Hash: hash}

	// autoMeta will create/update JSON metadata with the upload details
	if autoMeta {
		do := data.Value.JSONObject()
		do["filename"] = mpart.Filename
		do["mimetype"] = mpart.Mimetype
		b, _ := json.Marshal(&do)
		data.Value = fftypes.JSONAnyPtrBytes(b)
	}
	if data.Validator == "" {
		data.Validator = fftypes.ValidatorTypeJSON
	}

	blob := &fftypes.Blob{
		Hash:       hash,
		Size:       blobSize,
		PayloadRef: payloadRef,
		Created:    fftypes.Now(),
	}

	err = bs.dm.checkValidation(ctx, ns, data.Validator, data.Datatype, data.Value)
	if err == nil {
		err = data.Seal(ctx, blob)
	}
	if err != nil {
		return nil, err
	}
	log.L(ctx).Infof("Uploaded BLOB blobhash=%s hash=%s (%s)", data.Blob.Hash, data.Hash, units.HumanSizeWithPrecision(float64(blobSize), 2))

	err = bs.database.RunAsGroup(ctx, func(ctx context.Context) error {
		err := bs.database.UpsertData(ctx, data, database.UpsertOptimizationNew)
		if err == nil {
			err = bs.database.InsertBlob(ctx, blob)
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (bs *blobStore) CopyBlobPStoDX(ctx context.Context, data *fftypes.Data) (blob *fftypes.Blob, err error) {

	reader, err := bs.publicstorage.RetrieveData(ctx, data.Blob.Public)
	if err != nil {
		return nil, err
	}
	if reader == nil {
		log.L(ctx).Infof("Blob '%s' not found in public storage", data.Blob.Public)
		return nil, nil
	}
	defer reader.Close()

	hash, blobSize, payloadRef, err := bs.uploadVerifyBLOB(ctx, data.Namespace, data.ID, data.Blob.Hash, reader)
	if err != nil {
		return nil, err
	}
	log.L(ctx).Infof("Transferred blob '%s' (%s) from public storage '%s' to local data exchange '%s'", hash, units.HumanSizeWithPrecision(float64(blobSize), 2), data.Blob.Public, payloadRef)

	blob = &fftypes.Blob{
		Hash:       hash,
		Size:       blobSize,
		PayloadRef: payloadRef,
		Created:    fftypes.Now(),
	}
	err = bs.database.InsertBlob(ctx, blob)
	if err != nil {
		return nil, err
	}
	return blob, nil
}

func (bs *blobStore) DownloadBLOB(ctx context.Context, ns, dataID string) (*fftypes.Blob, io.ReadCloser, error) {

	if err := fftypes.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
		return nil, nil, err
	}
	id, err := fftypes.ParseUUID(ctx, dataID)
	if err != nil {
		return nil, nil, err
	}

	data, err := bs.database.GetDataByID(ctx, id, false)
	if err != nil {
		return nil, nil, err
	}
	if data == nil || data.Namespace != ns {
		return nil, nil, i18n.NewError(ctx, i18n.Msg404NoResult)
	}
	if data.Blob == nil || data.Blob.Hash == nil {
		return nil, nil, i18n.NewError(ctx, i18n.MsgDataDoesNotHaveBlob)
	}

	blob, err := bs.database.GetBlobMatchingHash(ctx, data.Blob.Hash)
	if err != nil {
		return nil, nil, err
	}
	if blob == nil {
		return nil, nil, i18n.NewError(ctx, i18n.MsgBlobNotFound, data.Blob.Hash)
	}

	reader, err := bs.exchange.DownloadBLOB(ctx, blob.PayloadRef)
	return blob, reader, err
}
