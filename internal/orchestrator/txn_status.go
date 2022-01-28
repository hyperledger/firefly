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

package orchestrator

import (
	"context"
	"sort"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func updateStatus(result *fftypes.TransactionStatus, newStatus fftypes.OpStatus) {
	if result.Status != fftypes.OpStatusFailed && newStatus != fftypes.OpStatusSucceeded {
		result.Status = newStatus
	}
}

func pendingPlaceholder(t fftypes.TransactionStatusType) *fftypes.TransactionStatusDetails {
	return &fftypes.TransactionStatusDetails{
		Type:   t,
		Status: fftypes.OpStatusPending,
	}
}

func (or *orchestrator) GetTransactionStatus(ctx context.Context, ns, id string) (*fftypes.TransactionStatus, error) {
	result := &fftypes.TransactionStatus{
		Status:  fftypes.OpStatusSucceeded,
		Details: make([]*fftypes.TransactionStatusDetails, 0),
	}

	tx, err := or.GetTransactionByID(ctx, ns, id)
	if err != nil {
		return nil, err
	} else if tx == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}

	ops, _, err := or.GetTransactionOperations(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	for _, op := range ops {
		result.Details = append(result.Details, &fftypes.TransactionStatusDetails{
			Status:    op.Status,
			Type:      fftypes.TransactionStatusTypeOperation,
			SubType:   op.Type.String(),
			Timestamp: op.Updated,
			ID:        op.ID,
			Error:     op.Error,
			Info:      op.Output,
		})
		updateStatus(result, op.Status)
	}

	events, _, err := or.GetTransactionBlockchainEvents(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		result.Details = append(result.Details, &fftypes.TransactionStatusDetails{
			Status:    fftypes.OpStatusSucceeded,
			Type:      fftypes.TransactionStatusTypeBlockchainEvent,
			SubType:   event.Name,
			Timestamp: event.Timestamp,
			ID:        event.ID,
			Info:      event.Info,
		})
	}

	switch tx.Type {
	case fftypes.TransactionTypeBatchPin:
		if len(events) == 0 {
			result.Details = append(result.Details, pendingPlaceholder(fftypes.TransactionStatusTypeBlockchainEvent))
			updateStatus(result, fftypes.OpStatusPending)
		}
		f := database.BatchQueryFactory.NewFilter(ctx)
		switch batches, _, err := or.database.GetBatches(ctx, f.Eq("tx.id", id)); {
		case err != nil:
			return nil, err
		case len(batches) == 0:
			result.Details = append(result.Details, pendingPlaceholder(fftypes.TransactionStatusTypeBatch))
			updateStatus(result, fftypes.OpStatusPending)
		default:
			result.Details = append(result.Details, &fftypes.TransactionStatusDetails{
				Status:    fftypes.OpStatusSucceeded,
				Type:      fftypes.TransactionStatusTypeBatch,
				SubType:   batches[0].Type.String(),
				Timestamp: batches[0].Confirmed,
				ID:        batches[0].ID,
			})
		}

	case fftypes.TransactionTypeTokenPool:
		if len(events) == 0 {
			result.Details = append(result.Details, pendingPlaceholder(fftypes.TransactionStatusTypeBlockchainEvent))
			updateStatus(result, fftypes.OpStatusPending)
		}
		f := database.TokenPoolQueryFactory.NewFilter(ctx)
		switch pools, _, err := or.database.GetTokenPools(ctx, f.Eq("tx.id", id)); {
		case err != nil:
			return nil, err
		case len(pools) == 0:
			result.Details = append(result.Details, pendingPlaceholder(fftypes.TransactionStatusTypeTokenPool))
			updateStatus(result, fftypes.OpStatusPending)
		case pools[0].State != fftypes.TokenPoolStateConfirmed:
			result.Details = append(result.Details, &fftypes.TransactionStatusDetails{
				Status:  fftypes.OpStatusPending,
				Type:    fftypes.TransactionStatusTypeTokenPool,
				SubType: pools[0].Type.String(),
				ID:      pools[0].ID,
			})
		default:
			result.Details = append(result.Details, &fftypes.TransactionStatusDetails{
				Status:    fftypes.OpStatusSucceeded,
				Type:      fftypes.TransactionStatusTypeTokenPool,
				SubType:   pools[0].Type.String(),
				Timestamp: pools[0].Created,
				ID:        pools[0].ID,
			})
		}

	case fftypes.TransactionTypeTokenTransfer:
		if len(events) == 0 {
			result.Details = append(result.Details, pendingPlaceholder(fftypes.TransactionStatusTypeBlockchainEvent))
			updateStatus(result, fftypes.OpStatusPending)
		}
		f := database.TokenTransferQueryFactory.NewFilter(ctx)
		switch transfers, _, err := or.database.GetTokenTransfers(ctx, f.Eq("tx.id", id)); {
		case err != nil:
			return nil, err
		case len(transfers) == 0:
			result.Details = append(result.Details, pendingPlaceholder(fftypes.TransactionStatusTypeTokenTransfer))
			updateStatus(result, fftypes.OpStatusPending)
		default:
			result.Details = append(result.Details, &fftypes.TransactionStatusDetails{
				Status:    fftypes.OpStatusSucceeded,
				Type:      fftypes.TransactionStatusTypeTokenTransfer,
				SubType:   transfers[0].Type.String(),
				Timestamp: transfers[0].Created,
				ID:        transfers[0].LocalID,
			})
		}

	case fftypes.TransactionTypeContractInvoke:
		// no blockchain events or other objects

	default:
		return nil, i18n.NewError(ctx, i18n.MsgUnknownTransactionType, tx.Type)
	}

	// Sort with nil timestamps first (ie Pending), then descending by timestamp
	sort.SliceStable(result.Details, func(i, j int) bool {
		x := result.Details[i].Timestamp
		y := result.Details[j].Timestamp
		switch {
		case y == nil:
			return false
		case x == nil:
			return true
		default:
			return x.Time().After(*y.Time())
		}
	})

	return result, nil
}
