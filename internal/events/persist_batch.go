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
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type messageAndData struct {
	message *fftypes.Message
	data    fftypes.DataArray
}

// persistBatch performs very simple validation on each message/data element (hashes) and either persists
// or discards them. Errors are returned only in the case of database failures, which should be retried.
func (em *eventManager) persistBatch(ctx context.Context, batch *fftypes.Batch) (persistedBatch *fftypes.BatchPersisted, valid bool, err error) {
	l := log.L(ctx)

	if batch.ID == nil || batch.Payload.TX.ID == nil || batch.Hash == nil {
		l.Errorf("Invalid batch. Missing ID (%v), transaction ID (%s) or hash (%s)", batch.ID, batch.Payload.TX.ID, batch.Hash)
		return nil, false, nil // This is not retryable. skip this batch
	}

	if len(batch.Payload.Messages) == 0 || len(batch.Payload.Data) == 0 {
		l.Errorf("Invalid batch '%s'. Missing messages (%d) or data (%d)", batch.ID, len(batch.Payload.Messages), len(batch.Payload.Data))
		return nil, false, nil // This is not retryable. skip this batch
	}

	switch batch.Payload.TX.Type {
	case fftypes.TransactionTypeBatchPin:
	case fftypes.TransactionTypeUnpinned:
	default:
		l.Errorf("Invalid batch '%s'. Invalid transaction type: %s", batch.ID, batch.Payload.TX.Type)
		return nil, false, nil // This is not retryable. skip this batch
	}

	// Set confirmed on the batch (the messages should not be confirmed at this point - that's the aggregator's job)
	persistedBatch, manifest := batch.Confirmed()
	manifestHash := fftypes.HashString(persistedBatch.Manifest.String())

	// Verify the hash calculation.
	if !manifestHash.Equals(batch.Hash) {
		// To cope with existing batches written by v0.13 and older environments, we have to do a more expensive
		// hashing of the whole payload before we reject.
		if batch.Payload.Hash().Equals(batch.Hash) {
			l.Infof("Persisting migrated batch '%s'. Hash is a payload hash: %s", batch.ID, batch.Hash)
		} else {
			l.Errorf("Invalid batch '%s'. Hash does not match payload. Found=%s Expected=%s", batch.ID, manifestHash, batch.Hash)
			return nil, false, nil // This is not retryable. skip this batch
		}
	}

	// Upsert the batch
	err = em.database.UpsertBatch(ctx, persistedBatch)
	if err != nil {
		if err == database.HashMismatch {
			l.Errorf("Invalid batch '%s'. Batch hash mismatch with existing record", batch.ID)
			return nil, false, nil // This is not retryable. skip this batch
		}
		l.Errorf("Failed to insert batch '%s': %s", batch.ID, err)
		return nil, false, err // a persistence failure here is considered retryable (so returned)
	}

	valid, err = em.validateAndPersistBatchContent(ctx, batch)
	if err != nil || !valid {
		return nil, valid, err
	}
	em.aggregator.cacheBatch(em.aggregator.getBatchCacheKey(persistedBatch.ID, persistedBatch.Hash), persistedBatch, manifest)
	return persistedBatch, true, err
}

func (em *eventManager) validateAndPersistBatchContent(ctx context.Context, batch *fftypes.Batch) (valid bool, err error) {

	// Insert the data entries
	dataByID := make(map[fftypes.UUID]*fftypes.Data)
	for i, data := range batch.Payload.Data {
		if valid = em.validateBatchData(ctx, batch, i, data); !valid {
			return false, nil
		}
		if valid, err = em.checkAndInitiateBlobDownloads(ctx, batch, i, data); !valid || err != nil {
			return false, err
		}
		dataByID[*data.ID] = data
	}

	// Insert the message entries
	for i, msg := range batch.Payload.Messages {
		if valid = em.validateBatchMessage(ctx, batch, i, msg); !valid {
			return false, nil
		}
	}

	// We require that the batch contains exactly the set of data that is in the messages - no more or less.
	// While this means an edge case inefficiencly of re-transmission of data when sent in multiple messages,
	// that is outweighed by the efficiency it allows in the insertion logic in the majority case.
	matchedData := make(map[fftypes.UUID]bool)
	matchedMsgs := make([]*messageAndData, len(batch.Payload.Messages))
	for iMsg, msg := range batch.Payload.Messages {
		msgData := make(fftypes.DataArray, len(msg.Data))
		for di, dataRef := range msg.Data {
			msgData[di] = dataByID[*dataRef.ID]
			if msgData[di] == nil || !msgData[di].Hash.Equals(dataRef.Hash) {
				log.L(ctx).Errorf("Message '%s' in batch '%s' - data not in-line in batch id='%s' hash='%s'", msg.Header.ID, batch.ID, dataRef.ID, dataRef.Hash)
				return false, nil
			}
			matchedData[*dataRef.ID] = true
		}
		matchedMsgs[iMsg] = &messageAndData{
			message: msg,
			data:    msgData,
		}
	}
	if len(matchedData) != len(dataByID) {
		log.L(ctx).Errorf("Batch '%s' contains %d unique data, but %d are referenced from messages", batch.ID, len(dataByID), len(matchedData))
		return false, nil
	}

	return em.persistBatchContent(ctx, batch, matchedMsgs)
}

func (em *eventManager) validateBatchData(ctx context.Context, batch *fftypes.Batch, i int, data *fftypes.Data) bool {

	l := log.L(ctx)
	l.Tracef("Batch '%s' data %d: %+v", batch.ID, i, data)

	if data == nil {
		l.Errorf("null data entry %d in batch '%s'", i, batch.ID)
		return false
	}

	hash, err := data.CalcHash(ctx)
	if err != nil {
		log.L(ctx).Errorf("Invalid data entry %d in batch '%s': %s", i, batch.ID, err)
		return false
	}
	if data.Hash == nil || *data.Hash != *hash {
		log.L(ctx).Errorf("Invalid data entry %d in batch '%s': Hash=%v Expected=%v", i, batch.ID, data.Hash, hash)
		return false
	}

	return true
}

func (em *eventManager) checkAndInitiateBlobDownloads(ctx context.Context, batch *fftypes.Batch, i int, data *fftypes.Data) (bool, error) {

	if data.Blob != nil && batch.Type == fftypes.BatchTypeBroadcast {
		// Need to check if we need to initiate a download
		blob, err := em.database.GetBlobMatchingHash(ctx, data.Blob.Hash)
		if err != nil {
			return false, err
		}
		if blob == nil {
			if data.Blob.Public == "" {
				log.L(ctx).Errorf("Invalid data entry %d id=%s in batch '%s' - missing public blob reference", i, data.ID, batch.ID)
				return false, nil
			}
			if err = em.sharedDownload.InitiateDownloadBlob(ctx, data.Namespace, batch.Payload.TX.ID, data.ID, data.Blob.Public); err != nil {
				return false, err
			}
		}

	}

	return true, nil
}

func (em *eventManager) validateBatchMessage(ctx context.Context, batch *fftypes.Batch, i int, msg *fftypes.Message) bool {

	l := log.L(ctx)
	if msg == nil {
		l.Errorf("null message entry %d in batch '%s'", i, batch.ID)
		return false
	}

	if msg.Header.Author != batch.Author || msg.Header.Key != batch.Key {
		log.L(ctx).Errorf("Mismatched key/author '%s'/'%s' on message entry %d in batch '%s'", msg.Header.Key, msg.Header.Author, i, batch.ID)
		return false
	}
	msg.BatchID = batch.ID

	l.Tracef("Batch '%s' message %d: %+v", batch.ID, i, msg)

	err := msg.Verify(ctx)
	if err != nil {
		l.Errorf("Invalid message entry %d in batch '%s': %s", i, batch.ID, err)
		return false
	}
	// Set the state to pending, for the insertion stage
	msg.State = fftypes.MessageStatePending

	return true
}

func (em *eventManager) sentByUs(ctx context.Context, batch *fftypes.Batch) bool {
	localNode := em.ni.GetNodeUUID(ctx)
	if batch.Node == nil {
		// This is from a node that hasn't yet completed registration, so we can't optimize
		return false
	} else if batch.Node.Equals(localNode) {
		// We sent the batch, so we should already have all the messages and data locally
		return true
	}
	// We didn't send the batch, so all the data should be new - optimize the DB operations for that
	return false
}

func (em *eventManager) verifyAlreadyStored(ctx context.Context, batch *fftypes.Batch) (valid bool, err error) {
	for _, msg := range batch.Payload.Messages {
		msgLocal, _, _, err := em.data.GetMessageWithDataCached(ctx, msg.Header.ID)
		if err != nil {
			return false, err
		}
		if msgLocal == nil {
			log.L(ctx).Errorf("Message entry %s in batch sent by this node, was not found", msg.Header.ID)
			return false, nil
		}
		if !msgLocal.Hash.Equals(msg.Hash) {
			log.L(ctx).Errorf("Message entry %s hash mismatch with already stored. Local=%s BatchMsg=%s", msg.Header.ID, msgLocal.Hash, msg.Hash)
			return false, nil
		}
	}
	return true, nil
}

func (em *eventManager) persistBatchContent(ctx context.Context, batch *fftypes.Batch, matchedMsgs []*messageAndData) (valid bool, err error) {

	// We want to insert the messages and data in the most efficient way we can.
	// If we are sure we wrote the batch, then we do a cached lookup of each in turn - which is efficient
	// because all of those should be in the cache as we wrote them recently.
	if em.sentByUs(ctx, batch) {
		allStored, err := em.verifyAlreadyStored(ctx, batch)
		if err != nil {
			return false, err
		}
		if allStored {
			return true, nil
		}
		// Fall through otherwise
		log.L(ctx).Warnf("Batch %s was sent by our UUID, but the content was not already stored. Assuming node has been reset", batch.ID)
	}

	// Otherwise try a one-shot insert of all the data, on the basis it's likely unique
	err = em.database.InsertDataArray(ctx, batch.Payload.Data)
	if err != nil {
		log.L(ctx).Debugf("Batch data insert optimization failed for batch '%s': %s", batch.ID, err)
		// Fall back to individual upserts
		for i, data := range batch.Payload.Data {
			if err := em.database.UpsertData(ctx, data, database.UpsertOptimizationExisting); err != nil {
				if err == database.HashMismatch {
					log.L(ctx).Errorf("Invalid data entry %d in batch '%s'. Hash mismatch with existing record with same UUID '%s' Hash=%s", i, batch.ID, data.ID, data.Hash)
					return false, nil
				}
				log.L(ctx).Errorf("Failed to insert data entry %d in batch '%s': %s", i, batch.ID, err)
				return false, err
			}
		}
	}

	// Then the same one-shot insert of all the mesages, on the basis they are likely unique (even if
	// one of the data elements wasn't unique). Likely reasons for exceptions here are idempotent replay,
	// or a root broadcast where "em.sentByUs" returned false, but we actually sent it.
	err = em.database.InsertMessages(ctx, batch.Payload.Messages)
	if err != nil {
		log.L(ctx).Debugf("Batch message insert optimization failed for batch '%s': %s", batch.ID, err)
		// Fall back to individual upserts
		for i, msg := range batch.Payload.Messages {
			if err = em.database.UpsertMessage(ctx, msg, database.UpsertOptimizationExisting); err != nil {
				if err == database.HashMismatch {
					log.L(ctx).Errorf("Invalid message entry %d in batch'%s'. Hash mismatch with existing record with same UUID '%s' Hash=%s", i, batch.ID, msg.Header.ID, msg.Hash)
					return false, nil // This is not retryable. skip this data entry
				}
				log.L(ctx).Errorf("Failed to insert message entry %d in batch '%s': %s", i, batch.ID, err)
				return false, err // a persistence failure here is considered retryable (so returned)
			}
		}
	}

	// If all is well, update the cache before we return
	for _, mm := range matchedMsgs {
		em.data.UpdateMessageCache(mm.message, mm.data)
	}
	return true, nil
}
