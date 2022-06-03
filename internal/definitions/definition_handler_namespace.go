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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
)

func (dh *definitionHandlers) handleNamespaceBroadcast(ctx context.Context, state DefinitionBatchState, msg *core.Message, data core.DataArray, tx *fftypes.UUID) (HandlerResult, error) {
	var ns core.Namespace
	if valid := dh.getSystemBroadcastPayload(ctx, msg, data, &ns); !valid {
		return HandlerResult{Action: ActionReject}, i18n.NewError(ctx, coremsgs.MsgDefRejectedBadPayload, "namespace", msg.Header.ID)
	}
	if err := ns.Validate(ctx, true); err != nil {
		return HandlerResult{Action: ActionReject}, i18n.NewError(ctx, coremsgs.MsgDefRejectedValidateFail, "namespace", ns.ID, err)
	}

	existing, err := dh.database.GetNamespace(ctx, ns.Name)
	if err != nil {
		return HandlerResult{Action: ActionRetry}, err
	}
	if existing != nil {
		if existing.Type != core.NamespaceTypeLocal {
			return HandlerResult{Action: ActionReject}, i18n.NewError(ctx, coremsgs.MsgDefRejectedConflict, "namespace", ns.ID, existing.ID)
		}
		// Remove the local definition
		if err = dh.database.DeleteNamespace(ctx, existing.ID); err != nil {
			return HandlerResult{Action: ActionRetry}, err
		}
	}

	if err = dh.database.UpsertNamespace(ctx, &ns, false); err != nil {
		return HandlerResult{Action: ActionRetry}, err
	}

	state.AddFinalize(func(ctx context.Context) error {
		event := core.NewEvent(core.EventTypeNamespaceConfirmed, ns.Name, ns.ID, tx, core.SystemTopicDefinitions)
		return dh.database.InsertEvent(ctx, event)
	})
	return HandlerResult{Action: ActionConfirm}, nil
}
