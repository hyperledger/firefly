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
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/adminevents"
	"github.com/hyperledger/firefly/internal/blockchain/bifactory"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/database/difactory"
	"github.com/hyperledger/firefly/internal/dataexchange/dxfactory"
	"github.com/hyperledger/firefly/internal/identity/iifactory"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/orchestrator"
	"github.com/hyperledger/firefly/internal/sharedstorage/ssfactory"
	"github.com/hyperledger/firefly/internal/tokens/tifactory"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/tokens"
)

var (
	blockchainConfig    = config.RootArray("plugins.blockchain")
	tokensConfig        = config.RootArray("plugins.tokens")
	databaseConfig      = config.RootArray("plugins.database")
	sharedstorageConfig = config.RootArray("plugins.sharedstorage")
	dataexchangeConfig  = config.RootArray("plugins.dataexchange")
	identityConfig      = config.RootArray("plugins.identity")

	// Deprecated configs
	deprecatedTokensConfig        = config.RootArray("tokens")
	deprecatedBlockchainConfig    = config.RootSection("blockchain")
	deprecatedDatabaseConfig      = config.RootSection("database")
	deprecatedSharedStorageConfig = config.RootSection("sharedstorage")
	deprecatedDataexchangeConfig  = config.RootSection("dataexchange")
)

type Manager interface {
	Init(ctx context.Context, cancelCtx context.CancelFunc) error
	Start() error
	WaitStop()

	Orchestrator(ns string) orchestrator.Orchestrator
	GetNamespaces(ctx context.Context, filter database.AndFilter) ([]*core.Namespace, *database.FilterResult, error)
}

type namespace struct {
	orchestrator  orchestrator.Orchestrator
	multiparty    bool
	database      orchestrator.DatabasePlugin
	blockchain    orchestrator.BlockchainPlugin
	dataexchange  orchestrator.DataexchangePlugin
	sharedstorage orchestrator.SharedStoragePlugin
	identity      orchestrator.IdentityPlugin
	tokens        map[string]orchestrator.TokensPlugin
}

type namespaceManager struct {
	ctx            context.Context
	cancelCtx      context.CancelFunc
	pluginNames    map[string]bool
	metrics        metrics.Manager
	blockchains    map[string]orchestrator.BlockchainPlugin
	identities     map[string]orchestrator.IdentityPlugin
	databases      map[string]orchestrator.DatabasePlugin
	sharedstorages map[string]orchestrator.SharedStoragePlugin
	dataexchanges  map[string]orchestrator.DataexchangePlugin
	tokens         map[string]orchestrator.TokensPlugin
	adminEvents    adminevents.Manager
	namespaces     map[string]namespace
}

func NewNamespaceManager(withDefaults bool) Manager {
	nm := &namespaceManager{}

	// Initialize the config on all the factories
	bifactory.InitConfigDeprecated(deprecatedBlockchainConfig)
	bifactory.InitConfig(blockchainConfig)
	difactory.InitConfigDeprecated(deprecatedDatabaseConfig)
	difactory.InitConfig(databaseConfig)
	ssfactory.InitConfigDeprecated(deprecatedSharedStorageConfig)
	ssfactory.InitConfig(sharedstorageConfig)
	dxfactory.InitConfig(dataexchangeConfig)
	dxfactory.InitConfigDeprecated(deprecatedDataexchangeConfig)
	iifactory.InitConfig(identityConfig)
	tifactory.InitConfigDeprecated(deprecatedTokensConfig)
	tifactory.InitConfig(tokensConfig)

	return nm
}

func (nm *namespaceManager) Init(ctx context.Context, cancelCtx context.CancelFunc) (err error) {
	nm.ctx = ctx
	nm.cancelCtx = cancelCtx

	err = nm.getPlugins(ctx)
	if err != nil {
		return err
	}

	nsConfig := buildNamespaceMap(ctx)
	if err = nm.initNamespaces(ctx, nsConfig); err != nil {
		return err
	}

	// Start an orchestrator per namespace
	for name, ns := range nm.namespaces {
		or := orchestrator.NewOrchestrator(name, ns.blockchain, ns.database, ns.sharedstorage, ns.dataexchange, ns.tokens, ns.identity, nm.metrics)
		if err = or.Init(ctx, cancelCtx); err != nil {
			return err
		}
		ns.orchestrator = or
	}
	return nil
}

func (nm *namespaceManager) Start() error {
	for _, ns := range nm.namespaces {
		if err := ns.orchestrator.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (nm *namespaceManager) WaitStop() {
	for _, ns := range nm.namespaces {
		ns.orchestrator.WaitStop()
	}
}

func (nm *namespaceManager) getPlugins(ctx context.Context) (err error) {
	//TODO: combine plugin/config section into a struct? to help w/ plugin initialization in orchestrator

	nm.pluginNames = make(map[string]bool)
	if nm.metrics == nil {
		nm.metrics = metrics.NewMetricsManager(ctx)
	}

	if nm.databases == nil {
		nm.databases, err = nm.getDatabasePlugins(ctx)
		if err != nil {
			return err
		}
	}

	// Not really a plugin, but this has to be initialized here after the database (at least temporarily).
	// Shortly after this step, namespaces will be synced to the database and will generate notifications to adminEvents.
	if nm.adminEvents == nil {
		nm.adminEvents = adminevents.NewAdminEventManager(ctx)
	}

	if nm.identities == nil {
		nm.identities, err = nm.getIdentityPlugins(ctx)
		if err != nil {
			return err
		}
	}

	if nm.blockchains == nil {
		nm.blockchains, err = nm.getBlockchainPlugins(ctx)
		if err != nil {
			return err
		}
	}

	if nm.sharedstorages == nil {
		nm.sharedstorages, err = nm.getSharedStoragePlugins(ctx)
		if err != nil {
			return err
		}
	}

	if nm.dataexchanges == nil {
		nm.dataexchanges, err = nm.getDataExchangePlugins(ctx)
		if err != nil {
			return err
		}
	}

	if nm.tokens == nil {
		nm.tokens, err = nm.getTokensPlugins(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (nm *namespaceManager) getTokensPlugins(ctx context.Context) (plugins map[string]orchestrator.TokensPlugin, err error) {
	plugins = make(map[string]orchestrator.TokensPlugin)

	tokensConfigArraySize := tokensConfig.ArraySize()
	for i := 0; i < tokensConfigArraySize; i++ {
		config := tokensConfig.ArrayEntry(i)
		if err = nm.validatePluginConfig(ctx, config, "tokens"); err != nil {
			return nil, err
		}

		plugin, err := tifactory.GetPlugin(ctx, config.GetString(coreconfig.PluginConfigType))
		if err != nil {
			return nil, err
		}

		tokensPlugin := orchestrator.TokensPlugin{
			Plugin: plugin,
			Config: config,
			Name:   config.GetString(coreconfig.PluginConfigName),
		}
		plugins[config.GetString(coreconfig.PluginConfigName)] = tokensPlugin
	}

	// If there still is no tokens config, check the deprecated structure for config
	if len(plugins) == 0 {
		tokensConfigArraySize = deprecatedTokensConfig.ArraySize()
		if tokensConfigArraySize > 0 {
			log.L(ctx).Warnf("Your tokens config uses a deprecated configuration structure - the tokens configuration has been moved under the 'plugins' section")
		}

		for i := 0; i < tokensConfigArraySize; i++ {
			deprecatedConfig := deprecatedTokensConfig.ArrayEntry(i)
			name := deprecatedConfig.GetString(coreconfig.PluginConfigName)
			pluginName := deprecatedConfig.GetString(tokens.TokensConfigPlugin)
			if name == "" {
				return nil, i18n.NewError(ctx, coremsgs.MsgMissingTokensPluginConfig)
			}
			if err = core.ValidateFFNameField(ctx, name, "name"); err != nil {
				return nil, err
			}

			log.L(ctx).Infof("Loading tokens plugin name=%s plugin=%s", name, pluginName)
			plugin, err := tifactory.GetPlugin(ctx, pluginName)
			if err != nil {
				return nil, err
			}

			tokensPlugin := orchestrator.TokensPlugin{
				Plugin: plugin,
				Config: deprecatedConfig,
				Name:   name,
			}
			plugins[name] = tokensPlugin
		}
	}

	return plugins, err
}

func (nm *namespaceManager) getDatabasePlugins(ctx context.Context) (plugins map[string]orchestrator.DatabasePlugin, err error) {
	plugins = make(map[string]orchestrator.DatabasePlugin)
	dbConfigArraySize := databaseConfig.ArraySize()
	for i := 0; i < dbConfigArraySize; i++ {
		config := databaseConfig.ArrayEntry(i)
		if err = nm.validatePluginConfig(ctx, config, "database"); err != nil {
			return nil, err
		}

		plugin, err := difactory.GetPlugin(ctx, config.GetString(coreconfig.PluginConfigType))
		if err != nil {
			return nil, err
		}

		dbPlugin := orchestrator.DatabasePlugin{
			Plugin: plugin,
			Config: config,
		}

		plugins[config.GetString(coreconfig.PluginConfigName)] = dbPlugin
	}

	// check for deprecated config
	if len(nm.databases) == 0 {
		plugin, err := difactory.GetPlugin(ctx, deprecatedDatabaseConfig.GetString(coreconfig.PluginConfigType))
		if err != nil {
			return nil, err
		}
		deprecatedPluginName := "database_0"
		dbPlugin := orchestrator.DatabasePlugin{
			Plugin: plugin,
			Config: deprecatedDatabaseConfig,
		}

		plugins[deprecatedPluginName] = dbPlugin
	}

	return plugins, err
}

func (nm *namespaceManager) validatePluginConfig(ctx context.Context, config config.Section, sectionName string) error {
	name := config.GetString(coreconfig.PluginConfigName)
	pluginType := config.GetString(coreconfig.PluginConfigType)

	if name == "" || pluginType == "" {
		return i18n.NewError(ctx, coremsgs.MsgInvalidPluginConfiguration, sectionName)
	}

	if err := core.ValidateFFNameField(ctx, name, "name"); err != nil {
		return err
	}

	if _, ok := nm.pluginNames[name]; ok {
		return i18n.NewError(ctx, coremsgs.MsgDuplicatePluginName, name)
	}
	nm.pluginNames[name] = true

	return nil
}

func (nm *namespaceManager) getDataExchangePlugins(ctx context.Context) (plugins map[string]orchestrator.DataexchangePlugin, err error) {
	plugins = make(map[string]orchestrator.DataexchangePlugin)
	dxConfigArraySize := dataexchangeConfig.ArraySize()
	for i := 0; i < dxConfigArraySize; i++ {
		config := dataexchangeConfig.ArrayEntry(i)
		if err = nm.validatePluginConfig(ctx, config, "dataexchange"); err != nil {
			return nil, err
		}
		plugin, err := dxfactory.GetPlugin(ctx, config.GetString(coreconfig.PluginConfigType))
		if err != nil {
			return nil, err
		}
		dxPlugin := orchestrator.DataexchangePlugin{
			Plugin: plugin,
			Config: config,
		}

		plugins[config.GetString(coreconfig.PluginConfigName)] = dxPlugin
	}

	if len(plugins) == 0 {
		log.L(ctx).Warnf("Your data exchange config uses a deprecated configuration structure - the data exchange configuration has been moved under the 'plugins' section")
		deprecatedPluginName := "dataexchange_0"
		dxType := deprecatedDataexchangeConfig.GetString(coreconfig.PluginConfigType)
		plugin, err := dxfactory.GetPlugin(ctx, dxType)
		if err != nil {
			return nil, err
		}

		dxPlugin := orchestrator.DataexchangePlugin{
			Plugin: plugin,
			Config: deprecatedDataexchangeConfig,
		}

		plugins[deprecatedPluginName] = dxPlugin
	}

	return plugins, err
}

func (nm *namespaceManager) getIdentityPlugins(ctx context.Context) (plugins map[string]orchestrator.IdentityPlugin, err error) {
	plugins = make(map[string]orchestrator.IdentityPlugin)
	configSize := identityConfig.ArraySize()
	for i := 0; i < configSize; i++ {
		config := identityConfig.ArrayEntry(i)
		if err = nm.validatePluginConfig(ctx, config, "identity"); err != nil {
			return nil, err
		}
		plugin, err := iifactory.GetPlugin(ctx, config.GetString(coreconfig.PluginConfigType))
		if err != nil {
			return nil, err
		}

		idPlugin := orchestrator.IdentityPlugin{
			Plugin: plugin,
			Config: config,
		}
		plugins[config.GetString(coreconfig.PluginConfigName)] = idPlugin
	}

	return plugins, err
}

func (nm *namespaceManager) getBlockchainPlugins(ctx context.Context) (plugins map[string]orchestrator.BlockchainPlugin, err error) {
	plugins = make(map[string]orchestrator.BlockchainPlugin)
	blockchainConfigArraySize := blockchainConfig.ArraySize()
	for i := 0; i < blockchainConfigArraySize; i++ {
		config := blockchainConfig.ArrayEntry(i)
		if err = nm.validatePluginConfig(ctx, config, "blockchain"); err != nil {
			return nil, err
		}

		plugin, err := bifactory.GetPlugin(ctx, config.GetString(coreconfig.PluginConfigType))
		if err != nil {
			return nil, err
		}

		bcPlugin := orchestrator.BlockchainPlugin{
			Plugin: plugin,
			Config: config,
		}
		plugins[config.GetString(coreconfig.PluginConfigName)] = bcPlugin
	}

	// check deprecated config
	if len(plugins) == 0 {
		deprecatedPluginName := "database_0"
		biType := deprecatedBlockchainConfig.GetString(coreconfig.PluginConfigType)
		plugin, err := bifactory.GetPlugin(ctx, biType)
		if err != nil {
			return nil, err
		}

		bcPlugin := orchestrator.BlockchainPlugin{
			Plugin: plugin,
			Config: deprecatedBlockchainConfig,
		}
		plugins[deprecatedPluginName] = bcPlugin
	}

	return plugins, err
}

func (nm *namespaceManager) getSharedStoragePlugins(ctx context.Context) (plugins map[string]orchestrator.SharedStoragePlugin, err error) {
	plugins = make(map[string]orchestrator.SharedStoragePlugin)
	configSize := sharedstorageConfig.ArraySize()
	for i := 0; i < configSize; i++ {
		config := sharedstorageConfig.ArrayEntry(i)
		if err = nm.validatePluginConfig(ctx, config, "sharedstorage"); err != nil {
			return nil, err
		}
		plugin, err := ssfactory.GetPlugin(ctx, config.GetString(coreconfig.PluginConfigType))
		if err != nil {
			return nil, err
		}

		ssPlugin := orchestrator.SharedStoragePlugin{
			Plugin: plugin,
			Config: config,
		}
		plugins[config.GetString(coreconfig.PluginConfigName)] = ssPlugin
	}

	// check deprecated config
	if len(plugins) == 0 {
		deprecatedPluginName := "sharedstorage_0"
		ssType := deprecatedSharedStorageConfig.GetString(coreconfig.PluginConfigType)
		plugin, err := ssfactory.GetPlugin(ctx, ssType)
		if err != nil {
			return nil, err
		}

		ssPlugin := orchestrator.SharedStoragePlugin{
			Plugin: plugin,
			Config: deprecatedSharedStorageConfig,
		}
		plugins[deprecatedPluginName] = ssPlugin
	}

	return plugins, err
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

func (nm *namespaceManager) initNamespaces(ctx context.Context, nsConfig map[string]config.Section) (err error) {
	defaultNS := config.GetString(coreconfig.NamespacesDefault)
	i := 0
	foundDefault := false
	for name, nsObject := range nsConfig {
		if err := nm.buildAndValidateNamespaces(ctx, name, i, nsObject); err != nil {
			return err
		}
		i++
		foundDefault = foundDefault || name == defaultNS
	}
	if !foundDefault {
		return i18n.NewError(ctx, coremsgs.MsgDefaultNamespaceNotFound, defaultNS)
	}
	return err
}

func (nm *namespaceManager) buildAndValidateNamespaces(ctx context.Context, name string, index int, conf config.Section) error {
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
		for plugin := range nm.blockchains {
			plugins = append(plugins, plugin)
		}

		for plugin := range nm.dataexchanges {
			plugins = append(plugins, plugin)
		}

		for plugin := range nm.sharedstorages {
			plugins = append(plugins, plugin)
		}

		for plugin := range nm.databases {
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

	ns := namespace{
		tokens:     make(map[string]orchestrator.TokensPlugin),
		multiparty: true,
	}

	for _, pluginName := range plugins {
		if instance, ok := nm.blockchains[pluginName]; ok {
			if bcPlugin {
				return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "blockchain")
			}
			bcPlugin = true
			ns.blockchain = instance
			continue
		}
		if instance, ok := nm.dataexchanges[pluginName]; ok {
			if dxPlugin {
				return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "dataexchange")
			}
			dxPlugin = true
			ns.dataexchange = instance
			continue
		}
		if instance, ok := nm.sharedstorages[pluginName]; ok {
			if ssPlugin {
				return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "sharedstorage")
			}
			ssPlugin = true
			ns.sharedstorage = instance
			continue
		}
		if instance, ok := nm.databases[pluginName]; ok {
			if dbPlugin {
				return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "database")
			}
			dbPlugin = true
			ns.database = instance
			continue
		}
		if instance, ok := nm.tokens[pluginName]; ok {
			ns.tokens[pluginName] = instance
			continue
		}
		if instance, ok := nm.identities[pluginName]; ok {
			ns.identity = instance
			continue
		}

		return i18n.NewError(ctx, coremsgs.MsgNamespaceUnknownPlugin, name, pluginName)
	}

	if !dbPlugin || !ssPlugin || !dxPlugin || !bcPlugin {
		return i18n.NewError(ctx, coremsgs.MsgNamespaceMultipartyConfiguration, name)
	}
	nm.namespaces[name] = ns

	return nil
}

func (nm *namespaceManager) validateGatewayConfig(ctx context.Context, name string, plugins []string) error {
	var dbPlugin bool
	var bcPlugin bool

	ns := namespace{
		tokens: make(map[string]orchestrator.TokensPlugin),
	}

	for _, pluginName := range plugins {
		if instance, ok := nm.blockchains[pluginName]; ok {
			if bcPlugin {
				return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "blockchain")
			}
			bcPlugin = true
			ns.blockchain = instance
			continue
		}
		if _, ok := nm.dataexchanges[pluginName]; ok {
			return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayInvalidPlugins, name)
		}
		if _, ok := nm.sharedstorages[pluginName]; ok {
			return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayInvalidPlugins, name)
		}
		if instance, ok := nm.databases[pluginName]; ok {
			if dbPlugin {
				return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "database")
			}
			dbPlugin = true
			ns.database = instance
			continue
		}
		if instance, ok := nm.tokens[pluginName]; ok {
			ns.tokens[pluginName] = instance
			continue
		}

		return i18n.NewError(ctx, coremsgs.MsgNamespaceUnknownPlugin, name, pluginName)
	}

	if !dbPlugin {
		return i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayNoDB, name)
	}
	nm.namespaces[name] = ns

	return nil
}

func (nm *namespaceManager) Orchestrator(ns string) orchestrator.Orchestrator {
	if namespace, ok := nm.namespaces[ns]; ok {
		return namespace.orchestrator
	}
	return nil
}

func (nm *namespaceManager) GetNamespaces(ctx context.Context, filter database.AndFilter) ([]*core.Namespace, *database.FilterResult, error) {
	return nil, nil, nil
}
