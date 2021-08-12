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

package tkfactory

import (
	"context"
	"strconv"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/tokens/https"
	"github.com/hyperledger-labs/firefly/pkg/tokens"
)

var plugins = []tokens.Plugin{
	&https.HTTPS{},
}

var pluginsByName = make(map[string]tokens.Plugin)

func init() {
	for _, p := range plugins {
		pluginsByName[p.ConnectorName()] = p
	}
}

func InitPrefix(prefix config.Prefix) {
	// TODO: add better config support for arrays
	for i := 0; i < 5; i++ {
		p := prefix.SubPrefix(strconv.Itoa(i))
		p.AddKnownKey("connector")
		p.AddKnownKey("name")
		for _, plugin := range plugins {
			// Accept a superset of configs allowed by all plugins
			plugin.InitPrefix(p)
		}
	}
}

func GetPlugin(ctx context.Context, pluginType string) (tokens.Plugin, error) {
	plugin, ok := pluginsByName[pluginType]
	if !ok {
		return nil, i18n.NewError(ctx, i18n.MsgUnknownTokensPlugin, pluginType)
	}
	return plugin, nil
}
