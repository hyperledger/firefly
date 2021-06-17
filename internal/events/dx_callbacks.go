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

package events

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/dataexchange"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (em *eventManager) MessageReceived(dx dataexchange.Plugin, peerID string, data []byte) {

	// Retry for persistence errors (not validation errors)
	_ = em.retry.Do(em.ctx, "private batch received", func(attempt int) (bool, error) {

		l := log.L(em.ctx)
		l.Infof("Message received from '%s' (len=%d)", peerID, len(data))

		// Try to de-serialize it as a batch
		var batch fftypes.Batch
		err := json.Unmarshal(data, &batch)
		if err != nil {
			l.Errorf("Invalid batch: %s", err)
			return false, nil
		}

		// Find the node associated with the peer
		filter := database.NodeQueryFactory.NewFilter(em.ctx).Eq("dx.peer", peerID)
		nodes, err := em.database.GetNodes(em.ctx, filter)
		if err != nil {
			l.Errorf("Failed to retrieve node: %v", err)
			return true, err // retry for persistence error
		}
		if len(nodes) < 1 {
			l.Errorf("Node not found for peer %s", peerID)
			return false, nil
		}
		node := nodes[0]

		// Find the identity in the mesage
		batchOrg, err := em.database.GetOrganizationByIdentity(em.ctx, batch.Author)
		if err != nil {
			l.Errorf("Failed to retrieve batch org: %v", err)
			return true, err // retry for persistence error
		}
		if batchOrg == nil {
			l.Errorf("Org not found for identity %s", batch.Author)
			return false, nil
		}

		// One of the orgs in the hierarchy of the batch author must be the owner of the peer node
		candidate := batchOrg
		foundNodeOrg := batch.Author == node.Owner
		for !foundNodeOrg && candidate.Parent != "" {
			parent := candidate.Parent
			candidate, err = em.database.GetOrganizationByIdentity(em.ctx, parent)
			if err != nil {
				l.Errorf("Failed to retrieve node org '%s': %v", parent, err)
				return true, err // retry for persistence error
			}
			if candidate == nil {
				l.Errorf("Did not find org '%s' in chain for identity '%s'", parent, batchOrg.Identity)
				return false, nil
			}
			foundNodeOrg = candidate.Identity == node.Owner
		}
		if !foundNodeOrg {
			l.Errorf("No org in the chain matches owner '%s' of node '%s' ('%s')", node.Owner, node.ID, node.Name)
			return false, nil
		}

		valid, err := em.persistBatch(em.ctx, &batch)
		if err != nil {
			l.Errorf("Batch received from %s/%s invalid: %s", node.Owner, node.Name, err)
			return true, err // retry - persistBatch only returns retryable errors
		}

		if valid {
			em.aggregator.offchainBatches <- batch.ID
		}
		return false, nil
	})

}

func (em *eventManager) BLOBReceived(dx dataexchange.Plugin, peerID string, hash *fftypes.Bytes32, payloadRef string) {
}

func (em *eventManager) TransferResult(dx dataexchange.Plugin, trackingID string, status fftypes.OpStatus, info string, additionalInfo fftypes.JSONObject) {
	log.L(em.ctx).Infof("Transfer result %s=%s info='%s'", trackingID, status, info)

	// Find a matching operation, for this plugin, with the specified ID.
	// We retry a few times, as there's an outside possibility of the event arriving before we're finished persisting the operation itself
	var operations []*fftypes.Operation
	fb := database.OperationQueryFactory.NewFilter(em.ctx)
	filter := fb.And(
		fb.Eq("backendid", trackingID),
		fb.Eq("plugin", dx.Name()),
	)
	err := em.retry.Do(em.ctx, fmt.Sprintf("correlate transfer %s", trackingID), func(attempt int) (retry bool, err error) {
		operations, err = em.database.GetOperations(em.ctx, filter)
		if err == nil && len(operations) == 0 {
			err = i18n.NewError(em.ctx, i18n.Msg404NotFound)
		}
		return (err != nil && attempt <= em.opCorrelationRetries), err
	})
	if err != nil {
		log.L(em.ctx).Warnf("Failed to correlate transfer ID '%s' with a submitted operation", trackingID)
		return
	}

	update := database.OperationQueryFactory.NewUpdate(em.ctx).
		Set("status", status).
		Set("error", info).
		Set("info", additionalInfo)
	for _, op := range operations {
		if err := em.database.UpdateOperation(em.ctx, op.ID, update); err != nil {
			return
		}
	}

}
