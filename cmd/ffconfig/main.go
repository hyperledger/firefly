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

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"

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

type ConfigItem struct {
	value  interface{}
	parent *ConfigItem
	name   string
}

type ConfigItemIterator struct {
	items []*ConfigItem
}

func (c *ConfigItem) hasChild(name string) (exists bool) {
	if v, ok := c.value.(map[interface{}]interface{}); ok {
		_, exists = v[name]
	}
	return exists
}

func (c *ConfigItem) deleteChild(name string) {
	if v, ok := c.value.(map[interface{}]interface{}); ok {
		delete(v, name)
	}
}

func (c *ConfigItem) setChild(name string, value interface{}) {
	if v, ok := c.value.(map[interface{}]interface{}); ok {
		v[name] = value
	}
}

func (c *ConfigItem) Get(name string) *ConfigItem {
	if v, ok := c.value.(map[interface{}]interface{}); ok {
		if child, ok := v[name]; ok {
			return &ConfigItem{value: child, parent: c, name: name}
		}
	}
	return &ConfigItem{value: nil, parent: c, name: name}
}

func (c *ConfigItemIterator) Get(name string) *ConfigItemIterator {
	items := make([]*ConfigItem, len(c.items))
	for i, item := range c.items {
		items[i] = item.Get(name)
	}
	return &ConfigItemIterator{items: items}
}

func (c *ConfigItem) Path() string {
	if c.parent == nil || c.parent.name == "" {
		return c.name
	}
	return c.parent.Path() + "." + c.name
}

func (c *ConfigItem) Exists() bool {
	return c.value != nil
}

func (c *ConfigItem) Length() int {
	if v, ok := c.value.([]interface{}); ok {
		return len(v)
	}
	return 0
}

func (c *ConfigItem) Each() *ConfigItemIterator {
	list, ok := c.value.([]interface{})
	if !ok {
		return &ConfigItemIterator{items: make([]*ConfigItem, 0)}
	}
	items := make([]*ConfigItem, len(list))
	for i, val := range list {
		items[i] = &ConfigItem{
			value:  val,
			parent: c.parent,
			name:   c.name,
		}
	}
	return &ConfigItemIterator{items: items}
}

func (c *ConfigItemIterator) Run(fn func(item *ConfigItem)) *ConfigItemIterator {
	for _, item := range c.items {
		fn(item)
	}
	return c
}

func (c *ConfigItem) Create() *ConfigItem {
	if !c.Exists() {
		if c.parent != nil {
			c.parent.Create()
		}
		c.value = make(map[interface{}]interface{})
		c.parent.setChild(c.name, c.value)
	}
	return c
}

func (c *ConfigItem) Set(value interface{}) *ConfigItem {
	fmt.Printf("Create: %s: %s\n", c.Path(), value)
	c.value = value
	c.parent.Create()
	c.parent.setChild(c.name, c.value)
	return c
}

func (c *ConfigItemIterator) Set(value interface{}) *ConfigItemIterator {
	for _, item := range c.items {
		item.Set(value)
	}
	return c
}

func (c *ConfigItem) SetIfEmpty(value interface{}) *ConfigItem {
	if !c.Exists() {
		c.Set(value)
	}
	return c
}

func (c *ConfigItemIterator) SetIfEmpty(value interface{}) *ConfigItemIterator {
	for _, item := range c.items {
		item.SetIfEmpty(value)
	}
	return c
}

func (c *ConfigItem) Delete() *ConfigItem {
	if c.Exists() {
		fmt.Printf("Delete: %s\n", c.Path())
		c.parent.deleteChild(c.name)
	}
	return c
}

func (c *ConfigItemIterator) Delete() *ConfigItemIterator {
	for _, item := range c.items {
		item.Delete()
	}
	return c
}

func (c *ConfigItem) RenameTo(name string) *ConfigItem {
	if c.Exists() {
		if c.parent.hasChild(name) {
			// Don't overwrite if the new key already exists
			c.Delete()
		} else {
			fmt.Printf("Rename: %s -> .%s\n", c.Path(), name)
			c.parent.deleteChild(c.name)
			c.parent.setChild(name, c.value)
			c.name = name
		}
	}
	return c
}

func (c *ConfigItemIterator) RenameTo(name string) *ConfigItemIterator {
	for _, item := range c.items {
		item.RenameTo(name)
	}
	return c
}

func (c *ConfigItem) ReplaceValue(old interface{}, new interface{}) *ConfigItem {
	if c.value == old {
		fmt.Printf("Change: %s: %s -> %s\n", c.Path(), old, new)
		c.value = new
		c.parent.setChild(c.name, new)
	}
	return c
}

func (c *ConfigItemIterator) ReplaceValue(old interface{}, new interface{}) *ConfigItemIterator {
	for _, item := range c.items {
		item.ReplaceValue(old, new)
	}
	return c
}

func run() error {
	yfile, err := ioutil.ReadFile("firefly_core_0.yml")
	if err != nil {
		return err
	}
	data := make(map[interface{}]interface{})
	err = yaml.Unmarshal(yfile, &data)
	if err != nil {
		return err
	}
	versions := make([]string, 0, len(migrations))
	for k := range migrations {
		versions = append(versions, k)
	}
	sort.Strings(versions)
	root := &ConfigItem{value: data}
	for _, version := range versions {
		fmt.Printf("Version %s\n", version)
		migrations[version](root)
		fmt.Println()
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Print(string(out))
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
