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
	"github.com/hyperledger-labs/firefly/internal/syncasync"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/identity"
	"github.com/hyperledger-labs/firefly/pkg/tokens"
)

type Manager interface {
	CreateTokenPool(ctx context.Context, ns string, typeName string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error)
	CreateTokenPoolWithID(ctx context.Context, ns string, id *fftypes.UUID, typeName string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error)
	GetTokenPools(ctx context.Context, ns string, typeName string, filter database.AndFilter) ([]*fftypes.TokenPool, *database.FilterResult, error)
	GetTokenPool(ctx context.Context, ns string, typeName string, name string) (*fftypes.TokenPool, error)
	Start() error
	WaitStop()
}

type assetManager struct {
	ctx       context.Context
	database  database.Plugin
	identity  identity.Plugin
	data      data.Manager
	syncasync syncasync.Bridge
	tokens    map[string]tokens.Plugin
}

func NewAssetManager(ctx context.Context, di database.Plugin, ii identity.Plugin, dm data.Manager, sa syncasync.Bridge, ti map[string]tokens.Plugin) (Manager, error) {
	if di == nil || ii == nil || sa == nil || ti == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	am := &assetManager{
		ctx:       ctx,
		database:  di,
		identity:  ii,
		data:      dm,
		syncasync: sa,
		tokens:    ti,
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

func (am *assetManager) CreateTokenPool(ctx context.Context, ns string, typeName string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error) {
	return am.CreateTokenPoolWithID(ctx, ns, fftypes.NewUUID(), typeName, pool, waitConfirm)
}

func (am *assetManager) CreateTokenPoolWithID(ctx context.Context, ns string, id *fftypes.UUID, typeName string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error) {
	pool.ID = id
	pool.Namespace = ns

	if err := am.data.VerifyNamespaceExists(ctx, ns); err != nil {
		return nil, err
	}

	if pool.Author == "" {
		pool.Author = config.GetString(config.OrgIdentity)
	}
	author, err := am.identity.Resolve(ctx, pool.Author)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgAuthorInvalid)
	}

	plugin, err := am.selectTokenPlugin(ctx, typeName)
	if err != nil {
		return nil, err
	}

	if waitConfirm {
		return am.syncasync.SendConfirmTokenPool(ctx, pool.Namespace, func(requestID *fftypes.UUID) error {
			_, err := am.CreateTokenPoolWithID(ctx, ns, requestID, typeName, pool, false)
			return err
		})
	}

	pool.TX = fftypes.TransactionRef{
		ID:   fftypes.NewUUID(),
		Type: fftypes.TransactionTypeTokenPool,
	}
	trackingID, err := plugin.CreateTokenPool(ctx, author, pool)
	if err != nil {
		return nil, err
	}

	tx := &fftypes.Transaction{
		ID: pool.TX.ID,
		Subject: fftypes.TransactionSubject{
			Namespace: pool.Namespace,
			Type:      pool.TX.Type,
			Signer:    author.OnChain, // The transaction records on the on-chain identity
			Reference: pool.ID,
		},
		Created: fftypes.Now(),
		Status:  fftypes.OpStatusPending,
	}
	tx.Hash = tx.Subject.Hash()
	err = am.database.UpsertTransaction(ctx, tx, false /* should be new, or idempotent replay */)
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
		author.Identifier)
	return pool, am.database.UpsertOperation(ctx, op, false)
}

func (am *assetManager) scopeNS(ns string, filter database.AndFilter) database.AndFilter {
	return filter.Condition(filter.Builder().Eq("namespace", ns))
}

func (am *assetManager) GetTokenPools(ctx context.Context, ns string, typeName string, filter database.AndFilter) ([]*fftypes.TokenPool, *database.FilterResult, error) {
	if _, err := am.selectTokenPlugin(ctx, typeName); err != nil {
		return nil, nil, err
	}
	return am.database.GetTokenPools(ctx, am.scopeNS(ns, filter))
}

func (am *assetManager) GetTokenPool(ctx context.Context, ns, typeName, name string) (*fftypes.TokenPool, error) {
	if _, err := am.selectTokenPlugin(ctx, typeName); err != nil {
		return nil, err
	}
	if err := fftypes.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
		return nil, err
	}
	if err := fftypes.ValidateFFNameField(ctx, name, "name"); err != nil {
		return nil, err
	}
	return am.database.GetTokenPool(ctx, ns, name)
}

func (am *assetManager) Start() error {
	return nil
}

func (am *assetManager) WaitStop() {
	// No go routines
}
