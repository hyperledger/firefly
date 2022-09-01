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
	"testing"
	"time"

	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/cachemocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestAssets(t *testing.T) (*assetManager, func()) {
	return newTestAssetsCommon(t, false)
}

func newTestAssetsWithMetrics(t *testing.T) (*assetManager, func()) {
	return newTestAssetsCommon(t, true)
}

func newTestAssetsCommon(t *testing.T, metrics bool) (*assetManager, func()) {
	coreconfig.Reset()
	mdi := &databasemocks.Plugin{}
	mim := &identitymanagermocks.Manager{}
	mdm := &datamocks.Manager{}
	msa := &syncasyncmocks.Bridge{}
	mbm := &broadcastmocks.Manager{}
	mpm := &privatemessagingmocks.Manager{}
	mti := &tokenmocks.Plugin{}
	mm := &metricsmocks.Manager{}
	cmi := &cachemocks.Manager{}
	ctx := context.Background()
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	mom := &operationmocks.Manager{}
	txHelper, _ := txcommon.NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)
	mm.On("IsMetricsEnabled").Return(metrics)
	mm.On("TransferSubmitted", mock.Anything)
	mom.On("RegisterHandler", mock.Anything, mock.Anything, mock.Anything)
	mti.On("Name").Return("ut").Maybe()
	ctx, cancel := context.WithCancel(ctx)
	a, err := NewAssetManager(ctx, "ns1", "blockchain_plugin", mdi, map[string]tokens.Plugin{"magic-tokens": mti}, mim, msa, mbm, mpm, mm, mom, txHelper)
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{a[1].(func(context.Context) error)(a[0].(context.Context))}
	}
	assert.NoError(t, err)
	am := a.(*assetManager)
	am.txHelper = &txcommonmocks.Helper{}
	return am, cancel
}

func TestInitFail(t *testing.T) {
	_, err := NewAssetManager(context.Background(), "", "", nil, nil, nil, nil, nil, nil, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestName(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	assert.Equal(t, "AssetManager", am.Name())
}

func TestGetTokenBalances(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenBalanceQueryFactory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenBalances", context.Background(), "ns1", f).Return([]*core.TokenBalance{}, nil, nil)
	_, _, err := am.GetTokenBalances(context.Background(), f)
	assert.NoError(t, err)
}

func TestGetTokenAccounts(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenBalanceQueryFactory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenAccounts", context.Background(), "ns1", f).Return([]*core.TokenAccount{}, nil, nil)
	_, _, err := am.GetTokenAccounts(context.Background(), f)
	assert.NoError(t, err)
}

func TestGetTokenAccountPools(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenBalanceQueryFactory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenAccountPools", context.Background(), "ns1", "0x1", f).Return([]*core.TokenAccountPool{}, nil, nil)
	_, _, err := am.GetTokenAccountPools(context.Background(), "0x1", f)
	assert.NoError(t, err)
}

func TestGetTokenConnectors(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	connectors := am.GetTokenConnectors(context.Background())
	assert.Equal(t, 1, len(connectors))
	assert.Equal(t, "magic-tokens", connectors[0].Name)
}
