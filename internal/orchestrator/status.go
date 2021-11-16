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

package orchestrator

import (
	"context"

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
		log.L(or.ctx).Infof("Node not yet registered")
	} else {
		or.node = status.Node.ID
	}
	return or.node
}

func (or *orchestrator) GetStatus(ctx context.Context) (status *fftypes.NodeStatus, err error) {

	orgKey := or.identity.GetOrgKey(ctx)
	status = &fftypes.NodeStatus{
		Node: fftypes.NodeStatusNode{
			Name: config.GetString(config.NodeName),
		},
		Org: fftypes.NodeStatusOrg{
			Name:     config.GetString(config.OrgName),
			Identity: orgKey,
		},
		Defaults: fftypes.NodeStatusDefaults{
			Namespace: config.GetString(config.NamespacesDefault),
		},
	}

	org, err := or.database.GetOrganizationByName(ctx, status.Org.Name)
	if err != nil {
		return nil, err
	}
	if org != nil {
		status.Org.Registered = true
		status.Org.ID = org.ID
		status.Org.Identity = org.Identity

		node, err := or.database.GetNode(ctx, org.Identity, status.Node.Name)
		if err != nil {
			return nil, err
		}

		if node != nil {
			status.Node.Registered = true
			status.Node.ID = node.ID
		}
	}

	return status, nil
}
