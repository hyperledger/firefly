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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
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
	mti.On("Name").Return("ut_tokens").Maybe()
	ctx, cancel := context.WithCancel(context.Background())
	a, err := NewAssetManager(ctx, mdi, mim, mdm, msa, mbm, mpm, map[string]tokens.Plugin{"magic-tokens": mti})
	rag := mdi.On("RunAsGroup", ctx, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{a[1].(func(context.Context) error)(a[0].(context.Context))}
	}
	assert.NoError(t, err)
	return a.(*assetManager), cancel
}

func TestInitFail(t *testing.T) {
	_, err := NewAssetManager(context.Background(), nil, nil, nil, nil, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestStartStop(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	am.Start()
	am.WaitStop()
}

func TestCreateTokenPoolBadNamespace(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdm := am.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", "test", &fftypes.TokenPool{}, false)
	assert.EqualError(t, err, "pop")
}

func TestCreateTokenPoolIdentityFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdm := am.data.(*datamocks.Manager)
	mim := am.identity.(*identitymanagermocks.Manager)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mim.On("GetLocalOrganization", context.Background()).Return(nil, fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", "test", &fftypes.TokenPool{}, false)
	assert.EqualError(t, err, "pop")
}

func TestCreateTokenPoolBadConnector(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdm := am.data.(*datamocks.Manager)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", "bad", &fftypes.TokenPool{}, false)
	assert.Regexp(t, "FF10272", err)
}

func TestCreateTokenPoolFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenPool
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	mti.On("CreateTokenPool", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", "magic-tokens", &fftypes.TokenPool{}, false)
	assert.Regexp(t, "pop", err)
}

func TestCreateTokenPoolTransactionFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.Anything, false).Return(fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", "magic-tokens", &fftypes.TokenPool{}, false)
	assert.Regexp(t, "pop", err)
}

func TestCreateTokenPoolOperationFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenPool
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", "magic-tokens", &fftypes.TokenPool{}, false)
	assert.Regexp(t, "pop", err)
}

func TestCreateTokenPoolSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mti.On("CreateTokenPool", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenPool
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", "magic-tokens", &fftypes.TokenPool{}, false)
	assert.NoError(t, err)
}

func TestCreateTokenPoolConfirm(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil).Times(2)
	mti.On("CreateTokenPool", context.Background(), mock.Anything, mock.Anything).Return(nil).Times(1)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenPool
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil).Times(1)
	msa.On("SendConfirmTokenPool", context.Background(), "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send()
		}).
		Return(nil, nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", "magic-tokens", &fftypes.TokenPool{}, true)
	assert.NoError(t, err)
}

func TestGetTokenPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "abc").Return(&fftypes.TokenPool{}, nil)
	_, err := am.GetTokenPool(context.Background(), "ns1", "magic-tokens", "abc")
	assert.NoError(t, err)
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

func TestGetTokenPools(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	u := fftypes.NewUUID()
	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenPoolQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	mdi.On("GetTokenPools", context.Background(), f).Return([]*fftypes.TokenPool{}, nil, nil)
	_, _, err := am.GetTokenPools(context.Background(), "ns1", "magic-tokens", f)
	assert.NoError(t, err)
}

func TestGetTokenPoolsBadPlugin(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	_, _, err := am.GetTokenPools(context.Background(), "", "", nil)
	assert.Regexp(t, "FF10272", err)
}

func TestGetTokenPoolsBadNamespace(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	u := fftypes.NewUUID()
	fb := database.TokenPoolQueryFactory.NewFilter(context.Background())
	f := fb.And(fb.Eq("id", u))
	_, _, err := am.GetTokenPools(context.Background(), "", "magic-tokens", f)
	assert.Regexp(t, "FF10131", err)
}

func TestGetTokenAccounts(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		ID: fftypes.NewUUID(),
	}
	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenAccountQueryFactory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenPool", context.Background(), "ns1", "test").Return(pool, nil)
	mdi.On("GetTokenAccounts", context.Background(), f).Return([]*fftypes.TokenAccount{}, nil, nil)
	_, _, err := am.GetTokenAccounts(context.Background(), "ns1", "magic-tokens", "test", f)
	assert.NoError(t, err)
}

func TestGetTokenAccountsBadPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenAccountQueryFactory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenPool", context.Background(), "ns1", "test").Return(nil, fmt.Errorf("pop"))
	_, _, err := am.GetTokenAccounts(context.Background(), "ns1", "magic-tokens", "test", f)
	assert.EqualError(t, err, "pop")
}

func TestValidateTokenPoolTx(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	err := am.ValidateTokenPoolTx(context.Background(), nil, "")
	assert.NoError(t, err)
}

func TestGetTokenTransfers(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	pool := &fftypes.TokenPool{
		ID: fftypes.NewUUID(),
	}
	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenTransferQueryFactory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenPool", context.Background(), "ns1", "test").Return(pool, nil)
	mdi.On("GetTokenTransfers", context.Background(), f).Return([]*fftypes.TokenTransfer{}, nil, nil)
	_, _, err := am.GetTokenTransfers(context.Background(), "ns1", "magic-tokens", "test", f)
	assert.NoError(t, err)
}

func TestGetTokenTransfersBadPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenTransferQueryFactory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenPool", context.Background(), "ns1", "test").Return(nil, fmt.Errorf("pop"))
	_, _, err := am.GetTokenTransfers(context.Background(), "ns1", "magic-tokens", "test", f)
	assert.EqualError(t, err, "pop")
}

func TestGetTokenTransfersNoPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenTransferQueryFactory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenPool", context.Background(), "ns1", "test").Return(nil, nil)
	_, _, err := am.GetTokenTransfers(context.Background(), "ns1", "magic-tokens", "test", f)
	assert.Regexp(t, "FF10109", err)
}

func TestMintTokensSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &fftypes.TokenTransfer{}
	mint.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("MintTokens", context.Background(), mock.Anything, mint).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)

	_, err := am.MintTokens(context.Background(), "ns1", "magic-tokens", "pool1", mint, false)
	assert.NoError(t, err)
}

func TestMintTokensBadPlugin(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)

	_, err := am.MintTokens(context.Background(), "", "", "", &fftypes.TokenTransfer{}, false)
	assert.Regexp(t, "FF10272", err)
}

func TestMintTokensBadPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &fftypes.TokenTransfer{}
	mint.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(nil, fmt.Errorf("pop"))

	_, err := am.MintTokens(context.Background(), "ns1", "magic-tokens", "pool1", mint, false)
	assert.EqualError(t, err, "pop")
}

func TestMintTokensIdentityFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &fftypes.TokenTransfer{}
	mint.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(nil, fmt.Errorf("pop"))
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)

	_, err := am.MintTokens(context.Background(), "ns1", "magic-tokens", "pool1", mint, false)
	assert.EqualError(t, err, "pop")
}

func TestMintTokensFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &fftypes.TokenTransfer{}
	mint.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	mti.On("MintTokens", context.Background(), mock.Anything, mint).Return(fmt.Errorf("pop"))
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)

	_, err := am.MintTokens(context.Background(), "ns1", "magic-tokens", "pool1", mint, false)
	assert.EqualError(t, err, "pop")
}

func TestMintTokensOperationFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &fftypes.TokenTransfer{}
	mint.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(fmt.Errorf("pop"))
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)

	_, err := am.MintTokens(context.Background(), "ns1", "magic-tokens", "pool1", mint, false)
	assert.EqualError(t, err, "pop")
}

func TestMintTokensConfirm(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &fftypes.TokenTransfer{}
	mint.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("MintTokens", context.Background(), mock.Anything, mint).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	msa.On("SendConfirmTokenTransfer", context.Background(), "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send()
		}).
		Return(nil, nil)

	_, err := am.MintTokens(context.Background(), "ns1", "magic-tokens", "pool1", mint, true)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	msa.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestBurnTokensSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	burn := &fftypes.TokenTransfer{}
	burn.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("BurnTokens", context.Background(), mock.Anything, burn).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)

	_, err := am.BurnTokens(context.Background(), "ns1", "magic-tokens", "pool1", burn, false)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestBurnTokensIdentityFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	burn := &fftypes.TokenTransfer{}
	burn.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(nil, fmt.Errorf("pop"))
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)

	_, err := am.BurnTokens(context.Background(), "ns1", "magic-tokens", "pool1", burn, false)
	assert.EqualError(t, err, "pop")
}

func TestTransferTokensSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			From: "A",
			To:   "B",
		},
	}
	transfer.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("TransferTokens", context.Background(), mock.Anything, &transfer.TokenTransfer).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)

	_, err := am.TransferTokens(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTransferTokensIdentityFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			From: "A",
			To:   "B",
		},
	}
	transfer.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(nil, fmt.Errorf("pop"))
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)

	_, err := am.TransferTokens(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.EqualError(t, err, "pop")
}

func TestTransferTokensNoFromOrTo(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &fftypes.TokenTransferInput{}

	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)

	_, err := am.TransferTokens(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.Regexp(t, "FF10280", err)

	mim.AssertExpectations(t)
}

func TestTransferTokensInvalidType(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &fftypes.TokenTransfer{
		From: "A",
		To:   "B",
	}
	transfer.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)

	assert.Panics(t, func() {
		am.transferTokensWithID(context.Background(), fftypes.NewUUID(), "ns1", "magic-tokens", "pool1", transfer, false)
	})

	mdi.AssertExpectations(t)
}

func TestTransferTokensTransactionFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			From: "A",
			To:   "B",
		},
	}
	transfer.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(fmt.Errorf("pop"))

	_, err := am.TransferTokens(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.EqualError(t, err, "pop")

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestTransferTokensWithBroadcastMessage(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			From: "A",
			To:   "B",
		},
		Message: &fftypes.MessageInOut{
			InlineData: fftypes.InlineData{
				{
					Value: []byte("test data"),
				},
			},
		},
	}
	transfer.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mbm := am.broadcast.(*broadcastmocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("TransferTokens", context.Background(), mock.Anything, &transfer.TokenTransfer).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mbm.On("BroadcastMessage", context.Background(), "ns1", transfer.Message, false).Return(&transfer.Message.Message, nil)

	_, err := am.TransferTokens(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.NoError(t, err)

	mbm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTransferTokensWithPrivateMessage(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			From: "A",
			To:   "B",
		},
		Message: &fftypes.MessageInOut{
			Message: fftypes.Message{
				Header: fftypes.MessageHeader{
					Type: fftypes.MessageTypeTransferPrivate,
				},
			},
			InlineData: fftypes.InlineData{
				{
					Value: []byte("test data"),
				},
			},
		},
	}
	transfer.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mpm := am.messaging.(*privatemessagingmocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("TransferTokens", context.Background(), mock.Anything, &transfer.TokenTransfer).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mpm.On("SendMessage", context.Background(), "ns1", transfer.Message, false).Return(&transfer.Message.Message, nil)

	_, err := am.TransferTokens(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.NoError(t, err)

	mpm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTransferTokensWithInvalidMessage(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			From: "A",
			To:   "B",
		},
		Message: &fftypes.MessageInOut{
			Message: fftypes.Message{
				Header: fftypes.MessageHeader{
					Type: fftypes.MessageTypeDefinition,
				},
			},
			InlineData: fftypes.InlineData{
				{
					Value: []byte("test data"),
				},
			},
		},
	}
	transfer.Amount.Int().SetInt64(5)

	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)

	_, err := am.TransferTokens(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.Regexp(t, "FF10287", err)

	mim.AssertExpectations(t)
}
