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

func TestJSONAnySerializeNull(t *testing.T) {

	type testStruct struct {
		Prop1 *JSONAny `json:"prop1"`
		Prop2 *JSONAny `json:"prop2,omitempty"`
	}

	ts := &testStruct{}

	err := json.Unmarshal([]byte(`{}`), &ts)
	assert.NoError(t, err)
	assert.Nil(t, ts.Prop1)
	assert.Nil(t, ts.Prop2)
	b, err := json.Marshal(&ts)
	assert.Equal(t, `{"prop1":null}`, string(b))

}

func TestJSONAnySerializeObjects(t *testing.T) {

	type testStruct struct {
		Prop *JSONAny `json:"prop,omitempty"`
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

	assert.Empty(t, "", ((*JSONAny)(nil)).JSONObject().GetString("notThere"))

}

func TestJSONAnyMarshalNull(t *testing.T) {

	var pb JSONAny
	b, err := pb.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, NullString, string(b))
	assert.Equal(t, NullString, pb.String())
	assert.True(t, pb.IsNil())

	err = pb.UnmarshalJSON([]byte(""))
	assert.NoError(t, err)
	assert.True(t, pb.IsNil())

	var ppb *JSONAny
	assert.Equal(t, NullString, ppb.String())
	assert.True(t, pb.IsNil())

}

func TestJSONAnyUnmarshalFail(t *testing.T) {

	var b JSONAny
	err := b.UnmarshalJSON([]byte(`!json`))
	assert.Error(t, err)

	jo := b.JSONObject()
	assert.Equal(t, JSONObject{}, jo)
}

func TestScan(t *testing.T) {

	var h JSONAny
	assert.Equal(t, int64(0), h.Length())
	assert.NoError(t, h.Scan(nil))
	assert.Equal(t, []byte(NullString), []byte(h))

	assert.NoError(t, h.Scan(`{"some": "stuff"}`))
	assert.Equal(t, "stuff", h.JSONObject().GetString("some"))

	assert.NoError(t, h.Scan([]byte(`{"some": "stuff"}`)))
	assert.Equal(t, "stuff", h.JSONObject().GetString("some"))

	assert.NoError(t, h.Scan(`"plainstring"`))
	assert.Equal(t, "", h.JSONObjectNowarn().GetString("some"))

	assert.Regexp(t, "FF10125", h.Scan(12345))

	assert.Equal(t, "test", JSONAnyPtrBytes([]byte(`{"val": "test"}`)).JSONObject().GetString("val"))
	assert.Nil(t, JSONAnyPtrBytes(nil))
	assert.Equal(t, int64(0), JSONAnyPtrBytes(nil).Length())

	assert.Nil(t, JSONAnyPtrBytes(nil).Bytes())
	assert.NotEmpty(t, JSONAnyPtr("{}").Bytes())
	assert.Equal(t, int64(2), JSONAnyPtr("{}").Length())

}
