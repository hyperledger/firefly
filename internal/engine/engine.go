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
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/persistence"
	"github.com/kaleido-io/firefly/internal/persistence/persistencefactory"
)

// Engine is the main interface behind the API, implementing the actions
type Engine interface {
	Close()
}

type engine struct {
	persistence      persistence.Plugin
	persistenceCap   *persistence.Capabilities
	blockchain       blockchain.Plugin
	blockchainCap    *blockchain.Capabilities
	blockchainEvents *blockchainEvents
	batch            batching.BatchManager
	broadcast        broadcast.Broadcast
}

func NewEngine(ctx context.Context, init bool) (Engine, error) {
	e := &engine{}
	e.blockchainEvents = &blockchainEvents{ctx, e}

	if init {

		if err := e.initPlugins(ctx); err != nil {
			return nil, err
		}

		if err := e.initComponents(ctx); err != nil {
			return nil, err
		}
	}

	return e, nil
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
	if err = e.initPersistencePlugin(ctx); err != nil {
		return err
	}

	if err = e.initBlockchainPlugin(ctx); err != nil {
		return err
	}

	return nil
}

func (e *engine) initComponents(ctx context.Context) (err error) {
	if e.batch, err = batching.NewBatchManager(ctx, e.persistence); err != nil {
		return err
	}

	if e.broadcast, err = broadcast.NewBroadcast(ctx, e.persistence, e.blockchain, e.batch); err != nil {
		return err
	}
	return nil
}

func (e *engine) initBlockchainPlugin(ctx context.Context) (err error) {
	pluginType := config.GetString(config.BlockchainType)
	e.blockchain, err = blockchainfactory.GetPlugin(ctx, pluginType)
	if err != nil {
		return err
	}
	conf := e.blockchain.ConfigInterface()
	err = config.UnmarshalKey(config.Blockchain, &conf)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgConfigFailed, config.Blockchain)
	}
	e.blockchainCap, err = e.blockchain.Init(ctx, conf, e.blockchainEvents)
	if err != nil {
		return err
	}
	return nil
}

func (e *engine) initPersistencePlugin(ctx context.Context) (err error) {
	pluginType := config.GetString(config.PersistenceType)
	e.persistence, err = persistencefactory.GetPlugin(ctx, pluginType)
	if err != nil {
		return err
	}
	conf := e.persistence.ConfigInterface()
	err = config.UnmarshalKey(config.Persistence, &conf)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgConfigFailed, config.Persistence)
	}
	e.persistenceCap, err = e.persistence.Init(ctx, conf, e)
	if err != nil {
		return err
	}
	return nil
}
