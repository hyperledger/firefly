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
	"io"

	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/blockchain"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

// BatchPinComplete is called in-line with a particular ledger's stream of events, so while we
// block here this blockchain event remains un-acknowledged, and no further events will arrive from this
// particular ledger.
//
// We must block here long enough to get the payload from the publicstorage, persist the messages in the correct
// sequence, and also persist all the data.
func (em *eventManager) BatchPinComplete(bi blockchain.Plugin, batchPin *blockchain.BatchPin, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error {

	log.L(em.ctx).Infof("-> BatchPinComplete txn=%s author=%s", protocolTxID, signingIdentity)
	defer func() {
		log.L(em.ctx).Infof("<- BatchPinComplete txn=%s author=%s", protocolTxID, signingIdentity)
	}()
	log.L(em.ctx).Tracef("BatchPinComplete info: %+v", additionalInfo)

	if batchPin.BatchPaylodRef != "" {
		return em.handleBroadcastPinComplete(batchPin, signingIdentity, protocolTxID, additionalInfo)
	}
	return em.handlePrivatePinComplete(batchPin, signingIdentity, protocolTxID, additionalInfo)
}

func (em *eventManager) handlePrivatePinComplete(batchPin *blockchain.BatchPin, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	// Here we simple record all the pins as parked, and emit an event for the aggregator
	// to check whether the messages in the batch have been written.
	return em.retry.Do(em.ctx, "persist private batch pins", func(attempt int) (bool, error) {
		// We process the batch into the DB as a single transaction (if transactions are supported), both for
		// efficiency and to minimize the chance of duplicates (although at-least-once delivery is the core model)
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			err := em.persistBatchTransaction(ctx, batchPin, signingIdentity, protocolTxID, additionalInfo)
			if err == nil {
				err = em.persistContexts(ctx, batchPin, true)
			}
			return err
		})
		return err != nil, err // retry indefinitely (until context closes)
	})
}

func (em *eventManager) persistBatchTransaction(ctx context.Context, batchPin *blockchain.BatchPin, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	// Get any existing record for the batch transaction record
	tx, err := em.database.GetTransactionByID(ctx, batchPin.TransactionID)
	if err != nil {
		return err // a peristence failure here is considered retryable (so returned)
	}
	if err := fftypes.ValidateFFNameField(ctx, batchPin.Namespace, "namespace"); err != nil {
		log.L(ctx).Errorf("Invalid batch '%s'. Transaction '%s' invalid namespace '%s': %a", batchPin.BatchID, batchPin.TransactionID, batchPin.Namespace, err)
		return nil // This is not retryable. skip this batch
	}
	if tx == nil {
		// We're the first to write the transaction record on this node
		tx = &fftypes.Transaction{
			ID: batchPin.TransactionID,
			Subject: fftypes.TransactionSubject{
				Namespace: batchPin.Namespace,
				Type:      fftypes.TransactionTypeBatchPin,
				Signer:    signingIdentity,
				Reference: batchPin.BatchID,
			},
			Created: fftypes.Now(),
		}
		tx.Hash = tx.Subject.Hash()
	} else if tx.Subject.Type != fftypes.TransactionTypeBatchPin ||
		tx.Subject.Signer != signingIdentity ||
		tx.Subject.Reference == nil ||
		*tx.Subject.Reference != *batchPin.BatchID ||
		tx.Subject.Namespace != batchPin.Namespace {
		log.L(ctx).Errorf("Invalid batch '%s'. Existing transaction '%s' does not match batch subject", batchPin.BatchID, tx.ID)
		return nil // This is not retryable. skip this batch
	}

	// Set the updates on the transaction
	tx.ProtocolID = protocolTxID
	tx.Info = additionalInfo
	tx.Status = fftypes.OpStatusSucceeded

	// Upsert the transaction, ensuring the hash does not change
	err = em.database.UpsertTransaction(ctx, tx, true, false)
	if err != nil {
		if err == database.HashMismatch {
			log.L(ctx).Errorf("Invalid batch '%s'. Transaction '%s' hash mismatch with existing record", batchPin.BatchID, tx.Hash)
			return nil // This is not retryable. skip this batch
		}
		log.L(ctx).Errorf("Failed to insert transaction for batch '%s': %s", batchPin.BatchID, err)
		return err // a peristence failure here is considered retryable (so returned)
	}

	return nil
}

func (em *eventManager) persistContexts(ctx context.Context, batchPin *blockchain.BatchPin, private bool) error {
	for idx, hash := range batchPin.Contexts {
		if err := em.database.UpsertPin(ctx, &fftypes.Pin{
			Masked:  private,
			Hash:    hash,
			Batch:   batchPin.BatchID,
			Index:   int64(idx),
			Created: fftypes.Now(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (em *eventManager) handleBroadcastPinComplete(batchPin *blockchain.BatchPin, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	var body io.ReadCloser
	if err := em.retry.Do(em.ctx, "retrieve data", func(attempt int) (retry bool, err error) {
		body, err = em.publicstorage.RetrieveData(em.ctx, batchPin.BatchPaylodRef)
		return err != nil, err // retry indefinitely (until context closes)
	}); err != nil {
		return err
	}
	defer body.Close()

	var batch *fftypes.Batch
	err := json.NewDecoder(body).Decode(&batch)
	if err != nil {
		log.L(em.ctx).Errorf("Failed to parse payload referred in batch ID '%s' from transaction '%s'", batchPin.BatchID, protocolTxID)
		return nil // log and swallow unprocessable data
	}
	body.Close()

	// At this point the batch is parsed, so any errors in processing need to be considered as:
	// 1) Retryable - any transient error returned by processBatch is retried indefinitely
	// 2) Swallowable - the data is invalid, and we have to move onto subsequent messages
	// 3) Server shutting down - the context is cancelled (handled by retry)
	return em.retry.Do(em.ctx, "persist batch", func(attempt int) (bool, error) {
		// We process the batch into the DB as a single transaction (if transactions are supported), both for
		// efficiency and to minimize the chance of duplicates (although at-least-once delivery is the core model)
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			err := em.persistBatchTransaction(ctx, batchPin, signingIdentity, protocolTxID, additionalInfo)
			if err == nil {
				err = em.persistBatchFromBroadcast(ctx, batch, batchPin.BatchHash, signingIdentity)
				if err == nil {
					err = em.persistContexts(ctx, batchPin, false)
				}
			}
			return err
		})
		return err != nil, err // retry indefinitely (until context closes)
	})
}
