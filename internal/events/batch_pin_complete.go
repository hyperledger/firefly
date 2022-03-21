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

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

// BatchPinComplete is called in-line with a particular ledger's stream of events, so while we
// block here this blockchain event remains un-acknowledged, and no further events will arrive from this
// particular ledger.
//
// We must block here long enough to get the payload from the sharedstorage, persist the messages in the correct
// sequence, and also persist all the data.
func (em *eventManager) BatchPinComplete(bi blockchain.Plugin, batchPin *blockchain.BatchPin, signingKey *fftypes.VerifierRef) error {
	if batchPin.TransactionID == nil {
		log.L(em.ctx).Errorf("Invalid BatchPin transaction - ID is nil")
		return nil // move on
	}
	if err := fftypes.ValidateFFNameField(em.ctx, batchPin.Namespace, "namespace"); err != nil {
		log.L(em.ctx).Errorf("Invalid transaction ID='%s' - invalid namespace '%s': %a", batchPin.TransactionID, batchPin.Namespace, err)
		return nil // move on
	}

	log.L(em.ctx).Infof("-> BatchPinComplete batch=%s txn=%s signingIdentity=%s", batchPin.BatchID, batchPin.Event.ProtocolID, signingKey.Value)
	defer func() {
		log.L(em.ctx).Infof("<- BatchPinComplete batch=%s txn=%s signingIdentity=%s", batchPin.BatchID, batchPin.Event.ProtocolID, signingKey.Value)
	}()
	log.L(em.ctx).Tracef("BatchPinComplete batch=%s info: %+v", batchPin.BatchID, batchPin.Event.Info)

	// Here we simple record all the pins as parked, and emit an event for the aggregator
	// to check whether the messages in the batch have been written.
	return em.retry.Do(em.ctx, "persist batch pins", func(attempt int) (bool, error) {
		// We process the batch into the DB as a single transaction (if transactions are supported), both for
		// efficiency and to minimize the chance of duplicates (although at-least-once delivery is the core model)
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			if err := em.persistBatchTransaction(ctx, batchPin); err != nil {
				return err
			}
			chainEvent := buildBlockchainEvent(batchPin.Namespace, nil, &batchPin.Event, &fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
				ID:   batchPin.TransactionID,
			})
			if err := em.persistBlockchainEvent(ctx, chainEvent); err != nil {
				return err
			}
			em.emitBlockchainEventMetric(&batchPin.Event)
			private := batchPin.BatchPayloadRef == ""
			if err := em.persistContexts(ctx, batchPin, signingKey, private); err != nil {
				return err
			}
			// Kick off a download for broadcast batches
			if !private {
				if err := em.sharedDownload.InitiateDownloadBatch(ctx, batchPin.Namespace, batchPin.TransactionID, batchPin.BatchPayloadRef); err != nil {
					return err
				}
			}
			return nil
		})
		return err != nil, err // retry indefinitely (until context closes)
	})
}

func (em *eventManager) persistBatchTransaction(ctx context.Context, batchPin *blockchain.BatchPin) error {
	_, err := em.txHelper.PersistTransaction(ctx, batchPin.Namespace, batchPin.TransactionID, fftypes.TransactionTypeBatchPin, batchPin.Event.BlockchainTXID)
	return err
}

func (em *eventManager) persistContexts(ctx context.Context, batchPin *blockchain.BatchPin, signingKey *fftypes.VerifierRef, private bool) error {
	pins := make([]*fftypes.Pin, len(batchPin.Contexts))
	for idx, hash := range batchPin.Contexts {
		pins[idx] = &fftypes.Pin{
			Masked:    private,
			Hash:      hash,
			Batch:     batchPin.BatchID,
			BatchHash: batchPin.BatchHash,
			Index:     int64(idx),
			Signer:    signingKey.Value, // We don't store the type as we can infer that from the blockchain
			Created:   fftypes.Now(),
		}
	}

	// First attempt a single batch insert
	err := em.database.InsertPins(ctx, pins)
	if err == nil {
		return nil
	}
	log.L(ctx).Warnf("Batch insert of pins failed - assuming replay and performing upserts: %s", err)

	// Fall back to an upsert
	for _, pin := range pins {
		if err := em.database.UpsertPin(ctx, pin); err != nil {
			return err
		}
	}
	return nil
}
