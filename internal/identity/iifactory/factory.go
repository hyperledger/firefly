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

package iifactory

import (
	"context"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/identity/tbd"
	"github.com/hyperledger/firefly/pkg/identity"
)

var pluginsByName = map[string]func() identity.Plugin{
	// Plugin interface is TBD at this point. Plugin with "onchain" naming, and TBD implementation provided to avoid config migration impact
	(*tbd.TBD)(nil).Name(): func() identity.Plugin { return &tbd.TBD{} },
}

func InitPrefix(prefix config.Prefix) {
	for name, plugin := range pluginsByName {
		plugin().InitPrefix(prefix.SubPrefix(name))
	}
}

func GetPlugin(ctx context.Context, pluginType string) (identity.Plugin, error) {
	plugin, ok := pluginsByName[pluginType]
	if !ok {
		return nil, i18n.NewError(ctx, i18n.MsgUnknownIdentityPlugin, pluginType)
	}
	return plugin(), nil
}
