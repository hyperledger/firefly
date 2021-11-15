// Copyright © 2021 Kaleido, Inc.
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
	"fmt"

	"github.com/hyperledger/firefly/internal/assets"
	"github.com/hyperledger/firefly/internal/batch"
	"github.com/hyperledger/firefly/internal/batchpin"
	"github.com/hyperledger/firefly/internal/blockchain/bifactory"
	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/contracts"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/database/difactory"
	"github.com/hyperledger/firefly/internal/dataexchange/dxfactory"
	"github.com/hyperledger/firefly/internal/definitions"
	"github.com/hyperledger/firefly/internal/events"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/identity/iifactory"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/networkmap"
	"github.com/hyperledger/firefly/internal/privatemessaging"
	"github.com/hyperledger/firefly/internal/publicstorage/psfactory"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/internal/tokens/tifactory"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
	idplugin "github.com/hyperledger/firefly/pkg/identity"
	"github.com/hyperledger/firefly/pkg/publicstorage"
	"github.com/hyperledger/firefly/pkg/tokens"
)

var (
	blockchainConfig    = config.NewPluginConfig("blockchain")
	databaseConfig      = config.NewPluginConfig("database")
	identityConfig      = config.NewPluginConfig("identity")
	publicstorageConfig = config.NewPluginConfig("publicstorage")
	dataexchangeConfig  = config.NewPluginConfig("dataexchange")
	tokensConfig        = config.NewPluginConfig("tokens").Array()
)

// Orchestrator is the main interface behind the API, implementing the actions
type Orchestrator interface {
	Init(ctx context.Context, cancelCtx context.CancelFunc) error
	Start() error
	WaitStop() // The close itself is performed by canceling the context
	Broadcast() broadcast.Manager
	PrivateMessaging() privatemessaging.Manager
	Events() events.EventManager
	NetworkMap() networkmap.Manager
	Data() data.Manager
	Assets() assets.Manager
	IsPreInit() bool

	// Status
	GetStatus(ctx context.Context) (*fftypes.NodeStatus, error)

	// Subscription management
	GetSubscriptions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Subscription, *database.FilterResult, error)
	GetSubscriptionByID(ctx context.Context, ns, id string) (*fftypes.Subscription, error)
	CreateSubscription(ctx context.Context, ns string, subDef *fftypes.Subscription) (*fftypes.Subscription, error)
	CreateUpdateSubscription(ctx context.Context, ns string, subDef *fftypes.Subscription) (*fftypes.Subscription, error)
	DeleteSubscription(ctx context.Context, ns, id string) error

	// Data Query
	GetNamespace(ctx context.Context, ns string) (*fftypes.Namespace, error)
	GetNamespaces(ctx context.Context, filter database.AndFilter) ([]*fftypes.Namespace, *database.FilterResult, error)
	GetTransactionByID(ctx context.Context, ns, id string) (*fftypes.Transaction, error)
	GetTransactionOperations(ctx context.Context, ns, id string) ([]*fftypes.Operation, *database.FilterResult, error)
	GetTransactions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Transaction, *database.FilterResult, error)
	GetMessageByID(ctx context.Context, ns, id string) (*fftypes.Message, error)
	GetMessageByIDWithData(ctx context.Context, ns, id string) (*fftypes.MessageInOut, error)
	GetMessages(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Message, *database.FilterResult, error)
	GetMessagesWithData(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.MessageInOut, *database.FilterResult, error)
	GetMessageTransaction(ctx context.Context, ns, id string) (*fftypes.Transaction, error)
	GetMessageOperations(ctx context.Context, ns, id string) ([]*fftypes.Operation, *database.FilterResult, error)
	GetMessageEvents(ctx context.Context, ns, id string, filter database.AndFilter) ([]*fftypes.Event, *database.FilterResult, error)
	GetMessageData(ctx context.Context, ns, id string) ([]*fftypes.Data, error)
	GetMessagesForData(ctx context.Context, ns, dataID string, filter database.AndFilter) ([]*fftypes.Message, *database.FilterResult, error)
	GetBatchByID(ctx context.Context, ns, id string) (*fftypes.Batch, error)
	GetBatches(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Batch, *database.FilterResult, error)
	GetDataByID(ctx context.Context, ns, id string) (*fftypes.Data, error)
	GetData(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Data, *database.FilterResult, error)
	GetDatatypeByID(ctx context.Context, ns, id string) (*fftypes.Datatype, error)
	GetDatatypeByName(ctx context.Context, ns, name, version string) (*fftypes.Datatype, error)
	GetDatatypes(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Datatype, *database.FilterResult, error)
	GetOperationByID(ctx context.Context, ns, id string) (*fftypes.Operation, error)
	GetOperations(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Operation, *database.FilterResult, error)
	GetEventByID(ctx context.Context, ns, id string) (*fftypes.Event, error)
	GetEvents(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Event, *database.FilterResult, error)

	// Config Management
	GetConfig(ctx context.Context) fftypes.JSONObject
	GetConfigRecord(ctx context.Context, key string) (*fftypes.ConfigRecord, error)
	GetConfigRecords(ctx context.Context, filter database.AndFilter) ([]*fftypes.ConfigRecord, *database.FilterResult, error)
	PutConfigRecord(ctx context.Context, key string, configRecord fftypes.Byteable) (outputValue fftypes.Byteable, err error)
	DeleteConfigRecord(ctx context.Context, key string) (err error)
	ResetConfig(ctx context.Context)

	// Message Routing
	RequestReply(ctx context.Context, ns string, msg *fftypes.MessageInOut) (reply *fftypes.MessageInOut, err error)

	// Custom smart contracts
	AddContractInterface(ctx context.Context, ns string, cd *fftypes.FFI, waitConfirm bool) (output *fftypes.FFI, err error)
	GetContractInterfaces(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.FFI, *database.FilterResult, error)
	GetContractInterfaceByID(ctx context.Context, id string) (output *fftypes.FFI, err error)
	GetContractInterfaceByNameAndVersion(ctx context.Context, ns, name, version string) (output *fftypes.FFI, err error)
	// AddContractInstance(ctx context.Context, ns string, cd *fftypes.FFI, waitConfirm bool) (output *fftypes.FFI, err error)
	// GetContractInstances(ctx context.Context, ns string, filter database.AndFilter) (output []*fftypes.FFI, res *database.FilterResult, err error)
	// GetContractInstanceByNameOrID(ctx context.Context, ns, nameOrID string) (output *fftypes.FFI, err error)
	InvokeContract(ctx context.Context, ns, contractInstanceNameOrId, method string, req *fftypes.ContractInvocationRequest) (interface{}, error)
}

type orchestrator struct {
	ctx            context.Context
	cancelCtx      context.CancelFunc
	started        bool
	database       database.Plugin
	blockchain     blockchain.Plugin
	identity       identity.Manager
	identityPlugin idplugin.Plugin
	publicstorage  publicstorage.Plugin
	dataexchange   dataexchange.Plugin
	events         events.EventManager
	networkmap     networkmap.Manager
	batch          batch.Manager
	broadcast      broadcast.Manager
	messaging      privatemessaging.Manager
	definitions    definitions.DefinitionHandlers
	data           data.Manager
	syncasync      syncasync.Bridge
	batchpin       batchpin.Submitter
	assets         assets.Manager
	tokens         map[string]tokens.Plugin
	bc             boundCallbacks
	preInitMode    bool
	contracts      *contracts.ContractManager
}

func NewOrchestrator() Orchestrator {
	or := &orchestrator{}

	// Initialize the config on all the factories
	bifactory.InitPrefix(blockchainConfig)
	difactory.InitPrefix(databaseConfig)
	psfactory.InitPrefix(publicstorageConfig)
	dxfactory.InitPrefix(dataexchangeConfig)
	tifactory.InitPrefix(tokensConfig)

	return or
}

func (or *orchestrator) Init(ctx context.Context, cancelCtx context.CancelFunc) (err error) {
	or.ctx = ctx
	or.cancelCtx = cancelCtx
	err = or.initPlugins(ctx)
	if or.preInitMode {
		return nil
	}
	if err == nil {
		err = or.initComponents(ctx)
	}
	if err == nil {
		err = or.initNamespaces(ctx)
	}
	// Bind together the blockchain interface callbacks, with the events manager
	or.bc.bi = or.blockchain
	or.bc.ei = or.events
	or.bc.dx = or.dataexchange
	or.bc.am = or.assets
	return err
}

func (or *orchestrator) Start() error {
	if or.preInitMode {
		log.L(or.ctx).Infof("Orchestrator in pre-init mode, waiting for initialization")
		return nil
	}
	err := or.blockchain.Start()
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
		for _, el := range or.tokens {
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
	or.started = false
}

func (or *orchestrator) IsPreInit() bool {
	return or.preInitMode
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

func (or *orchestrator) NetworkMap() networkmap.Manager {
	return or.networkmap
}

func (or *orchestrator) Data() data.Manager {
	return or.data
}

func (or *orchestrator) Assets() assets.Manager {
	return or.assets
}

func (or *orchestrator) initDatabaseCheckPreinit(ctx context.Context) (err error) {

	if or.database == nil {
		diType := config.GetString(config.DatabaseType)
		if or.database, err = difactory.GetPlugin(ctx, diType); err != nil {
			return err
		}
	}
	if err = or.database.Init(ctx, databaseConfig.SubPrefix(or.database.Name()), or); err != nil {
		return err
	}

	// Read configuration from DB and merge with existing config
	var configRecords []*fftypes.ConfigRecord
	filter := database.ConfigRecordQueryFactory.NewFilter(ctx).And()
	if configRecords, _, err = or.GetConfigRecords(ctx, filter); err != nil {
		return err
	}
	if len(configRecords) == 0 && config.GetBool(config.AdminPreinit) {
		or.preInitMode = true
		return nil
	}
	return config.MergeConfig(configRecords)
}

func (or *orchestrator) initPlugins(ctx context.Context) (err error) {

	if err = or.initDatabaseCheckPreinit(ctx); err != nil {
		return err
	} else if or.preInitMode {
		return nil
	}

	if or.identityPlugin == nil {
		iiType := config.GetString(config.IdentityType)
		if or.identityPlugin, err = iifactory.GetPlugin(ctx, iiType); err != nil {
			return err
		}
	}
	if err = or.identityPlugin.Init(ctx, identityConfig.SubPrefix(or.identityPlugin.Name()), or); err != nil {
		return err
	}

	if or.blockchain == nil {
		biType := config.GetString(config.BlockchainType)
		if or.blockchain, err = bifactory.GetPlugin(ctx, biType); err != nil {
			return err
		}
	}
	if err = or.blockchain.Init(ctx, blockchainConfig.SubPrefix(or.blockchain.Name()), &or.bc); err != nil {
		return err
	}

	if or.publicstorage == nil {
		psType := config.GetString(config.PublicStorageType)
		if or.publicstorage, err = psfactory.GetPlugin(ctx, psType); err != nil {
			return err
		}
	}
	if err = or.publicstorage.Init(ctx, publicstorageConfig.SubPrefix(or.publicstorage.Name()), or); err != nil {
		return err
	}

	if or.dataexchange == nil {
		dxType := config.GetString(config.DataexchangeType)
		if or.dataexchange, err = dxfactory.GetPlugin(ctx, dxType); err != nil {
			return err
		}
	}
	if err = or.dataexchange.Init(ctx, dataexchangeConfig.SubPrefix(or.dataexchange.Name()), &or.bc); err != nil {
		return err
	}

	if or.tokens == nil {
		or.tokens = make(map[string]tokens.Plugin)
		tokensConfigArraySize := tokensConfig.ArraySize()
		for i := 0; i < tokensConfigArraySize; i++ {
			prefix := tokensConfig.ArrayEntry(i)
			name := prefix.GetString(tokens.TokensConfigName)
			pluginName := prefix.GetString(tokens.TokensConfigPlugin)
			if name == "" {
				return i18n.NewError(ctx, i18n.MsgMissingTokensPluginConfig)
			}
			if err = fftypes.ValidateFFNameField(ctx, name, "name"); err != nil {
				return err
			}
			if pluginName == "" {
				// Migration path for old config key
				// TODO: eventually make this fatal
				pluginName = prefix.GetString(tokens.TokensConfigConnector)
				if pluginName == "" {
					return i18n.NewError(ctx, i18n.MsgMissingTokensPluginConfig)
				}
				log.L(ctx).Warnf("Your tokens config uses the deprecated 'connector' key - please change to 'plugin' instead")
			}
			if pluginName == "https" {
				// Migration path for old plugin name
				// TODO: eventually make this fatal
				log.L(ctx).Warnf("Your tokens config uses the old plugin name 'https' - this plugin has been renamed to 'fftokens'")
				pluginName = "fftokens"
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
	}

	return nil
}

func (or *orchestrator) initComponents(ctx context.Context) (err error) {

	if or.identity == nil {
		or.identity, err = identity.NewIdentityManager(ctx, or.database, or.identityPlugin, or.blockchain)
		if err != nil {
			return err
		}
	}

	if or.data == nil {
		or.data, err = data.NewDataManager(ctx, or.database, or.publicstorage, or.dataexchange)
		if err != nil {
			return err
		}
	}

	if or.batch == nil {
		or.batch, err = batch.NewBatchManager(ctx, or.database, or.data)
		if err != nil {
			return err
		}
	}

	or.syncasync = syncasync.NewSyncAsyncBridge(ctx, or.database, or.data)
	or.batchpin = batchpin.NewBatchPinSubmitter(or.database, or.identity, or.blockchain)

	if or.messaging == nil {
		if or.messaging, err = privatemessaging.NewPrivateMessaging(ctx, or.database, or.identity, or.dataexchange, or.blockchain, or.batch, or.data, or.syncasync, or.batchpin); err != nil {
			return err
		}
	}

	if or.broadcast == nil {
		if or.broadcast, err = broadcast.NewBroadcastManager(ctx, or.database, or.identity, or.data, or.blockchain, or.dataexchange, or.publicstorage, or.batch, or.syncasync, or.batchpin); err != nil {
			return err
		}
	}

	if or.assets == nil {
		or.assets, err = assets.NewAssetManager(ctx, or.database, or.identity, or.data, or.syncasync, or.broadcast, or.messaging, or.tokens)
		if err != nil {
			return err
		}
	}

	or.definitions = definitions.NewDefinitionHandlers(or.database, or.dataexchange, or.data, or.broadcast, or.messaging, or.assets)

	if or.events == nil {
		or.events, err = events.NewEventManager(ctx, or.publicstorage, or.database, or.identity, or.definitions, or.data, or.broadcast, or.messaging)
		if err != nil {
			return err
		}
	}

	if or.networkmap == nil {
		or.networkmap, err = networkmap.NewNetworkMap(ctx, or.database, or.broadcast, or.dataexchange, or.identity)
		if err != nil {
			return err
		}
	}

	or.syncasync.Init(or.events)

	or.contracts = contracts.NewContractManager(or.database, or.publicstorage, or.broadcast, or.identity, or.blockchain)

	return nil
}

func (or *orchestrator) getPrefdefinedNamespaces(ctx context.Context) ([]*fftypes.Namespace, error) {
	defaultNS := config.GetString(config.NamespacesDefault)
	predefined := config.GetObjectArray(config.NamespacesPredefined)
	namespaces := []*fftypes.Namespace{
		{
			Name:        fftypes.SystemNamespace,
			Type:        fftypes.NamespaceTypeSystem,
			Description: i18n.Expand(ctx, i18n.MsgSystemNSDescription),
		},
	}
	foundDefault := false
	for i, nsObject := range predefined {
		name := nsObject.GetString("name")
		err := fftypes.ValidateFFNameField(ctx, name, fmt.Sprintf("namespaces.predefined[%d].name", i))
		if err != nil {
			return nil, err
		}
		foundDefault = foundDefault || name == defaultNS
		description := nsObject.GetString("description")
		dup := false
		for _, existing := range namespaces {
			if existing.Name == name {
				log.L(ctx).Warnf("Duplicate predefined namespace (ignored): %s", name)
				dup = true
			}
		}
		if !dup {
			namespaces = append(namespaces, &fftypes.Namespace{
				Type:        fftypes.NamespaceTypeLocal,
				Name:        name,
				Description: description,
			})
		}
	}
	if !foundDefault {
		return nil, i18n.NewError(ctx, i18n.MsgDefaultNamespaceNotFound, defaultNS)
	}
	return namespaces, nil
}

func (or *orchestrator) initNamespaces(ctx context.Context) error {
	predefined, err := or.getPrefdefinedNamespaces(ctx)
	if err != nil {
		return err
	}
	for _, newNS := range predefined {
		ns, err := or.database.GetNamespace(ctx, newNS.Name)
		if err != nil {
			return err
		}
		var updated bool
		if ns == nil {
			updated = true
			newNS.ID = fftypes.NewUUID()
			newNS.Created = fftypes.Now()
		} else {
			// Only update if the description has changed, and the one in our DB is locally defined
			updated = ns.Description != newNS.Description && ns.Type == fftypes.NamespaceTypeLocal
		}
		if updated {
			if err := or.database.UpsertNamespace(ctx, newNS, true); err != nil {
				return err
			}
		}
	}
	return nil
}

func (or *orchestrator) AddContractInterface(ctx context.Context, ns string, ffi *fftypes.FFI, waitConfirm bool) (output *fftypes.FFI, err error) {
	return or.contracts.BroadcastContractInterface(ctx, ns, ffi, waitConfirm)
}

func (or *orchestrator) GetContractInterfaces(ctx context.Context, ns string, filter database.AndFilter) (output []*fftypes.FFI, res *database.FilterResult, err error) {
	return or.contracts.GetContractInterfaces(ctx, ns, filter)
}

func (or *orchestrator) GetContractInterfaceByID(ctx context.Context, id string) (output *fftypes.FFI, err error) {
	return or.contracts.GetContractInterfaceByID(ctx, id)
}

func (or *orchestrator) GetContractInterfaceByNameAndVersion(ctx context.Context, ns, name, version string) (output *fftypes.FFI, err error) {
	return or.contracts.GetContractInterfaceByNameAndVersion(ctx, ns, name, version)
}

// func (or *orchestrator) AddContractInstance(ctx context.Context, ns string, ci *fftypes.ContractInstance, waitConfirm bool) (output *fftypes.ContractInstance, err error) {
// 	return or.contracts.AddContractInstance(ctx, ns, ci, waitConfirm)
// }

// func (or *orchestrator) GetContractInstances(ctx context.Context, ns string, filter database.AndFilter) (output []*fftypes.ContractInstance, res *database.FilterResult, err error) {
// 	return or.contracts.GetContractInstances(ctx, ns, filter)
// }

// func (or *orchestrator) GetContractInstanceByNameOrID(ctx context.Context, ns, nameOrID string) (output *fftypes.ContractInstance, err error) {
// 	return or.contracts.GetContractInstanceByNameOrID(ctx, ns, nameOrID)
// }

func (or *orchestrator) InvokeContract(ctx context.Context, ns, contractInstanceNameOrId, method string, req *fftypes.ContractInvocationRequest) (interface{}, error) {
	return or.contracts.InvokeContract(ctx, ns, contractInstanceNameOrId, method, req)
}
