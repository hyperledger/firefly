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
	"database/sql/driver"
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
		l.Errorf("Invalid transmission from %s peer '%s': %s", dx.Name(), peerID, err)
		return "", nil
	}
	if wrapper.Batch == nil {
		l.Errorf("Invalid transmission: nil batch")
		return "", nil
	}
	l.Infof("Private batch received from %s peer '%s' (len=%d)", dx.Name(), peerID, len(data))

	if wrapper.Batch.Payload.TX.Type == fftypes.TransactionTypeUnpinned {
		valid, err := em.definitions.EnsureLocalGroup(em.ctx, wrapper.Group)
		if err != nil {
			return "", err
		}
		if !valid {
			l.Errorf("Invalid transmission: invalid group")
			return "", nil
		}
	}

	manifestString, err := em.privateBatchReceived(peerID, wrapper.Batch)
	return manifestString, err
}

// Check data exchange peer the data came from, has been registered to the org listed in the batch.
// Note the on-chain identity check is performed separately by the aggregator (across broadcast and private consistently).
func (em *eventManager) checkReceivedOffchainIdentity(ctx context.Context, peerID, author string) (node *fftypes.Identity, err error) {
	l := log.L(em.ctx)

	// Resolve the node for the peer ID
	node, err = em.identity.FindIdentityForVerifier(ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: peerID,
	})
	if err != nil {
		return nil, err
	}

	// Find the identity in the mesage
	org, retryable, err := em.identity.CachedIdentityLookup(ctx, author)
	if err != nil && retryable {
		l.Errorf("Failed to retrieve org: %v", err)
		return nil, err // retryable error
	}
	if org == nil || err != nil {
		l.Errorf("Identity %s not found", author)
		return nil, nil
	}

	// One of the orgs in the hierarchy of the author must be the owner of the peer node
	candidate := org
	foundNodeOrg := org.ID.Equals(node.Parent)
	for !foundNodeOrg && candidate.Parent != nil {
		parent := candidate.Parent
		candidate, err = em.identity.CachedIdentityLookupByID(ctx, parent)
		if err != nil {
			l.Errorf("Failed to retrieve node org '%s': %v", parent, err)
			return nil, err // retry for persistence error
		}
		if candidate == nil {
			l.Errorf("Did not find org '%s' in chain for identity '%s' (%s)", parent, org.DID, org.ID)
			return nil, nil
		}
		foundNodeOrg = candidate.ID.Equals(node.Parent)
	}
	if !foundNodeOrg {
		l.Errorf("No org in the chain matches owner '%s' of node '%s' ('%s')", node.Parent, node.ID, node.Name)
		return nil, nil
	}

	return node, nil
}

func (em *eventManager) privateBatchReceived(peerID string, batch *fftypes.Batch) (manifest string, err error) {

	// Retry for persistence errors (not validation errors)
	err = em.retry.Do(em.ctx, "private batch received", func(attempt int) (bool, error) {
		return true, em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			l := log.L(ctx)

			node, err := em.checkReceivedOffchainIdentity(ctx, peerID, batch.Author)
			if err != nil {
				return err
			}
			if node == nil {
				l.Errorf("Batch received from invalid author '%s' for peer ID '%s'", batch.Author, peerID)
				return nil
			}

			persistedBatch, valid, err := em.persistBatch(ctx, batch)
			if err != nil || !valid {
				l.Errorf("Batch received from org=%s node=%s processing failed valid=%t: %s", node.Parent, node.Name, valid, err)
				return err // retry - persistBatch only returns retryable errors
			}

			if batch.Payload.TX.Type == fftypes.TransactionTypeBatchPin {
				// Poke the aggregator to do its stuff
				em.aggregator.rewindBatches <- batch.ID
			} else if batch.Payload.TX.Type == fftypes.TransactionTypeUnpinned {
				// We need to confirm all these messages immediately.
				if err := em.markUnpinnedMessagesConfirmed(ctx, batch); err != nil {
					return err
				}
			}
			manifest = persistedBatch.Manifest.String()
			return nil
		})
	})
	return manifest, err

}

func (em *eventManager) markUnpinnedMessagesConfirmed(ctx context.Context, batch *fftypes.Batch) error {

	// Update all the messages in the batch with the batch ID
	msgIDs := make([]driver.Value, len(batch.Payload.Messages))
	for i, msg := range batch.Payload.Messages {
		msgIDs[i] = msg.Header.ID
	}
	fb := database.MessageQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.In("id", msgIDs),
		fb.Eq("state", fftypes.MessageStatePending), // In the outside chance another state transition happens first (which supersedes this)
	)

	// Immediate confirmation if no transaction
	update := database.MessageQueryFactory.NewUpdate(ctx).
		Set("batch", batch.ID).
		Set("state", fftypes.MessageStateConfirmed).
		Set("confirmed", fftypes.Now())

	if err := em.database.UpdateMessages(ctx, filter, update); err != nil {
		return err
	}

	for _, msg := range batch.Payload.Messages {
		for _, topic := range msg.Header.Topics {
			// One event per topic
			event := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, batch.Namespace, msg.Header.ID, batch.Payload.TX.ID, topic)
			event.Correlator = msg.Header.CID
			if err := em.database.InsertEvent(ctx, event); err != nil {
				return err
			}
		}
	}

	return nil
}

func (em *eventManager) PrivateBLOBReceived(dx dataexchange.Plugin, peerID string, hash fftypes.Bytes32, size int64, payloadRef string) error {
	l := log.L(em.ctx)
	l.Infof("Blob received event from data exchange %s: Peer='%s' Hash='%v' PayloadRef='%s'", dx.Name(), peerID, &hash, payloadRef)

	if peerID == "" || len(peerID) > 256 || payloadRef == "" || len(payloadRef) > 1024 {
		l.Errorf("Invalid blob received event from data exhange: Peer='%s' Hash='%v' PayloadRef='%s'", peerID, &hash, payloadRef)
		return nil // we consume the event still
	}

	return em.blobReceivedCommon(peerID, hash, size, payloadRef)
}

func (em *eventManager) blobReceivedCommon(peerID string, hash fftypes.Bytes32, size int64, payloadRef string) error {
	// We process the event in a retry loop (which will break only if the context is closed), so that
	// we only confirm consumption of the event to the plugin once we've processed it.
	return em.retry.Do(em.ctx, "blob reference insert", func(attempt int) (retry bool, err error) {
		return true, em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
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
			return em.aggregator.rewindForBlobArrival(ctx, &hash)
		})
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

		// The maniest should exactly match that stored into the operation input, if supported
		op := operations[0]
		if status == fftypes.OpStatusSucceeded && dx.Capabilities().Manifest {
			switch op.Type {
			case fftypes.OpTypeDataExchangeSendBatch:
				batchID, _ := fftypes.ParseUUID(em.ctx, op.Input.GetString("batch"))
				expectedManifest := ""
				if batchID != nil {
					batch, err := em.database.GetBatchByID(em.ctx, batchID)
					if err != nil {
						return true, err
					}
					if batch != nil {
						expectedManifest = batch.Manifest.String()
					}
				}
				if update.Manifest != expectedManifest {
					// Log and map to failure for user to see that the receiver did not provide a matching acknowledgement
					mismatchErr := i18n.NewError(em.ctx, i18n.MsgManifestMismatch, status, update.Manifest)
					log.L(em.ctx).Errorf("%s transfer %s: %s", dx.Name(), trackingID, mismatchErr.Error())
					update.Error = mismatchErr.Error()
					status = fftypes.OpStatusFailed
				}
			case fftypes.OpTypeDataExchangeSendBlob:
				expectedHash := op.Input.GetString("hash")
				if update.Hash != expectedHash {
					// Log and map to failure for user to see that the receiver did not provide a matching hash
					mismatchErr := i18n.NewError(em.ctx, i18n.MsgBlobHashMismatch, expectedHash, update.Hash)
					log.L(em.ctx).Errorf("%s transfer %s: %s", dx.Name(), trackingID, mismatchErr.Error())
					update.Error = mismatchErr.Error()
					status = fftypes.OpStatusFailed
				}
			}
		}

		// Resolve the operation
		// Note that we don't need the manifest to be kept here, as it's already in the input
		if err := em.database.ResolveOperation(em.ctx, op.ID, status, update.Error, update.Info); err != nil {
			return true, err // this is always retryable
		}
		return false, nil
	})

}
