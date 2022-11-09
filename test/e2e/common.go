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

package e2e

import (
	"crypto/rand"
	"fmt"
	"os"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func ReadConfig(t *testing.T, configFile string) map[string]interface{} {
	yfile, err := os.ReadFile(configFile)
	assert.NoError(t, err)
	data := make(map[string]interface{})
	err = yaml.Unmarshal(yfile, &data)
	assert.NoError(t, err)
	return data
}

func WriteConfig(t *testing.T, configFile string, data map[string]interface{}) {
	out, err := yaml.Marshal(data)
	assert.NoError(t, err)
	f, err := os.Create(configFile)
	assert.NoError(t, err)
	_, err = f.Write(out)
	assert.NoError(t, err)
	f.Close()
}

func AddNamespace(data map[string]interface{}, ns map[string]interface{}) {
	if _, ok := data["namespaces"]; !ok {
		data["namespaces"] = make(map[interface{}]interface{})
	}
	namespaces := data["namespaces"].(map[interface{}]interface{})
	if _, ok := namespaces["default"]; !ok {
		namespaces["default"] = "default"
	}
	if _, ok := namespaces["predefined"]; !ok {
		namespaces["predefined"] = []interface{}{
			map[string]interface{}{"name": namespaces["default"]},
		}
	}
	predefined := namespaces["predefined"].([]interface{})
	namespaces["predefined"] = append(predefined, ns)
}

func ResetFireFly(t *testing.T, client *resty.Client) {
	resp, err := client.R().
		SetBody(map[string]interface{}{}).
		Post("/reset")
	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode())
}

func RandomName(t *testing.T) string {
	b := make([]byte, 5)
	_, err := rand.Read(b)
	assert.NoError(t, err)
	return fmt.Sprintf("e2e_%x", b)
}

func AddPluginBroadcastName(data map[string]interface{}, pluginType, broadcastName string) {
	pluginsConfig := data["plugins"].(map[interface{}]interface{})
	plugins := pluginsConfig[pluginType].([]interface{})
	plugin := plugins[0].(map[interface{}]interface{})
	plugin["broadcastName"] = broadcastName
}

// ChangeDefaultNSPluginLocalName changes the plugin local name and updates the plugin list for the default namespace
func ChangeDefaultNSPluginLocalName(data map[string]interface{}, pluginType, newLocalName string) {
	pluginsConfig := data["plugins"].(map[interface{}]interface{})
	plugins := pluginsConfig[pluginType].([]interface{})
	plugin := plugins[0].(map[interface{}]interface{})
	oldName := plugin["name"]
	plugin["name"] = newLocalName

	namespaces := data["namespaces"].(map[interface{}]interface{})
	predefined := namespaces["predefined"].([]interface{})
	defaultNs := predefined[0].(map[interface{}]interface{})
	pluginList := defaultNs["plugins"].([]interface{})
	var newPluginList []interface{}
	for _, plugin := range pluginList {
		if plugin == oldName {
			newPluginList = append(newPluginList, newLocalName)
		} else {
			newPluginList = append(newPluginList, plugin)
		}
	}

	defaultNs["plugins"] = newPluginList
}
