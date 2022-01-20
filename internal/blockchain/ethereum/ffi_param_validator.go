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
	"fmt"
	"regexp"
	"strconv"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type FFIParamValidator struct{}

var intRegex, _ = regexp.Compile("^u?int([0-9]{1,3})$")

func (v *FFIParamValidator) Compile(ctx jsonschema.CompilerContext, m map[string]interface{}) (jsonschema.ExtSchema, error) {
	valid := true
	if details, ok := m["details"]; ok {
		n, _ := details.(map[string]interface{})
		blockchainType := n["type"].(string)
		jsonType := m["type"].(string)
		switch jsonType {
		case "string":
			if blockchainType != "string" && blockchainType != "address" {
				valid = false
			}
		case "integer":
			// make sure the given format is a valid solidity integer type
			matches := intRegex.FindStringSubmatch(blockchainType)
			if len(matches) == 2 {
				i, err := strconv.ParseInt(matches[1], 10, 0)
				if err == nil && i >= 8 && i <= 256 && i%8 == 0 {
					// valid
					break
				}
			}
			valid = false
		case "boolean":
			if blockchainType != "bool" {
				valid = false
			}
		case "array":
			// TODO: Anything else here?
			valid = true
		case "object":
			// TODO: Anything else here?
			valid = true
		}

		if valid {
			return detailsSchema(n), nil
		}
		return nil, fmt.Errorf("cannot cast %v to %v", jsonType, blockchainType)
	}
	return nil, nil
}

func (v *FFIParamValidator) GetMetaSchema() *jsonschema.Schema {
	return jsonschema.MustCompileString("ffiParamDetails.json", `{
	"properties" : {
		"details": {
			"type": "object",
			"properties": {
				"type": {
					"type": "string"
				},
				"indexed": {
					"type": "boolean"
				}
			},
			"required": ["type"]
		}
	},
	"required": ["details"]
}`)
}

func (v *FFIParamValidator) GetExtensionName() string {
	return "details"
}

type detailsSchema map[string]interface{}

func (s detailsSchema) Validate(ctx jsonschema.ValidationContext, v interface{}) error {
	// TODO: Additional validation of actual input possible in the future
	return nil
}
