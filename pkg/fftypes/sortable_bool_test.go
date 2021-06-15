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

func TestSortableBoolScanValue(t *testing.T) {

	sb := SortableBool(true)
	sv, err := sb.Value()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), sv)
	sb = SortableBool(false)
	sv, err = sb.Value()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), sv)

	assert.NoError(t, sb.Scan(int64(1)))
	assert.True(t, bool(sb))
	assert.NoError(t, sb.Scan(int64(0)))
	assert.False(t, bool(sb))
	assert.NoError(t, sb.Scan(true))
	assert.True(t, bool(sb))

	assert.NoError(t, sb.Scan("false"))
	assert.False(t, bool(sb))
	assert.NoError(t, sb.Scan("True"))
	assert.True(t, bool(sb))

	assert.NoError(t, sb.Scan(float64(12345)))
	assert.False(t, bool(sb))
}
