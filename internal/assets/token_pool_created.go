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

package assets

import (
	"context"

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

func updatePool(storedPool *fftypes.TokenPool, chainPool *tokens.TokenPool) *fftypes.TokenPool {
	storedPool.Type = chainPool.Type
	storedPool.ProtocolID = chainPool.ProtocolID
	storedPool.Key = chainPool.Key
	storedPool.Connector = chainPool.Connector
	storedPool.Standard = chainPool.Standard
	if chainPool.TransactionID != nil {
		storedPool.TX = fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   chainPool.TransactionID,
		}
	}
	return storedPool
}

func poolTransaction(pool *fftypes.TokenPool, status fftypes.OpStatus, protocolTxID string, additionalInfo fftypes.JSONObject) *fftypes.Transaction {
	return &fftypes.Transaction{
		ID:     pool.TX.ID,
		Status: status,
		Subject: fftypes.TransactionSubject{
			Namespace: pool.Namespace,
			Type:      pool.TX.Type,
			Signer:    pool.Key,
			Reference: pool.ID,
		},
		ProtocolID: protocolTxID,
		Info:       additionalInfo,
	}
}

func (am *assetManager) confirmPool(ctx context.Context, pool *fftypes.TokenPool, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	tx := poolTransaction(pool, fftypes.OpStatusSucceeded, protocolTxID, additionalInfo)
	if valid, err := am.txhelper.PersistTransaction(ctx, tx); !valid || err != nil {
		return err
	}
	pool.State = fftypes.TokenPoolStateConfirmed
	if err := am.database.UpsertTokenPool(ctx, pool); err != nil {
		return err
	}
	log.L(ctx).Infof("Token pool confirmed id=%s author=%s", pool.ID, pool.Key)
	event := fftypes.NewEvent(fftypes.EventTypePoolConfirmed, pool.Namespace, pool.ID)
	return am.database.InsertEvent(ctx, event)
}

func (am *assetManager) findTokenPoolCreateOp(ctx context.Context, tx *fftypes.UUID) (*fftypes.Operation, error) {
	// Find a matching operation within this transaction
	fb := database.OperationQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("tx", tx),
		fb.Eq("type", fftypes.OpTypeTokenCreatePool),
	)
	if operations, _, err := am.database.GetOperations(ctx, filter); err != nil {
		return nil, err
	} else if len(operations) > 0 {
		return operations[0], nil
	}
	return nil, nil
}

func (am *assetManager) TokenPoolCreated(ti tokens.Plugin, pool *tokens.TokenPool, protocolTxID string, additionalInfo fftypes.JSONObject) (err error) {
	var announcePool *fftypes.TokenPool

	err = am.retry.Do(am.ctx, "persist token pool transaction", func(attempt int) (bool, error) {
		err := am.database.RunAsGroup(am.ctx, func(ctx context.Context) error {
			// See if this is a confirmation of an unconfirmed pool
			if existingPool, err := am.database.GetTokenPoolByProtocolID(ctx, pool.Connector, pool.ProtocolID); err != nil {
				return err
			} else if existingPool != nil {
				if existingPool.State == fftypes.TokenPoolStateConfirmed {
					return nil // already confirmed
				}
				return am.confirmPool(ctx, updatePool(existingPool, pool), protocolTxID, additionalInfo)
			}

			// See if this pool was submitted locally and needs to be announced
			if op, err := am.findTokenPoolCreateOp(ctx, pool.TransactionID); err != nil {
				return err
			} else if op != nil {
				announcePool = updatePool(&fftypes.TokenPool{}, pool)
				if err = retrieveTokenPoolCreateInputs(ctx, op, announcePool); err != nil {
					log.L(ctx).Errorf("Error loading pool info for transaction '%s' (%s) - ignoring: %v", pool.TransactionID, err, op.Input)
					announcePool = nil
					return nil
				}
				nextOp := fftypes.NewTXOperation(
					ti,
					op.Namespace,
					op.Transaction,
					"",
					fftypes.OpTypeTokenAnnouncePool,
					fftypes.OpStatusPending)
				return am.database.UpsertOperation(ctx, nextOp, false)
			}

			// Otherwise this event can be ignored
			log.L(ctx).Debugf("Ignoring token pool transaction '%s' - pool %s is not active", pool.TransactionID, pool.ProtocolID)
			return nil
		})
		return err != nil, err
	})

	if err == nil && announcePool != nil {
		// Announce the details of the new token pool and the transaction object
		// Other nodes will pass these details to their own token connector for validation/activation of the pool
		broadcast := &fftypes.TokenPoolAnnouncement{
			Pool: announcePool,
			TX:   poolTransaction(announcePool, fftypes.OpStatusPending, protocolTxID, additionalInfo),
		}
		log.L(am.ctx).Infof("Announcing token pool id=%s author=%s", announcePool.ID, pool.Key)
		_, err = am.broadcast.BroadcastTokenPool(am.ctx, announcePool.Namespace, broadcast, false)
	}

	return err
}
