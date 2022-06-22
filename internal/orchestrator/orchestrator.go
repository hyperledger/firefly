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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/assets"
	"github.com/hyperledger/firefly/internal/batch"
	"github.com/hyperledger/firefly/internal/batchpin"
	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/contracts"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/definitions"
	"github.com/hyperledger/firefly/internal/events"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/networkmap"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/internal/privatemessaging"
	"github.com/hyperledger/firefly/internal/shareddownload"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	eventsplugin "github.com/hyperledger/firefly/pkg/events"
	idplugin "github.com/hyperledger/firefly/pkg/identity"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
	"github.com/hyperledger/firefly/pkg/tokens"
)

// Orchestrator is the main interface behind the API, implementing the actions
type Orchestrator interface {
	Init(ctx context.Context, cancelCtx context.CancelFunc) error
	Start() error
	WaitStop() // The close itself is performed by canceling the context
	Assets() assets.Manager
	BatchManager() batch.Manager
	Broadcast() broadcast.Manager
	Contracts() contracts.Manager
	Data() data.Manager
	Events() events.EventManager
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
	GetTransactionByID(ctx context.Context, id string) (*core.Transaction, error)
	GetTransactionOperations(ctx context.Context, id string) ([]*core.Operation, *database.FilterResult, error)
	GetTransactionBlockchainEvents(ctx context.Context, id string) ([]*core.BlockchainEvent, *database.FilterResult, error)
	GetTransactionStatus(ctx context.Context, id string) (*core.TransactionStatus, error)
	GetTransactions(ctx context.Context, filter database.AndFilter) ([]*core.Transaction, *database.FilterResult, error)
	GetMessageByID(ctx context.Context, id string) (*core.Message, error)
	GetMessageByIDWithData(ctx context.Context, id string) (*core.MessageInOut, error)
	GetMessages(ctx context.Context, filter database.AndFilter) ([]*core.Message, *database.FilterResult, error)
	GetMessagesWithData(ctx context.Context, filter database.AndFilter) ([]*core.MessageInOut, *database.FilterResult, error)
	GetMessageTransaction(ctx context.Context, id string) (*core.Transaction, error)
	GetMessageEvents(ctx context.Context, id string, filter database.AndFilter) ([]*core.Event, *database.FilterResult, error)
	GetMessageData(ctx context.Context, id string) (core.DataArray, error)
	GetMessagesForData(ctx context.Context, dataID string, filter database.AndFilter) ([]*core.Message, *database.FilterResult, error)
	GetBatchByID(ctx context.Context, id string) (*core.BatchPersisted, error)
	GetBatches(ctx context.Context, filter database.AndFilter) ([]*core.BatchPersisted, *database.FilterResult, error)
	GetDataByID(ctx context.Context, id string) (*core.Data, error)
	GetData(ctx context.Context, filter database.AndFilter) (core.DataArray, *database.FilterResult, error)
	GetDatatypeByID(ctx context.Context, id string) (*core.Datatype, error)
	GetDatatypeByName(ctx context.Context, name, version string) (*core.Datatype, error)
	GetDatatypes(ctx context.Context, filter database.AndFilter) ([]*core.Datatype, *database.FilterResult, error)
	GetOperationByID(ctx context.Context, id string) (*core.Operation, error)
	GetOperations(ctx context.Context, filter database.AndFilter) ([]*core.Operation, *database.FilterResult, error)
	GetEventByID(ctx context.Context, id string) (*core.Event, error)
	GetEvents(ctx context.Context, filter database.AndFilter) ([]*core.Event, *database.FilterResult, error)
	GetEventsWithReferences(ctx context.Context, filter database.AndFilter) ([]*core.EnrichedEvent, *database.FilterResult, error)
	GetBlockchainEventByID(ctx context.Context, id string) (*core.BlockchainEvent, error)
	GetBlockchainEvents(ctx context.Context, filter database.AndFilter) ([]*core.BlockchainEvent, *database.FilterResult, error)
	GetPins(ctx context.Context, filter database.AndFilter) ([]*core.Pin, *database.FilterResult, error)

	// Charts
	GetChartHistogram(ctx context.Context, ns string, startTime int64, endTime int64, buckets int64, tableName database.CollectionName) ([]*core.ChartHistogram, error)

	// Message Routing
	RequestReply(ctx context.Context, msg *core.MessageInOut) (reply *core.MessageInOut, err error)

	// Network Operations
	SubmitNetworkAction(ctx context.Context, action *core.NetworkAction) error
}

type BlockchainPlugin struct {
	Name   string
	Plugin blockchain.Plugin
}

type DatabasePlugin struct {
	Name   string
	Plugin database.Plugin
}

type DataExchangePlugin struct {
	Name   string
	Plugin dataexchange.Plugin
}

type SharedStoragePlugin struct {
	Name   string
	Plugin sharedstorage.Plugin
}

type TokensPlugin struct {
	Name   string
	Plugin tokens.Plugin
}

type IdentityPlugin struct {
	Name   string
	Plugin idplugin.Plugin
}

type Plugins struct {
	Blockchain    BlockchainPlugin
	Identity      IdentityPlugin
	SharedStorage SharedStoragePlugin
	DataExchange  DataExchangePlugin
	Database      DatabasePlugin
	Tokens        []TokensPlugin
	Events        map[string]eventsplugin.Plugin
}

type Config struct {
	DefaultKey string
	Multiparty struct {
		Enabled bool
		OrgName string
		OrgDesc string
		OrgKey  string
	}
}

type orchestrator struct {
	ctx            context.Context
	cancelCtx      context.CancelFunc
	started        bool
	namespace      string
	config         Config
	plugins        Plugins
	identity       identity.Manager
	events         events.EventManager
	networkmap     networkmap.Manager
	batch          batch.Manager
	broadcast      broadcast.Manager
	messaging      privatemessaging.Manager
	definitions    definitions.DefinitionHandler
	data           data.Manager
	syncasync      syncasync.Bridge
	batchpin       batchpin.Submitter
	assets         assets.Manager
	bc             boundCallbacks
	contracts      contracts.Manager
	node           *fftypes.UUID
	metrics        metrics.Manager
	operations     operations.Manager
	sharedDownload shareddownload.Manager
	txHelper       txcommon.Helper
}

func NewOrchestrator(ns string, config Config, plugins Plugins, metrics metrics.Manager) Orchestrator {
	or := &orchestrator{
		namespace: ns,
		config:    config,
		plugins:   plugins,
		metrics:   metrics,
	}
	return or
}

func (or *orchestrator) Init(ctx context.Context, cancelCtx context.CancelFunc) (err error) {
	or.ctx = log.WithLogField(ctx, "ns", or.namespace)
	or.cancelCtx = cancelCtx
	err = or.initPlugins(or.ctx)
	if err == nil {
		err = or.initComponents(or.ctx)
	}
	// Bind together the blockchain interface callbacks, with the events manager
	or.bc.bi = or.plugins.Blockchain.Plugin
	or.bc.ei = or.events
	or.bc.dx = or.plugins.DataExchange.Plugin
	or.bc.ss = or.plugins.SharedStorage.Plugin
	or.bc.om = or.operations
	return err
}

func (or *orchestrator) database() database.Plugin {
	return or.plugins.Database.Plugin
}

func (or *orchestrator) blockchain() blockchain.Plugin {
	return or.plugins.Blockchain.Plugin
}

func (or *orchestrator) dataexchange() dataexchange.Plugin {
	return or.plugins.DataExchange.Plugin
}

func (or *orchestrator) sharedstorage() sharedstorage.Plugin {
	return or.plugins.SharedStorage.Plugin
}

func (or *orchestrator) tokens() map[string]tokens.Plugin {
	result := make(map[string]tokens.Plugin, len(or.plugins.Tokens))
	for _, plugin := range or.plugins.Tokens {
		result[plugin.Name] = plugin.Plugin
	}
	return result
}

func (or *orchestrator) Start() (err error) {
	var ns *core.Namespace
	ns, err = or.database().GetNamespace(or.ctx, or.namespace)
	if err == nil {
		if ns == nil {
			ns = &core.Namespace{
				Name:    or.namespace,
				Created: fftypes.Now(),
			}
		}
		err = or.blockchain().ConfigureContract(or.ctx, &ns.Contracts)
	}
	if err == nil {
		err = or.blockchain().Start()
	}
	if err == nil {
		err = or.database().UpsertNamespace(or.ctx, ns, true)
	}
	if err == nil {
		err = or.batch.Start()
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
		for _, el := range or.tokens() {
			if err = el.Start(); err != nil {
				break
			}
		}
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

func (or *orchestrator) Operations() operations.Manager {
	return or.operations
}

func (or *orchestrator) initPlugins(ctx context.Context) (err error) {
	or.plugins.Database.Plugin.SetHandler(or.namespace, or)
	or.plugins.Blockchain.Plugin.SetHandler(&or.bc)
	or.plugins.SharedStorage.Plugin.SetHandler(or.namespace, &or.bc)

	fb := database.IdentityQueryFactory.NewFilter(ctx)
	nodes, _, err := or.database().GetIdentities(ctx, or.namespace, fb.And(
		fb.Eq("type", core.IdentityTypeNode),
	))
	if err != nil {
		return err
	}
	nodeInfo := make([]fftypes.JSONObject, len(nodes))
	for i, node := range nodes {
		nodeInfo[i] = node.Profile
	}
	or.plugins.DataExchange.Plugin.SetNodes(nodeInfo)
	or.plugins.DataExchange.Plugin.SetHandler(or.namespace, &or.bc)

	for _, token := range or.plugins.Tokens {
		if err := token.Plugin.SetHandler(or.namespace, &or.bc); err != nil {
			return err
		}
	}

	return nil
}

func (or *orchestrator) initComponents(ctx context.Context) (err error) {

	if or.data == nil {
		or.data, err = data.NewDataManager(ctx, or.namespace, or.database(), or.sharedstorage(), or.dataexchange())
		if err != nil {
			return err
		}
	}

	if or.txHelper == nil {
		or.txHelper = txcommon.NewTransactionHelper(or.namespace, or.database(), or.data)
	}

	if or.identity == nil {
		or.identity, err = identity.NewIdentityManager(ctx, or.namespace, or.config.DefaultKey, or.config.Multiparty.OrgName, or.config.Multiparty.OrgKey, or.database(), or.blockchain(), or.data)
		if err != nil {
			return err
		}
	}

	if or.batch == nil {
		or.batch, err = batch.NewBatchManager(ctx, or.namespace, or, or.database(), or.data, or.txHelper)
		if err != nil {
			return err
		}
	}

	if or.operations == nil {
		if or.operations, err = operations.NewOperationsManager(ctx, or.namespace, or.database(), or.txHelper); err != nil {
			return err
		}
	}

	or.syncasync = syncasync.NewSyncAsyncBridge(ctx, or.namespace, or.database(), or.data)

	if or.batchpin == nil {
		if or.batchpin, err = batchpin.NewBatchPinSubmitter(ctx, or.namespace, or.database(), or.identity, or.blockchain(), or.metrics, or.operations); err != nil {
			return err
		}
	}

	if or.messaging == nil {
		if or.messaging, err = privatemessaging.NewPrivateMessaging(ctx, or.namespace, or.database(), or.identity, or.dataexchange(), or.blockchain(), or.batch, or.data, or.syncasync, or.batchpin, or.metrics, or.operations); err != nil {
			return err
		}
	}

	if or.broadcast == nil {
		if or.broadcast, err = broadcast.NewBroadcastManager(ctx, or.namespace, or.database(), or.identity, or.data, or.blockchain(), or.dataexchange(), or.sharedstorage(), or.batch, or.syncasync, or.batchpin, or.metrics, or.operations); err != nil {
			return err
		}
	}

	if or.assets == nil {
		or.assets, err = assets.NewAssetManager(ctx, or.namespace, or.database(), or.identity, or.data, or.syncasync, or.broadcast, or.messaging, or.tokens(), or.metrics, or.operations, or.txHelper)
		if err != nil {
			return err
		}
	}

	if or.contracts == nil {
		or.contracts, err = contracts.NewContractManager(ctx, or.namespace, or.database(), or.broadcast, or.identity, or.blockchain(), or.operations, or.txHelper, or.syncasync)
		if err != nil {
			return err
		}
	}

	if or.definitions == nil {
		or.definitions, err = definitions.NewDefinitionHandler(ctx, or.namespace, or.database(), or.blockchain(), or.dataexchange(), or.data, or.identity, or.assets, or.contracts)
		if err != nil {
			return err
		}
	}

	if or.sharedDownload == nil {
		or.sharedDownload, err = shareddownload.NewDownloadManager(ctx, or.namespace, or.database(), or.sharedstorage(), or.dataexchange(), or.operations, &or.bc)
		if err != nil {
			return err
		}
	}

	if or.events == nil {
		or.events, err = events.NewEventManager(ctx, or.namespace, or, or.sharedstorage(), or.database(), or.blockchain(), or.identity, or.definitions, or.data, or.broadcast, or.messaging, or.assets, or.sharedDownload, or.metrics, or.txHelper, or.plugins.Events)
		if err != nil {
			return err
		}
	}

	or.syncasync.Init(or.events)

	if or.networkmap == nil {
		or.networkmap, err = networkmap.NewNetworkMap(ctx, or.namespace, or.config.Multiparty.OrgName, or.config.Multiparty.OrgDesc, or.database(), or.data, or.broadcast, or.dataexchange(), or.identity, or.syncasync)
	}
	return err
}

func (or *orchestrator) SubmitNetworkAction(ctx context.Context, action *core.NetworkAction) error {
	key, err := or.identity.NormalizeSigningKey(ctx, "", identity.KeyNormalizationBlockchainPlugin)
	if err != nil {
		return err
	}
	if action.Type == core.NetworkActionTerminate {
		if or.namespace != core.LegacySystemNamespace {
			// For now, "terminate" only works on ff_system
			return i18n.NewError(ctx, coremsgs.MsgTerminateNotSupported, or.namespace)
		}
	} else {
		return i18n.NewError(ctx, coremsgs.MsgUnrecognizedNetworkAction, action.Type)
	}
	// TODO: This should be a new operation type
	po := &core.PreparedOperation{
		Namespace: or.namespace,
		ID:        fftypes.NewUUID(),
	}
	return or.blockchain().SubmitNetworkAction(ctx, po.NamespacedIDString(), key, action.Type)
}
