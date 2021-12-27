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
	"testing"

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
				Name: "sum",
				Params: fftypes.FFIParams{
					{
						Name:    "x",
						Type:    "integer",
						Details: []byte(`{}`),
					},
					{
						Name:    "y",
						Type:    "integer",
						Details: []byte(`{}`),
					},
				},
				Returns: fftypes.FFIParams{
					{
						Name:    "result",
						Type:    "integer",
						Details: []byte(`{}`),
					},
				},
			},
		},
		Events: []*fftypes.FFIEvent{
			{
				ID: fftypes.NewUUID(),
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "event1",
					Params: fftypes.FFIParams{
						{
							Name:    "result",
							Type:    "integer",
							Details: []byte(`{}`),
						},
					},
				},
			},
		},
	}
}
func TestGenerate(t *testing.T) {
	g := NewFFISwaggerGen()
	swagger, err := g.Generate(context.Background(), "http://localhost:12345", testFFI())
	assert.NoError(t, err)
	// TODO: this needs to actually check the swagger once it's complete
	assert.Nil(t, swagger)
}
