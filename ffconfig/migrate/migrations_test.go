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
