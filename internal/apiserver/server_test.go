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
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gorilla/mux"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/oapispec"
	"github.com/hyperledger-labs/firefly/mocks/orchestratormocks"
	"github.com/stretchr/testify/assert"
)

const configDir = "../../test/data/config"

func TestStartStopServer(t *testing.T) {
	config.Reset()
	config.Set(config.HTTPPort, 0)
	config.Set(config.UIPath, "test")
	config.Set(config.AdminEnabled, true)
	config.Set(config.AdminPort, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // server will immediately shut down
	err := Serve(ctx, &orchestratormocks.Orchestrator{})
	assert.NoError(t, err)
}

func TestInvalidListener(t *testing.T) {
	config.Reset()
	config.Set(config.HTTPAddress, "...")
	_, err := createListener(context.Background())
	assert.Error(t, err)
}

func TestInvalidAdminListener(t *testing.T) {
	config.Reset()
	config.Set(config.AdminAddress, "...")
	_, err := createAdminListener(context.Background())
	assert.Error(t, err)
}

func TestServeFail(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	l.Close() // So server will fail
	s := &http.Server{}
	err := serveHTTP(context.Background(), l, s)
	assert.Error(t, err)
}

func TestMissingCAFile(t *testing.T) {
	config.Reset()
	config.Set(config.HTTPTLSCAFile, "badness")
	r := mux.NewRouter()
	_, err := createServer(context.Background(), r)
	assert.Regexp(t, "FF10105", err)
}

func TestBadCAFile(t *testing.T) {
	config.Reset()
	config.Set(config.HTTPTLSCAFile, configDir+"/firefly.core.yaml")
	r := mux.NewRouter()
	_, err := createServer(context.Background(), r)
	assert.Regexp(t, "FF10106", err)
}

func TestTLSServerSelfSignedWithClientAuth(t *testing.T) {

	// Create an X509 certificate pair
	privatekey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publickey := &privatekey.PublicKey
	var privateKeyBytes []byte = x509.MarshalPKCS1PrivateKey(privatekey)
	privateKeyFile, _ := ioutil.TempFile("", "key.pem")
	defer os.Remove(privateKeyFile.Name())
	privateKeyBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes}
	pem.Encode(privateKeyFile, privateKeyBlock)
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	x509Template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Unit Tests"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(100 * time.Second),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, x509Template, x509Template, publickey, privatekey)
	assert.NoError(t, err)
	publicKeyFile, _ := ioutil.TempFile("", "cert.pem")
	defer os.Remove(publicKeyFile.Name())
	pem.Encode(publicKeyFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	// Start up a listener configured for TLS Mutual auth
	config.Reset()
	config.Set(config.HTTPTLSEnabled, true)
	config.Set(config.HTTPTLSClientAuth, true)
	config.Set(config.HTTPTLSKeyFile, privateKeyFile.Name())
	config.Set(config.HTTPTLSCertFile, publicKeyFile.Name())
	config.Set(config.HTTPTLSCAFile, publicKeyFile.Name())
	config.Set(config.HTTPPort, 0)
	ctx, cancelCtx := context.WithCancel(context.Background())
	l, err := createListener(ctx)
	assert.NoError(t, err)
	r := mux.NewRouter()
	r.HandleFunc("/test", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(200)
		json.NewEncoder(res).Encode(map[string]interface{}{"hello": "world"})
	})
	s, err := createServer(ctx, r)
	assert.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err := serveHTTP(ctx, l, s)
		assert.NoError(t, err)
		wg.Done()
	}()

	// Attempt a request, with a client certificate
	rootCAs := x509.NewCertPool()
	caPEM, _ := ioutil.ReadFile(publicKeyFile.Name())
	ok := rootCAs.AppendCertsFromPEM(caPEM)
	assert.True(t, ok)
	c := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
					clientKeyPair, err := tls.LoadX509KeyPair(publicKeyFile.Name(), privateKeyFile.Name())
					return &clientKeyPair, err
				},
				RootCAs: rootCAs,
			},
		},
	}
	httpsAddr := fmt.Sprintf("https://%s/test", l.Addr().String())
	res, err := c.Get(httpsAddr)
	assert.NoError(t, err)
	if res != nil {
		assert.Equal(t, 200, res.StatusCode)
		var resBody map[string]interface{}
		json.NewDecoder(res.Body).Decode(&resBody)
		assert.Equal(t, "world", resBody["hello"])
	}

	// Close down the server and wait for it to complete
	cancelCtx()
	wg.Wait()
}

func TestJSONHTTPServePOST201(t *testing.T) {
	mo := &orchestratormocks.Orchestrator{}
	handler := routeHandler(mo, &oapispec.Route{
		Name:            "testRoute",
		Path:            "/test",
		Method:          "POST",
		JSONInputValue:  func() interface{} { return make(map[string]interface{}) },
		JSONOutputValue: func() interface{} { return make(map[string]interface{}) },
		JSONOutputCode:  201,
		JSONHandler: func(r oapispec.APIRequest) (output interface{}, err error) {
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
	mo := &orchestratormocks.Orchestrator{}
	handler := routeHandler(mo, &oapispec.Route{
		Name:            "testRoute",
		Path:            "/test",
		Method:          "GET",
		JSONInputValue:  nil,
		JSONOutputValue: func() interface{} { return make(map[string]interface{}) },
		JSONOutputCode:  200,
		JSONHandler: func(r oapispec.APIRequest) (output interface{}, err error) {
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
	mo := &orchestratormocks.Orchestrator{}
	handler := routeHandler(mo, &oapispec.Route{
		Name:            "testRoute",
		Path:            "/test",
		Method:          "GET",
		JSONInputValue:  nil,
		JSONOutputValue: func() interface{} { return make(map[string]interface{}) },
		JSONOutputCode:  200,
		JSONHandler: func(r oapispec.APIRequest) (output interface{}, err error) {
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
	mo := &orchestratormocks.Orchestrator{}
	handler := routeHandler(mo, &oapispec.Route{
		Name:            "testRoute",
		Path:            "/test",
		Method:          "GET",
		JSONInputValue:  nil,
		JSONOutputValue: func() interface{} { return make(map[string]interface{}) },
		JSONOutputCode:  200,
		JSONHandler: func(r oapispec.APIRequest) (output interface{}, err error) {
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
	mo := &orchestratormocks.Orchestrator{}
	handler := routeHandler(mo, &oapispec.Route{
		Name:            "testRoute",
		Path:            "/test",
		Method:          "GET",
		JSONInputValue:  nil,
		JSONOutputValue: func() interface{} { return make(map[string]interface{}) },
		JSONOutputCode:  200,
		JSONHandler: func(r oapispec.APIRequest) (output interface{}, err error) {
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
	mo := &orchestratormocks.Orchestrator{}
	handler := routeHandler(mo, &oapispec.Route{
		Name:            "testRoute",
		Path:            "/test",
		Method:          "POST",
		JSONInputValue:  nil,
		JSONOutputValue: func() interface{} { return make(map[string]interface{}) },
		JSONOutputCode:  204,
		JSONHandler: func(r oapispec.APIRequest) (output interface{}, err error) {
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
	handler := apiWrapper(notFoundHandler)
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	res, err := http.Get(fmt.Sprintf("http://%s/test", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 404, res.StatusCode)
	var resJSON map[string]interface{}
	json.NewDecoder(res.Body).Decode(&resJSON)
	assert.Regexp(t, "FF10109", resJSON["error"])
}

func TestSwaggerUI(t *testing.T) {
	handler := apiWrapper(swaggerUIHandler)
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	res, err := http.Get(fmt.Sprintf("http://%s/api", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	b, _ := ioutil.ReadAll(res.Body)
	assert.Regexp(t, "html", string(b))
}

func TestAdminSwaggerUI(t *testing.T) {
	handler := apiWrapper(swaggerAdminUIHandler)
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	res, err := http.Get(fmt.Sprintf("http://%s/admin/api", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	b, _ := ioutil.ReadAll(res.Body)
	assert.Regexp(t, "html", string(b))
}

func TestSwaggerYAML(t *testing.T) {
	handler := apiWrapper(swaggerHandler)
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

func TestAdminSwaggerYAML(t *testing.T) {
	handler := apiWrapper(adminSwaggerHandler)
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	res, err := http.Get(fmt.Sprintf("http://%s/admin/api/swagger.yaml", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	b, _ := ioutil.ReadAll(res.Body)
	doc, err := openapi3.NewLoader().LoadFromData(b)
	assert.NoError(t, err)
	err = doc.Validate(context.Background())
	assert.NoError(t, err)
}

func TestSwaggerJSON(t *testing.T) {
	mo := &orchestratormocks.Orchestrator{}
	r := createMuxRouter(mo)
	s := httptest.NewServer(r)
	defer s.Close()

	res, err := http.Get(fmt.Sprintf("http://%s/api/swagger.json", s.Listener.Addr()))
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	b, _ := ioutil.ReadAll(res.Body)
	err = json.Unmarshal(b, &openapi3.T{})
	assert.NoError(t, err)
}

func TestAdminSwaggerJSON(t *testing.T) {
	mo := &orchestratormocks.Orchestrator{}
	r := createAdminMuxRouter(mo)
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
	chl1 <- fmt.Errorf("pop1")

	err := waitForServerStop(chl1, chl2)
	assert.EqualError(t, err, "pop1")

	chl2 <- fmt.Errorf("pop2")
	err = waitForServerStop(chl1, chl2)
	assert.EqualError(t, err, "pop2")

}
