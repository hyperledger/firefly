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

package eifactory

import (
	"context"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/events/system"
	"github.com/hyperledger/firefly/internal/events/webhooks"
	"github.com/hyperledger/firefly/internal/events/websockets"
	"github.com/hyperledger/firefly/pkg/events"
)

var websocketsFactory = websockets.Factory{}
var webhooksFactory = webhooks.Factory{}
var systemEventsFactory = system.Factory{}
var pluginFactoriesByType = map[string]events.Factory{}

func init() {
	RegisterFactory(&websocketsFactory)
	RegisterFactory(&webhooksFactory)
	RegisterFactory(&systemEventsFactory)
}

func RegisterFactory(factory events.Factory) {
	pluginFactoriesByType[factory.Type()] = factory

}

func InitConfig(config config.Section) {
	for pluginType, factory := range pluginFactoriesByType {
		factory.InitConfig(config.SubSection(pluginType))
	}
}

func NewInstance(ctx context.Context, pluginType string) (events.Plugin, error) {
	log.L(ctx).Infof("eifactory: creating instance of '%s'", pluginType)

	factory, ok := pluginFactoriesByType[pluginType]
	if !ok {
		log.L(ctx).Warnf("eifactory: plugin not found '%s'", pluginType)
		return nil, i18n.NewError(ctx, coremsgs.MsgUnknownEventTransportPlugin, pluginType)
	}
	return factory.NewInstance(), nil
}
