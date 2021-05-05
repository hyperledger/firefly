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
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/p2pfs"
	"github.com/kaleido-io/firefly/internal/p2pfs/p2pfsfactory"
	"github.com/kaleido-io/firefly/internal/persistence"
	"github.com/kaleido-io/firefly/internal/persistence/persistencefactory"
)

// Engine is the main interface behind the API, implementing the actions
type Engine interface {
	Init(ctx context.Context) error
	Close()

	// Definitions
	BroadcastSchemaDefinition(ctx context.Context, s *fftypes.Schema) (*fftypes.Message, error)

	// Data Queryuery
	GetTransactionById(ctx context.Context, ns, id string) (*fftypes.Transaction, error)
	GetMessageById(ctx context.Context, ns, id string) (*fftypes.Message, error)
	GetBatchById(ctx context.Context, ns, id string) (*fftypes.Batch, error)
	GetDataById(ctx context.Context, ns, id string) (*fftypes.Data, error)
}

type engine struct {
	persistence      persistence.Plugin
	blockchain       blockchain.Plugin
	p2pfs            p2pfs.Plugin
	blockchainEvents *blockchainEvents
	batch            batching.BatchManager
	broadcast        broadcast.Broadcast
	nodeIdentity     string
}

func NewEngine() Engine {
	e := &engine{}
	return e
}

func (e *engine) Init(ctx context.Context) (err error) {
	e.blockchainEvents = &blockchainEvents{ctx, e}
	err = e.initPlugins(ctx)
	if err == nil {
		err = e.initComponents(ctx)
	}
	return err
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

	if e.persistence == nil {
		if e.persistence, err = e.initPersistencePlugin(ctx); err != nil {
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
		if e.batch, err = batching.NewBatchManager(ctx, e.persistence); err != nil {
			return err
		}
	}

	if e.broadcast == nil {
		if e.broadcast, err = broadcast.NewBroadcast(ctx, e.persistence, e.blockchain, e.p2pfs, e.batch); err != nil {
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
	conf := blockchain.ConfigInterface()
	err = config.UnmarshalKey(ctx, config.Blockchain, &conf)
	if err == nil {
		err = blockchain.Init(ctx, conf, e.blockchainEvents)
	}
	if err == nil {
		suppliedIdentity := config.GetString(config.NodeIdentity)
		e.nodeIdentity, err = blockchain.VerifyIdentitySyntax(ctx, suppliedIdentity)
		if err != nil {
			log.L(ctx).Errorf("Invalid node identity: %s", suppliedIdentity)
		}
	}
	return blockchain, err
}

func (e *engine) initPersistencePlugin(ctx context.Context) (persistence.Plugin, error) {
	pluginType := config.GetString(config.DatabaseType)
	persistence, err := persistencefactory.GetPlugin(ctx, pluginType)
	if err != nil {
		return nil, err
	}
	conf := persistence.ConfigInterface()
	err = config.UnmarshalKey(ctx, config.Database, &conf)
	if err == nil {
		err = persistence.Init(ctx, conf, e)
	}
	return persistence, err
}

func (e *engine) initP2PFilesystemPlugin(ctx context.Context) (p2pfs.Plugin, error) {
	pluginType := config.GetString(config.P2PFSType)
	p2pfs, err := p2pfsfactory.GetPlugin(ctx, pluginType)
	if err != nil {
		return nil, err
	}
	conf := p2pfs.ConfigInterface()
	err = config.UnmarshalKey(ctx, config.P2PFS, &conf)
	if err == nil {
		err = p2pfs.Init(ctx, conf, e)
	}
	return p2pfs, err
}
