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

package psfactory

import (
	"context"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/publicstorage/ipfs"
	"github.com/hyperledger/firefly/pkg/publicstorage"
)

var plugins = []publicstorage.Plugin{
	&ipfs.IPFS{},
}

var pluginsByName = make(map[string]publicstorage.Plugin)

func init() {
	for _, p := range plugins {
		pluginsByName[p.Name()] = p
	}
}

func InitPrefix(prefix config.Prefix) {
	for _, plugin := range plugins {
		plugin.InitPrefix(prefix.SubPrefix(plugin.Name()))
	}
}

func GetPlugin(ctx context.Context, pluginType string) (publicstorage.Plugin, error) {
	plugin, ok := pluginsByName[pluginType]
	if !ok {
		return nil, i18n.NewError(ctx, i18n.MsgUnknownPublicStoragePlugin, pluginType)
	}
	return plugin, nil
}
