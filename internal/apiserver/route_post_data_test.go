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

package apiserver

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/mocks/datamocks"
	"github.com/hyperledger-labs/firefly/mocks/orchestratormocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPostDataJSON(t *testing.T) {
	o := &orchestratormocks.Orchestrator{}
	mdm := &datamocks.Manager{}
	o.On("Data").Return(mdm)
	as := &apiServer{}
	r := as.createMuxRouter(o)
	input := fftypes.Data{}
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(&input)
	req := httptest.NewRequest("POST", "/api/v1/namespaces/ns1/data", &buf)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res := httptest.NewRecorder()

	mdm.On("UploadJSON", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.Data")).
		Return(&fftypes.Data{}, nil)
	r.ServeHTTP(res, req)

	assert.Equal(t, 201, res.Result().StatusCode)
}

func TestPostDataBinary(t *testing.T) {
	log.SetLevel("debug")

	o := &orchestratormocks.Orchestrator{}
	mdm := &datamocks.Manager{}
	o.On("Data").Return(mdm)
	as := &apiServer{}
	r := as.createMuxRouter(o)

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	writer, err := w.CreateFormFile("file", "filename.ext")
	assert.NoError(t, err)
	writer.Write([]byte(`some data`))
	w.Close()
	req := httptest.NewRequest("POST", "/api/v1/namespaces/ns1/data", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())

	res := httptest.NewRecorder()

	mdm.On("UploadBLOB", mock.Anything, "ns1", mock.AnythingOfType("*multipart.Part")).
		Return(&fftypes.Data{}, nil)
	r.ServeHTTP(res, req)

	assert.Equal(t, 201, res.Result().StatusCode)
}

func TestPostDataBinaryMissing(t *testing.T) {
	log.SetLevel("debug")

	o := &orchestratormocks.Orchestrator{}
	mdm := &datamocks.Manager{}
	o.On("Data").Return(mdm)
	as := &apiServer{}
	r := as.createMuxRouter(o)

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	formField, _ := w.CreateFormField("Not a file")
	formField.Write([]byte(`some value`))
	w.Close()
	req := httptest.NewRequest("POST", "/api/v1/namespaces/ns1/data", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())

	res := httptest.NewRecorder()

	mdm.On("UploadBLOB", mock.Anything, "ns1", mock.AnythingOfType("*multipart.Part")).
		Return(&fftypes.Data{}, nil)
	r.ServeHTTP(res, req)

	assert.Equal(t, 400, res.Result().StatusCode)
}

func TestPostDataBadForm(t *testing.T) {
	log.SetLevel("debug")

	o := &orchestratormocks.Orchestrator{}
	mdm := &datamocks.Manager{}
	o.On("Data").Return(mdm)
	as := &apiServer{}
	r := as.createMuxRouter(o)

	var b bytes.Buffer
	req := httptest.NewRequest("POST", "/api/v1/namespaces/ns1/data", &b)
	req.Header.Set("Content-Type", "multipart/form-data")

	res := httptest.NewRecorder()

	mdm.On("UploadBLOB", mock.Anything, "ns1", mock.AnythingOfType("*multipart.Part")).
		Return(&fftypes.Data{}, nil)
	r.ServeHTTP(res, req)

	assert.Equal(t, 400, res.Result().StatusCode)
}
