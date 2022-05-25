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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
)

func (nm *networkMap) UpdateIdentity(ctx context.Context, ns, uuidStr string, dto *core.IdentityUpdateDTO, waitConfirm bool) (identity *core.Identity, err error) {
	id, err := fftypes.ParseUUID(ctx, uuidStr)
	if err != nil {
		return nil, err
	}
	return nm.updateIdentityID(ctx, ns, id, dto, waitConfirm)
}

func (nm *networkMap) updateIdentityID(ctx context.Context, ns string, id *fftypes.UUID, dto *core.IdentityUpdateDTO, waitConfirm bool) (identity *core.Identity, err error) {

	// Get the original identity
	identity, err = nm.identity.CachedIdentityLookupByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if identity == nil || identity.Namespace != ns {
		return nil, i18n.NewError(ctx, coremsgs.Msg404NoResult)
	}

	// Resolve the signer of the original claim
	updateSigner, err := nm.identity.ResolveIdentitySigner(ctx, identity)
	if err != nil {
		return nil, err
	}

	identity.IdentityProfile = dto.IdentityProfile
	if err := identity.Validate(ctx); err != nil {
		return nil, err
	}

	// Send the update
	updateMsg, err := nm.broadcast.BroadcastDefinition(ctx, identity.Namespace, &core.IdentityUpdate{
		Identity: identity.IdentityBase,
		Updates:  dto.IdentityProfile,
	}, updateSigner, core.SystemTagIdentityUpdate, waitConfirm)
	if err != nil {
		return nil, err
	}
	identity.Messages.Update = updateMsg.Header.ID

	return identity, err
}
