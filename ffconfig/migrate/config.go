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
	"io"
)

type ConfigItem struct {
	value  interface{}
	parent *ConfigItem
	name   string
	writer io.Writer
}

type ConfigItemIterator struct {
	items []*ConfigItem
}

func NewConfigItem(value interface{}, writer io.Writer) *ConfigItem {
	return &ConfigItem{value: value, writer: writer}
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
			return &ConfigItem{value: child, parent: c, name: name, writer: c.writer}
		}
	}
	return &ConfigItem{value: nil, parent: c, name: name, writer: c.writer}
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
			writer: c.writer,
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
	fmt.Fprintf(c.writer, "Create: %s: %s\n", c.Path(), value)
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

func (c *ConfigItem) Delete() *ConfigItem {
	if c.Exists() {
		fmt.Fprintf(c.writer, "Delete: %s\n", c.Path())
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
			fmt.Fprintf(c.writer, "Rename: %s -> .%s\n", c.Path(), name)
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
		fmt.Fprintf(c.writer, "Change: %s: %s -> %s\n", c.Path(), old, new)
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
