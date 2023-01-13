// Copyright © 2023 Kaleido, Inc.
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

	"github.com/hyperledger/firefly-common/pkg/auth"
	"github.com/hyperledger/firefly-common/pkg/ffapi"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/assets"
	"github.com/hyperledger/firefly/internal/batch"
	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/contracts"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/definitions"
	"github.com/hyperledger/firefly/internal/events"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/multiparty"
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

	MultiParty() multiparty.Manager             // only for multiparty
	BatchManager() batch.Manager                // only for multiparty
	Broadcast() broadcast.Manager               // only for multiparty
	PrivateMessaging() privatemessaging.Manager // only for multiparty
	Assets() assets.Manager
	DefinitionSender() definitions.Sender
	Contracts() contracts.Manager
	Data() data.Manager
	Events() events.EventManager
	NetworkMap() networkmap.Manager
	Operations() operations.Manager
	Identity() identity.Manager

	// Status
	GetStatus(ctx context.Context) (*core.NamespaceStatus, error)

	// Subscription management
	GetSubscriptions(ctx context.Context, filter ffapi.AndFilter) ([]*core.Subscription, *ffapi.FilterResult, error)
	GetSubscriptionByID(ctx context.Context, id string) (*core.Subscription, error)
	GetSubscriptionByIDWithStatus(ctx context.Context, id string) (*core.SubscriptionWithStatus, error)
	CreateSubscription(ctx context.Context, subDef *core.Subscription) (*core.Subscription, error)
	CreateUpdateSubscription(ctx context.Context, subDef *core.Subscription) (*core.Subscription, error)
	DeleteSubscription(ctx context.Context, id string) error

	// Data Query
	GetNamespace(ctx context.Context) *core.Namespace
	GetTransactionByID(ctx context.Context, id string) (*core.Transaction, error)
	GetTransactionOperations(ctx context.Context, id string) ([]*core.Operation, *ffapi.FilterResult, error)
	GetTransactionBlockchainEvents(ctx context.Context, id string) ([]*core.BlockchainEvent, *ffapi.FilterResult, error)
	GetTransactionStatus(ctx context.Context, id string) (*core.TransactionStatus, error)
	GetTransactions(ctx context.Context, filter ffapi.AndFilter) ([]*core.Transaction, *ffapi.FilterResult, error)
	GetMessageByID(ctx context.Context, id string) (*core.Message, error)
	GetMessageByIDWithData(ctx context.Context, id string) (*core.MessageInOut, error)
	GetMessages(ctx context.Context, filter ffapi.AndFilter) ([]*core.Message, *ffapi.FilterResult, error)
	GetMessagesWithData(ctx context.Context, filter ffapi.AndFilter) ([]*core.MessageInOut, *ffapi.FilterResult, error)
	GetMessageTransaction(ctx context.Context, id string) (*core.Transaction, error)
	GetMessageEvents(ctx context.Context, id string, filter ffapi.AndFilter) ([]*core.Event, *ffapi.FilterResult, error)
	GetMessageData(ctx context.Context, id string) (core.DataArray, error)
	GetMessagesForData(ctx context.Context, dataID string, filter ffapi.AndFilter) ([]*core.Message, *ffapi.FilterResult, error)
	GetBatchByID(ctx context.Context, id string) (*core.BatchPersisted, error)
	GetBatches(ctx context.Context, filter ffapi.AndFilter) ([]*core.BatchPersisted, *ffapi.FilterResult, error)
	GetDataByID(ctx context.Context, id string) (*core.Data, error)
	GetData(ctx context.Context, filter ffapi.AndFilter) (core.DataArray, *ffapi.FilterResult, error)
	GetDatatypeByID(ctx context.Context, id string) (*core.Datatype, error)
	GetDatatypeByName(ctx context.Context, name, version string) (*core.Datatype, error)
	GetDatatypes(ctx context.Context, filter ffapi.AndFilter) ([]*core.Datatype, *ffapi.FilterResult, error)
	GetOperationByID(ctx context.Context, id string) (*core.Operation, error)
	GetOperationByIDWithStatus(ctx context.Context, id string) (*core.OperationWithDetailedStatus, error)
	GetOperations(ctx context.Context, filter ffapi.AndFilter) ([]*core.Operation, *ffapi.FilterResult, error)
	GetEventByID(ctx context.Context, id string) (*core.Event, error)
	GetEventByIDWithReference(ctx context.Context, id string) (*core.EnrichedEvent, error)
	GetEvents(ctx context.Context, filter ffapi.AndFilter) ([]*core.Event, *ffapi.FilterResult, error)
	GetEventsWithReferences(ctx context.Context, filter ffapi.AndFilter) ([]*core.EnrichedEvent, *ffapi.FilterResult, error)
	GetBlockchainEventByID(ctx context.Context, id string) (*core.BlockchainEvent, error)
	GetBlockchainEvents(ctx context.Context, filter ffapi.AndFilter) ([]*core.BlockchainEvent, *ffapi.FilterResult, error)
	GetPins(ctx context.Context, filter ffapi.AndFilter) ([]*core.Pin, *ffapi.FilterResult, error)
	RewindPins(ctx context.Context, rewind *core.PinRewind) (*core.PinRewind, error)

	// Charts
	GetChartHistogram(ctx context.Context, startTime int64, endTime int64, buckets int64, tableName database.CollectionName) ([]*core.ChartHistogram, error)

	// Message Routing
	RequestReply(ctx context.Context, msg *core.MessageInOut) (reply *core.MessageInOut, err error)

	// Network Operations
	SubmitNetworkAction(ctx context.Context, action *core.NetworkAction) error

	// Authorizer
	Authorize(ctx context.Context, authReq *fftypes.AuthReq) error
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

type AuthPlugin struct {
	Name   string
	Plugin auth.Plugin
}

type Plugins struct {
	Blockchain    BlockchainPlugin
	Identity      IdentityPlugin
	SharedStorage SharedStoragePlugin
	DataExchange  DataExchangePlugin
	Database      DatabasePlugin
	Tokens        []TokensPlugin
	Events        map[string]eventsplugin.Plugin
	Auth          AuthPlugin
}

type Config struct {
	DefaultKey          string
	KeyNormalization    string
	Multiparty          multiparty.Config
	TokenBroadcastNames map[string]string
}

type orchestrator struct {
	ctx            context.Context
	cancelCtx      context.CancelFunc
	started        bool
	namespace      *core.Namespace
	config         Config
	plugins        *Plugins
	multiparty     multiparty.Manager       // only for multiparty
	batch          batch.Manager            // only for multiparty
	broadcast      broadcast.Manager        // only for multiparty
	messaging      privatemessaging.Manager // only for multiparty
	sharedDownload shareddownload.Manager   // only for multiparty
	identity       identity.Manager
	events         events.EventManager
	networkmap     networkmap.Manager
	defhandler     definitions.Handler
	defsender      definitions.Sender
	data           data.Manager
	syncasync      syncasync.Bridge
	assets         assets.Manager
	bc             boundCallbacks
	contracts      contracts.Manager
	metrics        metrics.Manager
	cacheManager   cache.Manager
	operations     operations.Manager
	txHelper       txcommon.Helper
}

func NewOrchestrator(ns *core.Namespace, config Config, plugins *Plugins, metrics metrics.Manager, cacheManager cache.Manager) Orchestrator {
	or := &orchestrator{
		namespace:    ns,
		config:       config,
		plugins:      plugins,
		metrics:      metrics,
		cacheManager: cacheManager,
	}
	return or
}

func (or *orchestrator) Init(ctx context.Context, cancelCtx context.CancelFunc) (err error) {
	namespaceLog := or.namespace.Name
	if or.namespace.NetworkName != "" && or.namespace.NetworkName != or.namespace.Name {
		namespaceLog += "->" + or.namespace.NetworkName
	}
	or.ctx, or.cancelCtx = context.WithCancel(log.WithLogField(ctx, "ns", namespaceLog))
	err = or.initComponents(or.ctx)
	if err == nil {
		err = or.initHandlers(or.ctx)
	}
	// Bind together the blockchain interface callbacks, with the events manager
	or.bc.ei = or.events
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
	or.data.Start()
	if or.config.Multiparty.Enabled {
		err = or.batch.Start()
		if err == nil {
			err = or.broadcast.Start()
		}
		if err == nil {
			err = or.sharedDownload.Start()
		}
	}
	if err == nil {
		err = or.events.Start()
	}
	if err == nil {
		err = or.operations.Start()
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
	if or.events != nil {
		or.events.WaitStop()
		or.events = nil
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

func (or *orchestrator) DefinitionSender() definitions.Sender {
	return or.defsender
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

func (or *orchestrator) MultiParty() multiparty.Manager {
	return or.multiparty
}

func (or *orchestrator) Identity() identity.Manager {
	return or.identity
}

func (or *orchestrator) initHandlers(ctx context.Context) (err error) {
	or.plugins.Database.Plugin.SetHandler(or.namespace.Name, or)

	if or.plugins.Blockchain.Plugin != nil {
		or.plugins.Blockchain.Plugin.SetHandler(or.namespace.Name, or.events)
		or.plugins.Blockchain.Plugin.SetOperationHandler(or.namespace.Name, &or.bc)
	}

	if or.plugins.SharedStorage.Plugin != nil {
		or.plugins.SharedStorage.Plugin.SetHandler(or.namespace.Name, &or.bc)
	}

	if or.plugins.DataExchange.Plugin != nil {
		fb := database.IdentityQueryFactory.NewFilter(ctx)
		nodes, _, err := or.database().GetIdentities(ctx, or.namespace.Name, fb.And(
			fb.Eq("type", core.IdentityTypeNode),
		))
		if err != nil {
			return err
		}
		for _, node := range nodes {
			err = or.plugins.DataExchange.Plugin.AddNode(ctx, or.namespace.NetworkName, node.Name, node.Profile)
			if err != nil {
				return err
			}
		}
		or.plugins.DataExchange.Plugin.SetHandler(or.namespace.NetworkName, or.config.Multiparty.Node.Name, or.events)
		or.plugins.DataExchange.Plugin.SetOperationHandler(or.namespace.Name, &or.bc)
	}

	for _, token := range or.plugins.Tokens {
		token.Plugin.SetHandler(or.namespace.Name, or.events)
		token.Plugin.SetOperationHandler(or.namespace.Name, &or.bc)
	}

	return nil
}

func (or *orchestrator) initMultiPartyComponents(ctx context.Context) (err error) {
	if or.batch == nil {
		or.batch, err = batch.NewBatchManager(ctx, or.namespace.Name, or.database(), or.data, or.identity, or.txHelper)
		if err != nil {
			return err
		}
	}

	if or.messaging == nil {
		if or.messaging, err = privatemessaging.NewPrivateMessaging(ctx, or.namespace, or.database(), or.dataexchange(), or.blockchain(), or.identity, or.batch, or.data, or.syncasync, or.multiparty, or.metrics, or.operations, or.cacheManager); err != nil {
			return err
		}
	}
	return nil
}

func (or *orchestrator) initManagers(ctx context.Context) (err error) {

	if or.txHelper == nil {
		if or.txHelper, err = txcommon.NewTransactionHelper(ctx, or.namespace.Name, or.database(), or.data, or.cacheManager); err != nil {
			return err
		}
	}

	if or.operations == nil {
		if or.operations, err = operations.NewOperationsManager(ctx, or.namespace.Name, or.database(), or.txHelper, or.cacheManager); err != nil {
			return err
		}
	}

	if or.config.Multiparty.Enabled {
		if or.multiparty == nil {
			or.multiparty, err = multiparty.NewMultipartyManager(or.ctx, or.namespace, or.config.Multiparty, or.database(), or.blockchain(), or.operations, or.metrics, or.txHelper)
			if err != nil {
				return err
			}
		}
		if err = or.multiparty.ConfigureContract(ctx); err != nil {
			return err
		}
	}

	if or.identity == nil {
		or.identity, err = identity.NewIdentityManager(ctx, or.namespace.Name, or.config.DefaultKey, or.database(), or.blockchain(), or.multiparty, or.cacheManager)
		if err != nil {
			return err
		}
	}

	or.syncasync = syncasync.NewSyncAsyncBridge(ctx, or.namespace.Name, or.database(), or.data, or.operations)

	if or.config.Multiparty.Enabled {
		if err = or.initMultiPartyComponents(ctx); err != nil {
			return err
		}
	}

	if or.dataexchange() != nil && or.sharedstorage() != nil {
		if or.broadcast == nil {
			if or.broadcast, err = broadcast.NewBroadcastManager(ctx, or.namespace, or.database(), or.blockchain(), or.dataexchange(), or.sharedstorage(), or.identity, or.data, or.batch, or.syncasync, or.multiparty, or.metrics, or.operations, or.txHelper); err != nil {
				return err
			}
		}

		if or.sharedDownload == nil {
			or.sharedDownload, err = shareddownload.NewDownloadManager(ctx, or.namespace, or.database(), or.sharedstorage(), or.dataexchange(), or.operations, &or.bc)
			if err != nil {
				return err
			}
		}
	}

	if or.blockchain() != nil {
		if or.contracts == nil {
			or.contracts, err = contracts.NewContractManager(ctx, or.namespace.Name, or.database(), or.blockchain(), or.identity, or.operations, or.txHelper, or.syncasync)
			if err != nil {
				return err
			}
		}
	}

	if or.assets == nil {
		or.assets, err = assets.NewAssetManager(ctx, or.namespace.Name, or.config.KeyNormalization, or.database(), or.tokens(), or.identity, or.syncasync, or.broadcast, or.messaging, or.metrics, or.operations, or.contracts, or.txHelper)
		if err != nil {
			return err
		}
	}

	if or.defsender == nil {
		or.defsender, or.defhandler, err = definitions.NewDefinitionSender(ctx, or.namespace, or.config.Multiparty.Enabled, or.database(), or.blockchain(), or.dataexchange(), or.broadcast, or.identity, or.data, or.assets, or.contracts, or.config.TokenBroadcastNames)
		if err != nil {
			return err
		}
	}

	if or.networkmap == nil {
		or.networkmap, err = networkmap.NewNetworkMap(ctx, or.namespace.Name, or.database(), or.dataexchange(), or.defsender, or.identity, or.syncasync, or.multiparty)
		if err != nil {
			return err
		}
	}

	return nil
}

func (or *orchestrator) initComponents(ctx context.Context) (err error) {
	if or.data == nil {
		or.data, err = data.NewDataManager(ctx, or.namespace, or.database(), or.dataexchange(), or.cacheManager)
		if err != nil {
			return err
		}
	}

	if err := or.initManagers(ctx); err != nil {
		return err
	}

	if or.events == nil {
		or.events, err = events.NewEventManager(ctx, or.namespace, or.database(), or.blockchain(), or.identity, or.defhandler, or.data, or.defsender, or.broadcast, or.messaging, or.assets, or.sharedDownload, or.metrics, or.operations, or.txHelper, or.plugins.Events, or.multiparty, or.cacheManager)
		if err != nil {
			return err
		}
	}

	or.syncasync.Init(or.events)

	return nil
}

func (or *orchestrator) SubmitNetworkAction(ctx context.Context, action *core.NetworkAction) error {
	if or.multiparty == nil {
		return i18n.NewError(ctx, coremsgs.MsgActionNotSupported)
	}
	key, err := or.identity.NormalizeSigningKey(ctx, "", identity.KeyNormalizationBlockchainPlugin)
	if err != nil {
		return err
	}
	return or.multiparty.SubmitNetworkAction(ctx, key, action)
}

func (or *orchestrator) Authorize(ctx context.Context, authReq *fftypes.AuthReq) error {
	authReq.Namespace = or.namespace.Name
	if or.plugins.Auth.Plugin != nil {
		return or.plugins.Auth.Plugin.Authorize(ctx, authReq)
	}
	return nil
}

func (or *orchestrator) RewindPins(ctx context.Context, rewind *core.PinRewind) (*core.PinRewind, error) {
	if rewind.Sequence > 0 {
		fb := database.PinQueryFactory.NewFilter(ctx)
		if pins, _, err := or.GetPins(ctx, fb.And(fb.Eq("seq", rewind.Sequence))); err != nil {
			return nil, err
		} else if len(pins) > 0 {
			rewind.Batch = pins[0].Batch
			or.events.QueueBatchRewind(rewind.Batch)
			return rewind, nil
		}
		return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
	}
	or.events.QueueBatchRewind(rewind.Batch)
	return rewind, nil
}
