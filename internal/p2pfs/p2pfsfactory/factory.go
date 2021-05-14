// Copyright © 2021 Kaleido, Inc.
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

package p2pfsfactory

import (
	"context"

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/p2pfs"
	"github.com/kaleido-io/firefly/internal/p2pfs/ipfs"
)

var plugins = []p2pfs.Plugin{
	&ipfs.IPFS{},
}

var pluginsByName = make(map[string]p2pfs.Plugin)

func init() {
	for _, p := range plugins {
		pluginsByName[p.Name()] = p
	}
}

func InitConfigPrefix(prefix config.ConfigPrefix) {
	for _, plugin := range plugins {
		plugin.InitConfigPrefix(prefix.SubPrefix(plugin.Name()))
	}
}

func GetPlugin(ctx context.Context, pluginType string) (p2pfs.Plugin, error) {
	plugin, ok := pluginsByName[pluginType]
	if !ok {
		return nil, i18n.NewError(ctx, i18n.MsgUnknownP2PFilesystemPlugin, pluginType)
	}
	return plugin, nil
}
