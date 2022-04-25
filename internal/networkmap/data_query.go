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
	"database/sql/driver"

	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/i18n"
	"github.com/hyperledger/firefly/pkg/log"
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
		return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
	}
	if org.Type != fftypes.IdentityTypeOrg {
		log.L(ctx).Warnf("Identity '%s' (%s) is not an org identity", org.DID, org.ID)
		return nil, nil
	}
	return org, nil
}

func (nm *networkMap) GetOrganizations(ctx context.Context, filter database.AndFilter) ([]*fftypes.Identity, *database.FilterResult, error) {
	filter.Condition(filter.Builder().Eq("type", fftypes.IdentityTypeOrg))
	return nm.GetIdentities(ctx, fftypes.SystemNamespace, filter)
}

func (nm *networkMap) GetOrganizationsWithVerifiers(ctx context.Context, filter database.AndFilter) ([]*fftypes.IdentityWithVerifiers, *database.FilterResult, error) {
	filter.Condition(filter.Builder().Eq("type", fftypes.IdentityTypeOrg))
	return nm.GetIdentitiesWithVerifiers(ctx, fftypes.SystemNamespace, filter)
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
		return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
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
		return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
	}
	return identity, nil
}

func (nm *networkMap) withVerifiers(ctx context.Context, identity *fftypes.Identity) (*fftypes.IdentityWithVerifiers, error) {
	fb := database.VerifierQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("namespace", identity.Namespace),
		fb.Eq("identity", identity.ID),
	)
	verifiers, _, err := nm.database.GetVerifiers(ctx, filter)
	if err != nil {
		return nil, err
	}
	refs := make([]*fftypes.VerifierRef, len(verifiers))
	for i, v := range verifiers {
		refs[i] = &v.VerifierRef
	}
	return &fftypes.IdentityWithVerifiers{
		Identity:  *identity,
		Verifiers: refs,
	}, nil
}

func (nm *networkMap) GetIdentityByIDWithVerifiers(ctx context.Context, ns, id string) (*fftypes.IdentityWithVerifiers, error) {
	identity, err := nm.GetIdentityByID(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	return nm.withVerifiers(ctx, identity)
}

func (nm *networkMap) GetIdentityByDID(ctx context.Context, did string) (*fftypes.Identity, error) {
	identity, _, err := nm.identity.CachedIdentityLookupMustExist(ctx, did)
	if err != nil {
		return nil, err
	}
	return identity, nil
}

func (nm *networkMap) GetIdentityByDIDWithVerifiers(ctx context.Context, did string) (*fftypes.IdentityWithVerifiers, error) {
	identity, _, err := nm.identity.CachedIdentityLookupMustExist(ctx, did)
	if err != nil {
		return nil, err
	}
	return nm.withVerifiers(ctx, identity)
}

func (nm *networkMap) GetIdentities(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Identity, *database.FilterResult, error) {
	filter.Condition(filter.Builder().Eq("namespace", ns))
	return nm.database.GetIdentities(ctx, filter)
}

func (nm *networkMap) GetIdentitiesGlobal(ctx context.Context, filter database.AndFilter) ([]*fftypes.Identity, *database.FilterResult, error) {
	return nm.database.GetIdentities(ctx, filter)
}

func (nm *networkMap) GetIdentitiesWithVerifiers(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.IdentityWithVerifiers, *database.FilterResult, error) {
	filter.Condition(filter.Builder().Eq("namespace", ns))
	return nm.GetIdentitiesWithVerifiersGlobal(ctx, filter)
}

func (nm *networkMap) GetIdentitiesWithVerifiersGlobal(ctx context.Context, filter database.AndFilter) ([]*fftypes.IdentityWithVerifiers, *database.FilterResult, error) {
	identities, res, err := nm.database.GetIdentities(ctx, filter)
	if err != nil {
		return nil, nil, err
	}
	iids := make([]driver.Value, len(identities))
	for idx, identity := range identities {
		iids[idx] = identity.ID
	}
	fb := database.VerifierQueryFactory.NewFilter(ctx)
	verifierFilter := fb.And(
		fb.In("identity", iids),
	)
	idsWithVerifiers := make([]*fftypes.IdentityWithVerifiers, len(identities))
	verifiers, _, err := nm.database.GetVerifiers(ctx, verifierFilter)
	if err != nil {
		return nil, nil, err
	}
	for idx, identity := range identities {
		idsWithVerifiers[idx] = &fftypes.IdentityWithVerifiers{
			Identity:  *identity,
			Verifiers: make([]*fftypes.VerifierRef, 0, 1),
		}
		for _, verifier := range verifiers {
			if verifier.Identity.Equals(identity.ID) {
				idsWithVerifiers[idx].Verifiers = append(idsWithVerifiers[idx].Verifiers, &verifier.VerifierRef)
			}
		}
	}
	return idsWithVerifiers, res, err
}

func (nm *networkMap) GetIdentityVerifiers(ctx context.Context, ns, id string, filter database.AndFilter) ([]*fftypes.Verifier, *database.FilterResult, error) {
	identity, err := nm.GetIdentityByID(ctx, ns, id)
	if err != nil {
		return nil, nil, err
	}
	filter.Condition(filter.Builder().Eq("identity", identity.ID))
	return nm.database.GetVerifiers(ctx, filter)
}

func (nm *networkMap) GetDIDDocForIdentityByID(ctx context.Context, ns, id string) (*DIDDocument, error) {
	identity, err := nm.GetIdentityByID(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	return nm.generateDIDDocument(ctx, identity)
}

func (nm *networkMap) GetDIDDocForIdentityByDID(ctx context.Context, did string) (*DIDDocument, error) {
	identity, err := nm.GetIdentityByDID(ctx, did)
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
		return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
	}
	return verifier, nil
}

func (nm *networkMap) GetVerifiers(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Verifier, *database.FilterResult, error) {
	filter.Condition(filter.Builder().Eq("namespace", ns))
	return nm.database.GetVerifiers(ctx, filter)
}
