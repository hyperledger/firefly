// Copyright © 2021 Kaleido, Inc.
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
	"encoding/json"
	"fmt"

	"github.com/hyperledger/firefly/internal/version"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var shortened, output = false, "yaml"

type Info struct {
	Version string `json:"Version,omitempty" yaml:"Version,omitempty"`
	Commit  string `json:"Commit,omitempty" yaml:"Commit,omitempty"`
	Date    string `json:"Date,omitempty" yaml:"Date,omitempty"`
	License string `json:"License,omitempty" yaml:"License,omitempty"`
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version info",
	Long: "Prints the version info of the Core binary",
	RunE: func(cmd *cobra.Command, args []string) error {
		if shortened {
			fmt.Println(version.Version)
		} else {
			info := &Info{
				Version: version.Version,
				Commit: version.Commit,
				Date: version.Date,
				License: version.License,
			}

			var (
				bytes []byte
				err error
			)

			switch output {
				case "json":
					bytes, err = json.MarshalIndent(info, "", "  ")
				case "yaml":
					bytes, err = yaml.Marshal(info)
				default:
					return fmt.Errorf("invalid output '%s'", output)
			}

			if err != nil {
				return err
			}

			fmt.Println(string(bytes))
		}

		return nil
	},
}

func init() {
	versionCmd.Flags().BoolVarP(&shortened, "short", "s", false, "Prints only the version number")
	versionCmd.Flags().StringVarP(&output, "output", "o", "json", "output format (\"yaml\"|\"json\")")
	rootCmd.AddCommand(versionCmd)
}
