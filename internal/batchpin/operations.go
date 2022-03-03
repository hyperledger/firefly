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

package batchpin

import (
	"context"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type batchPinData struct {
	Batch    *fftypes.Batch     `json:"batch"`
	Contexts []*fftypes.Bytes32 `json:"contexts"`
}

func addBatchPinInputs(op *fftypes.Operation, batchID *fftypes.UUID, contexts []*fftypes.Bytes32) {
	contextStr := make([]string, len(contexts))
	for i, c := range contexts {
		contextStr[i] = c.String()
	}
	op.Input = fftypes.JSONObject{
		"batch":    batchID.String(),
		"contexts": contextStr,
	}
}

func retrieveBatchPinInputs(ctx context.Context, op *fftypes.Operation) (batchID *fftypes.UUID, contexts []*fftypes.Bytes32, err error) {
	batchID, err = fftypes.ParseUUID(ctx, op.Input.GetString("batch"))
	if err != nil {
		return nil, nil, err
	}
	contextStr := op.Input.GetStringArray("contexts")
	contexts = make([]*fftypes.Bytes32, len(contextStr))
	for i, c := range contextStr {
		contexts[i], err = fftypes.ParseBytes32(ctx, c)
		if err != nil {
			return nil, nil, err
		}
	}
	return batchID, contexts, nil
}

func (bp *batchPinSubmitter) PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error) {
	switch op.Type {
	case fftypes.OpTypeBlockchainBatchPin:
		batchID, contexts, err := retrieveBatchPinInputs(ctx, op)
		if err != nil {
			return nil, err
		}
		batch, err := bp.database.GetBatchByID(ctx, batchID)
		if err != nil {
			return nil, err
		} else if batch == nil {
			return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
		}
		return opBatchPin(op, batch, contexts), nil

	default:
		return nil, i18n.NewError(ctx, i18n.MsgOperationNotSupported)
	}
}

func (bp *batchPinSubmitter) RunOperation(ctx context.Context, op *fftypes.PreparedOperation) (complete bool, err error) {
	switch data := op.Data.(type) {
	case batchPinData:
		batch := data.Batch
		return false, bp.blockchain.SubmitBatchPin(ctx, op.ID, nil /* TODO: ledger selection */, batch.Key, &blockchain.BatchPin{
			Namespace:       batch.Namespace,
			TransactionID:   batch.Payload.TX.ID,
			BatchID:         batch.ID,
			BatchHash:       batch.Hash,
			BatchPayloadRef: batch.PayloadRef,
			Contexts:        data.Contexts,
		})

	default:
		return false, i18n.NewError(ctx, i18n.MsgOperationNotSupported)
	}
}

func opBatchPin(op *fftypes.Operation, batch *fftypes.Batch, contexts []*fftypes.Bytes32) *fftypes.PreparedOperation {
	return &fftypes.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
		Data: batchPinData{Batch: batch, Contexts: contexts},
	}
}
