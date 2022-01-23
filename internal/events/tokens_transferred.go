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
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

func (em *eventManager) loadTransferOperation(ctx context.Context, tx *fftypes.UUID, transfer *fftypes.TokenTransfer) error {
	transfer.LocalID = nil

	// Find a matching operation within this transaction
	fb := database.OperationQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("tx", tx),
		fb.Eq("type", fftypes.OpTypeTokenTransfer),
	)
	operations, _, err := em.database.GetOperations(ctx, filter)
	if err != nil {
		return err
	}
	if len(operations) > 0 {
		if err = txcommon.RetrieveTokenTransferInputs(ctx, operations[0], transfer); err != nil {
			log.L(ctx).Warnf("Failed to read operation inputs for token transfer '%s': %s", transfer.ProtocolID, err)
		}
	}

	if transfer.LocalID == nil {
		transfer.LocalID = fftypes.NewUUID()
	}
	return nil
}

func (em *eventManager) persistTokenTransfer(ctx context.Context, transfer *tokens.TokenTransfer) (valid bool, err error) {
	// Check that transfer has not already been recorded
	if existing, err := em.database.GetTokenTransferByProtocolID(ctx, transfer.Connector, transfer.ProtocolID); err != nil {
		return false, err
	} else if existing != nil {
		log.L(ctx).Warnf("Token transfer '%s' has already been recorded - ignoring", transfer.ProtocolID)
		return false, nil
	}

	// Check that this is from a known pool
	// TODO: should cache this lookup for efficiency
	pool, err := em.database.GetTokenPoolByProtocolID(ctx, transfer.Connector, transfer.PoolProtocolID)
	if err != nil {
		return false, err
	}
	if pool == nil {
		log.L(ctx).Infof("Token transfer received for unknown pool '%s' - ignoring: %s", transfer.PoolProtocolID, transfer.Event.ProtocolID)
		return false, nil
	}
	transfer.Namespace = pool.Namespace
	transfer.Pool = pool.ID

	if transfer.TX.ID != nil {
		if err := em.loadTransferOperation(ctx, transfer.TX.ID, &transfer.TokenTransfer); err != nil {
			return false, err
		}

		tx := &fftypes.Transaction{
			ID:        transfer.TX.ID,
			Status:    fftypes.OpStatusSucceeded,
			Namespace: transfer.Namespace,
			Type:      transfer.TX.Type,
		}
		if err := em.database.UpsertTransaction(ctx, tx); err != nil {
			return false, err
		}

		// Some operations result in multiple transfer events - if the protocol ID was unique but the
		// local ID is not unique, generate a unique local ID now.
		if existing, err := em.database.GetTokenTransfer(ctx, transfer.LocalID); err != nil {
			return false, err
		} else if existing != nil {
			transfer.LocalID = fftypes.NewUUID()
		}
	} else {
		transfer.LocalID = fftypes.NewUUID()
	}

	chainEvent := buildBlockchainEvent(pool.Namespace, nil, &transfer.Event, &transfer.TX)
	transfer.BlockchainEvent = chainEvent.ID
	if err := em.persistBlockchainEvent(ctx, chainEvent); err != nil {
		return false, err
	}

	if err := em.database.UpsertTokenTransfer(ctx, &transfer.TokenTransfer); err != nil {
		log.L(ctx).Errorf("Failed to record token transfer '%s': %s", transfer.ProtocolID, err)
		return false, err
	}
	if err := em.database.UpdateTokenBalances(ctx, &transfer.TokenTransfer); err != nil {
		log.L(ctx).Errorf("Failed to update accounts %s -> %s for token transfer '%s': %s", transfer.From, transfer.To, transfer.ProtocolID, err)
		return false, err
	}
	log.L(ctx).Infof("Token transfer recorded id=%s author=%s", transfer.ProtocolID, transfer.Key)
	return true, nil
}

func (em *eventManager) TokensTransferred(ti tokens.Plugin, transfer *tokens.TokenTransfer) error {
	var batchID *fftypes.UUID

	err := em.retry.Do(em.ctx, "persist token transfer", func(attempt int) (bool, error) {
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			if valid, err := em.persistTokenTransfer(ctx, transfer); !valid || err != nil {
				return err
			}

			if transfer.Message != nil {
				if msg, err := em.database.GetMessageByID(ctx, transfer.Message); err != nil {
					return err
				} else if msg != nil {
					if msg.State == fftypes.MessageStateStaged {
						// Message can now be sent
						msg.State = fftypes.MessageStateReady
						if err := em.database.UpdateAndBumpMessage(ctx, msg); err != nil {
							return err
						}
					} else {
						// Message was already received - aggregator will need to be rewound
						batchID = msg.BatchID
					}
				}
			}

			event := fftypes.NewEvent(fftypes.EventTypeTransferConfirmed, transfer.Namespace, transfer.LocalID)
			return em.database.InsertEvent(ctx, event)
		})
		return err != nil, err // retry indefinitely (until context closes)
	})

	// Initiate a rewind if a batch was potentially completed by the arrival of this transfer
	if err == nil && batchID != nil {
		log.L(em.ctx).Infof("Batch '%s' contains reference to received transfer. Transfer='%s' Message='%s'", batchID, transfer.ProtocolID, transfer.Message)
		em.aggregator.offchainBatches <- batchID
	}

	return err
}
