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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
)

func (nm *networkMap) RegisterIdentity(ctx context.Context, dto *core.IdentityCreateDTO, waitConfirm bool) (identity *core.Identity, err error) {

	// The parent can be a UUID directly
	var parent *fftypes.UUID
	if dto.Parent != "" {
		parent, err = fftypes.ParseUUID(ctx, dto.Parent)
		if err != nil {
			// Or a DID
			parentIdentity, _, err := nm.identity.CachedIdentityLookupMustExist(ctx, dto.Parent)
			if err != nil {
				return nil, err
			}
			parent = parentIdentity.ID
		}
	}

	// Parse the input DTO
	identity = &core.Identity{
		IdentityBase: core.IdentityBase{
			ID:        fftypes.NewUUID(),
			Namespace: nm.namespace,
			Name:      dto.Name,
			Type:      dto.Type,
			Parent:    parent,
		},
		IdentityProfile: core.IdentityProfile{
			Description: dto.Description,
			Profile:     dto.Profile,
		},
	}

	if err := nm.data.VerifyNamespaceExists(ctx, nm.namespace); err != nil {
		return nil, err
	}

	// Set defaults
	if identity.Type == "" {
		identity.Type = core.IdentityTypeCustom
	}

	identity.DID, _ = identity.GenerateDID(ctx)

	// Verify the chain
	immediateParent, _, err := nm.identity.VerifyIdentityChain(ctx, identity)
	if err != nil {
		return nil, err
	}

	// Resolve if we need to perform a validation
	var parentSigner *core.SignerRef
	if immediateParent != nil {
		parentSigner, err = nm.identity.ResolveIdentitySigner(ctx, immediateParent)
		if err != nil {
			return nil, err
		}
	}

	// Determine claim signer
	var claimSigner *core.SignerRef
	if dto.Type == core.IdentityTypeNode {
		// Nodes are special - as they need the claim to be signed directly by the parent
		claimSigner = parentSigner
		parentSigner = nil
	} else {
		if dto.Key == "" {
			return nil, i18n.NewError(ctx, coremsgs.MsgBlockchainKeyNotSet)
		}
		claimSigner = &core.SignerRef{
			Key: dto.Key,
		}
		claimSigner.Author = identity.DID
	}

	if waitConfirm {
		return nm.syncasync.WaitForIdentity(ctx, identity.ID, func(ctx context.Context) error {
			return nm.sendIdentityRequest(ctx, identity, claimSigner, parentSigner)
		})
	}
	err = nm.sendIdentityRequest(ctx, identity, claimSigner, parentSigner)
	if err != nil {
		return nil, err
	}
	return identity, nil
}

func (nm *networkMap) sendIdentityRequest(ctx context.Context, identity *core.Identity, claimSigner *core.SignerRef, parentSigner *core.SignerRef) error {

	// Send the claim - we disable the check on the DID author here, as we are registering the identity so it will not exist
	claimMsg, err := nm.broadcast.BroadcastIdentityClaim(ctx, identity.Namespace, &core.IdentityClaim{
		Identity: identity,
	}, claimSigner, core.SystemTagIdentityClaim, false)
	if err != nil {
		return err
	}
	identity.Messages.Claim = claimMsg.Header.ID

	// Send the verification if one is required.
	if parentSigner != nil {
		verifyMsg, err := nm.broadcast.BroadcastDefinition(ctx, identity.Namespace, &core.IdentityVerification{
			Claim: core.MessageRef{
				ID:   claimMsg.Header.ID,
				Hash: claimMsg.Hash,
			},
			Identity: identity.IdentityBase,
		}, parentSigner, core.SystemTagIdentityVerification, false)
		if err != nil {
			return err
		}
		identity.Messages.Verification = verifyMsg.Header.ID
	}
	return nil
}
