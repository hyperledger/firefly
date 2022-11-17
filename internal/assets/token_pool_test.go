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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateTokenPoolBadName(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPoolInput{}

	_, err := am.CreateTokenPool(context.Background(), pool, false)
	assert.Regexp(t, "FF00140", err)
}

func TestCreateTokenPoolGetError(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPoolInput{
		TokenPool: core.TokenPool{
			Name: "testpool",
		},
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "testpool").Return(nil, fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), pool, false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestCreateTokenPoolDuplicateName(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPoolInput{
		TokenPool: core.TokenPool{
			Name: "testpool",
		},
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "testpool").Return(&core.TokenPool{}, nil)

	_, err := am.CreateTokenPool(context.Background(), pool, false)
	assert.Regexp(t, "FF10275.*testpool", err)

	mdi.AssertExpectations(t)
}

func TestCreateTokenPoolDefaultConnectorSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPoolInput{
		TokenPool: core.TokenPool{
			Name: "testpool",
		},
		IdempotencyKey: "idem1",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mdi.On("GetTokenPool", context.Background(), "ns1", "testpool").Return(nil, nil)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("resolved-key", nil)
	mth.On("SubmitNewTransaction", context.Background(), core.TransactionTypeTokenPool, core.IdempotencyKey("idem1")).Return(fftypes.NewUUID(), nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(createPoolData)
		return op.Type == core.OpTypeTokenCreatePool && data.Pool == &pool.TokenPool
	})).Return(nil, nil)

	_, err := am.CreateTokenPool(context.Background(), pool, false)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestCreateTokenPoolDefaultConnectorNoConnectors(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPoolInput{
		TokenPool: core.TokenPool{
			Name: "testpool",
		},
	}

	am.tokens = make(map[string]tokens.Plugin)

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "testpool").Return(nil, nil)

	_, err := am.CreateTokenPool(context.Background(), pool, false)
	assert.Regexp(t, "FF10292", err)

	mdi.AssertExpectations(t)
}

func TestCreateTokenPoolDefaultConnectorMultipleConnectors(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPoolInput{
		TokenPool: core.TokenPool{
			Name: "testpool",
		},
	}

	am.tokens["magic-tokens"] = nil
	am.tokens["magic-tokens2"] = nil

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "testpool").Return(nil, nil)

	_, err := am.CreateTokenPool(context.Background(), pool, false)
	assert.Regexp(t, "FF10292", err)

	mdi.AssertExpectations(t)
}

func TestCreateTokenPoolNoConnectors(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	am.tokens = nil

	pool := &core.TokenPoolInput{
		TokenPool: core.TokenPool{
			Name: "testpool",
		},
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "testpool").Return(nil, nil)

	_, err := am.CreateTokenPool(context.Background(), pool, false)
	assert.Regexp(t, "FF10292", err)

	mdi.AssertExpectations(t)
}

func TestCreateTokenPoolIdentityFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPoolInput{
		TokenPool: core.TokenPool{
			Name: "testpool",
		},
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mdi.On("GetTokenPool", context.Background(), "ns1", "testpool").Return(nil, nil)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("", fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), pool, false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestCreateTokenPoolWrongConnector(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPoolInput{
		TokenPool: core.TokenPool{
			Connector: "wrongun",
			Name:      "testpool",
		},
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mdi.On("GetTokenPool", context.Background(), "ns1", "testpool").Return(nil, nil)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)

	_, err := am.CreateTokenPool(context.Background(), pool, false)
	assert.Regexp(t, "FF10272", err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestCreateTokenPoolFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPoolInput{
		TokenPool: core.TokenPool{
			Connector: "magic-tokens",
			Name:      "testpool",
		},
		IdempotencyKey: "idem1",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mdi.On("GetTokenPool", context.Background(), "ns1", "testpool").Return(nil, nil)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mth.On("SubmitNewTransaction", context.Background(), core.TransactionTypeTokenPool, core.IdempotencyKey("idem1")).Return(fftypes.NewUUID(), nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(createPoolData)
		return op.Type == core.OpTypeTokenCreatePool && data.Pool == &pool.TokenPool
	})).Return(nil, fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), pool, false)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestCreateTokenPoolTransactionFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPoolInput{
		TokenPool: core.TokenPool{
			Connector: "magic-tokens",
			Name:      "testpool",
		},
		IdempotencyKey: "idem1",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mdi.On("GetTokenPool", context.Background(), "ns1", "testpool").Return(nil, nil)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mth.On("SubmitNewTransaction", context.Background(), core.TransactionTypeTokenPool, core.IdempotencyKey("idem1")).Return(nil, fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), pool, false)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestCreateTokenPoolOpInsertFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPoolInput{
		TokenPool: core.TokenPool{
			Connector: "magic-tokens",
			Name:      "testpool",
		},
		IdempotencyKey: "idem1",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mdi.On("GetTokenPool", context.Background(), "ns1", "testpool").Return(nil, nil)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mth.On("SubmitNewTransaction", context.Background(), core.TransactionTypeTokenPool, core.IdempotencyKey("idem1")).Return(fftypes.NewUUID(), nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.Anything).Return(fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), pool, false)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestCreateTokenPoolSyncSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPoolInput{
		TokenPool: core.TokenPool{
			Connector: "magic-tokens",
			Name:      "testpool",
		},
		IdempotencyKey: "idem1",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mdi.On("GetTokenPool", context.Background(), "ns1", "testpool").Return(nil, nil)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mth.On("SubmitNewTransaction", context.Background(), core.TransactionTypeTokenPool, core.IdempotencyKey("idem1")).Return(fftypes.NewUUID(), nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(createPoolData)
		return op.Type == core.OpTypeTokenCreatePool && data.Pool == &pool.TokenPool
	})).Return(nil, nil)

	_, err := am.CreateTokenPool(context.Background(), pool, false)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestCreateTokenPoolAsyncSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPoolInput{
		TokenPool: core.TokenPool{
			Connector: "magic-tokens",
			Name:      "testpool",
		},
		IdempotencyKey: "idem1",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mdi.On("GetTokenPool", context.Background(), "ns1", "testpool").Return(nil, nil)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mth.On("SubmitNewTransaction", context.Background(), core.TransactionTypeTokenPool, core.IdempotencyKey("idem1")).Return(fftypes.NewUUID(), nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(createPoolData)
		return op.Type == core.OpTypeTokenCreatePool && data.Pool == &pool.TokenPool
	})).Return(nil, nil)

	_, err := am.CreateTokenPool(context.Background(), pool, false)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestCreateTokenPoolConfirm(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPoolInput{
		TokenPool: core.TokenPool{
			Connector: "magic-tokens",
			Name:      "testpool",
		},
		IdempotencyKey: "idem1",
	}

	mdi := am.database.(*databasemocks.Plugin)
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mdi.On("GetTokenPool", context.Background(), "ns1", "testpool").Return(nil, nil)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mth.On("SubmitNewTransaction", context.Background(), core.TransactionTypeTokenPool, core.IdempotencyKey("idem1")).Return(fftypes.NewUUID(), nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.Anything).Return(nil)
	msa.On("WaitForTokenPool", context.Background(), mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[2].(syncasync.SendFunction)
			send(context.Background())
		}).
		Return(nil, nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(createPoolData)
		return op.Type == core.OpTypeTokenCreatePool && data.Pool == &pool.TokenPool
	})).Return(nil, nil)

	_, err := am.CreateTokenPool(context.Background(), pool, true)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	msa.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestActivateTokenPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Connector: "magic-tokens",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mom := am.operations.(*operationmocks.Manager)
	mdi.On("GetOperations", context.Background(), "ns1", mock.Anything).Return(nil, nil, nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.MatchedBy(func(op *core.Operation) bool {
		return op.Type == core.OpTypeTokenActivatePool
	})).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(activatePoolData)
		return op.Type == core.OpTypeTokenActivatePool && data.Pool == pool
	})).Return(nil, nil)

	err := am.ActivateTokenPool(context.Background(), pool)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestActivateTokenPoolBadConnector(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPool{
		Namespace: "ns1",
		Connector: "bad",
	}

	err := am.ActivateTokenPool(context.Background(), pool)
	assert.Regexp(t, "FF10272", err)
}

func TestActivateTokenPoolOpInsertFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPool{
		Namespace: "ns1",
		Connector: "magic-tokens",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mom := am.operations.(*operationmocks.Manager)
	mdi.On("GetOperations", context.Background(), "ns1", mock.Anything).Return(nil, nil, nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.MatchedBy(func(op *core.Operation) bool {
		return op.Type == core.OpTypeTokenActivatePool
	})).Return(fmt.Errorf("pop"))

	err := am.ActivateTokenPool(context.Background(), pool)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestActivateTokenPoolFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPool{
		Namespace: "ns1",
		Connector: "magic-tokens",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mom := am.operations.(*operationmocks.Manager)
	mdi.On("GetOperations", context.Background(), "ns1", mock.Anything).Return(nil, nil, nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.MatchedBy(func(op *core.Operation) bool {
		return op.Type == core.OpTypeTokenActivatePool
	})).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(activatePoolData)
		return op.Type == core.OpTypeTokenActivatePool && data.Pool == pool
	})).Return(nil, fmt.Errorf("pop"))

	err := am.ActivateTokenPool(context.Background(), pool)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestActivateTokenPoolExisting(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPool{
		Namespace: "ns1",
		Connector: "magic-tokens",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", context.Background(), "ns1", mock.Anything).Return([]*core.Operation{{}}, nil, nil)

	err := am.ActivateTokenPool(context.Background(), pool)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestActivateTokenPoolExistingFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPool{
		Namespace: "ns1",
		Connector: "magic-tokens",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", context.Background(), "ns1", mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	err := am.ActivateTokenPool(context.Background(), pool)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestActivateTokenPoolSyncSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &core.TokenPool{
		Namespace: "ns1",
		Connector: "magic-tokens",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mdi.On("GetOperations", context.Background(), "ns1", mock.Anything).Return(nil, nil, nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.MatchedBy(func(op *core.Operation) bool {
		return op.Type == core.OpTypeTokenActivatePool
	})).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(activatePoolData)
		return op.Type == core.OpTypeTokenActivatePool && data.Pool == pool
	})).Return(nil, nil)

	err := am.ActivateTokenPool(context.Background(), pool)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestGetTokenPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "abc").Return(&core.TokenPool{}, nil)
	_, err := am.GetTokenPool(context.Background(), "magic-tokens", "abc")
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestGetTokenPoolNotFound(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "abc").Return(nil, nil)
	_, err := am.GetTokenPool(context.Background(), "magic-tokens", "abc")
	assert.Regexp(t, "FF10109", err)

	mdi.AssertExpectations(t)
}

func TestGetTokenPoolFailed(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "abc").Return(nil, fmt.Errorf("pop"))
	_, err := am.GetTokenPool(context.Background(), "magic-tokens", "abc")
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestGetTokenPoolBadPlugin(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	_, err := am.GetTokenPool(context.Background(), "", "")
	assert.Regexp(t, "FF10272", err)
}

func TestGetTokenPoolBadName(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	_, err := am.GetTokenPool(context.Background(), "magic-tokens", "")
	assert.Regexp(t, "FF00140", err)
}

func TestGetTokenPoolByID(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	u := fftypes.NewUUID()
	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), "ns1", u).Return(&core.TokenPool{}, nil)
	_, err := am.GetTokenPoolByNameOrID(context.Background(), u.String())
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestGetTokenPoolByIDBadNamespace(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	_, err := am.GetTokenPoolByNameOrID(context.Background(), "")
	assert.Regexp(t, "FF00140", err)
}

func TestGetTokenPoolByIDBadID(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	u := fftypes.NewUUID()
	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), "ns1", u).Return(nil, fmt.Errorf("pop"))
	_, err := am.GetTokenPoolByNameOrID(context.Background(), u.String())
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestGetTokenPoolByIDNilPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	u := fftypes.NewUUID()
	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), "ns1", u).Return(nil, nil)
	_, err := am.GetTokenPoolByNameOrID(context.Background(), u.String())
	assert.Regexp(t, "FF10109", err)

	mdi.AssertExpectations(t)
}

func TestGetTokenPoolByName(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "abc").Return(&core.TokenPool{}, nil)
	_, err := am.GetTokenPoolByNameOrID(context.Background(), "abc")
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestGetTokenPoolByNameBadName(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	_, err := am.GetTokenPoolByNameOrID(context.Background(), "")
	assert.Regexp(t, "FF00140", err)
}

func TestGetTokenPoolByNameNilPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "abc").Return(nil, fmt.Errorf("pop"))
	_, err := am.GetTokenPoolByNameOrID(context.Background(), "abc")
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
	mdi.On("GetTokenPools", context.Background(), "ns1", f).Return([]*core.TokenPool{}, nil, nil)
	_, _, err := am.GetTokenPools(context.Background(), f)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}
