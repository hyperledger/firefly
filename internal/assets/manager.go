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

	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

type Manager interface {
	CreateTokenPool(ctx context.Context, ns, typeName string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error)
	CreateTokenPoolWithID(ctx context.Context, ns string, id *fftypes.UUID, typeName string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error)
	GetTokenPools(ctx context.Context, ns, typeName string, filter database.AndFilter) ([]*fftypes.TokenPool, *database.FilterResult, error)
	GetTokenPool(ctx context.Context, ns, typeName, name string) (*fftypes.TokenPool, error)
	GetTokenAccounts(ctx context.Context, ns, typeName, name string, filter database.AndFilter) ([]*fftypes.TokenAccount, *database.FilterResult, error)
	Start() error
	WaitStop()
}

type assetManager struct {
	ctx       context.Context
	database  database.Plugin
	identity  identity.Manager
	data      data.Manager
	syncasync syncasync.Bridge
	tokens    map[string]tokens.Plugin
}

func NewAssetManager(ctx context.Context, di database.Plugin, im identity.Manager, dm data.Manager, sa syncasync.Bridge, ti map[string]tokens.Plugin) (Manager, error) {
	if di == nil || im == nil || sa == nil || ti == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	am := &assetManager{
		ctx:       ctx,
		database:  di,
		identity:  im,
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
	if err := am.data.VerifyNamespaceExists(ctx, ns); err != nil {
		return nil, err
	}

	err := am.identity.ResolveInputIdentity(ctx, &pool.Identity)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgAuthorInvalid)
	}

	plugin, err := am.selectTokenPlugin(ctx, typeName)
	if err != nil {
		return nil, err
	}

	if waitConfirm {
		return am.syncasync.SendConfirmTokenPool(ctx, ns, func(requestID *fftypes.UUID) error {
			_, err := am.CreateTokenPoolWithID(ctx, ns, requestID, typeName, pool, false)
			return err
		})
	}

	tx := &fftypes.Transaction{
		ID: fftypes.NewUUID(),
		Subject: fftypes.TransactionSubject{
			Namespace: ns,
			Type:      fftypes.TransactionTypeTokenPool,
			Signer:    pool.Key,
			Reference: id,
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
		tx.ID,
		"",
		fftypes.OpTypeTokensCreatePool,
		fftypes.OpStatusPending,
		pool.Author)
	err = am.database.UpsertOperation(ctx, op, false)
	if err != nil {
		return nil, err
	}

	pool.ID = id
	pool.Namespace = ns
	pool.TX = fftypes.TransactionRef{
		ID:   tx.ID,
		Type: tx.Subject.Type,
	}
	return pool, plugin.CreateTokenPool(ctx, pool)
}

func (am *assetManager) scopeNS(ns string, filter database.AndFilter) database.AndFilter {
	return filter.Condition(filter.Builder().Eq("namespace", ns))
}

func (am *assetManager) GetTokenPools(ctx context.Context, ns string, typeName string, filter database.AndFilter) ([]*fftypes.TokenPool, *database.FilterResult, error) {
	if _, err := am.selectTokenPlugin(ctx, typeName); err != nil {
		return nil, nil, err
	}
	if err := fftypes.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
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

func (am *assetManager) GetTokenAccounts(ctx context.Context, ns, typeName, name string, filter database.AndFilter) ([]*fftypes.TokenAccount, *database.FilterResult, error) {
	pool, err := am.GetTokenPool(ctx, ns, typeName, name)
	if err != nil {
		return nil, nil, err
	}
	return am.database.GetTokenAccounts(ctx, filter.Condition(filter.Builder().Eq("poolid", pool.ID)))
}

func (am *assetManager) Start() error {
	return nil
}

func (am *assetManager) WaitStop() {
	// No go routines
}
