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
	"crypto/sha256"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSealBareMessage(t *testing.T) {
	msg := MessageRefsOnly{}
	msg.Seal()
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
			{ID: &data1},
			{ID: &data2},
			{ID: nil /* ignored */},
			{ID: &data3},
		},
	}
	msg.Seal()

	// Data IDs have been sorted for hash and JSON encoded
	dataHashData := `["189c8185-2b92-481a-847a-e57595ab3541","cc66b23f-d340-4333-82d5-b63adc1c3c07","e3a3b714-7e49-4c73-a4ea-87a50b19961a"]`
	var dataHash Bytes32 = sha256.Sum256([]byte(dataHashData))
	assert.Equal(t, `c26f9e0616d934e5d9cc805c4cb100f5b8b9e7a704a31f8d7db82699c311922e`, dataHash.String())
	assert.Equal(t, dataHash, *msg.Header.DataHash)

	// Header contains the data hash, and is hashed into the message hash
	actualHeader, _ := json.Marshal(&msg.Header)
	expectedHeader := `{"id":"2cd37805-5f40-4e12-962e-67868cde3049","cid":"39296b6e-91b9-4a61-b279-833c85b04d94","type":"private","author":"0x12345","created":1620104103000,"namespace":"ns1","topic":"topic1","context":"context1","group":"5cd8afa6-f483-42f1-b11b-5a6f6421c81d","datahash":"c26f9e0616d934e5d9cc805c4cb100f5b8b9e7a704a31f8d7db82699c311922e"}`
	var msgHash Bytes32 = sha256.Sum256([]byte(expectedHeader))
	assert.Equal(t, expectedHeader, string(actualHeader))
	assert.Equal(t, `6c101b593c7303173378b80bacb85fbe482fef592a2d76a54dbc65fbf3120ed3`, msgHash.String())
	assert.Equal(t, msgHash, *msg.Hash)
}
