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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestByteableSerializeNull(t *testing.T) {

	type testStruct struct {
		Prop1 *Byteable `json:"prop1"`
		Prop2 *Byteable `json:"prop2,omitempty"`
	}

	ts := &testStruct{}

	err := json.Unmarshal([]byte(`{}`), &ts)
	assert.NoError(t, err)
	assert.Nil(t, ts.Prop1)
	assert.Nil(t, ts.Prop2)
	b, err := json.Marshal(&ts)
	assert.Equal(t, `{"prop1":null}`, string(b))

}

func TestByteableSerializeObjects(t *testing.T) {

	type testStruct struct {
		Prop *Byteable `json:"prop,omitempty"`
	}

	ts := &testStruct{}

	err := json.Unmarshal([]byte(`{"prop": {
		"a": 12345,
		"b": "test",
		"c": {
			"d": false,
			"e": 1.00000000001
		}
	}}`), &ts)
	assert.NoError(t, err)
	b, err := json.Marshal(&ts)
	assert.Equal(t, `{"prop":{"a":12345,"b":"test","c":{"d":false,"e":1.00000000001}}}`, string(b))
	b, err = json.Marshal(&ts.Prop)
	assert.Equal(t, `{"a":12345,"b":"test","c":{"d":false,"e":1.00000000001}}`, string(b))
	assert.Equal(t, "5aa173f985912461abe6bad6408ca97efd972a95f0f51ec22badd8118ffdd819", ts.Prop.Hash().String())

	jo, err := ts.Prop.JSONObject()
	assert.NoError(t, err)
	assert.Equal(t, "test", jo.GetString(context.Background(), "b"))

}

func TestByteableUnmarshalFail(t *testing.T) {

	var b Byteable
	err := b.UnmarshalJSON([]byte(`!json`))
	assert.Error(t, err)

}
