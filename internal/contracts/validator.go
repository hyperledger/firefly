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
	"math"
	"math/big"
	"reflect"
	"regexp"
	"strings"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

/* TODO:
* Move error messages to the translations file
 */

var inputTypeMap = map[string]string{
	"bool":    "boolean",
	"float64": "integer",
	"string":  "string",
	"byte":    "byte",
}

var arrayRegex, _ = regexp.Compile(`(\[\])+$`)

func checkParam(ctx context.Context, input interface{}, param *fftypes.FFIParam) error {
	switch {
	case arrayRegex.MatchString(param.Type):
		// check to see if this a string that should be treated as a byte array
		if strings.TrimSuffix(param.Type, "[]") == "byte" && reflect.TypeOf(input).Name() == "string" {
			return checkByteArray(ctx, input, param)
		}
		return checkArrayType(ctx, input, param)
	case len(param.Components) > 0:
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
	default:
		rt := reflect.TypeOf(input)
		if rt == nil {
			return i18n.NewError(ctx, i18n.MsgContractMapInputType, rt, param.Type)
		}
		mappedType, ok := inputTypeMap[rt.Name()]
		switch {
		case !ok:
			return i18n.NewError(ctx, i18n.MsgContractMapInputType, rt, param.Type)
		case param.Type == "integer":
			return checkInteger(ctx, input, param)
		case mappedType != param.Type:
			return i18n.NewError(ctx, i18n.MsgContractWrongInputType, input, mappedType, param.Type)
		}
	}
	return nil
}

// Recursively check the i-th dimension of an an n-dimensional array to make sure
// that each dimension is an array or all leaf elements are of type t
func checkArrayType(ctx context.Context, input interface{}, param *fftypes.FFIParam) error {
	elementType := strings.TrimSuffix(param.Type, "[]")
	elementParam := &fftypes.FFIParam{
		Name:       param.Name,
		Type:       elementType,
		Details:    param.Details,
		Components: param.Components,
	}
	inputArray, ok := input.([]interface{})

	if !ok {
		return i18n.NewError(ctx, i18n.MsgContractWrongInputType, input, reflect.TypeOf(input), param.Type)
	}

	for _, v := range inputArray {
		if arrayRegex.MatchString(elementType) {
			// input should be an array
			arr, ok := v.([]interface{})
			rt := reflect.TypeOf(v)
			if rt != reflect.TypeOf(arr) || !ok {
				return i18n.NewError(ctx, i18n.MsgContractWrongInputType, v, rt, reflect.TypeOf(arr))
			}
			if err := checkArrayType(ctx, arr, elementParam); err != nil {
				return err
			}
		} else if err := checkParam(ctx, v, elementParam); err != nil {
			return err
		}
	}
	return nil
}

func checkByteArray(ctx context.Context, input interface{}, param *fftypes.FFIParam) error {
	inputString := input.(string)
	// if it is prefixed with "0X", treat it as hex, otherwise base64 encoded
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

func checkInteger(ctx context.Context, input interface{}, param *fftypes.FFIParam) error {
	switch t := input.(type) {
	case float64:
		if t != math.Trunc(t) {
			return i18n.NewError(ctx, i18n.MsgContractWrongInputType, input, reflect.TypeOf(t), param.Type)
		}
	case string:
		i := new(big.Int)
		var ok bool
		if strings.HasPrefix(t, "0x") {
			t = strings.TrimPrefix(t, "0x")
			_, ok = i.SetString(t, 16)
		} else {
			_, ok = i.SetString(t, 10)
		}
		if !ok {
			return i18n.NewError(ctx, i18n.MsgContractWrongInputType, t, reflect.TypeOf(t), param.Type)
		}
	default:
		return i18n.NewError(ctx, i18n.MsgContractWrongInputType, t, reflect.TypeOf(t), param.Type)
	}
	return nil
}
