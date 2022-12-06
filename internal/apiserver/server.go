// Copyright © 2022 Kaleido, Inc.
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
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ghodss/yaml"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gorilla/mux"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffapi"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/httpserver"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/events/eifactory"
	"github.com/hyperledger/firefly/internal/events/websockets"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/namespace"
	"github.com/hyperledger/firefly/internal/orchestrator"
)

var (
	spiConfig     = config.RootSection("spi")
	apiConfig     = config.RootSection("http")
	metricsConfig = config.RootSection("metrics")
	corsConfig    = config.RootSection("cors")
)

// Server is the external interface for the API Server
type Server interface {
	Serve(ctx context.Context, mgr namespace.Manager) error
}

type apiServer struct {
	// Defaults set with config
	apiTimeout     time.Duration
	apiMaxTimeout  time.Duration
	metricsEnabled bool
	ffiSwaggerGen  FFISwaggerGen
}

func InitConfig() {
	httpserver.InitHTTPConfig(apiConfig, 5000)
	httpserver.InitHTTPConfig(spiConfig, 5001)
	httpserver.InitHTTPConfig(metricsConfig, 6000)
	httpserver.InitCORSConfig(corsConfig)
	initMetricsConfig(metricsConfig)
}

func NewAPIServer() Server {
	return &apiServer{
		apiTimeout:     config.GetDuration(coreconfig.APIRequestTimeout),
		apiMaxTimeout:  config.GetDuration(coreconfig.APIRequestMaxTimeout),
		metricsEnabled: config.GetBool(coreconfig.MetricsEnabled),
		ffiSwaggerGen:  NewFFISwaggerGen(),
	}
}

// Serve is the main entry point for the API Server
func (as *apiServer) Serve(ctx context.Context, mgr namespace.Manager) (err error) {
	httpErrChan := make(chan error)
	spiErrChan := make(chan error)
	metricsErrChan := make(chan error)

	apiHTTPServer, err := httpserver.NewHTTPServer(ctx, "api", as.createMuxRouter(ctx, mgr), httpErrChan, apiConfig, corsConfig, &httpserver.ServerOptions{
		MaximumRequestTimeout: as.apiMaxTimeout,
	})
	if err != nil {
		return err
	}
	go apiHTTPServer.ServeHTTP(ctx)

	if config.GetBool(coreconfig.SPIEnabled) {
		spiHTTPServer, err := httpserver.NewHTTPServer(ctx, "spi", as.createAdminMuxRouter(mgr), spiErrChan, spiConfig, corsConfig, &httpserver.ServerOptions{
			MaximumRequestTimeout: as.apiMaxTimeout,
		})
		if err != nil {
			return err
		}
		go spiHTTPServer.ServeHTTP(ctx)
	} else if config.GetBool(coreconfig.LegacyAdminEnabled) {
		log.L(ctx).Warnf("Your config includes an 'admin' section, which should be renamed to 'spi' - SPI server will not be enabled until this is corrected")
	}

	if as.metricsEnabled {
		metricsHTTPServer, err := httpserver.NewHTTPServer(ctx, "metrics", as.createMetricsMuxRouter(), metricsErrChan, metricsConfig, corsConfig, &httpserver.ServerOptions{
			MaximumRequestTimeout: as.apiMaxTimeout,
		})
		if err != nil {
			return err
		}
		go metricsHTTPServer.ServeHTTP(ctx)
	}

	return as.waitForServerStop(httpErrChan, spiErrChan, metricsErrChan)
}

func (as *apiServer) waitForServerStop(httpErrChan, spiErrChan, metricsErrChan chan error) error {
	select {
	case err := <-httpErrChan:
		return err
	case err := <-spiErrChan:
		return err
	case err := <-metricsErrChan:
		return err
	}
}

func (as *apiServer) getPublicURL(conf config.Section, pathPrefix string) string {
	publicURL := conf.GetString(httpserver.HTTPConfPublicURL)
	if publicURL == "" {
		proto := "https"
		if !conf.GetBool(httpserver.HTTPConfTLSEnabled) {
			proto = "http"
		}
		publicURL = fmt.Sprintf("%s://%s:%s", proto, conf.GetString(httpserver.HTTPConfAddress), conf.GetString(httpserver.HTTPConfPort))
	}
	if pathPrefix != "" {
		publicURL += "/" + pathPrefix
	}
	return publicURL
}

func (as *apiServer) swaggerGenConf(apiBaseURL string) *ffapi.Options {
	return &ffapi.Options{
		BaseURL:                   apiBaseURL,
		Title:                     "FireFly",
		Version:                   "1.0",
		PanicOnMissingDescription: config.GetBool(coreconfig.APIOASPanicOnMissingDescription),
		DefaultRequestTimeout:     config.GetDuration(coreconfig.APIRequestTimeout),
		APIDefaultFilterLimit:     config.GetString(coreconfig.APIDefaultFilterLimit),
		APIMaxFilterLimit:         config.GetUint(coreconfig.APIMaxFilterLimit),
		APIMaxFilterSkip:          config.GetUint(coreconfig.APIMaxFilterSkip),
	}
}

func (as *apiServer) swaggerHandler(generator func(req *http.Request) (*openapi3.T, error)) func(res http.ResponseWriter, req *http.Request) (status int, err error) {
	return func(res http.ResponseWriter, req *http.Request) (status int, err error) {
		vars := mux.Vars(req)
		doc, err := generator(req)
		if err != nil {
			return 500, err
		}
		if vars["ext"] == ".json" {
			res.Header().Add("Content-Type", "application/json")
			b, _ := json.Marshal(&doc)
			_, _ = res.Write(b)
		} else {
			res.Header().Add("Content-Type", "application/x-yaml")
			b, _ := yaml.Marshal(&doc)
			_, _ = res.Write(b)
		}
		return 200, nil
	}
}

func (as *apiServer) swaggerGenerator(routes []*ffapi.Route, apiBaseURL string) func(req *http.Request) (*openapi3.T, error) {
	swg := ffapi.NewSwaggerGen(as.swaggerGenConf(apiBaseURL))
	return func(req *http.Request) (*openapi3.T, error) {
		return swg.Generate(req.Context(), routes), nil
	}
}

func (as *apiServer) contractSwaggerGenerator(mgr namespace.Manager, apiBaseURL string) func(req *http.Request) (*openapi3.T, error) {
	return func(req *http.Request) (*openapi3.T, error) {
		vars := mux.Vars(req)
		or := mgr.Orchestrator(vars["ns"])
		if or == nil {
			return nil, i18n.NewError(req.Context(), coremsgs.MsgNamespaceDoesNotExist)
		}
		cm := or.Contracts()
		api, err := cm.GetContractAPI(req.Context(), apiBaseURL, vars["apiName"])
		if err != nil {
			return nil, err
		} else if api == nil || api.Interface == nil {
			return nil, i18n.NewError(req.Context(), coremsgs.Msg404NoResult)
		}

		ffi, err := cm.GetFFIByIDWithChildren(req.Context(), api.Interface.ID)
		if err != nil {
			return nil, err
		}

		baseURL := fmt.Sprintf("%s/namespaces/%s/apis/%s", apiBaseURL, vars["ns"], vars["apiName"])
		return as.ffiSwaggerGen.Generate(req.Context(), baseURL, api, ffi), nil
	}
}

func getOrchestrator(ctx context.Context, mgr namespace.Manager, tag string, r *ffapi.APIRequest) (or orchestrator.Orchestrator, err error) {
	switch tag {
	case routeTagDefaultNamespace:
		or = mgr.Orchestrator(config.GetString(coreconfig.NamespacesDefault))
	case routeTagNonDefaultNamespace:
		vars := mux.Vars(r.Req)
		if ns, ok := vars["ns"]; ok {
			or = mgr.Orchestrator(ns)
		}
	default:
		return nil, nil
	}
	if or == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceDoesNotExist)
	}
	return or, nil
}

func (as *apiServer) routeHandler(hf *ffapi.HandlerFactory, mgr namespace.Manager, apiBaseURL string, route *ffapi.Route) http.HandlerFunc {
	// We extend the base ffapi functionality, with standardized DB filter support for all core resources.
	// We also pass the Orchestrator context through
	ce := route.Extensions.(*coreExtensions)
	route.JSONHandler = func(r *ffapi.APIRequest) (output interface{}, err error) {
		or, err := getOrchestrator(r.Req.Context(), mgr, route.Tag, r)
		if err != nil {
			return nil, err
		}

		// Authorize the request
		authReq := &fftypes.AuthReq{
			Method: r.Req.Method,
			URL:    r.Req.URL,
			Header: r.Req.Header,
		}
		if or != nil {
			if err := or.Authorize(r.Req.Context(), authReq); err != nil {
				return nil, err
			}
		}

		if ce.EnabledIf != nil && !ce.EnabledIf(or) {
			return nil, i18n.NewError(r.Req.Context(), coremsgs.MsgActionNotSupported)
		}

		cr := &coreRequest{
			mgr:        mgr,
			or:         or,
			ctx:        r.Req.Context(),
			apiBaseURL: apiBaseURL,
		}
		return ce.CoreJSONHandler(r, cr)
	}
	if ce.CoreFormUploadHandler != nil {
		route.FormUploadHandler = func(r *ffapi.APIRequest) (output interface{}, err error) {
			or, err := getOrchestrator(r.Req.Context(), mgr, route.Tag, r)
			if err != nil {
				return nil, err
			}
			if ce.EnabledIf != nil && !ce.EnabledIf(or) {
				return nil, i18n.NewError(r.Req.Context(), coremsgs.MsgActionNotSupported)
			}

			cr := &coreRequest{
				mgr:        mgr,
				or:         or,
				ctx:        r.Req.Context(),
				apiBaseURL: apiBaseURL,
			}
			return ce.CoreFormUploadHandler(r, cr)
		}
	}
	return hf.RouteHandler(route)
}

func (as *apiServer) handlerFactory() *ffapi.HandlerFactory {
	return &ffapi.HandlerFactory{
		DefaultFilterLimit:    uint64(config.GetUint(coreconfig.APIDefaultFilterLimit)),
		MaxFilterLimit:        uint64(config.GetUint(coreconfig.APIMaxFilterLimit)),
		MaxFilterSkip:         uint64(config.GetUint(coreconfig.APIMaxFilterSkip)),
		DefaultRequestTimeout: config.GetDuration(coreconfig.APIRequestTimeout),
		MaxTimeout:            config.GetDuration(coreconfig.APIRequestMaxTimeout),
	}
}

func (as *apiServer) createMuxRouter(ctx context.Context, mgr namespace.Manager) *mux.Router {
	r := mux.NewRouter()
	hf := as.handlerFactory()

	if as.metricsEnabled {
		r.Use(metrics.GetRestServerInstrumentation().Middleware)
	}

	publicURL := as.getPublicURL(apiConfig, "")
	apiBaseURL := fmt.Sprintf("%s/api/v1", publicURL)
	for _, route := range routes {
		if ce, ok := route.Extensions.(*coreExtensions); ok {
			if ce.CoreJSONHandler != nil {
				r.HandleFunc(fmt.Sprintf("/api/v1/%s", route.Path), as.routeHandler(hf, mgr, apiBaseURL, route)).
					Methods(route.Method)
			}
		}
	}

	r.HandleFunc(`/api/v1/namespaces/{ns}/apis/{apiName}/api/swagger{ext:\.yaml|\.json|}`, hf.APIWrapper(as.swaggerHandler(as.contractSwaggerGenerator(mgr, apiBaseURL))))
	r.HandleFunc(`/api/v1/namespaces/{ns}/apis/{apiName}/api`, func(rw http.ResponseWriter, req *http.Request) {
		url := req.URL.String() + "/swagger.yaml"
		handler := hf.APIWrapper(hf.SwaggerUIHandler(url))
		handler(rw, req)
	})

	r.HandleFunc(`/api/swagger{ext:\.yaml|\.json|}`, hf.APIWrapper(as.swaggerHandler(as.swaggerGenerator(routes, apiBaseURL))))
	r.HandleFunc(`/api`, hf.APIWrapper(hf.SwaggerUIHandler(publicURL+"/api/swagger.yaml")))
	r.HandleFunc(`/favicon{any:.*}.png`, favIcons)

	ws, _ := eifactory.GetPlugin(ctx, "websockets")
	ws.(*websockets.WebSockets).SetAuthorizer(mgr)
	r.HandleFunc(`/ws`, ws.(*websockets.WebSockets).ServeHTTP)

	uiPath := config.GetString(coreconfig.UIPath)
	if uiPath != "" && config.GetBool(coreconfig.UIEnabled) {
		r.PathPrefix(`/ui`).Handler(newStaticHandler(uiPath, "index.html", `/ui`))
	}

	r.NotFoundHandler = hf.APIWrapper(as.notFoundHandler)
	return r
}

func (as *apiServer) notFoundHandler(res http.ResponseWriter, req *http.Request) (status int, err error) {
	res.Header().Add("Content-Type", "application/json")
	return 404, i18n.NewError(req.Context(), coremsgs.Msg404NotFound)
}

func (as *apiServer) spiWSHandler(mgr namespace.Manager) http.HandlerFunc {
	// The SPI events listener will be initialized when we start, so we access it it from Orchestrator on demand
	return func(w http.ResponseWriter, r *http.Request) {
		mgr.SPIEvents().ServeHTTPWebSocketListener(w, r)
	}
}

func (as *apiServer) createAdminMuxRouter(mgr namespace.Manager) *mux.Router {
	r := mux.NewRouter()
	if as.metricsEnabled {
		r.Use(metrics.GetAdminServerInstrumentation().Middleware)
	}
	hf := as.handlerFactory()

	publicURL := as.getPublicURL(spiConfig, "spi")
	apiBaseURL := fmt.Sprintf("%s/v1", publicURL)
	for _, route := range spiRoutes {
		if ce, ok := route.Extensions.(*coreExtensions); ok {
			if ce.CoreJSONHandler != nil {
				r.HandleFunc(fmt.Sprintf("/spi/v1/%s", route.Path), as.routeHandler(hf, mgr, apiBaseURL, route)).
					Methods(route.Method)
			}
		}
	}
	r.HandleFunc(`/spi/swagger{ext:\.yaml|\.json|}`, hf.APIWrapper(as.swaggerHandler(as.swaggerGenerator(spiRoutes, apiBaseURL))))
	r.HandleFunc(`/spi`, hf.APIWrapper(hf.SwaggerUIHandler(publicURL+"/swagger.yaml")))
	r.HandleFunc(`/favicon{any:.*}.png`, favIcons)

	r.HandleFunc(`/spi/ws`, as.spiWSHandler(mgr))

	return r
}

func (as *apiServer) createMetricsMuxRouter() *mux.Router {
	r := mux.NewRouter()

	r.Path(config.GetString(coreconfig.MetricsPath)).Handler(promhttp.InstrumentMetricHandler(metrics.Registry(),
		promhttp.HandlerFor(metrics.Registry(), promhttp.HandlerOpts{})))

	return r
}

func syncRetcode(isSync bool) int {
	if isSync {
		return http.StatusOK
	}
	return http.StatusAccepted
}
