// Copyright © 2021 Kaleido, Inc.
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

package broadcast

import (
	"context"

	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (bm *broadcastManager) handleNodeBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (valid bool, err error) {
	l := log.L(ctx)

	var node fftypes.Node
	valid = bm.getSystemBroadcastPayload(ctx, msg, data, &node)
	if !valid {
		return false, nil
	}

	if err = node.Validate(ctx, true); err != nil {
		l.Warnf("Unable to process node broadcast %s - validate failed: %s", msg.Header.ID, err)
		return false, nil
	}

	owner, err := bm.database.GetOrganizationByIdentity(ctx, node.Owner)
	if err != nil {
		return false, err // We only return database errors
	}
	if owner == nil {
		l.Warnf("Unable to process node broadcast %s - parent identity not found: %s", msg.Header.ID, node.Owner)
		return false, nil
	}

	id, err := bm.identity.Resolve(ctx, node.Owner)
	if err != nil {
		l.Warnf("Unable to process node broadcast %s - resolve owner identity failed: %s", msg.Header.ID, err)
		return false, nil
	}

	if msg.Header.Author != id.OnChain {
		l.Warnf("Unable to process node broadcast %s - incorrect signature. Expected=%s Received=%s", msg.Header.ID, id.OnChain, msg.Header.Author)
		return false, nil
	}

	existing, err := bm.database.GetNode(ctx, node.Owner, node.Name)
	if err == nil && existing == nil {
		existing, err = bm.database.GetNodeByID(ctx, node.ID)
	}
	if err != nil {
		return false, err // We only return database errors
	}
	if existing != nil {
		if existing.Owner != node.Owner {
			l.Warnf("Unable to process node broadcast %s - mismatch with existing %v", msg.Header.ID, existing.ID)
			return false, nil
		}
		node.ID = nil // we keep the existing ID
	}

	if err = bm.database.UpsertNode(ctx, &node, true); err != nil {
		return false, err
	}

	// Tell the data exchange about this node. Treat these errors like database errors - and return for retry processing
	if err = bm.exchange.AddPeer(ctx, &node); err != nil {
		return false, err
	}

	return true, nil
}
