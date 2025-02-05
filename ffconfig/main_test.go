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
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

var configPath string = "../test/data/config/firefly.core.yaml"

func TestMainFail(t *testing.T) {
	// Run the crashing code when FLAG is set
	if os.Getenv("FLAG") == "1" {
		main()
		return
	}
	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestMainFail")
	cmd.Env = append(os.Environ(), "FLAG=1")
	err := cmd.Run()

	// Cast the error as *exec.ExitError and compare the result
	e, ok := err.(*exec.ExitError)
	expectedErrorString := "exit status 1"
	assert.Equal(t, true, ok)
	assert.Equal(t, expectedErrorString, e.Error())
}

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

func TestMain(t *testing.T) {
	// Run the exiting code when FLAG is set
	if os.Getenv("FLAG") == "0" {
		rootCmd.SetArgs([]string{"migrate", "-f", configPath})
		main()
		return
	}

	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestMain")
	cmd.Env = append(os.Environ(), "FLAG=0")
	err := cmd.Run()

	// Cast the error as *exec.ExitError and compare the result
	_, ok := err.(*exec.ExitError)
	assert.Equal(t, false, ok)
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
