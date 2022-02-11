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

func (dh *definitionHandlers) handleNodeBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (DefinitionMessageAction, *DefinitionBatchActions, error) {
	l := log.L(ctx)

	var node fftypes.Node
	valid := dh.getSystemBroadcastPayload(ctx, msg, data, &node)
	if !valid {
		return ActionReject, nil, nil
	}

	if err := node.Validate(ctx, true); err != nil {
		l.Warnf("Unable to process node broadcast %s - validate failed: %s", msg.Header.ID, err)
		return ActionReject, nil, nil
	}

	owner, err := dh.database.GetOrganizationByIdentity(ctx, node.Owner)
	if err != nil {
		return ActionRetry, nil, err // We only return database errors
	}
	if owner == nil {
		l.Warnf("Unable to process node broadcast %s - parent identity not found: %s", msg.Header.ID, node.Owner)
		return ActionReject, nil, nil
	}

	if msg.Header.Key != node.Owner {
		l.Warnf("Unable to process node broadcast %s - incorrect signature. Expected=%s Received=%s", msg.Header.ID, node.Owner, msg.Header.Author)
		return ActionReject, nil, nil
	}

	existing, err := dh.database.GetNode(ctx, node.Owner, node.Name)
	if err == nil && existing == nil {
		existing, err = dh.database.GetNodeByID(ctx, node.ID)
	}
	if err != nil {
		return ActionRetry, nil, err // We only return database errors
	}
	if existing != nil {
		if existing.Owner != node.Owner {
			l.Warnf("Unable to process node broadcast %s - mismatch with existing %v", msg.Header.ID, existing.ID)
			return ActionReject, nil, nil
		}
		node.ID = nil // we keep the existing ID
	}

	if err = dh.database.UpsertNode(ctx, &node, true); err != nil {
		return ActionRetry, nil, err
	}

	return ActionConfirm, &DefinitionBatchActions{
		PreFinalize: func(ctx context.Context) error {
			// Tell the data exchange about this node. Treat these errors like database errors - and return for retry processing
			return dh.exchange.AddPeer(ctx, node.DX)
		},
	}, nil
}
