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

package data

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"

	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/dataexchange"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

type blobStore struct {
	database database.Plugin
	exchange dataexchange.Plugin
}

func (bs *blobStore) UploadBLOB(ctx context.Context, ns string, inData *fftypes.DataRefOrValue, blob *fftypes.Multipart, autoMeta bool) (*fftypes.Data, error) {

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

	hash := sha256.New()
	dxReader, dx := io.Pipe()
	storeAndHash := io.MultiWriter(hash, dx)

	var written int64
	copyDone := make(chan error, 1)
	go func() {
		var err error
		written, err = io.Copy(storeAndHash, blob.Data)
		log.L(ctx).Debugf("Upload BLOB streamed %d bytes (err=%v)", written, err)
		_ = dx.Close()
		copyDone <- err
	}()

	payloadRef, uploadHash, dxErr := bs.exchange.UploadBLOB(ctx, ns, *data.ID, dxReader)
	dxReader.Close()
	copyErr := <-copyDone
	if dxErr != nil {
		return nil, dxErr
	}
	if copyErr != nil {
		return nil, i18n.WrapError(ctx, copyErr, i18n.MsgBlobStreamingFailed)
	}

	// autoMeta will create/update JSON metadata with the upload details
	if autoMeta {
		do := data.Value.JSONObject()
		do["filename"] = blob.Filename
		do["mimetype"] = blob.Mimetype
		do["size"] = float64(written)
		data.Value, _ = json.Marshal(&do)
	}

	data.Blob = fftypes.HashResult(hash)
	if *uploadHash != *data.Blob {
		return nil, i18n.NewError(ctx, i18n.MsgDXBadHash, uploadHash, data.Blob)
	}
	_ = data.Seal(ctx)
	log.L(ctx).Infof("Uploaded BLOB %.2fkb hash=%s", float64(written)/1024, data.Hash)

	err := bs.database.RunAsGroup(ctx, func(ctx context.Context) error {
		err := bs.database.UpsertData(ctx, data, false, false)
		if err == nil {
			err = bs.database.InsertBlob(ctx, &fftypes.Blob{
				Hash:       uploadHash,
				PayloadRef: payloadRef,
				Created:    fftypes.Now(),
			})
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	return data, nil
}
