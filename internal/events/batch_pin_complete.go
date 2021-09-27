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
func (em *eventManager) BatchPinComplete(bi blockchain.Plugin, batchPin *blockchain.BatchPin, author string, protocolTxID string, additionalInfo fftypes.JSONObject) error {

	log.L(em.ctx).Infof("-> BatchPinComplete txn=%s author=%s", protocolTxID, author)
	defer func() {
		log.L(em.ctx).Infof("<- BatchPinComplete txn=%s author=%s", protocolTxID, author)
	}()
	log.L(em.ctx).Tracef("BatchPinComplete info: %+v", additionalInfo)

	if batchPin.BatchPaylodRef != "" {
		return em.handleBroadcastPinComplete(batchPin, author, protocolTxID, additionalInfo)
	}
	return em.handlePrivatePinComplete(batchPin, author, protocolTxID, additionalInfo)
}

func (em *eventManager) handlePrivatePinComplete(batchPin *blockchain.BatchPin, author string, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	// Here we simple record all the pins as parked, and emit an event for the aggregator
	// to check whether the messages in the batch have been written.
	return em.retry.Do(em.ctx, "persist private batch pins", func(attempt int) (bool, error) {
		// We process the batch into the DB as a single transaction (if transactions are supported), both for
		// efficiency and to minimize the chance of duplicates (although at-least-once delivery is the core model)
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			valid, err := em.persistBatchTransaction(ctx, batchPin, author, protocolTxID, additionalInfo)
			if valid && err == nil {
				err = em.persistContexts(ctx, batchPin, true)
			}
			return err
		})
		return err != nil, err // retry indefinitely (until context closes)
	})
}

func subjectMatch(a *fftypes.TransactionSubject, b *fftypes.TransactionSubject) bool {
	return a.Type == b.Type &&
		a.Signer == b.Signer &&
		a.Reference != nil && b.Reference != nil &&
		*a.Reference == *b.Reference &&
		a.Namespace == b.Namespace
}

func (em *eventManager) persistTransaction(ctx context.Context, tx *fftypes.Transaction) (valid bool, err error) {
	if err := fftypes.ValidateFFNameField(ctx, tx.Subject.Namespace, "namespace"); err != nil {
		log.L(ctx).Errorf("Invalid transaction ID='%s' Reference='%s' - invalid namespace '%s': %a", tx.ID, tx.Subject.Reference, tx.Subject.Namespace, err)
		return false, nil // this is not retryable
	}
	existing, err := em.database.GetTransactionByID(ctx, tx.ID)
	if err != nil {
		return false, err // a peristence failure here is considered retryable (so returned)
	}

	switch {
	case existing == nil:
		// We're the first to write the transaction record on this node
		tx.Created = fftypes.Now()
		tx.Hash = tx.Subject.Hash()

	case subjectMatch(&tx.Subject, &existing.Subject):
		// This is an update to an existing transaction, but the subject is the same
		tx.Created = existing.Created
		tx.Hash = existing.Hash

	default:
		log.L(ctx).Errorf("Invalid transaction ID='%s' Reference='%s' - does not match existing subject", tx.ID, tx.Subject.Reference)
		return false, nil // this is not retryable
	}

	// Upsert the transaction, ensuring the hash does not change
	tx.Status = fftypes.OpStatusSucceeded
	err = em.database.UpsertTransaction(ctx, tx, false)
	if err != nil {
		if err == database.HashMismatch {
			log.L(ctx).Errorf("Invalid transaction ID='%s' Reference='%s' - hash mismatch with existing record '%s'", tx.ID, tx.Subject.Reference, tx.Hash)
			return false, nil // this is not retryable
		}
		log.L(ctx).Errorf("Failed to insert transaction ID='%s' Reference='%s': %a", tx.ID, tx.Subject.Reference, err)
		return false, err // a peristence failure here is considered retryable (so returned)
	}

	return true, nil
}

func (em *eventManager) persistBatchTransaction(ctx context.Context, batchPin *blockchain.BatchPin, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) (valid bool, err error) {
	return em.persistTransaction(ctx, &fftypes.Transaction{
		ID: batchPin.TransactionID,
		Subject: fftypes.TransactionSubject{
			Namespace: batchPin.Namespace,
			Type:      fftypes.TransactionTypeBatchPin,
			Signer:    signingIdentity,
			Reference: batchPin.BatchID,
		},
		ProtocolID: protocolTxID,
		Info:       additionalInfo,
	})
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
			valid, err := em.persistBatchTransaction(ctx, batchPin, signingIdentity, protocolTxID, additionalInfo)
			// Note that in the case of a bad batch broadcast, we don't store the pin. Because we know we
			// are never going to be able to process it (we retrieved it successfully, it's just invalid).
			if valid && err == nil {
				valid, err = em.persistBatchFromBroadcast(ctx, batch, batchPin.BatchHash, signingIdentity)
				if valid && err == nil {
					err = em.persistContexts(ctx, batchPin, false)
				}
			}
			return err
		})
		return err != nil, err // retry indefinitely (until context closes)
	})
}
