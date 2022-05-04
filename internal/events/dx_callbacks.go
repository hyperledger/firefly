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

	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/log"
)

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
	org, retryable, err := em.identity.CachedIdentityLookupMustExist(ctx, author)
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

func (em *eventManager) privateBatchReceived(peerID string, batch *fftypes.Batch, wrapperGroup *fftypes.Group) (manifest string, err error) {

	// Retry for persistence errors (not validation errors)
	err = em.retry.Do(em.ctx, "private batch received", func(attempt int) (bool, error) {
		return true, em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			l := log.L(ctx)

			if wrapperGroup != nil && batch.Payload.TX.Type == fftypes.TransactionTypeUnpinned {
				valid, err := em.messaging.EnsureLocalGroup(ctx, wrapperGroup)
				if err != nil {
					return err
				}
				if !valid {
					l.Errorf("Invalid transmission: invalid group: %+v", wrapperGroup)
					return nil
				}
			}

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

			if batch.Payload.TX.Type == fftypes.TransactionTypeUnpinned {
				// We need to confirm all these messages immediately.
				if err := em.markUnpinnedMessagesConfirmed(ctx, batch); err != nil {
					return err
				}
			}
			manifest = persistedBatch.Manifest.String()
			return nil
		})
	})
	if err != nil {
		return "", err
	}
	// Poke the aggregator to do its stuff - after we have committed the transaction so the pins are visible
	if batch.Payload.TX.Type == fftypes.TransactionTypeBatchPin {
		em.aggregator.queueBatchRewind(batch.ID)
	}
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

func (em *eventManager) DXEvent(dx dataexchange.Plugin, event dataexchange.DXEvent) {
	switch event.Type() {
	case dataexchange.DXEventTypePrivateBlobReceived:
		em.privateBlobReceived(dx, event)
	case dataexchange.DXEventTypeMessageReceived:
		// Batches are significant items of work in their own right, so get dispatched to their own routines
		go em.messageReceived(dx, event)
	default:
		log.L(em.ctx).Errorf("Invalid DX event type from %s: %d", dx.Name(), event.Type())
		event.Ack() // still ack
	}
}

func (em *eventManager) messageReceived(dx dataexchange.Plugin, event dataexchange.DXEvent) {
	l := log.L(em.ctx)

	mr := event.MessageReceived()

	// De-serializae the transport wrapper
	var wrapper *fftypes.TransportWrapper
	err := json.Unmarshal(mr.Data, &wrapper)
	if err != nil {
		l.Errorf("Invalid transmission from %s peer '%s': %s", dx.Name(), mr.PeerID, err)
		event.AckWithManifest("")
		return
	}
	if wrapper.Batch == nil {
		l.Errorf("Invalid transmission: nil batch")
		event.AckWithManifest("")
		return
	}
	l.Infof("Private batch received from %s peer '%s' (len=%d)", dx.Name(), mr.PeerID, len(mr.Data))

	manifestString, err := em.privateBatchReceived(mr.PeerID, wrapper.Batch, wrapper.Group)
	if err != nil {
		l.Warnf("Exited while persisting batch: %s", err)
		// We do NOT ack here as we broke out of the retry
		return
	}
	event.AckWithManifest(manifestString)
}

func (em *eventManager) privateBlobReceived(dx dataexchange.Plugin, event dataexchange.DXEvent) {
	br := event.PrivateBlobReceived()
	log.L(em.ctx).Infof("Blob received event from data exchange %s: Peer='%s' Hash='%v' PayloadRef='%s'", dx.Name(), br.PeerID, &br.Hash, br.PayloadRef)

	if br.PeerID == "" || len(br.PeerID) > 256 || br.PayloadRef == "" || len(br.PayloadRef) > 1024 {
		log.L(em.ctx).Errorf("Invalid blob received event from data exhange: Peer='%s' Hash='%v' PayloadRef='%s'", br.PeerID, &br.Hash, br.PayloadRef)
		event.Ack() // Still confirm the event
		return
	}

	// Dispatch to the blob receiver for efficient batch DB operations
	em.blobReceiver.blobReceived(em.ctx, &blobNotification{
		blob: &fftypes.Blob{
			Peer:       br.PeerID,
			PayloadRef: br.PayloadRef,
			Hash:       &br.Hash,
			Size:       br.Size,
			Created:    fftypes.Now(),
		},
		onComplete: func() {
			event.Ack()
		},
	})
}
