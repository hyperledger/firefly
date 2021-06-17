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
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"testing"
	"testing/iotest"

	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
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
	mdi.On("UpsertData", mock.Anything, mock.Anything, false, false).Return(nil)

	dxID := make(chan fftypes.UUID, 1)
	mdx := dm.exchange.(*dataexchangemocks.Plugin)
	dxUpload := mdx.On("UploadBLOB", ctx, "ns1", mock.Anything, mock.Anything)
	dxUpload.RunFn = func(a mock.Arguments) {
		readBytes, err := ioutil.ReadAll(a[3].(io.Reader))
		assert.Nil(t, err)
		assert.Equal(t, b, readBytes)
		dxID <- a[2].(fftypes.UUID)
		dxUpload.ReturnArguments = mock.Arguments{err}
	}

	data, err := dm.UploadBLOB(ctx, "ns1", &fftypes.Data{}, &fftypes.Multipart{Data: bytes.NewReader(b)}, false)
	assert.NoError(t, err)

	// Check the hashes and other details of the data
	assert.Equal(t, [32]byte(sha256.Sum256(b)), [32]byte(*data.Hash))
	assert.Equal(t, <-dxID, *data.ID)
	assert.Empty(t, data.Validator)
	assert.Nil(t, data.Datatype)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)

}

func TestUploadBlobAutoMetaOk(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("UpsertData", mock.Anything, mock.Anything, false, false).Return(nil)

	dxID := make(chan fftypes.UUID, 1)
	mdx := dm.exchange.(*dataexchangemocks.Plugin)
	dxUpload := mdx.On("UploadBLOB", ctx, "ns1", mock.Anything, mock.Anything)
	dxUpload.RunFn = func(a mock.Arguments) {
		_, err := ioutil.ReadAll(a[3].(io.Reader))
		assert.Nil(t, err)
		dxID <- a[2].(fftypes.UUID)
		dxUpload.ReturnArguments = mock.Arguments{err}
	}

	data, err := dm.UploadBLOB(ctx, "ns1", &fftypes.Data{
		Value: fftypes.Byteable(`{"custom": "value1"}`),
	}, &fftypes.Multipart{
		Data:     bytes.NewReader([]byte(`hello`)),
		Filename: "myfile.csv",
		Mimetype: "text/csv",
	}, true)
	assert.NoError(t, err)
	assert.Equal(t, "myfile.csv", data.Value.JSONObject().GetString("filename"))
	assert.Equal(t, float64(5), map[string]interface{}(data.Value.JSONObject())["size"])
	assert.Equal(t, "text/csv", data.Value.JSONObject().GetString("mimetype"))
	assert.Equal(t, "value1", data.Value.JSONObject().GetString("custom"))

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)

}

func TestUploadBlobReadFail(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	mdx := dm.exchange.(*dataexchangemocks.Plugin)
	dxUpload := mdx.On("UploadBLOB", ctx, "ns1", mock.Anything, mock.Anything).Return(nil)
	dxUpload.RunFn = func(a mock.Arguments) {
		_, err := ioutil.ReadAll(a[3].(io.Reader))
		assert.NoError(t, err)
	}

	_, err := dm.UploadBLOB(ctx, "ns1", &fftypes.Data{}, &fftypes.Multipart{Data: iotest.ErrReader(fmt.Errorf("pop"))}, false)
	assert.Regexp(t, "FF10217.*pop", err)

}

func TestUploadBlobWriteFailDoesNotRead(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	mdx := dm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("UploadBLOB", ctx, "ns1", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := dm.UploadBLOB(ctx, "ns1", &fftypes.Data{}, &fftypes.Multipart{Data: bytes.NewReader([]byte(`any old data`))}, false)
	assert.Regexp(t, "pop", err)

}

func TestUploadBlobUpsertFail(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	mdx := dm.exchange.(*dataexchangemocks.Plugin)
	dxUpload := mdx.On("UploadBLOB", ctx, "ns1", mock.Anything, mock.Anything).Return(nil)
	dxUpload.RunFn = func(a mock.Arguments) {
		_, err := ioutil.ReadAll(a[3].(io.Reader))
		assert.Nil(t, err)
	}
	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("UpsertData", mock.Anything, mock.Anything, false, false).Return(fmt.Errorf("pop"))

	_, err := dm.UploadBLOB(ctx, "ns1", &fftypes.Data{}, &fftypes.Multipart{Data: bytes.NewReader([]byte(`any old data`))}, false)
	assert.Regexp(t, "pop", err)

}
