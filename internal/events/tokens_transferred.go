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
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

func (em *eventManager) loadTransferOperation(ctx context.Context, transfer *fftypes.TokenTransfer) error {
	transfer.LocalID = nil

	// Find a matching operation within this transaction
	fb := database.OperationQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("tx", transfer.TX.ID),
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

func (em *eventManager) persistTokenTransaction(ctx context.Context, ns string, transfer *fftypes.TokenTransfer, protocolTxID string, additionalInfo fftypes.JSONObject) (valid bool, err error) {
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

func (em *eventManager) getMessageForTransfer(ctx context.Context, transfer *fftypes.TokenTransfer) (*fftypes.Message, error) {
	var messages []*fftypes.Message
	fb := database.MessageQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("confirmed", nil),
		fb.Eq("hash", transfer.MessageHash),
	)
	messages, _, err := em.database.GetMessages(ctx, filter)
	if err != nil || len(messages) == 0 {
		return nil, err
	}
	return messages[0], nil
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
			transfer.Namespace = pool.Namespace

			if transfer.TX.ID != nil {
				if err := em.loadTransferOperation(ctx, transfer); err != nil {
					return err
				}
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
			if err := em.database.UpdateTokenAccountBalances(ctx, transfer); err != nil {
				log.L(ctx).Errorf("Failed to update accounts %s -> %s for token transfer '%s': %s", transfer.From, transfer.To, transfer.ProtocolID, err)
				return err
			}
			log.L(ctx).Infof("Token transfer recorded id=%s author=%s", transfer.ProtocolID, transfer.Key)

			if transfer.MessageHash != nil {
				msg, err := em.getMessageForTransfer(ctx, transfer)
				if err != nil {
					return err
				}
				if msg != nil {
					if msg.State == fftypes.MessageStateStaged {
						// Message can now be sent
						msg.State = fftypes.MessageStateReady
						if err := em.database.UpsertMessage(ctx, msg, true, false); err != nil {
							return err
						}
					} else {
						// Message was already received - aggregator will need to be rewound
						batchID = msg.BatchID
					}
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
