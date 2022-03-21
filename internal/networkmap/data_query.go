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

func (nm *networkMap) GetOrganizationByNameOrID(ctx context.Context, nameOrID string) (org *fftypes.Identity, err error) {
	u, err := fftypes.ParseUUID(ctx, nameOrID)
	if err != nil {
		if err := fftypes.ValidateFFNameField(ctx, nameOrID, "name"); err != nil {
			return nil, err
		}
		if org, err = nm.database.GetIdentityByName(ctx, fftypes.IdentityTypeOrg, fftypes.SystemNamespace, nameOrID); err != nil {
			return nil, err
		}
	} else if org, err = nm.database.GetIdentityByID(ctx, u); err != nil {
		return nil, err
	}
	if org == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	if org.Type != fftypes.IdentityTypeOrg {
		log.L(ctx).Warnf("Identity '%s' (%s) is not an org identity", org.DID, org.ID)
		return nil, nil
	}
	return org, nil
}

func (nm *networkMap) GetOrganizations(ctx context.Context, filter database.AndFilter) ([]*fftypes.Identity, *database.FilterResult, error) {
	filter.Condition(filter.Builder().Eq("type", fftypes.IdentityTypeOrg))
	filter.Condition(filter.Builder().Eq("namespace", fftypes.SystemNamespace))
	return nm.database.GetIdentities(ctx, filter)
}

func (nm *networkMap) GetNodeByNameOrID(ctx context.Context, nameOrID string) (node *fftypes.Identity, err error) {
	u, err := fftypes.ParseUUID(ctx, nameOrID)
	if err != nil {
		if err := fftypes.ValidateFFNameField(ctx, nameOrID, "name"); err != nil {
			return nil, err
		}
		if node, err = nm.database.GetIdentityByName(ctx, fftypes.IdentityTypeNode, fftypes.SystemNamespace, nameOrID); err != nil {
			return nil, err
		}
	} else if node, err = nm.database.GetIdentityByID(ctx, u); err != nil {
		return nil, err
	}
	if node == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	if node.Type != fftypes.IdentityTypeNode {
		log.L(ctx).Warnf("Identity '%s' (%s) is not a node identity", node.DID, node.ID)
		return nil, nil
	}
	return node, nil
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

func (nm *networkMap) GetIdentityByDID(ctx context.Context, did string) (*fftypes.Identity, error) {
	fb := database.IdentityQueryFactory.NewFilter(ctx)
	filter := fb.And().Condition(fb.Eq("did", did))
	results, _, err := nm.database.GetIdentities(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	return results[0], nil
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

func (nm *networkMap) GetDIDDocForIndentityByID(ctx context.Context, ns, id string) (*DIDDocument, error) {
	identity, err := nm.GetIdentityByID(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	return nm.generateDIDDocument(ctx, identity)
}

func (nm *networkMap) GetVerifierByHash(ctx context.Context, ns, hash string) (*fftypes.Verifier, error) {
	b32, err := fftypes.ParseBytes32(ctx, hash)
	if err != nil {
		return nil, err
	}
	verifier, err := nm.database.GetVerifierByHash(ctx, b32)
	if err != nil {
		return nil, err
	}
	if verifier == nil || verifier.Namespace != ns {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	return verifier, nil
}

func (nm *networkMap) GetVerifiers(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Verifier, *database.FilterResult, error) {
	filter.Condition(filter.Builder().Eq("namespace", ns))
	return nm.database.GetVerifiers(ctx, filter)
}
