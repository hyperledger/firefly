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

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (nm *networkMap) GetOrganizationByID(ctx context.Context, id string) (*fftypes.Identity, error) {
	u, err := fftypes.ParseUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	o, err := nm.database.GetIdentityByID(ctx, u)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	if o.Type != fftypes.IdentityTypeOrg {
		log.L(ctx).Warnf("Identity '%s' (%s) is not an org identity", o.DID, o.ID)
		return nil, nil
	}
	return o, nil
}

func (nm *networkMap) GetOrganizations(ctx context.Context, filter database.AndFilter) ([]*fftypes.Identity, *database.FilterResult, error) {
	filter.Condition(filter.Builder().Eq("type", fftypes.IdentityTypeOrg))
	filter.Condition(filter.Builder().Eq("namespace", fftypes.SystemNamespace))
	return nm.database.GetIdentities(ctx, filter)
}

func (nm *networkMap) GetNodeByID(ctx context.Context, id string) (*fftypes.Identity, error) {
	u, err := fftypes.ParseUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	n, err := nm.database.GetIdentityByID(ctx, u)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	if n.Type != fftypes.IdentityTypeNode {
		log.L(ctx).Warnf("Identity '%s' (%s) is not a node identity", n.DID, n.ID)
		return nil, nil
	}
	return n, nil
}

func (nm *networkMap) GetNodes(ctx context.Context, filter database.AndFilter) ([]*fftypes.Identity, *database.FilterResult, error) {
	filter.Condition(filter.Builder().Eq("type", fftypes.IdentityTypeNode))
	filter.Condition(filter.Builder().Eq("namespace", fftypes.SystemNamespace))
	return nm.database.GetIdentities(ctx, filter)
}

func (nm *networkMap) GetIdentityByID(ctx context.Context, ns, id string) (*fftypes.Identity, error) {
	u, err := fftypes.ParseUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	identity, err := nm.database.GetIdentityByID(ctx, u)
	if err != nil {
		return nil, err
	}
	if identity == nil || identity.Namespace != ns {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	return identity, nil
}

func (nm *networkMap) GetIdentities(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Identity, *database.FilterResult, error) {
	filter.Condition(filter.Builder().Eq("namespace", ns))
	return nm.database.GetIdentities(ctx, filter)
}

func (nm *networkMap) GetIdentityVerifiers(ctx context.Context, ns, id string, filter database.AndFilter) ([]*fftypes.Verifier, *database.FilterResult, error) {
	identity, err := nm.GetIdentityByID(ctx, ns, id)
	if err != nil {
		return nil, nil, err
	}
	filter.Condition(filter.Builder().Eq("identity", identity.ID))
	return nm.database.GetVerifiers(ctx, filter)
}
