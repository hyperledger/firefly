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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFFNameArrayVerifyTooLong(t *testing.T) {
	na := make(FFNameArray, 16)
	for i := 0; i < 16; i++ {
		na[i] = fmt.Sprintf("item_%d", i)
	}
	err := na.Validate(context.Background(), "field1")
	assert.Regexp(t, `FF10227.*field1`, err)
}

func TestFFNameArrayVerifyBadName(t *testing.T) {
	na := FFNameArray{"!valid"}
	err := na.Validate(context.Background(), "field1")
	assert.Regexp(t, `FF10131.*field1\[0\]`, err)
}

func TestFFNameArrayScanValue(t *testing.T) {

	na1 := FFNameArray{"name1", "name2"}
	v, err := na1.Value()
	assert.NoError(t, err)
	assert.Equal(t, "name1,name2", v)

	var na2 FFNameArray
	assert.Equal(t, "", na2.String())
	v, err = na2.Value()
	assert.Equal(t, "", v)
	err = na2.Scan("name1,name2")
	assert.NoError(t, err)
	assert.Equal(t, "name1,name2", na2.String())

	var na3 FFNameArray
	err = na3.Scan([]byte("name1,name2"))
	assert.NoError(t, err)
	assert.Equal(t, "name1,name2", na3.String())

	var na4 FFNameArray
	err = na4.Scan([]byte(nil))
	assert.NoError(t, err)
	assert.Equal(t, "", na4.String())
	err = na4.Scan(nil)
	assert.NoError(t, err)
	assert.Equal(t, "", na4.String())
	v, err = na4.Value()
	assert.NoError(t, err)
	assert.Equal(t, "", v)

	var na5 FFNameArray
	err = na5.Scan("")
	assert.NoError(t, err)
	assert.Equal(t, FFNameArray{}, na5)
	assert.Equal(t, "", na5.String())

	var na6 FFNameArray
	err = na6.Scan(42)
	assert.Regexp(t, "FF10125", err)

	var na7 FFNameArray
	err = na7.Scan(FFNameArray{"test1", "test2"})
	assert.Equal(t, FFNameArray{"test1", "test2"}, na7)

}
