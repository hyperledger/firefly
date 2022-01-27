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
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

func addPoolDetailsFromPlugin(ffPool *fftypes.TokenPool, pluginPool *tokens.TokenPool) {
	ffPool.Type = pluginPool.Type
	ffPool.ProtocolID = pluginPool.ProtocolID
	ffPool.Key = pluginPool.Key
	ffPool.Connector = pluginPool.Connector
	ffPool.Standard = pluginPool.Standard
	if pluginPool.TransactionID != nil {
		ffPool.TX = fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   pluginPool.TransactionID,
		}
	}
}

func poolTransaction(pool *fftypes.TokenPool, status fftypes.OpStatus, blockchainTXID string) *fftypes.Transaction {
	return &fftypes.Transaction{
		ID:            pool.TX.ID,
		Status:        status,
		Namespace:     pool.Namespace,
		Type:          pool.TX.Type,
		BlockchainIDs: fftypes.NewFFStringArray(blockchainTXID),
	}
}

func (em *eventManager) confirmPool(ctx context.Context, pool *fftypes.TokenPool, ev *blockchain.Event, blockchainTXID string) error {
	chainEvent := buildBlockchainEvent(pool.Namespace, nil, ev, &pool.TX)
	if err := em.persistBlockchainEvent(ctx, chainEvent); err != nil {
		return err
	}
	tx := poolTransaction(pool, fftypes.OpStatusSucceeded, blockchainTXID)
	if err := em.database.UpsertTransaction(ctx, tx); err != nil {
		return err
	}
	pool.State = fftypes.TokenPoolStateConfirmed
	if err := em.database.UpsertTokenPool(ctx, pool); err != nil {
		return err
	}
	log.L(ctx).Infof("Token pool confirmed id=%s author=%s", pool.ID, pool.Key)
	event := fftypes.NewEvent(fftypes.EventTypePoolConfirmed, pool.Namespace, pool.ID)
	return em.database.InsertEvent(ctx, event)
}

func (em *eventManager) findTokenPoolCreateOp(ctx context.Context, tx *fftypes.UUID) (*fftypes.Operation, error) {
	// Find a matching operation within this transaction
	fb := database.OperationQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("tx", tx),
		fb.Eq("type", fftypes.OpTypeTokenCreatePool),
	)
	if operations, _, err := em.database.GetOperations(ctx, filter); err != nil {
		return nil, err
	} else if len(operations) > 0 {
		return operations[0], nil
	}
	return nil, nil
}

func (em *eventManager) shouldConfirm(ctx context.Context, pool *tokens.TokenPool) (existingPool *fftypes.TokenPool, err error) {
	if existingPool, err = em.database.GetTokenPoolByProtocolID(ctx, pool.Connector, pool.ProtocolID); err != nil || existingPool == nil {
		return existingPool, err
	}
	addPoolDetailsFromPlugin(existingPool, pool)

	if existingPool.State == fftypes.TokenPoolStateUnknown {
		// Unknown pool state - should only happen on first run after database migration
		// Activate the pool, then immediately confirm
		// TODO: can this state eventually be removed?
		ev := buildBlockchainEvent(existingPool.Namespace, nil, &pool.Event, &existingPool.TX)
		if err = em.assets.ActivateTokenPool(ctx, existingPool, ev); err != nil {
			log.L(ctx).Errorf("Failed to activate token pool '%s': %s", existingPool.ID, err)
			return nil, err
		}
	}
	return existingPool, nil
}

func (em *eventManager) shouldAnnounce(ctx context.Context, ti tokens.Plugin, pool *tokens.TokenPool) (announcePool *fftypes.TokenPool, err error) {
	op, err := em.findTokenPoolCreateOp(ctx, pool.TransactionID)
	if err != nil {
		return nil, err
	} else if op == nil {
		return nil, nil
	}

	announcePool = &fftypes.TokenPool{}
	if err = txcommon.RetrieveTokenPoolCreateInputs(ctx, op, announcePool); err != nil {
		log.L(ctx).Errorf("Error loading pool info for transaction '%s' (%s) - ignoring: %v", pool.TransactionID, err, op.Input)
		return nil, nil
	}
	addPoolDetailsFromPlugin(announcePool, pool)

	nextOp := fftypes.NewTXOperation(
		ti,
		op.Namespace,
		op.Transaction,
		"",
		fftypes.OpTypeTokenAnnouncePool,
		fftypes.OpStatusPending)
	return announcePool, em.database.InsertOperation(ctx, nextOp)
}

// It is expected that this method might be invoked twice for each pool, depending on the behavior of the connector.
// It will be at least invoked on the submitter when the pool is first created, to trigger the submitter to announce it.
// It will be invoked on every node (including the submitter) after the pool is announced+activated, to trigger confirmation of the pool.
// When received in any other scenario, it should be ignored.
func (em *eventManager) TokenPoolCreated(ti tokens.Plugin, pool *tokens.TokenPool) (err error) {
	if pool.TransactionID == nil {
		log.L(em.ctx).Errorf("Invalid token pool transaction - ID is nil")
		return nil // move on
	}

	var batchID *fftypes.UUID
	var announcePool *fftypes.TokenPool

	err = em.retry.Do(em.ctx, "persist token pool transaction", func(attempt int) (bool, error) {
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			// See if this is a confirmation of an unconfirmed pool
			if existingPool, err := em.shouldConfirm(ctx, pool); err != nil {
				return err
			} else if existingPool != nil {
				if existingPool.State == fftypes.TokenPoolStateConfirmed {
					return nil // already confirmed
				}
				if msg, err := em.database.GetMessageByID(ctx, existingPool.Message); err != nil {
					return err
				} else if msg != nil {
					batchID = msg.BatchID // trigger rewind after completion of database transaction
				}
				return em.confirmPool(ctx, existingPool, &pool.Event, pool.Event.BlockchainTXID)
			}

			// See if this pool was submitted locally and needs to be announced
			if announcePool, err = em.shouldAnnounce(ctx, ti, pool); err != nil {
				return err
			} else if announcePool != nil {
				return nil // trigger announce after completion of database transaction
			}

			// Otherwise this event can be ignored
			log.L(ctx).Debugf("Ignoring token pool transaction '%s' - pool %s is not active", pool.Event.ProtocolID, pool.ProtocolID)
			return nil
		})
		return err != nil, err
	})

	if err == nil {
		// Initiate a rewind if a batch was potentially completed by the arrival of this transaction
		if batchID != nil {
			log.L(em.ctx).Infof("Batch '%s' contains reference to received pool '%s'", batchID, pool.ProtocolID)
			em.aggregator.offchainBatches <- batchID
		}

		// Announce the details of the new token pool with the blockchain event details
		// Other nodes will pass these details to their own token connector for validation/activation of the pool
		if announcePool != nil {
			broadcast := &fftypes.TokenPoolAnnouncement{
				Pool:  announcePool,
				Event: buildBlockchainEvent(announcePool.Namespace, nil, &pool.Event, &announcePool.TX),
			}
			log.L(em.ctx).Infof("Announcing token pool id=%s author=%s", announcePool.ID, pool.Key)
			_, err = em.broadcast.BroadcastTokenPool(em.ctx, announcePool.Namespace, broadcast, false)
		}
	}

	return err
}
