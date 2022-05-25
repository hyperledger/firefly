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

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/pkg/core"
)

func (nm *networkMap) RegisterNode(ctx context.Context, waitConfirm bool) (identity *core.Identity, err error) {

	nodeOwningOrg, err := nm.identity.GetNodeOwnerOrg(ctx)
	if err != nil {
		return nil, err
	}

	nodeRequest := &core.IdentityCreateDTO{
		Parent: nodeOwningOrg.ID.String(),
		Name:   config.GetString(coreconfig.NodeName),
		Type:   core.IdentityTypeNode,
		IdentityProfile: core.IdentityProfile{
			Description: config.GetString(coreconfig.NodeDescription),
		},
	}
	if nodeRequest.Name == "" {
		if nodeOwningOrg.Name != "" {
			nodeRequest.Name = fmt.Sprintf("%s.node", nodeOwningOrg.Name)
		}
	}

	dxInfo, err := nm.exchange.GetEndpointInfo(ctx)
	if err != nil {
		return nil, err
	}
	nodeRequest.Profile = dxInfo

	return nm.RegisterIdentity(ctx, core.SystemNamespace, nodeRequest, waitConfirm)
}
