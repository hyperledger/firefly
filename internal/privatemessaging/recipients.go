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

package privatemessaging

import (
	"context"
	"fmt"

	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (pm *privateMessaging) resolveReceipientList(ctx context.Context, sender *fftypes.Identity, in *fftypes.MessageInput) error {
	if in.Header.Group != nil {
		log.L(ctx).Debugf("Group '%s' specified for message", in.Header.Group)
		return nil // validity of existing group checked later
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
		return pm.groupManager.groupInit(ctx, sender, group)
	}
	return err
}

func (pm *privateMessaging) resolveOrg(ctx context.Context, orgInput string) (org *fftypes.Organization, err error) {
	orgID, err := fftypes.ParseUUID(ctx, orgInput)
	if err == nil {
		org, err = pm.database.GetOrganizationByID(ctx, orgID)
	} else {
		org, err = pm.database.GetOrganizationByName(ctx, orgInput)
		if err == nil && org == nil {
			org, err = pm.database.GetOrganizationByIdentity(ctx, orgInput)
		}
	}
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, i18n.NewError(ctx, i18n.MsgOrgNotFound, orgInput)
	}
	return org, nil
}

func (pm *privateMessaging) resolveNode(ctx context.Context, org *fftypes.Organization, nodeInput string) (node *fftypes.Node, err error) {
	if nodeInput != "" {
		var nodeID *fftypes.UUID
		nodeID, err = fftypes.ParseUUID(ctx, nodeInput)
		if err == nil {
			node, err = pm.database.GetNodeByID(ctx, nodeID)
		} else {
			node, err = pm.database.GetNode(ctx, org.Identity, nodeInput)
		}
	} else {
		// Find any node owned by this organization
		var nodes []*fftypes.Node
		originalOrgName := fmt.Sprintf("%s/%s", org.Name, org.Identity)
		for org != nil && node == nil {
			filter := database.NodeQueryFactory.NewFilterLimit(ctx, 1).Eq("owner", org.Identity)
			nodes, err = pm.database.GetNodes(ctx, filter)
			switch {
			case err == nil && len(nodes) > 0:
				// This org owns a node
				node = nodes[0]
			case err == nil && org.Parent != "":
				// This org has a parent, maybe that org owns a node
				org, err = pm.database.GetOrganizationByIdentity(ctx, org.Parent)
			default:
				return nil, i18n.NewError(ctx, i18n.MsgNodeNotFoundInOrg, originalOrgName)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, i18n.NewError(ctx, i18n.MsgNodeNotFound, nodeInput)
	}
	return node, nil
}

func (pm *privateMessaging) getReceipients(ctx context.Context, in *fftypes.MessageInput) (gi *fftypes.GroupIdentity, err error) {
	foundLocal := false
	gi = &fftypes.GroupIdentity{
		Namespace: in.Message.Header.Namespace,
		Name:      in.Group.Name,
		Ledger:    in.Group.Ledger,
		Members:   make(fftypes.Members, len(in.Group.Members)),
	}
	for i, rInput := range in.Group.Members {
		// Resolve the org
		org, err := pm.resolveOrg(ctx, rInput.Identity)
		if err != nil {
			return nil, err
		}
		// Resolve the node
		node, err := pm.resolveNode(ctx, org, rInput.Node)
		if err != nil {
			return nil, err
		}
		foundLocal = foundLocal || (node.Owner == pm.localOrgIdentity && node.Name == pm.localNodeName)
		gi.Members[i] = &fftypes.Member{
			Identity: org.Identity,
			Node:     node.ID,
		}
	}
	if !foundLocal {
		return nil, i18n.NewError(ctx, i18n.MsgOneMemberLocal)
	}
	return gi, nil
}

func (pm *privateMessaging) findOrGenerateGroup(ctx context.Context, in *fftypes.MessageInput) (group *fftypes.Group, isNew bool, err error) {
	gi, err := pm.getReceipients(ctx, in)
	if err != nil {
		return nil, false, err
	}
	hash := gi.Hash()
	filter := database.GroupQueryFactory.NewFilterLimit(ctx, 1).Eq("hash", hash)
	groups, err := pm.database.GetGroups(ctx, filter)
	if err != nil {
		return nil, false, err
	}
	if len(groups) > 0 {
		return groups[0], false, nil
	}

	// Generate a new group on the fly here.
	// It will need to be sent to the group ahead of the message the user is trying to send.
	group = &fftypes.Group{
		GroupIdentity: *gi,
		Hash:          hash,
		Created:       fftypes.Now(),
	}
	group.Seal()
	return group, true, nil
}
