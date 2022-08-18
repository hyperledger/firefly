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
	"os"
	"strings"

	"github.com/blang/semver/v4"
	"gopkg.in/yaml.v2"
)

var migrations = map[string]func(root *ConfigItem){
	"1.0.0": func(root *ConfigItem) {
		root.Get("org").Get("identity").RenameTo("key")
		root.Get("tokens").Each().Get("connector").RenameTo("plugin").ReplaceValue("https", "fftokens")
		root.Get("dataexchange").Get("type").ReplaceValue("https", "ffdx")
		root.Get("dataexchange").Get("https").RenameTo("ffdx")
	},

	"1.0.3": func(root *ConfigItem) {
		root.Get("dataexchange").Get("type").SetIfEmpty("ffdx")
	},

	"1.0.4": func(root *ConfigItem) {
		root.Get("publicstorage").RenameTo("sharedstorage")
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

func Run(cfg []byte, fromVersion, toVersion string) (result []byte, err error) {
	var from, to semver.Version
	if fromVersion != "" {
		if from, err = semver.Parse(strings.TrimPrefix(fromVersion, "v")); err != nil {
			return nil, fmt.Errorf("bad 'from' version: %s", err)
		}
	}
	if toVersion != "" {
		if to, err = semver.Parse(strings.TrimPrefix(toVersion, "v")); err != nil {
			return nil, fmt.Errorf("bad 'to' version: %s", err)
		}
	}

	data := make(map[interface{}]interface{})
	err = yaml.Unmarshal(cfg, &data)
	if err != nil {
		return nil, err
	}

	root := NewConfigItem(data, os.Stderr)
	for _, version := range getVersions() {
		if fromVersion != "" && version.LT(from) {
			continue
		}
		if toVersion != "" && version.GT(to) {
			break
		}
		migrateVersion(root, version.String())
	}

	return yaml.Marshal(data)
}
