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
	"context"
	"encoding/json"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (em *eventManager) MessageReceived(dx dataexchange.Plugin, peerID string, data []byte) error {

	l := log.L(em.ctx)

	// De-serializae the transport wrapper
	var wrapper fftypes.TransportWrapper
	err := json.Unmarshal(data, &wrapper)
	if err != nil {
		l.Errorf("Invalid transmission from '%s': %s", peerID, err)
		return nil
	}

	l.Infof("%s received from '%s' (len=%d)", wrapper.Type, peerID, len(data))

	switch wrapper.Type {
	case fftypes.TransportPayloadTypeBatch:
		if wrapper.Batch == nil {
			l.Errorf("Invalid transmission: nil batch")
			return nil
		}
		return em.pinedBatchReceived(peerID, wrapper.Batch)
	case fftypes.TransportPayloadTypeMessage:
		if wrapper.Message == nil {
			l.Errorf("Invalid transmission: nil message")
			return nil
		}
		if wrapper.Group == nil {
			l.Errorf("Invalid transmission: nil group")
			return nil
		}
		return em.unpinnedMessageReceived(peerID, wrapper.Message, wrapper.Group, wrapper.Data)
	default:
		l.Errorf("Invalid transmission: unknonwn type '%s'", wrapper.Type)
		return nil
	}

}

func (em *eventManager) checkReceivedIdentity(ctx context.Context, peerID, author, signingKey string) (node *fftypes.Node, err error) {
	l := log.L(em.ctx)

	// Find the node associated with the peer
	filter := database.NodeQueryFactory.NewFilter(ctx).Eq("dx.peer", peerID)
	nodes, _, err := em.database.GetNodes(ctx, filter)
	if err != nil {
		l.Errorf("Failed to retrieve node: %v", err)
		return nil, err // retry for persistence error
	}
	if len(nodes) < 1 {
		l.Errorf("Node not found for peer %s", peerID)
		return nil, nil
	}
	node = nodes[0]

	// Find the identity in the mesage
	org, err := em.database.GetOrganizationByIdentity(ctx, signingKey)
	if err != nil {
		l.Errorf("Failed to retrieve org: %v", err)
		return nil, err // retry for persistence error
	}
	if org == nil {
		l.Errorf("Org not found for identity %s", author)
		return nil, nil
	}

	// One of the orgs in the hierarchy of the author must be the owner of the peer node
	candidate := org
	foundNodeOrg := signingKey == node.Owner
	for !foundNodeOrg && candidate.Parent != "" {
		parent := candidate.Parent
		candidate, err = em.database.GetOrganizationByIdentity(ctx, parent)
		if err != nil {
			l.Errorf("Failed to retrieve node org '%s': %v", parent, err)
			return nil, err // retry for persistence error
		}
		if candidate == nil {
			l.Errorf("Did not find org '%s' in chain for identity '%s'", parent, org.Identity)
			return nil, nil
		}
		foundNodeOrg = candidate.Identity == node.Owner
	}
	if !foundNodeOrg {
		l.Errorf("No org in the chain matches owner '%s' of node '%s' ('%s')", node.Owner, node.ID, node.Name)
		return nil, nil
	}

	return node, nil
}

func (em *eventManager) pinedBatchReceived(peerID string, batch *fftypes.Batch) error {

	// Retry for persistence errors (not validation errors)
	return em.retry.Do(em.ctx, "private batch received", func(attempt int) (bool, error) {
		return true, em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			l := log.L(ctx)

			node, err := em.checkReceivedIdentity(ctx, peerID, batch.Author, batch.Key)
			if err != nil {
				return err
			}
			if node == nil {
				l.Errorf("Batch received from invalid author '%s' for peer ID '%s'", batch.Author, peerID)
				return nil
			}

			valid, err := em.persistBatch(ctx, batch)
			if err != nil {
				l.Errorf("Batch received from %s/%s invalid: %s", node.Owner, node.Name, err)
				return err // retry - persistBatch only returns retryable errors
			}

			if valid {
				em.aggregator.offchainBatches <- batch.ID
			}
			return nil
		})
	})

}

func (em *eventManager) BLOBReceived(dx dataexchange.Plugin, peerID string, hash fftypes.Bytes32, payloadRef string) error {
	l := log.L(em.ctx)
	l.Debugf("Blob received event from data exhange: Peer='%s' Hash='%v' PayloadRef='%s'", peerID, &hash, payloadRef)

	if peerID == "" || len(peerID) > 256 || payloadRef == "" || len(payloadRef) > 1024 {
		l.Errorf("Invalid blob received event from data exhange: Peer='%s' Hash='%v' PayloadRef='%s'", peerID, &hash, payloadRef)
		return nil // we consume the event still
	}

	// We process the event in a retry loop (which will break only if the context is closed), so that
	// we only confirm consumption of the event to the plugin once we've processed it.
	return em.retry.Do(em.ctx, "blob reference insert", func(attempt int) (retry bool, err error) {

		batchIDs := make(map[fftypes.UUID]bool)

		err = em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			// Insert the blob into the detabase
			err := em.database.InsertBlob(ctx, &fftypes.Blob{
				Peer:       peerID,
				PayloadRef: payloadRef,
				Hash:       &hash,
				Created:    fftypes.Now(),
			})
			if err != nil {
				return err
			}

			// Now we need to work out what pins potentially are unblocked by the arrival of this data

			// Find any data associated with this blob
			var data []*fftypes.DataRef
			filter := database.DataQueryFactory.NewFilter(ctx).Eq("blob.hash", &hash)
			data, _, err = em.database.GetDataRefs(ctx, filter)
			if err != nil {
				return err
			}

			// Find the messages assocated with that data
			var messages []*fftypes.Message
			for _, data := range data {
				fb := database.MessageQueryFactory.NewFilter(ctx)
				filter := fb.And(fb.Eq("confirmed", nil))
				messages, _, err = em.database.GetMessagesForData(ctx, data.ID, filter)
				if err != nil {
					return err
				}
			}

			// Find the unique batch IDs for all the messages
			for _, msg := range messages {
				if msg.BatchID != nil {
					batchIDs[*msg.BatchID] = true
				}
			}
			return nil
		})
		if err != nil {
			return true, err
		}

		// Initiate rewinds for all the batchIDs that are potentially completed by the arrival of this data
		for bid := range batchIDs {
			var batchID = bid // cannot use the address of the loop var
			l.Infof("Batch '%s' contains reference to received blob. Peer='%s' Hash='%v' PayloadRef='%s'", &bid, peerID, &hash, payloadRef)
			em.aggregator.offchainBatches <- &batchID
		}

		return false, nil
	})
}

func (em *eventManager) TransferResult(dx dataexchange.Plugin, trackingID string, status fftypes.OpStatus, info string, opOutput fftypes.JSONObject) error {
	log.L(em.ctx).Infof("Transfer result %s=%s info='%s'", trackingID, status, info)

	// We process the event in a retry loop (which will break only if the context is closed), so that
	// we only confirm consumption of the event to the plugin once we've processed it.
	return em.retry.Do(em.ctx, "blob reference insert", func(attempt int) (retry bool, err error) {

		// Find a matching operation, for this plugin, with the specified ID.
		// We retry a few times, as there's an outside possibility of the event arriving before we're finished persisting the operation itself
		var operations []*fftypes.Operation
		fb := database.OperationQueryFactory.NewFilter(em.ctx)
		filter := fb.And(
			fb.Eq("backendid", trackingID),
			fb.Eq("plugin", dx.Name()),
		)
		operations, _, err = em.database.GetOperations(em.ctx, filter)
		if err != nil {
			return true, err
		}
		if len(operations) == 0 {
			// we have a limit on how long we wait to correlate an operation if we don't have a DB erro,
			// as it should only be a short window where the DB transaction to insert the operation is still
			// outstanding
			if attempt >= em.opCorrelationRetries {
				log.L(em.ctx).Warnf("Unable to correlate %s event %s", dx.Name(), trackingID)
				return false, nil // just skip this
			}
			return true, i18n.NewError(em.ctx, i18n.Msg404NotFound)
		}

		update := database.OperationQueryFactory.NewUpdate(em.ctx).
			Set("status", status).
			Set("error", info).
			Set("output", opOutput)
		for _, op := range operations {
			if err := em.database.UpdateOperation(em.ctx, op.ID, update); err != nil {
				return true, err // this is always retryable
			}
		}
		return false, nil
	})

}

func (em *eventManager) unpinnedMessageReceived(peerID string, message *fftypes.Message, group *fftypes.Group, data []*fftypes.Data) error {
	if message.Header.TxType != fftypes.TransactionTypeNone {
		log.L(em.ctx).Errorf("Unpinned message '%s' transaction type must be 'none'. TxType=%s", message.Header.ID, message.Header.TxType)
		return nil
	}

	// Because we received this off chain, it's entirely possible the group init has not made it
	// to us yet. So we need to go through the same processing as if we had initiated the group.
	// This might result in both sides broadcasting a group-init message, but that's fine.

	return em.retry.Do(em.ctx, "unpinned message received", func(attempt int) (bool, error) {
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {

			if valid, err := em.syshandlers.EnsureLocalGroup(ctx, group); err != nil || !valid {
				return err
			}

			node, err := em.checkReceivedIdentity(ctx, peerID, message.Header.Author, message.Header.Key)
			if err != nil {
				return err
			}
			if node == nil {
				log.L(ctx).Errorf("Message received from invalid author '%s' for peer ID '%s'", message.Header.Author, peerID)
				return nil
			}

			// Persist the data
			for i, d := range data {
				if ok, err := em.persistReceivedData(ctx, i, d, "message", message.Header.ID); err != nil || !ok {
					return err
				}
			}

			// Persist the message - immediately considered confirmed as this is an unpinned receive
			message.Confirmed = fftypes.Now()
			if ok, err := em.persistReceivedMessage(ctx, 0, message, "message", message.Header.ID); err != nil || !ok {
				return err
			}

			// Assuming all was good, we
			event := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, message.Header.Namespace, message.Header.ID)
			return em.database.InsertEvent(ctx, event)
		})
		return err != nil, err
	})

}
