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

package migrate

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/blang/semver/v4"
	"gopkg.in/yaml.v2"
)

var migrations = map[string]func(root *ConfigItem){
	"1.0.0": func(root *ConfigItem) {
		root.Get("org").Get("identity").RenameTo("key")
		root.Get("publicstorage").RenameTo("sharedstorage")
		root.Get("tokens").Each().Get("connector").RenameTo("plugin").ReplaceValue("https", "fftokens")
		root.Get("dataexchange").Get("type").ReplaceValue("https", "ffdx")
		root.Get("dataexchange").Get("https").RenameTo("ffdx")
	},

	"1.0.3": func(root *ConfigItem) {
		root.Get("dataexchange").Get("type").SetIfEmpty("ffdx")
	},

	"1.1.0": func(root *ConfigItem) {
		root.Get("admin").RenameTo("spi")

		movePlugin := func(name string) {
			old := root.Get(name)
			new := root.Get("plugins").Get(name)
			if new.Length() == 0 {
				new.Set([]interface{}{old.value})
				new.Each().Get("name").Set(name + "0")
			}
			old.Delete()
		}

		movePlugin("blockchain")
		movePlugin("database")
		movePlugin("dataexchange")
		movePlugin("sharedstorage")

		oldTokens := root.Get("tokens")
		newTokens := root.Get("plugins").Get("tokens")
		if newTokens.Length() == 0 && oldTokens.Length() > 0 {
			items := make([]interface{}, 0, oldTokens.Length())
			oldTokens.Each().Run(func(item *ConfigItem) {
				items = append(items, map[interface{}]interface{}{
					"name":     item.Get("name").Delete().value,
					"type":     item.Get("plugin").Delete().value,
					"fftokens": item.value,
				})
			})
			newTokens.Set(items)
		}
		oldTokens.Delete()

		defaultNS := root.Get("namespaces").Get("default").SetIfEmpty("default").value
		namespaces := root.Get("namespaces").Get("predefined").Create()
		if namespaces.Length() == 0 {
			namespaces.Set([]interface{}{
				map[interface{}]interface{}{
					"name": defaultNS,
				},
			})
		}

		rootOrg := root.Get("org")
		rootNode := root.Get("node")
		if rootOrg.Get("name").Exists() || rootOrg.Get("key").Exists() {
			namespaces.Each().Run(func(item *ConfigItem) {
				if item.Get("multiparty").Get("enabled").value != false {
					item.Get("multiparty").Get("enabled").Set(true)
					item.Get("multiparty").Get("org").SetIfEmpty(rootOrg.value)
					item.Get("multiparty").Get("node").SetIfEmpty(rootNode.value)
				}
			})
		}
		rootOrg.Delete()
		rootNode.Delete()

		root.Get("plugins").Get("blockchain").Each().Get("ethereum").Get("ethconnect").Run(func(ethconnect *ConfigItem) {
			contract := map[interface{}]interface{}{
				"location": map[interface{}]interface{}{
					"address": ethconnect.Get("instance").Delete().value,
				},
			}
			fromBlock := ethconnect.Get("fromBlock").Delete()
			if fromBlock.Exists() {
				contract["firstEvent"] = fromBlock.value
			}
			namespaces.Each().Run(func(namespace *ConfigItem) {
				if namespace.Get("multiparty").Get("enabled").value == true {
					namespace.Get("multiparty").Get("contract").SetIfEmpty(contract)
				}
			})
		})

		root.Get("plugins").Get("blockchain").Each().Get("fabric").Get("fabconnect").Run(func(fabconnect *ConfigItem) {
			contract := map[interface{}]interface{}{
				"location": map[interface{}]interface{}{
					"chaincode": fabconnect.Get("chaincode").Delete().value,
					"channel":   fabconnect.Get("channel").value,
				},
			}
			namespaces.Each().Run(func(namespace *ConfigItem) {
				if namespace.Get("multiparty").Get("enabled").value == true {
					namespace.Get("multiparty").Get("contract").SetIfEmpty(contract)
				}
			})
		})
	},
}

func getVersions() []semver.Version {
	versions := make([]semver.Version, 0, len(migrations))
	for k := range migrations {
		versions = append(versions, semver.MustParse(k))
	}
	semver.Sort(versions)
	return versions
}

func migrateVersion(root *ConfigItem, version string) {
	fmt.Fprintf(os.Stderr, "Version %s\n", version)
	migrations[version](root)
	fmt.Fprintln(os.Stderr)
}

func Run(cfgFile string) error {
	yfile, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return err
	}
	data := make(map[interface{}]interface{})
	err = yaml.Unmarshal(yfile, &data)
	if err != nil {
		return err
	}
	root := &ConfigItem{value: data, writer: os.Stderr}
	for _, version := range getVersions() {
		migrateVersion(root, version.String())
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Print(string(out))
	return nil
}
