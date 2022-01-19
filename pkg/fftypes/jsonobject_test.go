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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONObject(t *testing.T) {

	data := JSONObject{
		"some": "data",
	}

	b, err := data.Value()
	assert.NoError(t, err)
	assert.IsType(t, "", b)

	var dataRead JSONObject
	err = dataRead.Scan(b)
	assert.NoError(t, err)

	assert.Equal(t, `{"some":"data"}`, fmt.Sprintf("%v", dataRead))

	j1, err := json.Marshal(&data)
	assert.NoError(t, err)
	j2, err := json.Marshal(&dataRead)
	assert.NoError(t, err)
	assert.Equal(t, string(j1), string(j2))
	j3 := dataRead.String()
	assert.Equal(t, string(j1), j3)

	err = dataRead.Scan("")
	assert.NoError(t, err)

	err = dataRead.Scan(nil)
	assert.NoError(t, err)

	var wrongType int
	err = dataRead.Scan(&wrongType)
	assert.Error(t, err)

	hash, err := dataRead.Hash("goodStuff")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	var badJson JSONObject = map[string]interface{}{"not": map[bool]string{true: "json"}}
	hash, err = badJson.Hash("badStuff")
	assert.Regexp(t, "FF10151.*badStuff", err)
	assert.Nil(t, hash)

	v, ok := JSONObject{"test": false}.GetStringOk("test")
	assert.True(t, ok)
	assert.Equal(t, "false", v)

	v, ok = JSONObject{"test": float64(12345)}.GetStringOk("test")
	assert.True(t, ok)
	assert.Equal(t, "12345", v)

	v, ok = JSONObject{}.GetStringOk("test")
	assert.False(t, ok)
	assert.Equal(t, "", v)

	v, ok = JSONObject{"test": map[string]int{"x": 12345}}.GetStringOk("test")
	assert.False(t, ok)
	assert.Equal(t, "", v)
}

func TestJSONObjectScan(t *testing.T) {

	data := JSONObject{"some": "data"}

	sv, err := data.Value()
	assert.NoError(t, err)
	assert.Equal(t, "{\"some\":\"data\"}", sv)

	var dataRead JSONObject
	err = dataRead.Scan(sv)
	assert.NoError(t, err)

	assert.Equal(t, `{"some":"data"}`, fmt.Sprintf("%v", dataRead))

	sv, err = ((JSONObject)(nil)).Value()
	assert.NoError(t, err)
	assert.Equal(t, NullString, sv)

	var badData JSONObject = map[string]interface{}{"bad": map[bool]bool{false: true}}
	_, err = badData.Value()
	assert.Error(t, err)

	j1, err := json.Marshal(&data)
	assert.NoError(t, err)
	j2, err := json.Marshal(&dataRead)
	assert.NoError(t, err)
	assert.Equal(t, string(j1), string(j2))
	j3 := dataRead.String()
	assert.Equal(t, string(j1), j3)

	err = dataRead.Scan("")
	assert.NoError(t, err)

	err = dataRead.Scan([]byte("{}"))
	assert.NoError(t, err)

	err = dataRead.Scan(`{"test": true}`)
	assert.NoError(t, err)
	assert.True(t, dataRead.GetBool("test"))

	err = dataRead.Scan(nil)
	assert.NoError(t, err)

	var wrongType int
	err = dataRead.Scan(&wrongType)
	assert.Error(t, err)

	hash, err := dataRead.Hash("goodStuff")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	var badJson JSONObjectArray = []JSONObject{{"not": map[bool]string{true: "json"}}}
	hash, err = badJson.Hash("badStuff")
	assert.Regexp(t, "FF10151.*badStuff", err)
	assert.Nil(t, hash)

}

func TestJSONNestedSafeGet(t *testing.T) {

	var jd JSONObject
	err := json.Unmarshal([]byte(`
		{
			"nested_array": [
				{
					"with": {
						"some": "value"
					}
				}
			],
			"string_array": ["str1","str2"],
			"wrong": null
		}
	`), &jd)
	assert.NoError(t, err)

	va, ok := jd.GetObjectArrayOk("wrong")
	assert.False(t, ok)
	assert.NotNil(t, va)

	vo, ok := jd.GetObjectOk("wrong")
	assert.False(t, ok)
	assert.NotNil(t, vo)
	vo, ok = jd.GetObjectOk("string_array")
	assert.False(t, ok)
	assert.NotNil(t, vo)
	jd["already_typed"] = JSONObject{"some": "stuff"}
	vo, ok = jd.GetObjectOk("already_typed")
	assert.True(t, ok)
	assert.Equal(t, "stuff", vo.GetString("some"))

	assert.Equal(t, "value",
		jd.GetObjectArray("nested_array")[0].
			GetObject("with").
			GetString("some"),
	)

	sa, ok := jd.GetStringArrayOk("wrong")
	assert.False(t, ok)
	assert.Empty(t, sa)

	sa, ok = jd.GetStringArrayOk("string_array")
	assert.True(t, ok)
	assert.Equal(t, []string{"str1", "str2"}, sa)

	assert.Equal(t, []string{"str1", "str2"}, jd.GetStringArray("string_array"))

	sa, ok = ToStringArray(jd.GetStringArray("string_array"))
	assert.True(t, ok)
	assert.Equal(t, []string{"str1", "str2"}, sa)

	remapped, ok := ToJSONObjectArray(jd.GetObjectArray("nested_array"))
	assert.True(t, ok)
	assert.Equal(t, "value",
		remapped[0].
			GetObject("with").
			GetString("some"),
	)

	assert.Equal(t, "",
		jd.GetObject("no").
			GetObject("path").
			GetObject("to").
			GetString("here"),
	)

}
