// Copyright © 2022 Kaleido, Inc.
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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateTokenPoolBadName(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{}

	mdm := am.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", pool, false)
	assert.Regexp(t, "FF10131", err)
}

func TestCreateTokenPoolUnknownConnectorSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Name: "testpool",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("resolved-key", nil)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", fftypes.TransactionTypeTokenPool).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(createPoolData)
		return op.Type == fftypes.OpTypeTokenCreatePool && data.Pool == pool
	})).Return(nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", pool, false)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestCreateTokenPoolUnknownConnectorNoConnectors(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Name: "testpool",
	}

	am.tokens = make(map[string]tokens.Plugin)

	mdm := am.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", pool, false)
	assert.Regexp(t, "FF10292", err)

	mdm.AssertExpectations(t)
}

func TestCreateTokenPoolUnknownConnectorMultipleConnectors(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Name: "testpool",
	}

	am.tokens["magic-tokens"] = nil
	am.tokens["magic-tokens2"] = nil

	mdm := am.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", pool, false)
	assert.Regexp(t, "FF10292", err)

	mdm.AssertExpectations(t)
}

func TestCreateTokenPoolMissingNamespace(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Name: "testpool",
	}

	mdm := am.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", pool, false)
	assert.EqualError(t, err, "pop")

	mdm.AssertExpectations(t)
}

func TestCreateTokenPoolNoConnectors(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	am.tokens = nil

	pool := &fftypes.TokenPool{
		Name: "testpool",
	}

	mdm := am.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", pool, false)
	assert.Regexp(t, "FF10292", err)

	mdm.AssertExpectations(t)
}

func TestCreateTokenPoolIdentityFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Name: "testpool",
	}

	mdm := am.data.(*datamocks.Manager)
	mim := am.identity.(*identitymanagermocks.Manager)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("", fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", pool, false)
	assert.EqualError(t, err, "pop")

	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestCreateTokenPoolWrongConnector(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Connector: "wrongun",
		Name:      "testpool",
	}

	mdm := am.data.(*datamocks.Manager)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", pool, false)
	assert.Regexp(t, "FF10272", err)

	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestCreateTokenPoolFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Connector: "magic-tokens",
		Name:      "testpool",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", fftypes.TransactionTypeTokenPool).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(createPoolData)
		return op.Type == fftypes.OpTypeTokenCreatePool && data.Pool == pool
	})).Return(fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", pool, false)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestCreateTokenPoolTransactionFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Connector: "magic-tokens",
		Name:      "testpool",
	}

	mdm := am.data.(*datamocks.Manager)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", fftypes.TransactionTypeTokenPool).Return(nil, fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", pool, false)
	assert.Regexp(t, "pop", err)

	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestCreateTokenPoolOpInsertFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Connector: "magic-tokens",
		Name:      "testpool",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", fftypes.TransactionTypeTokenPool).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", pool, false)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestCreateTokenPoolSyncSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Connector: "magic-tokens",
		Name:      "testpool",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", fftypes.TransactionTypeTokenPool).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(createPoolData)
		return op.Type == fftypes.OpTypeTokenCreatePool && data.Pool == pool
	})).Return(nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", pool, false)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestCreateTokenPoolAsyncSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Connector: "magic-tokens",
		Name:      "testpool",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", fftypes.TransactionTypeTokenPool).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(createPoolData)
		return op.Type == fftypes.OpTypeTokenCreatePool && data.Pool == pool
	})).Return(nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", pool, false)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestCreateTokenPoolConfirm(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Connector: "magic-tokens",
		Name:      "testpool",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", fftypes.TransactionTypeTokenPool).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	msa.On("WaitForTokenPool", context.Background(), "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(context.Background())
		}).
		Return(nil, nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(createPoolData)
		return op.Type == fftypes.OpTypeTokenCreatePool && data.Pool == pool
	})).Return(nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", pool, true)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	msa.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestActivateTokenPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Connector: "magic-tokens",
	}
	info := fftypes.JSONObject{
		"some": "info",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mom := am.operations.(*operationmocks.Manager)
	mdi.On("InsertOperation", context.Background(), mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Type == fftypes.OpTypeTokenActivatePool
	})).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(activatePoolData)
		assert.Equal(t, info, data.BlockchainInfo)
		return op.Type == fftypes.OpTypeTokenActivatePool && data.Pool == pool
	})).Return(nil)

	err := am.ActivateTokenPool(context.Background(), pool, info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestActivateTokenPoolBadConnector(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Namespace: "ns1",
		Connector: "bad",
	}
	info := fftypes.JSONObject{}

	err := am.ActivateTokenPool(context.Background(), pool, info)
	assert.Regexp(t, "FF10272", err)
}

func TestActivateTokenPoolOpInsertFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Namespace: "ns1",
		Connector: "magic-tokens",
	}
	info := fftypes.JSONObject{}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("InsertOperation", context.Background(), mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Type == fftypes.OpTypeTokenActivatePool
	})).Return(fmt.Errorf("pop"))

	err := am.ActivateTokenPool(context.Background(), pool, info)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestActivateTokenPoolFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Namespace: "ns1",
		Connector: "magic-tokens",
	}
	info := fftypes.JSONObject{
		"some": "info",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mdi.On("InsertOperation", context.Background(), mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Type == fftypes.OpTypeTokenActivatePool
	})).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(activatePoolData)
		assert.Equal(t, info, data.BlockchainInfo)
		return op.Type == fftypes.OpTypeTokenActivatePool && data.Pool == pool
	})).Return(fmt.Errorf("pop"))

	err := am.ActivateTokenPool(context.Background(), pool, info)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestActivateTokenPoolSyncSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		Namespace: "ns1",
		Connector: "magic-tokens",
	}
	info := fftypes.JSONObject{
		"some": "info",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mdi.On("InsertOperation", context.Background(), mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Type == fftypes.OpTypeTokenActivatePool
	})).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(activatePoolData)
		assert.Equal(t, info, data.BlockchainInfo)
		return op.Type == fftypes.OpTypeTokenActivatePool && data.Pool == pool
	})).Return(nil)

	err := am.ActivateTokenPool(context.Background(), pool, info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestGetTokenPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "abc").Return(&fftypes.TokenPool{}, nil)
	_, err := am.GetTokenPool(context.Background(), "ns1", "magic-tokens", "abc")
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestGetTokenPoolNotFound(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "abc").Return(nil, nil)
	_, err := am.GetTokenPool(context.Background(), "ns1", "magic-tokens", "abc")
	assert.Regexp(t, "FF10109", err)

	mdi.AssertExpectations(t)
}

func TestGetTokenPoolFailed(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "abc").Return(nil, fmt.Errorf("pop"))
	_, err := am.GetTokenPool(context.Background(), "ns1", "magic-tokens", "abc")
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestGetTokenPoolBadPlugin(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	_, err := am.GetTokenPool(context.Background(), "", "", "")
	assert.Regexp(t, "FF10272", err)
}

func TestGetTokenPoolBadNamespace(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	_, err := am.GetTokenPool(context.Background(), "", "magic-tokens", "")
	assert.Regexp(t, "FF10131", err)
}

func TestGetTokenPoolBadName(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	_, err := am.GetTokenPool(context.Background(), "ns1", "magic-tokens", "")
	assert.Regexp(t, "FF10131", err)
}

func TestGetTokenPoolByID(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	u := fftypes.NewUUID()
	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), u).Return(&fftypes.TokenPool{}, nil)
	_, err := am.GetTokenPoolByNameOrID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestGetTokenPoolByIDBadNamespace(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	_, err := am.GetTokenPoolByNameOrID(context.Background(), "", "")
	assert.Regexp(t, "FF10131", err)
}

func TestGetTokenPoolByIDBadID(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	u := fftypes.NewUUID()
	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), u).Return(nil, fmt.Errorf("pop"))
	_, err := am.GetTokenPoolByNameOrID(context.Background(), "ns1", u.String())
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestGetTokenPoolByIDNilPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	u := fftypes.NewUUID()
	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), u).Return(nil, nil)
	_, err := am.GetTokenPoolByNameOrID(context.Background(), "ns1", u.String())
	assert.Regexp(t, "FF10109", err)

	mdi.AssertExpectations(t)
}

func TestGetTokenPoolByName(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "abc").Return(&fftypes.TokenPool{}, nil)
	_, err := am.GetTokenPoolByNameOrID(context.Background(), "ns1", "abc")
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestGetTokenPoolByNameBadName(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	_, err := am.GetTokenPoolByNameOrID(context.Background(), "ns1", "")
	assert.Regexp(t, "FF10131", err)
}

func TestGetTokenPoolByNameNilPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "abc").Return(nil, fmt.Errorf("pop"))
	_, err := am.GetTokenPoolByNameOrID(context.Background(), "ns1", "abc")
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestGetTokenPools(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	u := fftypes.NewUUID()
	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenPoolQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	mdi.On("GetTokenPools", context.Background(), f).Return([]*fftypes.TokenPool{}, nil, nil)
	_, _, err := am.GetTokenPools(context.Background(), "ns1", f)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestGetTokenPoolsBadNamespace(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	u := fftypes.NewUUID()
	fb := database.TokenPoolQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := am.GetTokenPools(context.Background(), "", f)
	assert.Regexp(t, "FF10131", err)
}

func TestGetTokenPoolByNameOrIDBadNamespace(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	_, err := am.GetTokenPoolByNameOrID(context.Background(), "!wrong", "magic-tokens")
	assert.Regexp(t, "FF10131", err)
}
