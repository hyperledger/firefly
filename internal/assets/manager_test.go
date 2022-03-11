// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in comdiliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or imdilied.
// See the License for the specific language governing permissions and
// limitations under the License.

package assets

import (
	"context"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestAssets(t *testing.T) (*assetManager, func()) {
	config.Reset()
	mdi := &databasemocks.Plugin{}
	mim := &identitymanagermocks.Manager{}
	mdm := &datamocks.Manager{}
	msa := &syncasyncmocks.Bridge{}
	mbm := &broadcastmocks.Manager{}
	mpm := &privatemessagingmocks.Manager{}
	mti := &tokenmocks.Plugin{}
	mm := &metricsmocks.Manager{}
	mom := &operationmocks.Manager{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	mti.On("Name").Return("ut_tokens").Maybe()
	mm.On("IsMetricsEnabled").Return(false)
	mom.On("RegisterHandler", mock.Anything, mock.Anything, mock.Anything)
	ctx, cancel := context.WithCancel(context.Background())
	a, err := NewAssetManager(ctx, mdi, mim, mdm, msa, mbm, mpm, map[string]tokens.Plugin{"magic-tokens": mti}, mm, mom, txHelper)
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{a[1].(func(context.Context) error)(a[0].(context.Context))}
	}
	assert.NoError(t, err)
	am := a.(*assetManager)
	am.txHelper = &txcommonmocks.Helper{}
	return am, cancel
}

func newTestAssetsWithMetrics(t *testing.T) (*assetManager, func()) {
	config.Reset()
	mdi := &databasemocks.Plugin{}
	mim := &identitymanagermocks.Manager{}
	mdm := &datamocks.Manager{}
	msa := &syncasyncmocks.Bridge{}
	mbm := &broadcastmocks.Manager{}
	mpm := &privatemessagingmocks.Manager{}
	mti := &tokenmocks.Plugin{}
	mm := &metricsmocks.Manager{}
	mom := &operationmocks.Manager{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	mti.On("Name").Return("ut_tokens").Maybe()
	mm.On("IsMetricsEnabled").Return(true)
	mm.On("TransferSubmitted", mock.Anything)
	mom.On("RegisterHandler", mock.Anything, mock.Anything, mock.Anything)
	ctx, cancel := context.WithCancel(context.Background())
	a, err := NewAssetManager(ctx, mdi, mim, mdm, msa, mbm, mpm, map[string]tokens.Plugin{"magic-tokens": mti}, mm, mom, txHelper)
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
	_, err := NewAssetManager(context.Background(), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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
	mdi.On("GetTokenBalances", context.Background(), f).Return([]*fftypes.TokenBalance{}, nil, nil)
	_, _, err := am.GetTokenBalances(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetTokenAccounts(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenBalanceQueryFactory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenAccounts", context.Background(), f).Return([]*fftypes.TokenAccount{}, nil, nil)
	_, _, err := am.GetTokenAccounts(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetTokenAccountPools(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenBalanceQueryFactory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenAccountPools", context.Background(), "0x1", f).Return([]*fftypes.TokenAccountPool{}, nil, nil)
	_, _, err := am.GetTokenAccountPools(context.Background(), "ns1", "0x1", f)
	assert.NoError(t, err)
}

func TestGetTokenConnectors(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	_, err := am.GetTokenConnectors(context.Background(), "ns1")
	assert.NoError(t, err)
}

func TestGetTokenConnectorsBadNamespace(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	_, err := am.GetTokenConnectors(context.Background(), "")
	assert.Regexp(t, "FF10131", err)
}
