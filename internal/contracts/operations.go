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

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type blockchainInvokeData struct {
	Request *fftypes.ContractCallRequest `json:"request"`
}

func addBlockchainInvokeInputs(op *fftypes.Operation, req *fftypes.ContractCallRequest) (err error) {
	var reqJSON []byte
	if reqJSON, err = json.Marshal(req); err == nil {
		err = json.Unmarshal(reqJSON, &op.Input)
	}
	return err
}

func retrieveBlockchainInvokeInputs(ctx context.Context, op *fftypes.Operation) (*fftypes.ContractCallRequest, error) {
	var req fftypes.ContractCallRequest
	s := op.Input.String()
	if err := json.Unmarshal([]byte(s), &req); err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgJSONObjectParseFailed, s)
	}
	return &req, nil
}

func (cm *contractManager) PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error) {
	switch op.Type {
	case fftypes.OpTypeBlockchainInvoke:
		req, err := retrieveBlockchainInvokeInputs(ctx, op)
		if err != nil {
			return nil, err
		}
		return opBlockchainInvoke(op, req), nil

	default:
		return nil, i18n.NewError(ctx, i18n.MsgOperationNotSupported)
	}
}

func (cm *contractManager) RunOperation(ctx context.Context, op *fftypes.PreparedOperation) (complete bool, err error) {
	switch data := op.Data.(type) {
	case blockchainInvokeData:
		req := data.Request
		return false, cm.blockchain.InvokeContract(ctx, op.ID, req.Key, req.Location, req.Method, req.Input)

	default:
		return false, i18n.NewError(ctx, i18n.MsgOperationNotSupported)
	}
}

func opBlockchainInvoke(op *fftypes.Operation, req *fftypes.ContractCallRequest) *fftypes.PreparedOperation {
	return &fftypes.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
		Data: blockchainInvokeData{Request: req},
	}
}
