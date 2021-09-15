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

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/syncasync"
	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/mocks/datamocks"
	"github.com/hyperledger-labs/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger-labs/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger-labs/firefly/mocks/tokenmocks"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestAssets(t *testing.T) (*assetManager, func()) {
	config.Reset()
	config.Set(config.OrgIdentity, "UTNodeID")
	mdi := &databasemocks.Plugin{}
	mim := &identitymanagermocks.Manager{}
	mdm := &datamocks.Manager{}
	msa := &syncasyncmocks.Bridge{}
	mti := &tokenmocks.Plugin{}
	mti.On("Name").Return("ut_tokens").Maybe()
	mim.On("ResolveInputIdentity", mock.Anything, mock.MatchedBy(func(identity *fftypes.Identity) bool {
		return identity.Author == "org1"
	})).Return(nil).Maybe()
	ctx, cancel := context.WithCancel(context.Background())
	a, err := NewAssetManager(ctx, mdi, mim, mdm, msa, map[string]tokens.Plugin{"magic-tokens": mti})
	assert.NoError(t, err)
	return a.(*assetManager), cancel
}

func TestInitFail(t *testing.T) {
	_, err := NewAssetManager(context.Background(), nil, nil, nil, nil, nil)
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

func TestCreateTokenPoolBadIdentity(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdm := am.data.(*datamocks.Manager)
	mim := am.identity.(*identitymanagermocks.Manager)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mim.On("ResolveInputIdentity", mock.Anything, mock.MatchedBy(func(identity *fftypes.Identity) bool {
		assert.Equal(t, "wrong", identity.Author)
		return true
	})).Return(fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", "test", &fftypes.TokenPool{Identity: fftypes.Identity{Author: "wrong"}}, false)
	assert.Regexp(t, "pop", err)
}

func TestCreateTokenPoolBadConnector(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdm := am.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", "bad", &fftypes.TokenPool{Identity: fftypes.Identity{Author: "org1"}}, false)
	assert.Regexp(t, "FF10272", err)
}

func TestCreateTokenPoolFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenPool
	}), false).Return(nil)
	mdi.On("UpsertOperation", mock.Anything, mock.Anything, false).Return(nil)
	mti.On("CreateTokenPool", context.Background(), mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", "magic-tokens", &fftypes.TokenPool{Identity: fftypes.Identity{Author: "org1"}}, false)
	assert.Regexp(t, "pop", err)
}

func TestCreateTokenPoolTransactionFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.Anything, false).Return(fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", "magic-tokens", &fftypes.TokenPool{Identity: fftypes.Identity{Author: "org1"}}, false)
	assert.Regexp(t, "pop", err)
}

func TestCreateTokenPoolOperationFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenPool
	}), false).Return(nil)
	mdi.On("UpsertOperation", mock.Anything, mock.Anything, false).Return(fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", "magic-tokens", &fftypes.TokenPool{Identity: fftypes.Identity{Author: "org1"}}, false)
	assert.Regexp(t, "pop", err)
}

func TestCreateTokenPoolSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mti.On("CreateTokenPool", context.Background(), mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenPool
	}), false).Return(nil)
	mdi.On("UpsertOperation", mock.Anything, mock.Anything, false).Return(nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", "magic-tokens", &fftypes.TokenPool{Identity: fftypes.Identity{Author: "org1"}}, false)
	assert.NoError(t, err)
}

func TestCreateTokenPoolConfirm(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	requestID := fftypes.NewUUID()

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil).Times(2)
	mti.On("CreateTokenPool", context.Background(), mock.Anything, mock.MatchedBy(func(pool *fftypes.TokenPool) bool {
		return pool.ID == requestID
	})).Return(nil).Times(1)
	mdi.On("UpsertTransaction", context.Background(), mock.MatchedBy(func(tx *fftypes.Transaction) bool {
		return tx.Subject.Type == fftypes.TransactionTypeTokenPool
	}), false).Return(nil)
	mdi.On("UpsertOperation", mock.Anything, mock.Anything, false).Return(nil).Times(1)
	msa.On("SendConfirmTokenPool", context.Background(), "ns1", mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[2].(syncasync.RequestSender)
			send(requestID)
		}).
		Return(nil, nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", "magic-tokens", &fftypes.TokenPool{Identity: fftypes.Identity{Author: "org1"}}, true)
	assert.NoError(t, err)
}

func TestGetTokenPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPool", context.Background(), "ns1", "abc").Return(nil, nil)
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
