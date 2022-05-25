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

package contracts

import (
	"context"
	"encoding/json"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/pkg/core"
)

type blockchainInvokeData struct {
	Request *core.ContractCallRequest `json:"request"`
}

func addBlockchainInvokeInputs(op *core.Operation, req *core.ContractCallRequest) (err error) {
	var reqJSON []byte
	if reqJSON, err = json.Marshal(req); err == nil {
		err = json.Unmarshal(reqJSON, &op.Input)
	}
	return err
}

func retrieveBlockchainInvokeInputs(ctx context.Context, op *core.Operation) (*core.ContractCallRequest, error) {
	var req core.ContractCallRequest
	s := op.Input.String()
	if err := json.Unmarshal([]byte(s), &req); err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgJSONObjectParseFailed, s)
	}
	return &req, nil
}

func (cm *contractManager) PrepareOperation(ctx context.Context, op *core.Operation) (*core.PreparedOperation, error) {
	switch op.Type {
	case core.OpTypeBlockchainInvoke:
		req, err := retrieveBlockchainInvokeInputs(ctx, op)
		if err != nil {
			return nil, err
		}
		return opBlockchainInvoke(op, req), nil

	default:
		return nil, i18n.NewError(ctx, coremsgs.MsgOperationNotSupported, op.Type)
	}
}

func (cm *contractManager) RunOperation(ctx context.Context, op *core.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error) {
	switch data := op.Data.(type) {
	case blockchainInvokeData:
		req := data.Request
		return nil, false, cm.blockchain.InvokeContract(ctx, op.ID, req.Key, req.Location, req.Method, req.Input)

	default:
		return nil, false, i18n.NewError(ctx, coremsgs.MsgOperationDataIncorrect, op.Data)
	}
}

func (cm *contractManager) OnOperationUpdate(ctx context.Context, op *core.Operation, update *operations.OperationUpdate) error {
	// Special handling for OpTypeBlockchainInvoke, which writes an event when it succeeds or fails
	if op.Type == core.OpTypeBlockchainInvoke {
		if update.Status == core.OpStatusSucceeded {
			event := core.NewEvent(core.EventTypeBlockchainInvokeOpSucceeded, op.Namespace, op.ID, op.Transaction, "")
			if err := cm.database.InsertEvent(ctx, event); err != nil {
				return err
			}
		}
		if update.Status == core.OpStatusFailed {
			event := core.NewEvent(core.EventTypeBlockchainInvokeOpFailed, op.Namespace, op.ID, op.Transaction, "")
			if err := cm.database.InsertEvent(ctx, event); err != nil {
				return err
			}
		}
	}
	return nil
}

func opBlockchainInvoke(op *core.Operation, req *core.ContractCallRequest) *core.PreparedOperation {
	return &core.PreparedOperation{
		ID:        op.ID,
		Namespace: op.Namespace,
		Type:      op.Type,
		Data:      blockchainInvokeData{Request: req},
	}
}
