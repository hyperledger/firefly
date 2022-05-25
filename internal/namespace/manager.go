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

package namespace

import (
	"context"
	"fmt"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
	"github.com/hyperledger/firefly/pkg/tokens"
)

type Manager interface {
	// Init initializes the manager
	Init(ctx context.Context, di database.Plugin) error
}

type namespaceManager struct {
	ctx           context.Context
	nsConfig      map[string]config.Section
	bcPlugins     map[string]blockchain.Plugin
	dbPlugins     map[string]database.Plugin
	dxPlugins     map[string]dataexchange.Plugin
	ssPlugins     map[string]sharedstorage.Plugin
	tokensPlugins map[string]tokens.Plugin
}

func NewNamespaceManager(ctx context.Context, bc map[string]blockchain.Plugin, db map[string]database.Plugin, dx map[string]dataexchange.Plugin, ss map[string]sharedstorage.Plugin, tokens map[string]tokens.Plugin) Manager {
	nm := &namespaceManager{
		ctx:           ctx,
		nsConfig:      buildNamespaceMap(ctx),
		bcPlugins:     bc,
		dbPlugins:     db,
		dxPlugins:     dx,
		ssPlugins:     ss,
		tokensPlugins: tokens,
	}
	return nm
}

func buildNamespaceMap(ctx context.Context) map[string]config.Section {
	conf := namespacePredefined
	namespaces := make(map[string]config.Section, conf.ArraySize())
	for i := 0; i < conf.ArraySize(); i++ {
		nsConfig := conf.ArrayEntry(i)
		name := nsConfig.GetString(coreconfig.NamespaceName)
		if name != "" {
			if _, ok := namespaces[name]; ok {
				log.L(ctx).Warnf("Duplicate predefined namespace (ignored): %s", name)
			}
			namespaces[name] = nsConfig
		}
	}
	return namespaces
}

func (nm *namespaceManager) Init(ctx context.Context, di database.Plugin) error {
	return nm.initNamespaces(ctx, di)
}

func (nm *namespaceManager) getPredefinedNamespaces(ctx context.Context) ([]*core.Namespace, error) {
	defaultNS := config.GetString(coreconfig.NamespacesDefault)
	namespaces := []*core.Namespace{
		{
			Name:        core.SystemNamespace,
			Type:        core.NamespaceTypeSystem,
			Description: i18n.Expand(ctx, coremsgs.CoreSystemNSDescription),
		},
	}
	i := 0
	foundDefault := false
	for name, nsObject := range nm.nsConfig {
		if err := nm.validateNamespaceConfig(ctx, name, i, nsObject); err != nil {
			return nil, err
		}
		i++
		foundDefault = foundDefault || name == defaultNS
		namespaces = append(namespaces, &core.Namespace{
			Type:        core.NamespaceTypeLocal,
			Name:        name,
			Description: nsObject.GetString("description"),
		})
	}
	if !foundDefault {
		return nil, i18n.NewError(ctx, coremsgs.MsgDefaultNamespaceNotFound, defaultNS)
	}
	return namespaces, nil
}

func (nm *namespaceManager) initNamespaces(ctx context.Context, di database.Plugin) error {
	predefined, err := nm.getPredefinedNamespaces(ctx)
	if err != nil {
		return err
	}
	for _, newNS := range predefined {
		ns, err := di.GetNamespace(ctx, newNS.Name)
		if err != nil {
			return err
		}
		var updated bool
		if ns == nil {
			updated = true
			newNS.ID = fftypes.NewUUID()
			newNS.Created = fftypes.Now()
		} else {
			// Only update if the description has changed, and the one in our DB is locally defined
			updated = ns.Description != newNS.Description && ns.Type == core.NamespaceTypeLocal
		}
		if updated {
			if err := di.UpsertNamespace(ctx, newNS, true); err != nil {
				return err
			}
		}
	}
	return nil
}

func (nm *namespaceManager) validateNamespaceConfig(ctx context.Context, name string, index int, conf config.Section) error {
	if err := core.ValidateFFNameField(ctx, name, fmt.Sprintf("namespaces.predefined[%d].name", index)); err != nil {
		return err
	}

	if name == core.SystemNamespace || conf.GetString(coreconfig.NamespaceRemoteName) == core.SystemNamespace {
		return i18n.NewError(ctx, coremsgs.MsgFFSystemReservedName, core.SystemNamespace)
	}

	mode := conf.GetString(coreconfig.NamespaceMode)
	plugins := conf.GetStringSlice(coreconfig.NamespacePlugins)

	// If no plugins are found when querying the config, assume older config file
	if len(plugins) == 0 {
		for plugin := range nm.bcPlugins {
			plugins = append(plugins, plugin)
		}

		for plugin := range nm.dxPlugins {
			plugins = append(plugins, plugin)
		}

		for plugin := range nm.ssPlugins {
			plugins = append(plugins, plugin)
		}

		for plugin := range nm.dbPlugins {
			plugins = append(plugins, plugin)
		}
	}

	switch mode {
	// Multiparty is the default mode when none is provided
	case "multiparty":
		if err := nm.validateMultiPartyConfig(ctx, name, plugins); err != nil {
			return err
		}
	case "gateway":
		if err := nm.validateGatewayConfig(ctx, name, plugins); err != nil {
			return err
		}
	default:
		return i18n.NewError(ctx, coremsgs.MsgInvalidNamespaceMode, name)
	}
	return nil
}

func (nm *namespaceManager) validateMultiPartyConfig(ctx context.Context, name string, plugins []string) error {
	var dbPlugin bool
	var ssPlugin bool
	var dxPlugin bool
	var bcPlugin bool

	for _, pluginName := range plugins {
		if _, ok := nm.bcPlugins[pluginName]; ok {
			if bcPlugin {
				return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "blockchain")
			}
			bcPlugin = true
			continue
		}
		if _, ok := nm.dxPlugins[pluginName]; ok {
			if dxPlugin {
				return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "dataexchange")
			}
			dxPlugin = true
			continue
		}
		if _, ok := nm.ssPlugins[pluginName]; ok {
			if ssPlugin {
				return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "sharedstorage")
			}
			ssPlugin = true
			continue
		}
		if _, ok := nm.dbPlugins[pluginName]; ok {
			if dbPlugin {
				return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "database")
			}
			dbPlugin = true
			continue
		}
		if _, ok := nm.tokensPlugins[pluginName]; ok {
			continue
		}

		return i18n.NewError(ctx, coremsgs.MsgNamespaceUnknownPlugin, name, pluginName)
	}

	if !dbPlugin || !ssPlugin || !dxPlugin || !bcPlugin {
		return i18n.NewError(ctx, coremsgs.MsgNamespaceMultipartyConfiguration, name)
	}

	return nil
}

func (nm *namespaceManager) validateGatewayConfig(ctx context.Context, name string, plugins []string) error {
	var dbPlugin bool
	var bcPlugin bool

	for _, pluginName := range plugins {
		if _, ok := nm.bcPlugins[pluginName]; ok {
			if bcPlugin {
				return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "blockchain")
			}
			bcPlugin = true
			continue
		}
		if _, ok := nm.dxPlugins[pluginName]; ok {
			return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayInvalidPlugins, name)
		}
		if _, ok := nm.ssPlugins[pluginName]; ok {
			return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayInvalidPlugins, name)
		}
		if _, ok := nm.dbPlugins[pluginName]; ok {
			if dbPlugin {
				return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "database")
			}
			dbPlugin = true
			continue
		}
		if _, ok := nm.tokensPlugins[pluginName]; ok {
			continue
		}

		return i18n.NewError(ctx, coremsgs.MsgNamespaceUnknownPlugin, name, pluginName)
	}

	if !dbPlugin {
		return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayNoDB, name)
	}

	return nil
}
