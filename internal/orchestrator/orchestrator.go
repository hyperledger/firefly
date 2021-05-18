// Copyright © 2021 Kaleido, Inc.
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

	"github.com/kaleido-io/firefly/internal/batch"
	"github.com/kaleido-io/firefly/pkg/blockchain"
	"github.com/kaleido-io/firefly/internal/blockchain/blockchainfactory"
	"github.com/kaleido-io/firefly/internal/broadcast"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/database/databasefactory"
	"github.com/kaleido-io/firefly/internal/events"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/publicstorage"
	"github.com/kaleido-io/firefly/internal/publicstorage/publicstoragefactory"
	"github.com/kaleido-io/firefly/pkg/database"
)

var (
	blockchainConfig    = config.NewPluginConfig("blockchain")
	databaseConfig      = config.NewPluginConfig("database")
	publicstorageConfig = config.NewPluginConfig("publicstorage")
)

// Orchestrator is the main interface behind the API, implementing the actions
type Orchestrator interface {
	blockchain.Events

	Init(ctx context.Context) error
	Start() error
	WaitStop() // The close itself is performed by canceling the context

	// Definitions
	BroadcastDataDefinition(ctx context.Context, ns string, s *fftypes.DataDefinition) (*fftypes.Message, error)

	// Data Query
	GetNamespace(ctx context.Context, ns string) (*fftypes.Namespace, error)
	GetNamespaces(ctx context.Context, filter database.AndFilter) ([]*fftypes.Namespace, error)
	GetTransactionById(ctx context.Context, ns, id string) (*fftypes.Transaction, error)
	GetTransactions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Transaction, error)
	GetMessageById(ctx context.Context, ns, id string) (*fftypes.Message, error)
	GetMessages(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Message, error)
	GetMessageOperations(ctx context.Context, ns, id string, filter database.AndFilter) ([]*fftypes.Operation, error)
	GetMessageEvents(ctx context.Context, ns, id string, filter database.AndFilter) ([]*fftypes.Event, error)
	GetMessagesForData(ctx context.Context, ns, dataId string, filter database.AndFilter) ([]*fftypes.Message, error)
	GetBatchById(ctx context.Context, ns, id string) (*fftypes.Batch, error)
	GetBatches(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Batch, error)
	GetDataById(ctx context.Context, ns, id string) (*fftypes.Data, error)
	GetData(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Data, error)
	GetDataDefinitionById(ctx context.Context, ns, id string) (*fftypes.DataDefinition, error)
	GetDataDefinitions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.DataDefinition, error)
	GetOperationById(ctx context.Context, ns, id string) (*fftypes.Operation, error)
	GetOperations(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Operation, error)
	GetEventById(ctx context.Context, ns, id string) (*fftypes.Event, error)
	GetEvents(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Event, error)
}

type orchestrator struct {
	ctx           context.Context
	started       bool
	database      database.Plugin
	blockchain    blockchain.Plugin
	publicstorage publicstorage.Plugin
	events        events.EventManager
	batch         batch.BatchManager
	broadcast     broadcast.BroadcastManager
	nodeIdentity  string
}

func NewOrchestrator() Orchestrator {
	or := &orchestrator{}

	// Initialize the config on all the factories
	blockchainfactory.InitConfigPrefix(blockchainConfig)
	databasefactory.InitConfigPrefix(databaseConfig)
	publicstoragefactory.InitConfigPrefix(publicstorageConfig)

	return or
}

func (or *orchestrator) Init(ctx context.Context) (err error) {
	or.ctx = ctx
	err = or.initPlugins(ctx)
	if err == nil {
		err = or.initComponents(ctx)
	}
	if err == nil {
		err = or.initNamespaces(ctx)
	}
	return err
}

func (or *orchestrator) Start() error {
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
}

func (or *orchestrator) initPlugins(ctx context.Context) (err error) {

	if or.database == nil {
		if or.database, err = or.initDatabasePlugin(ctx); err != nil {
			return err
		}
	}

	if or.blockchain == nil {
		if or.blockchain, err = or.initBlockchainPlugin(ctx); err != nil {
			return err
		}
	}

	if or.publicstorage == nil {
		if or.publicstorage, err = or.initPublicStoragePlugin(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (or *orchestrator) initComponents(ctx context.Context) (err error) {
	if or.events == nil {
		or.events = events.NewEventManager(ctx, or.publicstorage, or.database)
	}

	if or.batch == nil {
		or.batch, err = batch.NewBatchManager(ctx, or.database)
		if err != nil {
			return err
		}
	}

	if or.broadcast == nil {
		if or.broadcast, err = broadcast.NewBroadcastManager(ctx, or.database, or.blockchain, or.publicstorage, or.batch); err != nil {
			return err
		}
	}
	return nil
}

func (or *orchestrator) initBlockchainPlugin(ctx context.Context) (blockchain.Plugin, error) {
	pluginType := config.GetString(config.BlockchainType)
	plugin, err := blockchainfactory.GetPlugin(ctx, pluginType)
	if err != nil {
		return nil, err
	}
	err = plugin.Init(ctx, blockchainConfig.SubPrefix(pluginType), or)
	if err == nil {
		suppliedIdentity := config.GetString(config.NodeIdentity)
		or.nodeIdentity, err = plugin.VerifyIdentitySyntax(ctx, suppliedIdentity)
		if err != nil {
			log.L(ctx).Errorf("Invalid node identity: %s", suppliedIdentity)
		}
	}
	return plugin, err
}

func (or *orchestrator) initDatabasePlugin(ctx context.Context) (database.Plugin, error) {
	pluginType := config.GetString(config.DatabaseType)
	plugin, err := databasefactory.GetPlugin(ctx, pluginType)
	if err != nil {
		return nil, err
	}
	err = plugin.Init(ctx, databaseConfig.SubPrefix(pluginType), or)
	return plugin, err
}

func (or *orchestrator) initPublicStoragePlugin(ctx context.Context) (publicstorage.Plugin, error) {
	pluginType := config.GetString(config.PublicStorageType)
	plugin, err := publicstoragefactory.GetPlugin(ctx, pluginType)
	if err != nil {
		return nil, err
	}
	err = plugin.Init(ctx, publicstorageConfig.SubPrefix(pluginType), or)
	return plugin, err
}

func (or *orchestrator) initNamespaces(ctx context.Context) error {
	defaultNS := config.GetString(config.NamespacesDefault)
	predefined := config.GetObjectArray(config.NamespacesPredefined)
	foundDefault := false
	for i, nsObject := range predefined {
		name := nsObject.GetString(ctx, "name")
		description := nsObject.GetString(ctx, "description")
		err := fftypes.ValidateFFNameField(ctx, name, fmt.Sprintf("namespaces.predefined[%d].name", i))
		if err != nil {
			return err
		}
		foundDefault = foundDefault || name == defaultNS

		ns, err := or.database.GetNamespace(ctx, name)
		if err != nil {
			return err
		}
		var updated bool
		if ns == nil {
			updated = true
			ns = &fftypes.Namespace{
				ID:          fftypes.NewUUID(),
				Name:        name,
				Description: description,
				Created:     fftypes.Now(),
			}
		} else {
			// Only update if the description has changed, and the one in our DB is locally defined
			updated = ns.Description != description && ns.Type == fftypes.NamespaceTypeStaticLocal
			ns.Description = description
		}
		if updated {
			if err := or.database.UpsertNamespace(ctx, ns, true); err != nil {
				return err
			}
		}
	}
	if !foundDefault {
		return i18n.NewError(ctx, i18n.MsgDefaultNamespaceNotFound, defaultNS)
	}
	return nil
}
