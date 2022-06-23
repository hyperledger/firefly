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
	"encoding/json"
	"fmt"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/blockchain/bifactory"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/database/difactory"
	"github.com/hyperledger/firefly/internal/dataexchange/dxfactory"
	"github.com/hyperledger/firefly/internal/events/eifactory"
	"github.com/hyperledger/firefly/internal/events/system"
	"github.com/hyperledger/firefly/internal/identity/iifactory"
	"github.com/hyperledger/firefly/internal/metrics"
	multipartyManager "github.com/hyperledger/firefly/internal/multiparty"
	"github.com/hyperledger/firefly/internal/orchestrator"
	"github.com/hyperledger/firefly/internal/sharedstorage/ssfactory"
	"github.com/hyperledger/firefly/internal/spievents"
	"github.com/hyperledger/firefly/internal/tokens/tifactory"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/events"
	"github.com/hyperledger/firefly/pkg/identity"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
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
	SPIEvents() spievents.Manager
	GetNamespaces(ctx context.Context) ([]*core.Namespace, error)
	GetOperationByNamespacedID(ctx context.Context, nsOpID string) (*core.Operation, error)
	ResolveOperationByNamespacedID(ctx context.Context, nsOpID string, op *core.OperationUpdateDTO) error
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
		blockchain    map[string]blockchainPlugin
		identity      map[string]identityPlugin
		database      map[string]databasePlugin
		sharedstorage map[string]sharedStoragePlugin
		dataexchange  map[string]dataExchangePlugin
		tokens        map[string]tokensPlugin
		events        map[string]eventsPlugin
	}
	metricsEnabled bool
	metrics        metrics.Manager
	adminEvents    spievents.Manager
	utOrchestrator orchestrator.Orchestrator
}

type blockchainPlugin struct {
	config config.Section
	plugin blockchain.Plugin
}

type databasePlugin struct {
	config config.Section
	plugin database.Plugin
}

type dataExchangePlugin struct {
	config config.Section
	plugin dataexchange.Plugin
}

type sharedStoragePlugin struct {
	config config.Section
	plugin sharedstorage.Plugin
}

type tokensPlugin struct {
	config config.Section
	plugin tokens.Plugin
}

type identityPlugin struct {
	config config.Section
	plugin identity.Plugin
}

type eventsPlugin struct {
	config config.Section
	plugin events.Plugin
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

	if err = nm.loadPlugins(ctx); err != nil {
		return err
	}
	if err = nm.initPlugins(ctx); err != nil {
		return err
	}
	if err = nm.loadNamespaces(ctx); err != nil {
		return err
	}

	defaultNS := config.GetString(coreconfig.NamespacesDefault)
	var systemNS *namespace

	// Start an orchestrator per namespace
	for name, ns := range nm.namespaces {
		if err := nm.initNamespace(name, ns); err != nil {
			return err
		}

		// If the default namespace is a multiparty V1 namespace, insert the legacy ff_system namespace
		if name == defaultNS && ns.config.Multiparty.Enabled && ns.orchestrator.MultiParty().GetNetworkVersion() == 1 {
			systemNS = &namespace{}
			*systemNS = *ns
			if err := nm.initNamespace(core.LegacySystemNamespace, systemNS); err != nil {
				return err
			}
		}
	}

	if systemNS != nil {
		nm.namespaces[core.LegacySystemNamespace] = systemNS
	}
	return nil
}

func (nm *namespaceManager) initNamespace(name string, ns *namespace) error {
	or := nm.utOrchestrator
	if or == nil {
		or = orchestrator.NewOrchestrator(name, ns.config, ns.plugins, nm.metrics)
	}
	if err := or.Init(nm.ctx, nm.cancelCtx); err != nil {
		return err
	}
	ns.orchestrator = or
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
		nm.adminEvents = spievents.NewAdminEventManager(ctx)
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

	if nm.plugins.events == nil {
		nm.plugins.events, err = nm.getEventPlugins(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (nm *namespaceManager) getTokensPlugins(ctx context.Context) (plugins map[string]tokensPlugin, err error) {
	plugins = make(map[string]tokensPlugin)

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

		plugins[name] = tokensPlugin{
			config: config.SubSection(pluginType),
			plugin: plugin,
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

			plugins[name] = tokensPlugin{
				config: deprecatedConfig,
				plugin: plugin,
			}
		}
	}

	return plugins, err
}

func (nm *namespaceManager) getDatabasePlugins(ctx context.Context) (plugins map[string]databasePlugin, err error) {
	plugins = make(map[string]databasePlugin)
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

		plugins[name] = databasePlugin{
			config: config.SubSection(pluginType),
			plugin: plugin,
		}
	}

	// check for deprecated config
	if len(plugins) == 0 {
		pluginType := deprecatedDatabaseConfig.GetString(coreconfig.PluginConfigType)
		plugin, err := difactory.GetPlugin(ctx, deprecatedDatabaseConfig.GetString(coreconfig.PluginConfigType))
		if err != nil {
			return nil, err
		}
		log.L(ctx).Warnf("Your database config uses a deprecated configuration structure - the database configuration has been moved under the 'plugins' section")
		name := "database_0"
		plugins[name] = databasePlugin{
			config: deprecatedDatabaseConfig.SubSection(pluginType),
			plugin: plugin,
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

func (nm *namespaceManager) getDataExchangePlugins(ctx context.Context) (plugins map[string]dataExchangePlugin, err error) {
	plugins = make(map[string]dataExchangePlugin)
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

		plugins[name] = dataExchangePlugin{
			config: config.SubSection(pluginType),
			plugin: plugin,
		}
	}

	if len(plugins) == 0 {
		pluginType := deprecatedDataexchangeConfig.GetString(coreconfig.PluginConfigType)
		plugin, err := dxfactory.GetPlugin(ctx, pluginType)
		if err != nil {
			return nil, err
		}
		log.L(ctx).Warnf("Your data exchange config uses a deprecated configuration structure - the data exchange configuration has been moved under the 'plugins' section")
		name := "dataexchange_0"
		plugins[name] = dataExchangePlugin{
			config: deprecatedDataexchangeConfig.SubSection(pluginType),
			plugin: plugin,
		}
	}

	return plugins, err
}

func (nm *namespaceManager) getIdentityPlugins(ctx context.Context) (plugins map[string]identityPlugin, err error) {
	plugins = make(map[string]identityPlugin)
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

		plugins[name] = identityPlugin{
			config: config.SubSection(pluginType),
			plugin: plugin,
		}
	}

	return plugins, err
}

func (nm *namespaceManager) getBlockchainPlugins(ctx context.Context) (plugins map[string]blockchainPlugin, err error) {
	plugins = make(map[string]blockchainPlugin)
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

		plugins[name] = blockchainPlugin{
			config: config.SubSection(pluginType),
			plugin: plugin,
		}
	}

	// check deprecated config
	if len(plugins) == 0 {
		pluginType := deprecatedBlockchainConfig.GetString(coreconfig.PluginConfigType)
		plugin, err := bifactory.GetPlugin(ctx, pluginType)
		if err != nil {
			return nil, err
		}
		log.L(ctx).Warnf("Your blockchain config uses a deprecated configuration structure - the blockchain configuration has been moved under the 'plugins' section")
		name := "blockchain_0"
		plugins[name] = blockchainPlugin{
			config: deprecatedBlockchainConfig.SubSection(pluginType),
			plugin: plugin,
		}
	}

	return plugins, err
}

func (nm *namespaceManager) getSharedStoragePlugins(ctx context.Context) (plugins map[string]sharedStoragePlugin, err error) {
	plugins = make(map[string]sharedStoragePlugin)
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

		plugins[name] = sharedStoragePlugin{
			config: config.SubSection(pluginType),
			plugin: plugin,
		}
	}

	// check deprecated config
	if len(plugins) == 0 {
		pluginType := deprecatedSharedStorageConfig.GetString(coreconfig.PluginConfigType)
		plugin, err := ssfactory.GetPlugin(ctx, pluginType)
		if err != nil {
			return nil, err
		}
		log.L(ctx).Warnf("Your shared storage config uses a deprecated configuration structure - the shared storage configuration has been moved under the 'plugins' section")
		name := "sharedstorage_0"
		plugins[name] = sharedStoragePlugin{
			config: deprecatedSharedStorageConfig.SubSection(pluginType),
			plugin: plugin,
		}
	}

	return plugins, err
}

func (nm *namespaceManager) initPlugins(ctx context.Context) (err error) {
	for _, entry := range nm.plugins.database {
		if err = entry.plugin.Init(ctx, entry.config); err != nil {
			return err
		}
		entry.plugin.SetHandler(database.GlobalHandler, nm)
	}
	for _, entry := range nm.plugins.blockchain {
		if err = entry.plugin.Init(ctx, entry.config, nm.metrics); err != nil {
			return err
		}
	}
	for _, entry := range nm.plugins.dataexchange {
		if err = entry.plugin.Init(ctx, entry.config); err != nil {
			return err
		}
	}
	for _, entry := range nm.plugins.sharedstorage {
		if err = entry.plugin.Init(ctx, entry.config); err != nil {
			return err
		}
	}
	for name, entry := range nm.plugins.tokens {
		if err = entry.plugin.Init(ctx, name, entry.config); err != nil {
			return err
		}
	}
	for _, entry := range nm.plugins.events {
		if err = entry.plugin.Init(nm.ctx, entry.config); err != nil {
			return err
		}
	}
	return nil
}

func (nm *namespaceManager) loadNamespaces(ctx context.Context) (err error) {
	defaultNS := config.GetString(coreconfig.NamespacesDefault)
	size := namespacePredefined.ArraySize()
	foundDefault := false
	names := make(map[string]bool, size)
	for i := 0; i < size; i++ {
		nsConfig := namespacePredefined.ArrayEntry(i)
		name := nsConfig.GetString(coreconfig.NamespaceName)
		if name == "" {
			continue
		}
		if _, ok := names[name]; ok {
			log.L(ctx).Warnf("Duplicate predefined namespace (ignored): %s", name)
			continue
		}
		names[name] = true
		foundDefault = foundDefault || name == defaultNS
		if nm.namespaces[name], err = nm.loadNamespace(ctx, name, i, nsConfig); err != nil {
			return err
		}
	}
	if !foundDefault {
		return i18n.NewError(ctx, coremsgs.MsgDefaultNamespaceNotFound, defaultNS)
	}
	return err
}

func (nm *namespaceManager) loadNamespace(ctx context.Context, name string, index int, conf config.Section) (*namespace, error) {
	if err := core.ValidateFFNameField(ctx, name, fmt.Sprintf("namespaces.predefined[%d].name", index)); err != nil {
		return nil, err
	}
	if name == core.LegacySystemNamespace || conf.GetString(coreconfig.NamespaceRemoteName) == core.LegacySystemNamespace {
		return nil, i18n.NewError(ctx, coremsgs.MsgFFSystemReservedName, core.LegacySystemNamespace)
	}

	multipartyConf := conf.SubSection(coreconfig.NamespaceMultiparty)
	// If any multiparty org information is configured (here or at the root), assume multiparty mode by default
	orgName := multipartyConf.GetString(coreconfig.NamespaceMultipartyOrgName)
	orgKey := multipartyConf.GetString(coreconfig.NamespaceMultipartyOrgKey)
	orgDesc := multipartyConf.GetString(coreconfig.NamespaceMultipartyOrgDescription)
	deprecatedOrgName := config.GetString(coreconfig.OrgName)
	deprecatedOrgKey := config.GetString(coreconfig.OrgKey)
	deprecatedOrgDesc := config.GetString(coreconfig.OrgDescription)
	if deprecatedOrgName != "" || deprecatedOrgKey != "" || deprecatedOrgDesc != "" {
		log.L(ctx).Warnf("Your org config uses a deprecated configuration structure - the org configuration has been moved under the 'namespaces.predefined[].multiparty' section")
	}
	if orgName == "" {
		orgName = deprecatedOrgName
	}
	if orgKey == "" {
		orgKey = deprecatedOrgKey
	}
	if orgDesc == "" {
		orgDesc = deprecatedOrgDesc
	}
	multiparty := multipartyConf.Get(coreconfig.NamespaceMultipartyEnabled)
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
	var err error
	if multiparty.(bool) {
		contractsConf := multipartyConf.SubArray(coreconfig.NamespaceMultipartyContract)
		contractConfArraySize := contractsConf.ArraySize()
		contracts := make([]multipartyManager.Contract, contractConfArraySize)

		for i := 0; i < contractConfArraySize; i++ {
			conf := contractsConf.ArrayEntry(i)
			locationObject := conf.GetObject(coreconfig.NamespaceMultipartyContractLocation)
			b, err := json.Marshal(locationObject)
			if err != nil {
				return nil, err
			}
			location := fftypes.JSONAnyPtrBytes(b)
			contract := multipartyManager.Contract{
				Location:   location,
				FirstEvent: conf.GetString(coreconfig.NamespaceMultipartyContractFirstEvent),
			}
			contracts[i] = contract
		}

		config.Multiparty.Enabled = true
		config.Multiparty.Org.Name = orgName
		config.Multiparty.Org.Key = orgKey
		config.Multiparty.Org.Description = orgDesc
		config.Multiparty.Contracts = contracts
		p, err = nm.validateMultiPartyConfig(ctx, name, plugins)
	} else {
		p, err = nm.validateGatewayConfig(ctx, name, plugins)
	}
	if err != nil {
		return nil, err
	}

	p.Events = make(map[string]events.Plugin, len(nm.plugins.events))
	for name, entry := range nm.plugins.events {
		p.Events[name] = entry.plugin
	}

	return &namespace{
		description: conf.GetString(coreconfig.NamespaceDescription),
		config:      config,
		plugins:     *p,
	}, nil
}

func (nm *namespaceManager) validateMultiPartyConfig(ctx context.Context, name string, plugins []string) (*orchestrator.Plugins, error) {
	var result orchestrator.Plugins
	for _, pluginName := range plugins {
		if instance, ok := nm.plugins.blockchain[pluginName]; ok {
			if result.Blockchain.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "blockchain")
			}
			result.Blockchain = orchestrator.BlockchainPlugin{
				Name:   pluginName,
				Plugin: instance.plugin,
			}
			continue
		}
		if instance, ok := nm.plugins.dataexchange[pluginName]; ok {
			if result.DataExchange.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "dataexchange")
			}
			result.DataExchange = orchestrator.DataExchangePlugin{
				Name:   pluginName,
				Plugin: instance.plugin,
			}
			continue
		}
		if instance, ok := nm.plugins.sharedstorage[pluginName]; ok {
			if result.SharedStorage.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "sharedstorage")
			}
			result.SharedStorage = orchestrator.SharedStoragePlugin{
				Name:   pluginName,
				Plugin: instance.plugin,
			}
			continue
		}
		if instance, ok := nm.plugins.database[pluginName]; ok {
			if result.Database.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayMultiplePluginType, name, "database")
			}
			result.Database = orchestrator.DatabasePlugin{
				Name:   pluginName,
				Plugin: instance.plugin,
			}
			continue
		}
		if instance, ok := nm.plugins.tokens[pluginName]; ok {
			result.Tokens = append(result.Tokens, orchestrator.TokensPlugin{
				Name:   pluginName,
				Plugin: instance.plugin,
			})
			continue
		}
		if instance, ok := nm.plugins.identity[pluginName]; ok {
			result.Identity = orchestrator.IdentityPlugin{
				Name:   pluginName,
				Plugin: instance.plugin,
			}
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
			result.Blockchain = orchestrator.BlockchainPlugin{
				Name:   pluginName,
				Plugin: instance.plugin,
			}
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
			result.Database = orchestrator.DatabasePlugin{
				Name:   pluginName,
				Plugin: instance.plugin,
			}
			continue
		}
		if instance, ok := nm.plugins.tokens[pluginName]; ok {
			result.Tokens = append(result.Tokens, orchestrator.TokensPlugin{
				Name:   pluginName,
				Plugin: instance.plugin,
			})
			continue
		}

		return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceUnknownPlugin, name, pluginName)
	}

	if result.Database.Plugin == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceGatewayNoDB, name)
	}

	return &result, nil
}

func (nm *namespaceManager) SPIEvents() spievents.Manager {
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

func (nm *namespaceManager) GetOperationByNamespacedID(ctx context.Context, nsOpID string) (*core.Operation, error) {
	ns, u, err := core.ParseNamespacedOpID(ctx, nsOpID)
	if err != nil {
		return nil, err
	}
	or := nm.Orchestrator(ns)
	if or == nil {
		return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
	}
	return or.GetOperationByID(ctx, u.String())
}

func (nm *namespaceManager) ResolveOperationByNamespacedID(ctx context.Context, nsOpID string, op *core.OperationUpdateDTO) error {
	ns, u, err := core.ParseNamespacedOpID(ctx, nsOpID)
	if err != nil {
		return err
	}
	or := nm.Orchestrator(ns)
	if or == nil {
		return i18n.NewError(ctx, coremsgs.Msg404NotFound)
	}
	return or.Operations().ResolveOperationByID(ctx, u, op)
}

func (nm *namespaceManager) getEventPlugins(ctx context.Context) (plugins map[string]eventsPlugin, err error) {
	plugins = make(map[string]eventsPlugin)
	enabledTransports := config.GetStringSlice(coreconfig.EventTransportsEnabled)
	uniqueTransports := make(map[string]bool)
	for _, transport := range enabledTransports {
		uniqueTransports[transport] = true
	}
	// Cannot disable the internal listener
	uniqueTransports[system.SystemEventsTransport] = true
	for transport := range uniqueTransports {
		plugin, err := eifactory.GetPlugin(ctx, transport)
		if err != nil {
			return nil, err
		}

		name := plugin.Name()
		section := config.RootSection("events").SubSection(name)
		plugin.InitConfig(section)
		plugins[name] = eventsPlugin{
			config: section,
			plugin: plugin,
		}
	}
	return plugins, err
}
