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

func (dh *definitionHandlers) handleNamespaceBroadcast(ctx context.Context, state DefinitionBatchState, msg *fftypes.Message, data fftypes.DataArray, tx *fftypes.UUID) (HandlerResult, error) {
	l := log.L(ctx)

	var ns fftypes.Namespace
	valid := dh.getSystemBroadcastPayload(ctx, msg, data, &ns)
	if !valid {
		return HandlerResult{Action: ActionReject}, nil
	}
	if err := ns.Validate(ctx, true); err != nil {
		l.Warnf("Unable to process namespace broadcast %s - validate failed: %s", msg.Header.ID, err)
		return HandlerResult{Action: ActionReject}, nil
	}

	existing, err := dh.database.GetNamespace(ctx, ns.Name)
	if err != nil {
		return HandlerResult{Action: ActionRetry}, err // We only return database errors
	}
	if existing != nil {
		if existing.Type != fftypes.NamespaceTypeLocal {
			l.Warnf("Unable to process namespace broadcast %s (name=%s) - duplicate of %v", msg.Header.ID, existing.Name, existing.ID)
			return HandlerResult{Action: ActionReject}, nil
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
		event := fftypes.NewEvent(fftypes.EventTypeNamespaceConfirmed, ns.Name, ns.ID, tx)
		return dh.database.InsertEvent(ctx, event)
	})
	return HandlerResult{Action: ActionConfirm}, nil
}
