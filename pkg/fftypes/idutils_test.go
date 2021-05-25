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

func TestShortIDGen(t *testing.T) {
	assert.Regexp(t, "[a-zA-Z0-9_]{8}", ShortID())
}

func TestSafeHashCompare(t *testing.T) {
	assert.False(t, SafeHashCompare(nil, NewRandB32()))
	assert.False(t, SafeHashCompare(NewRandB32(), nil))
	assert.True(t, SafeHashCompare(nil, nil))
	assert.False(t, SafeHashCompare(NewRandB32(), NewRandB32()))
	r1 := NewRandB32()
	var r2 Bytes32 = *r1
	assert.True(t, SafeHashCompare(r1, &r2))
}
