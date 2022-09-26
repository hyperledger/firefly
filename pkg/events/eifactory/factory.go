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
	"fmt"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/events/system"
	"github.com/hyperledger/firefly/internal/events/webhooks"
	"github.com/hyperledger/firefly/internal/events/websockets"
	"github.com/hyperledger/firefly/pkg/events"
)

var websocketsPlugin = websockets.WebSockets{}
var webhooksPlugin = webhooks.WebHooks{}
var systemEventsPlugin = system.Events{}
var pluginsByName = map[string]func() events.Plugin{
	websockets.Name(): func() events.Plugin { return &websocketsPlugin },
	webhooks.Name():   func() events.Plugin { return &webhooksPlugin },
	system.Name():     func() events.Plugin { return &systemEventsPlugin },
}

// Allows other code to register additional implementations of Event plugins
// that can also be initialized with this factory
func RegisterPlugins(plugins map[string]func() events.Plugin) {
	fmt.Println("registering plugins")

	for k, plugin := range plugins {
		fmt.Printf("registering '%s'\n", k)
		pluginsByName[k] = plugin
	}
}

func GetPlugin(ctx context.Context, pluginName string) (events.Plugin, error) {
	plugin, ok := pluginsByName[pluginName]
	if !ok {
		log.L(ctx).Infof("plugin not found '%s'", pluginName)
		return nil, i18n.NewError(ctx, coremsgs.MsgUnknownEventTransportPlugin, pluginName)
	}
	return plugin(), nil
}
