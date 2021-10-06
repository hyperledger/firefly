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
	"fmt"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (nm *networkMap) RegisterNode(ctx context.Context, waitConfirm bool) (node *fftypes.Node, msg *fftypes.Message, err error) {

	localOrgSigningKey, err := nm.getLocalOrgSigningKey(ctx)
	if err != nil {
		return nil, nil, err
	}

	node = &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Created:     fftypes.Now(),
		Owner:       localOrgSigningKey, // TODO: Switch hierarchy to DID based, not signing key. Introducing an intermediate identity object
		Name:        config.GetString(config.NodeName),
		Description: config.GetString(config.NodeDescription),
	}
	if node.Name == "" {
		orgName := config.GetString(config.OrgName)
		if orgName != "" {
			node.Name = fmt.Sprintf("%s.node", orgName)
		}
	}
	if node.Owner == "" || node.Name == "" {
		return nil, nil, i18n.NewError(ctx, i18n.MsgNodeAndOrgIDMustBeSet)
	}

	node.DX.Peer, node.DX.Endpoint, err = nm.exchange.GetEndpointInfo(ctx)
	if err != nil {
		return nil, nil, err
	}

	err = node.Validate(ctx, false)
	if err != nil {
		return nil, nil, err
	}

	if err = nm.findOrgsToRoot(ctx, "node", node.Name, node.Owner); err != nil {
		return nil, nil, err
	}

	msg, err = nm.broadcast.BroadcastDefinitionAsNode(ctx, node, fftypes.SystemTagDefineNode, waitConfirm)
	if msg != nil {
		node.Message = msg.Header.ID
	}
	return node, msg, err
}
