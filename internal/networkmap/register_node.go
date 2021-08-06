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

func (nm *networkMap) RegisterNode(ctx context.Context, waitConfirm bool) (msg *fftypes.Message, err error) {

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Created:     fftypes.Now(),
		Owner:       config.GetString(config.OrgIdentity),
		Name:        config.GetString(config.NodeName),
		Description: config.GetString(config.NodeDescription),
	}
	if node.Name == "" {
		node.Name = config.GetString(config.OrgIdentity)
	}
	if node.Owner == "" || node.Name == "" {
		return nil, i18n.NewError(ctx, i18n.MsgNodeAndOrgIDMustBeSet)
	}

	node.DX.Peer, node.DX.Endpoint, err = nm.exchange.GetEndpointInfo(ctx)
	if err != nil {
		return nil, err
	}

	err = node.Validate(ctx, false)
	if err != nil {
		return nil, err
	}

	if err = nm.findOrgsToRoot(ctx, "node", node.Name, node.Owner); err != nil {
		return nil, err
	}

	signingIdentity, err := nm.identity.Resolve(ctx, node.Owner)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgInvalidSigningIdentity)
	}

	return nm.broadcast.BroadcastDefinition(ctx, node, signingIdentity, fftypes.SystemTagDefineNode, waitConfirm)
}
