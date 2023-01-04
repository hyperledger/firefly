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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gorilla/mux"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/httpserver"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/mocks/apiservermocks"
	"github.com/hyperledger/firefly/mocks/contractmocks"
	"github.com/hyperledger/firefly/mocks/namespacemocks"
	"github.com/hyperledger/firefly/mocks/orchestratormocks"
	"github.com/hyperledger/firefly/mocks/spieventsmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const configDir = "../../test/data/config"

func newTestServer() (*namespacemocks.Manager, *orchestratormocks.Orchestrator, *apiServer) {
	coreconfig.Reset()
	InitConfig()
	mgr := &namespacemocks.Manager{}
	o := &orchestratormocks.Orchestrator{}
	mgr.On("Orchestrator", "default").Return(o).Maybe()
	mgr.On("Orchestrator", "mynamespace").Return(o).Maybe()
	mgr.On("Orchestrator", "ns1").Return(o).Maybe()
	config.Set(coreconfig.APIMaxFilterLimit, 100)
	as := &apiServer{
		apiTimeout:    5 * time.Second,
		ffiSwaggerGen: &apiservermocks.FFISwaggerGen{},
	}
	return mgr, o, as
}

func newTestAPIServer() (*orchestratormocks.Orchestrator, *mux.Router) {
	mgr, o, as := newTestServer()
	r := as.createMuxRouter(context.Background(), mgr)
	return o, r
}

func newTestSPIServer() (*orchestratormocks.Orchestrator, *mux.Router) {
	config.Set(coreconfig.NamespacesDefault, "default")
	mgr, o, as := newTestServer()
	mae := &spieventsmocks.Manager{}
	mgr.On("SPIEvents").Return(mae)
	r := as.createAdminMuxRouter(mgr)
	return o, r
}

func TestStartStopServer(t *testing.T) {
	coreconfig.Reset()
	metrics.Clear()
	InitConfig()
	apiConfig.Set(httpserver.HTTPConfPort, 0)
	spiConfig.Set(httpserver.HTTPConfPort, 0)
	config.Set(coreconfig.UIPath, "test")
	config.Set(coreconfig.SPIEnabled, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // server will immediately shut down
	as := NewAPIServer()
	mgr := &namespacemocks.Manager{}
	mae := &spieventsmocks.Manager{}
	mgr.On("SPIEvents").Return(mae)
	err := as.Serve(ctx, mgr)
	assert.NoError(t, err)
}

func TestStartLegacyAdminConfig(t *testing.T) {
	coreconfig.Reset()
	metrics.Clear()
	InitConfig()
	apiConfig.Set(httpserver.HTTPConfPort, 0)
	spiConfig.Set(httpserver.HTTPConfPort, 0)
	config.Set(coreconfig.UIPath, "test")
	config.Set(coreconfig.LegacyAdminEnabled, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // server will immediately shut down
	as := NewAPIServer()
	mgr := &namespacemocks.Manager{}
	mae := &spieventsmocks.Manager{}
	mgr.On("SPIEvents").Return(mae)
	err := as.Serve(ctx, mgr)
	assert.NoError(t, err)
}

func TestStartAPIFail(t *testing.T) {
	coreconfig.Reset()
	metrics.Clear()
	InitConfig()
	apiConfig.Set(httpserver.HTTPConfAddress, "...://")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // server will immediately shut down
	as := NewAPIServer()
	mgr := &namespacemocks.Manager{}
	err := as.Serve(ctx, mgr)
	assert.Regexp(t, "FF00151", err)
}

func TestStartAdminFail(t *testing.T) {
	coreconfig.Reset()
	metrics.Clear()
	InitConfig()
	spiConfig.Set(httpserver.HTTPConfAddress, "...://")
	config.Set(coreconfig.SPIEnabled, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // server will immediately shut down
	as := NewAPIServer()
	mgr := &namespacemocks.Manager{}
	mae := &spieventsmocks.Manager{}
	mgr.On("SPIEvents").Return(mae)
	err := as.Serve(ctx, mgr)
	assert.Regexp(t, "FF00151", err)
}

func TestStartAdminWSHandler(t *testing.T) {
	coreconfig.Reset()
	metrics.Clear()
	InitConfig()
	spiConfig.Set(httpserver.HTTPConfAddress, "...://")
	config.Set(coreconfig.SPIEnabled, true)
	as := NewAPIServer().(*apiServer)
	mgr := &namespacemocks.Manager{}
	mae := &spieventsmocks.Manager{}
	mgr.On("SPIEvents").Return(mae)
	mae.On("ServeHTTPWebSocketListener", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		res := args[0].(http.ResponseWriter)
		res.WriteHeader(200)
	}).Return()
	res := httptest.NewRecorder()
	as.spiWSHandler(mgr).ServeHTTP(res, httptest.NewRequest("GET", "/", nil))
	assert.Equal(t, 200, res.Result().StatusCode)
}

func TestStartMetricsFail(t *testing.T) {
	coreconfig.Reset()
	metrics.Clear()
	InitConfig()
	metricsConfig.Set(httpserver.HTTPConfAddress, "...://")
	config.Set(coreconfig.MetricsEnabled, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // server will immediately shut down
	as := NewAPIServer()
	mgr := &namespacemocks.Manager{}
	mae := &spieventsmocks.Manager{}
	mgr.On("SPIEvents").Return(mae)
	err := as.Serve(ctx, mgr)
	assert.Regexp(t, "FF00151", err)
}

func TestNotFound(t *testing.T) {
	_, _, as := newTestServer()
	handler := as.handlerFactory().APIWrapper(as.notFoundHandler)
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	res, err := http.Get(fmt.Sprintf("http://%s/test", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 404, res.StatusCode)
	var resJSON map[string]interface{}
	json.NewDecoder(res.Body).Decode(&resJSON)
	assert.Regexp(t, "FF10109", resJSON["error"])
}

func TestFilterTooMany(t *testing.T) {
	mgr, o, as := newTestServer()
	o.On("Authorize", mock.Anything, mock.Anything).Return(nil)
	handler := as.routeHandler(as.handlerFactory(), mgr, "", getBatches)

	req := httptest.NewRequest("GET", "http://localhost:12345/test?limit=99999999999", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	assert.Equal(t, 400, res.Result().StatusCode)
	var resJSON map[string]interface{}
	json.NewDecoder(res.Body).Decode(&resJSON)
	assert.Regexp(t, "FF00192", resJSON["error"])
}

func TestUnauthorized(t *testing.T) {
	mgr, o, as := newTestServer()
	o.On("Authorize", mock.Anything, mock.Anything).Return(i18n.NewError(context.Background(), i18n.MsgUnauthorized))
	handler := as.routeHandler(as.handlerFactory(), mgr, "", getBatches)

	req := httptest.NewRequest("GET", "http://localhost:12345/test", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	assert.Equal(t, 401, res.Result().StatusCode)
	var resJSON map[string]interface{}
	json.NewDecoder(res.Body).Decode(&resJSON)
	assert.Regexp(t, "FF00169", resJSON["error"])
}

func TestSwaggerYAML(t *testing.T) {
	_, _, as := newTestServer()
	handler := as.handlerFactory().APIWrapper(as.swaggerHandler(as.swaggerGenerator(routes, "http://localhost:12345/api/v1")))
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	res, err := http.Get(fmt.Sprintf("http://%s/api/swagger.yaml", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	b, _ := ioutil.ReadAll(res.Body)
	doc, err := openapi3.NewLoader().LoadFromData(b)
	assert.NoError(t, err)
	err = doc.Validate(context.Background())
	assert.NoError(t, err)
}

func TestSwaggerJSON(t *testing.T) {
	o, r := newTestAPIServer()
	o.On("Authorize", mock.Anything, mock.Anything).Return(nil)
	s := httptest.NewServer(r)
	defer s.Close()

	res, err := http.Get(fmt.Sprintf("http://%s/api/swagger.json", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	b, _ := ioutil.ReadAll(res.Body)
	err = json.Unmarshal(b, &openapi3.T{})
	assert.NoError(t, err)
}

func TestSwaggerAdminJSON(t *testing.T) {
	_, r := newTestSPIServer()
	s := httptest.NewServer(r)
	defer s.Close()

	res, err := http.Get(fmt.Sprintf("http://%s/spi/swagger.json", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	b, _ := ioutil.ReadAll(res.Body)
	err = json.Unmarshal(b, &openapi3.T{})
	assert.NoError(t, err)
}

func TestWaitForServerStop(t *testing.T) {

	chl1 := make(chan error, 1)
	chl2 := make(chan error, 1)
	chl3 := make(chan error, 1)
	chl1 <- fmt.Errorf("pop1")

	as := &apiServer{}
	err := as.waitForServerStop(chl1, chl2, chl3)
	assert.EqualError(t, err, "pop1")

	chl2 <- fmt.Errorf("pop2")
	err = as.waitForServerStop(chl1, chl2, chl3)
	assert.EqualError(t, err, "pop2")

	chl3 <- fmt.Errorf("pop3")
	err = as.waitForServerStop(chl1, chl2, chl3)
	assert.EqualError(t, err, "pop3")
}

func TestContractAPISwaggerJSON(t *testing.T) {
	mgr, o, as := newTestServer()
	r := as.createMuxRouter(context.Background(), mgr)
	mcm := &contractmocks.Manager{}
	o.On("Contracts").Return(mcm)
	mffi := as.ffiSwaggerGen.(*apiservermocks.FFISwaggerGen)
	s := httptest.NewServer(r)
	defer s.Close()

	ffi := &fftypes.FFI{}
	api := &core.ContractAPI{
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}

	mcm.On("GetContractAPI", mock.Anything, "http://127.0.0.1:5000/api/v1", "my-api").Return(api, nil)
	mcm.On("GetFFIByIDWithChildren", mock.Anything, api.Interface.ID).Return(ffi, nil)
	mffi.On("Generate", mock.Anything, "http://127.0.0.1:5000/api/v1/namespaces/default/apis/my-api", api, ffi).Return(&openapi3.T{})

	res, err := http.Get(fmt.Sprintf("http://%s/api/v1/namespaces/default/apis/my-api/api/swagger.json", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
}

func TestContractAPISwaggerJSONGetAPIFail(t *testing.T) {
	mgr, o, as := newTestServer()
	r := as.createMuxRouter(context.Background(), mgr)
	mcm := &contractmocks.Manager{}
	o.On("Contracts").Return(mcm)
	s := httptest.NewServer(r)
	defer s.Close()

	mcm.On("GetContractAPI", mock.Anything, "http://127.0.0.1:5000/api/v1", "my-api").Return(nil, fmt.Errorf("pop"))

	res, err := http.Get(fmt.Sprintf("http://%s/api/v1/namespaces/default/apis/my-api/api/swagger.json", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 500, res.StatusCode)
}

func TestContractAPISwaggerJSONGetAPINotFound(t *testing.T) {
	mgr, o, as := newTestServer()
	r := as.createMuxRouter(context.Background(), mgr)
	mcm := &contractmocks.Manager{}
	o.On("Contracts").Return(mcm)
	s := httptest.NewServer(r)
	defer s.Close()

	mcm.On("GetContractAPI", mock.Anything, "http://127.0.0.1:5000/api/v1", "my-api").Return(nil, nil)

	res, err := http.Get(fmt.Sprintf("http://%s/api/v1/namespaces/default/apis/my-api/api/swagger.json", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 404, res.StatusCode)
}

func TestContractAPISwaggerJSONGetFFIFail(t *testing.T) {
	mgr, o, as := newTestServer()
	r := as.createMuxRouter(context.Background(), mgr)
	mcm := &contractmocks.Manager{}
	o.On("Contracts").Return(mcm)
	s := httptest.NewServer(r)
	defer s.Close()

	api := &core.ContractAPI{
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}

	mcm.On("GetContractAPI", mock.Anything, "http://127.0.0.1:5000/api/v1", "my-api").Return(api, nil)
	mcm.On("GetFFIByIDWithChildren", mock.Anything, api.Interface.ID).Return(nil, fmt.Errorf("pop"))

	res, err := http.Get(fmt.Sprintf("http://%s/api/v1/namespaces/default/apis/my-api/api/swagger.json", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 500, res.StatusCode)
}

func TestContractAPISwaggerJSONBadNamespace(t *testing.T) {
	mgr, o, as := newTestServer()
	r := as.createMuxRouter(context.Background(), mgr)
	mcm := &contractmocks.Manager{}
	o.On("Contracts").Return(mcm)
	s := httptest.NewServer(r)
	defer s.Close()

	mgr.On("Orchestrator", "BAD").Return(nil)

	res, err := http.Get(fmt.Sprintf("http://%s/api/v1/namespaces/BAD/apis/my-api/api/swagger.json", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 404, res.StatusCode)
}

func TestContractAPISwaggerUI(t *testing.T) {
	o, r := newTestAPIServer()
	o.On("Authorize", mock.Anything, mock.Anything).Return(nil)
	s := httptest.NewServer(r)
	defer s.Close()

	res, err := http.Get(fmt.Sprintf("http://%s/api/v1/namespaces/default/apis/my-api/api", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	b, _ := ioutil.ReadAll(res.Body)
	assert.Regexp(t, "html", string(b))
}

func TestJSONBadNamespace(t *testing.T) {
	mgr, _, as := newTestServer()
	r := as.createMuxRouter(context.Background(), mgr)
	s := httptest.NewServer(r)
	defer s.Close()

	mgr.On("Orchestrator", "BAD").Return(nil)

	var b bytes.Buffer
	req := httptest.NewRequest("GET", "/api/v1/namespaces/BAD/apis", &b)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	assert.Equal(t, 404, res.Result().StatusCode)
}

func TestFormDataBadNamespace(t *testing.T) {
	mgr, _, as := newTestServer()
	r := as.createMuxRouter(context.Background(), mgr)
	s := httptest.NewServer(r)
	defer s.Close()

	mgr.On("Orchestrator", "BAD").Return(nil)

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	writer, err := w.CreateFormFile("file", "filename.ext")
	assert.NoError(t, err)
	writer.Write([]byte(`some data`))
	w.Close()
	req := httptest.NewRequest("POST", "/api/v1/namespaces/BAD/data", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	assert.Equal(t, 404, res.Result().StatusCode)
}

func TestJSONDisabledRoute(t *testing.T) {
	mgr, o, as := newTestServer()
	o.On("Authorize", mock.Anything, mock.Anything).Return(nil)
	r := as.createMuxRouter(context.Background(), mgr)
	s := httptest.NewServer(r)
	defer s.Close()

	o.On("PrivateMessaging").Return(nil)

	var b bytes.Buffer
	req := httptest.NewRequest("GET", "/api/v1/namespaces/ns1/groups", &b)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	assert.Equal(t, 400, res.Result().StatusCode)
}

func TestFormDataDisabledRoute(t *testing.T) {
	mgr, o, as := newTestServer()
	o.On("Authorize", mock.Anything, mock.Anything).Return(nil)
	o.On("Data").Return(nil)
	r := as.createMuxRouter(context.Background(), mgr)
	s := httptest.NewServer(r)
	defer s.Close()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	writer, err := w.CreateFormFile("file", "filename.ext")
	assert.NoError(t, err)
	writer.Write([]byte(`some data`))
	w.Close()
	req := httptest.NewRequest("POST", "/api/v1/namespaces/ns1/data", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	assert.Equal(t, 400, res.Result().StatusCode)
}
