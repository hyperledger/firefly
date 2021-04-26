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
	"github.com/kaleido-io/firefly/internal/log"
)

// Serve is the main entry point for the API Server
func Serve(ctx context.Context) error {
	r := mux.NewRouter()

	for _, route := range routes {
		r.HandleFunc(route.Path, func(res http.ResponseWriter, req *http.Request) {
			l := log.L(req.Context())
			l.Infof("--> %s %s", req.Method, req.URL.Path)
			input := route.JSONInputValue()
			output := route.JSONOutputValue()
			status := 500
			err := json.NewDecoder(req.Body).Decode(&input)
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
		}).
			Methods(route.Method)
	}

	listenAddr := fmt.Sprintf("%s:%d", config.GetString(config.HttpPort), config.GetUint(config.HttpPort))
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("Unable to start listener on %s: %s", listenAddr, err)
	}
	log.L(ctx).Infof("Listening on HTTP %s", listener.Addr())

	clientAuth := tls.NoClientCert
	if config.GetBool(config.HttpTLSClientAuth) {
		clientAuth = tls.RequireAndVerifyClientCert
	}
	srv := &http.Server{
		Handler:      r,
		Addr:         listenAddr,
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

	if config.GetBool(config.HttpTLSEnabled) {
		err = srv.ServeTLS(listener, config.GetString(config.HttpTLSCertsFile), config.GetString(config.HttpTLSKeyFile))
	} else {
		err = srv.Serve(listener)
	}

	return err

}
