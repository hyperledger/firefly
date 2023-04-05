// Copyright © 2023 Kaleido, Inc.
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

	"github.com/hyperledger/firefly-common/pkg/ffapi"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/contracts"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/internal/privatemessaging"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/tokens"
)

type Manager interface {
	core.Named

	CreateTokenPool(ctx context.Context, pool *core.TokenPoolInput, waitConfirm bool) (*core.TokenPool, error)
	ActivateTokenPool(ctx context.Context, pool *core.TokenPool) error
	GetTokenPools(ctx context.Context, filter ffapi.AndFilter) ([]*core.TokenPool, *ffapi.FilterResult, error)
	GetTokenPool(ctx context.Context, connector, poolName string) (*core.TokenPool, error)
	GetTokenPoolByNameOrID(ctx context.Context, poolNameOrID string) (*core.TokenPool, error)
	ResolvePoolMethods(ctx context.Context, pool *core.TokenPool) error
	DeleteTokenPool(ctx context.Context, poolNameOrID string) error

	GetTokenBalances(ctx context.Context, filter ffapi.AndFilter) ([]*core.TokenBalance, *ffapi.FilterResult, error)
	GetTokenAccounts(ctx context.Context, filter ffapi.AndFilter) ([]*core.TokenAccount, *ffapi.FilterResult, error)
	GetTokenAccountPools(ctx context.Context, key string, filter ffapi.AndFilter) ([]*core.TokenAccountPool, *ffapi.FilterResult, error)

	GetTokenTransfers(ctx context.Context, filter ffapi.AndFilter) ([]*core.TokenTransfer, *ffapi.FilterResult, error)
	GetTokenTransferByID(ctx context.Context, id string) (*core.TokenTransfer, error)

	NewTransfer(transfer *core.TokenTransferInput) syncasync.Sender
	MintTokens(ctx context.Context, transfer *core.TokenTransferInput, waitConfirm bool) (*core.TokenTransfer, error)
	BurnTokens(ctx context.Context, transfer *core.TokenTransferInput, waitConfirm bool) (*core.TokenTransfer, error)
	TransferTokens(ctx context.Context, transfer *core.TokenTransferInput, waitConfirm bool) (*core.TokenTransfer, error)

	GetTokenConnectors(ctx context.Context) []*core.TokenConnector

	NewApproval(approve *core.TokenApprovalInput) syncasync.Sender
	TokenApproval(ctx context.Context, approval *core.TokenApprovalInput, waitConfirm bool) (*core.TokenApproval, error)
	GetTokenApprovals(ctx context.Context, filter ffapi.AndFilter) ([]*core.TokenApproval, *ffapi.FilterResult, error)

	// From operations.OperationHandler
	PrepareOperation(ctx context.Context, op *core.Operation) (*core.PreparedOperation, error)
	RunOperation(ctx context.Context, op *core.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error)
}

type assetManager struct {
	ctx              context.Context
	namespace        string
	database         database.Plugin
	txHelper         txcommon.Helper
	identity         identity.Manager
	syncasync        syncasync.Bridge
	broadcast        broadcast.Manager        // optional
	messaging        privatemessaging.Manager // optional
	tokens           map[string]tokens.Plugin
	metrics          metrics.Manager
	operations       operations.Manager
	contracts        contracts.Manager
	keyNormalization int
}

func NewAssetManager(ctx context.Context, ns, keyNormalization string, di database.Plugin, ti map[string]tokens.Plugin, im identity.Manager, sa syncasync.Bridge, bm broadcast.Manager, pm privatemessaging.Manager, mm metrics.Manager, om operations.Manager, cm contracts.Manager, txHelper txcommon.Helper) (Manager, error) {
	if di == nil || im == nil || sa == nil || ti == nil || mm == nil || om == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "AssetManager")
	}
	am := &assetManager{
		ctx:              ctx,
		namespace:        ns,
		database:         di,
		txHelper:         txHelper,
		identity:         im,
		syncasync:        sa,
		broadcast:        bm,
		messaging:        pm,
		tokens:           ti,
		keyNormalization: identity.ParseKeyNormalizationConfig(keyNormalization),
		metrics:          mm,
		operations:       om,
		contracts:        cm,
	}
	om.RegisterHandler(ctx, am, []core.OpType{
		core.OpTypeTokenCreatePool,
		core.OpTypeTokenActivatePool,
		core.OpTypeTokenTransfer,
		core.OpTypeTokenApproval,
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
	return nil, i18n.NewError(ctx, coremsgs.MsgUnknownTokensPlugin, name)
}

func (am *assetManager) GetTokenBalances(ctx context.Context, filter ffapi.AndFilter) ([]*core.TokenBalance, *ffapi.FilterResult, error) {
	return am.database.GetTokenBalances(ctx, am.namespace, filter)
}

func (am *assetManager) GetTokenAccounts(ctx context.Context, filter ffapi.AndFilter) ([]*core.TokenAccount, *ffapi.FilterResult, error) {
	return am.database.GetTokenAccounts(ctx, am.namespace, filter)
}

func (am *assetManager) GetTokenAccountPools(ctx context.Context, key string, filter ffapi.AndFilter) ([]*core.TokenAccountPool, *ffapi.FilterResult, error) {
	return am.database.GetTokenAccountPools(ctx, am.namespace, key, filter)
}

func (am *assetManager) GetTokenConnectors(ctx context.Context) []*core.TokenConnector {
	connectors := []*core.TokenConnector{}
	for token := range am.tokens {
		connectors = append(
			connectors,
			&core.TokenConnector{
				Name: token,
			},
		)
	}
	return connectors
}

func (am *assetManager) getDefaultTokenConnector(ctx context.Context) (string, error) {
	tokenConnectors := am.GetTokenConnectors(ctx)
	if len(tokenConnectors) != 1 {
		return "", i18n.NewError(ctx, coremsgs.MsgFieldNotSpecified, "connector")
	}
	return tokenConnectors[0].Name, nil
}

func (am *assetManager) getDefaultTokenPool(ctx context.Context) (*core.TokenPool, error) {
	f := database.TokenPoolQueryFactory.NewFilter(ctx).And()
	f.Limit(1).Count(true)
	tokenPools, fr, err := am.GetTokenPools(ctx, f)
	if err != nil {
		return nil, err
	}
	if *fr.TotalCount != 1 {
		return nil, i18n.NewError(ctx, coremsgs.MsgFieldNotSpecified, "pool")
	}
	return tokenPools[0], nil
}
