// Copyright © 2022 Kaleido, Inc.
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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIdempotencyKeyNilValuer(t *testing.T) {
	v, err := ((IdempotencyKey)("")).Value()
	assert.NoError(t, err)
	assert.Nil(t, v)

	v, err = ((IdempotencyKey)("testValue")).Value()
	assert.NoError(t, err)
	assert.Equal(t, "testValue", v.(string))
}

func TestIdempotencyKeyNilScanner(t *testing.T) {
	var ik *IdempotencyKey
	err := ik.Scan(nil)
	assert.NoError(t, err)
	assert.Nil(t, ik)

	ik = new(IdempotencyKey)
	err = ik.Scan("testValue")
	assert.NoError(t, err)
	assert.Equal(t, "testValue", (string)(*ik))

	ik = new(IdempotencyKey)
	err = ik.Scan([]byte("testValue2"))
	assert.NoError(t, err)
	assert.Equal(t, "testValue2", (string)(*ik))

	ik = new(IdempotencyKey)
	err = ik.Scan(12345)
	assert.Regexp(t, "FF00105", err)
}
