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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransactionHash(t *testing.T) {
	batchid := MustParseUUID("39296b6e-91b9-4a61-b279-833c85b04d94")
	tx := &Transaction{}
	tx.Subject = TransactionSubject{
		Signer:    "0x12345",
		Type:      TransactionTypeBatchPin,
		Reference: batchid,
	}
	assert.Equal(t, "43ee7fc01a0bf867c2fb55858174d4597079fb85942f74ab7b6bc785382e35f1", tx.Subject.Hash().String())
}
