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

	"github.com/hyperledger/firefly-common/pkg/ffapi"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetTokenApprovals(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenApprovalQueryFactory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenApprovals", context.Background(), "ns1", f).Return([]*core.TokenApproval{}, nil, nil)
	_, _, err := am.GetTokenApprovals(context.Background(), f)
	assert.NoError(t, err)
}

func TestTokenApprovalSuccess(t *testing.T) {
	am, cancel := newTestAssetsWithMetrics(t)
	defer cancel()

	approval := &core.TokenApprovalInput{
		TokenApproval: core.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool:           "pool1",
		IdempotencyKey: "idem1",
	}
	pool := &core.TokenPool{
		Locator:   "F1",
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "key", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), core.TransactionTypeTokenApproval, core.IdempotencyKey("idem1")).Return(fftypes.NewUUID(), nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(approvalData)
		return op.Type == core.OpTypeTokenApproval && data.Pool == pool && data.Approval == &approval.TokenApproval
	})).Return(nil, nil)

	_, err := am.TokenApproval(context.Background(), approval, false)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestTokenApprovalSuccessUnknownIdentity(t *testing.T) {
	am, cancel := newTestAssetsWithMetrics(t)
	defer cancel()

	approval := &core.TokenApprovalInput{
		TokenApproval: core.TokenApproval{
			Approved: true,
			Operator: "operator",
		},
		Pool:           "pool1",
		IdempotencyKey: "idem1",
	}
	pool := &core.TokenPool{
		Locator:   "F1",
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), core.TransactionTypeTokenApproval, core.IdempotencyKey("idem1")).Return(fftypes.NewUUID(), nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(approvalData)
		return op.Type == core.OpTypeTokenApproval && data.Pool == pool && data.Approval == &approval.TokenApproval
	})).Return(nil, nil)

	_, err := am.TokenApproval(context.Background(), approval, false)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestApprovalBadConnector(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &core.TokenApprovalInput{
		TokenApproval: core.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Locator:   "F1",
		Connector: "bad",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "key", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)

	_, err := am.TokenApproval(context.Background(), approval, false)
	assert.Regexp(t, "FF10272", err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestApprovalDefaultPoolSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &core.TokenApprovalInput{
		TokenApproval: core.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		IdempotencyKey: "idem1",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	fb := database.TokenPoolQueryFactory.NewFilter(context.Background())
	f := fb.And()
	f.Limit(1).Count(true)
	tokenPools := []*core.TokenPool{
		{
			Name:      "pool1",
			Locator:   "F1",
			Connector: "magic-tokens",
			State:     core.TokenPoolStateConfirmed,
		},
	}
	totalCount := int64(1)
	filterResult := &ffapi.FilterResult{
		TotalCount: &totalCount,
	}
	mim.On("NormalizeSigningKey", context.Background(), "key", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPools", context.Background(), "ns1", mock.MatchedBy((func(f ffapi.AndFilter) bool {
		info, _ := f.Finalize()
		return info.Count && info.Limit == 1
	}))).Return(tokenPools, filterResult, nil)
	mth.On("SubmitNewTransaction", context.Background(), core.TransactionTypeTokenApproval, core.IdempotencyKey("idem1")).Return(fftypes.NewUUID(), nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(approvalData)
		return op.Type == core.OpTypeTokenApproval && data.Pool == tokenPools[0] && data.Approval == &approval.TokenApproval
	})).Return(nil, nil)

	_, err := am.TokenApproval(context.Background(), approval, false)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestApprovalDefaultPoolNoPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &core.TokenApprovalInput{
		TokenApproval: core.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
	}

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenPoolQueryFactory.NewFilter(context.Background())
	f := fb.And()
	f.Limit(1).Count(true)
	tokenPools := []*core.TokenPool{}
	totalCount := int64(0)
	filterResult := &ffapi.FilterResult{
		TotalCount: &totalCount,
	}
	mdi.On("GetTokenPools", context.Background(), "ns1", mock.MatchedBy((func(f ffapi.AndFilter) bool {
		info, _ := f.Finalize()
		return info.Count && info.Limit == 1
	}))).Return(tokenPools, filterResult, nil)

	_, err := am.TokenApproval(context.Background(), approval, false)
	assert.Regexp(t, "FF10292", err)
}

func TestApprovalBadPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &core.TokenApprovalInput{
		TokenApproval: core.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool: "pool1",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "key", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(nil, fmt.Errorf("pop"))

	_, err := am.TokenApproval(context.Background(), approval, false)
	assert.EqualError(t, err, "pop")
}

func TestApprovalUnconfirmedPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &core.TokenApprovalInput{
		TokenApproval: core.TokenApproval{
			Approved: true,
			Operator: "operator",
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Locator:   "F1",
		Connector: "magic-tokens",
		State:     core.TokenPoolStatePending,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)

	_, err := am.TokenApproval(context.Background(), approval, false)
	assert.Regexp(t, "FF10293", err)

	mdi.AssertExpectations(t)
}

func TestApprovalIdentityFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &core.TokenApprovalInput{
		TokenApproval: core.TokenApproval{
			Approved: true,
			Operator: "operator",
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Locator:   "F1",
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("", fmt.Errorf("pop"))
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)

	_, err := am.TokenApproval(context.Background(), approval, false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestApprovalFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &core.TokenApprovalInput{
		TokenApproval: core.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool:           "pool1",
		IdempotencyKey: "idem1",
	}
	pool := &core.TokenPool{
		Locator:   "F1",
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "key", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), core.TransactionTypeTokenApproval, core.IdempotencyKey("idem1")).Return(fftypes.NewUUID(), nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(approvalData)
		return op.Type == core.OpTypeTokenApproval && data.Pool == pool && data.Approval == &approval.TokenApproval
	})).Return(nil, fmt.Errorf("pop"))

	_, err := am.TokenApproval(context.Background(), approval, false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestApprovalTransactionFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &core.TokenApprovalInput{
		TokenApproval: core.TokenApproval{
			Approved: true,
			Operator: "operator",
		},
		Pool:           "pool1",
		IdempotencyKey: "idem1",
	}
	pool := &core.TokenPool{
		Locator:   "F1",
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), core.TransactionTypeTokenApproval, core.IdempotencyKey("idem1")).Return(nil, fmt.Errorf("pop"))

	_, err := am.TokenApproval(context.Background(), approval, false)
	assert.EqualError(t, err, "pop")

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestApprovalOperationsFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &core.TokenApprovalInput{
		TokenApproval: core.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool:           "pool1",
		IdempotencyKey: "idem1",
	}
	pool := &core.TokenPool{
		Locator:   "F1",
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)

	mim.On("NormalizeSigningKey", context.Background(), "key", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), core.TransactionTypeTokenApproval, core.IdempotencyKey("idem1")).Return(fftypes.NewUUID(), nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.Anything).Return(fmt.Errorf("pop"))

	_, err := am.TokenApproval(context.Background(), approval, false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mom.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestTokenApprovalConfirm(t *testing.T) {
	am, cancel := newTestAssetsWithMetrics(t)
	defer cancel()

	approval := &core.TokenApprovalInput{
		TokenApproval: core.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool:           "pool1",
		IdempotencyKey: "idem1",
	}
	pool := &core.TokenPool{
		Locator:   "F1",
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "key", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), core.TransactionTypeTokenApproval, core.IdempotencyKey("idem1")).Return(fftypes.NewUUID(), nil)
	mom.On("AddOrReuseOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(approvalData)
		return op.Type == core.OpTypeTokenApproval && data.Pool == pool && data.Approval == &approval.TokenApproval
	})).Return(nil, nil)

	msa.On("WaitForTokenApproval", context.Background(), mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[2].(syncasync.SendFunction)
			send(context.Background())
		}).
		Return(&core.TokenApproval{}, nil)

	_, err := am.TokenApproval(context.Background(), approval, true)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	msa.AssertExpectations(t)
	mom.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestApprovalPrepare(t *testing.T) {
	am, cancel := newTestAssetsWithMetrics(t)
	defer cancel()

	approval := &core.TokenApprovalInput{
		TokenApproval: core.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Locator:   "F1",
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	sender := am.NewApproval(approval)

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "key", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)

	err := sender.Prepare(context.Background())
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}
