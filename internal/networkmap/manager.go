// Copyright © 2022 Kaleido, Inc.
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

package networkmap

import (
	"context"

	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/i18n"
)

type Manager interface {
	Start() error

	RegisterOrganization(ctx context.Context, org *fftypes.IdentityCreateDTO, waitConfirm bool) (identity *fftypes.Identity, err error)
	RegisterNode(ctx context.Context, waitConfirm bool) (node *fftypes.Identity, err error)
	RegisterNodeOrganization(ctx context.Context, waitConfirm bool) (org *fftypes.Identity, err error)
	RegisterIdentity(ctx context.Context, ns string, dto *fftypes.IdentityCreateDTO, waitConfirm bool) (identity *fftypes.Identity, err error)
	UpdateIdentity(ctx context.Context, ns string, id string, dto *fftypes.IdentityUpdateDTO, waitConfirm bool) (identity *fftypes.Identity, err error)

	GetOrganizationByNameOrID(ctx context.Context, nameOrID string) (*fftypes.Identity, error)
	GetOrganizations(ctx context.Context, filter database.AndFilter) ([]*fftypes.Identity, *database.FilterResult, error)
	GetOrganizationsWithVerifiers(ctx context.Context, filter database.AndFilter) ([]*fftypes.IdentityWithVerifiers, *database.FilterResult, error)
	GetNodeByNameOrID(ctx context.Context, nameOrID string) (*fftypes.Identity, error)
	GetNodes(ctx context.Context, filter database.AndFilter) ([]*fftypes.Identity, *database.FilterResult, error)
	GetIdentityByID(ctx context.Context, ns string, id string) (*fftypes.Identity, error)
	GetIdentityByIDWithVerifiers(ctx context.Context, ns, id string) (*fftypes.IdentityWithVerifiers, error)
	GetIdentityByDID(ctx context.Context, did string) (*fftypes.Identity, error)
	GetIdentityByDIDWithVerifiers(ctx context.Context, did string) (*fftypes.IdentityWithVerifiers, error)
	GetIdentities(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Identity, *database.FilterResult, error)
	GetIdentitiesGlobal(ctx context.Context, filter database.AndFilter) ([]*fftypes.Identity, *database.FilterResult, error)
	GetIdentitiesWithVerifiers(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.IdentityWithVerifiers, *database.FilterResult, error)
	GetIdentitiesWithVerifiersGlobal(ctx context.Context, filter database.AndFilter) ([]*fftypes.IdentityWithVerifiers, *database.FilterResult, error)
	GetIdentityVerifiers(ctx context.Context, ns, id string, filter database.AndFilter) ([]*fftypes.Verifier, *database.FilterResult, error)
	GetVerifiers(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Verifier, *database.FilterResult, error)
	GetVerifierByHash(ctx context.Context, ns, hash string) (*fftypes.Verifier, error)
	GetDIDDocForIdentityByID(ctx context.Context, ns, id string) (*DIDDocument, error)
	GetDIDDocForIdentityByDID(ctx context.Context, did string) (*DIDDocument, error)
}

type networkMap struct {
	ctx       context.Context
	database  database.Plugin
	broadcast broadcast.Manager
	exchange  dataexchange.Plugin
	identity  identity.Manager
	syncasync syncasync.Bridge
	metrics   metrics.Manager
}

func NewNetworkMap(ctx context.Context, di database.Plugin, bm broadcast.Manager, dx dataexchange.Plugin, im identity.Manager, sa syncasync.Bridge, mm metrics.Manager) (Manager, error) {
	if di == nil || bm == nil || dx == nil || im == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError)
	}

	nm := &networkMap{
		ctx:       ctx,
		database:  di,
		broadcast: bm,
		exchange:  dx,
		identity:  im,
		syncasync: sa,
		metrics:   mm,
	}

	return nm, nil
}

func (nm *networkMap) Start() error {
	if nm.metrics.IsMetricsEnabled() {
		go nm.networkMetricsLoop()
	}
	return nil
}
