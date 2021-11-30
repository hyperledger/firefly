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
	"encoding/base64"
	"encoding/hex"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

/* TODO:
* Move error messages to the translations file
 */

var inputTypeMap = map[string]string{
	"bool":    "boolean",
	"int":     "number",
	"float64": "number",
	"string":  "string",
	"byte":    "byte",
}

var arrayRegex, _ = regexp.Compile(`(\[\])+$`)

func checkParam(ctx context.Context, input interface{}, param *fftypes.FFIParam) error {
	if arrayRegex.MatchString(param.Type) {
		// input should be an array
		arrayType := strings.TrimSuffix(param.Type, "[]")
		if arrayType == "byte" && reflect.TypeOf(input).Name() == "string" {
			inputString := input.(string)
			// Is it hex (0x prefix) or base64 encoded?
			if strings.HasPrefix(inputString, "0x") {
				if _, err := hex.DecodeString(strings.TrimPrefix(inputString, "0x")); err != nil {
					return i18n.WrapError(ctx, err, i18n.MsgContractByteDecode, param.Name)
				}
			} else {
				if _, err := base64.StdEncoding.DecodeString(inputString); err != nil {
					return i18n.WrapError(ctx, err, i18n.MsgContractByteDecode, param.Name)
				}

			}
			return nil
		}
		arrayParam := &fftypes.FFIParam{
			Type:       arrayType,
			Components: param.Components,
		}
		inputArray, ok := input.([]interface{})

		if !ok {
			return i18n.NewError(ctx, i18n.MsgContractWrongInputType, input, reflect.TypeOf(input), param.Type)
		}
		return checkArrayType(ctx, inputArray, arrayParam)

	} else if len(param.Components) > 0 {
		// input should be an object
		inputMap, ok := input.(map[string]interface{})
		if !ok {
			return i18n.NewError(ctx, i18n.MsgContractWrongInputType, input, reflect.TypeOf(input), param.Type)
		}
		for _, component := range param.Components {
			childParam, ok := inputMap[component.Name]
			if !ok {
				return i18n.NewError(ctx, i18n.MsgContractMissingInputField, param.Type, component.Name)
			}
			if err := checkParam(ctx, childParam, component); err != nil {
				return err
			}
		}
	} else {
		rt := reflect.TypeOf(input)
		if rt == nil {
			return i18n.NewError(ctx, i18n.MsgContractMapInputType, rt, param.Type)
		}
		mappedType, ok := inputTypeMap[rt.Name()]
		if !ok {
			return i18n.NewError(ctx, i18n.MsgContractMapInputType, rt, param.Type)
		} else if mappedType == "string" && param.Type == "number" {
			inputString := input.(string)
			// check to see if it's a number represented as a string
			_, err := strconv.ParseFloat(inputString, 64)
			if err == nil {
				return nil
			}
			if strings.HasPrefix(inputString, "0x") {
				_, err = strconv.ParseUint(inputString, 16, 32)
				if err != nil {
					return nil
				}
			} else {
				return i18n.NewError(ctx, i18n.MsgContractWrongInputType, input, mappedType, param.Type)
			}
		} else if mappedType != param.Type {
			return i18n.NewError(ctx, i18n.MsgContractWrongInputType, input, mappedType, param.Type)
		}
	}
	return nil
}

// Recursively check the i-th dimension of an an n-dimensional array to make sure
// that each dimension is an array or all leaf elements are of type t
func checkArrayType(ctx context.Context, input []interface{}, param *fftypes.FFIParam) error {
	for _, v := range input {
		if arrayRegex.MatchString(param.Type) {
			// input should be an array
			arrayType := strings.TrimSuffix(param.Type, "[]")
			arrayParam := &fftypes.FFIParam{
				Type:       arrayType,
				Components: param.Components,
			}
			arr, ok := v.([]interface{})
			rt := reflect.TypeOf(v)
			if rt != reflect.TypeOf(arr) || !ok {
				return i18n.NewError(ctx, i18n.MsgContractWrongInputType, v, rt, reflect.TypeOf(arr))
			}
			if err := checkArrayType(ctx, arr, arrayParam); err != nil {
				return err
			}
		} else if err := checkParam(ctx, v, param); err != nil {
			return err
		}
	}
	return nil
}
