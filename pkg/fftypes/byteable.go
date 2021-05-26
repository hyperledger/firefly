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
	"bytes"
	"crypto/sha256"
	"encoding/json"
)

// Byteable de-serializes any valid JSON field type into JSON bytes, that can be restored as they came in.
// Noting that when a JSON object is passed in, things like whitespace and duplicate keys will be lost.
type Byteable []byte

func (h *Byteable) UnmarshalJSON(b []byte) error {
	var src interface{}

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	err := dec.Decode(&src)
	if err != nil {
		return err
	}
	*h, err = json.Marshal(&src)
	return err
}

func (h Byteable) MarshalJSON() ([]byte, error) {
	return []byte(h), nil
}

func (h Byteable) Hash() *Bytes32 {
	var b32 Bytes32 = sha256.Sum256([]byte(h))
	return &b32
}

// JSONObject attempts to de-serailize the contained structure as a JSON Object (map)
// Will fail if the type is array, string, bool, number etc.
func (h Byteable) JSONObject() (*JSONObject, error) {
	var jo *JSONObject
	err := json.Unmarshal(h, &jo)
	return jo, err
}
