// Copyright © 2021 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package assets

import (
	"context"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/pkg/blockchain"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/identity"
)

type Manager interface {
	CreateTokenPool(ctx context.Context, in *fftypes.TokenPoolCreate) (*fftypes.TokenPoolCreate, error)
	Start() error
	WaitStop()
}

type assetManager struct {
	ctx        context.Context
	database   database.Plugin
	identity   identity.Plugin
	blockchain blockchain.Plugin
}

func NewAssetManager(ctx context.Context, di database.Plugin, ii identity.Plugin, bi blockchain.Plugin) (Manager, error) {
	if di == nil || ii == nil || bi == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	am := &assetManager{
		ctx:        ctx,
		database:   di,
		identity:   ii,
		blockchain: bi,
	}
	return am, nil
}

func (am *assetManager) getNodeSigningIdentity(ctx context.Context) (*fftypes.Identity, error) {
	orgIdentity := config.GetString(config.OrgIdentity)
	id, err := am.identity.Resolve(ctx, orgIdentity)
	if err != nil {
		return nil, err
	}
	return id, nil
}

func (am *assetManager) CreateTokenPool(ctx context.Context, in *fftypes.TokenPoolCreate) (*fftypes.TokenPoolCreate, error) {
	id, err := am.getNodeSigningIdentity(ctx)
	if err == nil {
		err = am.blockchain.VerifyIdentitySyntax(ctx, id)
	}
	if err != nil {
		return nil, err
	}

	blockchainTrackingID, err := am.blockchain.CreateTokenPool(ctx, nil, id, &blockchain.TokenPool{
		BaseURI: in.BaseURI,
		Type:    in.Type,
	})
	if err != nil {
		return nil, err
	}

	op := fftypes.NewTXOperation(
		am.blockchain,
		"",
		fftypes.NewUUID(),
		blockchainTrackingID,
		fftypes.OpTypeBlockchainBatchPin,
		fftypes.OpStatusPending,
		"")
	return in, am.database.UpsertOperation(ctx, op, false)
}

func (am *assetManager) Start() error {
	return nil
}

func (am *assetManager) WaitStop() {
	// No go routines
}
