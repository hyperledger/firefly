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

package core

import (
	"database/sql/driver"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFFEnumStringCompareEtc(t *testing.T) {

	txtype1 := TransactionType("Batch_Pin")
	assert.True(t, TransactionTypeBatchPin.Equals(txtype1))
	assert.Equal(t, TransactionTypeBatchPin, txtype1.Lower())

	var tdv driver.Valuer = txtype1
	v, err := tdv.Value()
	assert.Nil(t, err)
	assert.Equal(t, "batch_pin", v)

	var utStruct struct {
		TXType TransactionType `json:"txType"`
	}
	err = json.Unmarshal([]byte(`{"txType": "Batch_PIN"}`), &utStruct)
	assert.NoError(t, err)
	assert.Equal(t, "batch_pin", string(utStruct.TXType))

}

func TestFFEnumValues(t *testing.T) {
	assert.Equal(t, FFEnum("test1"), ffEnum("ut", "test1"))
	assert.Equal(t, FFEnum("test2"), ffEnum("ut", "test2"))
	assert.Equal(t, []interface{}{"test1", "test2"}, FFEnumValues("ut"))
}
