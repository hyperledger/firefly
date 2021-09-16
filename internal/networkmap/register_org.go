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

package networkmap

import (
	"context"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (nm *networkMap) findOrgsToRoot(ctx context.Context, idType, identity, parent string) (err error) {

	var root *fftypes.Organization
	for parent != "" {
		root, err = nm.database.GetOrganizationByIdentity(ctx, parent)
		if err != nil {
			return err
		}
		if root == nil {
			return i18n.NewError(ctx, i18n.MsgParentIdentityNotFound, parent, idType, identity)
		}
		parent = root.Parent
	}
	return err
}

func (nm *networkMap) getLocalOrgSigningKey(ctx context.Context) (localOrgSigningKey string, err error) {
	localOrgSigningKey = config.GetString(config.OrgKey)
	if localOrgSigningKey == "" {
		localOrgSigningKey = config.GetString(config.OrgIdentityDeprecated)
	}
	localOrgSigningKey, err = nm.identity.ResolveSigningKey(ctx, localOrgSigningKey)
	if err != nil {
		return "", err
	}
	if localOrgSigningKey == "" {
		return "", i18n.NewError(ctx, i18n.MsgNodeAndOrgIDMustBeSet)
	}
	return localOrgSigningKey, nil
}

// RegisterNodeOrganization is a convenience helper to register the org configured on the node, without any extra info
func (nm *networkMap) RegisterNodeOrganization(ctx context.Context, waitConfirm bool) (org *fftypes.Organization, msg *fftypes.Message, err error) {

	localOrgSigningKey, err := nm.getLocalOrgSigningKey(ctx)
	if err != nil {
		return nil, nil, err
	}

	org = &fftypes.Organization{
		Name:        config.GetString(config.OrgName),
		Identity:    localOrgSigningKey, // TODO: Switch hierarchy to DID based, not signing key. Introducing an intermediate identity object
		Description: config.GetString(config.OrgDescription),
	}
	if org.Identity == "" || org.Name == "" {
		return nil, nil, i18n.NewError(ctx, i18n.MsgNodeAndOrgIDMustBeSet)
	}
	msg, err = nm.RegisterOrganization(ctx, org, waitConfirm)
	if msg != nil {
		org.Message = msg.Header.ID
	}
	return org, msg, err
}

func (nm *networkMap) RegisterOrganization(ctx context.Context, org *fftypes.Organization, waitConfirm bool) (*fftypes.Message, error) {

	err := org.Validate(ctx, false)
	if err != nil {
		return nil, err
	}
	org.ID = fftypes.NewUUID()
	org.Created = fftypes.Now()

	// If we're a root identity, we self-sign
	signingIdentityString := org.Identity
	if org.Parent != "" {
		// Check the identity itself is ok
		if err = nm.identity.ResolveInputIdentity(ctx, &fftypes.Identity{
			Key: signingIdentityString,
		}); err != nil {
			return nil, err
		}

		// Otherwise we must have access to the signing key of the parent, and the parents
		// must already have been broadcast to the network
		signingIdentityString = org.Parent
		if err = nm.findOrgsToRoot(ctx, "organization", org.Identity, signingIdentityString); err != nil {
			return nil, err
		}
	}

	return nm.broadcast.BroadcastDefinition(ctx, org, &fftypes.Identity{
		Key: signingIdentityString,
	}, fftypes.SystemTagDefineOrganization, waitConfirm)
}
