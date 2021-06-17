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
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestInvalidListener(t *testing.T) {
	cp := config.NewPluginConfig("ut")
	initHTTPConfPrefx(cp, 0)
	cp.Set(HTTPConfAddress, "...")
	_, err := newHTTPServer(context.Background(), "ut", mux.NewRouter(), make(chan error), cp)
	assert.Error(t, err)
}

func TestServeFail(t *testing.T) {
	config.Reset()
	cp := config.NewPluginConfig("ut")
	initHTTPConfPrefx(cp, 0)
	errChan := make(chan error)
	hs, err := newHTTPServer(context.Background(), "ut", mux.NewRouter(), errChan, cp)
	assert.NoError(t, err)
	hs.l.Close() // So server will fail
	go hs.serveHTTP(context.Background())
	err = <-errChan
	assert.Error(t, err)
}

func TestMissingCAFile(t *testing.T) {
	cp := config.NewPluginConfig("ut")
	initHTTPConfPrefx(cp, 0)
	cp.Set(HTTPConfTLSCAFile, "badness")
	_, err := newHTTPServer(context.Background(), "ut", mux.NewRouter(), make(chan error), cp)
	assert.Regexp(t, "FF10105", err)
}

func TestBadCAFile(t *testing.T) {
	cp := config.NewPluginConfig("ut")
	initHTTPConfPrefx(cp, 0)
	cp.Set(HTTPConfTLSCAFile, configDir+"/firefly.core.yaml")
	_, err := newHTTPServer(context.Background(), "ut", mux.NewRouter(), make(chan error), cp)
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
	cp := config.NewPluginConfig("ut")
	initHTTPConfPrefx(cp, 0)
	cp.Set(HTTPConfAddress, "127.0.0.1")
	cp.Set(HTTPConfTLSEnabled, true)
	cp.Set(HTTPConfTLSClientAuth, true)
	cp.Set(HTTPConfTLSKeyFile, privateKeyFile.Name())
	cp.Set(HTTPConfTLSCertFile, publicKeyFile.Name())
	cp.Set(HTTPConfTLSCAFile, publicKeyFile.Name())
	cp.Set(HTTPConfPort, 0)
	ctx, cancelCtx := context.WithCancel(context.Background())
	r := mux.NewRouter()
	r.HandleFunc("/test", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(200)
		json.NewEncoder(res).Encode(map[string]interface{}{"hello": "world"})
	})
	errChan := make(chan error)
	hs, err := newHTTPServer(context.Background(), "ut", r, errChan, cp)
	assert.NoError(t, err)
	go hs.serveHTTP(ctx)

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
	httpsAddr := fmt.Sprintf("https://%s/test", hs.l.Addr().String())
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
	err = <-errChan
	assert.NoError(t, err)
}
