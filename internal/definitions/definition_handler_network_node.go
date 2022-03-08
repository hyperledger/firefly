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

package definitions

import (
	"context"

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (dh *definitionHandlers) handleDeprecatedNodeBroadcast(ctx context.Context, state DefinitionBatchState, msg *fftypes.Message, data fftypes.DataArray) (HandlerResult, error) {
	l := log.L(ctx)

	var nodeOld fftypes.DeprecatedNode
	valid := dh.getSystemBroadcastPayload(ctx, msg, data, &nodeOld)
	if !valid {
		return HandlerResult{Action: ActionReject}, nil
	}

	owner, err := dh.identity.FindIdentityForVerifier(ctx, []fftypes.IdentityType{fftypes.IdentityTypeOrg}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  dh.blockchain.VerifierType(),
		Value: nodeOld.Owner,
	})
	if err != nil {
		return HandlerResult{Action: ActionRetry}, err // We only return database errors
	}
	if owner == nil {
		l.Warnf("Unable to process node broadcast %s - parent identity not found: %s", msg.Header.ID, nodeOld.Owner)
		return HandlerResult{Action: ActionReject}, nil
	}

	return dh.handleIdentityClaim(ctx, state, msg, nodeOld.AddMigratedParent(owner.ID), nil)

}
