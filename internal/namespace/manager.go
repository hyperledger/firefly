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
	"strconv"
	"sync"

	"github.com/hyperledger/firefly-common/pkg/auth"
	"github.com/hyperledger/firefly-common/pkg/auth/authfactory"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/blockchain/bifactory"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/database/difactory"
	"github.com/hyperledger/firefly/internal/dataexchange/dxfactory"
	"github.com/hyperledger/firefly/internal/events/eifactory"
	"github.com/hyperledger/firefly/internal/events/system"
	"github.com/hyperledger/firefly/internal/identity/iifactory"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/multiparty"
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
	authConfig          = config.RootArray("plugins.auth")

	// Deprecated configs
	deprecatedTokensConfig        = config.RootArray("tokens")
	deprecatedBlockchainConfig    = config.RootSection("blockchain")
	deprecatedDatabaseConfig      = config.RootSection("database")
	deprecatedSharedStorageConfig = config.RootSection("sharedstorage")
	deprecatedDataexchangeConfig  = config.RootSection("dataexchange")
)

type Manager interface {
	Init(ctx context.Context, cancelCtx context.CancelFunc, reset chan bool) error
	Start() error
	WaitStop()
	Reset(ctx context.Context)

	Orchestrator(ns string) orchestrator.Orchestrator
	SPIEvents() spievents.Manager
	GetNamespaces(ctx context.Context) ([]*core.Namespace, error)
	GetOperationByNamespacedID(ctx context.Context, nsOpID string) (*core.Operation, error)
	ResolveOperationByNamespacedID(ctx context.Context, nsOpID string, op *core.OperationUpdateDTO) error
	Authorize(ctx context.Context, authReq *fftypes.AuthReq) error
}

type namespace struct {
	core.Namespace
	orchestrator orchestrator.Orchestrator
	config       orchestrator.Config
	plugins      []string
}

type namespaceManager struct {
	reset       chan bool
	nsMux       sync.Mutex
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
		auth          map[string]authPlugin
	}
	metricsEnabled      bool
	cacheManager        cache.Manager
	metrics             metrics.Manager
	adminEvents         spievents.Manager
	utOrchestrator      orchestrator.Orchestrator
	tokenBroadcastNames map[string]string
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

type authPlugin struct {
	config config.Section
	plugin auth.Plugin
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func NewNamespaceManager(withDefaults bool) Manager {
	nm := &namespaceManager{
		namespaces:          make(map[string]*namespace),
		metricsEnabled:      config.GetBool(coreconfig.MetricsEnabled),
		tokenBroadcastNames: make(map[string]string),
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
	authfactory.InitConfigArray(authConfig)

	// Events still live at the root of the config
	eifactory.InitConfig(config.RootSection("events"))

	return nm
}

func (nm *namespaceManager) Init(ctx context.Context, cancelCtx context.CancelFunc, reset chan bool) (err error) {
	nm.reset = reset

	if err = nm.loadPlugins(ctx); err != nil {
		return err
	}
	if err = nm.initPlugins(ctx, cancelCtx); err != nil {
		return err
	}
	if err = nm.loadNamespaces(ctx); err != nil {
		return err
	}

	if nm.metricsEnabled {
		// Ensure metrics are registered
		metrics.Registry()
	}

	// In network version 1, the blockchain plugin and multiparty contract were global and singular.
	// Therefore, if any namespace was EVER pointed at a V1 contract, that contract and that namespace's plugins
	// become the de facto configuration for ff_system as well. There can only be one V1 contract in the history
	// of a given FireFly node, because it's impossible to re-create ff_system against a different contract
	// or different set of plugins.
	var v1Namespace *namespace
	var v1Contract *core.MultipartyContract

	for _, ns := range nm.namespaces {
		if err := nm.initNamespace(ctx, ns); err != nil {
			return err
		}
		multiparty := ns.config.Multiparty.Enabled
		version := "n/a"
		if multiparty {
			version = fmt.Sprintf("%d", ns.Namespace.Contracts.Active.Info.Version)
		}
		log.L(ctx).Infof("Initialized namespace '%s' multiparty=%s version=%s", ns.Name, strconv.FormatBool(multiparty), version)
		if multiparty {
			contract := nm.findV1Contract(ns)
			if contract != nil {
				if v1Namespace == nil {
					v1Namespace = ns
					v1Contract = contract
				} else if !stringSlicesEqual(v1Namespace.plugins, ns.plugins) ||
					v1Contract.Location.String() != contract.Location.String() ||
					v1Contract.FirstEvent != contract.FirstEvent {
					return i18n.NewError(ctx, coremsgs.MsgCannotInitLegacyNS, core.LegacySystemNamespace, v1Namespace.Name, ns.Name)
				}
			}
		}
	}

	if v1Namespace != nil {
		systemNS := &namespace{
			Namespace: v1Namespace.Namespace,
			config:    v1Namespace.config,
			plugins:   v1Namespace.plugins,
		}
		systemNS.Name = core.LegacySystemNamespace
		systemNS.NetworkName = core.LegacySystemNamespace
		nm.namespaces[core.LegacySystemNamespace] = systemNS
		err = nm.initNamespace(ctx, systemNS)
		log.L(ctx).Infof("Initialized namespace '%s' as a copy of '%s'", core.LegacySystemNamespace, v1Namespace.Name)
	}
	return err
}

func (nm *namespaceManager) findV1Contract(ns *namespace) *core.MultipartyContract {
	if ns.Contracts.Active.Info.Version == 1 {
		return ns.Contracts.Active
	}
	for _, contract := range ns.Contracts.Terminated {
		if contract.Info.Version == 1 {
			return contract
		}
	}
	return nil
}

func (nm *namespaceManager) initNamespace(ctx context.Context, ns *namespace) (err error) {
	var plugins *orchestrator.Plugins
	if ns.config.Multiparty.Enabled {
		plugins, err = nm.validateMultiPartyConfig(ctx, ns.Name, ns.plugins)
	} else {
		plugins, err = nm.validateNonMultipartyConfig(ctx, ns.Name, ns.plugins)
	}
	if err != nil {
		return err
	}

	database := plugins.Database.Plugin
	existing, err := database.GetNamespace(ctx, ns.Name)
	switch {
	case err != nil:
		return err
	case existing != nil:
		ns.Created = existing.Created
		ns.Contracts = existing.Contracts
		if ns.NetworkName != existing.NetworkName {
			log.L(ctx).Warnf("Namespace '%s' - network name unexpectedly changed from '%s' to '%s'", ns.Name, existing.NetworkName, ns.NetworkName)
		}
	default:
		ns.Created = fftypes.Now()
		ns.Contracts = &core.MultipartyContracts{
			Active: &core.MultipartyContract{},
		}
	}
	if err = database.UpsertNamespace(ctx, &ns.Namespace, true); err != nil {
		return err
	}

	or := nm.utOrchestrator
	if or == nil {
		or = orchestrator.NewOrchestrator(&ns.Namespace, ns.config, plugins, nm.metrics, nm.cacheManager)
	}
	ns.orchestrator = or
	orCtx, orCancel := context.WithCancel(ctx)
	if err := or.Init(orCtx, orCancel); err != nil {
		return err
	}
	go func() {
		<-orCtx.Done()
		nm.nsMux.Lock()
		defer nm.nsMux.Unlock()
		log.L(ctx).Infof("Terminated namespace '%s'", ns.Name)
		delete(nm.namespaces, ns.Name)
	}()
	return nil
}

func (nm *namespaceManager) Start() error {
	// Orchestrators must be started before plugins so as not to miss events
	for _, ns := range nm.namespaces {
		if err := ns.orchestrator.Start(); err != nil {
			return err
		}
	}
	for _, plugin := range nm.plugins.blockchain {
		if err := plugin.plugin.Start(); err != nil {
			return err
		}
	}
	for _, plugin := range nm.plugins.dataexchange {
		if err := plugin.plugin.Start(); err != nil {
			return err
		}
	}
	for _, plugin := range nm.plugins.tokens {
		if err := plugin.plugin.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (nm *namespaceManager) WaitStop() {
	nm.nsMux.Lock()
	namespaces := make(map[string]*namespace, len(nm.namespaces))
	for k, v := range nm.namespaces {
		namespaces[k] = v
	}
	nm.nsMux.Unlock()

	for _, ns := range namespaces {
		ns.orchestrator.WaitStop()
	}
	nm.adminEvents.WaitStop()
}

func (nm *namespaceManager) Reset(ctx context.Context) {
	// Queue a restart of the root context to pick up a configuration change.
	// Caller is responsible for terminating the passed context to trigger the actual reset
	// (allows caller to cleanly finish processing the current request/event).
	go func() {
		<-ctx.Done()
		nm.reset <- true
	}()
}

func (nm *namespaceManager) loadPlugins(ctx context.Context) (err error) {
	nm.pluginNames = make(map[string]bool)
	if nm.metrics == nil {
		nm.metrics = metrics.NewMetricsManager(ctx)
	}

	if nm.cacheManager == nil {
		nm.cacheManager = cache.NewCacheManager(ctx)
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

	if nm.plugins.auth == nil {
		nm.plugins.auth, err = nm.getAuthPlugin(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (nm *namespaceManager) getTokensPlugins(ctx context.Context) (plugins map[string]tokensPlugin, err error) {
	plugins = make(map[string]tokensPlugin)
	// Broadcast names must be unique
	broadcastNames := make(map[string]bool)

	tokensConfigArraySize := tokensConfig.ArraySize()
	for i := 0; i < tokensConfigArraySize; i++ {
		config := tokensConfig.ArrayEntry(i)
		name, pluginType, err := nm.validatePluginConfig(ctx, config, "tokens")
		if err != nil {
			return nil, err
		}
		broadcastName := config.GetString(coreconfig.PluginBroadcastName)
		// If there is no broadcast name, use the plugin name
		if broadcastName == "" {
			broadcastName = name
		}
		if _, exists := broadcastNames[broadcastName]; exists {
			return nil, i18n.NewError(ctx, coremsgs.MsgDuplicatePluginBroadcastName, "tokens", broadcastName)
		}
		broadcastNames[broadcastName] = true
		nm.tokenBroadcastNames[name] = broadcastName

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
			if err = fftypes.ValidateFFNameField(ctx, name, "name"); err != nil {
				return nil, err
			}
			nm.tokenBroadcastNames[name] = name

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
		if pluginType != "" {
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
	}

	return plugins, err
}

func (nm *namespaceManager) validatePluginConfig(ctx context.Context, config config.Section, sectionName string) (name, pluginType string, err error) {
	name = config.GetString(coreconfig.PluginConfigName)
	pluginType = config.GetString(coreconfig.PluginConfigType)

	if name == "" || pluginType == "" {
		return "", "", i18n.NewError(ctx, coremsgs.MsgInvalidPluginConfiguration, sectionName)
	}

	if err := fftypes.ValidateFFNameField(ctx, name, "name"); err != nil {
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

	// check deprecated config
	if len(plugins) == 0 {
		pluginType := deprecatedDataexchangeConfig.GetString(coreconfig.PluginConfigType)
		if pluginType != "" {
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
		if pluginType != "" {
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
		if pluginType != "" {
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
	}

	return plugins, err
}

func (nm *namespaceManager) initPlugins(ctx context.Context, cancelCtx context.CancelFunc) (err error) {
	for _, entry := range nm.plugins.database {
		if err = entry.plugin.Init(ctx, entry.config); err != nil {
			return err
		}
		entry.plugin.SetHandler(database.GlobalHandler, nm)
	}
	for _, entry := range nm.plugins.blockchain {
		if err = entry.plugin.Init(ctx, cancelCtx, entry.config, nm.metrics, nm.cacheManager); err != nil {
			return err
		}
	}
	for _, entry := range nm.plugins.dataexchange {
		if err = entry.plugin.Init(ctx, cancelCtx, entry.config); err != nil {
			return err
		}
	}
	for _, entry := range nm.plugins.sharedstorage {
		if err = entry.plugin.Init(ctx, entry.config); err != nil {
			return err
		}
	}
	for name, entry := range nm.plugins.tokens {
		if err = entry.plugin.Init(ctx, cancelCtx, name, entry.config); err != nil {
			return err
		}
	}
	for _, entry := range nm.plugins.events {
		if err = entry.plugin.Init(ctx, entry.config); err != nil {
			return err
		}
	}
	for name, entry := range nm.plugins.auth {
		if err = entry.plugin.Init(ctx, name, entry.config); err != nil {
			return err
		}
	}
	return nil
}

func (nm *namespaceManager) loadNamespaces(ctx context.Context) (err error) {
	defaultName := config.GetString(coreconfig.NamespacesDefault)
	size := namespacePredefined.ArraySize()
	foundDefault := false
	for i := 0; i < size; i++ {
		nsConfig := namespacePredefined.ArrayEntry(i)
		name := nsConfig.GetString(coreconfig.NamespaceName)
		if name == "" {
			log.L(ctx).Warnf("Skipping unnamed entry at namespaces.predefined[%d]", i)
			continue
		}
		if _, ok := nm.namespaces[name]; ok {
			log.L(ctx).Warnf("Duplicate predefined namespace (ignored): %s", name)
			continue
		}
		foundDefault = foundDefault || name == defaultName
		if nm.namespaces[name], err = nm.loadNamespace(ctx, name, i, nsConfig); err != nil {
			return err
		}
	}
	if !foundDefault {
		return i18n.NewError(ctx, coremsgs.MsgDefaultNamespaceNotFound, defaultName)
	}
	return err
}

func (nm *namespaceManager) loadNamespace(ctx context.Context, name string, index int, conf config.Section) (*namespace, error) {
	if err := fftypes.ValidateFFNameField(ctx, name, fmt.Sprintf("namespaces.predefined[%d].name", index)); err != nil {
		return nil, err
	}
	if name == core.LegacySystemNamespace {
		return nil, i18n.NewError(ctx, coremsgs.MsgFFSystemReservedName, core.LegacySystemNamespace)
	}

	keyNormalization := conf.GetString(coreconfig.NamespaceAssetKeyNormalization)
	if keyNormalization == "" {
		keyNormalization = config.GetString(coreconfig.AssetManagerKeyNormalization)
	}

	multipartyConf := conf.SubSection(coreconfig.NamespaceMultiparty)
	// If any multiparty org information is configured (here or at the root), assume multiparty mode by default
	orgName := multipartyConf.GetString(coreconfig.NamespaceMultipartyOrgName)
	orgKey := multipartyConf.GetString(coreconfig.NamespaceMultipartyOrgKey)
	orgDesc := multipartyConf.GetString(coreconfig.NamespaceMultipartyOrgDescription)
	nodeName := multipartyConf.GetString(coreconfig.NamespaceMultipartyNodeName)
	nodeDesc := multipartyConf.GetString(coreconfig.NamespaceMultipartyNodeDescription)
	deprecatedOrgName := config.GetString(coreconfig.OrgName)
	deprecatedOrgKey := config.GetString(coreconfig.OrgKey)
	deprecatedOrgDesc := config.GetString(coreconfig.OrgDescription)
	deprecatedNodeName := config.GetString(coreconfig.NodeName)
	deprecatedNodeDesc := config.GetString(coreconfig.NodeDescription)
	if deprecatedOrgName != "" || deprecatedOrgKey != "" || deprecatedOrgDesc != "" {
		log.L(ctx).Warnf("Your org config uses a deprecated configuration structure - the org configuration has been moved under the 'namespaces.predefined[].multiparty' section")
	}
	if deprecatedNodeName != "" || deprecatedNodeDesc != "" {
		log.L(ctx).Warnf("Your node config uses a deprecated configuration structure - the node configuration has been moved under the 'namespaces.predefined[].multiparty' section")
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
	if nodeName == "" {
		nodeName = deprecatedNodeName
	}
	if nodeDesc == "" {
		nodeDesc = deprecatedNodeDesc
	}

	multipartyEnabled := multipartyConf.Get(coreconfig.NamespaceMultipartyEnabled)
	if multipartyEnabled == nil {
		multipartyEnabled = orgName != "" || orgKey != ""
	}

	networkName := name
	if multipartyEnabled.(bool) {
		mpNetworkName := multipartyConf.GetString(coreconfig.NamespaceMultipartyNetworkNamespace)
		if mpNetworkName == core.LegacySystemNamespace {
			return nil, i18n.NewError(ctx, coremsgs.MsgFFSystemReservedName, core.LegacySystemNamespace)
		}
		if mpNetworkName != "" {
			networkName = mpNetworkName
		}
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
		DefaultKey:          conf.GetString(coreconfig.NamespaceDefaultKey),
		TokenBroadcastNames: nm.tokenBroadcastNames,
		KeyNormalization:    keyNormalization,
	}
	if multipartyEnabled.(bool) {
		contractsConf := multipartyConf.SubArray(coreconfig.NamespaceMultipartyContract)
		contractConfArraySize := contractsConf.ArraySize()
		contracts := make([]multiparty.Contract, contractConfArraySize)

		for i := 0; i < contractConfArraySize; i++ {
			conf := contractsConf.ArrayEntry(i)
			locationObject := conf.GetObject(coreconfig.NamespaceMultipartyContractLocation)
			b, err := json.Marshal(locationObject)
			if err != nil {
				return nil, err
			}
			location := fftypes.JSONAnyPtrBytes(b)
			contract := multiparty.Contract{
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
		config.Multiparty.Node.Name = nodeName
		config.Multiparty.Node.Description = nodeDesc
	}

	return &namespace{
		Namespace: core.Namespace{
			Name:        name,
			NetworkName: networkName,
			Description: conf.GetString(coreconfig.NamespaceDescription),
		},
		config:  config,
		plugins: plugins,
	}, nil
}

func (nm *namespaceManager) validatePlugins(ctx context.Context, name string, plugins []string) (*orchestrator.Plugins, error) {
	var result orchestrator.Plugins
	for _, pluginName := range plugins {
		if instance, ok := nm.plugins.blockchain[pluginName]; ok {
			if result.Blockchain.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceMultiplePluginType, name, "blockchain")
			}
			result.Blockchain = orchestrator.BlockchainPlugin{
				Name:   pluginName,
				Plugin: instance.plugin,
			}
			continue
		}
		if instance, ok := nm.plugins.dataexchange[pluginName]; ok {
			if result.DataExchange.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceMultiplePluginType, name, "dataexchange")
			}
			result.DataExchange = orchestrator.DataExchangePlugin{
				Name:   pluginName,
				Plugin: instance.plugin,
			}
			continue
		}
		if instance, ok := nm.plugins.sharedstorage[pluginName]; ok {
			if result.SharedStorage.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceMultiplePluginType, name, "sharedstorage")
			}
			result.SharedStorage = orchestrator.SharedStoragePlugin{
				Name:   pluginName,
				Plugin: instance.plugin,
			}
			continue
		}
		if instance, ok := nm.plugins.database[pluginName]; ok {
			if result.Database.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceMultiplePluginType, name, "database")
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
		if instance, ok := nm.plugins.auth[pluginName]; ok {
			result.Auth = orchestrator.AuthPlugin{
				Name:   pluginName,
				Plugin: instance.plugin,
			}
			continue
		}

		return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceUnknownPlugin, name, pluginName)
	}
	return &result, nil
}

func (nm *namespaceManager) validateMultiPartyConfig(ctx context.Context, name string, plugins []string) (*orchestrator.Plugins, error) {

	result, err := nm.validatePlugins(ctx, name, plugins)
	if err != nil {
		return nil, err
	}

	if result.Database.Plugin == nil ||
		result.SharedStorage.Plugin == nil ||
		result.DataExchange.Plugin == nil ||
		result.Blockchain.Plugin == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceWrongPluginsMultiparty, name)
	}

	result.Events = make(map[string]events.Plugin, len(nm.plugins.events))
	for name, entry := range nm.plugins.events {
		result.Events[name] = entry.plugin
	}
	return result, nil
}

func (nm *namespaceManager) validateNonMultipartyConfig(ctx context.Context, name string, plugins []string) (*orchestrator.Plugins, error) {

	result, err := nm.validatePlugins(ctx, name, plugins)
	if err != nil {
		return nil, err
	}

	if result.Database.Plugin == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceNoDatabase, name)
	}

	result.Events = make(map[string]events.Plugin, len(nm.plugins.events))
	for name, entry := range nm.plugins.events {
		result.Events[name] = entry.plugin
	}
	return result, nil
}

func (nm *namespaceManager) SPIEvents() spievents.Manager {
	return nm.adminEvents
}

func (nm *namespaceManager) Orchestrator(ns string) orchestrator.Orchestrator {
	nm.nsMux.Lock()
	defer nm.nsMux.Unlock()
	if namespace, ok := nm.namespaces[ns]; ok {
		return namespace.orchestrator
	}
	return nil
}

func (nm *namespaceManager) GetNamespaces(ctx context.Context) ([]*core.Namespace, error) {
	nm.nsMux.Lock()
	defer nm.nsMux.Unlock()
	results := make([]*core.Namespace, 0, len(nm.namespaces))
	for _, ns := range nm.namespaces {
		results = append(results, &ns.Namespace)
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

func (nm *namespaceManager) getAuthPlugin(ctx context.Context) (plugins map[string]authPlugin, err error) {
	plugins = make(map[string]authPlugin)

	authConfigArraySize := authConfig.ArraySize()
	for i := 0; i < authConfigArraySize; i++ {
		config := authConfig.ArrayEntry(i)
		name, pluginType, err := nm.validatePluginConfig(ctx, config, "auth")
		if err != nil {
			return nil, err
		}

		plugin, err := authfactory.GetPlugin(ctx, pluginType)
		if err != nil {
			return nil, err
		}

		plugins[name] = authPlugin{
			config: config.SubSection(pluginType),
			plugin: plugin,
		}
	}
	return plugins, err
}

func (nm *namespaceManager) Authorize(ctx context.Context, authReq *fftypes.AuthReq) error {
	return nm.Orchestrator(authReq.Namespace).Authorize(ctx, authReq)
}
