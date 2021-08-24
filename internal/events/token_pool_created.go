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
	"github.com/hyperledger-labs/firefly/pkg/tokens"
)

func (em *eventManager) persistTokenPoolTransaction(ctx context.Context, pool *fftypes.TokenPool, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) (valid bool, err error) {
	if pool.ID == nil || pool.TransactionID == nil {
		log.L(ctx).Errorf("Invalid token pool '%s'. Missing ID (%v) or transaction ID (%v)", pool.ID, pool.ID, pool.TransactionID)
		return false, nil // This is not retryable
	}
	if err := fftypes.ValidateFFNameField(ctx, pool.Namespace, "namespace"); err != nil {
		log.L(ctx).Errorf("Invalid token pool '%s'. Transaction '%s' invalid namespace '%s': %a", pool.ID, pool.TransactionID, pool.Namespace, err)
		return false, nil // This is not retryable
	}
	// Get any existing record for the batch transaction record
	tx, err := em.database.GetTransactionByID(ctx, pool.TransactionID)
	if err != nil {
		return false, err // a peristence failure here is considered retryable (so returned)
	}
	if tx == nil {
		// We're the first to write the transaction record on this node
		tx = &fftypes.Transaction{
			ID: pool.TransactionID,
			Subject: fftypes.TransactionSubject{
				Namespace: pool.Namespace,
				Type:      fftypes.TransactionTypeTokenPool,
				Signer:    signingIdentity,
				Reference: pool.ID,
			},
			Created: fftypes.Now(),
		}
		tx.Hash = tx.Subject.Hash()
	} else if tx.Subject.Type != fftypes.TransactionTypeTokenPool ||
		tx.Subject.Signer != signingIdentity ||
		tx.Subject.Reference == nil ||
		*tx.Subject.Reference != *pool.ID ||
		tx.Subject.Namespace != pool.Namespace {
		log.L(ctx).Errorf("Invalid token pool '%s'. Existing transaction '%s' does not match subject", pool.ID, tx.ID)
		return false, nil // This is not retryable. skip this batch
	}

	// Set the updates on the transaction
	tx.ProtocolID = protocolTxID
	tx.Info = additionalInfo
	tx.Status = fftypes.OpStatusSucceeded

	// Upsert the transaction, ensuring the hash does not change
	err = em.database.UpsertTransaction(ctx, tx, true, false)
	if err != nil {
		if err == database.HashMismatch {
			log.L(ctx).Errorf("Invalid token pool '%s'. Transaction '%s' hash mismatch with existing record", pool.ID, tx.Hash)
			return false, nil // This is not retryable. skip this batch
		}
		log.L(ctx).Errorf("Failed to insert transaction for token pool '%s': %s", pool.ID, err)
		return false, err // a peristence failure here is considered retryable (so returned)
	}

	return true, nil
}

func (em *eventManager) persistTokenPool(ctx context.Context, pool *fftypes.TokenPool) (valid bool, err error) {
	l := log.L(ctx)
	if err := fftypes.ValidateFFNameField(ctx, pool.Name, "name"); err != nil {
		l.Errorf("Invalid token pool '%s' - invalid name '%s': %a", pool.ID, pool.Name, err)
		return false, nil // This is not retryable
	}
	err = em.database.UpsertTokenPool(ctx, pool, false)
	if err != nil {
		if err == database.IDMismatch {
			log.L(ctx).Errorf("Invalid token pool '%s'. ID mismatch with existing record", pool.ID)
			return false, nil // This is not retryable
		}
		l.Errorf("Failed to insert token pool '%s': %s", pool.ID, err)
		return false, err // a peristence failure here is considered retryable (so returned)
	}
	return true, nil
}

func (em *eventManager) TokenPoolCreated(tk tokens.Plugin, pool *fftypes.TokenPool, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	return em.retry.Do(em.ctx, "persist token pool", func(attempt int) (bool, error) {
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			valid, err := em.persistTokenPoolTransaction(ctx, pool, signingIdentity, protocolTxID, additionalInfo)
			if valid && err == nil {
				valid, err = em.persistTokenPool(ctx, pool)
			}
			if err != nil {
				return err
			}
			if !valid {
				log.L(em.ctx).Warnf("Token pool rejected id=%s author=%s", pool.ID, signingIdentity)
				event := fftypes.NewEvent(fftypes.EventTypePoolRejected, pool.Namespace, pool.ID)
				return em.database.InsertEvent(em.ctx, event)
			}
			log.L(em.ctx).Infof("Token pool created id=%s author=%s", pool.ID, signingIdentity)
			event := fftypes.NewEvent(fftypes.EventTypePoolConfirmed, pool.Namespace, pool.ID)
			return em.database.InsertEvent(em.ctx, event)
		})
		return err != nil, err // retry indefinitely (until context closes)
	})
}
