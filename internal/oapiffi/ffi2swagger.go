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

package oapiffi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hyperledger/firefly/internal/oapispec"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type FFISwaggerGen interface {
	Generate(ctx context.Context, baseURL string, api *fftypes.ContractAPI, ffi *fftypes.FFI) *openapi3.T
}

// ffiSwaggerGen generates OpenAPI3 (Swagger) definitions for FFIs
type ffiSwaggerGen struct {
}

func NewFFISwaggerGen() FFISwaggerGen {
	return &ffiSwaggerGen{}
}

func (og *ffiSwaggerGen) Generate(ctx context.Context, baseURL string, api *fftypes.ContractAPI, ffi *fftypes.FFI) (swagger *openapi3.T) {
	hasLocation := !api.Location.IsNil()

	routes := []*oapispec.Route{}
	for _, method := range ffi.Methods {
		routes = og.addMethod(routes, method, hasLocation)
	}
	for _, event := range ffi.Events {
		routes = og.addEvent(routes, event, hasLocation)
	}

	return oapispec.SwaggerGen(ctx, routes, &oapispec.SwaggerGenConfig{
		Title:       ffi.Name,
		Version:     ffi.Version,
		Description: ffi.Description,
		BaseURL:     baseURL,
	})
}

func (og *ffiSwaggerGen) addMethod(routes []*oapispec.Route, method *fftypes.FFIMethod, hasLocation bool) []*oapispec.Route {
	routes = append(routes, &oapispec.Route{
		Name:             fmt.Sprintf("invoke_%s", method.Pathname),
		Path:             fmt.Sprintf("invoke/%s", method.Pathname), // must match a route defined in apiserver routes!
		Method:           http.MethodPost,
		JSONInputSchema:  func(ctx context.Context) string { return contractCallJSONSchema(&method.Params, hasLocation).String() },
		JSONOutputSchema: func(ctx context.Context) string { return ffiParamsJSONSchema(&method.Returns).String() },
		JSONOutputCodes:  []int{http.StatusOK},
	})
	routes = append(routes, &oapispec.Route{
		Name:             fmt.Sprintf("query_%s", method.Pathname),
		Path:             fmt.Sprintf("query/%s", method.Pathname), // must match a route defined in apiserver routes!
		Method:           http.MethodGet,
		JSONOutputSchema: func(ctx context.Context) string { return ffiParamsJSONSchema(&method.Returns).String() },
		JSONOutputCodes:  []int{http.StatusOK},
	})
	return routes
}

func (og *ffiSwaggerGen) addEvent(routes []*oapispec.Route, event *fftypes.FFIEvent, hasLocation bool) []*oapispec.Route {
	// If the API has a location specified, there are no fields left to specify in the request body.
	// Instead of masking them all (which causes Swagger UI some issues), explicitly set the schema to an empty object.
	var schema func(ctx context.Context) string
	if hasLocation {
		schema = func(ctx context.Context) string { return `{"type": "object"}` }
	}
	return append(routes, &oapispec.Route{
		Name:            fmt.Sprintf("subscribe_%s", event.Pathname),
		Path:            fmt.Sprintf("subscribe/%s", event.Pathname), // must match a route defined in apiserver routes!
		Method:          http.MethodPost,
		JSONInputValue:  func() interface{} { return &fftypes.ContractSubscribeRequest{} },
		JSONInputMask:   []string{"Interface", "Event"},
		JSONInputSchema: schema,
		JSONOutputValue: func() interface{} { return &fftypes.ContractSubscription{} },
		JSONOutputCodes: []int{http.StatusOK},
	})
}

/**
 * Parse the FFI and build a corresponding JSON Schema to describe the request body for "invoke".
 * Returns the JSON Schema as an `fftypes.JSONObject`.
 */
func contractCallJSONSchema(params *fftypes.FFIParams, hasLocation bool) *fftypes.JSONObject {
	req := &fftypes.ContractCallRequest{
		Input: *ffiParamsJSONSchema(params),
	}
	if !hasLocation {
		req.Location = fftypes.JSONAnyPtr(`{}`)
	}
	return &fftypes.JSONObject{
		"type":       "object",
		"properties": req,
	}
}

func ffiParamsJSONSchema(params *fftypes.FFIParams) *fftypes.JSONObject {
	out := make(fftypes.JSONObject, len(*params))
	for _, param := range *params {
		out[param.Name] = ffiParamJSONSchema(param)
	}
	return &fftypes.JSONObject{
		"type":       "object",
		"properties": out,
	}
}

func ffiParamJSONSchema(param *fftypes.FFIParam) *fftypes.JSONObject {
	out := fftypes.JSONObject{}
	switch {
	case strings.HasSuffix(param.Type, "[]"):
		baseType := strings.TrimSuffix(param.Type, "[]")
		if baseType == "byte" {
			out["type"] = "string"
		} else {
			out["type"] = "array"
			out["items"] = ffiParamJSONSchema(&fftypes.FFIParam{
				Type:       baseType,
				Components: param.Components,
			})
		}

	case len(param.Components) > 0:
		out = *ffiParamsJSONSchema(&param.Components)

	case param.Type == "string", param.Type == "integer", param.Type == "boolean":
		out["type"] = param.Type
	}
	return &out
}
