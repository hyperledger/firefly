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
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/mocks/sysmessagingmocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetTokenTransfers(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenTransferQueryFactory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenTransfers", context.Background(), f).Return([]*core.TokenTransfer{}, nil, nil)
	_, _, err := am.GetTokenTransfers(context.Background(), "ns1", f)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestGetTokenTransferByID(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	u := fftypes.NewUUID()
	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenTransferByID", context.Background(), u).Return(&core.TokenTransfer{}, nil)
	_, err := am.GetTokenTransferByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestGetTokenTransferByIDBadID(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	_, err := am.GetTokenTransferByID(context.Background(), "ns1", "badUUID")
	assert.Regexp(t, "FF00138", err)
}

func TestMintTokensSuccess(t *testing.T) {
	am, cancel := newTestAssetsWithMetrics(t)
	defer cancel()

	mint := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", core.TransactionTypeTokenTransfer).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(transferData)
		return op.Type == core.OpTypeTokenTransfer && data.Pool == pool && data.Transfer == &mint.TokenTransfer
	})).Return(nil, nil)

	_, err := am.MintTokens(context.Background(), "ns1", mint, false)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestMintTokensBadConnector(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Connector: "bad",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)

	_, err := am.MintTokens(context.Background(), "ns1", mint, false)
	assert.Regexp(t, "FF10272.*bad", err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestMintTokenBadNamespace(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
	}

	_, err := am.MintTokens(context.Background(), "", mint, false)
	assert.Regexp(t, "FF00140", err)
}

func TestMintTokenDefaultPoolSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
		},
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
			Connector: "magic-tokens",
			State:     core.TokenPoolStateConfirmed,
		},
	}
	totalCount := int64(1)
	filterResult := &database.FilterResult{
		TotalCount: &totalCount,
	}
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPools", context.Background(), mock.MatchedBy((func(f database.AndFilter) bool {
		info, _ := f.Finalize()
		return info.Count && info.Limit == 1
	}))).Return(tokenPools, filterResult, nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", core.TransactionTypeTokenTransfer).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(transferData)
		return op.Type == core.OpTypeTokenTransfer && data.Pool == tokenPools[0] && data.Transfer == &mint.TokenTransfer
	})).Return(nil, nil)

	_, err := am.MintTokens(context.Background(), "ns1", mint, false)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestMintTokenDefaultPoolNoPools(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
		},
	}

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenPoolQueryFactory.NewFilter(context.Background())
	f := fb.And()
	f.Limit(1).Count(true)
	tokenPools := []*core.TokenPool{}
	totalCount := int64(0)
	filterResult := &database.FilterResult{
		TotalCount: &totalCount,
	}
	mdi.On("GetTokenPools", context.Background(), mock.MatchedBy((func(f database.AndFilter) bool {
		info, _ := f.Finalize()
		return info.Count && info.Limit == 1
	}))).Return(tokenPools, filterResult, nil)

	_, err := am.MintTokens(context.Background(), "ns1", mint, false)
	assert.Regexp(t, "FF10292", err)

	mdi.AssertExpectations(t)
}

func TestMintTokenDefaultPoolMultiplePools(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
		},
	}

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenPoolQueryFactory.NewFilter(context.Background())
	f := fb.And()
	f.Limit(1).Count(true)
	tokenPools := []*core.TokenPool{
		{
			Name: "pool1",
		},
		{
			Name: "pool2",
		},
	}
	totalCount := int64(2)
	filterResult := &database.FilterResult{
		TotalCount: &totalCount,
	}
	mdi.On("GetTokenPools", context.Background(), mock.MatchedBy((func(f database.AndFilter) bool {
		info, _ := f.Finalize()
		return info.Count && info.Limit == 1
	}))).Return(tokenPools, filterResult, nil)

	_, err := am.MintTokens(context.Background(), "ns1", mint, false)
	assert.Regexp(t, "FF10292", err)

	mdi.AssertExpectations(t)
}

func TestMintTokenDefaultPoolBadNamespace(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
		},
	}

	_, err := am.MintTokens(context.Background(), "", mint, false)
	assert.Regexp(t, "FF00140", err)
}

func TestMintTokensGetPoolsError(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
		},
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPools", context.Background(), mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	_, err := am.MintTokens(context.Background(), "ns1", mint, false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestMintTokensBadPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(nil, fmt.Errorf("pop"))

	_, err := am.MintTokens(context.Background(), "ns1", mint, false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestMintTokensIdentityFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("", fmt.Errorf("pop"))
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)

	_, err := am.MintTokens(context.Background(), "ns1", mint, false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestMintTokensFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", core.TransactionTypeTokenTransfer).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(transferData)
		return op.Type == core.OpTypeTokenTransfer && data.Pool == pool && data.Transfer == &mint.TokenTransfer
	})).Return(nil, fmt.Errorf("pop"))

	_, err := am.MintTokens(context.Background(), "ns1", mint, false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestMintTokensOperationFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
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
	mth := am.txHelper.(*txcommonmocks.Helper)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", core.TransactionTypeTokenTransfer).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(fmt.Errorf("pop"))

	_, err := am.MintTokens(context.Background(), "ns1", mint, false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestMintTokensConfirm(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", core.TransactionTypeTokenTransfer).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	msa.On("WaitForTokenTransfer", context.Background(), "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(context.Background())
		}).
		Return(&core.TokenTransfer{}, nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(transferData)
		return op.Type == core.OpTypeTokenTransfer && data.Pool == pool && data.Transfer == &mint.TokenTransfer
	})).Return(nil, fmt.Errorf("pop"))

	_, err := am.MintTokens(context.Background(), "ns1", mint, true)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	msa.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestBurnTokensSuccess(t *testing.T) {
	am, cancel := newTestAssetsWithMetrics(t)
	defer cancel()

	burn := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", core.TransactionTypeTokenTransfer).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(transferData)
		return op.Type == core.OpTypeTokenTransfer && data.Pool == pool && data.Transfer == &burn.TokenTransfer
	})).Return(nil, nil)

	_, err := am.BurnTokens(context.Background(), "ns1", burn, false)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestBurnTokensIdentityFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	burn := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("", fmt.Errorf("pop"))
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)

	_, err := am.BurnTokens(context.Background(), "ns1", burn, false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestBurnTokensConfirm(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	burn := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", core.TransactionTypeTokenTransfer).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	msa.On("WaitForTokenTransfer", context.Background(), "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(context.Background())
		}).
		Return(&core.TokenTransfer{}, nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(transferData)
		return op.Type == core.OpTypeTokenTransfer && data.Pool == pool && data.Transfer == &burn.TokenTransfer
	})).Return(nil, nil)

	_, err := am.BurnTokens(context.Background(), "ns1", burn, true)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	msa.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestTransferTokensSuccess(t *testing.T) {
	am, cancel := newTestAssetsWithMetrics(t)
	defer cancel()

	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			From:   "A",
			To:     "B",
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", core.TransactionTypeTokenTransfer).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(transferData)
		return op.Type == core.OpTypeTokenTransfer && data.Pool == pool && data.Transfer == &transfer.TokenTransfer
	})).Return(nil, nil)

	_, err := am.TransferTokens(context.Background(), "ns1", transfer, false)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestTransferTokensUnconfirmedPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			From:   "A",
			To:     "B",
			Amount: *fftypes.NewFFBigInt(5),
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

	_, err := am.TransferTokens(context.Background(), "ns1", transfer, false)
	assert.Regexp(t, "FF10293", err)

	mdi.AssertExpectations(t)
}

func TestTransferTokensIdentityFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			From:   "A",
			To:     "B",
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("", fmt.Errorf("pop"))
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)

	_, err := am.TransferTokens(context.Background(), "ns1", transfer, false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestTransferTokensNoFromOrTo(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &core.TokenTransferInput{
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)

	_, err := am.TransferTokens(context.Background(), "ns1", transfer, false)
	assert.Regexp(t, "FF10280", err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestTransferTokensTransactionFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			From:   "A",
			To:     "B",
			Amount: *fftypes.NewFFBigInt(5),
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
	mth := am.txHelper.(*txcommonmocks.Helper)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", core.TransactionTypeTokenTransfer).Return(nil, fmt.Errorf("pop"))

	_, err := am.TransferTokens(context.Background(), "ns1", transfer, false)
	assert.EqualError(t, err, "pop")

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestTransferTokensWithBroadcastMessage(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	msgID := fftypes.NewUUID()
	hash := fftypes.NewRandB32()
	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			From:   "A",
			To:     "B",
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
		Message: &core.MessageInOut{
			Message: core.Message{
				Header: core.MessageHeader{
					ID: msgID,
				},
				Hash: hash,
			},
			InlineData: core.InlineData{
				{
					Value: fftypes.JSONAnyPtr("test data"),
				},
			},
		},
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mbm := am.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", core.TransactionTypeTokenTransfer).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	mbm.On("NewBroadcast", "ns1", transfer.Message).Return(mms)
	mms.On("Prepare", context.Background()).Return(nil)
	mms.On("Send", context.Background()).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(transferData)
		return op.Type == core.OpTypeTokenTransfer && data.Pool == pool && data.Transfer == &transfer.TokenTransfer
	})).Return(nil, nil)

	_, err := am.TransferTokens(context.Background(), "ns1", transfer, false)
	assert.NoError(t, err)
	assert.Equal(t, *msgID, *transfer.TokenTransfer.Message)
	assert.Equal(t, *hash, *transfer.TokenTransfer.MessageHash)

	mbm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mms.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestTransferTokensWithBroadcastMessageSendFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	msgID := fftypes.NewUUID()
	hash := fftypes.NewRandB32()
	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			From:   "A",
			To:     "B",
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
		Message: &core.MessageInOut{
			Message: core.Message{
				Header: core.MessageHeader{
					ID: msgID,
				},
				Hash: hash,
			},
			InlineData: core.InlineData{
				{
					Value: fftypes.JSONAnyPtr("test data"),
				},
			},
		},
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mbm := am.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", core.TransactionTypeTokenTransfer).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	mbm.On("NewBroadcast", "ns1", transfer.Message).Return(mms)
	mms.On("Prepare", context.Background()).Return(nil)
	mms.On("Send", context.Background()).Return(fmt.Errorf("pop"))

	_, err := am.TransferTokens(context.Background(), "ns1", transfer, false)
	assert.Regexp(t, "pop", err)
	assert.Equal(t, *msgID, *transfer.TokenTransfer.Message)
	assert.Equal(t, *hash, *transfer.TokenTransfer.MessageHash)

	mbm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mms.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestTransferTokensWithBroadcastPrepareFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			From:   "A",
			To:     "B",
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
		Message: &core.MessageInOut{
			InlineData: core.InlineData{
				{
					Value: fftypes.JSONAnyPtr("test data"),
				},
			},
		},
	}

	mbm := am.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}
	mbm.On("NewBroadcast", "ns1", transfer.Message).Return(mms)
	mms.On("Prepare", context.Background()).Return(fmt.Errorf("pop"))

	_, err := am.TransferTokens(context.Background(), "ns1", transfer, false)
	assert.EqualError(t, err, "pop")

	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestTransferTokensWithPrivateMessage(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	msgID := fftypes.NewUUID()
	hash := fftypes.NewRandB32()
	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			From:   "A",
			To:     "B",
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
		Message: &core.MessageInOut{
			Message: core.Message{
				Header: core.MessageHeader{
					ID:   msgID,
					Type: core.MessageTypeTransferPrivate,
				},
				Hash: hash,
			},
			InlineData: core.InlineData{
				{
					Value: fftypes.JSONAnyPtr("test data"),
				},
			},
		},
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mpm := am.messaging.(*privatemessagingmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", core.TransactionTypeTokenTransfer).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	mpm.On("NewMessage", "ns1", transfer.Message).Return(mms)
	mms.On("Prepare", context.Background()).Return(nil)
	mms.On("Send", context.Background()).Return(nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(transferData)
		return op.Type == core.OpTypeTokenTransfer && data.Pool == pool && data.Transfer == &transfer.TokenTransfer
	})).Return(nil, nil)

	_, err := am.TransferTokens(context.Background(), "ns1", transfer, false)
	assert.NoError(t, err)
	assert.Equal(t, *msgID, *transfer.TokenTransfer.Message)
	assert.Equal(t, *hash, *transfer.TokenTransfer.MessageHash)

	mpm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mms.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestTransferTokensWithInvalidMessage(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			From:   "A",
			To:     "B",
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
		Message: &core.MessageInOut{
			Message: core.Message{
				Header: core.MessageHeader{
					Type: core.MessageTypeDefinition,
				},
			},
			InlineData: core.InlineData{
				{
					Value: fftypes.JSONAnyPtr("test data"),
				},
			},
		},
	}

	_, err := am.TransferTokens(context.Background(), "ns1", transfer, false)
	assert.Regexp(t, "FF10287", err)
}

func TestTransferTokensConfirm(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			From:   "A",
			To:     "B",
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", core.TransactionTypeTokenTransfer).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	msa.On("WaitForTokenTransfer", context.Background(), "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(context.Background())
		}).
		Return(&core.TokenTransfer{}, nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(transferData)
		return op.Type == core.OpTypeTokenTransfer && data.Pool == pool && data.Transfer == &transfer.TokenTransfer
	})).Return(nil, nil)

	_, err := am.TransferTokens(context.Background(), "ns1", transfer, true)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	msa.AssertExpectations(t)
	mim.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestTransferTokensWithBroadcastConfirm(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	msgID := fftypes.NewUUID()
	hash := fftypes.NewRandB32()
	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			From:   "A",
			To:     "B",
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
		Message: &core.MessageInOut{
			Message: core.Message{
				Header: core.MessageHeader{
					ID: msgID,
				},
				Hash: hash,
			},
			InlineData: core.InlineData{
				{
					Value: fftypes.JSONAnyPtr("test data"),
				},
			},
		},
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mbm := am.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mom := am.operations.(*operationmocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", core.TransactionTypeTokenTransfer).Return(fftypes.NewUUID(), nil)
	mbm.On("NewBroadcast", "ns1", transfer.Message).Return(mms)
	mms.On("Prepare", context.Background()).Return(nil)
	mms.On("Send", context.Background()).Return(nil)
	msa.On("WaitForMessage", context.Background(), "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(context.Background())
		}).
		Return(&core.Message{}, nil)
	msa.On("WaitForTokenTransfer", context.Background(), "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(context.Background())
		}).
		Return(&transfer.TokenTransfer, nil)
	mom.On("RunOperation", context.Background(), mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(transferData)
		return op.Type == core.OpTypeTokenTransfer && data.Pool == pool && data.Transfer == &transfer.TokenTransfer
	})).Return(nil, nil)

	_, err := am.TransferTokens(context.Background(), "ns1", transfer, true)
	assert.NoError(t, err)
	assert.Equal(t, *msgID, *transfer.TokenTransfer.Message)
	assert.Equal(t, *hash, *transfer.TokenTransfer.MessageHash)

	mbm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mms.AssertExpectations(t)
	msa.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestTransferTokensPoolNotFound(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			From:   "A",
			To:     "B",
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(nil, nil)

	_, err := am.TransferTokens(context.Background(), "ns1", transfer, false)
	assert.Regexp(t, "FF10109", err)

	mdi.AssertExpectations(t)
}

func TestTransferPrepare(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			Type:   core.TokenTransferTypeTransfer,
			From:   "A",
			To:     "B",
			Amount: *fftypes.NewFFBigInt(5),
		},
		Pool: "pool1",
	}
	pool := &core.TokenPool{
		Connector: "magic-tokens",
		State:     core.TokenPoolStateConfirmed,
	}

	sender := am.NewTransfer("ns1", transfer)

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("NormalizeSigningKey", context.Background(), "", identity.KeyNormalizationBlockchainPlugin).Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)

	err := sender.Prepare(context.Background())
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)
}
