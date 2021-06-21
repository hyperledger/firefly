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

	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (em *eventManager) persistBatchFromBroadcast(ctx context.Context /* db TX context*/, batch *fftypes.Batch, onchainHash *fftypes.Bytes32, author string) error {
	l := log.L(ctx)

	// Verify the author matches
	id, err := em.identity.Resolve(ctx, batch.Author)
	if err != nil {
		l.Errorf("Invalid batch '%s'. Author '%s' cound not be resolved: %s", batch.ID, batch.Author, err)
		return nil // This is not retryable. skip this batch
	}
	if author != id.OnChain {
		l.Errorf("Invalid batch '%s'. Author '%s' does not match transaction submitter '%s'", batch.ID, id.OnChain, author)
		return nil // This is not retryable. skip this batch
	}

	if !onchainHash.Equals(batch.Hash) {
		l.Errorf("Invalid batch '%s'. Hash in batch '%s' does not match transaction hash '%s'", batch.ID, batch.Hash, onchainHash)
		return nil // This is not retryable. skip this batch
	}

	_, err = em.persistBatch(ctx, batch)
	return err
}

// persistBatch performs very simple validation on each message/data element (hashes) and either persists
// or discards them. Errors are returned only in the case of database failures, which should be retried.
func (em *eventManager) persistBatch(ctx context.Context /* db TX context*/, batch *fftypes.Batch) (valid bool, err error) {
	l := log.L(ctx)
	now := fftypes.Now()

	if batch.ID == nil || batch.Payload.TX.ID == nil {
		l.Errorf("Invalid batch '%s'. Missing ID (%v) or transaction ID (%v)", batch.ID, batch.ID, batch.Payload.TX.ID)
		return false, nil // This is not retryable. skip this batch
	}

	// Verify the hash calculation
	hash := batch.Payload.Hash()
	if batch.Hash == nil || *batch.Hash != *hash {
		l.Errorf("Invalid batch '%s'. Hash does not match payload. Found=%s Expected=%s", batch.ID, hash, batch.Hash)
		return false, nil // This is not retryable. skip this batch
	}

	// Set confirmed on the batch (the messages should not be confirmed at this point - that's the aggregator's job)
	batch.Confirmed = now

	// Upsert the batch itself, ensuring the hash does not change
	err = em.database.UpsertBatch(ctx, batch, true, false)
	if err != nil {
		if err == database.HashMismatch {
			l.Errorf("Invalid batch '%s'. Batch hash mismatch with existing record", batch.ID)
			return false, nil // This is not retryable. skip this batch
		}
		l.Errorf("Failed to insert batch '%s': %s", batch.ID, err)
		return false, err // a peristence failure here is considered retryable (so returned)
	}

	// Insert the data entries
	for i, data := range batch.Payload.Data {
		if err = em.persistBatchData(ctx, batch, i, data); err != nil {
			return false, err
		}
	}

	// Insert the message entries
	for i, msg := range batch.Payload.Messages {
		if err = em.persistBatchMessage(ctx, batch, i, msg); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (em *eventManager) persistBatchData(ctx context.Context /* db TX context*/, batch *fftypes.Batch, i int, data *fftypes.Data) error {
	l := log.L(ctx)
	l.Tracef("Batch %s data %d: %+v", batch.ID, i, data)

	if data == nil {
		l.Errorf("null data entry %d in batch '%s'", i, batch.ID)
		return nil // skip data entry
	}

	hash, err := data.CalcHash(ctx)
	if err != nil {
		l.Errorf("Invalid data entry %d in batch '%s': %s", i, batch.ID, err)
		return nil //
	}
	if data.Hash == nil || *data.Hash != *hash {
		l.Errorf("Invalid data entry %d in batch '%s': Hash=%v Expected=%v", i, batch.ID, data.Hash, hash)
		return nil // skip data entry
	}

	// Insert the data, ensuring the hash doesn't change
	if err := em.database.UpsertData(ctx, data, true, false); err != nil {
		if err == database.HashMismatch {
			l.Errorf("Invalid data entry %d in batch '%s'. Hash mismatch with existing record with same UUID '%s' Hash=%s", i, batch.ID, data.ID, data.Hash)
			return nil // This is not retryable. skip this data entry
		}
		l.Errorf("Failed to insert data entry %d in batch '%s': %s", i, batch.ID, err)
		return err // a peristence failure here is considered retryable (so returned)
	}

	return nil
}

func (em *eventManager) persistBatchMessage(ctx context.Context /* db TX context*/, batch *fftypes.Batch, i int, msg *fftypes.Message) error {
	l := log.L(ctx)
	l.Tracef("Batch %s message %d: %+v", batch.ID, i, msg)

	if msg == nil {
		l.Errorf("null message entry %d in batch '%s'", i, batch.ID)
		return nil // skip entry
	}

	if msg.Header.Author != batch.Author {
		l.Errorf("Mismatched author '%s' on message entry %d in batch '%s'", msg.Header.Author, i, batch.ID)
		return nil // skip entry
	}

	err := msg.Verify(ctx)
	if err != nil {
		l.Errorf("Invalid message entry %d in batch '%s': %s", i, batch.ID, err)
		return nil // skip message entry
	}

	// Insert the message, ensuring the hash doesn't change.
	// We do not mark it as confirmed at this point, that's the job of the aggregator.
	if err = em.database.UpsertMessage(ctx, msg, true, false); err != nil {
		if err == database.HashMismatch {
			l.Errorf("Invalid message entry %d in batch '%s'. Hash mismatch with existing record with same UUID '%s' Hash=%s", i, batch.ID, msg.Header.ID, msg.Hash)
			return nil // This is not retryable. skip this data entry
		}
		l.Errorf("Failed to insert message entry %d in batch '%s': %s", i, batch.ID, err)
		return err // a peristence failure here is considered retryable (so returned)
	}

	return nil
}
