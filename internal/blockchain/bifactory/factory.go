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

package bifactory

import (
	"context"

	"github.com/hyperledger/firefly/internal/blockchain/ethereum"
	"github.com/hyperledger/firefly/internal/blockchain/fabric"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/blockchain"
)

var pluginsByName = map[string]func() blockchain.Plugin{
	(*ethereum.Ethereum)(nil).Name(): func() blockchain.Plugin { return &ethereum.Ethereum{} },
	(*fabric.Fabric)(nil).Name():     func() blockchain.Plugin { return &fabric.Fabric{} },
}

func InitPrefix(prefix config.Prefix) {
	for name, plugin := range pluginsByName {
		plugin().InitPrefix(prefix.SubPrefix(name))
	}
}

func GetPlugin(ctx context.Context, pluginType string) (blockchain.Plugin, error) {
	plugin, ok := pluginsByName[pluginType]
	if !ok {
		return nil, i18n.NewError(ctx, i18n.MsgUnknownBlockchainPlugin, pluginType)
	}
	return plugin(), nil
}
