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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var configPath string = "../test/data/config/firefly.core.yaml"

func TestConfigMigrateRootCmdErrorNoArgs(t *testing.T) {
	rootCmd.SetArgs([]string{})
	defer rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.Regexp(t, "a command is required", err)
}

func TestConfigMigrateCmdMissingConfig(t *testing.T) {
	rootCmd.SetArgs([]string{"migrate"})
	defer rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.Regexp(t, "no such file or directory", err)
}

func TestConfigMigrateCmd(t *testing.T) {
	rootCmd.SetArgs([]string{"migrate", "-f", configPath})
	defer rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	assert.NoError(t, err)
}

func TestConfigMigrateCmdWriteOutput(t *testing.T) {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "out")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	rootCmd.SetArgs([]string{"migrate", "-f", configPath, "-o", fmt.Sprintf(tmpDir, "out.config")})
	defer rootCmd.SetArgs([]string{})
	err = rootCmd.Execute()
	assert.NoError(t, err)
}

func TestConfigMigrateCmdBadVersion(t *testing.T) {
	rootCmd.SetArgs([]string{"migrate", "-f", configPath, "--from", "badversion"})
	defer rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.Regexp(t, "bad 'from' version", err)
}
