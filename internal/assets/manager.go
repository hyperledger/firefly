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

	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/privatemessaging"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

type Manager interface {
	CreateTokenPool(ctx context.Context, ns, connector string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error)
	GetTokenPools(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.TokenPool, *database.FilterResult, error)
	GetTokenPoolsByType(ctx context.Context, ns, connector string, filter database.AndFilter) ([]*fftypes.TokenPool, *database.FilterResult, error)
	GetTokenPool(ctx context.Context, ns, connector, poolName string) (*fftypes.TokenPool, error)
	GetTokenPoolByNameOrID(ctx context.Context, ns string, poolNameOrID string) (*fftypes.TokenPool, error)
	ValidateTokenPoolTx(ctx context.Context, pool *fftypes.TokenPool, protocolTxID string) error
	GetTokenAccounts(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.TokenAccount, *database.FilterResult, error)
	GetTokenAccountsByPool(ctx context.Context, ns, connector, poolName string, filter database.AndFilter) ([]*fftypes.TokenAccount, *database.FilterResult, error)
	GetTokenTransfers(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.TokenTransfer, *database.FilterResult, error)
	GetTokenTransferByID(ctx context.Context, ns, id string) (*fftypes.TokenTransfer, error)
	GetTokenTransfersByPool(ctx context.Context, ns, connector, poolName string, filter database.AndFilter) ([]*fftypes.TokenTransfer, *database.FilterResult, error)
	NewTransfer(ns, connector, poolName string, transfer *fftypes.TokenTransferInput) sysmessaging.MessageSender
	MintTokens(ctx context.Context, ns string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (*fftypes.TokenTransfer, error)
	MintTokensByType(ctx context.Context, ns, connector, poolName string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (*fftypes.TokenTransfer, error)
	BurnTokens(ctx context.Context, ns string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (*fftypes.TokenTransfer, error)
	BurnTokensByType(ctx context.Context, ns, connector, poolName string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (*fftypes.TokenTransfer, error)
	TransferTokens(ctx context.Context, ns string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (*fftypes.TokenTransfer, error)
	TransferTokensByType(ctx context.Context, ns, connector, poolName string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (*fftypes.TokenTransfer, error)
	GetTokenConnectors(ctx context.Context, ns string) ([]*fftypes.TokenConnector, error)

	// Bound token callbacks
	TokenPoolCreated(tk tokens.Plugin, pool *fftypes.TokenPool, protocolTxID string, additionalInfo fftypes.JSONObject) error

	Start() error
	WaitStop()
}

type assetManager struct {
	ctx       context.Context
	database  database.Plugin
	identity  identity.Manager
	data      data.Manager
	syncasync syncasync.Bridge
	broadcast broadcast.Manager
	messaging privatemessaging.Manager
	tokens    map[string]tokens.Plugin
	retry     retry.Retry
	txhelper  txcommon.Helper
}

func NewAssetManager(ctx context.Context, di database.Plugin, im identity.Manager, dm data.Manager, sa syncasync.Bridge, bm broadcast.Manager, pm privatemessaging.Manager, ti map[string]tokens.Plugin) (Manager, error) {
	if di == nil || im == nil || sa == nil || bm == nil || pm == nil || ti == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	am := &assetManager{
		ctx:       ctx,
		database:  di,
		identity:  im,
		data:      dm,
		syncasync: sa,
		broadcast: bm,
		messaging: pm,
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

func (am *assetManager) scopeNS(ns string, filter database.AndFilter) database.AndFilter {
	return filter.Condition(filter.Builder().Eq("namespace", ns))
}

func (am *assetManager) GetTokenAccounts(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.TokenAccount, *database.FilterResult, error) {
	return am.database.GetTokenAccounts(ctx, am.scopeNS(ns, filter))
}

func (am *assetManager) GetTokenAccountsByPool(ctx context.Context, ns, connector, poolName string, filter database.AndFilter) ([]*fftypes.TokenAccount, *database.FilterResult, error) {
	pool, err := am.GetTokenPool(ctx, ns, connector, poolName)
	if err != nil {
		return nil, nil, err
	}
	return am.database.GetTokenAccounts(ctx, filter.Condition(filter.Builder().Eq("poolprotocolid", pool.ProtocolID)))
}

func (am *assetManager) GetTokenConnectors(ctx context.Context, ns string) ([]*fftypes.TokenConnector, error) {
	if err := fftypes.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
		return nil, err
	}

	connectors := []*fftypes.TokenConnector{}
	for token := range am.tokens {
		connectors = append(
			connectors,
			&fftypes.TokenConnector{
				Name: token,
			},
		)
	}

	return connectors, nil
}

func (am *assetManager) Start() error {
	return nil
}

func (am *assetManager) WaitStop() {
	// No go routines
}
