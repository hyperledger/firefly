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

package contracts

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestNestedArrayTypeCheck(t *testing.T) {
	param := &fftypes.FFIParam{
		Type: "integer[][]",
	}
	jsonInput := `[[1,2,3],[4,5,6]]`
	var arr []interface{}
	json.Unmarshal([]byte(jsonInput), &arr)
	err := checkArrayType(context.Background(), arr, param)
	assert.NoError(t, err)
}

func TestBadNestedArrayTypeCheck(t *testing.T) {
	param := &fftypes.FFIParam{
		Type: "integer[][]",
	}
	jsonInput := `[[1,2,3],[4,"bad",6]]`
	var arr []interface{}
	err := json.Unmarshal([]byte(jsonInput), &arr)
	assert.NoError(t, err)
	err = checkArrayType(context.Background(), arr, param)
	assert.Regexp(t, "bad.*string", err)
}

func TestInvalidArrayTypeCheck(t *testing.T) {
	param := &fftypes.FFIParam{
		Type: "integer[][]",
	}
	jsonInput := `[[1,2,3],"bad"]`
	var arr []interface{}
	err := json.Unmarshal([]byte(jsonInput), &arr)
	assert.NoError(t, err)
	err = checkArrayType(context.Background(), arr, param)
	assert.Regexp(t, "bad.*string", err)
}

func TestNotAnArrayTypeCheck(t *testing.T) {
	param := &fftypes.FFIParam{
		Type: "integer[]",
	}
	jsonInput := `{"isArray": "nope"}`
	var arr interface{}
	json.Unmarshal([]byte(jsonInput), &arr)
	err := checkParam(context.Background(), arr, param)
	assert.Regexp(t, "Input.*not expected.*integer", err)
}

func TestCustomTypeSimple(t *testing.T) {
	param := &fftypes.FFIParam{
		Name:    "testParam",
		Type:    "struct CustomType",
		Details: []byte("{\"type\":\"struct\"}"),
		Components: []*fftypes.FFIParam{
			{
				Name:    "foo",
				Type:    "string",
				Details: []byte("{\"type\":\"string\"}"),
			},
			{
				Name:    "count",
				Type:    "integer",
				Details: []byte("{\"type\":\"uint256\"}"),
			},
		},
	}
	jsonInput := `{"foo": "bar", "count": 10}`
	var i interface{}
	err := json.Unmarshal([]byte(jsonInput), &i)
	assert.NoError(t, err)
	err = checkParam(context.Background(), i, param)
	assert.NoError(t, err)
}

func TestCustomTypeWrongType(t *testing.T) {
	param := &fftypes.FFIParam{
		Name:    "testParam",
		Type:    "struct CustomType",
		Details: []byte("\"type\":\"struct\"}"),
		Components: []*fftypes.FFIParam{
			{
				Name:    "foo",
				Type:    "string",
				Details: []byte("\"type\":\"string\"}"),
			},
			{
				Name:    "count",
				Type:    "integer",
				Details: []byte("\"type\":\"uint256\"}"),
			},
		},
	}
	jsonInput := `{"foo": "bar", "count": "bad"}`
	var i interface{}
	err := json.Unmarshal([]byte(jsonInput), &i)
	assert.NoError(t, err)
	err = checkParam(context.Background(), i, param)
	assert.Regexp(t, "bad.*string", err)
}

func TestCustomTypeFieldMissing(t *testing.T) {
	param := &fftypes.FFIParam{
		Name:    "testParam",
		Type:    "struct CustomType",
		Details: []byte("\"type\":\"struct\"}"),
		Components: []*fftypes.FFIParam{
			{
				Name:    "foo",
				Type:    "string",
				Details: []byte("\"type\":\"string\"}"),
			},
			{
				Name:    "count",
				Type:    "integer",
				Details: []byte("\"type\":\"uint256\"}"),
			},
		},
	}
	jsonInput := `{"foo": "bar"}`
	var i interface{}
	err := json.Unmarshal([]byte(jsonInput), &i)
	assert.NoError(t, err)
	err = checkParam(context.Background(), i, param)
	assert.Regexp(t, "count.*missing", err)
}

func TestCustomTypeInvalid(t *testing.T) {
	param := &fftypes.FFIParam{
		Name:    "testParam",
		Type:    "struct CustomType",
		Details: []byte("\"type\":\"struct\"}"),
		Components: []*fftypes.FFIParam{
			{
				Name:    "foo",
				Type:    "string",
				Details: []byte("\"type\":\"string\"}"),
			},
			{
				Name:    "count",
				Type:    "integer",
				Details: []byte("\"type\":\"uint256\"}"),
			},
		},
	}
	jsonInput := `["one", "two", "three"]`
	var i interface{}
	err := json.Unmarshal([]byte(jsonInput), &i)
	assert.NoError(t, err)
	err = checkParam(context.Background(), i, param)
	assert.Regexp(t, "Input.*not expected type", err)
}

func TestArrayOfCustomType(t *testing.T) {
	param := &fftypes.FFIParam{
		Name:    "testParam",
		Type:    "struct CustomType[]",
		Details: []byte("\"type\":\"struct[]\"}"),
		Components: []*fftypes.FFIParam{
			{
				Name:    "foo",
				Type:    "string",
				Details: []byte("\"type\":\"string\"}"),
			},
			{
				Name:    "count",
				Type:    "integer",
				Details: []byte("\"type\":\"uint256\"}"),
			},
		},
	}
	jsonInput := `[
		{
			"foo": "bar one",
			"count": 1
		},
		{
			"foo": "bar two",
			"count": 2
		}
	]
	`
	var i interface{}
	err := json.Unmarshal([]byte(jsonInput), &i)
	assert.NoError(t, err)
	err = checkParam(context.Background(), i, param)
	assert.NoError(t, err)
}

func TestNullValue(t *testing.T) {
	param := &fftypes.FFIParam{
		Name:    "testParam",
		Type:    "struct CustomType",
		Details: []byte("\"type\":\"struct\"}"),
		Components: []*fftypes.FFIParam{
			{
				Name:    "foo",
				Type:    "string",
				Details: []byte("\"type\":\"string\"}"),
			},
		},
	}
	jsonInput := `{"foo": null}`
	var i interface{}
	err := json.Unmarshal([]byte(jsonInput), &i)
	assert.NoError(t, err)
	err = checkParam(context.Background(), i, param)
	assert.Regexp(t, "Unable to map input", err)
}

func TestInvalidTypeMapping(t *testing.T) {
	param := &fftypes.FFIParam{
		Name:    "foo",
		Type:    "integer",
		Details: []byte("\"type\":\"uint256\"}"),
	}
	var i float32 = 32
	err := checkParam(context.Background(), i, param)
	assert.Regexp(t, "Unable to map input", err)
}

func TestComplexCustomType(t *testing.T) {
	param := &fftypes.FFIParam{
		Name:    "testParam",
		Type:    "struct CustomType[][]",
		Details: []byte("\"type\":\"struct\"}"),
		Components: []*fftypes.FFIParam{
			{
				Name:    "x",
				Type:    "integer",
				Details: []byte("\"type\":\"uint256\"}"),
			},
			{
				Name:    "y",
				Type:    "integer",
				Details: []byte("\"type\":\"uint256\"}"),
			},
			{
				Name:    "foo",
				Type:    "struct Foo",
				Details: []byte("\"type\":\"struct\"}"),
				Components: []*fftypes.FFIParam{
					{
						Name:    "bar",
						Type:    "string",
						Details: []byte("\"type\":\"string\"}"),
					},
				},
			},
		},
	}
	jsonInput := `[
		[{"x": 0, "y": 0, "foo": {"bar": "one"}}, {"x": 1, "y": 0, "foo": {"bar": "two"}}],
		[{"x": 0, "y": 1, "foo": {"bar": "three"}}, {"x": 1, "y": 1, "foo": {"bar": "four"}}]
	]`
	var i interface{}
	err := json.Unmarshal([]byte(jsonInput), &i)
	assert.NoError(t, err)
	err = checkParam(context.Background(), i, param)
	assert.NoError(t, err)
}

func TestBadComplexCustomType(t *testing.T) {
	param := &fftypes.FFIParam{
		Name:    "testParam",
		Type:    "struct CustomType[][]",
		Details: []byte("\"type\":\"struct\"}"),
		Components: []*fftypes.FFIParam{
			{
				Name:    "x",
				Type:    "integer",
				Details: []byte("\"type\":\"uint256\"}"),
			},
			{
				Name:    "y",
				Type:    "integer",
				Details: []byte("\"type\":\"uint256\"}"),
			},
			{
				Name:    "foo",
				Type:    "struct Foo",
				Details: []byte("\"type\":\"struct\"}"),
				Components: []*fftypes.FFIParam{
					{
						Name:    "bar",
						Type:    "string",
						Details: []byte("\"type\":\"string\"}"),
					},
				},
			},
		},
	}
	jsonInput := `[
		[{"x": 0, "y": 0, "foo": {"bar": "one"}}, {"x": 1, "y": 0, "foo": {"bar": "two"}}],
		[{"x": 0, "y": 1, "foo": {"bar": "three"}}, {"x": 1, "y": 1, "foo": {"bar": 4}}]
	]`
	var i interface{}
	err := json.Unmarshal([]byte(jsonInput), &i)
	assert.NoError(t, err)
	err = checkParam(context.Background(), i, param)
	assert.Regexp(t, "Input.*integer.*not expected type", err)
}

func TestBytesBase64(t *testing.T) {
	param := &fftypes.FFIParam{
		Type: "byte[]",
	}
	err := checkParam(context.Background(), "ZmlyZWZseQ==", param)
	assert.NoError(t, err)
}

func TestBadBytesBase64(t *testing.T) {
	param := &fftypes.FFIParam{
		Type: "byte[]",
	}
	err := checkParam(context.Background(), "&^$*^#^$*^%@#", param)
	assert.Regexp(t, "illegal base64 data at input byte", err)
}

func TestBytesHex(t *testing.T) {
	param := &fftypes.FFIParam{
		Type: "byte[]",
	}
	err := checkParam(context.Background(), "0x66697265666C79", param)
	assert.NoError(t, err)
}

func TestBadBytesHex(t *testing.T) {
	param := &fftypes.FFIParam{
		Type: "byte[]",
	}
	err := checkParam(context.Background(), "0xG88J49FJMK30DFJ2R", param)
	assert.Regexp(t, "encoding/hex: invalid byte:", err)
}

func TestIntegerAsString(t *testing.T) {
	param := &fftypes.FFIParam{
		Type: "integer",
	}
	err := checkParam(context.Background(), "1", param)
	assert.NoError(t, err)
}

func TestIntegerAsHexString(t *testing.T) {
	param := &fftypes.FFIParam{
		Type: "integer",
	}
	err := checkParam(context.Background(), "0xffffff", param)
	assert.NoError(t, err)
}

func TestStringAsInteger(t *testing.T) {
	param := &fftypes.FFIParam{
		Type: "integer",
	}
	err := checkParam(context.Background(), "123456789012345678901234567890", param)
	assert.NoError(t, err)
}

func TestIntegerAsHexStringBad(t *testing.T) {
	param := &fftypes.FFIParam{
		Type: "integer",
	}
	err := checkParam(context.Background(), "ffffff", param)
	assert.Regexp(t, "Input.*string.*not expected type", err)
}

func TestFloatInvalid(t *testing.T) {
	param := &fftypes.FFIParam{
		Type: "integer",
	}
	err := checkParam(context.Background(), 0.5, param)
	assert.Regexp(t, "Input.*float64.*not expected type", err)
}

func TestIntegerInvalid(t *testing.T) {
	param := &fftypes.FFIParam{
		Type: "integer",
	}
	err := checkParam(context.Background(), false, param)
	assert.Regexp(t, "FF10297", err)
}
