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

func TestSQLSerializedMessageArray(t *testing.T) {

	msgID1 := NewUUID()
	msgID2 := NewUUID()
	batchPayload := BatchPayload{
		Messages: []*Message{
			{Header: MessageHeader{ID: msgID1}},
			{Header: MessageHeader{ID: msgID2}},
		},
	}

	b, err := batchPayload.Value()
	assert.NoError(t, err)
	assert.IsType(t, []byte{}, b)

	var batchPayloadRead BatchPayload
	err = batchPayloadRead.Scan(b)
	assert.NoError(t, err)

	j1, err := json.Marshal(&batchPayload)
	assert.NoError(t, err)
	j2, err := json.Marshal(&batchPayloadRead)
	assert.NoError(t, err)
	assert.Equal(t, string(j1), string(j2))

	err = batchPayloadRead.Scan("")
	assert.NoError(t, err)

	err = batchPayloadRead.Scan("{}")
	assert.NoError(t, err)

	err = batchPayloadRead.Scan(nil)
	assert.NoError(t, err)

	var wrongType int
	err = batchPayloadRead.Scan(&wrongType)
	assert.Error(t, err)

	hash := batchPayload.Hash()
	assert.NotNil(t, hash)

}
