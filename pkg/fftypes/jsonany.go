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

package fftypes

import (
	"context"
	"crypto/sha256"
	"encoding/json"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
)

const (
	NullString = "null"
)

// JSONAny uses raw encode/decode to preserve field order, and can handle any types of field.
// It validates the JSON can be unmarshalled, but does not change the order.
// It does however trim out whitespace
type JSONAny string

func JSONAnyPtr(str string) *JSONAny {
	return (*JSONAny)(&str)
}

func JSONAnyPtrBytes(b []byte) *JSONAny {
	if b == nil {
		return nil
	}
	ja := JSONAny(b)
	return &ja
}

func (h *JSONAny) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		*h = JSONAny(NullString)
		return nil
	}
	var flattener json.RawMessage
	err := json.Unmarshal(b, &flattener)
	if err != nil {
		return err
	}
	standardizedBytes, err := json.Marshal(flattener)
	if err == nil {
		*h = JSONAny(standardizedBytes)
	}
	return err
}

func (h JSONAny) MarshalJSON() ([]byte, error) {
	if h == "" {
		h = NullString
	}
	return []byte(h), nil
}

func (h *JSONAny) Hash() *Bytes32 {
	if h == nil {
		return nil
	}
	var b32 Bytes32 = sha256.Sum256([]byte(*h))
	return &b32
}

func (h *JSONAny) String() string {
	if h == nil {
		return NullString
	}
	b, _ := h.MarshalJSON()
	return string(b)
}

func (h *JSONAny) Length() int64 {
	if h == nil {
		return 0
	}
	return int64(len(*h))
}

func (h *JSONAny) Bytes() []byte {
	if h == nil {
		return nil
	}
	return []byte(*h)
}

func (h *JSONAny) IsNil() bool {
	return h == nil || *h == "" || *h == NullString
}

func (h JSONAny) JSONObjectOk(noWarn ...bool) (JSONObject, bool) {
	var jo JSONObject
	err := json.Unmarshal([]byte(h), &jo)
	if err != nil {
		if len(noWarn) == 0 || !noWarn[0] {
			log.L(context.Background()).Warnf("Unable to deserialize as JSON object: %s", string(h))
		}
		jo = JSONObject{}
	}
	return jo, err == nil
}

// JSONObject attempts to de-serailize the contained structure as a JSON Object (map)
// Safe and will never return nil
// Will return an empty object if the type is array, string, bool, number etc.
func (h JSONAny) JSONObject() JSONObject {
	jo, _ := h.JSONObjectOk()
	return jo
}

// JSONObjectNowarn acts the same as JSONObject, but does not warn if the value cannot
// be parsed as an object
func (h JSONAny) JSONObjectNowarn() JSONObject {
	jo, _ := h.JSONObjectOk(true)
	return jo
}

// Scan implements sql.Scanner
func (h *JSONAny) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		*h = NullString
		return nil
	case []byte:
		return h.UnmarshalJSON(src)
	case string:
		return h.UnmarshalJSON([]byte(src))
	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, h)
	}
}
