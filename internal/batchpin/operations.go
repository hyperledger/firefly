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

	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/i18n"
)

type batchPinData struct {
	Batch      *fftypes.BatchPersisted `json:"batch"`
	Contexts   []*fftypes.Bytes32      `json:"contexts"`
	PayloadRef string                  `json:"payloadRef"`
}

func addBatchPinInputs(op *fftypes.Operation, batchID *fftypes.UUID, contexts []*fftypes.Bytes32, payloadRef string) {
	contextStr := make([]string, len(contexts))
	for i, c := range contexts {
		contextStr[i] = c.String()
	}
	op.Input = fftypes.JSONObject{
		"batch":      batchID.String(),
		"contexts":   contextStr,
		"payloadRef": payloadRef,
	}
}

func retrieveBatchPinInputs(ctx context.Context, op *fftypes.Operation) (batchID *fftypes.UUID, contexts []*fftypes.Bytes32, payloadRef string, err error) {
	batchID, err = fftypes.ParseUUID(ctx, op.Input.GetString("batch"))
	if err != nil {
		return nil, nil, "", err
	}
	contextStr := op.Input.GetStringArray("contexts")
	contexts = make([]*fftypes.Bytes32, len(contextStr))
	for i, c := range contextStr {
		contexts[i], err = fftypes.ParseBytes32(ctx, c)
		if err != nil {
			return nil, nil, "", err
		}
	}
	payloadRef = op.Input.GetString("payloadRef")
	return batchID, contexts, payloadRef, nil
}

func (bp *batchPinSubmitter) PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error) {
	switch op.Type {
	case fftypes.OpTypeBlockchainPinBatch:
		batchID, contexts, payloadRef, err := retrieveBatchPinInputs(ctx, op)
		if err != nil {
			return nil, err
		}
		batch, err := bp.database.GetBatchByID(ctx, batchID)
		if err != nil {
			return nil, err
		} else if batch == nil {
			return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
		}
		return opBatchPin(op, batch, contexts, payloadRef), nil

	default:
		return nil, i18n.NewError(ctx, coremsgs.MsgOperationNotSupported, op.Type)
	}
}

func (bp *batchPinSubmitter) RunOperation(ctx context.Context, op *fftypes.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error) {
	switch data := op.Data.(type) {
	case batchPinData:
		batch := data.Batch
		return nil, false, bp.blockchain.SubmitBatchPin(ctx, op.ID, nil /* TODO: ledger selection */, batch.Key, &blockchain.BatchPin{
			Namespace:       batch.Namespace,
			TransactionID:   batch.TX.ID,
			BatchID:         batch.ID,
			BatchHash:       batch.Hash,
			BatchPayloadRef: data.PayloadRef,
			Contexts:        data.Contexts,
		})

	default:
		return nil, false, i18n.NewError(ctx, coremsgs.MsgOperationDataIncorrect, op.Data)
	}
}

func (bp *batchPinSubmitter) OnOperationUpdate(ctx context.Context, op *fftypes.Operation, update *operations.OperationUpdate) error {
	return nil
}

func opBatchPin(op *fftypes.Operation, batch *fftypes.BatchPersisted, contexts []*fftypes.Bytes32, payloadRef string) *fftypes.PreparedOperation {
	return &fftypes.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
		Data: batchPinData{Batch: batch, Contexts: contexts, PayloadRef: payloadRef},
	}
}
