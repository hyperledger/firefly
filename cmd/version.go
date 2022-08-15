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
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/spf13/cobra"
)

var shortened, output = false, "yaml"

var BuildDate string
var BuildCommit string
var BuildVersionOverride string

func setBuildInfo(info *core.Version, buildInfo *debug.BuildInfo, ok bool) {
	if ok {
		info.Version = buildInfo.Main.Version
	}
}

func getVersion() core.Version {
	info := core.Version{
		Date:       BuildDate,
		Commit:     BuildCommit,
		Version:    BuildVersionOverride,
		License:    "Apache-2.0",
		APIVersion: apiVersion,
	}

	// Where you are using go install, we will get good version information usefully from Go
	// When we're in go-releaser in a Github action, we will have the version passed in explicitly
	if info.Version == "" {
		buildInfo, ok := debug.ReadBuildInfo()
		setBuildInfo(&info, buildInfo, ok)
	}

	if info.Version != "(devel)" && !strings.HasPrefix(info.Version, fmt.Sprintf("v%s", apiVersion)) {
		panic(i18n.NewError(context.Background(), coremsgs.MsgInvalidBuildVersion, apiVersion, info.Version))
	}

	return info
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version info",
	Long:  "Prints the version info of the Core binary",
	RunE: func(cmd *cobra.Command, args []string) error {

		info := getVersion()
		if shortened {
			fmt.Println(info.Version)
		} else {

			var (
				bytes []byte
				err   error
			)

			switch output {
			case "json":
				bytes, err = json.MarshalIndent(info, "", "  ")
			case "yaml":
				bytes, err = yaml.Marshal(info)
			default:
				err = i18n.NewError(context.Background(), coremsgs.MsgInvalidOutputOption, output)
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
