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

package ethereum

import (
	"context"
	"testing"

	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestABI2FFI(t *testing.T) {
	e, _ := newTestEthereum()

	method := &fftypes.FFIMethod{
		Name: "set",
		Params: []*fftypes.FFIParam{
			{
				Name: "newValue",
				Schema: fftypes.JSONAnyPtr(`{
					"type": "integer",
					"details": {
						"type": "uint256"
					}
				}`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}

	expectedABIElement := ABIElementMarshaling{
		Name: "set",
		Inputs: []ABIArgumentMarshaling{
			{
				Name:    "newValue",
				Type:    "uint256",
				Indexed: false,
			},
		},
		Outputs: []ABIArgumentMarshaling{},
	}

	abi, err := e.FFI2ABI(context.Background(), method)
	assert.NoError(t, err)
	assert.Equal(t, expectedABIElement, abi)
}

func TestABI2FFIObject(t *testing.T) {
	e, _ := newTestEthereum()

	method := &fftypes.FFIMethod{
		Name: "set",
		Params: []*fftypes.FFIParam{
			{
				Name: "widget",
				Schema: fftypes.JSONAnyPtr(`{
					"type": "object",
					"details": {
						"type": "tuple"
					},
					"properties": {
						"radius": {
							"type": "integer",
							"details": {
								"type": "uint256",
								"index": 0,
								"indexed": true
							}
						},
						"numbers": {
							"type": "array",
							"details": {
								"type": "uint256[]",
								"index": 1
							},
							"items": {
								"type": "integer",
								"details": {
									"type": "uint256"
								}
							}
						}
					}
				}`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}

	expectedABIElement := ABIElementMarshaling{
		Name: "set",
		Inputs: []ABIArgumentMarshaling{
			{
				Name:    "widget",
				Type:    "tuple",
				Indexed: false,
				Components: []ABIArgumentMarshaling{
					{
						Name:         "radius",
						Type:         "uint256",
						Indexed:      true,
						InternalType: "",
					},
					{
						Name:         "numbers",
						Type:         "uint256[]",
						Indexed:      false,
						InternalType: "",
					},
				},
			},
		},
		Outputs: []ABIArgumentMarshaling{},
	}

	abi, err := e.FFI2ABI(context.Background(), method)
	assert.NoError(t, err)
	assert.Equal(t, expectedABIElement, abi)
}

func TestABI2FFIInvalidJSON(t *testing.T) {
	e, _ := newTestEthereum()

	method := &fftypes.FFIMethod{
		Name: "set",
		Params: []*fftypes.FFIParam{
			{
				Name:   "newValue",
				Schema: fftypes.JSONAnyPtr(`{#!`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}

	_, err := e.FFI2ABI(context.Background(), method)
	assert.Regexp(t, "invalid json", err)
}

func TestABI2FFIBadSchema(t *testing.T) {
	e, _ := newTestEthereum()

	method := &fftypes.FFIMethod{
		Name: "set",
		Params: []*fftypes.FFIParam{
			{
				Name: "newValue",
				Schema: fftypes.JSONAnyPtr(`{
					"type": "integer",
					"detailz": {
						"type": "uint256"
					}
				}`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}

	_, err := e.FFI2ABI(context.Background(), method)
	assert.Regexp(t, "compilation failed", err)
}
