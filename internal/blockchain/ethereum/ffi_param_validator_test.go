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
	"encoding/json"
	"strings"
	"testing"

	"github.com/hyperledger/firefly/internal/contracts"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/assert"
)

func NewTestSchema(input string) (*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft2020
	f := contracts.FFIParamValidator{}
	c.RegisterExtension(f.GetExtensionName(), f.GetMetaSchema(), f)
	e, _ := newTestEthereum()
	v, err := e.GetFFIParamValidator(context.Background())
	if err != nil {
		return nil, err
	}
	c.RegisterExtension(v.GetExtensionName(), v.GetMetaSchema(), v)
	err = c.AddResource("schema.json", strings.NewReader(input))
	if err != nil {
		return nil, err
	}
	return c.Compile("schema.json")
}

func jsonDecode(input string) interface{} {
	var output interface{}
	json.Unmarshal([]byte(input), &output)
	return output
}

func TestSchemaValid(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "integer",
	"details": {
		"type": "uint256"
	}
}`)
	assert.NoError(t, err)
}

func TestSchemaValidBytes(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "string",
	"details": {
		"type": "bytes"
	}
}`)
	assert.NoError(t, err)
}

func TestSchemaValidBytes32(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "string",
	"details": {
		"type": "bytes32"
	}
}`)
	assert.NoError(t, err)
}

func TestSchemaTypeInvalid(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "foobar",
	"details": {
		"type": "uint256"
	}
}`)
	assert.Regexp(t, "'/type' does not validate", err)
}

func TestSchemaTypeInvalidFFIType(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "number",
	"details": {
		"type": "uint256"
	}
}`)
	assert.Regexp(t, "'/type' does not validate", err)
}

func TestSchemaTypeMissing(t *testing.T) {
	_, err := NewTestSchema(`{}`)
	assert.Regexp(t, "missing properties: 'type'", err)
}

func TestSchemaDetailsTypeMissing(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "string",
	"details": {
		"indexed": true
	}
}`)
	assert.Regexp(t, "missing properties: 'type'", err)
}

func TestSchemaDetailsIndexedWrongType(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "string",
	"details": {
		"type": "string",
		"indexed": "string"
	}
}`)
	assert.Regexp(t, "expected boolean, but got string", err)
}

func TestSchemaTypeMismatch(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "string",
	"details": {
		"type": "boolean"
	}
}`)
	assert.Regexp(t, "cannot cast string to boolean", err)
}

func TestInputString(t *testing.T) {
	s, err := NewTestSchema(`
{
	"type": "string",
	"details": {
		"type": "string"
	}
}`)
	assert.NoError(t, err)
	err = s.Validate(`"banana"`)
	assert.NoError(t, err)
}

func TestInputInteger(t *testing.T) {
	s, err := NewTestSchema(`
{
	"type": "integer",
	"details": {
		"type": "uint256"
	}
}`)
	assert.NoError(t, err)
	err = s.Validate(1)
	assert.NoError(t, err)
}

func TestInputBoolean(t *testing.T) {
	s, err := NewTestSchema(`
{
	"type": "boolean",
	"details": {
		"type": "bool"
	}
}`)
	assert.NoError(t, err)
	err = s.Validate(true)
	assert.NoError(t, err)
}

func TestInputStruct(t *testing.T) {
	s, err := NewTestSchema(`
{
	"type": "object",
	"details": {
		"type": "struct"
	},
	"properties": {
		"x": {
			"type": "integer",
			"details": {
				"type": "uint8"
			}
		},
		"y": {
			"type": "integer",
			"details": {
				"type": "uint8"
			}
		},
		"z": {
			"type": "integer",
			"details": {
				"type": "uint8"
			}
		}
	},
	"required": ["x", "y", "z"]
}`)

	input := `{
	"x": 123,
	"y": 456,
	"z": 789		
}`

	assert.NoError(t, err)
	err = s.Validate(jsonDecode(input))
	assert.NoError(t, err)
}

func TestInputArray(t *testing.T) {
	s, err := NewTestSchema(`
{
	"type": "array",
	"details": {
		"type": "uint8[]"
	},
	"items": {
		"type": "integer"
	}
}`)

	input := `[123,456,789]`

	assert.NoError(t, err)
	err = s.Validate(jsonDecode(input))
	assert.NoError(t, err)
}

func TestInputInvalidBlockchainType(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "boolean",
	"details": {
		"type": "foobar"
	}
}`)
	assert.Regexp(t, "cannot cast boolean to foobar", err)
}

func TestInputInvalidNestedBlockchainType(t *testing.T) {
	_, err := NewTestSchema(`
{
	"type": "object",
	"details": {
		"type": "struct"
	},
	"properties": {
		"amount": {
			"type": "integer",
			"details": {
				"type": "string"
			}
		}
	}
}`)
	assert.Regexp(t, "cannot cast integer to string", err)
}

func TestInputNoAdditionalProperties(t *testing.T) {
	s, err := NewTestSchema(`
{
	"type": "object",
	"details": {
		"type": "struct"
	},
	"properties": {
		"foo": {
			"type": "string",
			"details": {
				"type": "string"
			}
		}
	},
	"additionalProperties": false
}`)

	input := `{
	"foo": "foo",
	"bar": "bar"		
}`

	assert.NoError(t, err)
	err = s.Validate(jsonDecode(input))
	assert.Regexp(t, "additionalProperties 'bar' not allowed", err)
}
