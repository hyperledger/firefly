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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPrepareAndRunBlockchainInvoke(t *testing.T) {
	cm := newTestContractManager()

	op := &core.Operation{
		Type:      core.OpTypeBlockchainInvoke,
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}
	req := &core.ContractCallRequest{
		Key:      "0x123",
		Location: fftypes.JSONAnyPtr(`{"address":"0x1111"}`),
		Method: &fftypes.FFIMethod{
			Name: "set",
		},
		Input: map[string]interface{}{
			"value": "1",
		},
	}
	err := addBlockchainReqInputs(op, req)
	assert.NoError(t, err)

	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mbi.On("InvokeContract", context.Background(), "ns1:"+op.ID.String(), "0x123", mock.MatchedBy(func(loc *fftypes.JSONAny) bool {
		return loc.String() == req.Location.String()
	}), mock.MatchedBy(func(method *fftypes.FFIMethod) bool {
		return method.Name == req.Method.Name
	}), req.Input, req.Options).Return(nil)

	po, err := cm.PrepareOperation(context.Background(), op)
	assert.NoError(t, err)
	assert.Equal(t, req, po.Data.(blockchainInvokeData).Request)

	_, complete, err := cm.RunOperation(context.Background(), po)

	assert.False(t, complete)
	assert.NoError(t, err)

	mbi.AssertExpectations(t)
}

func TestPrepareAndRunBlockchainContractDeploy(t *testing.T) {
	cm := newTestContractManager()

	op := &core.Operation{
		Type:      core.OpTypeBlockchainContractDeploy,
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}
	signingKey := "0x2468"
	req := &core.ContractDeployRequest{
		Key:        signingKey,
		Definition: fftypes.JSONAnyPtr("[]"),
		Contract:   fftypes.JSONAnyPtr("\"0x123456\""),
		Input:      []interface{}{"one", "two", "three"},
	}
	err := addBlockchainReqInputs(op, req)
	assert.NoError(t, err)

	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mbi.On("DeployContract", context.Background(), "ns1:"+op.ID.String(), signingKey, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	po, err := cm.PrepareOperation(context.Background(), op)
	assert.NoError(t, err)
	assert.Equal(t, req, po.Data.(blockchainContractDeployData).Request)

	_, complete, err := cm.RunOperation(context.Background(), po)

	assert.False(t, complete)
	assert.NoError(t, err)

	mbi.AssertExpectations(t)
}

func TestPrepareOperationNotSupported(t *testing.T) {
	cm := newTestContractManager()

	po, err := cm.PrepareOperation(context.Background(), &core.Operation{})

	assert.Nil(t, po)
	assert.Regexp(t, "FF10371", err)
}

func TestPrepareOperationBlockchainDeployBadInput(t *testing.T) {
	cm := newTestContractManager()

	op := &core.Operation{
		Type:  core.OpTypeBlockchainContractDeploy,
		Input: fftypes.JSONObject{"input": "bad"},
	}

	_, err := cm.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF00127", err)
}

func TestPrepareOperationBlockchainInvokeBadInput(t *testing.T) {
	cm := newTestContractManager()

	op := &core.Operation{
		Type:  core.OpTypeBlockchainInvoke,
		Input: fftypes.JSONObject{"interface": "bad"},
	}

	_, err := cm.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF00127", err)
}

func TestRunOperationNotSupported(t *testing.T) {
	cm := newTestContractManager()

	_, complete, err := cm.RunOperation(context.Background(), &core.PreparedOperation{})

	assert.False(t, complete)
	assert.Regexp(t, "FF10378", err)
}

func TestOperationUpdate(t *testing.T) {
	cm := newTestContractManager()

	op := &core.Operation{}

	err := cm.OnOperationUpdate(context.Background(), op, nil)
	assert.NoError(t, err)
}

func TestOperationUpdateDeploySucceed(t *testing.T) {
	cm := newTestContractManager()

	op := &core.Operation{
		ID:   fftypes.NewUUID(),
		Type: core.OpTypeBlockchainContractDeploy,
	}
	update := &core.OperationUpdate{
		Status: core.OpStatusSucceeded,
	}

	mdi := cm.database.(*databasemocks.Plugin)
	mdi.On("InsertEvent", context.Background(), mock.MatchedBy(func(event *core.Event) bool {
		return event.Type == core.EventTypeBlockchainContractDeployOpSucceeded && *event.Reference == *op.ID
	})).Return(fmt.Errorf("pop"))

	err := cm.OnOperationUpdate(context.Background(), op, update)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestOperationUpdateDeployFail(t *testing.T) {
	cm := newTestContractManager()

	op := &core.Operation{
		ID:   fftypes.NewUUID(),
		Type: core.OpTypeBlockchainContractDeploy,
	}
	update := &core.OperationUpdate{
		Status: core.OpStatusFailed,
	}

	mdi := cm.database.(*databasemocks.Plugin)
	mdi.On("InsertEvent", context.Background(), mock.MatchedBy(func(event *core.Event) bool {
		return event.Type == core.EventTypeBlockchainContractDeployOpFailed && *event.Reference == *op.ID
	})).Return(fmt.Errorf("pop"))

	err := cm.OnOperationUpdate(context.Background(), op, update)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestOperationUpdateInvokeSucceed(t *testing.T) {
	cm := newTestContractManager()

	op := &core.Operation{
		ID:   fftypes.NewUUID(),
		Type: core.OpTypeBlockchainInvoke,
	}
	update := &core.OperationUpdate{
		Status: core.OpStatusSucceeded,
	}

	mdi := cm.database.(*databasemocks.Plugin)
	mdi.On("InsertEvent", context.Background(), mock.MatchedBy(func(event *core.Event) bool {
		return event.Type == core.EventTypeBlockchainInvokeOpSucceeded && *event.Reference == *op.ID
	})).Return(fmt.Errorf("pop"))

	err := cm.OnOperationUpdate(context.Background(), op, update)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestOperationUpdateInvokeFail(t *testing.T) {
	cm := newTestContractManager()

	op := &core.Operation{
		ID:   fftypes.NewUUID(),
		Type: core.OpTypeBlockchainInvoke,
	}
	update := &core.OperationUpdate{
		Status: core.OpStatusFailed,
	}

	mdi := cm.database.(*databasemocks.Plugin)
	mdi.On("InsertEvent", context.Background(), mock.MatchedBy(func(event *core.Event) bool {
		return event.Type == core.EventTypeBlockchainInvokeOpFailed && *event.Reference == *op.ID
	})).Return(fmt.Errorf("pop"))

	err := cm.OnOperationUpdate(context.Background(), op, update)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}
