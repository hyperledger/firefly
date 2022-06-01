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

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
)

// RegisterNodeOrganization is a convenience helper to register the org configured on the node, without any extra info
func (nm *networkMap) RegisterNodeOrganization(ctx context.Context, ns string, waitConfirm bool) (*core.Identity, error) {

	key, err := nm.identity.GetNodeOwnerBlockchainKey(ctx, ns)
	if err != nil {
		return nil, err
	}

	orgRequest := &core.IdentityCreateDTO{
		Name: nm.namespace.GetMultipartyConfig(ns, coreconfig.OrgName),
		IdentityProfile: core.IdentityProfile{
			Description: nm.namespace.GetMultipartyConfig(ns, coreconfig.OrgDescription),
		},
		Key: key.Value,
	}
	if orgRequest.Name == "" {
		return nil, i18n.NewError(ctx, coremsgs.MsgNodeAndOrgIDMustBeSet)
	}
	return nm.RegisterOrganization(ctx, ns, orgRequest, waitConfirm)
}

func (nm *networkMap) RegisterOrganization(ctx context.Context, ns string, orgRequest *core.IdentityCreateDTO, waitConfirm bool) (*core.Identity, error) {
	orgRequest.Type = core.IdentityTypeOrg
	return nm.RegisterIdentity(ctx, ns, orgRequest, waitConfirm)
}
