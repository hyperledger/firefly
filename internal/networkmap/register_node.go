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
	"fmt"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (nm *networkMap) RegisterNode(ctx context.Context, waitConfirm bool) (node *fftypes.Identity, err error) {

	nodeOwningOrg, err := nm.identity.GetNodeOwnerOrg(ctx)
	if err != nil {
		return nil, err
	}

	// There is only one message for a node broadcast, as there is no second blockchain key
	nodeClaim := &fftypes.IdentityClaim{
		Identity: fftypes.Identity{
			IdentityBase: fftypes.IdentityBase{
				ID:        fftypes.NewUUID(),
				Parent:    nodeOwningOrg.ID,
				Namespace: fftypes.SystemNamespace,
				Name:      config.GetString(config.NodeName),
			},
			IdentityProfile: fftypes.IdentityProfile{
				Description: config.GetString(config.NodeDescription),
			},
		},
	}
	node = &nodeClaim.Identity
	node.DID, _ = node.GenerateDID(ctx)
	if node.Name == "" {
		if nodeOwningOrg.Name != "" {
			node.Name = fmt.Sprintf("%s.node", nodeOwningOrg.Name)
		}
	}

	dxInfo, err := nm.exchange.GetEndpointInfo(ctx)
	if err != nil {
		return nil, err
	}
	node.Profile = dxInfo.Endpoint

	if _, err = nm.identity.VerifyIdentityChain(ctx, node); err != nil {
		return nil, err
	}

	msg, err = nm.broadcast.BroadcastDefinitionAsNode(ctx, fftypes.SystemNamespace, nodeClaim, fftypes.SystemTagDefineNode, waitConfirm)
	if msg != nil {
		node.Message = msg.Header.ID
	}
	return node, msg, err
}
