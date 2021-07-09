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
	GetNodeSigningIdentity(ctx context.Context) (*fftypes.Identity, error)
	CreateTokenPool(ctx context.Context, in *fftypes.TokenPool) (out *fftypes.TokenPool, err error)
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
	bm := &assetManager{
		ctx:        ctx,
		database:   di,
		identity:   ii,
		blockchain: bi,
	}
	return bm, nil
}

func (bm *assetManager) GetNodeSigningIdentity(ctx context.Context) (*fftypes.Identity, error) {
	orgIdentity := config.GetString(config.OrgIdentity)
	id, err := bm.identity.Resolve(ctx, orgIdentity)
	if err != nil {
		return nil, err
	}
	return id, nil
}

func (bm *assetManager) CreateTokenPool(ctx context.Context, in *fftypes.TokenPool) (out *fftypes.TokenPool, err error) {
	id, err := bm.GetNodeSigningIdentity(ctx)
	if err == nil {
		err = bm.blockchain.VerifyIdentitySyntax(ctx, id)
	}
	if err != nil {
		return nil, err
	}

	// TODO: create token pool
	return in, nil
}

func (bm *assetManager) Start() error {
	return nil
}

func (bm *assetManager) WaitStop() {
	// No go routines
}
