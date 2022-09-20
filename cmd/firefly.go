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
	"fmt"

	"github.com/hyperledger/firefly/pkg/firefly"
	"github.com/spf13/cobra"
)

var cfgFile string

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "config file")
	rootCmd.AddCommand(showConfigCommand)
}

var rootCmd = &cobra.Command{
	Use:   "firefly",
	Short: "FireFly is a complete stack for enterprises to build and scale secure Web3 applications",
	Long: `Hyperledger FireFly is the first open source Supernode: a complete stack for
enterprises to build and scale secure Web3 applications. The FireFly API for digital
assets, data flows, and blockchain transactions makes it radically faster to build
production-ready apps on popular chains and protocols.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return firefly.Run(cfgFile)
	},
}

var showConfigCommand = &cobra.Command{
	Use:     "showconfig",
	Aliases: []string{"showconf"},
	Short:   "List out the configuration options",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(firefly.ShowConfig(cfgFile))

	},
}

// Execute is called by the main method of the package
func Execute() error {
	return rootCmd.Execute()
}
