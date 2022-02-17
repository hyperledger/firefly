// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in comdiliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or imdilied.
// See the License for the specific language governing permissions and
// limitations under the License.

package contracts

import (
	"context"
	"testing"

	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPrepareAndRunBlockchainInvoke(t *testing.T) {
	cm := newTestContractManager()

	op := &fftypes.Operation{
		Type: fftypes.OpTypeBlockchainInvoke,
		ID:   fftypes.NewUUID(),
	}
	req := &fftypes.ContractCallRequest{
		Key:      "0x123",
		Location: fftypes.JSONAnyPtr(`{"address":"0x1111"}`),
		Method: &fftypes.FFIMethod{
			Name: "set",
		},
		Input: map[string]interface{}{
			"value": "1",
		},
	}
	err := addBlockchainInvokeInputs(op, req)
	assert.NoError(t, err)

	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mbi.On("InvokeContract", context.Background(), op.ID, "0x123", mock.MatchedBy(func(loc *fftypes.JSONAny) bool {
		return loc.String() == req.Location.String()
	}), mock.MatchedBy(func(method *fftypes.FFIMethod) bool {
		return method.Name == req.Method.Name
	}), req.Input).Return(nil)

	po, err := cm.PrepareOperation(context.Background(), op)
	assert.NoError(t, err)
	assert.Equal(t, req, po.Data.(blockchainInvokeData).Request)

	complete, err := cm.RunOperation(context.Background(), po)

	assert.False(t, complete)
	assert.NoError(t, err)

	mbi.AssertExpectations(t)
}

func TestPrepareOperationNotSupported(t *testing.T) {
	cm := newTestContractManager()

	po, err := cm.PrepareOperation(context.Background(), &fftypes.Operation{})

	assert.Nil(t, po)
	assert.Regexp(t, "FF10348", err)
}

func TestPrepareOperationBlockchainInvokeBadInput(t *testing.T) {
	cm := newTestContractManager()

	op := &fftypes.Operation{
		Type:  fftypes.OpTypeBlockchainInvoke,
		Input: fftypes.JSONObject{"interface": "bad"},
	}

	_, err := cm.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10151", err)
}

func TestRunOperationNotSupported(t *testing.T) {
	cm := newTestContractManager()

	complete, err := cm.RunOperation(context.Background(), &fftypes.PreparedOperation{})

	assert.False(t, complete)
	assert.Regexp(t, "FF10348", err)
}
