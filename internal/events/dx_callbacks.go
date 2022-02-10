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

func (em *eventManager) MessageReceived(dx dataexchange.Plugin, peerID string, data []byte) (manifest string, err error) {

	l := log.L(em.ctx)

	// De-serializae the transport wrapper
	var wrapper *fftypes.TransportWrapper
	err = json.Unmarshal(data, &wrapper)
	if err != nil {
		l.Errorf("Invalid transmission from '%s': %s", peerID, err)
		return "", nil
	}
	if wrapper.Batch == nil {
		l.Errorf("Invalid transmission: nil batch")
		return "", nil
	}
	l.Infof("Private batch received from '%s' (len=%d)", peerID, len(data))

	if wrapper.Group != nil {
		valid, err := em.definitions.EnsureLocalGroup(em.ctx, wrapper.Group)
		if err != nil {
			return "", err
		}
		if !valid {
			l.Errorf("Invalid transmission: invalid group")
			return "", nil
		}
	}

	mf, err := em.privateBatchReceived(peerID, wrapper.Batch)
	manifestBytes := []byte{}
	if err == nil && mf != nil {
		manifestBytes, err = json.Marshal(&mf)
	}
	return string(manifestBytes), err
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

func (em *eventManager) privateBatchReceived(peerID string, batch *fftypes.Batch) (manifest *fftypes.Manifest, err error) {

	// Retry for persistence errors (not validation errors)
	err = em.retry.Do(em.ctx, "private batch received", func(attempt int) (bool, error) {
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
			if err != nil || !valid {
				l.Errorf("Batch received from %s/%s processing failed valid=%t: %s", node.Owner, node.Name, valid, err)
				return err // retry - persistBatch only returns retryable errors
			}

			if batch.Payload.TX.Type == fftypes.TransactionTypeBatchPin {
				// Poke the aggregator to do its stuff
				em.aggregator.offchainBatches <- batch.ID
			} else {

			}
			manifest = batch.Manifest()
			return nil
		})
	})
	return manifest, err

}

func (em *eventManager) BLOBReceived(dx dataexchange.Plugin, peerID string, hash fftypes.Bytes32, size int64, payloadRef string) error {
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
				Size:       size,
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

func (em *eventManager) TransferResult(dx dataexchange.Plugin, trackingID string, status fftypes.OpStatus, update fftypes.TransportStatusUpdate) error {
	log.L(em.ctx).Infof("Transfer result %s=%s error='%s' manifest='%s' info='%s'", trackingID, status, update.Error, update.Manifest, update.Info)

	// We process the event in a retry loop (which will break only if the context is closed), so that
	// we only confirm consumption of the event to the plugin once we've processed it.
	return em.retry.Do(em.ctx, "operation update", func(attempt int) (retry bool, err error) {

		// Find a matching operation, for this plugin, with the specified ID.
		// We retry a few times, as there's an outside possibility of the event arriving before we're finished persisting the operation itself
		var operations []*fftypes.Operation
		fb := database.OperationQueryFactory.NewFilter(em.ctx)
		filter := fb.And(
			fb.Eq("id", trackingID),
			fb.Eq("plugin", dx.Name()),
		)
		operations, _, err = em.database.GetOperations(em.ctx, filter)
		if err != nil {
			return true, err
		}
		if len(operations) != 1 {
			// we have a limit on how long we wait to correlate an operation if we don't have a DB erro,
			// as it should only be a short window where the DB transaction to insert the operation is still
			// outstanding
			if attempt >= em.opCorrelationRetries {
				log.L(em.ctx).Warnf("Unable to correlate %s event %s", dx.Name(), trackingID)
				return false, nil // just skip this
			}
			return true, i18n.NewError(em.ctx, i18n.Msg404NotFound)
		}

		// The manifest should exactly match that stored into the operation input, if supported
		op := operations[0]
		if status == fftypes.OpStatusSucceeded && dx.Capabilities().Manifest {
			expectedManifest := op.Input.GetString("manifest")
			if update.Manifest != expectedManifest {
				// Log and map to failure for user to see that the receiver did not provide a matching acknowledgement
				mismatchErr := i18n.NewError(em.ctx, i18n.MsgManifestMismatch, status, update.Manifest)
				log.L(em.ctx).Errorf("%s transfer %s: %s", dx.Name(), trackingID, mismatchErr.Error())
				update.Error = mismatchErr.Error()
				status = fftypes.OpStatusFailed
			}
		}

		update := database.OperationQueryFactory.NewUpdate(em.ctx).
			Set("status", status).
			Set("error", update.Error).
			Set("output", update.Info) // Note that we don't need the manifest to be kept here, as it's already in the input
		if err := em.database.UpdateOperation(em.ctx, op.ID, update); err != nil {
			return true, err // this is always retryable
		}
		return false, nil
	})

}
