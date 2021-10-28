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

	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/mocks/sysmessagingmocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetTokenTransfers(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenTransferQueryFactory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenTransfers", context.Background(), f).Return([]*fftypes.TokenTransfer{}, nil, nil)
	_, _, err := am.GetTokenTransfers(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestGetTokenTransferByID(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	u := fftypes.NewUUID()
	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenTransfer", context.Background(), u).Return(&fftypes.TokenTransfer{}, nil)
	_, err := am.GetTokenTransferByID(context.Background(), "ns1", u.String())
	assert.NoError(t, err)
}

func TestGetTokenTransferByIDBadID(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	_, err := am.GetTokenTransferByID(context.Background(), "ns1", "badUUID")
	assert.Regexp(t, "FF10142", err)
}

func TestGetTokenTransfersByPool(t *testing.T) {
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
	_, _, err := am.GetTokenTransfersByPool(context.Background(), "ns1", "magic-tokens", "test", f)
	assert.NoError(t, err)
}

func TestGetTokenTransfersByPoolBadPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenTransferQueryFactory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenPool", context.Background(), "ns1", "test").Return(nil, fmt.Errorf("pop"))
	_, _, err := am.GetTokenTransfersByPool(context.Background(), "ns1", "magic-tokens", "test", f)
	assert.EqualError(t, err, "pop")
}

func TestGetTokenTransfersByPoolNoPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenTransferQueryFactory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenPool", context.Background(), "ns1", "test").Return(nil, nil)
	_, _, err := am.GetTokenTransfersByPool(context.Background(), "ns1", "magic-tokens", "test", f)
	assert.Regexp(t, "FF10109", err)
}

func TestMintTokensSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &fftypes.TokenTransferInput{}
	mint.Amount.Int().SetInt64(5)
	mint.Connector = "magic-tokens"
	mint.Pool = "pool1"

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("MintTokens", context.Background(), mock.Anything, &mint.TokenTransfer).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)

	_, err := am.MintTokens(context.Background(), "ns1", mint, false)
	assert.NoError(t, err)
}

func TestMintTokensByTypeSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &fftypes.TokenTransferInput{}
	mint.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("MintTokens", context.Background(), mock.Anything, &mint.TokenTransfer).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)

	_, err := am.MintTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", mint, false)
	assert.NoError(t, err)
}

func TestMintTokensByTypeBadPlugin(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)

	_, err := am.MintTokensByType(context.Background(), "", "", "", &fftypes.TokenTransferInput{}, false)
	assert.Regexp(t, "FF10272", err)
}

func TestMintTokensByTypeBadPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &fftypes.TokenTransferInput{}
	mint.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(nil, fmt.Errorf("pop"))

	_, err := am.MintTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", mint, false)
	assert.EqualError(t, err, "pop")
}

func TestMintTokensByTypeIdentityFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &fftypes.TokenTransferInput{}
	mint.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(nil, fmt.Errorf("pop"))
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)

	_, err := am.MintTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", mint, false)
	assert.EqualError(t, err, "pop")
}

func TestMintTokensByTypeFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &fftypes.TokenTransferInput{}
	mint.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("MintTokens", context.Background(), mock.Anything, &mint.TokenTransfer).Return(fmt.Errorf("pop"))
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)

	_, err := am.MintTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", mint, false)
	assert.EqualError(t, err, "pop")
}

func TestMintTokensByTypeOperationFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &fftypes.TokenTransferInput{}
	mint.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(fmt.Errorf("pop"))

	_, err := am.MintTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", mint, false)
	assert.EqualError(t, err, "pop")
}

func TestMintTokensByTypeConfirm(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mint := &fftypes.TokenTransferInput{}
	mint.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("MintTokens", context.Background(), mock.Anything, &mint.TokenTransfer).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	msa.On("WaitForTokenTransfer", context.Background(), "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(context.Background())
		}).
		Return(&fftypes.TokenTransfer{}, nil)

	_, err := am.MintTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", mint, true)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	msa.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestBurnTokensSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	burn := &fftypes.TokenTransferInput{}
	burn.Amount.Int().SetInt64(5)
	burn.Connector = "magic-tokens"
	burn.Pool = "pool1"

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("BurnTokens", context.Background(), mock.Anything, &burn.TokenTransfer).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)

	_, err := am.BurnTokens(context.Background(), "ns1", burn, false)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestBurnTokensByTypeSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	burn := &fftypes.TokenTransferInput{}
	burn.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("BurnTokens", context.Background(), mock.Anything, &burn.TokenTransfer).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)

	_, err := am.BurnTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", burn, false)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestBurnTokensByTypeIdentityFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	burn := &fftypes.TokenTransferInput{}
	burn.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(nil, fmt.Errorf("pop"))
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)

	_, err := am.BurnTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", burn, false)
	assert.EqualError(t, err, "pop")
}

func TestBurnTokensByTypeConfirm(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	burn := &fftypes.TokenTransferInput{}
	burn.Amount.Int().SetInt64(5)

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("BurnTokens", context.Background(), mock.Anything, &burn.TokenTransfer).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	msa.On("WaitForTokenTransfer", context.Background(), "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(context.Background())
		}).
		Return(&fftypes.TokenTransfer{}, nil)

	_, err := am.BurnTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", burn, true)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	msa.AssertExpectations(t)
	mti.AssertExpectations(t)
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
	transfer.Connector = "magic-tokens"
	transfer.Pool = "pool1"

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("TransferTokens", context.Background(), mock.Anything, &transfer.TokenTransfer).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)

	_, err := am.TransferTokens(context.Background(), "ns1", transfer, false)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTransferTokensByTypeSuccess(t *testing.T) {
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
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)

	_, err := am.TransferTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTransferTokensByTypeIdentityFail(t *testing.T) {
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

	_, err := am.TransferTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.EqualError(t, err, "pop")
}

func TestTransferTokensByTypeNoFromOrTo(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &fftypes.TokenTransferInput{}

	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)

	_, err := am.TransferTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.Regexp(t, "FF10280", err)

	mim.AssertExpectations(t)
}

func TestTransferTokensByTypeInvalidType(t *testing.T) {
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
	mdi.On("GetTokenPool", am.ctx, "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mdi.On("UpsertTransaction", am.ctx, mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mdi.On("UpsertOperation", am.ctx, mock.Anything, false).Return(nil)

	sender := &transferSender{
		mgr:       am,
		namespace: "ns1",
		connector: "magic-tokens",
		poolName:  "pool1",
		transfer:  transfer,
	}
	assert.PanicsWithValue(t, "unknown transfer type: ", func() {
		sender.Send(am.ctx)
	})
}

func TestTransferTokensByTypeTransactionFail(t *testing.T) {
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

	_, err := am.TransferTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.EqualError(t, err, "pop")

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestTransferTokensByTypeWithBroadcastMessage(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	hash := fftypes.NewRandB32()
	transfer := &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			From: "A",
			To:   "B",
		},
		Message: &fftypes.MessageInOut{
			Message: fftypes.Message{
				Hash: hash,
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
	mbm := am.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("TransferTokens", context.Background(), mock.Anything, &transfer.TokenTransfer).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	mbm.On("NewBroadcast", "ns1", transfer.Message).Return(mms)
	mms.On("Prepare", context.Background()).Return(nil)
	mms.On("Send", context.Background()).Return(nil)

	_, err := am.TransferTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.NoError(t, err)
	assert.Equal(t, *hash, *transfer.MessageHash)

	mbm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestTransferTokensByTypeWithBroadcastMessageFail(t *testing.T) {
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

	mim := am.identity.(*identitymanagermocks.Manager)
	mbm := am.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mbm.On("NewBroadcast", "ns1", transfer.Message).Return(mms)
	mms.On("Prepare", context.Background()).Return(nil)
	mms.On("Send", context.Background()).Return(fmt.Errorf("pop"))

	_, err := am.TransferTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.EqualError(t, err, "pop")

	mbm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestTransferTokensByTypeWithBroadcastPrepareFail(t *testing.T) {
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

	mim := am.identity.(*identitymanagermocks.Manager)
	mbm := am.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mbm.On("NewBroadcast", "ns1", transfer.Message).Return(mms)
	mms.On("Prepare", context.Background()).Return(fmt.Errorf("pop"))

	_, err := am.TransferTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.EqualError(t, err, "pop")

	mbm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestTransferTokensByTypeWithPrivateMessage(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	hash := fftypes.NewRandB32()
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
				Hash: hash,
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
	mms := &sysmessagingmocks.MessageSender{}
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("TransferTokens", context.Background(), mock.Anything, &transfer.TokenTransfer).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	mpm.On("NewMessage", "ns1", transfer.Message).Return(mms)
	mms.On("Prepare", context.Background()).Return(nil)
	mms.On("Send", context.Background()).Return(nil)

	_, err := am.TransferTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.NoError(t, err)
	assert.Equal(t, *hash, *transfer.MessageHash)

	mpm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestTransferTokensByTypeWithInvalidMessage(t *testing.T) {
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

	_, err := am.TransferTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", transfer, false)
	assert.Regexp(t, "FF10287", err)

	mim.AssertExpectations(t)
}

func TestTransferTokensByTypeConfirm(t *testing.T) {
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
	mdm := am.data.(*datamocks.Manager)
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("TransferTokens", context.Background(), mock.Anything, &transfer.TokenTransfer).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	msa.On("WaitForTokenTransfer", context.Background(), "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(context.Background())
		}).
		Return(&fftypes.TokenTransfer{}, nil)

	_, err := am.TransferTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", transfer, true)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	msa.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTransferPrepare(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	transfer := &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			Type: fftypes.TokenTransferTypeTransfer,
			From: "A",
			To:   "B",
		},
	}
	transfer.Amount.Int().SetInt64(5)

	sender := am.NewTransfer("ns1", "magic-tokens", "pool1", transfer)

	err := sender.Prepare(context.Background())
	assert.NoError(t, err)
}

func TestTransferTokensByTypeWithBroadcastConfirm(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	hash := fftypes.NewRandB32()
	transfer := &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			From: "A",
			To:   "B",
		},
		Message: &fftypes.MessageInOut{
			Message: fftypes.Message{
				Hash: hash,
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
	mbm := am.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("TransferTokens", context.Background(), mock.Anything, &transfer.TokenTransfer).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mbm.On("NewBroadcast", "ns1", transfer.Message).Return(mms)
	mms.On("Prepare", context.Background()).Return(nil)
	mms.On("SendAndWait", context.Background()).Return(nil)
	msa.On("WaitForTokenTransfer", context.Background(), "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(context.Background())
		}).
		Return(&transfer.TokenTransfer, nil)

	_, err := am.TransferTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", transfer, true)
	assert.NoError(t, err)
	assert.Equal(t, *hash, *transfer.MessageHash)

	mbm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
	mms.AssertExpectations(t)
	msa.AssertExpectations(t)
}

func TestTransferTokensByTypeWithBroadcastConfirmTransferFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	hash := fftypes.NewRandB32()
	transfer := &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			From: "A",
			To:   "B",
		},
		Message: &fftypes.MessageInOut{
			Message: fftypes.Message{
				Hash: hash,
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
	mbm := am.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mim.On("GetLocalOrganization", context.Background()).Return(&fftypes.Organization{Identity: "0x12345"}, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(&fftypes.TokenPool{}, nil)
	mti.On("TransferTokens", context.Background(), mock.Anything, &transfer.TokenTransfer).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenTransfer
	}), false).Return(nil)
	mdi.On("UpsertOperation", context.Background(), mock.Anything, false).Return(nil)
	mbm.On("NewBroadcast", "ns1", transfer.Message).Return(mms)
	mms.On("Prepare", context.Background()).Return(nil)
	msa.On("WaitForTokenTransfer", context.Background(), "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(context.Background())
		}).
		Return(nil, fmt.Errorf("pop"))

	_, err := am.TransferTokensByType(context.Background(), "ns1", "magic-tokens", "pool1", transfer, true)
	assert.EqualError(t, err, "pop")

	mbm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
	mms.AssertExpectations(t)
	msa.AssertExpectations(t)
}
