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

func (em *eventManager) persistOpUpdate(ctx context.Context, operationID *fftypes.UUID, txState fftypes.OpStatus, errorMessage string, opOutput fftypes.JSONObject) error {
	op, err := em.database.GetOperationByID(ctx, operationID)
	if err != nil || op == nil {
		log.L(ctx).Warnf("Operation update '%s' ignored, as it was not submitted by this node", operationID)
		return nil
	}

	update := database.OperationQueryFactory.NewUpdate(ctx).
		Set("status", txState).
		Set("error", errorMessage).
		Set("output", opOutput)
	if err := em.database.UpdateOperation(ctx, op.ID, update); err != nil {
		return err
	}

	// Special handling for operations that have cascading effects on other objects
	switch op.Type {
	case fftypes.OpTypeTokenTransfer:
		if txState == fftypes.OpStatusFailed {
			txUpdate := database.TransactionQueryFactory.NewUpdate(ctx).Set("status", txState)
			if err := em.database.UpdateTransaction(ctx, op.Transaction, txUpdate); err != nil {
				return err
			}

			event := fftypes.NewEvent(fftypes.EventTypeTransferOpFailed, op.Namespace, op.ID)
			if err := em.database.InsertEvent(ctx, event); err != nil {
				return err
			}
		}

	case fftypes.OpTypeContractInvoke:
		txUpdate := database.TransactionQueryFactory.NewUpdate(ctx).Set("status", txState)
		if err := em.database.UpdateTransaction(ctx, op.Transaction, txUpdate); err != nil {
			return err
		}
	}

	return nil
}

func (em *eventManager) OperationUpdate(plugin fftypes.Named, operationID *fftypes.UUID, status fftypes.OpStatus, errorMessage string, opOutput fftypes.JSONObject) error {
	return em.retry.Do(em.ctx, "persist operation status", func(attempt int) (bool, error) {
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			return em.persistOpUpdate(ctx, operationID, status, errorMessage, opOutput)
		})
		return err != nil, err
	})
}
