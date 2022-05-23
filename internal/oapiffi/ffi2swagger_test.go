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

package oapiffi

import (
	"context"
	"fmt"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ghodss/yaml"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
)

func testFFI() *core.FFI {
	return &core.FFI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "math",
		Version:   "v1.0.0",
		Methods: []*core.FFIMethod{
			{
				Name:     "method1",
				Pathname: "method1",
				Params: core.FFIParams{
					{
						Name:   "x",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer"}`),
					},
					{
						Name:   "y",
						Schema: fftypes.JSONAnyPtr(`{"type": "string", "contentEncoding": "base64"}`),
					},
					{
						Name: "z",
						Schema: fftypes.JSONAnyPtr(`
{
	"type": "object",
	"properties": {
		"name": {"type": "string"},
		"price": {"type": "integer"}
	}
}`),
					},
				},
				Returns: core.FFIParams{
					{
						Name:   "success",
						Schema: fftypes.JSONAnyPtr(`{"type": "boolean"}`),
					},
				},
			},
			{
				Name:     "method2",
				Pathname: "method2",
				/* no params */
			},
		},
		Events: []*core.FFIEvent{
			{
				ID:       fftypes.NewUUID(),
				Pathname: "event1",
				FFIEventDefinition: core.FFIEventDefinition{
					Name: "event1",
					Params: core.FFIParams{
						{
							Name:   "result",
							Schema: fftypes.JSONAnyPtr(`{"type": "integer"}`),
						},
					},
				},
			},
		},
	}
}

func pathNames(p openapi3.Paths) []string {
	var keys []string
	for k := range p {
		keys = append(keys, k)
	}
	return keys
}

func paramNames(p openapi3.Schemas) []string {
	var keys []string
	for k := range p {
		keys = append(keys, k)
	}
	return keys
}

func TestGenerate(t *testing.T) {
	g := NewFFISwaggerGen()
	api := &core.ContractAPI{}
	doc := g.Generate(context.Background(), "http://localhost:12345", api, testFFI())

	b, err := yaml.Marshal(doc)
	assert.NoError(t, err)
	fmt.Print(string(b))

	assert.ElementsMatch(t, []string{"/interface", "/invoke/method1", "/invoke/method2", "/query/method1", "/query/method2", "/listeners/event1"}, pathNames(doc.Paths))

	invokeMethod1 := doc.Paths["/invoke/method1"].Post.RequestBody.Value.Content.Get("application/json").Schema.Value
	assert.Equal(t, "object", invokeMethod1.Type)
	assert.ElementsMatch(t, []string{"input", "location"}, paramNames(invokeMethod1.Properties))
	assert.Equal(t, "object", invokeMethod1.Properties["input"].Value.Type)
	assert.ElementsMatch(t, []string{"x", "y", "z"}, paramNames(invokeMethod1.Properties["input"].Value.Properties))

	invokeMethod2 := doc.Paths["/invoke/method2"].Post.RequestBody.Value.Content.Get("application/json").Schema.Value
	assert.Equal(t, "object", invokeMethod2.Type)
	assert.ElementsMatch(t, []string{"input", "location"}, paramNames(invokeMethod2.Properties))
	assert.Equal(t, "object", invokeMethod2.Properties["input"].Value.Type)
	assert.ElementsMatch(t, []string{}, paramNames(invokeMethod2.Properties["input"].Value.Properties))

	queryMethod1 := doc.Paths["/query/method1"].Post.RequestBody.Value.Content.Get("application/json").Schema.Value
	assert.Equal(t, "object", queryMethod1.Type)
	assert.ElementsMatch(t, []string{"input", "location"}, paramNames(queryMethod1.Properties))
	assert.Equal(t, "object", queryMethod1.Properties["input"].Value.Type)
	assert.ElementsMatch(t, []string{"x", "y", "z"}, paramNames(queryMethod1.Properties["input"].Value.Properties))

	queryMethod2 := doc.Paths["/query/method2"].Post.RequestBody.Value.Content.Get("application/json").Schema.Value
	assert.Equal(t, "object", queryMethod2.Type)
	assert.ElementsMatch(t, []string{"input", "location"}, paramNames(queryMethod2.Properties))
	assert.Equal(t, "object", queryMethod2.Properties["input"].Value.Type)
	assert.ElementsMatch(t, []string{}, paramNames(queryMethod2.Properties["input"].Value.Properties))
}

func TestGenerateWithLocation(t *testing.T) {
	g := NewFFISwaggerGen()
	api := &core.ContractAPI{Location: fftypes.JSONAnyPtr(`{}`)}
	doc := g.Generate(context.Background(), "http://localhost:12345", api, testFFI())

	b, err := yaml.Marshal(doc)
	assert.NoError(t, err)
	fmt.Print(string(b))

	assert.ElementsMatch(t, []string{"/interface", "/invoke/method1", "/invoke/method2", "/query/method1", "/query/method2", "/listeners/event1"}, pathNames(doc.Paths))

	invokeMethod1 := doc.Paths["/invoke/method1"].Post.RequestBody.Value.Content.Get("application/json").Schema.Value
	assert.Equal(t, "object", invokeMethod1.Type)
	assert.ElementsMatch(t, []string{"input"}, paramNames(invokeMethod1.Properties))
	assert.Equal(t, "object", invokeMethod1.Properties["input"].Value.Type)
	assert.ElementsMatch(t, []string{"x", "y", "z"}, paramNames(invokeMethod1.Properties["input"].Value.Properties))

	invokeMethod2 := doc.Paths["/invoke/method2"].Post.RequestBody.Value.Content.Get("application/json").Schema.Value
	assert.Equal(t, "object", invokeMethod2.Type)
	assert.ElementsMatch(t, []string{"input"}, paramNames(invokeMethod2.Properties))
	assert.Equal(t, "object", invokeMethod2.Properties["input"].Value.Type)
	assert.ElementsMatch(t, []string{}, paramNames(invokeMethod2.Properties["input"].Value.Properties))

	queryMethod1 := doc.Paths["/query/method1"].Post.RequestBody.Value.Content.Get("application/json").Schema.Value
	assert.Equal(t, "object", queryMethod1.Type)
	assert.ElementsMatch(t, []string{"input"}, paramNames(queryMethod1.Properties))
	assert.Equal(t, "object", queryMethod1.Properties["input"].Value.Type)
	assert.ElementsMatch(t, []string{"x", "y", "z"}, paramNames(queryMethod1.Properties["input"].Value.Properties))

	queryMethod2 := doc.Paths["/query/method2"].Post.RequestBody.Value.Content.Get("application/json").Schema.Value
	assert.Equal(t, "object", queryMethod2.Type)
	assert.ElementsMatch(t, []string{"input"}, paramNames(queryMethod2.Properties))
	assert.Equal(t, "object", queryMethod2.Properties["input"].Value.Type)
	assert.ElementsMatch(t, []string{}, paramNames(queryMethod2.Properties["input"].Value.Properties))
}

func TestFFIParamBadSchema(t *testing.T) {
	param := &core.FFIParam{
		Name:   "test",
		Schema: fftypes.JSONAnyPtr(`{`),
	}
	r := ffiParamJSONSchema(param)
	assert.Nil(t, r)
}
