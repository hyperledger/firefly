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

package ssfactory

import (
	"context"
	"fmt"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/sharedstorage/ipfs"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
)

var pluginFactoriesByType = map[string]sharedstorage.Factory{}

func init() {
	RegisterFactory(&ipfs.Factory{})
}

func RegisterFactory(factory sharedstorage.Factory) {
	fmt.Printf("ssfactory: registering factory  %s\n", factory.Type())
	pluginFactoriesByType[factory.Type()] = factory
}

func InitConfig(config config.ArraySection) {
	config.AddKnownKey(coreconfig.PluginConfigType)
	config.AddKnownKey(coreconfig.PluginConfigName)
	for name, factory := range pluginFactoriesByType {
		factory.InitConfig(config.SubSection(name))
	}
}

func InitConfigDeprecated(config config.Section) {
	config.AddKnownKey(coreconfig.PluginConfigType)
	for name, factory := range pluginFactoriesByType {
		factory.InitConfig(config.SubSection(name))
	}
}

func NewInstance(ctx context.Context, pluginType string) (sharedstorage.Plugin, error) {
	log.L(ctx).Infof("ssfactory: creating instance of '%s'", pluginType)

	factory, ok := pluginFactoriesByType[pluginType]
	if !ok {
		log.L(ctx).Warnf("eifactory: plugin not found '%s'", pluginType)
		return nil, i18n.NewError(ctx, coremsgs.MsgUnknownSharedStoragePlugin, pluginType)
	}
	return factory.NewInstance(), nil
}
