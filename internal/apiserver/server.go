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
	"encoding/json"
	"fmt"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/gorilla/mux"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/events/eifactory"
	"github.com/hyperledger/firefly/internal/events/websockets"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/oapispec"
	"github.com/hyperledger/firefly/internal/orchestrator"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/prometheus/client_golang/prometheus"
	muxprom "gitlab.com/msvechla/mux-prometheus/pkg/middleware"
)

var ffcodeExtractor = regexp.MustCompile(`^(FF\d+):`)

var (
	adminConfigPrefix   = config.NewPluginConfig("admin")
	apiConfigPrefix     = config.NewPluginConfig("http")
	metricsConfigPrefix = config.NewPluginConfig("metrics")
)

// Server is the external interface for the API Server
type Server interface {
	Serve(ctx context.Context, o orchestrator.Orchestrator) error
}

type apiServer struct {
	// Defaults set with config
	defaultFilterLimit uint64
	maxFilterLimit     uint64
	maxFilterSkip      uint64
	apiTimeout         time.Duration
	apiMaxTimeout      time.Duration
	metricsEnabled     bool
}

func InitConfig() {
	initHTTPConfPrefx(apiConfigPrefix, 5000)
	initHTTPConfPrefx(adminConfigPrefix, 5001)
	initHTTPConfPrefx(metricsConfigPrefix, 6000)
	initMetricsConfPrefix(metricsConfigPrefix)
}

func NewAPIServer() Server {
	return &apiServer{
		defaultFilterLimit: uint64(config.GetUint(config.APIDefaultFilterLimit)),
		maxFilterLimit:     uint64(config.GetUint(config.APIMaxFilterLimit)),
		maxFilterSkip:      uint64(config.GetUint(config.APIMaxFilterSkip)),
		apiTimeout:         config.GetDuration(config.APIRequestTimeout),
		apiMaxTimeout:      config.GetDuration(config.APIRequestMaxTimeout),
		metricsEnabled:     config.GetBool(config.MetricsEnabled),
	}
}

// Serve is the main entry point for the API Server
func (as *apiServer) Serve(ctx context.Context, o orchestrator.Orchestrator) (err error) {
	httpErrChan := make(chan error)
	adminErrChan := make(chan error)
	metricsErrChan := make(chan error)

	if !o.IsPreInit() {
		apiHTTPServer, err := newHTTPServer(ctx, "api", as.createMuxRouter(ctx, o), httpErrChan, apiConfigPrefix)
		if err != nil {
			return err
		}
		go apiHTTPServer.serveHTTP(ctx)
	}

	if config.GetBool(config.AdminEnabled) {
		adminHTTPServer, err := newHTTPServer(ctx, "admin", as.createAdminMuxRouter(o), adminErrChan, adminConfigPrefix)
		if err != nil {
			return err
		}
		go adminHTTPServer.serveHTTP(ctx)
	}

	if as.metricsEnabled {
		metricsHTTPServer, err := newHTTPServer(ctx, "metrics", as.createMetricsMuxRouter(ctx), metricsErrChan, metricsConfigPrefix)
		if err != nil {
			return err
		}
		go metricsHTTPServer.serveHTTP(ctx)
	}

	return as.waitForServerStop(httpErrChan, adminErrChan, metricsErrChan)
}

func (as *apiServer) waitForServerStop(httpErrChan, adminErrChan, metricsErrChan chan error) error {
	select {
	case err := <-httpErrChan:
		return err
	case err := <-adminErrChan:
		return err
	case err := <-metricsErrChan:
		return err
	}
}

type multipartState struct {
	mpr        *multipart.Reader
	formParams map[string]string
	part       *fftypes.Multipart
	close      func()
}

func (as *apiServer) getFilePart(req *http.Request) (*multipartState, error) {

	formParams := make(map[string]string)
	ctx := req.Context()
	l := log.L(ctx)
	mpr, err := req.MultipartReader()
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgMultiPartFormReadError)
	}
	for {
		part, err := mpr.NextPart()
		if err != nil {
			return nil, i18n.WrapError(ctx, err, i18n.MsgMultiPartFormReadError)
		}
		if part.FileName() == "" {
			value, _ := ioutil.ReadAll(part)
			formParams[part.FormName()] = string(value)
		} else {
			l.Debugf("Processing multi-part upload. Field='%s' Filename='%s'", part.FormName(), part.FileName())
			mp := &fftypes.Multipart{
				Data:     part,
				Filename: part.FileName(),
				Mimetype: part.Header.Get("Content-Disposition"),
			}
			return &multipartState{
				mpr:        mpr,
				formParams: formParams,
				part:       mp,
				close:      func() { _ = part.Close() },
			}, nil
		}
	}
}

func (as *apiServer) getParams(req *http.Request, route *oapispec.Route) (queryParams, pathParams map[string]string) {
	queryParams = make(map[string]string)
	pathParams = make(map[string]string)
	if len(route.PathParams) > 0 {
		v := mux.Vars(req)
		for _, pp := range route.PathParams {
			pathParams[pp.Name] = v[pp.Name]
		}
	}
	for _, qp := range route.QueryParams {
		val, exists := req.URL.Query()[qp.Name]
		if qp.IsBool {
			if exists && (len(val) == 0 || val[0] == "" || strings.EqualFold(val[0], "true")) {
				val = []string{"true"}
			} else {
				val = []string{"false"}
			}
		}
		if exists && len(val) > 0 {
			queryParams[qp.Name] = val[0]
		}
	}
	return queryParams, pathParams
}

func (as *apiServer) routeHandler(o orchestrator.Orchestrator, route *oapispec.Route) http.HandlerFunc {
	// Check the mandatory parts are ok at startup time
	return as.apiWrapper(func(res http.ResponseWriter, req *http.Request) (int, error) {

		var jsonInput interface{}
		if route.JSONInputValue != nil {
			jsonInput = route.JSONInputValue()
		}
		var queryParams, pathParams map[string]string
		var multipart *multipartState
		contentType := req.Header.Get("Content-Type")
		var err error
		if req.Method != http.MethodGet && req.Method != http.MethodDelete {
			switch {
			case strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data") && route.FormUploadHandler != nil:
				multipart, err = as.getFilePart(req)
				if err != nil {
					return 400, err
				}
				defer multipart.close()
			case strings.HasPrefix(strings.ToLower(contentType), "application/json"):
				if jsonInput != nil {
					err = json.NewDecoder(req.Body).Decode(&jsonInput)
				}
			default:
				return 415, i18n.NewError(req.Context(), i18n.MsgInvalidContentType)
			}
		}

		var filter database.AndFilter
		var status = 400 // if fail parsing input
		var output interface{}
		if err == nil {
			queryParams, pathParams = as.getParams(req, route)
			if route.FilterFactory != nil {
				filter, err = as.buildFilter(req, route.FilterFactory)
			}
		}

		if err == nil {
			r := &oapispec.APIRequest{
				Ctx:           req.Context(),
				Or:            o,
				Req:           req,
				PP:            pathParams,
				QP:            queryParams,
				Filter:        filter,
				Input:         jsonInput,
				SuccessStatus: http.StatusOK,
			}
			if len(route.JSONOutputCodes) > 0 {
				r.SuccessStatus = route.JSONOutputCodes[0]
			}
			if multipart != nil {
				r.FP = multipart.formParams
				r.Part = multipart.part
				output, err = route.FormUploadHandler(r)
			} else {
				output, err = route.JSONHandler(r)
			}
			status = r.SuccessStatus // Can be updated by the route
		}
		if err == nil && multipart != nil {
			// Catch the case that someone puts form fields after the file in a multi-part body.
			// We don't support that, so that we can stream through the core rather than having
			// to hold everything in memory.
			trailing, expectEOF := multipart.mpr.NextPart()
			if expectEOF == nil {
				err = i18n.NewError(req.Context(), i18n.MsgFieldsAfterFile, trailing.FormName())
			}
		}
		if err == nil {
			status, err = as.handleOutput(req.Context(), res, status, output)
		}
		return status, err
	})
}

func (as *apiServer) handleOutput(ctx context.Context, res http.ResponseWriter, status int, output interface{}) (int, error) {
	vOutput := reflect.ValueOf(output)
	outputKind := vOutput.Kind()
	isPointer := outputKind == reflect.Ptr
	invalid := outputKind == reflect.Invalid
	isNil := output == nil || invalid || (isPointer && vOutput.IsNil())
	var reader io.ReadCloser
	var marshalErr error
	if !isNil && vOutput.CanInterface() {
		reader, _ = vOutput.Interface().(io.ReadCloser)
	}
	switch {
	case isNil:
		if status != 204 {
			return 404, i18n.NewError(ctx, i18n.Msg404NoResult)
		}
		res.WriteHeader(204)
	case reader != nil:
		defer reader.Close()
		res.Header().Add("Content-Type", "application/octet-stream")
		res.WriteHeader(status)
		_, marshalErr = io.Copy(res, reader)
	default:
		res.Header().Add("Content-Type", "application/json")
		res.WriteHeader(status)
		marshalErr = json.NewEncoder(res).Encode(output)
	}
	if marshalErr != nil {
		err := i18n.WrapError(ctx, marshalErr, i18n.MsgResponseMarshalError)
		log.L(ctx).Errorf(err.Error())
		return 500, err
	}
	return status, nil
}

func (as *apiServer) getTimeout(req *http.Request) time.Duration {
	// Configure a server-side timeout on each request, to try and avoid cases where the API requester
	// times out, and we continue to churn indefinitely processing the request.
	// Long-running processes should be dispatched asynchronously (API returns 202 Accepted asap),
	// and the caller can either listen on the websocket for updates, or poll the status of the affected object.
	// This is dependent on the context being passed down through to all blocking operations down the stack
	// (while avoiding passing the context to asynchronous tasks that are dispatched as a result of the request)
	reqTimeout := as.apiTimeout
	reqTimeoutHeader := req.Header.Get("Request-Timeout")
	if reqTimeoutHeader != "" {
		customTimeout, err := fftypes.ParseDurationString(reqTimeoutHeader, time.Second /* default is seconds */)
		if err != nil {
			log.L(req.Context()).Warnf("Invalid Request-Timeout header '%s': %s", reqTimeoutHeader, err)
		} else {
			reqTimeout = time.Duration(customTimeout)
			if reqTimeout > as.apiMaxTimeout {
				reqTimeout = as.apiMaxTimeout
			}
		}
	}
	return reqTimeout
}

func (as *apiServer) apiWrapper(handler func(res http.ResponseWriter, req *http.Request) (status int, err error)) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		reqTimeout := as.getTimeout(req)
		ctx, cancel := context.WithTimeout(req.Context(), reqTimeout)
		httpReqID := fftypes.ShortID()
		ctx = log.WithLogField(ctx, "httpreq", httpReqID)
		req = req.WithContext(ctx)
		defer cancel()

		// Wrap the request itself in a log wrapper, that gives minimal request/response and timing info
		l := log.L(ctx)
		l.Infof("--> %s %s", req.Method, req.URL.Path)
		startTime := time.Now()
		status, err := handler(res, req)
		durationMS := float64(time.Since(startTime)) / float64(time.Millisecond)
		if err != nil {

			// Routers don't need to tweak the status code when sending errors.
			// .. either the FF12345 error they raise is mapped to a status hint
			ffcodeExtract := ffcodeExtractor.FindStringSubmatch(err.Error())
			if len(ffcodeExtract) >= 2 {
				if statusHint, ok := i18n.GetStatusHint(ffcodeExtract[1]); ok {
					status = statusHint
				}
			}

			// If the context is done, we wrap in 408
			if status != http.StatusRequestTimeout {
				select {
				case <-ctx.Done():
					l.Errorf("Request failed and context is closed. Returning %d (overriding %d): %s", http.StatusRequestTimeout, status, err)
					status = http.StatusRequestTimeout
					err = i18n.WrapError(ctx, err, i18n.MsgRequestTimeout, httpReqID, durationMS)
				default:
				}
			}

			// ... or we default to 500
			if status < 300 {
				status = 500
			}
			l.Infof("<-- %s %s [%d] (%.2fms): %s", req.Method, req.URL.Path, status, durationMS, err)
			res.Header().Add("Content-Type", "application/json")
			res.WriteHeader(status)
			_ = json.NewEncoder(res).Encode(&fftypes.RESTError{
				Error: err.Error(),
			})
		} else {
			l.Infof("<-- %s %s [%d] (%.2fms)", req.Method, req.URL.Path, status, durationMS)
		}
	}
}

func (as *apiServer) notFoundHandler(res http.ResponseWriter, req *http.Request) (status int, err error) {
	res.Header().Add("Content-Type", "application/json")
	return 404, i18n.NewError(req.Context(), i18n.Msg404NotFound)
}

func (as *apiServer) swaggerUIHandler(url string) func(res http.ResponseWriter, req *http.Request) (status int, err error) {
	return func(res http.ResponseWriter, req *http.Request) (status int, err error) {
		res.Header().Add("Content-Type", "text/html")
		_, _ = res.Write(oapispec.SwaggerUIHTML(req.Context(), url))
		return 200, nil
	}
}

func (as *apiServer) getPublicURL(conf config.Prefix, pathPrefix string) string {
	publicURL := conf.GetString(HTTPConfPublicURL)
	if publicURL == "" {
		proto := "https"
		if !conf.GetBool(HTTPConfTLSEnabled) {
			proto = "http"
		}
		publicURL = fmt.Sprintf("%s://%s:%s", proto, conf.GetString(HTTPConfAddress), conf.GetString(HTTPConfPort))
	}
	if pathPrefix != "" {
		publicURL += "/" + pathPrefix
	}
	return publicURL
}

func (as *apiServer) swaggerHandler(routes []*oapispec.Route, url string) func(res http.ResponseWriter, req *http.Request) (status int, err error) {
	return func(res http.ResponseWriter, req *http.Request) (status int, err error) {
		vars := mux.Vars(req)
		if vars["ext"] == ".json" {
			res.Header().Add("Content-Type", "application/json")
			doc := oapispec.SwaggerGen(req.Context(), routes, url)
			b, _ := json.Marshal(&doc)
			_, _ = res.Write(b)
		} else {
			res.Header().Add("Content-Type", "application/x-yaml")
			doc := oapispec.SwaggerGen(req.Context(), routes, url)
			b, _ := yaml.Marshal(&doc)
			_, _ = res.Write(b)
		}
		return 200, nil
	}
}

func (as *apiServer) configurePrometheusInstrumentation(namespace, subsystem string, r *mux.Router) {
	if as.metricsEnabled {
		instrumentation := muxprom.NewCustomInstrumentation(
			true,
			namespace,
			subsystem,
			prometheus.DefBuckets,
			map[string]string{},
			metrics.Registry(),
		)
		r.Use(instrumentation.Middleware)
	}
}

func (as *apiServer) createMuxRouter(ctx context.Context, o orchestrator.Orchestrator) *mux.Router {
	r := mux.NewRouter()
	as.configurePrometheusInstrumentation("apiserver", "rest", r)

	for _, route := range routes {
		if route.JSONHandler != nil {
			r.HandleFunc(fmt.Sprintf("/api/v1/%s", route.Path), as.routeHandler(o, route)).
				Methods(route.Method)
		}
	}
	ws, _ := eifactory.GetPlugin(ctx, "websockets")
	publicURL := as.getPublicURL(apiConfigPrefix, "")
	r.HandleFunc(`/api/swagger{ext:\.yaml|\.json|}`, as.apiWrapper(as.swaggerHandler(routes, publicURL)))
	r.HandleFunc(`/api`, as.apiWrapper(as.swaggerUIHandler(publicURL)))
	r.HandleFunc(`/favicon{any:.*}.png`, favIcons)

	r.HandleFunc(`/ws`, ws.(*websockets.WebSockets).ServeHTTP)

	uiPath := config.GetString(config.UIPath)
	if uiPath != "" && config.GetBool(config.UIEnabled) {
		r.PathPrefix(`/ui`).Handler(newStaticHandler(uiPath, "index.html", `/ui`))
	}

	r.NotFoundHandler = as.apiWrapper(as.notFoundHandler)
	return r
}

func (as *apiServer) createAdminMuxRouter(o orchestrator.Orchestrator) *mux.Router {
	r := mux.NewRouter()
	as.configurePrometheusInstrumentation("apiserver", "admin", r)

	for _, route := range adminRoutes {
		if route.JSONHandler != nil {
			r.HandleFunc(fmt.Sprintf("/admin/api/v1/%s", route.Path), as.routeHandler(o, route)).
				Methods(route.Method)
		}
	}
	publicURL := as.getPublicURL(adminConfigPrefix, "admin")
	r.HandleFunc(`/admin/api/swagger{ext:\.yaml|\.json|}`, as.apiWrapper(as.swaggerHandler(adminRoutes, publicURL)))
	r.HandleFunc(`/admin/api`, as.apiWrapper(as.swaggerUIHandler(publicURL)))
	r.HandleFunc(`/favicon{any:.*}.png`, favIcons)

	return r
}

func (as *apiServer) createMetricsMuxRouter(ctx context.Context) *mux.Router {
	r := mux.NewRouter()

	r.Path(config.GetString(config.MetricsPath)).Handler(promhttp.InstrumentMetricHandler(metrics.Registry(),
		promhttp.HandlerFor(metrics.Registry(), promhttp.HandlerOpts{})))

	return r
}
