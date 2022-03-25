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

package orchestrator

import (
	"context"
	"fmt"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (or *orchestrator) GetNodeUUID(ctx context.Context) (node *fftypes.UUID) {
	if or.node != nil {
		return or.node
	}
	status, err := or.GetStatus(ctx)
	if err != nil {
		log.L(or.ctx).Warnf("Failed to query local node UUID: %s", err)
		return nil
	}
	if status.Node.Registered {
		or.node = status.Node.ID
	} else {
		log.L(or.ctx).Infof("Node not yet registered")
	}
	return or.node
}

func (or *orchestrator) GetStatus(ctx context.Context) (status *fftypes.NodeStatus, err error) {

	org, err := or.identity.GetNodeOwnerOrg(ctx)
	if err != nil {
		log.L(ctx).Warnf("Failed to query local org for status: %s", err)
	}
	status = &fftypes.NodeStatus{
		Node: fftypes.NodeStatusNode{
			Name: config.GetString(config.NodeName),
		},
		Org: fftypes.NodeStatusOrg{
			Name: config.GetString(config.OrgName),
		},
		Defaults: fftypes.NodeStatusDefaults{
			Namespace: config.GetString(config.NamespacesDefault),
		},
	}

	if org != nil {
		status.Org.Registered = true
		status.Org.ID = org.ID
		status.Org.DID = org.DID

		node, _, err := or.identity.CachedIdentityLookupNilOK(ctx, fmt.Sprintf("%s%s", fftypes.FireFlyNodeDIDPrefix, status.Node.Name))
		if err != nil {
			return nil, err
		}
		if node != nil && !node.Parent.Equals(org.ID) {
			log.L(ctx).Errorf("Specified node name is in use by another org: %s", err)
			node = nil
		}
		if node != nil {
			status.Node.Registered = true
			status.Node.ID = node.ID
		}
	}

	return status, nil
}
