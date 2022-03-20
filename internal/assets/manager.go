// Copyright © 2022 Kaleido, Inc.
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
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/internal/privatemessaging"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

type Manager interface {
	fftypes.Named

	CreateTokenPool(ctx context.Context, ns string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error)
	ActivateTokenPool(ctx context.Context, pool *fftypes.TokenPool, blockchainInfo fftypes.JSONObject) error
	GetTokenPools(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.TokenPool, *database.FilterResult, error)
	GetTokenPool(ctx context.Context, ns, connector, poolName string) (*fftypes.TokenPool, error)
	GetTokenPoolByNameOrID(ctx context.Context, ns string, poolNameOrID string) (*fftypes.TokenPool, error)

	GetTokenBalances(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.TokenBalance, *database.FilterResult, error)
	GetTokenAccounts(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.TokenAccount, *database.FilterResult, error)
	GetTokenAccountPools(ctx context.Context, ns, key string, filter database.AndFilter) ([]*fftypes.TokenAccountPool, *database.FilterResult, error)

	GetTokenTransfers(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.TokenTransfer, *database.FilterResult, error)
	GetTokenTransferByID(ctx context.Context, ns, id string) (*fftypes.TokenTransfer, error)

	NewTransfer(ns string, transfer *fftypes.TokenTransferInput) sysmessaging.MessageSender
	MintTokens(ctx context.Context, ns string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (*fftypes.TokenTransfer, error)
	BurnTokens(ctx context.Context, ns string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (*fftypes.TokenTransfer, error)
	TransferTokens(ctx context.Context, ns string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (*fftypes.TokenTransfer, error)

	GetTokenConnectors(ctx context.Context, ns string) ([]*fftypes.TokenConnector, error)

	NewApproval(ns string, approve *fftypes.TokenApprovalInput) sysmessaging.MessageSender
	TokenApproval(ctx context.Context, ns string, approval *fftypes.TokenApprovalInput, waitConfirm bool) (*fftypes.TokenApproval, error)
	GetTokenApprovals(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.TokenApproval, *database.FilterResult, error)

	// From operations.OperationHandler
	PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error)
	RunOperation(ctx context.Context, op *fftypes.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error)
}

type assetManager struct {
	ctx              context.Context
	database         database.Plugin
	txHelper         txcommon.Helper
	identity         identity.Manager
	data             data.Manager
	syncasync        syncasync.Bridge
	broadcast        broadcast.Manager
	messaging        privatemessaging.Manager
	tokens           map[string]tokens.Plugin
	metrics          metrics.Manager
	operations       operations.Manager
	keyNormalization int
}

func NewAssetManager(ctx context.Context, di database.Plugin, im identity.Manager, dm data.Manager, sa syncasync.Bridge, bm broadcast.Manager, pm privatemessaging.Manager, ti map[string]tokens.Plugin, mm metrics.Manager, om operations.Manager, txHelper txcommon.Helper) (Manager, error) {
	if di == nil || im == nil || sa == nil || bm == nil || pm == nil || ti == nil || mm == nil || om == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	am := &assetManager{
		ctx:              ctx,
		database:         di,
		txHelper:         txHelper,
		identity:         im,
		data:             dm,
		syncasync:        sa,
		broadcast:        bm,
		messaging:        pm,
		tokens:           ti,
		keyNormalization: identity.ParseKeyNormalizationConfig(config.GetString(config.AssetManagerKeyNormalization)),
		metrics:          mm,
		operations:       om,
	}
	om.RegisterHandler(ctx, am, []fftypes.OpType{
		fftypes.OpTypeTokenCreatePool,
		fftypes.OpTypeTokenActivatePool,
		fftypes.OpTypeTokenTransfer,
		fftypes.OpTypeTokenApproval,
	})
	return am, nil
}

func (am *assetManager) Name() string {
	return "AssetManager"
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

func (am *assetManager) GetTokenBalances(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.TokenBalance, *database.FilterResult, error) {
	return am.database.GetTokenBalances(ctx, am.scopeNS(ns, filter))
}

func (am *assetManager) GetTokenAccounts(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.TokenAccount, *database.FilterResult, error) {
	return am.database.GetTokenAccounts(ctx, am.scopeNS(ns, filter))
}

func (am *assetManager) GetTokenAccountPools(ctx context.Context, ns, key string, filter database.AndFilter) ([]*fftypes.TokenAccountPool, *database.FilterResult, error) {
	return am.database.GetTokenAccountPools(ctx, key, am.scopeNS(ns, filter))
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

func (am *assetManager) getTokenConnectorName(ctx context.Context, ns string) (string, error) {
	tokenConnectors, err := am.GetTokenConnectors(ctx, ns)
	if err != nil {
		return "", err
	}
	if len(tokenConnectors) != 1 {
		return "", i18n.NewError(ctx, i18n.MsgFieldNotSpecified, "connector")
	}
	return tokenConnectors[0].Name, nil
}

func (am *assetManager) getTokenPoolName(ctx context.Context, ns string) (string, error) {
	f := database.TokenPoolQueryFactory.NewFilter(ctx).And()
	f.Limit(1).Count(true)
	tokenPools, fr, err := am.GetTokenPools(ctx, ns, f)
	if err != nil {
		return "", err
	}
	if *fr.TotalCount != 1 {
		return "", i18n.NewError(ctx, i18n.MsgFieldNotSpecified, "pool")
	}
	return tokenPools[0].Name, nil
}
