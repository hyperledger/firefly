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

package config

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const configDir = "../../test/data/config"

func TestInitConfigOK(t *testing.T) {
	viper.Reset()
	err := ReadConfig("")
	assert.Regexp(t, "Not Found", err)
}

func TestDefaults(t *testing.T) {
	cwd, err := os.Getwd()
	assert.NoError(t, err)
	os.Chdir(configDir)
	defer os.Chdir(cwd)

	err = ReadConfig("")
	assert.NoError(t, err)

	assert.Equal(t, "info", GetString(LogLevel))
	assert.True(t, GetBool(CorsAllowCredentials))
	assert.Equal(t, uint(0), GetUint(HTTPPort))
	assert.Equal(t, int(0), GetInt(DebugPort))
	assert.Equal(t, int64(0), GetInt64(DebugPort))
	assert.Equal(t, 250*time.Millisecond, GetDuration(BatchRetryInitDelay))
	assert.Equal(t, float64(2.0), GetFloat64(EventAggregatorRetryFactor))
	assert.Equal(t, []string{"*"}, GetStringSlice(CorsAllowedOrigins))
	assert.NotEmpty(t, GetObjectArray(NamespacesPredefined))
	assert.Equal(t, int64(1024*1024), GetByteSize(ValidatorCacheSize))
}

func TestSpecificConfigFileOk(t *testing.T) {
	err := ReadConfig(configDir + "/firefly.core.yaml")
	assert.NoError(t, err)
}

func TestSpecificConfigFileFail(t *testing.T) {
	err := ReadConfig(configDir + "/no.hope.yaml")
	assert.Error(t, err)
}

func TestAttemptToAccessRandomKey(t *testing.T) {
	assert.Panics(t, func() {
		GetString("any.key")
	})
}

func TestSetGetMap(t *testing.T) {
	defer Reset()
	Set(BroadcastBatchSize, map[string]interface{}{"some": "map"})
	assert.Equal(t, fftypes.JSONObject{"some": "map"}, GetObject(BroadcastBatchSize))
}

func TestSetGetRawInterace(t *testing.T) {
	defer Reset()
	type myType struct{ name string }
	Set(BroadcastBatchSize, &myType{name: "test"})
	v := Get(BroadcastBatchSize)
	assert.Equal(t, myType{name: "test"}, *(v.(*myType)))
}

func TestGetBadDurationMillisDefault(t *testing.T) {
	defer Reset()
	Set(BroadcastBatchTimeout, "12345")
	assert.Equal(t, time.Duration(12345)*time.Millisecond, GetDuration(BroadcastBatchTimeout))
}

func TestGetBadDurationZero(t *testing.T) {
	defer Reset()
	Set(BroadcastBatchTimeout, "!a number or duration")
	assert.Equal(t, time.Duration(0), GetDuration(BroadcastBatchTimeout))
}

func TestPluginConfig(t *testing.T) {
	pic := NewPluginConfig("my")
	pic.AddKnownKey("special.config", 12345)
	assert.Equal(t, 12345, pic.GetInt("special.config"))
}

func TestPluginConfigArrayInit(t *testing.T) {
	pic := NewPluginConfig("my").SubPrefix("special")
	pic.AddKnownKey("config", "val1", "val2", "val3")
	assert.Equal(t, []string{"val1", "val2", "val3"}, pic.GetStringSlice("config"))
}

func TestGetKnownKeys(t *testing.T) {
	knownKeys := GetKnownKeys()
	assert.NotEmpty(t, knownKeys)
	for _, k := range knownKeys {
		assert.NotEmpty(t, root.Resolve(k))
	}
}

func TestSetupLogging(t *testing.T) {
	SetupLogging(context.Background())
}
