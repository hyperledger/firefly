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
	"context"
	"crypto/sha256"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSealBareMessage(t *testing.T) {
	msg := MessageRefsOnly{}
	err := msg.Seal(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, msg.Header.ID)
	assert.NotNil(t, msg.Header.DataHash)
	assert.NotNil(t, msg.Hash)
}

func TestSealKnownMessage(t *testing.T) {
	msgid := uuid.MustParse("2cd37805-5f40-4e12-962e-67868cde3049")
	cid := uuid.MustParse("39296b6e-91b9-4a61-b279-833c85b04d94")
	gid := uuid.MustParse("5cd8afa6-f483-42f1-b11b-5a6f6421c81d")
	data1 := uuid.MustParse("e3a3b714-7e49-4c73-a4ea-87a50b19961a")
	data2 := uuid.MustParse("cc66b23f-d340-4333-82d5-b63adc1c3c07")
	data3 := uuid.MustParse("189c8185-2b92-481a-847a-e57595ab3541")
	var hash1, hash2, hash3 Bytes32
	hash1.UnmarshalText([]byte("3fcc7e07069e441f07c9f6b26f16fcb2dc896222d72888675082fd308440d9ae"))
	hash2.UnmarshalText([]byte("1d1462e02d7acee49a8448267c65067e0bec893c9a0c050b9835efa376fec046"))
	hash3.UnmarshalText([]byte("284b535da66aa0734af56c708426d756331baec3bce3079e508003bcf4738ee6"))
	msg := MessageRefsOnly{
		Header: MessageHeader{
			ID:        &msgid,
			CID:       &cid,
			Type:      MessageTypePrivate,
			Author:    "0x12345",
			Namespace: "ns1",
			Topic:     "topic1",
			Context:   "context1",
			Created:   1620104103000,
			Group:     &gid,
		},
		Data: DataRefSortable{
			{ID: &data1, Hash: &hash1},
			{ID: &data2, Hash: &hash2},
			{ID: &data3, Hash: &hash3},
		},
	}
	err := msg.Seal(context.Background())
	assert.NoError(t, err)

	// Data IDs have been sorted for hash and JSON encoded
	dataHashData := `["1d1462e02d7acee49a8448267c65067e0bec893c9a0c050b9835efa376fec046","284b535da66aa0734af56c708426d756331baec3bce3079e508003bcf4738ee6","3fcc7e07069e441f07c9f6b26f16fcb2dc896222d72888675082fd308440d9ae"]`
	var dataHash Bytes32 = sha256.Sum256([]byte(dataHashData))
	assert.Equal(t, `0d13e20a54f00f0c044c586022a3483ae830b076b0ae9a971c306c788afde9e9`, dataHash.String())
	assert.Equal(t, dataHash, *msg.Header.DataHash)

	// Header contains the data hash, and is hashed into the message hash
	actualHeader, _ := json.Marshal(&msg.Header)
	expectedHeader := `{"id":"2cd37805-5f40-4e12-962e-67868cde3049","cid":"39296b6e-91b9-4a61-b279-833c85b04d94","type":"private","author":"0x12345","created":1620104103000,"namespace":"ns1","topic":"topic1","context":"context1","group":"5cd8afa6-f483-42f1-b11b-5a6f6421c81d","datahash":"0d13e20a54f00f0c044c586022a3483ae830b076b0ae9a971c306c788afde9e9"}`
	var msgHash Bytes32 = sha256.Sum256([]byte(expectedHeader))
	assert.Equal(t, expectedHeader, string(actualHeader))
	assert.Equal(t, `37e7ebcf33abe9239ef119dfb74711530d11e222acfdae0c78cbf3ecc6891cfa`, msgHash.String())
	assert.Equal(t, msgHash, *msg.Hash)
}
