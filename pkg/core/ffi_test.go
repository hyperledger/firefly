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

package core

import (
	"context"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestValidateFFI(t *testing.T) {
	ffi := &FFI{
		Name:      "math",
		Namespace: "default",
		Version:   "v1.0.0",
		Methods: []*FFIMethod{
			{
				Name: "sum",
				Params: []*FFIParam{
					{
						Name:   "x",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer"}, "details": {"type": "uint256"}`),
					},
					{
						Name:   "y",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer"}, "details": {"type": "uint256"}`),
					},
				},
				Returns: []*FFIParam{
					{
						Name:   "z",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer"}, "details": {"type": "uint256"}`),
					},
				},
			},
		},
		Events: []*FFIEvent{
			{
				FFIEventDefinition: FFIEventDefinition{
					Name: "sum",
					Params: []*FFIParam{
						{
							Name:   "z",
							Schema: fftypes.JSONAnyPtr(`{"type": "integer"}, "details": {"type": "uint256"}`),
						},
					},
				},
			},
		},
	}
	err := ffi.Validate(context.Background(), true)
	assert.NoError(t, err)
}

func TestValidateFFIBadVersion(t *testing.T) {
	ffi := &FFI{
		Name:      "math",
		Namespace: "default",
		Version:   "*(&!$%^)",
	}
	err := ffi.Validate(context.Background(), true)
	assert.Regexp(t, "FF00140", err)
}

func TestValidateFFIBadName(t *testing.T) {
	ffi := &FFI{
		Name:      "(*%&#%)",
		Namespace: "default",
		Version:   "v1.0.0",
	}
	err := ffi.Validate(context.Background(), true)
	assert.Regexp(t, "FF00140", err)
}

func TestValidateFFIBadNamespace(t *testing.T) {
	ffi := &FFI{
		Name:      "math",
		Namespace: "",
		Version:   "v1.0.0",
	}
	err := ffi.Validate(context.Background(), true)
	assert.Regexp(t, "FF00140", err)
}

func TestFFIParamsScan(t *testing.T) {
	params := &FFIParams{}
	err := params.Scan([]byte(`[{"name": "x", "type": "integer", "internalType": "uint256"}]`))
	assert.NoError(t, err)
}

func TestFFIParamsScanString(t *testing.T) {
	params := &FFIParams{}
	err := params.Scan(`[{"name": "x", "type": "integer", "internalType": "uint256"}]`)
	assert.NoError(t, err)
}

func TestFFIParamsScanNil(t *testing.T) {
	params := &FFIParams{}
	err := params.Scan(nil)
	assert.Nil(t, err)
}

func TestFFIParamsScanError(t *testing.T) {
	params := &FFIParams{}
	err := params.Scan(map[string]interface{}{"type": "not supported for scanning FFIParams"})
	assert.Regexp(t, "FF00105", err)
}

func TestFFIParamsValue(t *testing.T) {
	params := &FFIParams{
		&FFIParam{
			Name:   "x",
			Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
		},
	}

	val, err := params.Value()
	assert.NoError(t, err)
	assert.Equal(t, []byte(`[{"name":"x","schema":{"type":"integer","details":{"type":"uint256"}}}]`), val)
}

func TestFFITopic(t *testing.T) {
	ffi := &FFI{
		Namespace: "ns1",
	}
	assert.Equal(t, "01a982a7251400a7ec64fccce6febee3942a56e37967fa2ba26d7d6f43523c82", ffi.Topic())
}

func TestFFISetBroadCastMessage(t *testing.T) {
	msgID := fftypes.NewUUID()
	ffi := &FFI{}
	ffi.SetBroadcastMessage(msgID)
	assert.Equal(t, ffi.Message, msgID)
}
