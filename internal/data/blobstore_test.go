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
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"testing"
	"testing/iotest"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUploadBlobOk(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	b := make([]byte, 10000+int(rand.Float32()*10000))
	for i := 0; i < len(b); i++ {
		b[i] = 'a' + byte(rand.Int()%26)
	}

	mdi := dm.database.(*databasemocks.Plugin)
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}
	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationNew).Return(nil)
	mdi.On("InsertBlob", mock.Anything, mock.Anything).Return(nil)

	dxID := make(chan fftypes.UUID, 1)
	mdx := dm.exchange.(*dataexchangemocks.Plugin)
	dxUpload := mdx.On("UploadBlob", ctx, "ns1", mock.Anything, mock.Anything)
	dxUpload.RunFn = func(a mock.Arguments) {
		readBytes, err := ioutil.ReadAll(a[3].(io.Reader))
		assert.Nil(t, err)
		assert.Equal(t, b, readBytes)
		uuid := a[2].(fftypes.UUID)
		dxID <- uuid
		var hash fftypes.Bytes32 = sha256.Sum256(b)
		dxUpload.ReturnArguments = mock.Arguments{fmt.Sprintf("ns1/%s", uuid), &hash, int64(len(b)), err}
	}

	data, err := dm.UploadBlob(ctx, "ns1", &core.DataRefOrValue{}, &core.Multipart{Data: bytes.NewReader(b)}, false)
	assert.NoError(t, err)

	// Check the hashes and other details of the data
	assert.Equal(t, [32]byte(sha256.Sum256(b)), [32]byte(*data.Hash))
	assert.Equal(t, <-dxID, *data.ID)
	assert.Equal(t, core.ValidatorTypeJSON, data.Validator)
	assert.Nil(t, data.Datatype)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)

}

func TestUploadBlobAutoMetaOk(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	mdi := dm.database.(*databasemocks.Plugin)
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}
	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationNew).Return(nil)
	mdi.On("InsertBlob", mock.Anything, mock.Anything).Return(nil)

	dxID := make(chan fftypes.UUID, 1)
	mdx := dm.exchange.(*dataexchangemocks.Plugin)
	dxUpload := mdx.On("UploadBlob", ctx, "ns1", mock.Anything, mock.Anything)
	dxUpload.RunFn = func(a mock.Arguments) {
		readBytes, err := ioutil.ReadAll(a[3].(io.Reader))
		assert.Nil(t, err)
		uuid := a[2].(fftypes.UUID)
		dxID <- uuid
		var hash fftypes.Bytes32 = sha256.Sum256(readBytes)
		dxUpload.ReturnArguments = mock.Arguments{fmt.Sprintf("ns1/%s", uuid), &hash, int64(len(readBytes)), err}
	}

	data, err := dm.UploadBlob(ctx, "ns1", &core.DataRefOrValue{
		Value: fftypes.JSONAnyPtr(`{"custom": "value1"}`),
	}, &core.Multipart{
		Data:     bytes.NewReader([]byte(`hello`)),
		Filename: "myfile.csv",
		Mimetype: "text/csv",
	}, true)
	assert.NoError(t, err)
	assert.Equal(t, "myfile.csv", data.Value.JSONObject().GetString("filename"))
	assert.Equal(t, "text/csv", data.Value.JSONObject().GetString("mimetype"))
	assert.Equal(t, "value1", data.Value.JSONObject().GetString("custom"))
	assert.NotZero(t, data.Blob.Size)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)

}

func TestUploadBlobBadValidator(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	dxID := make(chan fftypes.UUID, 1)
	mdx := dm.exchange.(*dataexchangemocks.Plugin)
	dxUpload := mdx.On("UploadBlob", ctx, "ns1", mock.Anything, mock.Anything)
	dxUpload.RunFn = func(a mock.Arguments) {
		readBytes, err := ioutil.ReadAll(a[3].(io.Reader))
		assert.Nil(t, err)
		uuid := a[2].(fftypes.UUID)
		dxID <- uuid
		var hash fftypes.Bytes32 = sha256.Sum256(readBytes)
		dxUpload.ReturnArguments = mock.Arguments{fmt.Sprintf("ns1/%s", uuid), &hash, int64(len(readBytes)), err}
	}

	_, err := dm.UploadBlob(ctx, "ns1", &core.DataRefOrValue{
		Value:     fftypes.JSONAnyPtr(`{"custom": "value1"}`),
		Validator: "wrong",
	}, &core.Multipart{
		Data:     bytes.NewReader([]byte(`hello`)),
		Filename: "myfile.csv",
		Mimetype: "text/csv",
	}, true)
	assert.Regexp(t, "FF00108", err)

	mdx.AssertExpectations(t)

}

func TestUploadBlobReadFail(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	mdx := dm.exchange.(*dataexchangemocks.Plugin)
	dxUpload := mdx.On("UploadBlob", ctx, "ns1", mock.Anything, mock.Anything).Return("", fftypes.NewRandB32(), int64(0), nil)
	dxUpload.RunFn = func(a mock.Arguments) {
		_, err := ioutil.ReadAll(a[3].(io.Reader))
		assert.NoError(t, err)
	}

	_, err := dm.UploadBlob(ctx, "ns1", &core.DataRefOrValue{}, &core.Multipart{Data: iotest.ErrReader(fmt.Errorf("pop"))}, false)
	assert.Regexp(t, "FF10217.*pop", err)

}

func TestUploadBlobWriteFailDoesNotRead(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	mdx := dm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("UploadBlob", ctx, "ns1", mock.Anything, mock.Anything).Return("", nil, int64(0), fmt.Errorf("pop"))

	_, err := dm.UploadBlob(ctx, "ns1", &core.DataRefOrValue{}, &core.Multipart{Data: bytes.NewReader([]byte(`any old data`))}, false)
	assert.Regexp(t, "pop", err)

}

func TestUploadBlobHashMismatchCalculated(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	b := []byte(`any old data`)

	mdx := dm.exchange.(*dataexchangemocks.Plugin)
	dxUpload := mdx.On("UploadBlob", ctx, "ns1", mock.Anything, mock.Anything).Return("", fftypes.NewRandB32(), int64(12345), nil)
	dxUpload.RunFn = func(a mock.Arguments) {
		_, err := ioutil.ReadAll(a[3].(io.Reader))
		assert.Nil(t, err)
	}

	_, err := dm.UploadBlob(ctx, "ns1", &core.DataRefOrValue{}, &core.Multipart{Data: bytes.NewReader([]byte(b))}, false)
	assert.Regexp(t, "FF10238", err)

}

func TestUploadBlobSizeMismatch(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	b := []byte(`any old data`)
	var hash fftypes.Bytes32 = sha256.Sum256(b)

	mdx := dm.exchange.(*dataexchangemocks.Plugin)
	dxUpload := mdx.On("UploadBlob", ctx, "ns1", mock.Anything, mock.Anything).Return("", &hash, int64(12345), nil)
	dxUpload.RunFn = func(a mock.Arguments) {
		_, err := ioutil.ReadAll(a[3].(io.Reader))
		assert.Nil(t, err)
	}

	_, err := dm.UploadBlob(ctx, "ns1", &core.DataRefOrValue{}, &core.Multipart{Data: bytes.NewReader([]byte(b))}, false)
	assert.Regexp(t, "FF10323", err)

}

func TestUploadBlobUpsertFail(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	b := []byte(`any old data`)
	var hash fftypes.Bytes32 = sha256.Sum256(b)

	mdx := dm.exchange.(*dataexchangemocks.Plugin)
	dxUpload := mdx.On("UploadBlob", ctx, "ns1", mock.Anything, mock.Anything).Return("", &hash, int64(len(b)), nil)
	dxUpload.RunFn = func(a mock.Arguments) {
		_, err := ioutil.ReadAll(a[3].(io.Reader))
		assert.Nil(t, err)
	}
	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("RunAsGroup", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := dm.UploadBlob(ctx, "ns1", &core.DataRefOrValue{}, &core.Multipart{Data: bytes.NewReader([]byte(b))}, false)
	assert.Regexp(t, "pop", err)

}

func TestDownloadBlobOk(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	blobHash := fftypes.NewRandB32()
	dataID := fftypes.NewUUID()

	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetDataByID", ctx, dataID, false).Return(&core.Data{
		ID:        dataID,
		Namespace: "ns1",
		Blob: &core.BlobRef{
			Hash: blobHash,
		},
	}, nil)
	mdi.On("GetBlobMatchingHash", ctx, blobHash).Return(&core.Blob{
		Hash:       blobHash,
		PayloadRef: "ns1/blob1",
	}, nil)

	mdx := dm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("DownloadBlob", ctx, "ns1/blob1").Return(
		ioutil.NopCloser(bytes.NewReader([]byte("some blob"))),
		nil)

	blob, reader, err := dm.DownloadBlob(ctx, "ns1", dataID.String())
	assert.NoError(t, err)
	assert.Equal(t, blobHash.String(), blob.Hash.String())
	b, err := ioutil.ReadAll(reader)
	reader.Close()
	assert.Equal(t, "some blob", string(b))

}

func TestDownloadBlobNotFound(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	blobHash := fftypes.NewRandB32()
	dataID := fftypes.NewUUID()

	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetDataByID", ctx, dataID, false).Return(&core.Data{
		ID:        dataID,
		Namespace: "ns1",
		Blob: &core.BlobRef{
			Hash: blobHash,
		},
	}, nil)
	mdi.On("GetBlobMatchingHash", ctx, blobHash).Return(nil, nil)

	_, _, err := dm.DownloadBlob(ctx, "ns1", dataID.String())
	assert.Regexp(t, "FF10239", err)

}

func TestDownloadBlobLookupErr(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	blobHash := fftypes.NewRandB32()
	dataID := fftypes.NewUUID()

	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetDataByID", ctx, dataID, false).Return(&core.Data{
		ID:        dataID,
		Namespace: "ns1",
		Blob: &core.BlobRef{
			Hash: blobHash,
		},
	}, nil)
	mdi.On("GetBlobMatchingHash", ctx, blobHash).Return(nil, fmt.Errorf("pop"))

	_, _, err := dm.DownloadBlob(ctx, "ns1", dataID.String())
	assert.Regexp(t, "pop", err)

}

func TestDownloadBlobNoBlob(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	dataID := fftypes.NewUUID()

	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetDataByID", ctx, dataID, false).Return(&core.Data{
		ID:        dataID,
		Namespace: "ns1",
		Blob:      &core.BlobRef{},
	}, nil)

	_, _, err := dm.DownloadBlob(ctx, "ns1", dataID.String())
	assert.Regexp(t, "FF10241", err)

}

func TestDownloadBlobNSMismatch(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	dataID := fftypes.NewUUID()

	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetDataByID", ctx, dataID, false).Return(&core.Data{
		ID:        dataID,
		Namespace: "ns2",
		Blob:      &core.BlobRef{},
	}, nil)

	_, _, err := dm.DownloadBlob(ctx, "ns1", dataID.String())
	assert.Regexp(t, "FF10143", err)

}

func TestDownloadBlobDataLookupErr(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	dataID := fftypes.NewUUID()

	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetDataByID", ctx, dataID, false).Return(nil, fmt.Errorf("pop"))

	_, _, err := dm.DownloadBlob(ctx, "ns1", dataID.String())
	assert.Regexp(t, "pop", err)

}

func TestDownloadBlobBadNS(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	dataID := fftypes.NewUUID()

	_, _, err := dm.DownloadBlob(ctx, "!wrong", dataID.String())
	assert.Regexp(t, "FF00140.*namespace", err)

}

func TestDownloadBlobBadID(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	_, _, err := dm.DownloadBlob(ctx, "ns1", "!uuid")
	assert.Regexp(t, "FF00138", err)

}
