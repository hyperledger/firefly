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
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (nm *networkMap) RegisterIdentity(ctx context.Context, ns string, dto *fftypes.IdentityCreateDTO, waitConfirm bool) (identity *fftypes.Identity, err error) {

	// Parse the input DTO
	identity = &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			Namespace: ns,
			Name:      dto.Name,
			Type:      dto.Type,
			Parent:    dto.Parent,
		},
		IdentityProfile: fftypes.IdentityProfile{
			Description: dto.Description,
			Profile:     dto.Profile,
		},
	}

	// Set defaults
	if identity.Namespace == fftypes.SystemNamespace || identity.Namespace == "" {
		identity.Namespace = fftypes.SystemNamespace
		if identity.Type == "" {
			identity.Type = fftypes.IdentityTypeOrg
		}
	} else if identity.Type == "" {
		identity.Type = fftypes.IdentityTypeCustom
	}

	identity.DID, _ = identity.GenerateDID(ctx)

	// Verify the chain
	immediateParent, _, err := nm.identity.VerifyIdentityChain(ctx, identity)
	if err != nil {
		return nil, err
	}

	// Resolve if we need to perform a validation
	var parentSigner *fftypes.SignerRef
	if immediateParent != nil {
		parentSigner, err = nm.identity.ResolveIdentitySigner(ctx, immediateParent)
		if err != nil {
			return nil, err
		}
	}

	// Determin claim signer
	var claimSigner *fftypes.SignerRef
	if dto.Type == fftypes.IdentityTypeNode {
		// Nodes are special - as they need the claim to be signed directly by the parent
		claimSigner = parentSigner
		parentSigner = nil
	} else {
		if dto.Key == "" {
			return nil, i18n.NewError(ctx, i18n.MsgBlockchainKeyNotSet)
		}
		claimSigner = &fftypes.SignerRef{
			Key: dto.Key,
		}
		claimSigner.Author = identity.DID
	}

	// Send the claim - we disable the check on the DID author here, as we are registerin the identity so it will not exist
	claimMsg, err := nm.broadcast.BroadcastDefinitionResolveKeyOnly(ctx, identity.Namespace, &fftypes.IdentityClaim{
		Identity: identity,
	}, claimSigner, fftypes.SystemTagIdentityClaim, waitConfirm)
	if err != nil {
		return nil, err
	}
	identity.Messages.Claim = claimMsg.Header.ID

	// Send the verification if one is required.
	if parentSigner != nil {
		verifyMsg, err := nm.broadcast.BroadcastDefinition(ctx, identity.Namespace, &fftypes.IdentityVerification{
			Claim: fftypes.MessageRef{
				ID:   claimMsg.Header.ID,
				Hash: claimMsg.Hash,
			},
			Identity: identity.IdentityBase,
		}, parentSigner, fftypes.SystemTagIdentityVerification, false /* confirm is on the claim only */)
		if err != nil {
			return nil, err
		}
		identity.Messages.Verification = verifyMsg.Header.ID
	}

	return identity, err
}
