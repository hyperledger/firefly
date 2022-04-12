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

package config

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/i18n"
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

	key1 := AddRootKey("key1")
	key2 := AddRootKey("key3")
	key3 := AddRootKey("key3")
	key4 := AddRootKey("key4")
	key5 := AddRootKey("key5")
	key6 := AddRootKey("key6")
	key7 := AddRootKey("key7")
	key8 := AddRootKey("key8")
	key9 := AddRootKey("key9")

	RootConfigReset(func() {
		viper.SetDefault(string(key1), "value1")
		viper.SetDefault(string(key2), true)
		viper.SetDefault(string(key3), 25)
		viper.SetDefault(string(key5), "250ms")
		viper.SetDefault(string(key6), "2.0")
		viper.SetDefault(string(key7), []string{"value1", "value2"})
		viper.SetDefault(string(key8), fftypes.JSONObjectArray{{"key1": "value1"}})
		viper.SetDefault(string(key9), "1Mb")
	})
	err = ReadConfig("")
	assert.NoError(t, err)

	assert.Equal(t, "value1", GetString(key1))
	assert.True(t, GetBool(key2))
	assert.Equal(t, uint(25), GetUint(key3))
	assert.Equal(t, int(0), GetInt(key4))
	assert.Equal(t, int64(0), GetInt64(key4))
	assert.Equal(t, 250*time.Millisecond, GetDuration(key5))
	assert.Equal(t, float64(2.0), GetFloat64(key6))
	assert.Equal(t, []string{"value1", "value2"}, GetStringSlice(key7))
	assert.NotEmpty(t, GetObjectArray(key8))
	assert.Equal(t, int64(1024*1024), GetByteSize(key9))
}

func TestSpecificConfigFileOk(t *testing.T) {
	RootConfigReset()
	err := ReadConfig(configDir + "/firefly.core.yaml")
	assert.NoError(t, err)
}

func TestSpecificConfigFileFail(t *testing.T) {
	RootConfigReset()
	err := ReadConfig(configDir + "/no.hope.yaml")
	assert.Error(t, err)
}

func TestAttemptToAccessRandomKey(t *testing.T) {
	assert.Panics(t, func() {
		GetString("any.key")
	})
}

func TestSetGetMap(t *testing.T) {
	defer RootConfigReset()
	key1 := AddRootKey("key1")
	Set(key1, map[string]interface{}{"some": "map"})
	assert.Equal(t, fftypes.JSONObject{"some": "map"}, GetObject(key1))
}

func TestSetGetRawInterace(t *testing.T) {
	defer RootConfigReset()
	type myType struct{ name string }
	key1 := AddRootKey("key1")
	Set(key1, &myType{name: "test"})
	v := Get(key1)
	assert.Equal(t, myType{name: "test"}, *(v.(*myType)))
}

func TestGetBadDurationMillisDefault(t *testing.T) {
	defer RootConfigReset()
	key1 := AddRootKey("key1")
	Set(key1, "12345")
	assert.Equal(t, time.Duration(12345)*time.Millisecond, GetDuration(key1))
}

func TestGetBadDurationZero(t *testing.T) {
	defer RootConfigReset()
	key1 := AddRootKey("key1")
	Set(key1, "!a number or duration")
	assert.Equal(t, time.Duration(0), GetDuration(key1))
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

func TestArrayOfPlugins(t *testing.T) {
	defer RootConfigReset()

	tokPlugins := NewPluginConfig("tokens").Array()
	tokPlugins.AddKnownKey("name")
	tokPlugins.AddKnownKey("key1", "default value")
	tokPlugins.AddKnownKey("key2", "def1", "def2")
	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
tokens:
- name: bob
- name: sally
  key1: explicit value
  key2:
  - arr1
  - arr2
`))
	assert.NoError(t, err)
	assert.Equal(t, 2, tokPlugins.ArraySize())
	assert.Equal(t, 0, NewPluginConfig("nonexistent").Array().ArraySize())
	bob := tokPlugins.ArrayEntry(0)
	assert.Equal(t, "bob", bob.GetString("name"))
	assert.Equal(t, "default value", bob.GetString("key1"))
	assert.Equal(t, []string{"def1", "def2"}, bob.GetStringSlice("key2"))
	sally := tokPlugins.ArrayEntry(1)
	assert.Equal(t, "sally", sally.GetString("name"))
	assert.Equal(t, "explicit value", sally.GetString("key1"))
	assert.Equal(t, []string{"arr1", "arr2"}, sally.GetStringSlice("key2"))
}

func TestMapOfAdminOverridePlugins(t *testing.T) {
	defer RootConfigReset()

	tokPlugins := NewPluginConfig("tokens").Array()
	tokPlugins.AddKnownKey("firstkey")
	tokPlugins.AddKnownKey("secondkey")
	viper.SetConfigType("json")
	err := viper.ReadConfig(strings.NewReader(`{
		"tokens": {
			"0": {
				"firstkey": "firstitemfirstkeyvalue",
				"secondkey": "firstitemsecondkeyvalue"
			},
			"1": {
				"firstkey": "seconditemfirstkeyvalue",
				"secondkey": "seconditemsecondkeyvalue"
			}
		}
	}`))
	assert.NoError(t, err)
	assert.Equal(t, 2, tokPlugins.ArraySize())
	assert.Equal(t, "firstitemfirstkeyvalue", tokPlugins.ArrayEntry(0).Get("firstkey"))
	assert.Equal(t, "seconditemsecondkeyvalue", tokPlugins.ArrayEntry(1).Get("secondkey"))
}

func TestGetKnownKeys(t *testing.T) {
	knownKeys := GetKnownKeys()
	assert.NotEmpty(t, knownKeys)
	for _, k := range knownKeys {
		assert.NotEmpty(t, root.Resolve(k))
	}
}

func TestSetupLoggingToFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logtest")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	fileName := path.Join(tmpDir, "test.log")
	RootConfigReset()
	Set(LogFilename, fileName)
	Set(LogLevel, "debug")
	Set(LogMaxAge, "72h")
	SetupLogging(context.Background())

	RootConfigReset()
	SetupLogging(context.Background())

	b, err := ioutil.ReadFile(fileName)
	assert.NoError(t, err)
	assert.Regexp(t, "Log level", string(b))
}

func TestSetupLogging(t *testing.T) {
	SetupLogging(context.Background())
}

func TestMergeConfigOk(t *testing.T) {

	conf1 := fftypes.JSONAnyPtr(`{
		"some":  {
			"nested": {
				"stuff": "value1"
			}
		}
	}`)
	confNumber := fftypes.JSONAnyPtr(`{
		"some":  {
			"more": {
				"stuff": 15
			}
		}
	}`)
	conf3 := fftypes.JSONAnyPtr(`"value3"`)
	confNestedSlice := fftypes.JSONAnyPtr(`{
		"nestedslice": [
			{
				"firstitemfirstkey": "firstitemfirstkeyvalue",
				"firstitemsecondkey": "firstitemsecondkeyvalue"
			},
			{
				"seconditemfirstkey": "seconditemfirstkeyvalue",
				"seconditemsecondkey": "seconditemsecondkeyvalue"
			}
		]
	}`)
	confBaseSlice := fftypes.JSONAnyPtr(`[
		{
			"firstitemfirstkey": "firstitemfirstkeyvalue",
			"firstitemsecondkey": "firstitemsecondkeyvalue"
		},
		{
			"seconditemfirstkey": "seconditemfirstkeyvalue",
			"seconditemsecondkey": "seconditemsecondkeyvalue"
		}
	]`)

	viper.Reset()
	viper.Set("base.something", "value4")
	err := MergeConfig([]*fftypes.ConfigRecord{
		{Key: "base", Value: conf1},
		{Key: "base", Value: confNumber},
		{Key: "base.some.plain", Value: conf3},
		{Key: "base", Value: confNestedSlice},
		{Key: "base.slice", Value: confBaseSlice},
	})
	assert.NoError(t, err)

	assert.Equal(t, "value1", viper.Get("base.some.nested.stuff"))
	assert.Equal(t, 15, viper.GetInt("base.some.more.stuff"))
	assert.Equal(t, "value3", viper.Get("base.some.plain"))
	assert.Equal(t, "value4", viper.Get("base.something"))
	assert.Equal(t, "firstitemfirstkeyvalue", viper.Get("base.nestedslice.0.firstitemfirstkey"))
	assert.Equal(t, "seconditemsecondkeyvalue", viper.Get("base.nestedslice.1.seconditemsecondkey"))
	assert.Equal(t, "firstitemfirstkeyvalue", viper.Get("base.slice.0.firstitemfirstkey"))
	assert.Equal(t, "seconditemsecondkeyvalue", viper.Get("base.slice.1.seconditemsecondkey"))

}

func TestMergeConfigBadJSON(t *testing.T) {
	err := MergeConfig([]*fftypes.ConfigRecord{
		{Key: "base", Value: fftypes.JSONAnyPtr(`!json`)},
	})
	assert.Error(t, err)
}

func TestGetConfig(t *testing.T) {
	RootConfigReset()
	conf := GetConfig()
	assert.Equal(t, "info", conf.GetObject("log").GetString("level"))
}

func TestGenerateConfigMarkdown(t *testing.T) {

	key1 := AddRootKey("level1_1.level2_1.level3_1")
	key2 := AddRootKey("level1_1.level2_1.level3_2")
	key3 := AddRootKey("level1_1.level2_2.level3")
	key4 := AddRootKey("level1_2.level2.level3")

	i18n.FFM(fmt.Sprintf("config.%s", key1), "Description 1")
	i18n.FFM(fmt.Sprintf("config.%s", key2), "Description 2")
	i18n.FFM("config.global.level2_2.level3", "Description 3")
	i18n.FFM(fmt.Sprintf("config.%s", key4), "Description 4")

	RootConfigReset(func() {
		viper.SetDefault(string(key1), "val1")
		viper.SetDefault(string(key2), "val2")
		viper.SetDefault(string(key3), "val3")
		viper.SetDefault(string(key4), "val4")
	})

	_, err := GenerateConfigMarkdown(context.Background(), []string{
		string(key1), string(key2), string(key3), string(key4),
	})
	assert.NoError(t, err)

}

func TestGenerateConfigMarkdownPanic(t *testing.T) {

	key1 := AddRootKey("bad.key.no.description")

	RootConfigReset()

	assert.Panics(t, func() {
		_, _ = GenerateConfigMarkdown(context.Background(), []string{
			string(key1),
		})
	})

}
