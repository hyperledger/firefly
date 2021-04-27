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
	"encoding/json"
	"fmt"
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
		s := createServer(ctx, r)
		err = serveHTTP(ctx, l, s)
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

func createServer(ctx context.Context, r *mux.Router) *http.Server {
	clientAuth := tls.NoClientCert
	if config.GetBool(config.HttpTLSClientAuth) {
		clientAuth = tls.RequireAndVerifyClientCert
	}
	srv := &http.Server{
		Handler:      r,
		WriteTimeout: time.Duration(config.GetUint(config.HttpWriteTimeout)) * time.Second,
		ReadTimeout:  time.Duration(config.GetUint(config.HttpReadTimeout)) * time.Second,
		TLSConfig: &tls.Config{
			ClientAuth: clientAuth,
		},
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			ctx = log.WithLogField(ctx, "r", c.RemoteAddr().String())
			ctx = log.WithLogField(ctx, "req", nanoid.New())
			return ctx
		},
	}
	return srv
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
		err = srv.ServeTLS(listener, config.GetString(config.HttpTLSCertsFile), config.GetString(config.HttpTLSKeyFile))
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

func jsonHandlerFor(route *Route) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		l := log.L(req.Context())
		l.Infof("--> %s %s (%s)", req.Method, req.URL.Path, route.Name)
		input := route.JSONInputValue()
		output := route.JSONOutputValue()
		status := 400
		var err error
		if input != nil {
			err = json.NewDecoder(req.Body).Decode(&input)
		}
		if err == nil {
			status, err = route.JSONHandler(req, input, output)
		}
		if err != nil {
			l.Infof("<-- %s %s ERROR: %s", req.Method, req.URL.Path, err)
			output = RESTError{
				Message: err.Error(),
			}
		}
		l.Infof("<-- %s %s [%d]", req.Method, req.URL.Path, status)
		res.WriteHeader(status)
		err = json.NewEncoder(res).Encode(output)
		if err != nil {
			l.Errorf("Failed to send HTTP response: %s", err)
		}
	}
}

func createMuxRouter() *mux.Router {
	r := mux.NewRouter()
	for _, route := range routes {
		if route.JSONHandler != nil {
			r.HandleFunc(route.Path, jsonHandlerFor(route)).
				HeadersRegexp("Content-Type", "application/json").
				Methods(route.Method)
		}
	}
	return r
}
