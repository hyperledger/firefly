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

package fftypes

import (
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/assert"
)

func TestGetBaseFFIParamValidator(t *testing.T) {
	c := NewFFISchemaCompiler()
	assert.NotNil(t, c)
}
func TestBaseFFIParamValidatorCompile(t *testing.T) {
	v := BaseFFIParamValidator{}
	c, err := v.Compile(jsonschema.CompilerContext{}, map[string]interface{}{})
	assert.Nil(t, c)
	assert.NoError(t, err)
}
