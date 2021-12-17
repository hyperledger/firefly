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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gorilla/mux"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/oapispec"
	"github.com/hyperledger/firefly/mocks/orchestratormocks"
	"github.com/stretchr/testify/assert"
)

const configDir = "../../test/data/config"

func newTestServer() (*orchestratormocks.Orchestrator, *apiServer) {
	InitConfig()
	mor := &orchestratormocks.Orchestrator{}
	as := &apiServer{
		apiTimeout: 5 * time.Second,
	}
	return mor, as
}

func newTestAPIServer() (*orchestratormocks.Orchestrator, *mux.Router) {
	mor, as := newTestServer()
	r := as.createMuxRouter(context.Background(), mor)
	return mor, r
}

func newTestAdminServer() (*orchestratormocks.Orchestrator, *mux.Router) {
	mor, as := newTestServer()
	r := as.createAdminMuxRouter(mor)
	return mor, r
}

func TestStartStopServer(t *testing.T) {
	config.Reset()
	metrics.Clear()
	InitConfig()
	apiConfigPrefix.Set(HTTPConfPort, 0)
	adminConfigPrefix.Set(HTTPConfPort, 0)
	config.Set(config.UIPath, "test")
	config.Set(config.AdminEnabled, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // server will immediately shut down
	as := NewAPIServer()
	mor := &orchestratormocks.Orchestrator{}
	mor.On("IsPreInit").Return(false)
	err := as.Serve(ctx, mor)
	assert.NoError(t, err)
}

func TestStartAPIFail(t *testing.T) {
	config.Reset()
	metrics.Clear()
	InitConfig()
	apiConfigPrefix.Set(HTTPConfAddress, "...://")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // server will immediately shut down
	as := NewAPIServer()
	mor := &orchestratormocks.Orchestrator{}
	mor.On("IsPreInit").Return(false)
	err := as.Serve(ctx, mor)
	assert.Regexp(t, "FF10104", err)
}

func TestStartAdminFail(t *testing.T) {
	config.Reset()
	metrics.Clear()
	InitConfig()
	adminConfigPrefix.Set(HTTPConfAddress, "...://")
	config.Set(config.AdminEnabled, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // server will immediately shut down
	as := NewAPIServer()
	mor := &orchestratormocks.Orchestrator{}
	mor.On("IsPreInit").Return(true)
	err := as.Serve(ctx, mor)
	assert.Regexp(t, "FF10104", err)
}

func TestStartMetricsFail(t *testing.T) {
	config.Reset()
	metrics.Clear()
	InitConfig()
	metricsConfigPrefix.Set(HTTPConfAddress, "...://")
	config.Set(config.MetricsEnabled, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // server will immediately shut down
	as := NewAPIServer()
	mor := &orchestratormocks.Orchestrator{}
	mor.On("IsPreInit").Return(true)
	err := as.Serve(ctx, mor)
	assert.Regexp(t, "FF10104", err)
}

func TestJSONHTTPServePOST201(t *testing.T) {
	mo, as := newTestServer()
	handler := as.routeHandler(mo, "http://localhost:5000/api/v1", &oapispec.Route{
		Name:            "testRoute",
		Path:            "/test",
		Method:          "POST",
		JSONInputValue:  func() interface{} { return make(map[string]interface{}) },
		JSONOutputValue: func() interface{} { return make(map[string]interface{}) },
		JSONOutputCodes: []int{201},
		JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
			assert.Equal(t, "value1", r.Input.(map[string]interface{})["input1"])
			return map[string]interface{}{"output1": "value2"}, nil
		},
	})
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	b, _ := json.Marshal(map[string]interface{}{"input1": "value1"})
	res, err := http.Post(fmt.Sprintf("http://%s/test", s.Listener.Addr()), "application/json", bytes.NewReader(b))
	assert.NoError(t, err)
	assert.Equal(t, 201, res.StatusCode)
	var resJSON map[string]interface{}
	json.NewDecoder(res.Body).Decode(&resJSON)
	assert.Equal(t, "value2", resJSON["output1"])
}

func TestJSONHTTPResponseEncodeFail(t *testing.T) {
	mo, as := newTestServer()
	handler := as.routeHandler(mo, "http://localhost:5000/api/v1", &oapispec.Route{
		Name:            "testRoute",
		Path:            "/test",
		Method:          "GET",
		JSONInputValue:  nil,
		JSONOutputValue: func() interface{} { return make(map[string]interface{}) },
		JSONOutputCodes: []int{200},
		JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
			v := map[string]interface{}{"unserializable": map[bool]interface{}{true: "not in JSON"}}
			return v, nil
		},
	})
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	b, _ := json.Marshal(map[string]interface{}{"input1": "value1"})
	res, err := http.Post(fmt.Sprintf("http://%s/test", s.Listener.Addr()), "application/json", bytes.NewReader(b))
	assert.NoError(t, err)
	var resJSON map[string]interface{}
	json.NewDecoder(res.Body).Decode(&resJSON)
	assert.Regexp(t, "FF10107", resJSON["error"])
}

func TestJSONHTTPNilResponseNon204(t *testing.T) {
	mo, as := newTestServer()
	handler := as.routeHandler(mo, "http://localhost:5000/api/v1", &oapispec.Route{
		Name:            "testRoute",
		Path:            "/test",
		Method:          "GET",
		JSONInputValue:  nil,
		JSONOutputValue: func() interface{} { return make(map[string]interface{}) },
		JSONOutputCodes: []int{200},
		JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
			return nil, nil
		},
	})
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	b, _ := json.Marshal(map[string]interface{}{"input1": "value1"})
	res, err := http.Post(fmt.Sprintf("http://%s/test", s.Listener.Addr()), "application/json", bytes.NewReader(b))
	assert.NoError(t, err)
	assert.Equal(t, 404, res.StatusCode)
	var resJSON map[string]interface{}
	json.NewDecoder(res.Body).Decode(&resJSON)
	assert.Regexp(t, "FF10143", resJSON["error"])
}

func TestJSONHTTPDefault500Error(t *testing.T) {
	mo, as := newTestServer()
	handler := as.routeHandler(mo, "http://localhost:5000/api/v1", &oapispec.Route{
		Name:            "testRoute",
		Path:            "/test",
		Method:          "GET",
		JSONInputValue:  nil,
		JSONOutputValue: func() interface{} { return make(map[string]interface{}) },
		JSONOutputCodes: []int{200},
		JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
			return nil, fmt.Errorf("pop")
		},
	})
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	b, _ := json.Marshal(map[string]interface{}{"input1": "value1"})
	res, err := http.Post(fmt.Sprintf("http://%s/test", s.Listener.Addr()), "application/json", bytes.NewReader(b))
	assert.NoError(t, err)
	assert.Equal(t, 500, res.StatusCode)
	var resJSON map[string]interface{}
	json.NewDecoder(res.Body).Decode(&resJSON)
	assert.Regexp(t, "pop", resJSON["error"])
}

func TestStatusCodeHintMapping(t *testing.T) {
	mo, as := newTestServer()
	handler := as.routeHandler(mo, "http://localhost:5000/api/v1", &oapispec.Route{
		Name:            "testRoute",
		Path:            "/test",
		Method:          "GET",
		JSONInputValue:  nil,
		JSONOutputValue: func() interface{} { return make(map[string]interface{}) },
		JSONOutputCodes: []int{200},
		JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
			return nil, i18n.NewError(r.Ctx, i18n.MsgResponseMarshalError)
		},
	})
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	b, _ := json.Marshal(map[string]interface{}{"input1": "value1"})
	res, err := http.Post(fmt.Sprintf("http://%s/test", s.Listener.Addr()), "application/json", bytes.NewReader(b))
	assert.NoError(t, err)
	assert.Equal(t, 400, res.StatusCode)
	var resJSON map[string]interface{}
	json.NewDecoder(res.Body).Decode(&resJSON)
	assert.Regexp(t, "FF10107", resJSON["error"])
}

func TestStatusInvalidContentType(t *testing.T) {
	mo, as := newTestServer()
	handler := as.routeHandler(mo, "http://localhost:5000/api/v1", &oapispec.Route{
		Name:            "testRoute",
		Path:            "/test",
		Method:          "POST",
		JSONInputValue:  nil,
		JSONOutputValue: func() interface{} { return make(map[string]interface{}) },
		JSONOutputCodes: []int{204},
		JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
			return nil, nil
		},
	})
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	res, err := http.Post(fmt.Sprintf("http://%s/test", s.Listener.Addr()), "application/text", bytes.NewReader([]byte{}))
	assert.NoError(t, err)
	assert.Equal(t, 415, res.StatusCode)
	var resJSON map[string]interface{}
	json.NewDecoder(res.Body).Decode(&resJSON)
	assert.Regexp(t, "FF10130", resJSON["error"])
}

func TestNotFound(t *testing.T) {
	_, as := newTestServer()
	handler := as.apiWrapper(as.notFoundHandler)
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	res, err := http.Get(fmt.Sprintf("http://%s/test", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 404, res.StatusCode)
	var resJSON map[string]interface{}
	json.NewDecoder(res.Body).Decode(&resJSON)
	assert.Regexp(t, "FF10109", resJSON["error"])
}

func TestTimeout(t *testing.T) {
	mo, as := newTestServer()
	handler := as.routeHandler(mo, "http://localhost:5000/api/v1", &oapispec.Route{
		Name:            "testRoute",
		Path:            "/test",
		Method:          "POST",
		JSONInputValue:  nil,
		JSONOutputValue: func() interface{} { return make(map[string]interface{}) },
		JSONOutputCodes: []int{204},
		JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
			<-r.Ctx.Done()
			return nil, fmt.Errorf("timeout error")
		},
	})
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/test", s.Listener.Addr()), bytes.NewReader([]byte(``)))
	assert.NoError(t, err)
	req.Header.Set("Request-Timeout", "250us")
	res, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 408, res.StatusCode)
	var resJSON map[string]interface{}
	json.NewDecoder(res.Body).Decode(&resJSON)
	assert.Regexp(t, "FF10260.*timeout error", resJSON["error"])
}

func TestBadTimeout(t *testing.T) {
	mo, as := newTestServer()
	handler := as.routeHandler(mo, "http://localhost:5000/api/v1", &oapispec.Route{
		Name:            "testRoute",
		Path:            "/test",
		Method:          "POST",
		JSONInputValue:  nil,
		JSONOutputValue: func() interface{} { return make(map[string]interface{}) },
		JSONOutputCodes: []int{204},
		JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
			return nil, nil
		},
	})
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/test", s.Listener.Addr()), bytes.NewReader([]byte(``)))
	assert.NoError(t, err)
	req.Header.Set("Request-Timeout", "bad timeout")
	res, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 204, res.StatusCode)
}

func TestSwaggerUI(t *testing.T) {
	_, as := newTestServer()
	handler := as.apiWrapper(as.swaggerUIHandler("http://localhost:5000/api/v1"))
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	res, err := http.Get(fmt.Sprintf("http://%s/api", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	b, _ := ioutil.ReadAll(res.Body)
	assert.Regexp(t, "html", string(b))
}

func TestSwaggerYAML(t *testing.T) {
	_, as := newTestServer()
	handler := as.apiWrapper(as.swaggerHandler(routes, "http://localhost:12345/api/v1"))
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
	_, r := newTestAPIServer()
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
	_, r := newTestAdminServer()
	s := httptest.NewServer(r)
	defer s.Close()

	res, err := http.Get(fmt.Sprintf("http://%s/admin/api/swagger.json", s.Listener.Addr()))
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

func TestGetTimeoutMax(t *testing.T) {
	_, as := newTestServer()
	as.apiMaxTimeout = 1 * time.Second
	req, err := http.NewRequest("GET", "http://test.example.com", bytes.NewReader([]byte(``)))
	req.Header.Set("Request-Timeout", "1h")
	assert.NoError(t, err)
	timeout := as.getTimeout(req)
	assert.Equal(t, 1*time.Second, timeout)
}
