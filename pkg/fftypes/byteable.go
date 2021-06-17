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

package fftypes

import (
	"context"
	"crypto/sha256"
	"encoding/json"

	"github.com/hyperledger-labs/firefly/internal/log"
)

// Byteable uses raw encode/decode to preserve field order, and can handle any types of field.
// It validates the JSON can be unmarshalled, but does not change the order.
// It does however trim out whitespace
type Byteable []byte

func (h *Byteable) UnmarshalJSON(b []byte) error {
	var flattener json.RawMessage
	err := json.Unmarshal(b, &flattener)
	if err != nil {
		return err
	}
	*h, err = json.Marshal(flattener)
	return err
}

func (h Byteable) MarshalJSON() ([]byte, error) {
	if h == nil {
		return []byte("null"), nil
	}
	return h, nil
}

func (h Byteable) Hash() *Bytes32 {
	var b32 Bytes32 = sha256.Sum256([]byte(h))
	return &b32
}

func (h Byteable) String() string {
	b, _ := h.MarshalJSON()
	return string(b)
}

func (h Byteable) JSONObjectOk() (JSONObject, bool) {
	var jo JSONObject
	err := json.Unmarshal(h, &jo)
	if err != nil {
		log.L(context.Background()).Warnf("Unable to deserialize as JSON object: %s", string(h))
		jo = JSONObject{}
	}
	return jo, err == nil
}

// JSONObject attempts to de-serailize the contained structure as a JSON Object (map)
// Will fail if the type is array, string, bool, number etc.
func (h Byteable) JSONObject() JSONObject {
	jo, _ := h.JSONObjectOk()
	return jo
}
