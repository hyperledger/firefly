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

package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestInitConfigOK(t *testing.T) {
	viper.Reset()
	err := ReadConfig()
	assert.Regexp(t, "Not Found", err.Error())
}

func TestDefaults(t *testing.T) {
	os.Chdir("../../test/config")
	err := ReadConfig()
	assert.NoError(t, err)

	assert.Equal(t, "info", GetString(LogLevel))
	assert.True(t, GetBool(LogColor))
}

func TestSet(t *testing.T) {
	Set("any.key", "any.value")
	assert.Equal(t, GetString("any.key"), "any.value")
}
