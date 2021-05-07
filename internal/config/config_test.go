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
	"context"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestInitConfigOK(t *testing.T) {
	viper.Reset()
	err := ReadConfig("")
	assert.Regexp(t, "Not Found", err.Error())
}

func TestDefaults(t *testing.T) {
	os.Chdir("../../test/config")
	err := ReadConfig("")
	assert.NoError(t, err)

	assert.Equal(t, "info", GetString(LogLevel))
	assert.True(t, GetBool(LogColor))
	assert.Equal(t, uint(0), GetUint(HttpPort))
	assert.Equal(t, int(0), GetInt(DebugPort))
	assert.Equal(t, []string{"*"}, GetStringSlice(CorsAllowedOrigins))
}

func TestSpecificConfigFileOk(t *testing.T) {
	err := ReadConfig("../../test/config/firefly.core.yaml")
	assert.NoError(t, err)
}

func TestSpecificConfigFileFail(t *testing.T) {
	err := ReadConfig("../../test/config/no.hope.yaml")
	assert.Error(t, err)
}

func TestSet(t *testing.T) {
	Set("any.key", "any.value")
	assert.Equal(t, GetString("any.key"), "any.value")
}

func TestUintWithDefault(t *testing.T) {
	assert.Equal(t, uint(10), UintWithDefault(nil, 10))
	var v uint = 10
	assert.Equal(t, uint(10), UintWithDefault(&v, 99))
}

func TestBoolWithDefault(t *testing.T) {
	assert.True(t, BoolWithDefault(nil, true))
	var v bool = false
	assert.False(t, BoolWithDefault(&v, true))
}

func TestUnmarshalKey(t *testing.T) {
	err := ReadConfig("../../test/config/firefly.core.yaml")
	assert.NoError(t, err)
	var conf map[string]interface{}
	err = UnmarshalKey(context.Background(), Blockchain, &conf)
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:8000", conf["ethconnect"].(map[string]interface{})["url"])
}

func TestUnmarshalKeyFail(t *testing.T) {
	err := ReadConfig("../../test/config/firefly.core.yaml")
	assert.NoError(t, err)
	var conf map[string]interface{}
	err = UnmarshalKey(context.Background(), HttpPort, &conf)
	assert.Regexp(t, "FF10101", err)
}
