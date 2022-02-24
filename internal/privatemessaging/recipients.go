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

package privatemessaging

import (
	"context"
	"fmt"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (pm *privateMessaging) resolveRecipientList(ctx context.Context, in *fftypes.MessageInOut) error {
	if in.Header.Group != nil {
		log.L(ctx).Debugf("Group '%s' specified for message", in.Header.Group)
		group, err := pm.database.GetGroupByHash(ctx, in.Header.Group)
		if err != nil {
			return err
		}
		if group == nil {
			return i18n.NewError(ctx, i18n.MsgGroupNotFound, in.Header.Group)
		}
		// We have a group already resolved
		return nil
	}
	if in.Group == nil || len(in.Group.Members) == 0 {
		return i18n.NewError(ctx, i18n.MsgGroupMustHaveMembers)
	}
	group, isNew, err := pm.findOrGenerateGroup(ctx, in)
	if err != nil {
		return err
	}
	log.L(ctx).Debugf("Resolved group '%s' for message. New=%t", group.Hash, isNew)
	in.Message.Header.Group = group.Hash

	// If the group is new, we need to do a group initialization, before we send the message itself.
	if isNew {
		return pm.groupManager.groupInit(ctx, &in.Header.SignerRef, group)
	}
	return err
}

func (pm *privateMessaging) resolveNode(ctx context.Context, org *fftypes.Identity, nodeInput string) (node *fftypes.Identity, err error) {
	retryable := true
	if nodeInput != "" {
		node, retryable, err = pm.identity.CachedIdentityLookup(ctx, nodeInput)
	} else {
		// Find any node owned by this organization
		var nodes []*fftypes.Identity
		originalOrgName := fmt.Sprintf("%s/%s", org.Name, org.ID)
		for org != nil && node == nil {
			fb := database.IdentityQueryFactory.NewFilterLimit(ctx, 1)
			filter := fb.And(
				fb.Eq("parent", org.ID),
				fb.Eq("type", fftypes.IdentityTypeNode),
			)
			nodes, _, err = pm.database.GetIdentities(ctx, filter)
			switch {
			case err == nil && len(nodes) > 0:
				// This org owns a node
				node = nodes[0]
			case err == nil && org.Parent != nil:
				// This org has a parent, maybe that org owns a node
				org, err = pm.identity.CachedIdentityLookupByID(ctx, org.Parent)
			default:
				return nil, i18n.NewError(ctx, i18n.MsgNodeNotFoundInOrg, originalOrgName)
			}
		}
	}
	if err != nil && retryable {
		return nil, err
	}
	if node == nil {
		return nil, i18n.NewError(ctx, i18n.MsgNodeNotFound, nodeInput)
	}
	return node, nil
}

func (pm *privateMessaging) getRecipients(ctx context.Context, in *fftypes.MessageInOut) (gi *fftypes.GroupIdentity, err error) {

	localOrg, err := pm.identity.GetNodeOwnerOrg(ctx)
	if err != nil {
		return nil, err
	}

	foundLocal := false
	gi = &fftypes.GroupIdentity{
		Namespace: in.Message.Header.Namespace,
		Name:      in.Group.Name,
		Ledger:    in.Group.Ledger,
		Members:   make(fftypes.Members, len(in.Group.Members)),
	}
	for i, rInput := range in.Group.Members {
		// Resolve the org
		org, _, err := pm.identity.CachedIdentityLookup(ctx, rInput.Identity)
		if err != nil {
			return nil, err
		}
		// Resolve the node
		node, err := pm.resolveNode(ctx, org, rInput.Node)
		if err != nil {
			return nil, err
		}
		foundLocal = foundLocal || (node.Parent.Equals(localOrg.ID) && node.Name == pm.localNodeName)
		gi.Members[i] = &fftypes.Member{
			Identity: org.DID,
			Node:     node.ID,
		}
	}
	if !foundLocal {
		// Add in the local org identity
		localNodeID, err := pm.resolveLocalNode(ctx, localOrg)
		if err != nil {
			return nil, err
		}
		gi.Members = append(gi.Members, &fftypes.Member{
			Identity: localOrg.DID,
			Node:     localNodeID,
		})
	}
	return gi, nil
}

func (pm *privateMessaging) resolveLocalNode(ctx context.Context, localOrg *fftypes.Identity) (*fftypes.UUID, error) {
	if pm.localNodeID != nil {
		return pm.localNodeID, nil
	}
	fb := database.IdentityQueryFactory.NewFilterLimit(ctx, 1)
	filter := fb.And(
		fb.Eq("parent", localOrg.ID),
		fb.Eq("type", fftypes.IdentityTypeNode),
		fb.Eq("name", pm.localNodeName),
	)
	nodes, _, err := pm.database.GetIdentities(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, i18n.NewError(ctx, i18n.MsgLocalNodeResolveFailed)
	}
	pm.localNodeID = nodes[0].ID
	return pm.localNodeID, nil
}

func (pm *privateMessaging) findOrGenerateGroup(ctx context.Context, in *fftypes.MessageInOut) (group *fftypes.Group, isNew bool, err error) {
	gi, err := pm.getRecipients(ctx, in)
	if err != nil {
		return nil, false, err
	}

	// Create the group structure, and seal it - which will sort the members, and
	// generate the deterministic hash. We then search on that group to see if it
	// exists. If it doesn't, we go ahead and create it. If it does - we don't return
	// this candidate - we return the existing group.
	newCandidate := &fftypes.Group{
		GroupIdentity: *gi,
		Created:       fftypes.Now(),
	}
	newCandidate.Seal()

	filter := database.GroupQueryFactory.NewFilterLimit(ctx, 1).Eq("hash", newCandidate.Hash)
	groups, _, err := pm.database.GetGroups(ctx, filter)
	if err != nil {
		return nil, false, err
	}
	if len(groups) > 0 {
		return groups[0], false, nil
	}
	return newCandidate, true, nil
}
