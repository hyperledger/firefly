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
	"testing"

	"github.com/ghodss/yaml"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func testFFI() *fftypes.FFI {
	return &fftypes.FFI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "math",
		Version:   "v1.0.0",
		Methods: []*fftypes.FFIMethod{
			{
				Name:     "method1",
				Pathname: "method1",
				Params: fftypes.FFIParams{
					{
						Name: "x",
						Type: "integer",
					},
					{
						Name: "y",
						Type: "byte[]",
					},
					{
						Name: "z",
						Type: "widget[]",
						Components: fftypes.FFIParams{
							{
								Name: "name",
								Type: "string",
							},
							{
								Name: "price",
								Type: "integer",
							},
						},
					},
				},
				Returns: fftypes.FFIParams{
					{
						Name: "success",
						Type: "boolean",
					},
				},
			},
		},
		Events: []*fftypes.FFIEvent{
			{
				ID:       fftypes.NewUUID(),
				Pathname: "event1",
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "event1",
					Params: fftypes.FFIParams{
						{
							Name: "result",
							Type: "integer",
						},
					},
				},
			},
		},
	}
}

func TestGenerate(t *testing.T) {
	g := NewFFISwaggerGen()
	api := &fftypes.ContractAPI{}
	doc, err := g.Generate(context.Background(), "http://localhost:12345", api, testFFI())
	assert.NoError(t, err)

	b, err := yaml.Marshal(doc)
	assert.NoError(t, err)
	fmt.Print(string(b))
}

func TestGenerateWithLocation(t *testing.T) {
	g := NewFFISwaggerGen()
	api := &fftypes.ContractAPI{Location: []byte(`{}`)}
	doc, err := g.Generate(context.Background(), "http://localhost:12345", api, testFFI())
	assert.NoError(t, err)

	b, err := yaml.Marshal(doc)
	assert.NoError(t, err)
	fmt.Print(string(b))
}
