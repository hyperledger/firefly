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

package cmd

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionCmdDefault(t *testing.T) {
	rootCmd.SetArgs([]string{"version"})
	defer rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	assert.NoError(t, err)
}

func TestVersionCmdYAML(t *testing.T) {
	rootCmd.SetArgs([]string{"version", "-o", "yaml"})
	defer rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	assert.NoError(t, err)
}

func TestVersionCmdJSON(t *testing.T) {
	rootCmd.SetArgs([]string{"version", "-o", "json"})
	defer rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	assert.NoError(t, err)
}

func TestVersionCmdInvalidType(t *testing.T) {
	rootCmd.SetArgs([]string{"version", "-o", "wrong"})
	defer rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	assert.Regexp(t, "FF10385", err)
}

func TestVersionCmdShorthand(t *testing.T) {
	rootCmd.SetArgs([]string{"version", "-s"})
	defer rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	assert.NoError(t, err)
}

func TestSetBuildInfoWithBI(t *testing.T) {
	info := &Info{}
	setBuildInfo(info, &debug.BuildInfo{Main: debug.Module{Version: "12345"}}, true)
	assert.Equal(t, "12345", info.Version)
}
