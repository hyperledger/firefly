// Copyright © 2021 Kaleido, Inc.
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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
)

// Serve is the main entry point for the API Server
func Serve(ctx context.Context) error {
	r := createMuxRouter()
	l, err := createListener(ctx)
	if err == nil {
		var s *http.Server
		s, err = createServer(ctx, r)
		if err == nil {
			err = serveHTTP(ctx, l, s)
		}
	}
	return err
}

func createListener(ctx context.Context) (net.Listener, error) {
	listenAddr := fmt.Sprintf("%s:%d", config.GetString(config.HttpAddress), config.GetUint(config.HttpPort))
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgAPIServerStartFailed, listenAddr)
	}
	log.L(ctx).Infof("Listening on HTTP %s", listener.Addr())
	return listener, err
}

func createServer(ctx context.Context, r *mux.Router) (srv *http.Server, err error) {

	// Support client auth
	clientAuth := tls.NoClientCert
	if config.GetBool(config.HttpTLSClientAuth) {
		clientAuth = tls.RequireAndVerifyClientCert
	}

	// Support custom CA file
	var rootCAs *x509.CertPool
	caFile := config.GetString(config.HttpTLSCAFile)
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
		Handler:      r,
		WriteTimeout: time.Duration(config.GetUint(config.HttpWriteTimeout)) * time.Second,
		ReadTimeout:  time.Duration(config.GetUint(config.HttpReadTimeout)) * time.Second,
		TLSConfig: &tls.Config{
			ClientAuth: clientAuth,
			ClientCAs:  rootCAs,
			RootCAs:    rootCAs,
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				cert := verifiedChains[0][0]
				log.L(ctx).Debugf("Client certificate provided Subject=%s Issuer=%s Expiry=%s", cert.Subject, cert.Issuer, cert.NotAfter)
				return nil
			},
		},
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			ctx = log.WithLogField(ctx, "r", c.RemoteAddr().String())
			ctx = log.WithLogField(ctx, "req", nanoid.New())
			return ctx
		},
	}
	return srv, nil
}

func serveHTTP(ctx context.Context, listener net.Listener, srv *http.Server) (err error) {
	serverEnded := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			log.L(ctx).Infof("API server context cancelled - shutting down")
			srv.Close()
		case <-serverEnded:
			return
		}
	}()

	if config.GetBool(config.HttpTLSEnabled) {
		err = srv.ServeTLS(listener, config.GetString(config.HttpTLSCertFile), config.GetString(config.HttpTLSKeyFile))
	} else {
		err = srv.Serve(listener)
	}
	if err == http.ErrServerClosed {
		err = nil
	}
	close(serverEnded)
	log.L(ctx).Infof("API server complete")

	return err
}

func jsonHandler(route *Route) http.HandlerFunc {
	// Check the mandatory parts are ok at startup time
	route.JSONInputValue()
	route.JSONOutputValue()
	return logWrapper(func(res http.ResponseWriter, req *http.Request) (status int, err error) {
		l := log.L(req.Context())
		input := route.JSONInputValue()
		output := route.JSONOutputValue()
		status = 400 // default if fail parsing input
		if input != nil {
			err = json.NewDecoder(req.Body).Decode(&input)
		}
		if err == nil {
			status, err = route.JSONHandler(req, input, output)
		}
		if err == nil {
			res.Header().Add("Content-Type", "application/json")
			res.WriteHeader(status)
			err = json.NewEncoder(res).Encode(output)
			if err != nil {
				err = i18n.WrapError(req.Context(), err, i18n.MsgResponseMarshalError)
				l.Errorf(err.Error())
			}
		}
		return status, err
	})
}

func logWrapper(handler func(res http.ResponseWriter, req *http.Request) (status int, err error)) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		l := log.L(req.Context())
		l.Infof("--> %s %s", req.Method, req.URL.Path)
		status, err := handler(res, req)
		if err != nil {
			if status < 300 {
				status = 500
			} // Ensure we return an error status
			l.Infof("<-- %s %s [%d]!: %s", req.Method, req.URL.Path, status, err)
			res.Header().Add("Content-Type", "application/json")
			res.WriteHeader(status)
			_ = json.NewEncoder(res).Encode(&RESTError{
				Message: err.Error(),
			})
		} else {
			l.Infof("<-- %s %s [%d]", req.Method, req.URL.Path, status)
		}
	}
}

func notFoundHandler(res http.ResponseWriter, req *http.Request) (status int, err error) {
	res.Header().Add("Content-Type", "application/json")
	return 404, i18n.NewError(req.Context(), i18n.Msg404NotFound)
}

func createMuxRouter() *mux.Router {
	r := mux.NewRouter()
	for _, route := range routes {
		if route.JSONHandler != nil {
			r.HandleFunc(route.Path, jsonHandler(route)).
				HeadersRegexp("Content-Type", "application/json").
				Methods(route.Method)
		}
	}
	r.NotFoundHandler = logWrapper(notFoundHandler)
	return r
}
