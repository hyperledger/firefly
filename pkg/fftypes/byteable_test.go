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

	err := json.Unmarshal([]byte(`{"prop":{"b":"test","b":"duplicate","a":12345,"c":{"e":1.00000000001,"d":false}}}`), &ts)
	assert.NoError(t, err)
	b, err := json.Marshal(&ts)
	assert.NoError(t, err)
	assert.Equal(t, `{"prop":{"b":"test","b":"duplicate","a":12345,"c":{"e":1.00000000001,"d":false}}}`, string(b))
	err = json.Unmarshal([]byte(`{"prop"
		:{"b":"test", "b":"duplicate","a" 				:12345,"c":{
			  "e":1.00000000001,
		"d":false}
	}}`), &ts)
	b, err = json.Marshal(&ts)
	assert.NoError(t, err)
	assert.Equal(t, `{"prop":{"b":"test","b":"duplicate","a":12345,"c":{"e":1.00000000001,"d":false}}}`, string(b))
	b, err = json.Marshal(&ts.Prop)
	assert.Equal(t, `{"b":"test","b":"duplicate","a":12345,"c":{"e":1.00000000001,"d":false}}`, string(b))
	assert.Equal(t, "8eff3083f052a77bda0934236bf8e5eccbd186d5ae81ada7a5bbee516ecd5726", ts.Prop.Hash().String())

	jo, ok := ts.Prop.JSONObjectOk()
	assert.True(t, ok)
	assert.Equal(t, "duplicate", jo.GetString("b"))

}

func TestByteableMarshalNull(t *testing.T) {

	var pb Byteable
	b, err := pb.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, NullString, string(b))
}

func TestByteableUnmarshalFail(t *testing.T) {

	var b Byteable
	err := b.UnmarshalJSON([]byte(`!json`))
	assert.Error(t, err)

	jo := b.JSONObject()
	assert.Equal(t, JSONObject{}, jo)
}

func TestScan(t *testing.T) {

	var h Byteable
	assert.NoError(t, h.Scan(nil))
	assert.Equal(t, []byte(NullString), []byte(h))

	assert.NoError(t, h.Scan(`{"some": "stuff"}`))
	assert.Equal(t, "stuff", h.JSONObject().GetString("some"))

	assert.NoError(t, h.Scan([]byte(`{"some": "stuff"}`)))
	assert.Equal(t, "stuff", h.JSONObject().GetString("some"))

	assert.NoError(t, h.Scan(`"plainstring"`))
	assert.Equal(t, "", h.JSONObjectNowarn().GetString("some"))

	assert.Regexp(t, "FF10125", h.Scan(12345))

}
