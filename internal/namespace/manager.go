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
	Init(ctx context.Context, cancelCtx context.CancelFunc) error
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
	name         string
	remoteName   string
	description  string
	orchestrator orchestrator.Orchestrator
	config       orchestrator.Config
	plugins      []string
}

type namespaceManager struct {
	ctx         context.Context
	cancelCtx   context.CancelFunc
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
	metricsEnabled   bool
	metrics          metrics.Manager
	adminEvents      spievents.Manager
	utOrchestrator   orchestrator.Orchestrator
	tokenRemoteNames map[string]string
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
		namespaces:       make(map[string]*namespace),
		metricsEnabled:   config.GetBool(coreconfig.MetricsEnabled),
		tokenRemoteNames: make(map[string]string),
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

	if nm.metricsEnabled {
		// Ensure metrics are registered
		metrics.Registry()
	}

	var v1Namespace *namespace
	for _, ns := range nm.namespaces {
		if err := nm.initNamespace(ns); err != nil {
			return err
		}
		multiparty := ns.config.Multiparty.Enabled
		version := "n/a"
		if multiparty {
			version = fmt.Sprintf("%d", ns.orchestrator.MultiParty().GetNetworkVersion())
		}
		log.L(ctx).Infof("Initialized namespace '%s' multiparty=%s version=%s", ns.name, strconv.FormatBool(multiparty), version)
		if multiparty && nm.checkForV1Contract(ctx, ns) {
			// TODO:
			// We should check the contract address too, AND should check previously-terminated contracts
			// in addition to the active contract. That implies:
			// - we must cache the address and version of each contract on the namespace in the database
			// - when the orchestrator loads that info from the database (triggered from Orchestrator.Init),
			//   it needs to somehow be available here
			//
			// In short, if any namespace was EVER pointed at a V1 contract, that contract and that namespace's plugins
			// become the de facto configuration for ff_system as well. There can only be one V1 contract in the history
			// of a given FireFly node, because it's impossible to re-create ff_system against a different contract
			// or different set of plugins.

			//TODO: still need to understand why we should check the contract address
			if v1Namespace == nil {
				v1Namespace = ns
			} else if !stringSlicesEqual(v1Namespace.plugins, ns.plugins) {
				// TODO: localize error
				return fmt.Errorf("could not initialize legacy '%s' namespace - found conflicting V1 multi-party config in %s and %s",
					core.LegacySystemNamespace, v1Namespace.name, ns.name)
			}
		}
	}

	// If any namespace is a multiparty V1 namespace, insert the legacy ff_system namespace.
	// Note that the contract address and plugin list must match for ALL V1 namespaces.
	if v1Namespace != nil {
		systemNS := *v1Namespace
		systemNS.name = core.LegacySystemNamespace
		systemNS.remoteName = core.LegacySystemNamespace
		log.L(ctx).Infof("Initializing legacy '%s' namespace as a copy of %s", core.LegacySystemNamespace, v1Namespace.name)
		err = nm.initNamespace(&systemNS)
		nm.namespaces[core.LegacySystemNamespace] = &systemNS
	}
	return err
}

func (nm *namespaceManager) checkForV1Contract(ctx context.Context, ns *namespace) bool {
	if ns.orchestrator.MultiParty().GetNetworkVersion() == 1 {
		return true
	}

	// check previously terminated contracts to see if they were ever V1
	stored := ns.orchestrator.GetNamespace(ctx)
	for _, contract := range stored.Contracts.Terminated {
		if contract.Version == 1 {
			return true
		}
	}

	return false
}

func (nm *namespaceManager) initNamespace(ns *namespace) (err error) {
	var plugins *orchestrator.Plugins
	if ns.config.Multiparty.Enabled {
		plugins, err = nm.validateMultiPartyConfig(nm.ctx, ns.name, ns.plugins)
	} else {
		plugins, err = nm.validateNonMultipartyConfig(nm.ctx, ns.name, ns.plugins)
	}
	if err != nil {
		return err
	}

	stored, err := plugins.Database.Plugin.GetNamespace(nm.ctx, ns.name)
	switch {
	case err != nil:
		return err
	case stored != nil:
		stored.RemoteName = ns.remoteName
		stored.Description = ns.description
		// TODO: should we check for discrepancies in the multiparty contract config?
	default:
		stored = &core.Namespace{
			LocalName:   ns.name,
			RemoteName:  ns.remoteName,
			Description: ns.description,
			Created:     fftypes.Now(),
		}
	}
	if err = plugins.Database.Plugin.UpsertNamespace(nm.ctx, stored, true); err != nil {
		return err
	}

	or := nm.utOrchestrator
	if or == nil {
		or = orchestrator.NewOrchestrator(stored, ns.config, plugins, nm.metrics)
	}
	orCtx, orCancel := context.WithCancel(nm.ctx)
	if err := or.Init(orCtx, orCancel); err != nil {
		return err
	}
	go func() {
		<-orCtx.Done()
		nm.nsMux.Lock()
		defer nm.nsMux.Unlock()
		delete(nm.namespaces, ns.name)
	}()
	ns.orchestrator = or
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
	// Restart the current context to pick up the configuration change
	go func() {
		<-ctx.Done()
		nm.cancelCtx()
	}()
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
	// Remote names must be unique
	remoteNames := make(map[string]bool)

	tokensConfigArraySize := tokensConfig.ArraySize()
	for i := 0; i < tokensConfigArraySize; i++ {
		config := tokensConfig.ArrayEntry(i)
		name, pluginType, err := nm.validatePluginConfig(ctx, config, "tokens")
		if err != nil {
			return nil, err
		}
		remoteName := config.GetString(coreconfig.PluginRemoteName)
		// If there is no remote name, use the plugin name
		if remoteName == "" {
			remoteName = name
		}
		if _, exists := remoteNames[remoteName]; exists {
			return nil, i18n.NewError(ctx, coremsgs.MsgDuplicatePluginRemoteName, "tokens", remoteName)
		}
		remoteNames[remoteName] = true
		nm.tokenRemoteNames[name] = remoteName

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
	for name, entry := range nm.plugins.auth {
		if err = entry.plugin.Init(nm.ctx, name, entry.config); err != nil {
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
	remoteName := conf.GetString(coreconfig.NamespaceRemoteName)
	if name == core.LegacySystemNamespace || remoteName == core.LegacySystemNamespace {
		return nil, i18n.NewError(ctx, coremsgs.MsgFFSystemReservedName, core.LegacySystemNamespace)
	}
	if remoteName == "" {
		remoteName = name
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
	multipartyEnabled := multipartyConf.Get(coreconfig.NamespaceMultipartyEnabled)
	if multipartyEnabled == nil {
		multipartyEnabled = orgName != "" || orgKey != ""
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
		DefaultKey:       conf.GetString(coreconfig.NamespaceDefaultKey),
		TokenRemoteNames: nm.tokenRemoteNames,
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
	}

	return &namespace{
		name:        name,
		remoteName:  remoteName,
		description: conf.GetString(coreconfig.NamespaceDescription),
		config:      config,
		plugins:     plugins,
	}, nil
}

func (nm *namespaceManager) validateMultiPartyConfig(ctx context.Context, name string, plugins []string) (*orchestrator.Plugins, error) {
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
	return &result, nil
}

func (nm *namespaceManager) validateNonMultipartyConfig(ctx context.Context, name string, plugins []string) (*orchestrator.Plugins, error) {
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
		if _, ok := nm.plugins.dataexchange[pluginName]; ok {
			return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceWrongPluginsNonMultiparty, name)
		}
		if _, ok := nm.plugins.sharedstorage[pluginName]; ok {
			return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceWrongPluginsNonMultiparty, name)
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
		if instance, ok := nm.plugins.auth[pluginName]; ok {
			result.Auth = orchestrator.AuthPlugin{
				Name:   pluginName,
				Plugin: instance.plugin,
			}
			continue
		}

		return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceUnknownPlugin, name, pluginName)
	}

	if result.Database.Plugin == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgNamespaceNoDatabase, name)
	}

	result.Events = make(map[string]events.Plugin, len(nm.plugins.events))
	for name, entry := range nm.plugins.events {
		result.Events[name] = entry.plugin
	}
	return &result, nil
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
		results = append(results, ns.orchestrator.GetNamespace(ctx))
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
