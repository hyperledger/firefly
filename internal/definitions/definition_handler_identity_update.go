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
	"github.com/hyperledger/firefly/pkg/database"
)

func (dh *definitionHandlers) handleIdentityUpdateBroadcast(ctx context.Context, state DefinitionBatchState, msg *core.Message, data core.DataArray) (HandlerResult, error) {
	var update core.IdentityUpdate
	valid := dh.getSystemBroadcastPayload(ctx, msg, data, &update)
	if !valid {
		return HandlerResult{Action: ActionReject}, fmt.Errorf("invalid identity update message %s - invalid payload", msg.Header.ID)
	}

	// See if we find the message to which it refers
	err := update.Identity.Validate(ctx)
	if err != nil {
		return HandlerResult{Action: ActionReject}, fmt.Errorf("invalid identity update message %s: %v", msg.Header.ID, err)
	}

	// Get the existing identity (must be a confirmed identity to at the point an update is issued)
	identity, err := dh.identity.CachedIdentityLookupByID(ctx, update.Identity.ID)
	if err != nil {
		return HandlerResult{Action: ActionRetry}, err
	}
	if identity == nil {
		return HandlerResult{Action: ActionReject}, fmt.Errorf("invalid identity update message %s - not found: %s", msg.Header.ID, update.Identity.ID)
	}

	// Check the author matches
	if identity.DID != msg.Header.Author {
		return HandlerResult{Action: ActionReject}, fmt.Errorf("invalid identity update message %s - wrong author: %s", msg.Header.ID, msg.Header.Author)
	}

	// Update the profile
	identity.IdentityProfile = update.Updates
	identity.Messages.Update = msg.Header.ID
	err = dh.database.UpsertIdentity(ctx, identity, database.UpsertOptimizationExisting)
	if err != nil {
		return HandlerResult{Action: ActionRetry}, err
	}

	state.AddFinalize(func(ctx context.Context) error {
		event := core.NewEvent(core.EventTypeIdentityUpdated, identity.Namespace, identity.ID, nil, core.SystemTopicDefinitions)
		return dh.database.InsertEvent(ctx, event)
	})
	return HandlerResult{Action: ActionConfirm}, err

}
