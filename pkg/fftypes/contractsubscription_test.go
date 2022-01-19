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

func TestFFISerializedEventScan(t *testing.T) {
	params := &FFISerializedEvent{}
	err := params.Scan([]byte(`{"name":"event1","description":"a super event","params":[{"name":"details","type":"integer","details":{"type":"uint256"}}]}`))
	assert.NoError(t, err)
}

func TestFFISerializedEventScanNil(t *testing.T) {
	params := &FFISerializedEvent{}
	err := params.Scan(nil)
	assert.Nil(t, err)
}

func TestFFISerializedEventScanString(t *testing.T) {
	params := &FFISerializedEvent{}
	err := params.Scan(`{"name":"event1","description":"a super event","params":[{"name":"details","type":"integer","details":{"type":"uint256"}}]}`)
	assert.NoError(t, err)
}

func TestFFISerializedEventScanError(t *testing.T) {
	params := &FFISerializedEvent{}
	err := params.Scan(map[string]interface{}{"this is": "not a supported serialization of a FFISerializedEvent"})
	assert.Regexp(t, "FF10125", err)
}

func TestFFISerializedEventValue(t *testing.T) {
	params := &FFISerializedEvent{
		FFIEventDefinition: FFIEventDefinition{
			Name:        "event1",
			Description: "a super event",
			Params: FFIParams{
				&FFIParam{Name: "details", Type: "integer", Details: JSONAnyPtr(`{"type": "uint256"}`)},
			},
		},
	}

	val, err := params.Value()
	assert.NoError(t, err)
	assert.Equal(t, `{"name":"event1","description":"a super event","params":[{"name":"details","type":"integer","details":{"type":"uint256"}}]}`, string(val.([]byte)))
}
