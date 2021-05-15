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

	"github.com/kaleido-io/firefly/internal/aggregator"
	"github.com/kaleido-io/firefly/internal/batching"
	"github.com/kaleido-io/firefly/internal/blockchain"
	"github.com/kaleido-io/firefly/internal/blockchain/blockchainfactory"
	"github.com/kaleido-io/firefly/internal/broadcast"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/database/databasefactory"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/p2pfs"
	"github.com/kaleido-io/firefly/internal/p2pfs/p2pfsfactory"
)

var (
	blockchainConfig = config.NewPluginConfig("blockchain")
	databaseConfig   = config.NewPluginConfig("database")
	p2pfsConfig      = config.NewPluginConfig("p2pfs")
)

// Orchestrator is the main interface behind the API, implementing the actions
type Orchestrator interface {
	blockchain.Events

	Init(ctx context.Context) error
	Start() error
	Close()

	// Definitions
	BroadcastDataDefinition(ctx context.Context, ns string, s *fftypes.DataDefinition) (*fftypes.Message, error)

	// Data Query
	GetTransactionById(ctx context.Context, ns, id string) (*fftypes.Transaction, error)
	GetTransactions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Transaction, error)
	GetMessageById(ctx context.Context, ns, id string) (*fftypes.Message, error)
	GetMessages(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Message, error)
	GetMessageOperations(ctx context.Context, ns, id string, filter database.AndFilter) ([]*fftypes.Operation, error)
	GetBatchById(ctx context.Context, ns, id string) (*fftypes.Batch, error)
	GetBatches(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Batch, error)
	GetDataById(ctx context.Context, ns, id string) (*fftypes.Data, error)
	GetData(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Data, error)
	GetDataDefinitionById(ctx context.Context, ns, id string) (*fftypes.DataDefinition, error)
	GetDataDefinitions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.DataDefinition, error)
}

type orchestrator struct {
	ctx          context.Context
	database     database.Plugin
	blockchain   blockchain.Plugin
	p2pfs        p2pfs.Plugin
	aggregator   aggregator.Aggregator
	batch        batching.BatchManager
	broadcast    broadcast.BroadcastManager
	nodeIdentity string
}

func NewOrchestrator() Orchestrator {
	o := &orchestrator{}

	// Initialize the config on all the factories
	blockchainfactory.InitConfigPrefix(blockchainConfig)
	databasefactory.InitConfigPrefix(databaseConfig)
	p2pfsfactory.InitConfigPrefix(p2pfsConfig)

	return o
}

func (o *orchestrator) Init(ctx context.Context) (err error) {
	o.ctx = ctx
	err = o.initPlugins(ctx)
	if err == nil {
		err = o.initComponents(ctx)
	}
	return err
}

func (o *orchestrator) Start() error {
	err := o.blockchain.Start()
	if err == nil {
		err = o.batch.Start()
	}
	return err
}

func (o *orchestrator) Close() {
	if o.batch != nil {
		o.batch.Close()
		o.batch = nil
	}
	if o.broadcast != nil {
		o.broadcast.Close()
		o.broadcast = nil
	}
}

func (o *orchestrator) initPlugins(ctx context.Context) (err error) {

	if o.database == nil {
		if o.database, err = o.initDatabasePlugin(ctx); err != nil {
			return err
		}
	}

	if o.blockchain == nil {
		if o.blockchain, err = o.initBlockchainPlugin(ctx); err != nil {
			return err
		}
	}

	if o.p2pfs == nil {
		if o.p2pfs, err = o.initP2PFilesystemPlugin(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (o *orchestrator) initComponents(ctx context.Context) (err error) {
	if o.aggregator == nil {
		o.aggregator = aggregator.NewAggregator(ctx, o.p2pfs, o.database)
	}

	if o.batch == nil {
		o.batch, err = batching.NewBatchManager(ctx, o.database)
		if err != nil {
			return err
		}
	}

	if o.broadcast == nil {
		if o.broadcast, err = broadcast.NewBroadcastManager(ctx, o.database, o.blockchain, o.p2pfs, o.batch); err != nil {
			return err
		}
	}
	return nil
}

func (o *orchestrator) initBlockchainPlugin(ctx context.Context) (blockchain.Plugin, error) {
	pluginType := config.GetString(config.BlockchainType)
	plugin, err := blockchainfactory.GetPlugin(ctx, pluginType)
	if err != nil {
		return nil, err
	}
	err = plugin.Init(ctx, blockchainConfig.SubPrefix(pluginType), o)
	if err == nil {
		suppliedIdentity := config.GetString(config.NodeIdentity)
		o.nodeIdentity, err = plugin.VerifyIdentitySyntax(ctx, suppliedIdentity)
		if err != nil {
			log.L(ctx).Errorf("Invalid node identity: %s", suppliedIdentity)
		}
	}
	return plugin, err
}

func (o *orchestrator) initDatabasePlugin(ctx context.Context) (database.Plugin, error) {
	pluginType := config.GetString(config.DatabaseType)
	plugin, err := databasefactory.GetPlugin(ctx, pluginType)
	if err != nil {
		return nil, err
	}
	err = plugin.Init(ctx, databaseConfig.SubPrefix(pluginType), o)
	return plugin, err
}

func (o *orchestrator) initP2PFilesystemPlugin(ctx context.Context) (p2pfs.Plugin, error) {
	pluginType := config.GetString(config.P2PFSType)
	plugin, err := p2pfsfactory.GetPlugin(ctx, pluginType)
	if err != nil {
		return nil, err
	}
	err = plugin.Init(ctx, p2pfsConfig.SubPrefix(pluginType), o)
	return plugin, err
}
