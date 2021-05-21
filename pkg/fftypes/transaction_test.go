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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransactionHash(t *testing.T) {
	msgid := MustParseUUID("2cd37805-5f40-4e12-962e-67868cde3049")
	batchid := MustParseUUID("39296b6e-91b9-4a61-b279-833c85b04d94")
	tx := &Transaction{}
	tx.Subject = TransactionSubject{
		Author:    "0x12345",
		Namespace: "ns1",
		Type:      TransactionTypePin,
		Message:   msgid,
		Batch:     batchid,
	}
	assert.Equal(t, "32fe939dee0ef781e1cdc685f24d1482551b604116b7f3f3588ab2c7eafebbe5", tx.Subject.Hash().String())
}
