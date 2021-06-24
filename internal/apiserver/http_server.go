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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

const (
	// HTTPConfAddress the local address to listen on
	HTTPConfAddress = "address"
	// HTTPConfPublicURL the public address of the node to advertise in the swagger
	HTTPConfPublicURL = "publicURL"
	// HTTPConfPort the local port to listen on for HTTP/Websocket connections
	HTTPConfPort = "port"
	// HTTPConfReadTimeout the write timeout for the HTTP server
	HTTPConfReadTimeout = "readTimeout"
	// HTTPConfWriteTimeout the write timeout for the HTTP server
	HTTPConfWriteTimeout = "writeTimeout"
	// HTTPConfTLSCAFile the TLS certificate authority file for the HTTP server
	HTTPConfTLSCAFile = "tls.caFile"
	// HTTPConfTLSCertFile the TLS certificate file for the HTTP server
	HTTPConfTLSCertFile = "tls.certFile"
	// HTTPConfTLSClientAuth whether the HTTP server requires a mutual TLS connection
	HTTPConfTLSClientAuth = "tls.clientAuth"
	// HTTPConfTLSEnabled whether TLS is enabled for the HTTP server
	HTTPConfTLSEnabled = "tls.enabled"
	// HTTPConfTLSKeyFile the private key file for TLS on the server
	HTTPConfTLSKeyFile = "tls.keyFile"
)

type httpServer struct {
	name        string
	s           *http.Server
	l           net.Listener
	conf        config.Prefix
	onClose     chan error
	tlsEnabled  bool
	tlsCertFile string
	tlsKeyFile  string
}

func initHTTPConfPrefx(prefix config.Prefix, defaultPort int) {
	prefix.AddKnownKey(HTTPConfAddress, "127.0.0.1")
	prefix.AddKnownKey(HTTPConfPublicURL)
	prefix.AddKnownKey(HTTPConfPort, defaultPort)
	prefix.AddKnownKey(HTTPConfReadTimeout, "15s")
	prefix.AddKnownKey(HTTPConfWriteTimeout, "15s")
	prefix.AddKnownKey(HTTPConfTLSCAFile)
	prefix.AddKnownKey(HTTPConfTLSCertFile)
	prefix.AddKnownKey(HTTPConfTLSClientAuth)
	prefix.AddKnownKey(HTTPConfTLSEnabled, false)
	prefix.AddKnownKey(HTTPConfTLSKeyFile)
}

func newHTTPServer(ctx context.Context, name string, r *mux.Router, onClose chan error, conf config.Prefix) (hs *httpServer, err error) {
	hs = &httpServer{
		name:        name,
		onClose:     onClose,
		conf:        conf,
		tlsEnabled:  conf.GetBool(HTTPConfTLSEnabled),
		tlsCertFile: conf.GetString(HTTPConfTLSCertFile),
		tlsKeyFile:  conf.GetString(HTTPConfTLSKeyFile),
	}
	hs.l, err = hs.createListener(ctx)
	if err == nil {
		hs.s, err = hs.createServer(ctx, r)
	}
	return hs, err
}

func (hs *httpServer) createListener(ctx context.Context) (net.Listener, error) {
	listenAddr := fmt.Sprintf("%s:%d", hs.conf.GetString(HTTPConfAddress), hs.conf.GetUint(HTTPConfPort))
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgAPIServerStartFailed, listenAddr)
	}
	log.L(ctx).Infof("%s listening on HTTP %s", hs.name, listener.Addr())
	return listener, err
}

func (hs *httpServer) createServer(ctx context.Context, r *mux.Router) (srv *http.Server, err error) {

	// Support client auth
	clientAuth := tls.NoClientCert
	if hs.conf.GetBool(HTTPConfTLSClientAuth) {
		clientAuth = tls.RequireAndVerifyClientCert
	}

	// Support custom CA file
	var rootCAs *x509.CertPool
	caFile := hs.conf.GetString(HTTPConfTLSCAFile)
	if caFile != "" {
		rootCAs = x509.NewCertPool()
		var caBytes []byte
		caBytes, err = ioutil.ReadFile(caFile)
		if err == nil {
			ok := rootCAs.AppendCertsFromPEM(caBytes)
			if !ok {
				err = i18n.NewError(ctx, i18n.MsgInvalidCAFile)
			}
		}
	} else {
		rootCAs, err = x509.SystemCertPool()
	}

	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgTLSConfigFailed)
	}

	srv = &http.Server{
		Handler:      wrapCorsIfEnabled(ctx, r),
		WriteTimeout: hs.conf.GetDuration(HTTPConfWriteTimeout),
		ReadTimeout:  hs.conf.GetDuration(HTTPConfReadTimeout),
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ClientAuth: clientAuth,
			ClientCAs:  rootCAs,
			RootCAs:    rootCAs,
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				cert := verifiedChains[0][0]
				log.L(ctx).Debugf("Client certificate provided Subject=%s Issuer=%s Expiry=%s", cert.Subject, cert.Issuer, cert.NotAfter)
				return nil
			},
		},
		ConnContext: func(newCtx context.Context, c net.Conn) context.Context {
			l := log.L(ctx).WithField("req", fftypes.ShortID())
			newCtx = log.WithLogger(newCtx, l)
			l.Debugf("New HTTP connection: remote=%s local=%s", c.RemoteAddr().String(), c.LocalAddr().String())
			return newCtx
		},
	}
	return srv, nil
}

func (hs *httpServer) serveHTTP(ctx context.Context) {
	serverEnded := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			log.L(ctx).Infof("API server context cancelled - shutting down")
			hs.s.Close()
		case <-serverEnded:
			return
		}
	}()

	var err error
	if hs.tlsEnabled {
		err = hs.s.ServeTLS(hs.l, hs.tlsCertFile, hs.tlsKeyFile)
	} else {
		err = hs.s.Serve(hs.l)
	}
	if err == http.ErrServerClosed {
		err = nil
	}
	close(serverEnded)
	log.L(ctx).Infof("API server complete")

	hs.onClose <- err
}
