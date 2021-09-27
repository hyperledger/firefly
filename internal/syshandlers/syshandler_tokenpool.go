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

package syshandlers

import (
	"context"

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (sh *systemHandlers) persistTokenPool(ctx context.Context, pool *fftypes.TokenPoolAnnouncement) (valid bool, err error) {
	// Find a matching operation within this transaction
	fb := database.OperationQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("tx", pool.TX.ID),
		fb.Eq("type", fftypes.OpTypeTokensAnnouncePool),
	)
	operations, _, err := sh.database.GetOperations(ctx, filter)
	if err != nil {
		return false, err // retryable
	}

	if len(operations) > 0 {
		// Mark announce operation completed
		update := database.OperationQueryFactory.NewUpdate(ctx).
			Set("status", fftypes.OpStatusSucceeded)
		if err := sh.database.UpdateOperation(ctx, operations[0].ID, update); err != nil {
			return false, err // retryable
		}

		// Validate received info matches the database
		transaction, err := sh.database.GetTransactionByID(ctx, pool.TX.ID)
		if err != nil {
			return false, err // retryable
		}
		if transaction.ProtocolID != pool.ProtocolTxID {
			log.L(ctx).Warnf("Ignoring token pool from transaction '%s' - unexpected protocol ID '%s'", pool.TX.ID, pool.ProtocolTxID)
			return false, nil // not retryable
		}

		// Mark transaction completed
		transaction.Status = fftypes.OpStatusSucceeded
		err = sh.database.UpsertTransaction(ctx, transaction, false)
		if err != nil {
			return false, err // retryable
		}
	} else {
		// No local announce operation found (broadcast originated from another node)
		log.L(ctx).Infof("Validating token pool transaction '%s' with protocol ID '%s'", pool.TX.ID, pool.ProtocolTxID)
		err = sh.assets.ValidateTokenPoolTx(ctx, &pool.TokenPool, pool.ProtocolTxID)
		if err != nil {
			log.L(ctx).Errorf("Failed to validate token pool transaction '%s': %v", pool.TX.ID, err)
			return false, err // retryable
		}
		transaction := &fftypes.Transaction{
			ID:     pool.TX.ID,
			Status: fftypes.OpStatusSucceeded,
			Subject: fftypes.TransactionSubject{
				Namespace: pool.Namespace,
				Type:      fftypes.TransactionTypeTokenPool,
				Signer:    pool.Author,
				Reference: pool.ID,
			},
			ProtocolID: pool.ProtocolTxID,
		}
		valid, err = sh.txhelper.PersistTransaction(ctx, transaction)
		if !valid || err != nil {
			return valid, err
		}
	}

	err = sh.database.UpsertTokenPool(ctx, &pool.TokenPool)
	if err != nil {
		if err == database.IDMismatch {
			log.L(ctx).Errorf("Invalid token pool '%s'. ID mismatch with existing record", pool.ID)
			return false, nil // not retryable
		}
		log.L(ctx).Errorf("Failed to insert token pool '%s': %s", pool.ID, err)
		return false, err // retryable
	}
	return true, nil
}

func (sh *systemHandlers) handleTokenPoolBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (valid bool, err error) {
	l := log.L(ctx)

	var pool fftypes.TokenPoolAnnouncement
	valid = sh.getSystemBroadcastPayload(ctx, msg, data, &pool)
	if valid {
		if err = pool.Validate(ctx, true); err != nil {
			l.Warnf("Unable to process token pool broadcast %s - validate failed: %s", msg.Header.ID, err)
			valid = false
		} else {
			pool.Message = msg.Header.ID
			valid, err = sh.persistTokenPool(ctx, &pool)
			if err != nil {
				return valid, err
			}
		}
	}

	var event *fftypes.Event
	if valid {
		l.Infof("Token pool created id=%s author=%s", pool.ID, msg.Header.Author)
		event = fftypes.NewEvent(fftypes.EventTypePoolConfirmed, pool.Namespace, pool.ID)
	} else {
		l.Warnf("Token pool rejected id=%s author=%s", pool.ID, msg.Header.Author)
		event = fftypes.NewEvent(fftypes.EventTypePoolRejected, pool.Namespace, pool.ID)
	}
	err = sh.database.InsertEvent(ctx, event)
	return valid, err
}
