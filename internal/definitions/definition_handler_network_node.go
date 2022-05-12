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
	"fmt"

	"github.com/hyperledger/firefly/pkg/core"
)

func (dh *definitionHandlers) handleDeprecatedNodeBroadcast(ctx context.Context, state DefinitionBatchState, msg *core.Message, data core.DataArray) (HandlerResult, error) {
	var nodeOld core.DeprecatedNode
	valid := dh.getSystemBroadcastPayload(ctx, msg, data, &nodeOld)
	if !valid {
		return HandlerResult{Action: ActionReject}, fmt.Errorf("unable to process node broadcast %s - invalid payload", msg.Header.ID)
	}

	owner, err := dh.identity.FindIdentityForVerifier(ctx, []core.IdentityType{core.IdentityTypeOrg}, core.SystemNamespace, &core.VerifierRef{
		Type:  dh.blockchain.VerifierType(),
		Value: nodeOld.Owner,
	})
	if err != nil {
		return HandlerResult{Action: ActionRetry}, err // We only return database errors
	}
	if owner == nil {
		return HandlerResult{Action: ActionReject}, fmt.Errorf("unable to process node broadcast %s - parent identity not found: %s", msg.Header.ID, nodeOld.Owner)
	}

	return dh.handleIdentityClaim(ctx, state, msg, nodeOld.AddMigratedParent(owner.ID), nil)

}
