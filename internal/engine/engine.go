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

package engine

import (
	"context"

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

// Engine is the main interface behind the API, implementing the actions
type Engine interface {
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
	GetBatchById(ctx context.Context, ns, id string) (*fftypes.Batch, error)
	GetBatches(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Batch, error)
	GetDataById(ctx context.Context, ns, id string) (*fftypes.Data, error)
	GetData(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Data, error)
	GetDataDefinitionById(ctx context.Context, ns, id string) (*fftypes.DataDefinition, error)
	GetDataDefinitions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.DataDefinition, error)
}

type engine struct {
	ctx              context.Context
	database         database.Plugin
	blockchain       blockchain.Plugin
	p2pfs            p2pfs.Plugin
	blockchainEvents *blockchainEvents
	batch            batching.BatchManager
	broadcast        broadcast.Broadcast
	nodeIdentity     string
}

func NewEngine() Engine {
	e := &engine{}

	// Initialize the config on all the factories
	blockchainfactory.InitConfigPrefix(blockchainConfig)
	databasefactory.InitConfigPrefix(databaseConfig)
	p2pfsfactory.InitConfigPrefix(p2pfsConfig)

	return e
}

func (e *engine) Init(ctx context.Context) (err error) {
	e.ctx = ctx
	e.blockchainEvents = &blockchainEvents{ctx, e}
	err = e.initPlugins(ctx)
	if err == nil {
		err = e.initComponents(ctx)
	}
	return err
}

func (e *engine) Start() error {
	return e.batch.Start()
}

func (e *engine) Close() {
	if e.batch != nil {
		e.batch.Close()
		e.batch = nil
	}
	if e.broadcast != nil {
		e.broadcast.Close()
		e.broadcast = nil
	}
}

func (e *engine) initPlugins(ctx context.Context) (err error) {

	if e.database == nil {
		if e.database, err = e.initDatabasePlugin(ctx); err != nil {
			return err
		}
	}

	if e.blockchain == nil {
		if e.blockchain, err = e.initBlockchainPlugin(ctx); err != nil {
			return err
		}
	}

	if e.p2pfs == nil {
		if e.p2pfs, err = e.initP2PFilesystemPlugin(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (e *engine) initComponents(ctx context.Context) (err error) {
	if e.batch == nil {
		e.batch, err = batching.NewBatchManager(ctx, e.database)
		if err != nil {
			return err
		}
	}

	if e.broadcast == nil {
		if e.broadcast, err = broadcast.NewBroadcast(ctx, e.database, e.blockchain, e.p2pfs, e.batch); err != nil {
			return err
		}
	}
	return nil
}

func (e *engine) initBlockchainPlugin(ctx context.Context) (blockchain.Plugin, error) {
	pluginType := config.GetString(config.BlockchainType)
	blockchain, err := blockchainfactory.GetPlugin(ctx, pluginType)
	if err != nil {
		return nil, err
	}
	err = blockchain.Init(ctx, blockchainConfig.SubPrefix(pluginType), e.blockchainEvents)
	if err == nil {
		suppliedIdentity := config.GetString(config.NodeIdentity)
		e.nodeIdentity, err = blockchain.VerifyIdentitySyntax(ctx, suppliedIdentity)
		if err != nil {
			log.L(ctx).Errorf("Invalid node identity: %s", suppliedIdentity)
		}
	}
	return blockchain, err
}

func (e *engine) initDatabasePlugin(ctx context.Context) (database.Plugin, error) {
	pluginType := config.GetString(config.DatabaseType)
	database, err := databasefactory.GetPlugin(ctx, pluginType)
	if err != nil {
		return nil, err
	}
	err = database.Init(ctx, databaseConfig.SubPrefix(pluginType), e)
	return database, err
}

func (e *engine) initP2PFilesystemPlugin(ctx context.Context) (p2pfs.Plugin, error) {
	pluginType := config.GetString(config.P2PFSType)
	p2pfs, err := p2pfsfactory.GetPlugin(ctx, pluginType)
	if err != nil {
		return nil, err
	}
	err = p2pfs.Init(ctx, p2pfsConfig.SubPrefix(pluginType), e)
	return p2pfs, err
}
