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
	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/mocks/datamocks"
	"github.com/hyperledger-labs/firefly/mocks/identitymocks"
	"github.com/hyperledger-labs/firefly/mocks/tokenmocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestAssets(t *testing.T) (*assetManager, func()) {
	config.Reset()
	config.Set(config.OrgIdentity, "UTNodeID")
	mdi := &databasemocks.Plugin{}
	mii := &identitymocks.Plugin{}
	mdm := &datamocks.Manager{}
	mti := &tokenmocks.Plugin{}
	mti.On("Name").Return("ut_tokens").Maybe()
	defaultIdentity := &fftypes.Identity{Identifier: "UTNodeID", OnChain: "0x12345"}
	mii.On("Resolve", mock.Anything, "UTNodeID").Return(defaultIdentity, nil).Maybe()
	ctx, cancel := context.WithCancel(context.Background())
	a, err := NewAssetManager(ctx, mdi, mii, mdm, map[string]tokens.Plugin{"magic-tokens": mti})
	assert.NoError(t, err)
	return a.(*assetManager), cancel
}

func TestInitFail(t *testing.T) {
	_, err := NewAssetManager(context.Background(), nil, nil, nil, nil)
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
	mii := am.identity.(*identitymocks.Plugin)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mii.On("Resolve", mock.Anything, "wrong").Return(nil, fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", "test", &fftypes.TokenPool{Author: "wrong"}, false)
	assert.Regexp(t, "pop", err)
}

func TestCreateTokenPoolBadConnector(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdm := am.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", "bad", &fftypes.TokenPool{}, false)
	assert.Regexp(t, "FF10272", err)
}

func TestCreateTokenPoolFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdm := am.data.(*datamocks.Manager)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mti.On("CreateTokenPool", context.Background(), mock.Anything, mock.Anything).Return("", fmt.Errorf("pop"))

	_, err := am.CreateTokenPool(context.Background(), "ns1", "magic-tokens", &fftypes.TokenPool{}, false)
	assert.Regexp(t, "pop", err)
}

func TestCreateTokenPoolSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	mdm := am.data.(*datamocks.Manager)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mdm.On("VerifyNamespaceExists", context.Background(), "ns1").Return(nil)
	mti.On("CreateTokenPool", context.Background(), mock.Anything, mock.Anything).Return("tx12345", nil)
	mdi.On("UpsertOperation", mock.Anything, mock.Anything, false).Return(nil)

	_, err := am.CreateTokenPool(context.Background(), "ns1", "magic-tokens", &fftypes.TokenPool{}, false)
	assert.NoError(t, err)
}
