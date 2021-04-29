// Copyright © 2021 Kaleido, Inc.
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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestConfigInterfaceCorrect(t *testing.T) {
	assert.Regexp(t, "[a-zA-Z0-9_]{8}", ShortID())
}

func TestUnmarshalBytes32(t *testing.T) {
	b := NewRandB32()
	var empty Bytes32
	hexString := hex.EncodeToString(b[0:32])
	emptyJson := []byte(`{}`)
	jsonWithValues := []byte(fmt.Sprintf(`{"value":"%s","valuePtr":"0x%s"}`, hexString, hexString))
	type testStruct struct {
		Value    Bytes32  `json:"value"`
		ValuePtr *Bytes32 `json:"valuePtr,omitempty"`
	}
	fmt.Printf("with: '%s' without: '%s'", jsonWithValues, emptyJson)
	var s1, s2 testStruct
	err := json.Unmarshal(emptyJson, &s1)
	assert.NoError(t, err)
	assert.Equal(t, empty, s1.Value)
	assert.Nil(t, s1.ValuePtr)
	err = json.Unmarshal(jsonWithValues, &s2)
	assert.NoError(t, err)
	assert.Equal(t, b, s2.Value)
	assert.Equal(t, b, *s2.ValuePtr)
}

func TestMarshalBytes32(t *testing.T) {
	b := NewRandB32()
	hexString := b.String()
	type testStruct struct {
		Value    Bytes32  `json:"value"`
		ValuePtr *Bytes32 `json:"valuePtr,omitempty"`
	}
	structWithValues := &testStruct{
		ValuePtr: &b,
		Value:    b,
	}
	json1, err := json.Marshal(structWithValues)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(`{"value":"%s","valuePtr":"%s"}`, hexString, hexString), string(json1))

	structWithoutValues := &testStruct{}
	json2, err := json.Marshal(structWithoutValues)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(`{"value":"0000000000000000000000000000000000000000000000000000000000000000"}`), string(json2))
}

func TestHexUUIDFromUUID(t *testing.T) {
	u := uuid.Must(uuid.NewRandom())
	b := HexUUIDFromUUID(u)
	var dec [16]byte
	hex.Decode(dec[0:16], b[0:32])
	assert.Equal(t, dec[0:16], u[0:16])
	assert.Equal(t, u.String(), uuid.UUID(dec).String())
}
