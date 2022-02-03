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

func (dh *definitionHandlers) handleDatatypeBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (DefinitionMessageAction, *DefinitionBatchActions, error) {
	l := log.L(ctx)

	var dt fftypes.Datatype
	valid := dh.getSystemBroadcastPayload(ctx, msg, data, &dt)
	if !valid {
		return ActionReject, nil, nil
	}

	if err := dt.Validate(ctx, true); err != nil {
		l.Warnf("Unable to process datatype broadcast %s - validate failed: %s", msg.Header.ID, err)
		return ActionReject, nil, nil
	}

	if err := dh.data.CheckDatatype(ctx, dt.Namespace, &dt); err != nil {
		l.Warnf("Unable to process datatype broadcast %s - schema check: %s", msg.Header.ID, err)
		return ActionReject, nil, nil
	}

	existing, err := dh.database.GetDatatypeByName(ctx, dt.Namespace, dt.Name, dt.Version)
	if err != nil {
		return ActionRetry, nil, err // We only return database errors
	}
	if existing != nil {
		l.Warnf("Unable to process datatype broadcast %s (%s:%s) - duplicate of %v", msg.Header.ID, dt.Namespace, dt, existing.ID)
		return ActionReject, nil, nil
	}

	if err = dh.database.UpsertDatatype(ctx, &dt, false); err != nil {
		return ActionRetry, nil, err
	}

	return ActionConfirm, &DefinitionBatchActions{
		Finalize: func(ctx context.Context) error {
			event := fftypes.NewEvent(fftypes.EventTypeDatatypeConfirmed, dt.Namespace, dt.ID)
			return dh.database.InsertEvent(ctx, event)
		},
	}, nil
}
