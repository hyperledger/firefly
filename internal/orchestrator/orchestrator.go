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

package orchestrator

import (
	"context"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/adminevents"
	"github.com/hyperledger/firefly/internal/assets"
	"github.com/hyperledger/firefly/internal/batch"
	"github.com/hyperledger/firefly/internal/batchpin"
	"github.com/hyperledger/firefly/internal/blockchain/bifactory"
	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/contracts"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/database/difactory"
	"github.com/hyperledger/firefly/internal/dataexchange/dxfactory"
	"github.com/hyperledger/firefly/internal/definitions"
	"github.com/hyperledger/firefly/internal/events"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/identity/iifactory"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/namespace"
	"github.com/hyperledger/firefly/internal/networkmap"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/internal/privatemessaging"
	"github.com/hyperledger/firefly/internal/shareddownload"
	"github.com/hyperledger/firefly/internal/sharedstorage/ssfactory"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/internal/tokens/tifactory"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	idplugin "github.com/hyperledger/firefly/pkg/identity"
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

// Orchestrator is the main interface behind the API, implementing the actions
type Orchestrator interface {
	Init(ctx context.Context, cancelCtx context.CancelFunc) error
	Start() error
	WaitStop() // The close itself is performed by canceling the context
	AdminEvents() adminevents.Manager
	Assets() assets.Manager
	BatchManager() batch.Manager
	Broadcast() broadcast.Manager
	Contracts() contracts.Manager
	Data() data.Manager
	Events() events.EventManager
	Metrics() metrics.Manager
	NetworkMap() networkmap.Manager
	Operations() operations.Manager
	PrivateMessaging() privatemessaging.Manager

	// Status
	GetStatus(ctx context.Context) (*core.NodeStatus, error)

	// Subscription management
	GetSubscriptions(ctx context.Context, ns string, filter database.AndFilter) ([]*core.Subscription, *database.FilterResult, error)
	GetSubscriptionByID(ctx context.Context, ns, id string) (*core.Subscription, error)
	CreateSubscription(ctx context.Context, ns string, subDef *core.Subscription) (*core.Subscription, error)
	CreateUpdateSubscription(ctx context.Context, ns string, subDef *core.Subscription) (*core.Subscription, error)
	DeleteSubscription(ctx context.Context, ns, id string) error

	// Data Query
	GetNamespace(ctx context.Context, ns string) (*core.Namespace, error)
	GetNamespaces(ctx context.Context, filter database.AndFilter) ([]*core.Namespace, *database.FilterResult, error)
	GetTransactionByID(ctx context.Context, ns, id string) (*core.Transaction, error)
	GetTransactionOperations(ctx context.Context, ns, id string) ([]*core.Operation, *database.FilterResult, error)
	GetTransactionBlockchainEvents(ctx context.Context, ns, id string) ([]*core.BlockchainEvent, *database.FilterResult, error)
	GetTransactionStatus(ctx context.Context, ns, id string) (*core.TransactionStatus, error)
	GetTransactions(ctx context.Context, ns string, filter database.AndFilter) ([]*core.Transaction, *database.FilterResult, error)
	GetMessageByID(ctx context.Context, ns, id string) (*core.Message, error)
	GetMessageByIDWithData(ctx context.Context, ns, id string) (*core.MessageInOut, error)
	GetMessages(ctx context.Context, ns string, filter database.AndFilter) ([]*core.Message, *database.FilterResult, error)
	GetMessagesWithData(ctx context.Context, ns string, filter database.AndFilter) ([]*core.MessageInOut, *database.FilterResult, error)
	GetMessageTransaction(ctx context.Context, ns, id string) (*core.Transaction, error)
	GetMessageOperations(ctx context.Context, ns, id string) ([]*core.Operation, *database.FilterResult, error)
	GetMessageEvents(ctx context.Context, ns, id string, filter database.AndFilter) ([]*core.Event, *database.FilterResult, error)
	GetMessageData(ctx context.Context, ns, id string) (core.DataArray, error)
	GetMessagesForData(ctx context.Context, ns, dataID string, filter database.AndFilter) ([]*core.Message, *database.FilterResult, error)
	GetBatchByID(ctx context.Context, ns, id string) (*core.BatchPersisted, error)
	GetBatches(ctx context.Context, ns string, filter database.AndFilter) ([]*core.BatchPersisted, *database.FilterResult, error)
	GetDataByID(ctx context.Context, ns, id string) (*core.Data, error)
	GetData(ctx context.Context, ns string, filter database.AndFilter) (core.DataArray, *database.FilterResult, error)
	GetDatatypeByID(ctx context.Context, ns, id string) (*core.Datatype, error)
	GetDatatypeByName(ctx context.Context, ns, name, version string) (*core.Datatype, error)
	GetDatatypes(ctx context.Context, ns string, filter database.AndFilter) ([]*core.Datatype, *database.FilterResult, error)
	GetOperationByIDNamespaced(ctx context.Context, ns, id string) (*core.Operation, error)
	GetOperationsNamespaced(ctx context.Context, ns string, filter database.AndFilter) ([]*core.Operation, *database.FilterResult, error)
	GetOperationByID(ctx context.Context, id string) (*core.Operation, error)
	GetOperations(ctx context.Context, filter database.AndFilter) ([]*core.Operation, *database.FilterResult, error)
	GetEventByID(ctx context.Context, ns, id string) (*core.Event, error)
	GetEvents(ctx context.Context, ns string, filter database.AndFilter) ([]*core.Event, *database.FilterResult, error)
	GetEventsWithReferences(ctx context.Context, ns string, filter database.AndFilter) ([]*core.EnrichedEvent, *database.FilterResult, error)
	GetBlockchainEventByID(ctx context.Context, ns, id string) (*core.BlockchainEvent, error)
	GetBlockchainEvents(ctx context.Context, ns string, filter database.AndFilter) ([]*core.BlockchainEvent, *database.FilterResult, error)
	GetPins(ctx context.Context, filter database.AndFilter) ([]*core.Pin, *database.FilterResult, error)

	// Charts
	GetChartHistogram(ctx context.Context, ns string, startTime int64, endTime int64, buckets int64, tableName database.CollectionName) ([]*core.ChartHistogram, error)

	// Message Routing
	RequestReply(ctx context.Context, ns string, msg *core.MessageInOut) (reply *core.MessageInOut, err error)

	// Network Operations
	MigrateNetwork(ctx context.Context) error
}

type orchestrator struct {
	ctx                  context.Context
	cancelCtx            context.CancelFunc
	started              bool
	database             database.Plugin
	databases            map[string]database.Plugin
	blockchain           blockchain.Plugin
	blockchains          map[string]blockchain.Plugin
	identity             identity.Manager
	identityPlugins      map[string]idplugin.Plugin
	sharedstorage        sharedstorage.Plugin
	sharedstoragePlugins map[string]sharedstorage.Plugin
	dataexchange         dataexchange.Plugin
	dataexchangePlugins  map[string]dataexchange.Plugin
	events               events.EventManager
	networkmap           networkmap.Manager
	batch                batch.Manager
	broadcast            broadcast.Manager
	messaging            privatemessaging.Manager
	definitions          definitions.DefinitionHandler
	data                 data.Manager
	syncasync            syncasync.Bridge
	batchpin             batchpin.Submitter
	assets               assets.Manager
	tokens               map[string]tokens.Plugin
	bc                   boundCallbacks
	contracts            contracts.Manager
	node                 *fftypes.UUID
	metrics              metrics.Manager
	operations           operations.Manager
	adminEvents          adminevents.Manager
	sharedDownload       shareddownload.Manager
	txHelper             txcommon.Helper
	namespace            namespace.Manager
}

func NewOrchestrator(withDefaults bool) Orchestrator {
	or := &orchestrator{}

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
	namespace.InitConfig(withDefaults)

	return or
}

func (or *orchestrator) Init(ctx context.Context, cancelCtx context.CancelFunc) (err error) {
	or.ctx = ctx
	or.cancelCtx = cancelCtx
	err = or.initPlugins(ctx)
	if err == nil {
		err = or.initNamespaces(ctx)
	}
	if err == nil {
		err = or.initComponents(ctx)
	}
	// Bind together the blockchain interface callbacks, with the events manager
	or.bc.bi = or.blockchain
	or.bc.ei = or.events
	or.bc.dx = or.dataexchange
	or.bc.ss = or.sharedstorage
	or.bc.om = or.operations
	return err
}

func (or *orchestrator) Start() (err error) {
	if err == nil {
		err = or.batch.Start()
	}
	var ns *core.Namespace
	if err == nil {
		ns, err = or.database.GetNamespace(or.ctx, core.SystemNamespace)
	}
	if err == nil {
		for _, el := range or.blockchains {
			if err = el.ConfigureContract(&ns.Contracts); err != nil {
				break
			}
			if err = el.Start(); err != nil {
				break
			}
		}
		if err == nil {
			err = or.database.UpsertNamespace(or.ctx, ns, true)
		}
	}
	if err == nil {
		err = or.events.Start()
	}
	if err == nil {
		err = or.broadcast.Start()
	}
	if err == nil {
		err = or.messaging.Start()
	}
	if err == nil {
		err = or.operations.Start()
	}
	if err == nil {
		err = or.sharedDownload.Start()
	}
	if err == nil {
		for _, el := range or.tokens {
			if err = el.Start(); err != nil {
				break
			}
		}
	}
	if err == nil {
		err = or.metrics.Start()
	}
	or.started = true
	return err
}

func (or *orchestrator) WaitStop() {
	if !or.started {
		return
	}
	if or.batch != nil {
		or.batch.WaitStop()
		or.batch = nil
	}
	if or.broadcast != nil {
		or.broadcast.WaitStop()
		or.broadcast = nil
	}
	if or.data != nil {
		or.data.WaitStop()
		or.data = nil
	}
	if or.sharedDownload != nil {
		or.sharedDownload.WaitStop()
		or.sharedDownload = nil
	}
	if or.operations != nil {
		or.operations.WaitStop()
		or.operations = nil
	}
	if or.adminEvents != nil {
		or.adminEvents.WaitStop()
		or.adminEvents = nil
	}
	or.started = false
}

func (or *orchestrator) Broadcast() broadcast.Manager {
	return or.broadcast
}

func (or *orchestrator) PrivateMessaging() privatemessaging.Manager {
	return or.messaging
}

func (or *orchestrator) Events() events.EventManager {
	return or.events
}

func (or *orchestrator) BatchManager() batch.Manager {
	return or.batch
}

func (or *orchestrator) NetworkMap() networkmap.Manager {
	return or.networkmap
}

func (or *orchestrator) Data() data.Manager {
	return or.data
}

func (or *orchestrator) Assets() assets.Manager {
	return or.assets
}

func (or *orchestrator) Contracts() contracts.Manager {
	return or.contracts
}

func (or *orchestrator) Metrics() metrics.Manager {
	return or.metrics
}

func (or *orchestrator) Operations() operations.Manager {
	return or.operations
}

func (or *orchestrator) AdminEvents() adminevents.Manager {
	return or.adminEvents
}

func (or *orchestrator) getDatabasePlugins(ctx context.Context) (plugins []database.Plugin, err error) {
	dbConfigArraySize := databaseConfig.ArraySize()
	plugins = make([]database.Plugin, dbConfigArraySize)
	for i := 0; i < dbConfigArraySize; i++ {
		config := databaseConfig.ArrayEntry(i)
		if err = or.validatePluginConfig(ctx, config, "database"); err != nil {
			return nil, err
		}
		plugins[i], err = difactory.GetPlugin(ctx, config.GetString(coreconfig.PluginConfigType))
		if err != nil {
			return nil, err
		}
	}

	return plugins, err
}

func (or *orchestrator) initDatabasePlugins(ctx context.Context, plugins []database.Plugin) (err error) {
	for idx, plugin := range plugins {
		config := databaseConfig.ArrayEntry(idx)
		err = plugin.Init(ctx, config.SubSection(config.GetString(coreconfig.PluginConfigType)), or)
		if err != nil {
			return err
		}
		name := config.GetString(coreconfig.PluginConfigName)
		or.databases[name] = plugin

		if or.database == nil {
			or.database = plugin
		}
	}

	return err
}

func (or *orchestrator) validatePluginConfig(ctx context.Context, config config.Section, sectionName string) error {
	name := config.GetString(coreconfig.PluginConfigName)
	dxType := config.GetString(coreconfig.PluginConfigType)

	if name == "" || dxType == "" {
		return i18n.NewError(ctx, coremsgs.MsgInvalidPluginConfiguration, sectionName)
	}

	if err := core.ValidateFFNameField(ctx, name, "name"); err != nil {
		return err
	}

	return nil
}

func (or *orchestrator) initDataExchange(ctx context.Context) (err error) {
	or.dataexchangePlugins = make(map[string]dataexchange.Plugin)
	dxConfigArraySize := dataexchangeConfig.ArraySize()
	plugins := make([]dataexchange.Plugin, dxConfigArraySize)
	for i := 0; i < dxConfigArraySize; i++ {
		config := dataexchangeConfig.ArrayEntry(i)
		if err = or.validatePluginConfig(ctx, config, "dataexchange"); err != nil {
			return err
		}
		plugins[i], err = dxfactory.GetPlugin(ctx, config.GetString(coreconfig.PluginConfigType))
		if err != nil {
			return err
		}
	}

	fb := database.IdentityQueryFactory.NewFilter(ctx)
	nodes, _, err := or.database.GetIdentities(ctx, fb.And(
		fb.Eq("type", core.IdentityTypeNode),
		fb.Eq("namespace", core.SystemNamespace),
	))
	if err != nil {
		return err
	}
	nodeInfo := make([]fftypes.JSONObject, len(nodes))
	for i, node := range nodes {
		nodeInfo[i] = node.Profile
	}

	if len(plugins) > 0 {
		for idx, plugin := range plugins {
			config := dataexchangeConfig.ArrayEntry(idx)
			err = plugin.Init(ctx, config.SubSection(config.GetString(coreconfig.PluginConfigType)), nodeInfo, &or.bc)
			if err != nil {
				return err
			}
			name := config.GetString(coreconfig.PluginConfigName)
			or.dataexchangePlugins[name] = plugin
			if or.dataexchange == nil {
				or.dataexchange = plugin
			}
		}
	} else {
		log.L(ctx).Warnf("Your data exchange config uses a deprecated configuration structure - the data exchange configuration has been moved under the 'plugins' section")
		dxType := deprecatedDataexchangeConfig.GetString(coreconfig.PluginConfigType)
		plugin, err := dxfactory.GetPlugin(ctx, dxType)
		if err != nil {
			return err
		}

		config := deprecatedDataexchangeConfig.SubSection(dxType)
		err = plugin.Init(ctx, config, nodeInfo, &or.bc)
		if err != nil {
			return err
		}
		or.dataexchangePlugins["dataexchange_0"] = plugin
		or.dataexchange = plugin
	}

	return err
}

func (or *orchestrator) initPlugins(ctx context.Context) (err error) {
	if or.metrics == nil {
		or.metrics = metrics.NewMetricsManager(ctx)
	}

	if or.databases == nil {
		or.databases = make(map[string]database.Plugin)
		dp, err := or.getDatabasePlugins(ctx)
		if err != nil {
			return err
		}
		err = or.initDatabasePlugins(ctx, dp)
		if err != nil {
			return err
		}
	}

	// check for deprecated db config
	if len(or.databases) == 0 {
		diType := deprecatedDatabaseConfig.GetString(coreconfig.PluginConfigType)
		plugin, err := difactory.GetPlugin(ctx, diType)
		if err != nil {
			return err
		}
		err = or.initDeprecatedDatabasePlugin(ctx, plugin)
		if err != nil {
			return err
		}
	}

	// Not really a plugin, but this has to be initialized here after the database (at least temporarily).
	// Shortly after this step, namespaces will be synced to the database and will generate notifications to adminEvents.
	if or.adminEvents == nil {
		or.adminEvents = adminevents.NewAdminEventManager(ctx)
	}

	if or.identityPlugins == nil {
		if err = or.initIdentity(ctx); err != nil {
			return err
		}
	}

	if or.blockchains == nil {
		or.blockchains = make(map[string]blockchain.Plugin)
		bp, err := or.getBlockchainPlugins(ctx)
		if err != nil {
			return err
		}
		err = or.initBlockchainPlugins(ctx, bp)
		if err != nil {
			return err
		}
	}

	// Check for deprecated blockchain config
	if len(or.blockchains) == 0 {
		biType := deprecatedBlockchainConfig.GetString(coreconfig.PluginConfigType)
		plugin, err := bifactory.GetPlugin(ctx, biType)
		if err != nil {
			return err
		}
		err = or.initDeprecatedBlockchainPlugin(ctx, plugin)
		if err != nil {
			return err
		}
	}

	if or.sharedstoragePlugins == nil {
		or.sharedstoragePlugins = make(map[string]sharedstorage.Plugin)
		ss, err := or.getSharedStoragePlugins(ctx)
		if err != nil {
			return err
		}

		if err = or.initSharedStoragePlugins(ctx, ss); err != nil {
			return err
		}
	}

	// Check for deprecated shared storage config
	if len(or.sharedstoragePlugins) == 0 {
		ssType := deprecatedSharedStorageConfig.GetString(coreconfig.PluginConfigType)
		plugin, err := ssfactory.GetPlugin(ctx, ssType)
		if err != nil {
			return err
		}

		if err = or.initDeprecatedSharedStoragePlugin(ctx, plugin); err != nil {
			return err
		}
	}

	if or.dataexchangePlugins == nil {
		if err = or.initDataExchange(ctx); err != nil {
			return err
		}
	}

	if or.tokens == nil {
		if err = or.initTokens(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (or *orchestrator) initSharedStoragePlugins(ctx context.Context, plugins []sharedstorage.Plugin) (err error) {
	for idx, plugin := range plugins {
		config := sharedstorageConfig.ArrayEntry(idx)
		err = plugin.Init(ctx, config.SubSection(config.GetString(coreconfig.PluginConfigType)), &or.bc)
		if err != nil {
			return err
		}
		name := config.GetString(coreconfig.PluginConfigName)
		or.sharedstoragePlugins[name] = plugin

		if or.sharedstorage == nil {
			or.sharedstorage = plugin
		}
	}

	return err
}

func (or *orchestrator) initDeprecatedSharedStoragePlugin(ctx context.Context, plugin sharedstorage.Plugin) (err error) {
	log.L(ctx).Warnf("Your shared storage config uses a deprecated configuration structure - the shared storage configuration has been moved under the 'plugins' section")
	err = plugin.Init(ctx, deprecatedSharedStorageConfig.SubSection(plugin.Name()), &or.bc)
	if err != nil {
		return err
	}

	or.sharedstoragePlugins["sharedstorage_0"] = plugin
	or.sharedstorage = plugin
	return err
}

func (or *orchestrator) getSharedStoragePlugins(ctx context.Context) (plugins []sharedstorage.Plugin, err error) {
	configSize := sharedstorageConfig.ArraySize()
	plugins = make([]sharedstorage.Plugin, configSize)
	for i := 0; i < configSize; i++ {
		config := sharedstorageConfig.ArrayEntry(i)
		if err = or.validatePluginConfig(ctx, config, "sharedstorage"); err != nil {
			return nil, err
		}
		plugins[i], err = ssfactory.GetPlugin(ctx, config.GetString(coreconfig.PluginConfigType))
		if err != nil {
			return nil, err
		}
	}

	return plugins, err
}

func (or *orchestrator) initIdentity(ctx context.Context) (err error) {
	or.identityPlugins = make(map[string]idplugin.Plugin)
	plugins, err := or.getIdentityPlugins(ctx)
	if err != nil {
		return err
	}
	// this is a no-op currently, inits the tbd plugin
	_ = or.initIdentityPlugins(ctx, plugins)

	return err
}

func (or *orchestrator) initIdentityPlugins(ctx context.Context, plugins []idplugin.Plugin) (err error) {
	for idx, plugin := range plugins {
		config := identityConfig.ArrayEntry(idx)
		err = plugin.Init(ctx, config.SubSection(config.GetString(coreconfig.PluginConfigType)), &or.bc)
		if err != nil {
			return err
		}
		name := config.GetString(coreconfig.PluginConfigName)
		or.identityPlugins[name] = plugin
	}

	return err
}

func (or *orchestrator) getIdentityPlugins(ctx context.Context) (plugins []idplugin.Plugin, err error) {
	configSize := identityConfig.ArraySize()
	plugins = make([]idplugin.Plugin, configSize)
	for i := 0; i < configSize; i++ {
		config := identityConfig.ArrayEntry(i)
		if err = or.validatePluginConfig(ctx, config, "identity"); err != nil {
			return nil, err
		}

		plugins[i], err = iifactory.GetPlugin(ctx, config.GetString(coreconfig.PluginConfigType))
		if err != nil {
			return nil, err
		}
	}
	return plugins, err
}

func (or *orchestrator) getBlockchainPlugins(ctx context.Context) (plugins []blockchain.Plugin, err error) {
	blockchainConfigArraySize := blockchainConfig.ArraySize()
	plugins = make([]blockchain.Plugin, blockchainConfigArraySize)
	for i := 0; i < blockchainConfigArraySize; i++ {
		config := blockchainConfig.ArrayEntry(i)
		if err = or.validatePluginConfig(ctx, config, "blockchain"); err != nil {
			return nil, err
		}

		plugins[i], err = bifactory.GetPlugin(ctx, config.GetString(coreconfig.PluginConfigType))
		if err != nil {
			return nil, err
		}
	}

	return plugins, err
}

func (or *orchestrator) initDeprecatedBlockchainPlugin(ctx context.Context, plugin blockchain.Plugin) (err error) {
	log.L(ctx).Warnf("Your blockchain config uses a deprecated configuration structure - the blockchain configuration has been moved under the 'plugins' section")
	err = plugin.Init(ctx, deprecatedBlockchainConfig.SubSection(plugin.Name()), &or.bc, or.metrics)
	if err != nil {
		return err
	}

	deprecatedPluginName := "blockchain_0"
	or.blockchains[deprecatedPluginName] = plugin
	or.blockchain = plugin
	return err
}

func (or *orchestrator) initDeprecatedDatabasePlugin(ctx context.Context, plugin database.Plugin) (err error) {
	log.L(ctx).Warnf("Your database config uses a deprecated configuration structure - the database configuration has been moved under the 'plugins' section")
	err = plugin.Init(ctx, deprecatedDatabaseConfig.SubSection(plugin.Name()), or)
	if err != nil {
		return err
	}

	deprecatedPluginName := "database_0"
	or.databases[deprecatedPluginName] = plugin
	or.database = plugin
	return err
}

func (or *orchestrator) initBlockchainPlugins(ctx context.Context, plugins []blockchain.Plugin) (err error) {
	for idx, plugin := range plugins {
		config := blockchainConfig.ArrayEntry(idx)
		err = plugin.Init(ctx, config, &or.bc, or.metrics)
		if err != nil {
			return err
		}
		name := config.GetString(coreconfig.PluginConfigName)
		or.blockchains[name] = plugin

		if or.blockchain == nil {
			or.blockchain = plugin
		}
	}

	return err
}

func (or *orchestrator) initTokens(ctx context.Context) (err error) {
	or.tokens = make(map[string]tokens.Plugin)
	tokensConfigArraySize := tokensConfig.ArraySize()
	for i := 0; i < tokensConfigArraySize; i++ {
		config := tokensConfig.ArrayEntry(i)
		name := config.GetString(coreconfig.PluginConfigName)
		pluginType := config.GetString(coreconfig.PluginConfigType)
		if err = or.validatePluginConfig(ctx, config, "tokens"); err != nil {
			return err
		}

		log.L(ctx).Infof("Loading tokens plugin name=%s type=%s", name, pluginType)
		pluginConfig := config.SubSection(pluginType)

		plugin, err := tifactory.GetPlugin(ctx, pluginType)
		if plugin != nil {
			err = plugin.Init(ctx, name, pluginConfig, &or.bc)
		}
		if err != nil {
			return err
		}
		or.tokens[name] = plugin
	}

	if len(or.tokens) > 0 {
		return nil
	}

	// If there still is no tokens config, check the deprecated structure for config
	tokensConfigArraySize = deprecatedTokensConfig.ArraySize()
	if tokensConfigArraySize > 0 {
		log.L(ctx).Warnf("Your tokens config uses a deprecated configuration structure - the tokens configuration has been moved under the 'plugins' section")
	}

	for i := 0; i < tokensConfigArraySize; i++ {
		prefix := deprecatedTokensConfig.ArrayEntry(i)
		name := prefix.GetString(coreconfig.PluginConfigName)
		pluginName := prefix.GetString(tokens.TokensConfigPlugin)
		if name == "" {
			return i18n.NewError(ctx, coremsgs.MsgMissingTokensPluginConfig)
		}
		if err = core.ValidateFFNameField(ctx, name, "name"); err != nil {
			return err
		}

		log.L(ctx).Infof("Loading tokens plugin name=%s plugin=%s", name, pluginName)
		plugin, err := tifactory.GetPlugin(ctx, pluginName)
		if plugin != nil {
			err = plugin.Init(ctx, name, prefix, &or.bc)
		}
		if err != nil {
			return err
		}
		or.tokens[name] = plugin
	}
	return nil
}

func (or *orchestrator) initComponents(ctx context.Context) (err error) {

	if or.data == nil {
		or.data, err = data.NewDataManager(ctx, or.database, or.sharedstorage, or.dataexchange)
		if err != nil {
			return err
		}
	}

	if or.txHelper == nil {
		or.txHelper = txcommon.NewTransactionHelper(or.database, or.data)
	}

	if or.identity == nil {
		or.identity, err = identity.NewIdentityManager(ctx, or.database, or.identityPlugins, or.blockchain, or.data)
		if err != nil {
			return err
		}
	}

	if or.batch == nil {
		or.batch, err = batch.NewBatchManager(ctx, or, or.database, or.data, or.txHelper)
		if err != nil {
			return err
		}
	}

	if or.operations == nil {
		if or.operations, err = operations.NewOperationsManager(ctx, or.database, or.txHelper); err != nil {
			return err
		}
	}

	or.syncasync = syncasync.NewSyncAsyncBridge(ctx, or.database, or.data)

	if or.batchpin == nil {
		if or.batchpin, err = batchpin.NewBatchPinSubmitter(ctx, or.database, or.identity, or.blockchain, or.metrics, or.operations); err != nil {
			return err
		}
	}

	if or.messaging == nil {
		if or.messaging, err = privatemessaging.NewPrivateMessaging(ctx, or.database, or.identity, or.dataexchange, or.blockchain, or.batch, or.data, or.syncasync, or.batchpin, or.metrics, or.operations); err != nil {
			return err
		}
	}

	if or.broadcast == nil {
		if or.broadcast, err = broadcast.NewBroadcastManager(ctx, or.database, or.identity, or.data, or.blockchain, or.dataexchange, or.sharedstorage, or.batch, or.syncasync, or.batchpin, or.metrics, or.operations); err != nil {
			return err
		}
	}

	if or.assets == nil {
		or.assets, err = assets.NewAssetManager(ctx, or.database, or.identity, or.data, or.syncasync, or.broadcast, or.messaging, or.tokens, or.metrics, or.operations, or.txHelper)
		if err != nil {
			return err
		}
	}

	if or.contracts == nil {
		or.contracts, err = contracts.NewContractManager(ctx, or.database, or.broadcast, or.identity, or.blockchain, or.operations, or.txHelper, or.syncasync)
		if err != nil {
			return err
		}
	}

	if or.definitions == nil {
		or.definitions, err = definitions.NewDefinitionHandler(ctx, or.database, or.blockchain, or.dataexchange, or.data, or.identity, or.assets, or.contracts)
		if err != nil {
			return err
		}
	}

	if or.sharedDownload == nil {
		or.sharedDownload, err = shareddownload.NewDownloadManager(ctx, or.database, or.sharedstorage, or.dataexchange, or.operations, &or.bc)
		if err != nil {
			return err
		}
	}

	if or.events == nil {
		or.events, err = events.NewEventManager(ctx, or, or.sharedstorage, or.database, or.blockchain, or.identity, or.definitions, or.data, or.broadcast, or.messaging, or.assets, or.sharedDownload, or.metrics, or.txHelper)
		if err != nil {
			return err
		}
	}

	if or.networkmap == nil {
		or.networkmap, err = networkmap.NewNetworkMap(ctx, or.database, or.broadcast, or.dataexchange, or.identity, or.syncasync)
		if err != nil {
			return err
		}
	}

	or.syncasync.Init(or.events)

	return nil
}

func (or *orchestrator) initNamespaces(ctx context.Context) (err error) {
	if or.namespace == nil {
		or.namespace = namespace.NewNamespaceManager(ctx)
	}
	return or.namespace.Init(ctx, or.database)
}

func (or *orchestrator) MigrateNetwork(ctx context.Context) error {
	verifier, err := or.identity.GetNodeOwnerBlockchainKey(ctx)
	if err != nil {
		return err
	}
	return or.blockchain.SubmitOperatorAction(ctx, fftypes.NewUUID(), verifier.Value, blockchain.OperatorActionTerminate)
}
