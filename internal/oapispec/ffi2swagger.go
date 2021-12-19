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

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type FFISwaggerGen interface {
	Generate(ctx context.Context, baseURL string, ffi *fftypes.FFI) (*openapi3.T, error)
}

// ffiSwaggerGen generates OpenAPI3 (Swagger) definitions for FFIs
type ffiSwaggerGen struct {
}

func NewFFISwaggerGen() FFISwaggerGen {
	return &ffiSwaggerGen{}
}

func (og *ffiSwaggerGen) Generate(ctx context.Context, baseURL string, ffi *fftypes.FFI) (swagger *openapi3.T, err error) {

	// routes := []*Route{}

	// for _, method := range ffi.Methods {
	// 	if routes, err = og.addMethod(ctx, routes, method); err != nil {
	// 		return nil, err
	// 	}
	// }

	// for _, event := range ffi.Events {
	// 	if routes, err = og.addEvent(ctx, routes, event); err != nil {
	// 		return nil, err
	// 	}
	// }

	return nil, nil
}

// func (og *ffiSwaggerGen) addMethod(ctx context.Context, routes []*Route, method *fftypes.FFIMethod) ([]*Route, error) {

// 	routes = append(routes, &Route{
// 		Name: fmt.Sprintf("method_%s_invoke", method.Name),
// 	})

// 	return routes, nil
// }

// func (og *ffiSwaggerGen) addEvent(ctx context.Context, routes []*Route, event *fftypes.FFIEvent) ([]*Route, error) {

// 	return routes, nil
// }
