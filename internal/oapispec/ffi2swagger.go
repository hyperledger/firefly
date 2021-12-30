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

package oapispec

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type FFISwaggerGen interface {
	Generate(ctx context.Context, baseURL string, api *fftypes.ContractAPI, ffi *fftypes.FFI) (*openapi3.T, error)
}

// ffiSwaggerGen generates OpenAPI3 (Swagger) definitions for FFIs
type ffiSwaggerGen struct {
}

func NewFFISwaggerGen() FFISwaggerGen {
	return &ffiSwaggerGen{}
}

func (og *ffiSwaggerGen) Generate(ctx context.Context, baseURL string, api *fftypes.ContractAPI, ffi *fftypes.FFI) (swagger *openapi3.T, err error) {
	hasLocation := !api.Location.IsNil()

	routes := []*Route{}
	for _, method := range ffi.Methods {
		routes = og.addMethod(routes, method, hasLocation)
	}
	for _, event := range ffi.Events {
		routes = og.addEvent(routes, event, hasLocation)
	}

	return SwaggerGen(ctx, routes, &SwaggerGenConfig{
		Title:       ffi.Name,
		Version:     ffi.Version,
		Description: ffi.Description,
		BaseURL:     baseURL,
	}), nil
}

func (og *ffiSwaggerGen) addMethod(routes []*Route, method *fftypes.FFIMethod, hasLocation bool) []*Route {
	return append(routes, &Route{
		Name:             fmt.Sprintf("invoke_%s", method.Pathname),
		Path:             fmt.Sprintf("invoke/%s", method.Pathname),
		Method:           http.MethodPost,
		JSONInputValue:   func() interface{} { return &fftypes.InvokeContractRequest{} },
		JSONInputSchema:  func(ctx context.Context) string { return invokeRequestJSON(&method.Params, hasLocation).String() },
		JSONOutputValue:  func() interface{} { return &fftypes.JSONObject{} },
		JSONOutputSchema: func(ctx context.Context) string { return ffiParamsJSON(&method.Returns).String() },
		JSONOutputCodes:  []int{http.StatusOK},
	})
}

func (og *ffiSwaggerGen) addEvent(routes []*Route, event *fftypes.FFIEvent, hasLocation bool) []*Route {
	// If the API has a location specified, there are no fields left to specify in the request body.
	// Instead of masking them all (which causes Swagger UI some issues), explicitly set the schema to an empty object.
	var schema func(ctx context.Context) string
	if hasLocation {
		schema = func(ctx context.Context) string { return `{"type": "object"}` }
	}
	return append(routes, &Route{
		Name:            fmt.Sprintf("subscribe_%s", event.Pathname),
		Path:            fmt.Sprintf("subscribe/%s", event.Pathname),
		Method:          http.MethodPost,
		JSONInputValue:  func() interface{} { return &fftypes.ContractSubscribeRequest{} },
		JSONInputMask:   []string{"Interface", "Event"},
		JSONInputSchema: schema,
		JSONOutputValue: func() interface{} { return &fftypes.ContractSubscription{} },
		JSONOutputCodes: []int{http.StatusOK},
	})
}

func invokeRequestJSON(params *fftypes.FFIParams, hasLocation bool) *fftypes.JSONObject {
	req := &fftypes.InvokeContractRequest{
		Params: *ffiParamsJSON(params),
	}
	if !hasLocation {
		req.Location = []byte(`{}`)
	}
	return &fftypes.JSONObject{
		"type":       "object",
		"properties": req,
	}
}

func ffiParamsJSON(params *fftypes.FFIParams) *fftypes.JSONObject {
	out := make(fftypes.JSONObject, len(*params))
	for _, param := range *params {
		out[param.Name] = ffiParamJSON(param)
	}
	return &fftypes.JSONObject{
		"type":       "object",
		"properties": out,
	}
}

func ffiParamJSON(param *fftypes.FFIParam) *fftypes.JSONObject {
	out := fftypes.JSONObject{}
	switch {
	case strings.HasSuffix(param.Type, "[]"):
		baseType := strings.TrimSuffix(param.Type, "[]")
		if baseType == "byte" {
			out["type"] = "string"
		} else {
			out["type"] = "array"
			out["items"] = ffiParamJSON(&fftypes.FFIParam{
				Type:       baseType,
				Components: param.Components,
			})
		}

	case len(param.Components) > 0:
		out = *ffiParamsJSON(&param.Components)

	case param.Type == "string", param.Type == "integer", param.Type == "boolean":
		out["type"] = param.Type
	}
	return &out
}
