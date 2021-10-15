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

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

func retrieveTokenTransferInputs(ctx context.Context, op *fftypes.Operation, transfer *fftypes.TokenTransfer) (err error) {
	input := &op.Input
	transfer.LocalID, err = fftypes.ParseUUID(ctx, input.GetString("id"))
	if err != nil {
		return err
	}
	return nil
}

func (em *eventManager) persistTokenTransaction(ctx context.Context, ns string, transfer *fftypes.TokenTransfer, protocolTxID string, additionalInfo fftypes.JSONObject) (valid bool, err error) {
	transfer.LocalID = nil

	// Find a matching operation within this transaction
	fb := database.OperationQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("tx", transfer.TX.ID),
		fb.Eq("type", fftypes.OpTypeTokenTransfer),
	)
	operations, _, err := em.database.GetOperations(ctx, filter)
	if err != nil {
		return false, err
	}
	if len(operations) > 0 {
		err = retrieveTokenTransferInputs(ctx, operations[0], transfer)
		if err != nil {
			log.L(ctx).Warnf("Failed to read operation inputs for token transfer '%s': %s", transfer.ProtocolID, err)
		}
	}

	if transfer.LocalID == nil {
		transfer.LocalID = fftypes.NewUUID()
	}

	transaction := &fftypes.Transaction{
		ID:     transfer.TX.ID,
		Status: fftypes.OpStatusSucceeded,
		Subject: fftypes.TransactionSubject{
			Namespace: ns,
			Type:      transfer.TX.Type,
			Signer:    transfer.Key,
			Reference: transfer.LocalID,
		},
		ProtocolID: protocolTxID,
		Info:       additionalInfo,
	}
	return em.txhelper.PersistTransaction(ctx, transaction)
}

func (em *eventManager) getBatchForTransfer(ctx context.Context, transfer *fftypes.TokenTransfer) (*fftypes.UUID, error) {
	// Find the messages assocated with that data
	var messages []*fftypes.Message
	fb := database.MessageQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("pending", true),
		fb.Eq("hash", transfer.MessageHash),
	)
	messages, _, err := em.database.GetMessages(ctx, filter)
	if err != nil || len(messages) == 0 {
		return nil, err
	}
	return messages[0].BatchID, nil
}

func (em *eventManager) TokensTransferred(tk tokens.Plugin, transfer *fftypes.TokenTransfer, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	var batchID *fftypes.UUID

	err := em.retry.Do(em.ctx, "persist token transfer", func(attempt int) (bool, error) {
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			// Check that this is from a known pool
			pool, err := em.database.GetTokenPoolByProtocolID(ctx, transfer.PoolProtocolID)
			if err != nil {
				return err
			}
			if pool == nil {
				log.L(ctx).Warnf("Token transfer received for unknown pool '%s' - ignoring: %s", transfer.PoolProtocolID, protocolTxID)
				return nil
			}

			if transfer.TX.ID != nil {
				if valid, err := em.persistTokenTransaction(ctx, pool.Namespace, transfer, protocolTxID, additionalInfo); err != nil || !valid {
					return err
				}
			} else {
				transfer.LocalID = fftypes.NewUUID()
			}

			if err := em.database.UpsertTokenTransfer(ctx, transfer); err != nil {
				log.L(ctx).Errorf("Failed to record token transfer '%s': %s", transfer.ProtocolID, err)
				return err
			}

			balance := &fftypes.TokenBalanceChange{
				PoolProtocolID: transfer.PoolProtocolID,
				TokenIndex:     transfer.TokenIndex,
			}

			if transfer.Type != fftypes.TokenTransferTypeMint {
				balance.Identity = transfer.From
				balance.Amount.Int().Neg(transfer.Amount.Int())
				if err := em.database.AddTokenAccountBalance(ctx, balance); err != nil {
					log.L(ctx).Errorf("Failed to update account '%s' for token transfer '%s': %s", balance.Identity, transfer.ProtocolID, err)
					return err
				}
			}

			if transfer.Type != fftypes.TokenTransferTypeBurn {
				balance.Identity = transfer.To
				balance.Amount.Int().Set(transfer.Amount.Int())
				if err := em.database.AddTokenAccountBalance(ctx, balance); err != nil {
					log.L(ctx).Errorf("Failed to update account '%s for token transfer '%s': %s", balance.Identity, transfer.ProtocolID, err)
					return err
				}
			}

			log.L(ctx).Infof("Token transfer recorded id=%s author=%s", transfer.ProtocolID, transfer.Key)

			if transfer.MessageHash != nil {
				if batchID, err = em.getBatchForTransfer(ctx, transfer); err != nil {
					log.L(ctx).Errorf("Failed to lookup batch for token transfer '%s': %s", transfer.ProtocolID, err)
					return err
				}
			}

			event := fftypes.NewEvent(fftypes.EventTypeTransferConfirmed, pool.Namespace, transfer.LocalID)
			return em.database.InsertEvent(ctx, event)
		})
		return err != nil, err // retry indefinitely (until context closes)
	})

	if err == nil {
		// Initiate a rewind if a batch was potentially completed by the arrival of this transfer
		if batchID != nil {
			log.L(em.ctx).Infof("Batch '%s' contains reference to received transfer. Transfer='%s' Message='%s'", batchID, transfer.ProtocolID, transfer.MessageHash)
			em.aggregator.offchainBatches <- batchID
		}
	}

	return err
}
