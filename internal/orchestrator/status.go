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

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (or *orchestrator) GetStatus(ctx context.Context) (status *fftypes.NodeStatus, err error) {

	orgKey := config.GetString(config.OrgKey)
	if orgKey == "" {
		orgKey = config.GetString(config.OrgIdentityDeprecated)
	}
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
