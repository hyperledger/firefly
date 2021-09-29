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
	"fmt"

	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/identity"
	"github.com/hyperledger/firefly/pkg/tokens"
)

type Manager interface {
	CreateTokenPool(ctx context.Context, ns, typeName string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error)
	CreateTokenPoolWithID(ctx context.Context, ns string, id *fftypes.UUID, typeName string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error)
	GetTokenPools(ctx context.Context, ns, typeName string, filter database.AndFilter) ([]*fftypes.TokenPool, *database.FilterResult, error)
	GetTokenPool(ctx context.Context, ns, typeName, name string) (*fftypes.TokenPool, error)
	GetTokenAccounts(ctx context.Context, ns, typeName, name string, filter database.AndFilter) ([]*fftypes.TokenAccount, *database.FilterResult, error)
	ValidateTokenPoolTx(ctx context.Context, pool *fftypes.TokenPool, protocolTxID string) error

	// Bound token callbacks
	TokenPoolCreated(tk tokens.Plugin, tokenType fftypes.TokenType, tx *fftypes.UUID, protocolID, signingIdentity, protocolTxID string, additionalInfo fftypes.JSONObject) error

	Start() error
	WaitStop()
}

type assetManager struct {
	ctx       context.Context
	database  database.Plugin
	identity  identity.Plugin
	data      data.Manager
	syncasync syncasync.Bridge
	broadcast broadcast.Manager
	tokens    map[string]tokens.Plugin
	retry     retry.Retry
	txhelper  txcommon.Helper
}

func NewAssetManager(ctx context.Context, di database.Plugin, ii identity.Plugin, dm data.Manager, sa syncasync.Bridge, bm broadcast.Manager, ti map[string]tokens.Plugin) (Manager, error) {
	if di == nil || ii == nil || sa == nil || ti == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	am := &assetManager{
		ctx:       ctx,
		database:  di,
		identity:  ii,
		data:      dm,
		syncasync: sa,
		broadcast: bm,
		tokens:    ti,
		retry: retry.Retry{
			InitialDelay: config.GetDuration(config.AssetManagerRetryInitialDelay),
			MaximumDelay: config.GetDuration(config.AssetManagerRetryMaxDelay),
			Factor:       config.GetFloat64(config.AssetManagerRetryFactor),
		},
		txhelper: txcommon.NewTransactionHelper(di),
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

func storeTokenOpInputs(op *fftypes.Operation, pool *fftypes.TokenPool) {
	op.Input = fftypes.JSONObject{
		"id":        pool.ID.String(),
		"namespace": pool.Namespace,
		"name":      pool.Name,
		"config":    pool.Config,
	}
}

func retrieveTokenOpInputs(ctx context.Context, op *fftypes.Operation, pool *fftypes.TokenPool) (err error) {
	input := &op.Input
	pool.ID, err = fftypes.ParseUUID(ctx, input.GetString("id"))
	if err != nil {
		return err
	}
	pool.Namespace = input.GetString("namespace")
	pool.Name = input.GetString("name")
	if pool.Namespace == "" || pool.Name == "" {
		return fmt.Errorf("namespace or name missing from inputs")
	}
	pool.Config = input.GetObject("config")
	return nil
}

func (am *assetManager) CreateTokenPool(ctx context.Context, ns string, typeName string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error) {
	return am.CreateTokenPoolWithID(ctx, ns, fftypes.NewUUID(), typeName, pool, waitConfirm)
}

func (am *assetManager) CreateTokenPoolWithID(ctx context.Context, ns string, id *fftypes.UUID, typeName string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error) {
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
			Signer:    author.OnChain, // The transaction records on the on-chain identity
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

	pool.ID = id
	pool.Namespace = ns
	pool.TX = fftypes.TransactionRef{
		ID:   tx.ID,
		Type: tx.Subject.Type,
	}

	op := fftypes.NewTXOperation(
		plugin,
		ns,
		tx.ID,
		"",
		fftypes.OpTypeTokensCreatePool,
		fftypes.OpStatusPending,
		author.Identifier)
	storeTokenOpInputs(op, pool)
	err = am.database.UpsertOperation(ctx, op, false)
	if err != nil {
		return nil, err
	}

	return pool, plugin.CreateTokenPool(ctx, op.ID, author, pool)
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
	return am.database.GetTokenAccounts(ctx, filter.Condition(filter.Builder().Eq("protocolid", pool.ProtocolID)))
}

func (am *assetManager) ValidateTokenPoolTx(ctx context.Context, pool *fftypes.TokenPool, protocolTxID string) error {
	// TODO: validate that the given token pool was created with the given protocolTxId
	return nil
}

func (am *assetManager) Start() error {
	return nil
}

func (am *assetManager) WaitStop() {
	// No go routines
}
