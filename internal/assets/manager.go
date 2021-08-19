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

package assets

import (
	"context"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/data"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/identity"
	"github.com/hyperledger-labs/firefly/pkg/tokens"
)

type Manager interface {
	CreateTokenPool(ctx context.Context, ns string, connector string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error)
	Start() error
	WaitStop()
}

type assetManager struct {
	ctx      context.Context
	database database.Plugin
	identity identity.Plugin
	data     data.Manager
	tokens   map[string]tokens.Plugin
}

func NewAssetManager(ctx context.Context, di database.Plugin, ii identity.Plugin, dm data.Manager, tk map[string]tokens.Plugin) (Manager, error) {
	if di == nil || ii == nil || tk == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	am := &assetManager{
		ctx:      ctx,
		database: di,
		identity: ii,
		data:     dm,
		tokens:   tk,
	}
	return am, nil
}

func (am *assetManager) selectTokenPlugin(ctx context.Context, name string) (tokens.Plugin, error) {
	for pluginName, plugin := range am.tokens {
		if pluginName == name {
			return plugin, nil
		}
	}
	return nil, i18n.NewError(ctx, i18n.MsgUnknownTokensPlugin, name)
}

func (am *assetManager) CreateTokenPool(ctx context.Context, ns string, connector string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error) {
	pool.ID = fftypes.NewUUID()
	pool.Namespace = ns

	if err := am.data.VerifyNamespaceExists(ctx, ns); err != nil {
		return nil, err
	}

	if pool.Author == "" {
		pool.Author = config.GetString(config.OrgIdentity)
	}
	id, err := am.identity.Resolve(ctx, pool.Author)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgAuthorInvalid)
	}

	plugin, err := am.selectTokenPlugin(ctx, connector)
	if err != nil {
		return nil, err
	}

	trackingID, err := plugin.CreateTokenPool(ctx, id, pool)
	if err != nil {
		return nil, err
	}

	op := fftypes.NewTXOperation(
		plugin,
		ns,
		fftypes.NewUUID(),
		trackingID,
		fftypes.OpTypeTokensCreatePool,
		fftypes.OpStatusPending,
		id.Identifier)
	return pool, am.database.UpsertOperation(ctx, op, false)
}

func (am *assetManager) Start() error {
	return nil
}

func (am *assetManager) WaitStop() {
	// No go routines
}
