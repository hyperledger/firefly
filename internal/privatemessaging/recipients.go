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

	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

func (pm *privateMessaging) resolveReceipientList(ctx context.Context, in *fftypes.MessageInput) error {
	group, isNew, err := pm.findOrCreateGroup(ctx, in)
	log.L(ctx).Debug("Resolved group. New=%t %+v", isNew, group)
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
			node, err = pm.database.GetNode(ctx, nodeInput)
		}
	} else {
		// Find any node owned by this organization
		var nodes []*fftypes.Node
		for org != nil && node == nil {
			filter := database.NodeQueryFactory.NewFilterLimit(ctx, 1).Eq("owner", org.ID)
			nodes, err = pm.database.GetNodes(ctx, filter)
			if err == nil && len(nodes) > 0 {
				// This org owns a node
				node = nodes[0]
			} else if err == nil && org.Parent != "" {
				// This org has a parent, maybe that org owns a node
				org, err = pm.database.GetOrganizationByIdentity(ctx, org.Parent)
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

func (pm *privateMessaging) getReceipients(ctx context.Context, in *fftypes.MessageInput) (recipients fftypes.Recipients, err error) {
	if len(in.Recipients) == 0 {
		return nil, i18n.NewError(ctx, i18n.MsgGroupMustHaveRecipients)
	}
	recipients = make(fftypes.Recipients, len(in.Recipients))
	for i, rInput := range in.Recipients {
		// Resolve the org
		org, err := pm.resolveOrg(ctx, rInput.Org)
		if err != nil {
			return nil, err
		}
		// Resolve the node
		node, err := pm.resolveNode(ctx, org, rInput.Node)
		if err != nil {
			return nil, err
		}
		recipients[i] = &fftypes.Recipient{
			Org:  org.ID,
			Node: node.ID,
		}
	}
	return recipients, nil
}

func (pm *privateMessaging) findOrCreateGroup(ctx context.Context, in *fftypes.MessageInput) (group *fftypes.Group, isNew bool, err error) {
	recipients, err := pm.getReceipients(ctx, in)
	if err != nil {
		return nil, false, err
	}
	fb := database.GroupQueryFactory.NewFilterLimit(ctx, 1)
	hash := recipients.Hash()
	filter := fb.And(
		fb.Eq("namespace", in.Header.Namespace),
		fb.Eq("ledger", in.Header.TX.Ledger),
		fb.Eq("hahs", hash),
	)
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
		ID:         fftypes.NewUUID(),
		Namespace:  in.Header.Namespace,
		Ledger:     in.Header.TX.Ledger,
		Hash:       hash,
		Recipients: recipients,
		Created:    fftypes.Now(),
	}
	return group, true, nil
}
