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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONData(t *testing.T) {

	data := JSONData{
		"some": "data",
	}

	b, err := data.Value()
	assert.NoError(t, err)
	assert.IsType(t, []byte{}, b)

	var dataRead JSONData
	err = dataRead.Scan(b)
	assert.NoError(t, err)

	j1, err := json.Marshal(&data)
	assert.NoError(t, err)
	j2, err := json.Marshal(&dataRead)
	assert.NoError(t, err)
	assert.Equal(t, string(j1), string(j2))
	j3 := dataRead.String()
	assert.Equal(t, string(j1), j3)

	err = dataRead.Scan("")
	assert.NoError(t, err)

	err = dataRead.Scan(nil)
	assert.NoError(t, err)

	var wrongType int
	err = dataRead.Scan(&wrongType)
	assert.Error(t, err)
}
