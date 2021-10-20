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
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
			send(context.Background())
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

func TestValidateTokenPoolTx(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	err := am.ValidateTokenPoolTx(context.Background(), nil, "")
	assert.NoError(t, err)
}
