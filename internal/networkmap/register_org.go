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

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

// RegisterNodeOrganization is a convenience helper to register the org configured on the node, without any extra info
func (nm *networkMap) RegisterNodeOrganization(ctx context.Context, waitConfirm bool) (*fftypes.Identity, error) {

	orgRequest := &fftypes.IdentityCreateDTO{
		Name: config.GetString(config.OrgName),
		IdentityProfile: fftypes.IdentityProfile{
			Description: config.GetString(config.OrgDescription),
		},
	}
	if orgRequest.Name == "" {
		return nil, i18n.NewError(ctx, i18n.MsgNodeAndOrgIDMustBeSet)
	}
	return nm.RegisterOrganization(ctx, orgRequest, waitConfirm)
}

func (nm *networkMap) RegisterOrganization(ctx context.Context, orgRequest *fftypes.IdentityCreateDTO, waitConfirm bool) (*fftypes.Identity, error) {

	orgRequest.Namespace = fftypes.SystemNamespace
	orgRequest.Type = fftypes.IdentityTypeOrg

	return nm.RegisterIdentity(ctx, orgRequest, waitConfirm)
}
