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
	AdminEvents() adminevents.Manager
	GetNamespaces(ctx context.Context) ([]*core.Namespace, error)
}

type namespace struct {
	description  string
	orchestrator orchestrator.Orchestrator
	config       orchestrator.Config
	plugins      orchestrator.Plugins
}

type namespaceManager struct {
	ctx         context.Context
	cancelCtx   context.CancelFunc
	namespaces  map[string]*namespace
	pluginNames map[string]bool
	plugins     struct {
		blockchain    map[string]orchestrator.BlockchainPlugin
		identity      map[string]orchestrator.IdentityPlugin
		database      map[string]orchestrator.DatabasePlugin
		sharedstorage map[string]orchestrator.SharedStoragePlugin
		dataexchange  map[string]orchestrator.DataExchangePlugin
		tokens        map[string]orchestrator.TokensPlugin
	}
	metricsEnabled bool
	metrics        metrics.Manager
	adminEvents    adminevents.Manager
	utOrchestrator orchestrator.Orchestrator
}

func NewNamespaceManager(withDefaults bool) Manager {
	nm := &namespaceManager{
		namespaces:     make(map[string]*namespace),
		metricsEnabled: config.GetBool(coreconfig.MetricsEnabled),
	}

	InitConfig(withDefaults)

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

	err = nm.loadPlugins(ctx)
	if err != nil {
		return err
	}

	if err = nm.loadNamespaces(ctx); err != nil {
		return err
	}

	// Start an orchestrator per namespace
	for name, ns := range nm.namespaces {
		or := nm.utOrchestrator
		if or == nil {
			or = orchestrator.NewOrchestrator(name, ns.config, ns.plugins, nm.metrics, nm.adminEvents)
		}
		if err = or.Init(ctx, cancelCtx); err != nil {
			return err
		}
		ns.orchestrator = or
	}
	return nil
}

func (nm *namespaceManager) Start() error {
	if nm.metricsEnabled {
		// Ensure metrics are registered
		metrics.Registry()
	}
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
	nm.adminEvents.WaitStop()
}

func (nm *namespaceManager) loadPlugins(ctx context.Context) (err error) {
	nm.pluginNames = make(map[string]bool)
	if nm.metrics == nil {
		nm.metrics = metrics.NewMetricsManager(ctx)
	}

	if nm.plugins.database == nil {
		nm.plugins.database, err = nm.getDatabasePlugins(ctx)
		if err != nil {
			return err
		}
	}

	if nm.adminEvents == nil {
		nm.adminEvents = adminevents.NewAdminEventManager(ctx)
	}

	if nm.plugins.identity == nil {
		nm.plugins.identity, err = nm.getIdentityPlugins(ctx)
		if err != nil {
			return err
		}
	}

	if nm.plugins.blockchain == nil {
		nm.plugins.blockchain, err = nm.getBlockchainPlugins(ctx)
		if err != nil {
			return err
		}
	}

	if nm.plugins.sharedstorage == nil {
		nm.plugins.sharedstorage, err = nm.getSharedStoragePlugins(ctx)
		if err != nil {
			return err
		}
	}

	if nm.plugins.dataexchange == nil {
		nm.plugins.dataexchange, err = nm.getDataExchangePlugins(ctx)
		if err != nil {
			return err
		}
	}

	if nm.plugins.tokens == nil {
		nm.plugins.tokens, err = nm.getTokensPlugins(ctx)
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
		name, pluginType, err := nm.validatePluginConfig(ctx, config, "tokens")
		if err != nil {
			return nil, err
		}

		plugin, err := tifactory.GetPlugin(ctx, pluginType)
		if err != nil {
			return nil, err
		}

		plugins[name] = orchestrator.TokensPlugin{
			Name:   name,
			Plugin: plugin,
			Config: config.SubSection(pluginType),
		}
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
			pluginType := deprecatedConfig.GetString(tokens.TokensConfigPlugin)
			if name == "" || pluginType == "" {
				return nil, i18n.NewError(ctx, coremsgs.MsgMissingTokensPluginConfig)
			}
			if err = core.ValidateFFNameField(ctx, name, "name"); err != nil {
				return nil, err
			}

			plugin, err := tifactory.GetPlugin(ctx, pluginType)
			if err != nil {
				return nil, err
			}

			plugins[name] = orchestrator.TokensPlugin{
				Name:   name,
				Plugin: plugin,
				Config: deprecatedConfig,
			}
		}
	}

	return plugins, err
}

func (nm *namespaceManager) getDatabasePlugins(ctx context.Context) (plugins map[string]orchestrator.DatabasePlugin, err error) {
	plugins = make(map[string]orchestrator.DatabasePlugin)
	dbConfigArraySize := databaseConfig.ArraySize()
	for i := 0; i < dbConfigArraySize; i++ {
		config := databaseConfig.ArrayEntry(i)
		name, pluginType, err := nm.validatePluginConfig(ctx, config, "database")
		if err != nil {
			return nil, err
		}

		plugin, err := difactory.GetPlugin(ctx, pluginType)
		if err != nil {
			return nil, err
		}

		plugins[name] = orchestrator.DatabasePlugin{
			Name:   name,
			Plugin: plugin,
			Config: config.SubSection(pluginType),
		}
	}

	// check for deprecated config
	if len(plugins) == 0 {
		pluginType := deprecatedDatabaseConfig.GetString(coreconfig.PluginConfigType)
		plugin, err := difactory.GetPlugin(ctx, deprecatedDatabaseConfig.GetString(coreconfig.PluginConfigType))
		if err != nil {
			return nil, err
		}
		name := "database_0"
		plugins[name] = orchestrator.DatabasePlugin{
			Name:   name,
			Plugin: plugin,
			Config: deprecatedDatabaseConfig.SubSection(pluginType),
		}
	}

	return plugins, err
}

func (nm *namespaceManager) validatePluginConfig(ctx context.Context, config config.Section, sectionName string) (name, pluginType string, err error) {
	name = config.GetString(coreconfig.PluginConfigName)
	pluginType = config.GetString(coreconfig.PluginConfigType)

	if name == "" || pluginType == "" {
		return "", "", i18n.NewError(ctx, coremsgs.MsgInvalidPluginConfiguration, sectionName)
	}

	if err := core.ValidateFFNameField(ctx, name, "name"); err != nil {
		return "", "", err
	}

	if _, ok := nm.pluginNames[name]; ok {
		return "", "", i18n.NewError(ctx, coremsgs.MsgDuplicatePluginName, name)
	}
	nm.pluginNames[name] = true

	return name, pluginType, nil
}

func (nm *namespaceManager) getDataExchangePlugins(ctx context.Context) (plugins map[string]orchestrator.DataExchangePlugin, err error) {
	plugins = make(map[string]orchestrator.DataExchangePlugin)
	dxConfigArraySize := dataexchangeConfig.ArraySize()
	for i := 0; i < dxConfigArraySize; i++ {
		config := dataexchangeConfig.ArrayEntry(i)
		name, pluginType, err := nm.validatePluginConfig(ctx, config, "dataexchange")
		if err != nil {
			return nil, err
		}

		plugin, err := dxfactory.GetPlugin(ctx, pluginType)
		if err != nil {
			return nil, err
		}

		plugins[name] = orchestrator.DataExchangePlugin{
			Name:   name,
			Plugin: plugin,
			Config: config.SubSection(pluginType),
		}
	}

	if len(plugins) == 0 {
		log.L(ctx).Warnf("Your data exchange config uses a deprecated configuration structure - the data exchange configuration has been moved under the 'plugins' section")
		pluginType := deprecatedDataexchangeConfig.GetString(coreconfig.PluginConfigType)
		plugin, err := dxfactory.GetPlugin(ctx, pluginType)
		if err != nil {
			return nil, err
		}

		name := "dataexchange_0"
		plugins[name] = orchestrator.DataExchangePlugin{
			Name:   name,
			Plugin: plugin,
			Config: deprecatedDataexchangeConfig.SubSection(pluginType),
		}
	}

	return plugins, err
}

func (nm *namespaceManager) getIdentityPlugins(ctx context.Context) (plugins map[string]orchestrator.IdentityPlugin, err error) {
	plugins = make(map[string]orchestrator.IdentityPlugin)
	configSize := identityConfig.ArraySize()
	for i := 0; i < configSize; i++ {
		config := identityConfig.ArrayEntry(i)
		name, pluginType, err := nm.validatePluginConfig(ctx, config, "identity")
		if err != nil {
			return nil, err
		}

		plugin, err := iifactory.GetPlugin(ctx, pluginType)
		if err != nil {
			return nil, err
		}

		plugins[name] = orchestrator.IdentityPlugin{
			Name:   name,
			Plugin: plugin,
			Config: config.SubSection(pluginType),
		}
	}

	return plugins, err
}

func (nm *namespaceManager) getBlockchainPlugins(ctx context.Context) (plugins map[string]orchestrator.BlockchainPlugin, err error) {
	plugins = make(map[string]orchestrator.BlockchainPlugin)
	blockchainConfigArraySize := blockchainConfig.ArraySize()
	for i := 0; i < blockchainConfigArraySize; i++ {
		config := blockchainConfig.ArrayEntry(i)
		name, pluginType, err := nm.validatePluginConfig(ctx, config, "blockchain")
		if err != nil {
			return nil, err
		}

		plugin, err := bifactory.GetPlugin(ctx, pluginType)
		if err != nil {
			return nil, err
		}

		plugins[name] = orchestrator.BlockchainPlugin{
			Name:   name,
			Plugin: plugin,
			Config: config.SubSection(pluginType),
		}
	}

	// check deprecated config
	if len(plugins) == 0 {
		pluginType := deprecatedBlockchainConfig.GetString(coreconfig.PluginConfigType)
		plugin, err := bifactory.GetPlugin(ctx, pluginType)
		if err != nil {
			return nil, err
		}

		name := "blockchain_0"
		plugins[name] = orchestrator.BlockchainPlugin{
			Name:   name,
			Plugin: plugin,
			Config: deprecatedBlockchainConfig.SubSection(pluginType),
		}
	}

	return plugins, err
}

func (nm *namespaceManager) getSharedStoragePlugins(ctx context.Context) (plugins map[string]orchestrator.SharedStoragePlugin, err error) {
	plugins = make(map[string]orchestrator.SharedStoragePlugin)
	configSize := sharedstorageConfig.ArraySize()
	for i := 0; i < configSize; i++ {
		config := sharedstorageConfig.ArrayEntry(i)
		name, pluginType, err := nm.validatePluginConfig(ctx, config, "sharedstorage")
		if err != nil {
			return nil, err
		}

		plugin, err := ssfactory.GetPlugin(ctx, pluginType)
		if err != nil {
			return nil, err
		}

		plugins[name] = orchestrator.SharedStoragePlugin{
			Name:   name,
			Plugin: plugin,
			Config: config.SubSection(pluginType),
		}
	}

	// check deprecated config
	if len(plugins) == 0 {
		pluginType := deprecatedSharedStorageConfig.GetString(coreconfig.PluginConfigType)
		plugin, err := ssfactory.GetPlugin(ctx, pluginType)
		if err != nil {
			return nil, err
		}

		name := "sharedstorage_0"
		plugins[name] = orchestrator.SharedStoragePlugin{
			Name:   name,
			Plugin: plugin,
			Config: deprecatedSharedStorageConfig.SubSection(pluginType),
		}
	}

	return plugins, err
}

func (nm *namespaceManager) loadNamespaces(ctx context.Context) (err error) {
	defaultNS := config.GetString(coreconfig.NamespacesDefault)
	size := namespacePredefined.ArraySize()
	namespaces := make(map[string]config.Section, size)
	for i := 0; i < size; i++ {
		nsConfig := namespacePredefined.ArrayEntry(i)
		name := nsConfig.GetString(coreconfig.NamespaceName)
		if name != "" {
			if _, ok := namespaces[name]; ok {
				log.L(ctx).Warnf("Duplicate predefined namespace (ignored): %s", name)
			}
			namespaces[name] = nsConfig
		}
	}

	i := 0
	foundDefault := false
	for name, nsConfig := range namespaces {
		if err := nm.loadNamespace(ctx, name, i, nsConfig); err != nil {
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

func (nm *namespaceManager) loadNamespace(ctx context.Context, name string, index int, conf config.Section) (err error) {
	if err := core.ValidateFFNameField(ctx, name, fmt.Sprintf("namespaces.predefined[%d].name", index)); err != nil {
		return err
	}

	if name == core.LegacySystemNamespace || conf.GetString(coreconfig.NamespaceRemoteName) == core.LegacySystemNamespace {
		return i18n.NewError(ctx, coremsgs.MsgFFSystemReservedName, core.LegacySystemNamespace)
	}

	// If any multiparty org information is configured (here or at the root), assume multiparty mode by default
	orgName := conf.GetString(coreconfig.NamespaceMultipartyOrgName)
	if orgName == "" {
		orgName = config.GetString(coreconfig.OrgName)
	}
	orgKey := conf.GetString(coreconfig.NamespaceMultipartyOrgKey)
	if orgKey == "" {
		orgKey = config.GetString(coreconfig.OrgKey)
	}
	orgDesc := conf.GetString(coreconfig.NamespaceMultipartyOrgDescription)
	if orgDesc == "" {
		orgDesc = config.GetString(coreconfig.OrgDescription)
	}
	multiparty := conf.Get(coreconfig.NamespaceMultipartyEnabled)
	if multiparty == nil {
		multiparty = orgName != "" || orgKey != ""
	}

	// If no plugins are listed under this namespace, use all defined plugins by default
	pluginsRaw := conf.Get(coreconfig.NamespacePlugins)
	plugins := conf.GetStringSlice(coreconfig.NamespacePlugins)
	if pluginsRaw == nil {
		for plugin := range nm.plugins.blockchain {
			plugins = append(plugins, plugin)
		}

		for plugin := range nm.plugins.dataexchange {
			plugins = append(plugins, plugin)
		}

		for plugin := range nm.plugins.sharedstorage {
			plugins = append(plugins, plugin)
		}

		for plugin := range nm.plugins.database {
			plugins = append(plugins, plugin)
		}

		for plugin := range nm.plugins.identity {
			plugins = append(plugins, plugin)
		}

		for plugin := range nm.plugins.tokens {
			plugins = append(plugins, plugin)
		}
	}

	config := orchestrator.Config{
		DefaultKey: conf.GetString(coreconfig.NamespaceDefaultKey),
	}
	var p *orchestrator.Plugins
	if multiparty.(bool) {
		config.Multiparty.Enabled = true
		config.Multiparty.OrgName = orgName
		config.Multiparty.OrgKey = orgKey
		config.Multiparty.OrgDesc = orgDesc
		p, err = nm.validateMultiPartyConfig(ctx, name, plugins)
	} else {
		p, err = nm.validateGatewayConfig(ctx, name, plugins)
	}

	if err == nil {
		nm.namespaces[name] = &namespace{
			description: conf.GetString(coreconfig.NamespaceDescription),
			config:      config,
			plugins:     *p,
		}
	}
	return err
}

func (nm *namespaceManager) validateMultiPartyConfig(ctx context.Context, name string, plugins []string) (*orchestrator.Plugins, error) {
	var result orchestrator.Plugins
	for _, pluginName := range plugins {
		if instance, ok := nm.plugins.blockchain[pluginName]; ok {
			if result.Blockchain.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "blockchain")
			}
			result.Blockchain = instance
			continue
		}
		if instance, ok := nm.plugins.dataexchange[pluginName]; ok {
			if result.DataExchange.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "dataexchange")
			}
			result.DataExchange = instance
			continue
		}
		if instance, ok := nm.plugins.sharedstorage[pluginName]; ok {
			if result.SharedStorage.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "sharedstorage")
			}
			result.SharedStorage = instance
			continue
		}
		if instance, ok := nm.plugins.database[pluginName]; ok {
			if result.Database.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "database")
			}
			result.Database = instance
			continue
		}
		if instance, ok := nm.plugins.tokens[pluginName]; ok {
			result.Tokens = append(result.Tokens, instance)
			continue
		}
		if instance, ok := nm.plugins.identity[pluginName]; ok {
			result.Identity = instance
			continue
		}

		return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceUnknownPlugin, name, pluginName)
	}

	if result.Database.Plugin == nil ||
		result.SharedStorage.Plugin == nil ||
		result.DataExchange.Plugin == nil ||
		result.Blockchain.Plugin == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceMultipartyConfiguration, name)
	}

	return &result, nil
}

func (nm *namespaceManager) validateGatewayConfig(ctx context.Context, name string, plugins []string) (*orchestrator.Plugins, error) {
	var result orchestrator.Plugins
	for _, pluginName := range plugins {
		if instance, ok := nm.plugins.blockchain[pluginName]; ok {
			if result.Blockchain.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "blockchain")
			}
			result.Blockchain = instance
			continue
		}
		if _, ok := nm.plugins.dataexchange[pluginName]; ok {
			return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayInvalidPlugins, name)
		}
		if _, ok := nm.plugins.sharedstorage[pluginName]; ok {
			return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayInvalidPlugins, name)
		}
		if instance, ok := nm.plugins.database[pluginName]; ok {
			if result.Database.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "database")
			}
			result.Database = instance
			continue
		}
		if instance, ok := nm.plugins.tokens[pluginName]; ok {
			result.Tokens = append(result.Tokens, instance)
			continue
		}

		return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceUnknownPlugin, name, pluginName)
	}

	if result.Database.Plugin == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayNoDB, name)
	}

	return &result, nil
}

func (nm *namespaceManager) AdminEvents() adminevents.Manager {
	return nm.adminEvents
}

func (nm *namespaceManager) Orchestrator(ns string) orchestrator.Orchestrator {
	if namespace, ok := nm.namespaces[ns]; ok {
		return namespace.orchestrator
	}
	return nil
}

func (nm *namespaceManager) GetNamespaces(ctx context.Context) ([]*core.Namespace, error) {
	results := make([]*core.Namespace, 0, len(nm.namespaces))
	for name, ns := range nm.namespaces {
		results = append(results, &core.Namespace{
			Name:        name,
			Description: ns.description,
		})
	}
	return results, nil
}
