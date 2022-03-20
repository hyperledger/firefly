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
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (em *eventManager) operationUpdateCtx(ctx context.Context, operationID *fftypes.UUID, txState fftypes.OpStatus, blockchainTXID, errorMessage string, opOutput fftypes.JSONObject) error {
	op, err := em.database.GetOperationByID(ctx, operationID)
	if err != nil || op == nil {
		log.L(ctx).Warnf("Operation update '%s' ignored, as it was not submitted by this node", operationID)
		return nil
	}

	if err := em.database.ResolveOperation(ctx, op.ID, txState, errorMessage, opOutput); err != nil {
		return err
	}

	// Special handling for OpTypeTokenTransfer, which writes an event when it fails
	if op.Type == fftypes.OpTypeTokenTransfer && txState == fftypes.OpStatusFailed {
		tokenTransfer, err := txcommon.RetrieveTokenTransferInputs(ctx, op)
		topic := ""
		if tokenTransfer != nil {
			topic = tokenTransfer.Pool.String()
		}
		event := fftypes.NewEvent(fftypes.EventTypeTransferOpFailed, op.Namespace, op.ID, op.Transaction, topic)
		if err != nil || tokenTransfer.LocalID == nil || tokenTransfer.Type == "" {
			log.L(em.ctx).Warnf("Could not parse token transfer: %s", err)
		} else {
			event.Correlator = tokenTransfer.LocalID
			if em.metrics.IsMetricsEnabled() {
				em.metrics.TransferConfirmed(tokenTransfer)
			}
		}
		if err := em.database.InsertEvent(ctx, event); err != nil {
			return err
		}
	}

	// Special handling for OpTypeTokenApproval, which writes an event when it fails
	if op.Type == fftypes.OpTypeTokenApproval && txState == fftypes.OpStatusFailed {
		tokenApproval, err := txcommon.RetrieveTokenApprovalInputs(ctx, op)
		topic := ""
		if tokenApproval != nil {
			topic = tokenApproval.Pool.String()
		}
		event := fftypes.NewEvent(fftypes.EventTypeApprovalOpFailed, op.Namespace, op.ID, op.Transaction, topic)
		if err != nil || tokenApproval.LocalID == nil {
			log.L(em.ctx).Warnf("Could not parse token approval: %s", err)
		} else {
			event.Correlator = tokenApproval.LocalID
		}
		if err := em.database.InsertEvent(ctx, event); err != nil {
			return err
		}
	}

	return em.txHelper.AddBlockchainTX(ctx, op.Transaction, blockchainTXID)
}

func (em *eventManager) OperationUpdate(plugin fftypes.Named, operationID *fftypes.UUID, txState fftypes.OpStatus, blockchainTXID, errorMessage string, opOutput fftypes.JSONObject) error {
	return em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
		return em.operationUpdateCtx(ctx, operationID, txState, blockchainTXID, errorMessage, opOutput)
	})
}
