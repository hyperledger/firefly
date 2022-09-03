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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigrate1_0_0(t *testing.T) {
	config := []byte(`database:
  type: sqlite3
dataexchange: {}
publicstorage:
  type: ipfs
`)
	result, err := Run(config, "", "1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, `database:
  type: sqlite3
dataexchange: {}
publicstorage:
  type: ipfs
`, string(result))
}

func TestMigratev1_0_0(t *testing.T) {
	config := []byte(`database:
  type: sqlite3
dataexchange: {}
`)
	result, err := Run(config, "", "v1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, `database:
  type: sqlite3
dataexchange: {}
`, string(result))
}

func TestMigrate1_0_3(t *testing.T) {
	config := []byte(`database:
  type: sqlite3
dataexchange: {}
`)
	result, err := Run(config, "", "1.0.3")
	assert.NoError(t, err)
	assert.Equal(t, `database:
  type: sqlite3
dataexchange:
  type: ffdx
`, string(result))
}

func TestMigrate1_0_3Only(t *testing.T) {
	config := []byte(`database:
  type: sqlite3
dataexchange: {}
publicstorage:
  type: ipfs
`)
	result, err := Run(config, "1.0.2", "1.0.3")
	assert.NoError(t, err)
	assert.Equal(t, `database:
  type: sqlite3
dataexchange:
  type: ffdx
publicstorage:
  type: ipfs
`, string(result))
}

func TestMigrate1_0_4(t *testing.T) {
	config := []byte(`database:
  type: sqlite3
dataexchange: {}
publicstorage:
  type: ipfs
`)
	result, err := Run(config, "", "1.0.4")
	assert.NoError(t, err)
	assert.Equal(t, `database:
  type: sqlite3
dataexchange:
  type: ffdx
sharedstorage:
  type: ipfs
`, string(result))
}

func TestMigrate1_1_0(t *testing.T) {
	config := []byte(`database:
  type: sqlite3
dataexchange: {}
publicstorage:
  type: ipfs
tokens:
- name: erc1155
  connector: https
org:
  name: org0
  key: testkey
node:
  name: node0
`)
	result, err := Run(config, "", "")
	assert.NoError(t, err)
	assert.Equal(t, `namespaces:
  default: default
  predefined:
  - multiparty:
      enabled: true
      node:
        name: node0
      org:
        key: testkey
        name: org0
    name: default
plugins:
  database:
  - name: database0
    type: sqlite3
  dataexchange:
  - name: dataexchange0
    type: ffdx
  sharedstorage:
  - name: sharedstorage0
    type: ipfs
  tokens:
  - fftokens: {}
    name: erc1155
    type: fftokens
`, string(result))
}

func TestMigrate1_1_0Ethconnect(t *testing.T) {
	config := []byte(`blockchain:
  type: ethereum
  ethereum:
    ethconnect:
      url: http://ethconnect_0:8080
      instance: 0x2a81bebb99b1541919970b609dec6f9a2b0a2550
      fromBlock: oldest
org:
  name: org0
  key: testkey
`)
	result, err := Run(config, "", "")
	assert.NoError(t, err)
	assert.Equal(t, `namespaces:
  default: default
  predefined:
  - multiparty:
      contract:
      - firstEvent: oldest
        location:
          address: 0x2a81bebb99b1541919970b609dec6f9a2b0a2550
      enabled: true
      org:
        key: testkey
        name: org0
    name: default
plugins:
  blockchain:
  - ethereum:
      ethconnect:
        url: http://ethconnect_0:8080
    name: blockchain0
    type: ethereum
  dataexchange:
  - name: dataexchange0
    type: ffdx
`, string(result))
}

func TestMigrate1_1_0Fabconnect(t *testing.T) {
	config := []byte(`blockchain:
  type: fabric
  fabric:
    fabconnect:
      url: http://fabconnect_0:8080
      channel: firefly
      chaincode: firefly
org:
  name: org0
  key: testkey
`)
	result, err := Run(config, "", "")
	assert.NoError(t, err)
	assert.Equal(t, `namespaces:
  default: default
  predefined:
  - multiparty:
      contract:
      - location:
          chaincode: firefly
          channel: firefly
      enabled: true
      org:
        key: testkey
        name: org0
    name: default
plugins:
  blockchain:
  - fabric:
      fabconnect:
        channel: firefly
        url: http://fabconnect_0:8080
    name: blockchain0
    type: fabric
  dataexchange:
  - name: dataexchange0
    type: ffdx
`, string(result))
}

func TestMigrateBadVersions(t *testing.T) {
	_, err := Run([]byte{}, "BAD!", "")
	assert.Regexp(t, "bad 'from' version", err)
	_, err = Run([]byte{}, "", "BAD!")
	assert.Regexp(t, "bad 'to' version", err)
}

func TestMigrateBadYAML(t *testing.T) {
	_, err := Run([]byte{'\t'}, "", "")
	assert.Regexp(t, "yaml: found character", err)
}
