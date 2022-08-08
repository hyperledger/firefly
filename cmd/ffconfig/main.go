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

	"github.com/hyperledger/firefly/cmd/ffconfig/migrate"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ffconfig",
	Short: "Tool for managing and migrating config files for Hyperledger FireFly",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("a command is required")
	},
}

var migrateCommand = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate a config file to the current version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return migrate.Run(cfgFile)
	},
}

var cfgFile string

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "firefly_core.yml", "config file")
	rootCmd.AddCommand(migrateCommand)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
