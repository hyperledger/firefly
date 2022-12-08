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
	"time"

	"github.com/fsnotify/fsnotify"
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
	"github.com/spf13/viper"
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
	Reset(ctx context.Context) error

	Orchestrator(ns string) orchestrator.Orchestrator
	SPIEvents() spievents.Manager
	GetNamespaces(ctx context.Context) ([]*core.Namespace, error)
	GetOperationByNamespacedID(ctx context.Context, nsOpID string) (*core.Operation, error)
	ResolveOperationByNamespacedID(ctx context.Context, nsOpID string, op *core.OperationUpdateDTO) error
	Authorize(ctx context.Context, authReq *fftypes.AuthReq) error
}

type namespace struct {
	core.Namespace
	ctx          context.Context
	cancelCtx    context.CancelFunc
	orchestrator orchestrator.Orchestrator
	loadTime     fftypes.FFTime
	config       orchestrator.Config
	configHash   *fftypes.Bytes32
	pluginNames  []string
	plugins      *orchestrator.Plugins
}

type namespaceManager struct {
	reset               chan bool
	ctx                 context.Context
	cancelCtx           context.CancelFunc
	nsMux               sync.Mutex
	namespaces          map[string]*namespace
	plugins             map[string]*plugin
	metricsEnabled      bool
	cacheManager        cache.Manager
	metrics             metrics.Manager
	adminEvents         spievents.Manager
	utOrchestrator      orchestrator.Orchestrator
	tokenBroadcastNames map[string]string
	watchConfig         func() // indirect from viper.WatchConfig for testing
}

type pluginCategory string

const (
	pluginCategoryBlockchain    pluginCategory = "blockchain"
	pluginCategoryDatabase      pluginCategory = "database"
	pluginCategoryDataexchange  pluginCategory = "dataexchange"
	pluginCategorySharedstorage pluginCategory = "sharedstorage"
	pluginCategoryTokens        pluginCategory = "tokens"
	pluginCategoryIdentity      pluginCategory = "identity"
	pluginCategoryEvents        pluginCategory = "events"
	pluginCategoryAuth          pluginCategory = "auth"
)

type plugin struct {
	name       string
	category   pluginCategory
	pluginType string
	ctx        context.Context
	cancelCtx  context.CancelFunc
	config     config.Section
	configHash *fftypes.Bytes32
	loadTime   time.Time

	blockchain    blockchain.Plugin
	database      database.Plugin
	dataexchange  dataexchange.Plugin
	sharedstorage sharedstorage.Plugin
	tokens        tokens.Plugin
	identity      identity.Plugin
	events        events.Plugin
	auth          auth.Plugin
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

func NewNamespaceManager() Manager {
	nm := &namespaceManager{
		namespaces:          make(map[string]*namespace),
		metricsEnabled:      config.GetBool(coreconfig.MetricsEnabled),
		tokenBroadcastNames: make(map[string]string),
		watchConfig:         viper.WatchConfig,
	}
	initAllConfig()
	return nm
}

func initAllConfig() {

	InitConfig()

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
}

func (nm *namespaceManager) startConfigListener(ctx context.Context) {
	go func() {
		for {
			log.L(ctx).Warnf("Starting configuration listener")
			// Note there is no viper interface to make this end, so (apart from in unit tests)
			// we never expect this to complete.
			// To avoid this leak, we disable the use of the /spi/v1/reset API when config file
			// listening is enabled.
			viper.OnConfigChange(nm.newConfigChangeListener(ctx))
			nm.watchConfig()
			select {
			case <-ctx.Done():
				log.L(ctx).Debugf("Configuration listener ended")
				return
			default:
			}
			log.L(ctx).Warnf("Configuration listener ended (restarting)")
			time.Sleep(5 * time.Second)
		}
	}()
}

func (nm *namespaceManager) newConfigChangeListener(ctx context.Context) func(in fsnotify.Event) {
	return func(in fsnotify.Event) {
		nm.configFileChanged(ctx, in.Name)
	}
}

func (nm *namespaceManager) dumpRootConfig() (jsonTree fftypes.JSONObject) {
	viperTree := viper.AllSettings()
	b, _ := json.Marshal(viperTree)
	_ = json.Unmarshal(b, &jsonTree)
	return
}

func (nm *namespaceManager) configFileChanged(ctx context.Context, filename string) {
	log.L(ctx).Infof("Detected configuration file reload: '%s'", filename)

	// Get Viper to dump the whole new config, with everything resolved across env vars
	// and the config file etc.
	// We use this to detect if anything has changed.
	rawConfig := nm.dumpRootConfig()

	newPlugins, err := nm.loadPlugins(ctx, rawConfig)
	if err != nil {
		log.L(ctx).Errorf("Failed to initialize plugins after config reload: %s", err)
		return
	}

	allNewNamespaces, err := nm.loadNamespaces(ctx, rawConfig, newPlugins)
	if err != nil {
		log.L(ctx).Errorf("Failed to load namespaces after config reload: %s", err)
		return
	}

	// From this point we need to block any API calls resolving namespaces, until the reload is complete
	nm.nsMux.Lock()
	defer nm.nsMux.Unlock()

	// Stop all defunct namespaces
	updatedNamespaces, err := nm.stopDefunctNamespaces(ctx, newPlugins, allNewNamespaces)
	if err != nil {
		log.L(ctx).Errorf("Failed to stop namespaces after config reload: %s", err)
		nm.cancelCtx() // stop the world
		return
	}

	// Stop all defunct plugins - now the namespaces using them are all stopped
	updatedPlugins, err := nm.stopDefunctPlugins(ctx, newPlugins)
	if err != nil {
		log.L(ctx).Errorf("Failed to stop namespaces after config reload: %s", err)
		nm.cancelCtx() // stop the world
		return
	}

	// Update the new lists
	nm.plugins = newPlugins
	nm.namespaces = allNewNamespaces

	// Only initialize updated plugins
	if err = nm.initPlugins(ctx, updatedPlugins); err != nil {
		log.L(ctx).Errorf("Failed to initialize plugins after config reload: %s", err)
		nm.cancelCtx() // stop the world
		return
	}

	// Only initialize the updated namespaces (which includes all that depend on above plugins)
	if err = nm.initNamespaces(ctx, updatedNamespaces); err != nil {
		log.L(ctx).Errorf("Failed to initialize namespaces after config reload: %s", err)
		nm.cancelCtx() // stop the world
		return
	}

	// Now finally we can start all the new things
	if err = nm.startNamespacesAndPlugins(updatedNamespaces, updatedPlugins); err != nil {
		log.L(ctx).Errorf("Failed to initialize namespaces after config reload: %s", err)
		nm.cancelCtx() // stop the world
		return
	}

}

func (nm *namespaceManager) stopDefunctNamespaces(ctx context.Context, newPlugins map[string]*plugin, newNamespaces map[string]*namespace) (map[string]*namespace, error) {

	// build a set of all the namespaces we've either added new, or have changed
	updatedNamespaces := make(map[string]*namespace)
	namespacesToStop := make(map[string]*namespace)
	for nsName, newNS := range newNamespaces {
		if existingNS := nm.namespaces[nsName]; existingNS != nil {
			unchanged := existingNS.configHash.Equals(newNS.configHash) &&
				len(existingNS.pluginNames) == len(newNS.pluginNames)
			for _, pluginName := range newNS.pluginNames {
				existingPlugin := nm.plugins[pluginName]
				newPlugin := newPlugins[pluginName]
				unchanged = existingPlugin != nil && newPlugin != nil &&
					existingPlugin.configHash.Equals(newPlugin.configHash)
			}
			if unchanged {
				log.L(ctx).Debugf("Namespace '%s' unchanged after config reload", nsName)
				continue
			}
			// We need to stop the existing namespace
			namespacesToStop[nsName] = existingNS
		}
		// This is either changed, or brand new - mark it in the map
		updatedNamespaces[nsName] = newNS
	}

	// Stop everything we need to stop
	for nsName, existingNS := range nm.namespaces {
		if namespacesToStop[nsName] != nil || newNamespaces[nsName] == nil {
			log.L(ctx).Debugf("Stopping namespace '%s' after config reload. Loaded at %s", nsName, existingNS.loadTime)
			nm.stopNamespace(ctx, existingNS)
		}
	}

	return updatedNamespaces, nil

}

func (nm *namespaceManager) stopDefunctPlugins(ctx context.Context, newPlugins map[string]*plugin) (map[string]*plugin, error) {

	// build a set of all the plugins we've either added new, or have changed
	updatedPlugins := make(map[string]*plugin)
	pluginsToStop := make(map[string]*plugin)
	for pluginName, newPlugin := range newPlugins {
		if existingPlugin := nm.plugins[pluginName]; existingPlugin != nil {
			if existingPlugin.configHash.Equals(newPlugin.configHash) {
				log.L(ctx).Debugf("Plugin '%s' unchanged after config reload", pluginName)
				continue
			}
			// We need to stop the existing plugin
			pluginsToStop[pluginName] = existingPlugin
		}
		// This is either changed, or brand new - mark it in the map
		updatedPlugins[pluginName] = newPlugin
	}

	// Stop everything we need to stop
	for pluginName, existingPlugin := range nm.plugins {
		if pluginsToStop[pluginName] != nil || newPlugins[pluginName] == nil {
			log.L(ctx).Debugf("Stopping plugin '%s' after config reload. Loaded at %s", pluginName, existingPlugin.loadTime)
			existingPlugin.cancelCtx()
		}
	}

	return updatedPlugins, nil
}

func (nm *namespaceManager) Init(ctx context.Context, cancelCtx context.CancelFunc, reset chan bool) (err error) {
	nm.reset = reset
	nm.cancelCtx = cancelCtx

	initTimeRawConfig := nm.dumpRootConfig()
	nm.loadManagers(ctx)
	if nm.plugins, err = nm.loadPlugins(ctx, initTimeRawConfig); err != nil {
		return err
	}
	if nm.namespaces, err = nm.loadNamespaces(ctx, initTimeRawConfig, nm.plugins); err != nil {
		return err
	}

	return nm.initComponents(ctx)
}

func (nm *namespaceManager) initComponents(ctx context.Context) (err error) {
	// Initialize all the plugins on initial startup
	if err = nm.initPlugins(ctx, nm.plugins); err != nil {
		return err
	}

	// Initialize all the namespaces on initial startup
	if err = nm.initNamespaces(ctx, nm.namespaces); err != nil {
		return err
	}

	if nm.metricsEnabled {
		// Ensure metrics are registered
		metrics.Registry()
	}

	if config.GetBool(coreconfig.ConfigAutoReload) {
		nm.startConfigListener(ctx)
	}

	return nil
}

func (nm *namespaceManager) initNamespaces(ctx context.Context, newNamespaces map[string]*namespace) error {
	// In network version 1, the blockchain plugin and multiparty contract were global and singular.
	// Therefore, if any namespace was EVER pointed at a V1 contract, that contract and that namespace's plugins
	// become the de facto configuration for ff_system as well. There can only be one V1 contract in the history
	// of a given FireFly node, because it's impossible to re-create ff_system against a different contract
	// or different set of plugins.
	var v1Namespace *namespace
	var v1Contract *core.MultipartyContract

	for _, ns := range newNamespaces {
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
				} else if !stringSlicesEqual(v1Namespace.pluginNames, ns.pluginNames) ||
					v1Contract.Location.String() != contract.Location.String() ||
					v1Contract.FirstEvent != contract.FirstEvent {
					return i18n.NewError(ctx, coremsgs.MsgCannotInitLegacyNS, core.LegacySystemNamespace, v1Namespace.Name, ns.Name)
				}
			}
		}
	}

	if v1Namespace != nil {
		systemNS := &namespace{
			Namespace:   v1Namespace.Namespace,
			config:      v1Namespace.config,
			pluginNames: v1Namespace.pluginNames,
			plugins:     v1Namespace.plugins,
			configHash:  v1Namespace.configHash,
		}
		systemNS.Name = core.LegacySystemNamespace
		systemNS.NetworkName = core.LegacySystemNamespace
		newNamespaces[core.LegacySystemNamespace] = systemNS
		if err := nm.initNamespace(ctx, systemNS); err != nil {
			return err
		}
		log.L(ctx).Infof("Initialized namespace '%s' as a copy of '%s'", core.LegacySystemNamespace, v1Namespace.Name)
	}

	return nil
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

	database := ns.plugins.Database.Plugin
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
		or = orchestrator.NewOrchestrator(&ns.Namespace, ns.config, ns.plugins, nm.metrics, nm.cacheManager)
	}
	ns.orchestrator = or
	ns.ctx, ns.cancelCtx = context.WithCancel(ctx)
	if err := or.Init(ns.ctx, ns.cancelCtx); err != nil {
		return err
	}
	return nil
}

func (nm *namespaceManager) stopNamespace(ctx context.Context, ns *namespace) {
	if ns.cancelCtx != nil {
		log.L(ctx).Infof("Requesting stop of namespace '%s'", ns.Name)
		ns.cancelCtx()
		ns.orchestrator.WaitStop()
		log.L(ctx).Infof("Namespace '%s' stopped", ns.Name)
	}
}

func (nm *namespaceManager) Start() error {
	// On initial start, we need to start everything
	return nm.startNamespacesAndPlugins(nm.namespaces, nm.plugins)
}

func (nm *namespaceManager) startNamespacesAndPlugins(namespacesToStart map[string]*namespace, pluginsToStart map[string]*plugin) error {
	// Orchestrators must be started before plugins so as not to miss events
	for _, ns := range namespacesToStart {
		if err := ns.orchestrator.Start(); err != nil {
			return err
		}
	}
	for _, plugin := range pluginsToStart {
		switch plugin.category {
		case pluginCategoryBlockchain:
			if err := plugin.blockchain.Start(); err != nil {
				return err
			}
		case pluginCategoryDataexchange:
			if err := plugin.dataexchange.Start(); err != nil {
				return err
			}
		case pluginCategoryTokens:
			if err := plugin.tokens.Start(); err != nil {
				return err
			}
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
		nm.stopNamespace(nm.ctx, ns)
	}
	nm.adminEvents.WaitStop()
}

func (nm *namespaceManager) Reset(ctx context.Context) error {
	if config.GetBool(coreconfig.ConfigAutoReload) {
		// We do not allow these settings to be combined, because viper does not provide a way to
		// stop the file listener on the old root Viper instance (before reset). So we would
		// leak file listeners in the background.
		return i18n.NewError(context.Background(), coremsgs.MsgDeprecatedResetWithAutoReload)
	}

	// Queue a restart of the root context to pick up a configuration change.
	// Caller is responsible for terminating the passed context to trigger the actual reset
	// (allows caller to cleanly finish processing the current request/event).
	go func() {
		<-ctx.Done()
		nm.reset <- true
	}()

	return nil
}

func (nm *namespaceManager) loadManagers(ctx context.Context) {
	if nm.metrics == nil {
		nm.metrics = metrics.NewMetricsManager(ctx)
	}

	if nm.cacheManager == nil {
		nm.cacheManager = cache.NewCacheManager(ctx)
	}

	if nm.adminEvents == nil {
		nm.adminEvents = spievents.NewAdminEventManager(ctx)
	}
}

func (nm *namespaceManager) loadPlugins(ctx context.Context, rawConfig fftypes.JSONObject) (newPlugins map[string]*plugin, err error) {

	newPlugins = make(map[string]*plugin)

	if err := nm.getDatabasePlugins(ctx, newPlugins, rawConfig); err != nil {
		return nil, err
	}

	if err := nm.getIdentityPlugins(ctx, newPlugins, rawConfig); err != nil {
		return nil, err
	}

	if err := nm.getBlockchainPlugins(ctx, newPlugins, rawConfig); err != nil {
		return nil, err
	}

	if err := nm.getSharedStoragePlugins(ctx, newPlugins, rawConfig); err != nil {
		return nil, err
	}

	if err := nm.getDataExchangePlugins(ctx, newPlugins, rawConfig); err != nil {
		return nil, err
	}

	if err := nm.getTokensPlugins(ctx, newPlugins, rawConfig); err != nil {
		return nil, err
	}

	if err := nm.getEventPlugins(ctx, newPlugins, rawConfig); err != nil {
		return nil, err
	}

	if err := nm.getAuthPlugin(ctx, newPlugins, rawConfig); err != nil {
		return nil, err
	}

	return newPlugins, nil
}

func (nm *namespaceManager) configHash(rawConfigObject fftypes.JSONObject) *fftypes.Bytes32 {
	b, _ := json.Marshal(rawConfigObject)
	return fftypes.HashString(string(b))
}

func (nm *namespaceManager) getTokensPlugins(ctx context.Context, plugins map[string]*plugin, rawConfig fftypes.JSONObject) (err error) {
	// Broadcast names must be unique
	broadcastNames := make(map[string]bool)

	tokensConfigArraySize := tokensConfig.ArraySize()
	rawPluginTokensConfig := rawConfig.GetObject("plugins").GetObjectArray("tokens")
	if len(rawPluginTokensConfig) != tokensConfigArraySize {
		log.L(ctx).Errorf("Expected len(%d) for plugins.tokens: %s", tokensConfigArraySize, rawPluginTokensConfig)
		return i18n.NewError(ctx, coremsgs.MsgConfigArrayVsRawConfigMismatch)
	}
	for i := 0; i < tokensConfigArraySize; i++ {
		config := tokensConfig.ArrayEntry(i)
		configHash := nm.configHash(rawPluginTokensConfig[i])
		pc, err := nm.validatePluginConfig(ctx, plugins, pluginCategoryTokens, config, configHash)
		if err != nil {
			return err
		}
		broadcastName := config.GetString(coreconfig.PluginBroadcastName)
		// If there is no broadcast name, use the plugin name
		if broadcastName == "" {
			broadcastName = pc.name
		}
		if _, exists := broadcastNames[broadcastName]; exists {
			return i18n.NewError(ctx, coremsgs.MsgDuplicatePluginBroadcastName, pluginCategoryTokens, broadcastName)
		}
		broadcastNames[broadcastName] = true
		nm.tokenBroadcastNames[pc.name] = broadcastName

		pc.tokens, err = tifactory.GetPlugin(ctx, pc.pluginType)
		if err != nil {
			return err
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
				return i18n.NewError(ctx, coremsgs.MsgMissingTokensPluginConfig)
			}
			if err = fftypes.ValidateFFNameField(ctx, name, "name"); err != nil {
				return err
			}
			nm.tokenBroadcastNames[name] = name

			configHash := nm.configHash(rawConfig.GetObject("plugins").GetObject("tokens"))
			pc, err := nm.newPluginCommon(ctx, plugins, pluginCategoryTokens, name, pluginType, deprecatedConfig, configHash)
			if err != nil {
				return err
			}

			pc.tokens, err = tifactory.GetPlugin(ctx, pluginType)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (nm *namespaceManager) getDatabasePlugins(ctx context.Context, plugins map[string]*plugin, rawConfig fftypes.JSONObject) (err error) {
	dbConfigArraySize := databaseConfig.ArraySize()
	rawPluginDatabaseConfig := rawConfig.GetObject("plugins").GetObjectArray("database")
	if len(rawPluginDatabaseConfig) != dbConfigArraySize {
		log.L(ctx).Errorf("Expected len(%d) for plugins.database: %s", dbConfigArraySize, rawPluginDatabaseConfig)
		return i18n.NewError(ctx, coremsgs.MsgConfigArrayVsRawConfigMismatch)
	}
	for i := 0; i < dbConfigArraySize; i++ {
		config := databaseConfig.ArrayEntry(i)
		configHash := nm.configHash(rawPluginDatabaseConfig[i])
		pc, err := nm.validatePluginConfig(ctx, plugins, pluginCategoryDatabase, config, configHash)
		if err != nil {
			return err
		}

		pc.database, err = difactory.GetPlugin(ctx, pc.pluginType)
		if err != nil {
			return err
		}
	}

	// check for deprecated config
	if len(plugins) == 0 {
		pluginType := deprecatedDatabaseConfig.GetString(coreconfig.PluginConfigType)
		if pluginType != "" {
			log.L(ctx).Warnf("Your database config uses a deprecated configuration structure - the database configuration has been moved under the 'plugins' section")
			configHash := nm.configHash(rawConfig.GetObject("plugins").GetObject("database"))
			pc, err := nm.newPluginCommon(ctx, plugins, pluginCategoryDatabase, "database_0", pluginType, deprecatedDatabaseConfig, configHash)
			if err != nil {
				return err
			}
			pc.database, err = difactory.GetPlugin(ctx, pluginType)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (nm *namespaceManager) validatePluginConfig(ctx context.Context, plugins map[string]*plugin, category pluginCategory, config config.Section, configHash *fftypes.Bytes32) (*plugin, error) {
	name := config.GetString(coreconfig.PluginConfigName)
	pluginType := config.GetString(coreconfig.PluginConfigType)

	if name == "" || pluginType == "" {
		return nil, i18n.NewError(ctx, coremsgs.MsgInvalidPluginConfiguration, category)
	}

	if err := fftypes.ValidateFFNameField(ctx, name, "name"); err != nil {
		return nil, err
	}

	return nm.newPluginCommon(ctx, plugins, category, name, pluginType, config, configHash)
}

func (nm *namespaceManager) newPluginCommon(ctx context.Context, plugins map[string]*plugin, category pluginCategory, name, pluginType string, config config.Section, configHash *fftypes.Bytes32) (*plugin, error) {
	if _, ok := plugins[name]; ok {
		return nil, i18n.NewError(ctx, coremsgs.MsgDuplicatePluginName, name)
	}

	pc := &plugin{
		name:       name,
		category:   category,
		pluginType: pluginType,
		config:     config,
		configHash: configHash,
		loadTime:   time.Now(),
	}
	plugins[name] = pc
	// context is always inherited from namespaceManager BG context _not_ the context of the caller
	pc.ctx, pc.cancelCtx = context.WithCancel(nm.ctx)
	return pc, nil
}

func (nm *namespaceManager) getDataExchangePlugins(ctx context.Context, plugins map[string]*plugin, rawConfig fftypes.JSONObject) (err error) {
	dxConfigArraySize := dataexchangeConfig.ArraySize()
	rawPluginDXConfig := rawConfig.GetObject("plugins").GetObjectArray("dataexchange")
	if len(rawPluginDXConfig) != dxConfigArraySize {
		log.L(ctx).Errorf("Expected len(%d) for plugins.dataexchange: %s", dxConfigArraySize, rawPluginDXConfig)
		return i18n.NewError(ctx, coremsgs.MsgConfigArrayVsRawConfigMismatch)
	}
	for i := 0; i < dxConfigArraySize; i++ {
		config := dataexchangeConfig.ArrayEntry(i)
		configHash := nm.configHash(rawPluginDXConfig[i])
		pc, err := nm.validatePluginConfig(ctx, plugins, pluginCategoryDataexchange, config, configHash)
		if err != nil {
			return err
		}

		pc.dataexchange, err = dxfactory.GetPlugin(ctx, pc.pluginType)
		if err != nil {
			return err
		}
	}

	// check deprecated config
	if len(plugins) == 0 {
		pluginType := deprecatedDataexchangeConfig.GetString(coreconfig.PluginConfigType)
		if pluginType != "" {
			log.L(ctx).Warnf("Your data exchange config uses a deprecated configuration structure - the data exchange configuration has been moved under the 'plugins' section")
			configHash := nm.configHash(rawConfig.GetObject("plugins").GetObject("dataexchange"))
			pc, err := nm.newPluginCommon(ctx, plugins, pluginCategoryDataexchange, "dataexchange_0", pluginType, deprecatedDataexchangeConfig, configHash)
			if err != nil {
				return err
			}
			pc.dataexchange, err = dxfactory.GetPlugin(ctx, pluginType)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (nm *namespaceManager) getIdentityPlugins(ctx context.Context, plugins map[string]*plugin, rawConfig fftypes.JSONObject) (err error) {
	configSize := identityConfig.ArraySize()
	rawPluginIdentityConfig := rawConfig.GetObject("plugins").GetObjectArray("identity")
	if len(rawPluginIdentityConfig) != configSize {
		log.L(ctx).Errorf("Expected len(%d) for plugins.identity: %s", configSize, rawPluginIdentityConfig)
		return i18n.NewError(ctx, coremsgs.MsgConfigArrayVsRawConfigMismatch)
	}
	for i := 0; i < configSize; i++ {
		config := identityConfig.ArrayEntry(i)
		configHash := nm.configHash(rawPluginIdentityConfig[i])
		pc, err := nm.validatePluginConfig(ctx, plugins, pluginCategoryIdentity, config, configHash)
		if err != nil {
			return err
		}

		pc.identity, err = iifactory.GetPlugin(ctx, pc.pluginType)
		if err != nil {
			return err
		}
	}

	return nil
}

func (nm *namespaceManager) getBlockchainPlugins(ctx context.Context, plugins map[string]*plugin, rawConfig fftypes.JSONObject) (err error) {
	blockchainConfigArraySize := blockchainConfig.ArraySize()
	rawPluginBlockchainsConfig := rawConfig.GetObject("plugins").GetObjectArray("blockchain")
	if len(rawPluginBlockchainsConfig) != blockchainConfigArraySize {
		log.L(ctx).Errorf("Expected len(%d) for plugins.blockchain: %s", blockchainConfigArraySize, rawPluginBlockchainsConfig)
		return i18n.NewError(ctx, coremsgs.MsgConfigArrayVsRawConfigMismatch)
	}
	for i := 0; i < blockchainConfigArraySize; i++ {
		config := blockchainConfig.ArrayEntry(i)
		configHash := nm.configHash(rawPluginBlockchainsConfig[i])
		pc, err := nm.validatePluginConfig(ctx, plugins, pluginCategoryBlockchain, config, configHash)
		if err != nil {
			return err
		}

		pc.blockchain, err = bifactory.GetPlugin(ctx, pc.pluginType)
		if err != nil {
			return err
		}
	}

	// check deprecated config
	if len(plugins) == 0 {
		pluginType := deprecatedBlockchainConfig.GetString(coreconfig.PluginConfigType)
		if pluginType != "" {
			log.L(ctx).Warnf("Your blockchain config uses a deprecated configuration structure - the blockchain configuration has been moved under the 'plugins' section")

			configHash := nm.configHash(rawConfig.GetObject("plugins").GetObject("blockchain"))
			pc, err := nm.newPluginCommon(ctx, plugins, pluginCategoryBlockchain, "blockchain_0", pluginType, deprecatedBlockchainConfig, configHash)
			if err != nil {
				return err
			}
			pc.blockchain, err = bifactory.GetPlugin(ctx, pluginType)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (nm *namespaceManager) getSharedStoragePlugins(ctx context.Context, plugins map[string]*plugin, rawConfig fftypes.JSONObject) (err error) {
	configSize := sharedstorageConfig.ArraySize()
	rawPluginSharedStorageConfig := rawConfig.GetObject("plugins").GetObjectArray("sharedstorage")
	if len(rawPluginSharedStorageConfig) != configSize {
		log.L(ctx).Errorf("Expected len(%d) for plugins.sharedstorage: %s", configSize, rawPluginSharedStorageConfig)
		return i18n.NewError(ctx, coremsgs.MsgConfigArrayVsRawConfigMismatch)
	}
	for i := 0; i < configSize; i++ {
		config := sharedstorageConfig.ArrayEntry(i)
		configHash := nm.configHash(rawPluginSharedStorageConfig[i])
		pc, err := nm.validatePluginConfig(ctx, plugins, pluginCategorySharedstorage, config, configHash)
		if err != nil {
			return err
		}

		pc.sharedstorage, err = ssfactory.GetPlugin(ctx, pc.pluginType)
		if err != nil {
			return err
		}
	}

	// check deprecated config
	if len(plugins) == 0 {
		pluginType := deprecatedSharedStorageConfig.GetString(coreconfig.PluginConfigType)
		if pluginType != "" {
			log.L(ctx).Warnf("Your shared storage config uses a deprecated configuration structure - the shared storage configuration has been moved under the 'plugins' section")

			configHash := nm.configHash(rawConfig.GetObject("plugins").GetObject("sharedstorage"))
			pc, err := nm.newPluginCommon(ctx, plugins, pluginCategorySharedstorage, "sharedstorage_0", pluginType, deprecatedBlockchainConfig, configHash)
			if err != nil {
				return err
			}
			pc.sharedstorage, err = ssfactory.GetPlugin(ctx, pluginType)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (nm *namespaceManager) initPlugins(ctx context.Context, pluginsToStart map[string]*plugin) (err error) {
	for name, p := range nm.plugins {
		if pluginsToStart[name] == nil {
			continue
		}
		switch p.category {
		case pluginCategoryDatabase:
			if err = p.database.Init(p.ctx, p.config); err != nil {
				return err
			}
			p.database.SetHandler(database.GlobalHandler, nm)
		case pluginCategoryBlockchain:
			if err = p.blockchain.Init(p.ctx, nm.cancelCtx /* allow plugin to stop whole process */, p.config, nm.metrics, nm.cacheManager); err != nil {
				return err
			}
		case pluginCategoryDataexchange:
			if err = p.dataexchange.Init(p.ctx, nm.cancelCtx /* allow plugin to stop whole process */, p.config); err != nil {
				return err
			}
		case pluginCategorySharedstorage:
			if err = p.sharedstorage.Init(p.ctx, p.config); err != nil {
				return err
			}
		case pluginCategoryTokens:
			if err = p.tokens.Init(p.ctx, nm.cancelCtx /* allow plugin to stop whole process */, name, p.config); err != nil {
				return err
			}
		case pluginCategoryEvents:
			if err = p.events.Init(p.ctx, p.config); err != nil {
				return err
			}
		case pluginCategoryAuth:
			if err = p.auth.Init(p.ctx, name, p.config); err != nil {
				return err
			}
		}
	}
	return nil
}

func (nm *namespaceManager) loadNamespaces(ctx context.Context, rawConfig fftypes.JSONObject, availablePlugins map[string]*plugin) (newNS map[string]*namespace, err error) {
	defaultName := config.GetString(coreconfig.NamespacesDefault)
	size := namespacePredefined.ArraySize()
	rawPredefinedNSConfig := rawConfig.GetObject("namespaces").GetObjectArray("predefined")
	if len(rawPredefinedNSConfig) != size {
		log.L(ctx).Errorf("Expected len(%d) for namespaces.predefined: %s", size, rawPredefinedNSConfig)
		return nil, i18n.NewError(ctx, coremsgs.MsgConfigArrayVsRawConfigMismatch)
	}
	foundDefault := false

	newNS = make(map[string]*namespace)
	for i := 0; i < size; i++ {
		nsConfig := namespacePredefined.ArrayEntry(i)
		name := nsConfig.GetString(coreconfig.NamespaceName)
		if name == "" {
			log.L(ctx).Warnf("Skipping unnamed entry at namespaces.predefined[%d]", i)
			continue
		}
		if _, ok := newNS[name]; ok {
			log.L(ctx).Warnf("Duplicate predefined namespace (ignored): %s", name)
			continue
		}
		foundDefault = foundDefault || name == defaultName
		if newNS[name], err = nm.loadNamespace(ctx, name, i, nsConfig, rawPredefinedNSConfig[i], availablePlugins); err != nil {
			return nil, err
		}
	}
	// We allow startup with zero namespaces defined, so that we can have a FF Core
	// ready to accept config updates to add new namespaces.
	if !foundDefault && size > 0 {
		return nil, i18n.NewError(ctx, coremsgs.MsgDefaultNamespaceNotFound, defaultName)
	}
	return newNS, err
}

func (nm *namespaceManager) loadNamespace(ctx context.Context, name string, index int, conf config.Section, rawNSConfig fftypes.JSONObject, availablePlugins map[string]*plugin) (ns *namespace, err error) {
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
	pluginNames := conf.GetStringSlice(coreconfig.NamespacePlugins)
	if pluginsRaw == nil {
		for pluginName := range nm.plugins {
			p := availablePlugins[pluginName]
			switch p.category {
			case pluginCategoryBlockchain,
				pluginCategoryDatabase,
				pluginCategoryDataexchange,
				pluginCategoryIdentity,
				pluginCategorySharedstorage,
				pluginCategoryTokens,
				pluginCategoryAuth:
				pluginNames = append(pluginNames, pluginName)
			}
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
			location := fftypes.JSONAnyPtr(conf.GetObject(coreconfig.NamespaceMultipartyContractLocation).String())
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

	ns = &namespace{
		Namespace: core.Namespace{
			Name:        name,
			NetworkName: networkName,
			Description: conf.GetString(coreconfig.NamespaceDescription),
		},
		config:      config,
		configHash:  nm.configHash(rawNSConfig),
		pluginNames: pluginNames,
	}

	if ns.plugins, err = nm.validateNSPlugins(ctx, ns, availablePlugins); err != nil {
		return nil, err
	}

	if ns.config.Multiparty.Enabled {
		err = nm.validateMultiPartyConfig(ctx, ns)
	} else {
		err = nm.validateNonMultipartyConfig(ctx, ns)
	}
	if err != nil {
		return nil, err
	}

	ns.plugins.Events = make(map[string]events.Plugin)
	for name, p := range nm.plugins {
		if p.category == pluginCategoryEvents {
			ns.plugins.Events[name] = p.events
		}
	}

	return ns, nil
}

func (nm *namespaceManager) validateNSPlugins(ctx context.Context, ns *namespace, availablePlugins map[string]*plugin) (*orchestrator.Plugins, error) {
	var result orchestrator.Plugins
	for _, pluginName := range ns.pluginNames {
		p := availablePlugins[pluginName]
		if p == nil {
			return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceUnknownPlugin, ns.Name, pluginName)
		}
		switch p.category {
		case pluginCategoryBlockchain:
			if result.Blockchain.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceMultiplePluginType, ns.Name, "blockchain")
			}
			result.Blockchain = orchestrator.BlockchainPlugin{
				Name:   pluginName,
				Plugin: p.blockchain,
			}
		case pluginCategoryDataexchange:
			if result.DataExchange.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceMultiplePluginType, ns.Name, "dataexchange")
			}
			result.DataExchange = orchestrator.DataExchangePlugin{
				Name:   pluginName,
				Plugin: p.dataexchange,
			}
		case pluginCategorySharedstorage:
			if result.SharedStorage.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceMultiplePluginType, ns.Name, "sharedstorage")
			}
			result.SharedStorage = orchestrator.SharedStoragePlugin{
				Name:   pluginName,
				Plugin: p.sharedstorage,
			}
		case pluginCategoryDatabase:
			if result.Database.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceMultiplePluginType, ns.Name, "database")
			}
			result.Database = orchestrator.DatabasePlugin{
				Name:   pluginName,
				Plugin: p.database,
			}
		case pluginCategoryTokens:
			result.Tokens = append(result.Tokens, orchestrator.TokensPlugin{
				Name:   pluginName,
				Plugin: p.tokens,
			})
		case pluginCategoryIdentity:
			if result.Identity.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceMultiplePluginType, ns.Name, "identity")
			}
			result.Identity = orchestrator.IdentityPlugin{
				Name:   pluginName,
				Plugin: p.identity,
			}
		case pluginCategoryAuth:
			if result.Auth.Plugin != nil {
				return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceMultiplePluginType, ns.Name, "auth")
			}
			result.Auth = orchestrator.AuthPlugin{
				Name:   pluginName,
				Plugin: p.auth,
			}
		default:
			return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceUnknownPlugin, ns.Name, pluginName)
		}
	}
	return &result, nil
}

func (nm *namespaceManager) validateMultiPartyConfig(ctx context.Context, ns *namespace) error {

	if ns.plugins.Database.Plugin == nil ||
		ns.plugins.SharedStorage.Plugin == nil ||
		ns.plugins.DataExchange.Plugin == nil ||
		ns.plugins.Blockchain.Plugin == nil {
		return i18n.NewError(ctx, coremsgs.MsgNamespaceWrongPluginsMultiparty, ns.Name)
	}

	return nil
}

func (nm *namespaceManager) validateNonMultipartyConfig(ctx context.Context, ns *namespace) error {

	if ns.plugins.Database.Plugin == nil {
		return i18n.NewError(ctx, coremsgs.MsgNamespaceNoDatabase, ns.Name)
	}

	return nil
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

func (nm *namespaceManager) getEventPlugins(ctx context.Context, plugins map[string]*plugin, rawConfig fftypes.JSONObject) (err error) {
	enabledTransports := config.GetStringSlice(coreconfig.EventTransportsEnabled)
	uniqueTransports := make(map[string]bool)
	for _, transport := range enabledTransports {
		uniqueTransports[transport] = true
	}
	// Cannot disable the internal listener
	uniqueTransports[system.SystemEventsTransport] = true
	for transport := range uniqueTransports {

		eventsPlugin, err := eifactory.GetPlugin(ctx, transport)
		if err != nil {
			return err
		}
		name := eventsPlugin.Name()
		section := config.RootSection("events").SubSection(name)
		rawEventConfig := rawConfig.GetObject("events").GetObject(name)

		pc, err := nm.newPluginCommon(ctx, plugins, pluginCategoryEvents, name, name /* name is category for events */, section, nm.configHash(rawEventConfig))
		if err != nil {
			return err
		}
		pc.events = eventsPlugin
		pc.events.InitConfig(section)
	}
	return nil
}

func (nm *namespaceManager) getAuthPlugin(ctx context.Context, plugins map[string]*plugin, rawConfig fftypes.JSONObject) (err error) {
	authConfigArraySize := authConfig.ArraySize()
	rawPluginAuthConfig := rawConfig.GetObject("plugins").GetObjectArray("auth")
	if len(rawPluginAuthConfig) != authConfigArraySize {
		log.L(ctx).Errorf("Expected len(%d) for plugins.auth: %s", authConfigArraySize, rawPluginAuthConfig)
		return i18n.NewError(ctx, coremsgs.MsgConfigArrayVsRawConfigMismatch)
	}
	for i := 0; i < authConfigArraySize; i++ {
		config := authConfig.ArrayEntry(i)
		configHash := nm.configHash(rawPluginAuthConfig[i])
		pc, err := nm.validatePluginConfig(ctx, plugins, pluginCategoryAuth, config, configHash)
		if err != nil {
			return err
		}

		pc.auth, err = authfactory.GetPlugin(ctx, pc.pluginType)
		if err != nil {
			return err
		}
	}
	return nil
}

func (nm *namespaceManager) Authorize(ctx context.Context, authReq *fftypes.AuthReq) error {
	return nm.Orchestrator(authReq.Namespace).Authorize(ctx, authReq)
}
