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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	assert.Equal(t, *b, s2.Value)
	assert.Equal(t, b, s2.ValuePtr)
}

func TestMarshalBytes32(t *testing.T) {
	b := NewRandB32()
	hexString := b.String()
	type testStruct struct {
		Value    Bytes32  `json:"value"`
		ValuePtr *Bytes32 `json:"valuePtr,omitempty"`
	}
	structWithValues := &testStruct{
		ValuePtr: b,
		Value:    *b,
	}
	json1, err := json.Marshal(structWithValues)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(`{"value":"%s","valuePtr":"%s"}`, hexString, hexString), string(json1))

	structWithoutValues := &testStruct{}
	json2, err := json.Marshal(structWithoutValues)
	assert.NoError(t, err)
	assert.Equal(t, `{"value":"0000000000000000000000000000000000000000000000000000000000000000"}`, string(json2))
}

func TestScanBytes32(t *testing.T) {

	b32 := &Bytes32{}

	b32.Scan(nil)
	assert.Equal(t, *b32, Bytes32{})

	b32.Scan("")
	assert.Equal(t, *b32, Bytes32{})

	rand := NewRandB32()
	b32.Scan(rand.String())
	assert.Equal(t, b32, rand)

	b32 = &Bytes32{}

	b32.Scan([]byte{})
	assert.Equal(t, *b32, Bytes32{})

	b32.Scan([]byte(rand.String()))
	assert.Equal(t, b32, rand)

	b32.Scan(rand[:])
	assert.Equal(t, b32, rand)

	err := b32.Scan(12345)
	assert.Error(t, err)

}

func TestValueBytes32(t *testing.T) {
	b32 := NewRandB32()
	s, _ := b32.Value()
	assert.Equal(t, b32.String(), s)

	b32 = nil
	s, _ = b32.Value()
	assert.Nil(t, s)

}

func TestValueBytes32Nil(t *testing.T) {
	var b32 *Bytes32
	assert.Equal(t, "", b32.String())
}

func TestUUIDBytes(t *testing.T) {
	u := NewUUID()
	b := UUIDBytes(u)
	assert.Equal(t, b[0:16], u[0:16])
}

func TestHashResult(t *testing.T) {
	v := []byte(`abcdefghijklmnopqrstuvwxyz`)
	hash := sha256.New()
	_, err := hash.Write(v)
	assert.NoError(t, err)
	h1 := sha256.Sum256(v)
	h2 := HashResult(hash)
	assert.Equal(t, [32]byte(h1), [32]byte(*h2))
}

func TestSafeEqualsBytes32(t *testing.T) {

	var b1, b2 *Bytes32
	assert.True(t, b1.Equals(b2))
	b1 = NewRandB32()
	assert.False(t, b1.Equals(b2))
	var vb2 Bytes32
	vb2 = *b1
	b2 = &vb2
	assert.True(t, b1.Equals(b2))
	b2 = NewRandB32()
	assert.False(t, b1.Equals(b2))

}

func TestParseBytes32(t *testing.T) {
	b32, err := ParseBytes32(context.Background(), "0xd907ee03ecbcfb416ce89d957682e8ef41ac548b0b571f65cb196f2b0ab4da05")
	assert.NoError(t, err)
	assert.Equal(t, "d907ee03ecbcfb416ce89d957682e8ef41ac548b0b571f65cb196f2b0ab4da05", b32.String())
	assert.Equal(t, "d907ee03ecbcfb416ce89d957682e8ef41ac548b0b571f65cb196f2b0ab4da05", MustParseBytes32("0xd907ee03ecbcfb416ce89d957682e8ef41ac548b0b571f65cb196f2b0ab4da05").String())

	_, err = ParseBytes32(context.Background(), "")
	assert.Regexp(t, "FF10232", err)

	_, err = ParseBytes32(context.Background(), "!!!!d907ee03ecbcfb416ce89d957682e8ef41ac548b0b571f65cb196f2b0ab4")
	assert.Regexp(t, "FF10231", err)

	assert.Panics(t, func() { MustParseBytes32("!!!!stuff") })
}
