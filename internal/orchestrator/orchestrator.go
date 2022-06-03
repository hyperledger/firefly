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
	"github.com/hyperledger/firefly/internal/adminevents"
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
	idplugin "github.com/hyperledger/firefly/pkg/identity"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
	"github.com/hyperledger/firefly/pkg/tokens"
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
	SubmitNetworkAction(ctx context.Context, action *core.NetworkAction) error
}

// this is definitely not the right place for these structs
// maybe in the factories?
type BlockchainPlugin struct {
	Plugin blockchain.Plugin
	Config config.Section
}

type DatabasePlugin struct {
	Plugin database.Plugin
	Config config.Section
}

type DataexchangePlugin struct {
	Plugin dataexchange.Plugin
	Config config.Section
}

type SharedStoragePlugin struct {
	Plugin sharedstorage.Plugin
	Config config.Section
}

type TokensPlugin struct {
	Plugin tokens.Plugin
	Config config.Section
	Name   string
}

type IdentityPlugin struct {
	Plugin idplugin.Plugin
	Config config.Section
}

type orchestrator struct {
	ctx            context.Context
	cancelCtx      context.CancelFunc
	started        bool
	namespace      string
	blockchain     BlockchainPlugin
	identity       identity.Manager
	idPlugin       IdentityPlugin
	sharedstorage  SharedStoragePlugin
	dataexchange   DataexchangePlugin
	database       DatabasePlugin
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
	tokens         map[string]TokensPlugin
	bc             boundCallbacks
	contracts      contracts.Manager
	node           *fftypes.UUID
	metrics        metrics.Manager
	operations     operations.Manager
	adminEvents    adminevents.Manager
	sharedDownload shareddownload.Manager
	txHelper       txcommon.Helper
}

func NewOrchestrator(ns string, bc BlockchainPlugin, db DatabasePlugin, ss SharedStoragePlugin, dx DataexchangePlugin, tokens map[string]TokensPlugin, id IdentityPlugin, metrics metrics.Manager) Orchestrator {
	or := &orchestrator{
		namespace:     ns,
		blockchain:    bc,
		database:      db,
		sharedstorage: ss,
		dataexchange:  dx,
		tokens:        tokens,
		idPlugin:      id,
		metrics:       metrics,
	}

	return or
}

func (or *orchestrator) Init(ctx context.Context, cancelCtx context.CancelFunc) (err error) {
	or.ctx = ctx
	or.cancelCtx = cancelCtx
	err = or.initPlugins(ctx)
	if err == nil {
		err = or.initComponents(ctx)
	}
	// Bind together the blockchain interface callbacks, with the events manager
	or.bc.bi = or.blockchain.Plugin
	or.bc.ei = or.events
	or.bc.dx = or.dataexchange.Plugin
	or.bc.ss = or.sharedstorage.Plugin
	or.bc.om = or.operations
	return err
}

func (or *orchestrator) Start() (err error) {
	var ns *core.Namespace
	if err == nil {
		ns, err = or.database.Plugin.GetNamespace(or.ctx, or.namespace)
	}
	if err == nil {
		err = or.blockchain.Plugin.ConfigureContract(or.ctx, &ns.Contracts)
	}
	if err == nil {
		err = or.blockchain.Plugin.Start()
	}
	if err == nil {
		err = or.database.Plugin.UpsertNamespace(or.ctx, ns, true)
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
		for _, el := range or.tokens {
			if err = el.Plugin.Start(); err != nil {
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

func (or *orchestrator) initPlugins(ctx context.Context) (err error) {
	err = or.database.Plugin.Init(ctx, or.database.Config, or)
	if err != nil {
		return err
	}

	err = or.blockchain.Plugin.Init(ctx, or.blockchain.Config, &or.bc, or.metrics)
	if err != nil {
		return err
	}

	fb := database.IdentityQueryFactory.NewFilter(ctx)
	nodes, _, err := or.database.Plugin.GetIdentities(ctx, fb.And(
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

	err = or.dataexchange.Plugin.Init(ctx, or.dataexchange.Config, nodeInfo, &or.bc)
	if err != nil {
		return err
	}

	err = or.sharedstorage.Plugin.Init(ctx, or.sharedstorage.Config, &or.bc)
	if err != nil {
		return err
	}

	for _, token := range or.tokens {
		err = token.Plugin.Init(ctx, token.Name, token.Config, &or.bc)
	}

	return err
}

func (or *orchestrator) initComponents(ctx context.Context) (err error) {

	if or.data == nil {
		or.data, err = data.NewDataManager(ctx, or.database.Plugin, or.sharedstorage.Plugin, or.dataexchange.Plugin)
		if err != nil {
			return err
		}
	}

	if or.txHelper == nil {
		or.txHelper = txcommon.NewTransactionHelper(or.database.Plugin, or.data)
	}

	if or.identity == nil {
		or.identity, err = identity.NewIdentityManager(ctx, or.database.Plugin, or.idPlugin.Plugin, or.blockchain.Plugin, or.data)
		if err != nil {
			return err
		}
	}

	if or.batch == nil {
		or.batch, err = batch.NewBatchManager(ctx, or, or.database.Plugin, or.data, or.txHelper)
		if err != nil {
			return err
		}
	}

	if or.operations == nil {
		if or.operations, err = operations.NewOperationsManager(ctx, or.database.Plugin, or.txHelper); err != nil {
			return err
		}
	}

	or.syncasync = syncasync.NewSyncAsyncBridge(ctx, or.database.Plugin, or.data)

	if or.batchpin == nil {
		if or.batchpin, err = batchpin.NewBatchPinSubmitter(ctx, or.database.Plugin, or.identity, or.blockchain.Plugin, or.metrics, or.operations); err != nil {
			return err
		}
	}

	if or.messaging == nil {
		if or.messaging, err = privatemessaging.NewPrivateMessaging(ctx, or.database.Plugin, or.identity, or.dataexchange.Plugin, or.blockchain.Plugin, or.batch, or.data, or.syncasync, or.batchpin, or.metrics, or.operations); err != nil {
			return err
		}
	}

	if or.broadcast == nil {
		if or.broadcast, err = broadcast.NewBroadcastManager(ctx, or.database.Plugin, or.identity, or.data, or.blockchain.Plugin, or.dataexchange.Plugin, or.sharedstorage.Plugin, or.batch, or.syncasync, or.batchpin, or.metrics, or.operations); err != nil {
			return err
		}
	}

	if or.assets == nil {
		or.assets, err = assets.NewAssetManager(ctx, or.database.Plugin, or.identity, or.data, or.syncasync, or.broadcast, or.messaging, nil, or.metrics, or.operations, or.txHelper)
		if err != nil {
			return err
		}
	}

	if or.contracts == nil {
		or.contracts, err = contracts.NewContractManager(ctx, or.database.Plugin, or.broadcast, or.identity, or.blockchain.Plugin, or.operations, or.txHelper, or.syncasync)
		if err != nil {
			return err
		}
	}

	if or.definitions == nil {
		or.definitions, err = definitions.NewDefinitionHandler(ctx, or.database.Plugin, or.blockchain.Plugin, or.dataexchange.Plugin, or.data, or.identity, or.assets, or.contracts)
		if err != nil {
			return err
		}
	}

	if or.sharedDownload == nil {
		or.sharedDownload, err = shareddownload.NewDownloadManager(ctx, or.database.Plugin, or.sharedstorage.Plugin, or.dataexchange.Plugin, or.operations, &or.bc)
		if err != nil {
			return err
		}
	}

	if or.events == nil {
		or.events, err = events.NewEventManager(ctx, or, or.sharedstorage.Plugin, or.database.Plugin, or.blockchain.Plugin, or.identity, or.definitions, or.data, or.broadcast, or.messaging, or.assets, or.sharedDownload, or.metrics, or.txHelper)
		if err != nil {
			return err
		}
	}

	if or.networkmap == nil {
		or.networkmap, err = networkmap.NewNetworkMap(ctx, or.database.Plugin, or.broadcast, or.dataexchange.Plugin, or.identity, or.syncasync)
		if err != nil {
			return err
		}
	}

	or.syncasync.Init(or.events)

	return nil
}

func (or *orchestrator) SubmitNetworkAction(ctx context.Context, action *core.NetworkAction) error {
	verifier, err := or.identity.GetNodeOwnerBlockchainKey(ctx)
	if err != nil {
		return err
	}
	if action.Type != core.NetworkActionTerminate {
		return i18n.NewError(ctx, coremsgs.MsgUnrecognizedNetworkAction, action.Type)
	}
	return or.blockchain.Plugin.SubmitNetworkAction(ctx, fftypes.NewUUID(), verifier.Value, action.Type)
}
