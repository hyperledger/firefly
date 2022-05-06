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

	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/log"
	"github.com/hyperledger/firefly/pkg/tokens"
)

// Determine if this transfer event should use a LocalID that was pre-assigned to an operation submitted by this node.
// This will ensure that the original LocalID provided to the user can later be used in a lookup, and also causes requests that
// use "confirm=true" to resolve as expected.
// Must follow these rules to reuse the LocalID:
// - The transaction ID on the transfer must match a transaction+operation initiated by this node.
// - The connector and pool for this event must match the connector and pool targeted by the initial operation. Connectors are
//   allowed to trigger side-effects in other pools, but only the event from the targeted pool should use the original LocalID.
// - The LocalID must not have been used yet. Connectors are allowed to emit multiple events in response to a single operation,
//   but only the first of them can use the original LocalID.
func (em *eventManager) loadTransferID(ctx context.Context, tx *fftypes.UUID, transfer *fftypes.TokenTransfer) (*fftypes.UUID, error) {
	// Find a matching operation within the transaction
	fb := database.OperationQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("tx", tx),
		fb.Eq("type", fftypes.OpTypeTokenTransfer),
	)
	operations, _, err := em.database.GetOperations(ctx, filter)
	if err != nil {
		return nil, err
	}

	if len(operations) > 0 {
		// This transfer matches a transfer transaction+operation submitted by this node.
		// Check the operation inputs to see if they match the connector and pool on this event.
		if input, err := txcommon.RetrieveTokenTransferInputs(ctx, operations[0]); err != nil {
			log.L(ctx).Warnf("Failed to read operation inputs for token transfer '%s': %s", transfer.ProtocolID, err)
		} else if input != nil && input.Connector == transfer.Connector && input.Pool.Equals(transfer.Pool) {
			// Check if the LocalID has already been used
			if existing, err := em.database.GetTokenTransferByID(ctx, input.LocalID); err != nil {
				return nil, err
			} else if existing == nil {
				// Everything matches - use the LocalID that was assigned up-front when the operation was submitted
				return input.LocalID, nil
			}
		}
	}

	return fftypes.NewUUID(), nil
}

func (em *eventManager) persistTokenTransfer(ctx context.Context, transfer *tokens.TokenTransfer) (valid bool, err error) {
	// Check that this is from a known pool
	// TODO: should cache this lookup for efficiency
	pool, err := em.database.GetTokenPoolByLocator(ctx, transfer.Connector, transfer.PoolLocator)
	if err != nil {
		return false, err
	}
	if pool == nil {
		log.L(ctx).Infof("Token transfer received for unknown pool '%s' - ignoring: %s", transfer.PoolLocator, transfer.Event.ProtocolID)
		return false, nil
	}
	transfer.Namespace = pool.Namespace
	transfer.Pool = pool.ID

	// Check that transfer has not already been recorded
	if existing, err := em.database.GetTokenTransferByProtocolID(ctx, transfer.Connector, transfer.ProtocolID); err != nil {
		return false, err
	} else if existing != nil {
		log.L(ctx).Warnf("Token transfer '%s' has already been recorded - ignoring", transfer.ProtocolID)
		return false, nil
	}

	if transfer.TX.ID == nil {
		transfer.LocalID = fftypes.NewUUID()
	} else {
		if transfer.LocalID, err = em.loadTransferID(ctx, transfer.TX.ID, &transfer.TokenTransfer); err != nil {
			return false, err
		}
		if valid, err := em.txHelper.PersistTransaction(ctx, transfer.Namespace, transfer.TX.ID, transfer.TX.Type, transfer.Event.BlockchainTXID); err != nil || !valid {
			return valid, err
		}
	}

	chainEvent := buildBlockchainEvent(pool.Namespace, nil, &transfer.Event, &fftypes.BlockchainTransactionRef{
		ID:           transfer.TX.ID,
		Type:         transfer.TX.Type,
		BlockchainID: transfer.Event.BlockchainTXID,
	})
	if err := em.maybePersistBlockchainEvent(ctx, chainEvent); err != nil {
		return false, err
	}
	em.emitBlockchainEventMetric(&transfer.Event)
	transfer.BlockchainEvent = chainEvent.ID

	if err := em.database.UpsertTokenTransfer(ctx, &transfer.TokenTransfer); err != nil {
		log.L(ctx).Errorf("Failed to record token transfer '%s': %s", transfer.ProtocolID, err)
		return false, err
	}
	if err := em.database.UpdateTokenBalances(ctx, &transfer.TokenTransfer); err != nil {
		log.L(ctx).Errorf("Failed to update accounts %s -> %s for token transfer '%s': %s", transfer.From, transfer.To, transfer.ProtocolID, err)
		return false, err
	}

	log.L(ctx).Infof("Token transfer recorded id=%s author=%s", transfer.ProtocolID, transfer.Key)
	if em.metrics.IsMetricsEnabled() {
		em.metrics.TransferConfirmed(&transfer.TokenTransfer)
	}

	return true, nil
}

func (em *eventManager) TokensTransferred(ti tokens.Plugin, transfer *tokens.TokenTransfer) error {
	var msgIDforRewind *fftypes.UUID

	err := em.retry.Do(em.ctx, "persist token transfer", func(attempt int) (bool, error) {
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			if valid, err := em.persistTokenTransfer(ctx, transfer); !valid || err != nil {
				return err
			}

			if transfer.Message != nil {
				msg, err := em.database.GetMessageByID(ctx, transfer.Message)
				switch {
				case err != nil:
					return err
				case msg != nil && msg.State == fftypes.MessageStateStaged:
					// Message can now be sent
					msg.State = fftypes.MessageStateReady
					if err := em.database.ReplaceMessage(ctx, msg); err != nil {
						return err
					}
				default:
					// Message might already have been received, we need to rewind
					msgIDforRewind = transfer.Message
				}
			}
			em.emitBlockchainEventMetric(&transfer.Event)

			event := fftypes.NewEvent(fftypes.EventTypeTransferConfirmed, transfer.Namespace, transfer.LocalID, transfer.TX.ID, transfer.Pool.String())
			return em.database.InsertEvent(ctx, event)
		})
		return err != nil, err // retry indefinitely (until context closes)
	})

	// Initiate a rewind if a batch was potentially completed by the arrival of this transfer
	if err == nil && msgIDforRewind != nil {
		em.aggregator.queueMessageRewind(msgIDforRewind)
	}

	return err
}
