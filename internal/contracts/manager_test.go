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

	"github.com/hyperledger/firefly/internal/blockchain/ethereum"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestContractManager() *contractManager {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mbm := &broadcastmocks.Manager{}
	mim := &identitymanagermocks.Manager{}
	mbi := &blockchainmocks.Plugin{}
	mom := &operationmocks.Manager{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	msa := &syncasyncmocks.Bridge{}
	mbi.On("GetFFIParamValidator", mock.Anything).Return(nil, nil)
	mom.On("RegisterHandler", mock.Anything, mock.Anything, mock.Anything)

	mbi.On("Name").Return("mockblockchain").Maybe()

	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}
	cm, _ := NewContractManager(context.Background(), mdi, mbm, mim, mbi, mom, txHelper, msa)
	cm.(*contractManager).txHelper = &txcommonmocks.Helper{}
	return cm.(*contractManager)
}

func TestNewContractManagerFail(t *testing.T) {
	_, err := NewContractManager(context.Background(), nil, nil, nil, nil, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestName(t *testing.T) {
	cm := newTestContractManager()
	assert.Equal(t, "ContractManager", cm.Name())
}

func TestNewContractManagerFFISchemaLoaderFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mbm := &broadcastmocks.Manager{}
	mim := &identitymanagermocks.Manager{}
	mbi := &blockchainmocks.Plugin{}
	mom := &operationmocks.Manager{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	msa := &syncasyncmocks.Bridge{}
	mbi.On("GetFFIParamValidator", mock.Anything).Return(nil, fmt.Errorf("pop"))
	_, err := NewContractManager(context.Background(), mdi, mbm, mim, mbi, mom, txHelper, msa)
	assert.Regexp(t, "pop", err)
}

func TestNewContractManagerFFISchemaLoader(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mbm := &broadcastmocks.Manager{}
	mim := &identitymanagermocks.Manager{}
	mbi := &blockchainmocks.Plugin{}
	mom := &operationmocks.Manager{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	msa := &syncasyncmocks.Bridge{}
	mbi.On("GetFFIParamValidator", mock.Anything).Return(&ethereum.FFIParamValidator{}, nil)
	mom.On("RegisterHandler", mock.Anything, mock.Anything, mock.Anything)
	_, err := NewContractManager(context.Background(), mdi, mbm, mim, mbi, mom, txHelper, msa)
	assert.NoError(t, err)
}

func TestBroadcastFFI(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	mbm := cm.broadcast.(*broadcastmocks.Manager)

	mdb.On("GetFFI", mock.Anything, "ns1", "test", "1.0.0").Return(nil, nil)
	mim.On("GetOrgKey", mock.Anything).Return("key", nil)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.FFI"), fftypes.SystemTagDefineFFI, false).Return(msg, nil)
	ffi := &fftypes.FFI{
		Name:    "test",
		Version: "1.0.0",
		ID:      fftypes.NewUUID(),
		Methods: []*fftypes.FFIMethod{
			{
				Name: "sum",
			},
		},
		Events: []*fftypes.FFIEvent{
			{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "changed",
				},
			},
		},
	}
	_, err := cm.BroadcastFFI(context.Background(), "ns1", ffi, false)
	assert.NoError(t, err)
}

func TestBroadcastFFIInvalid(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mbm := cm.broadcast.(*broadcastmocks.Manager)

	mdb.On("GetFFI", mock.Anything, "ns1", "test", "1.0.0").Return(nil, nil)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.FFI"), fftypes.SystemTagDefineFFI, false).Return(msg, nil)
	ffi := &fftypes.FFI{
		Name:    "test",
		Version: "1.0.0",
		ID:      fftypes.NewUUID(),
		Methods: []*fftypes.FFIMethod{
			{
				Name: "sum",
				Params: []*fftypes.FFIParam{
					{
						Name:   "x",
						Schema: fftypes.JSONAnyPtr(`{"type": "number"}`),
					},
				},
			},
		},
	}
	_, err := cm.BroadcastFFI(context.Background(), "ns1", ffi, false)
	assert.Regexp(t, "does not validate", err)
}

func TestBroadcastFFIExists(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mbm := cm.broadcast.(*broadcastmocks.Manager)

	mdb.On("GetFFI", mock.Anything, "ns1", "test", "1.0.0").Return(&fftypes.FFI{}, nil)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.FFI"), fftypes.SystemTagDefineFFI, false).Return(msg, nil)
	ffi := &fftypes.FFI{
		Name:    "test",
		Version: "1.0.0",
		ID:      fftypes.NewUUID(),
	}
	_, err := cm.BroadcastFFI(context.Background(), "ns1", ffi, false)
	assert.Regexp(t, "FF10302", err)
}

func TestBroadcastFFIFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mbm := cm.broadcast.(*broadcastmocks.Manager)
	mim := cm.identity.(*identitymanagermocks.Manager)

	mdb.On("GetFFI", mock.Anything, "ns1", "test", "1.0.0").Return(nil, nil)
	mim.On("GetOrgKey", mock.Anything).Return("key", nil)

	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.FFI"), fftypes.SystemTagDefineFFI, false).Return(nil, fmt.Errorf("pop"))
	ffi := &fftypes.FFI{
		Name:    "test",
		Version: "1.0.0",
		ID:      fftypes.NewUUID(),
		Methods: []*fftypes.FFIMethod{
			{
				Name: "sum",
			},
		},
	}
	_, err := cm.BroadcastFFI(context.Background(), "ns1", ffi, false)
	assert.Regexp(t, "pop", err)
}

func TestValidateInvokeContractRequest(t *testing.T) {
	cm := newTestContractManager()
	req := &fftypes.ContractCallRequest{
		Type: fftypes.CallTypeInvoke,
		Method: &fftypes.FFIMethod{
			Name: "sum",
			Params: []*fftypes.FFIParam{
				{
					Name:   "x",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
				},
				{
					Name:   "y",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
				},
			},
			Returns: []*fftypes.FFIParam{
				{
					Name:   "z",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
				},
			},
		},
		Input: map[string]interface{}{
			"x": float64(1),
			"y": float64(2),
		},
	}
	err := cm.validateInvokeContractRequest(context.Background(), req)
	assert.NoError(t, err)
}

func TestValidateInvokeContractRequestMissingInput(t *testing.T) {
	cm := newTestContractManager()
	req := &fftypes.ContractCallRequest{
		Type: fftypes.CallTypeInvoke,
		Method: &fftypes.FFIMethod{
			Name: "sum",
			Params: []*fftypes.FFIParam{
				{
					Name:   "x",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
				},
				{
					Name:   "y",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
				},
			},
			Returns: []*fftypes.FFIParam{
				{
					Name:   "z",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
				},
			},
		},
		Input: map[string]interface{}{
			"x": float64(1),
		},
	}
	err := cm.validateInvokeContractRequest(context.Background(), req)
	assert.Regexp(t, "Missing required input argument 'y'", err)
}

func TestValidateInvokeContractRequestInputWrongType(t *testing.T) {
	cm := newTestContractManager()
	req := &fftypes.ContractCallRequest{
		Type: fftypes.CallTypeInvoke,
		Method: &fftypes.FFIMethod{
			Name: "sum",
			Params: []*fftypes.FFIParam{
				{
					Name:   "x",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
				},
				{
					Name:   "y",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
				},
			},
			Returns: []*fftypes.FFIParam{
				{
					Name:   "z",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
				},
			},
		},
		Input: map[string]interface{}{
			"x": float64(1),
			"y": "two",
		},
	}
	err := cm.validateInvokeContractRequest(context.Background(), req)
	assert.Regexp(t, "expected integer, but got string", err)
}

func TestValidateInvokeContractRequestInvalidParam(t *testing.T) {
	cm := newTestContractManager()
	req := &fftypes.ContractCallRequest{
		Type: fftypes.CallTypeInvoke,
		Method: &fftypes.FFIMethod{
			Name: "sum",
			Params: []*fftypes.FFIParam{
				{
					Name:   "x",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
				},
				{
					Name:   "y",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
				},
			},
			Returns: []*fftypes.FFIParam{
				{
					Name:   "z",
					Schema: fftypes.JSONAnyPtr(`{"type": "number"}`),
				},
			},
		},
		Input: map[string]interface{}{
			"x": float64(1),
			"y": float64(2),
		},
	}

	err := cm.validateInvokeContractRequest(context.Background(), req)
	assert.Regexp(t, "does not validate", err)
}

func TestValidateFFI(t *testing.T) {
	cm := newTestContractManager()
	ffi := &fftypes.FFI{
		Name:      "math",
		Version:   "1.0.0",
		Namespace: "default",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "sum",
				Params: []*fftypes.FFIParam{
					{
						Name:   "x",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
					},
					{
						Name:   "y",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
					},
				},
				Returns: []*fftypes.FFIParam{
					{
						Name:   "z",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
					},
				},
			},
			{
				Name:        "sum",
				Description: "Override of sum method with different args",
				Params:      []*fftypes.FFIParam{},
				Returns:     []*fftypes.FFIParam{},
			},
		},
		Events: []*fftypes.FFIEvent{
			{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "sum",
					Params: []*fftypes.FFIParam{
						{
							Name:   "z",
							Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
						},
					},
				},
			},
			{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name:        "sum",
					Description: "Override of event with different params",
					Params:      []*fftypes.FFIParam{},
				},
			},
		},
	}

	err := cm.ValidateFFIAndSetPathnames(context.Background(), ffi)
	assert.NoError(t, err)

	assert.Equal(t, "sum", ffi.Methods[0].Pathname)
	assert.Equal(t, "sum_1", ffi.Methods[1].Pathname)
	assert.Equal(t, "sum", ffi.Events[0].Pathname)
	assert.Equal(t, "sum_1", ffi.Events[1].Pathname)
}

func TestValidateFFIFail(t *testing.T) {
	cm := newTestContractManager()
	ffi := &fftypes.FFI{
		Namespace: "default",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "sum",
				Params: []*fftypes.FFIParam{
					{
						Name:   "x",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
					},
					{
						Name:   "y",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
					},
				},
				Returns: []*fftypes.FFIParam{
					{
						Name:   "z",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
					},
				},
			},
			{
				Name:        "sum",
				Description: "Override of sum method with different args",
				Params:      []*fftypes.FFIParam{},
				Returns:     []*fftypes.FFIParam{},
			},
		},
		Events: []*fftypes.FFIEvent{
			{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "sum",
					Params: []*fftypes.FFIParam{
						{
							Name:   "z",
							Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
						},
					},
				},
			},
			{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name:        "sum",
					Description: "Override of event with different params",
					Params:      []*fftypes.FFIParam{},
				},
			},
		},
	}

	err := cm.ValidateFFIAndSetPathnames(context.Background(), ffi)
	assert.Regexp(t, "FF00140", err)
}

func TestValidateFFIBadMethod(t *testing.T) {
	cm := newTestContractManager()
	ffi := &fftypes.FFI{
		Name:      "math",
		Version:   "1.0.0",
		Namespace: "default",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "",
				Params: []*fftypes.FFIParam{
					{
						Name:   "x",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
					},
					{
						Name:   "y",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
					},
				},
				Returns: []*fftypes.FFIParam{
					{
						Name:   "z",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
					},
				},
			},
		},
		Events: []*fftypes.FFIEvent{
			{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "sum",
					Params: []*fftypes.FFIParam{
						{
							Name:   "z",
							Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
						},
					},
				},
			},
		},
	}

	err := cm.ValidateFFIAndSetPathnames(context.Background(), ffi)
	assert.Regexp(t, "FF10320", err)
}

func TestValidateFFIBadEventParam(t *testing.T) {
	cm := newTestContractManager()
	ffi := &fftypes.FFI{
		Name:      "math",
		Version:   "1.0.0",
		Namespace: "default",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "sum",
				Params: []*fftypes.FFIParam{
					{
						Name:   "x",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
					},
					{
						Name:   "y",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
					},
				},
				Returns: []*fftypes.FFIParam{
					{
						Name:   "z",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
					},
				},
			},
		},
		Events: []*fftypes.FFIEvent{
			{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "",
					Params: []*fftypes.FFIParam{
						{
							Name:   "z",
							Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
						},
					},
				},
			},
		},
	}

	err := cm.ValidateFFIAndSetPathnames(context.Background(), ffi)
	assert.Regexp(t, "FF10319", err)
}

func TestAddContractListenerInline(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Event: &fftypes.FFISerializedEvent{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "changed",
					Params: fftypes.FFIParams{
						{
							Name:   "value",
							Schema: fftypes.JSONAnyPtr(`{"type": "integer"}`),
						},
					},
				},
			},
			Options: &fftypes.ContractListenerOptions{},
			Topic:   "test-topic",
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), sub.Location).Return(sub.Location, nil)
	mbi.On("GenerateEventSignature", context.Background(), mock.Anything).Return("changed")
	mdi.On("GetContractListeners", context.Background(), mock.Anything).Return(nil, nil, nil)
	mbi.On("AddContractListener", context.Background(), sub).Return(nil)
	mdi.On("UpsertContractListener", context.Background(), &sub.ContractListener).Return(nil)

	result, err := cm.AddContractListener(context.Background(), "ns", sub)
	assert.NoError(t, err)
	assert.NotNil(t, result.ID)
	assert.NotNil(t, result.Event)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractListenerByEventPath(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	interfaceID := fftypes.NewUUID()

	event := &fftypes.FFIEvent{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		FFIEventDefinition: fftypes.FFIEventDefinition{
			Name: "changed",
			Params: fftypes.FFIParams{
				{
					Name:   "value",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer"}`),
				},
			},
		},
	}

	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{
			Interface: &fftypes.FFIReference{
				ID: interfaceID,
			},
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Topic: "test-topic",
		},
		EventPath: "changed",
	}

	mbi.On("NormalizeContractLocation", context.Background(), sub.Location).Return(sub.Location, nil)
	mbi.On("GenerateEventSignature", context.Background(), mock.Anything).Return("changed")
	mdi.On("GetContractListeners", context.Background(), mock.Anything).Return(nil, nil, nil)
	mbi.On("AddContractListener", context.Background(), sub).Return(nil)
	mdi.On("GetFFIByID", context.Background(), interfaceID).Return(&fftypes.FFI{}, nil)
	mdi.On("GetFFIEvent", context.Background(), "ns1", interfaceID, sub.EventPath).Return(event, nil)
	mdi.On("UpsertContractListener", context.Background(), &sub.ContractListener).Return(nil)

	result, err := cm.AddContractListener(context.Background(), "ns1", sub)
	assert.NoError(t, err)
	assert.NotNil(t, result.ID)
	assert.NotNil(t, result.Event)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractListenerBadLocation(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{
			Interface: &fftypes.FFIReference{
				ID: fftypes.NewUUID(),
			},
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Topic: "test-topic",
		},
		EventPath: "changed",
	}

	mbi.On("NormalizeContractLocation", context.Background(), sub.Location).Return(nil, fmt.Errorf("pop"))

	_, err := cm.AddContractListener(context.Background(), "ns1", sub)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
}

func TestAddContractListenerFFILookupFail(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	interfaceID := fftypes.NewUUID()

	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{
			Interface: &fftypes.FFIReference{
				ID: interfaceID,
			},
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Topic: "test-topic",
		},
		EventPath: "changed",
	}

	mbi.On("NormalizeContractLocation", context.Background(), sub.Location).Return(sub.Location, nil)
	mdi.On("GetFFIByID", context.Background(), interfaceID).Return(nil, fmt.Errorf("pop"))

	_, err := cm.AddContractListener(context.Background(), "ns1", sub)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractListenerEventLookupFail(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	interfaceID := fftypes.NewUUID()

	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{
			Interface: &fftypes.FFIReference{
				ID: interfaceID,
			},
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Topic: "test-topic",
		},
		EventPath: "changed",
	}

	mbi.On("NormalizeContractLocation", context.Background(), sub.Location).Return(sub.Location, nil)
	mdi.On("GetFFIByID", context.Background(), interfaceID).Return(&fftypes.FFI{}, nil)
	mdi.On("GetFFIEvent", context.Background(), "ns1", interfaceID, sub.EventPath).Return(nil, fmt.Errorf("pop"))

	_, err := cm.AddContractListener(context.Background(), "ns1", sub)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractListenerEventLookupNotFound(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	interfaceID := fftypes.NewUUID()

	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{
			Interface: &fftypes.FFIReference{
				ID: interfaceID,
			},
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Topic: "test-topic",
		},
		EventPath: "changed",
	}

	mbi.On("NormalizeContractLocation", context.Background(), sub.Location).Return(sub.Location, nil)
	mdi.On("GetFFIByID", context.Background(), interfaceID).Return(&fftypes.FFI{}, nil)
	mdi.On("GetFFIEvent", context.Background(), "ns1", interfaceID, sub.EventPath).Return(nil, nil)

	_, err := cm.AddContractListener(context.Background(), "ns1", sub)
	assert.Regexp(t, "FF10370", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractListenerMissingEventOrID(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Topic: "test-topic",
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), sub.Location).Return(sub.Location, nil)

	_, err := cm.AddContractListener(context.Background(), "ns2", sub)
	assert.Regexp(t, "FF10317", err)

	mbi.AssertExpectations(t)
}

func TestAddContractListenerBadNamespace(t *testing.T) {
	cm := newTestContractManager()
	sub := &fftypes.ContractListenerInput{}

	_, err := cm.AddContractListener(context.Background(), "!bad", sub)
	assert.Regexp(t, "FF00140.*'namespace'", err)
}

func TestAddContractListenerBadName(t *testing.T) {
	cm := newTestContractManager()
	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{
			Name: "!bad",
		},
	}

	_, err := cm.AddContractListener(context.Background(), "ns", sub)
	assert.Regexp(t, "FF00140.*'name'", err)
}

func TestAddContractListenerMissingTopic(t *testing.T) {
	cm := newTestContractManager()
	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{},
	}

	_, err := cm.AddContractListener(context.Background(), "ns", sub)
	assert.Regexp(t, "FF00140.*'topic'", err)
}

func TestAddContractListenerNameConflict(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{
			Name: "sub1",
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Topic: "test-topic",
		},
		EventPath: "changed",
	}

	mbi.On("NormalizeContractLocation", context.Background(), sub.Location).Return(sub.Location, nil)
	mdi.On("GetContractListener", context.Background(), "ns", "sub1").Return(&fftypes.ContractListener{}, nil)

	_, err := cm.AddContractListener(context.Background(), "ns", sub)
	assert.Regexp(t, "FF10312", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractListenerNameError(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{
			Name: "sub1",
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Topic: "test-topic",
		},
		EventPath: "changed",
	}

	mbi.On("NormalizeContractLocation", context.Background(), sub.Location).Return(sub.Location, nil)
	mdi.On("GetContractListener", context.Background(), "ns", "sub1").Return(nil, fmt.Errorf("pop"))

	_, err := cm.AddContractListener(context.Background(), "ns", sub)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractListenerTopicConflict(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Event: &fftypes.FFISerializedEvent{},
			Topic: "test-topic",
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), sub.Location).Return(sub.Location, nil)
	mbi.On("GenerateEventSignature", context.Background(), mock.Anything).Return("changed")
	mdi.On("GetContractListeners", context.Background(), mock.Anything).Return([]*fftypes.ContractListener{{}}, nil, nil)

	_, err := cm.AddContractListener(context.Background(), "ns", sub)
	assert.Regexp(t, "FF10383", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractListenerTopicError(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Event: &fftypes.FFISerializedEvent{},
			Topic: "test-topic",
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), sub.Location).Return(sub.Location, nil)
	mbi.On("GenerateEventSignature", context.Background(), mock.Anything).Return("changed")
	mdi.On("GetContractListeners", context.Background(), mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	_, err := cm.AddContractListener(context.Background(), "ns", sub)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractListenerValidateFail(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Event: &fftypes.FFISerializedEvent{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "changed",
					Params: fftypes.FFIParams{
						{
							Name:   "value",
							Schema: fftypes.JSONAnyPtr(`{"type": "number"}`),
						},
					},
				},
			},
			Topic: "test-topic",
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), sub.Location).Return(sub.Location, nil)
	mbi.On("GenerateEventSignature", context.Background(), mock.Anything).Return("changed")
	mdi.On("GetContractListeners", context.Background(), mock.Anything).Return(nil, nil, nil)

	_, err := cm.AddContractListener(context.Background(), "ns", sub)
	assert.Regexp(t, "does not validate", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractListenerBlockchainFail(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Event: &fftypes.FFISerializedEvent{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "changed",
					Params: fftypes.FFIParams{
						{
							Name:   "value",
							Schema: fftypes.JSONAnyPtr(`{"type": "integer"}`),
						},
					},
				},
			},
			Topic: "test-topic",
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), sub.Location).Return(sub.Location, nil)
	mbi.On("GenerateEventSignature", context.Background(), mock.Anything).Return("changed")
	mdi.On("GetContractListeners", context.Background(), mock.Anything).Return(nil, nil, nil)
	mbi.On("AddContractListener", context.Background(), sub).Return(fmt.Errorf("pop"))

	_, err := cm.AddContractListener(context.Background(), "ns", sub)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractListenerUpsertSubFail(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractListenerInput{
		ContractListener: fftypes.ContractListener{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Event: &fftypes.FFISerializedEvent{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "changed",
					Params: fftypes.FFIParams{
						{
							Name:   "value",
							Schema: fftypes.JSONAnyPtr(`{"type": "integer"}`),
						},
					},
				},
			},
			Topic: "test-topic",
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), sub.Location).Return(sub.Location, nil)
	mbi.On("GenerateEventSignature", context.Background(), mock.Anything).Return("changed")
	mdi.On("GetContractListeners", context.Background(), mock.Anything).Return(nil, nil, nil)
	mbi.On("AddContractListener", context.Background(), sub).Return(nil)
	mdi.On("UpsertContractListener", context.Background(), &sub.ContractListener).Return(fmt.Errorf("pop"))

	_, err := cm.AddContractListener(context.Background(), "ns", sub)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractAPIListener(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	interfaceID := fftypes.NewUUID()
	api := &fftypes.ContractAPI{
		Interface: &fftypes.FFIReference{
			ID: interfaceID,
		},
		Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
			"address": "0x123",
		}.String()),
	}
	listener := &fftypes.ContractListener{
		Topic: "test-topic",
	}
	event := &fftypes.FFIEvent{
		FFIEventDefinition: fftypes.FFIEventDefinition{
			Name: "changed",
		},
	}

	mdi.On("GetContractAPIByName", context.Background(), "ns", "simple").Return(api, nil)
	mbi.On("NormalizeContractLocation", context.Background(), api.Location).Return(listener.Location, nil)
	mdi.On("GetFFIByID", context.Background(), interfaceID).Return(&fftypes.FFI{}, nil)
	mdi.On("GetFFIEvent", context.Background(), "ns", interfaceID, "changed").Return(event, nil)
	mbi.On("GenerateEventSignature", context.Background(), mock.Anything).Return("changed")
	mdi.On("GetContractListeners", context.Background(), mock.Anything).Return(nil, nil, nil)
	mbi.On("AddContractListener", context.Background(), mock.MatchedBy(func(l *fftypes.ContractListenerInput) bool {
		return *l.Interface.ID == *interfaceID && l.EventPath == "changed" && l.Topic == "test-topic"
	})).Return(nil)
	mdi.On("UpsertContractListener", context.Background(), mock.MatchedBy(func(l *fftypes.ContractListener) bool {
		return *l.Interface.ID == *interfaceID && l.Event.Name == "changed" && l.Topic == "test-topic"
	})).Return(nil)

	_, err := cm.AddContractAPIListener(context.Background(), "ns", "simple", "changed", listener)
	assert.NoError(t, err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractAPIListenerNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	listener := &fftypes.ContractListener{
		Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
			"address": "0x123",
		}.String()),
		Topic: "test-topic",
	}

	mdi.On("GetContractAPIByName", context.Background(), "ns", "simple").Return(nil, nil)

	_, err := cm.AddContractAPIListener(context.Background(), "ns", "simple", "changed", listener)
	assert.Regexp(t, "FF10109", err)

	mdi.AssertExpectations(t)
}

func TestAddContractAPIListenerFail(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	listener := &fftypes.ContractListener{
		Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
			"address": "0x123",
		}.String()),
		Topic: "test-topic",
	}

	mdi.On("GetContractAPIByName", context.Background(), "ns", "simple").Return(nil, fmt.Errorf("pop"))

	_, err := cm.AddContractAPIListener(context.Background(), "ns", "simple", "changed", listener)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestGetFFI(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mdb.On("GetFFI", mock.Anything, "ns1", "ffi", "v1.0.0").Return(&fftypes.FFI{}, nil)
	_, err := cm.GetFFI(context.Background(), "ns1", "ffi", "v1.0.0")
	assert.NoError(t, err)
}

func TestGetFFIWithChildren(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	cid := fftypes.NewUUID()
	mdb.On("GetFFI", mock.Anything, "ns1", "ffi", "v1.0.0").Return(&fftypes.FFI{ID: cid}, nil)
	mdb.On("GetFFIMethods", mock.Anything, mock.Anything).Return([]*fftypes.FFIMethod{
		{ID: fftypes.NewUUID(), Name: "method1"},
	}, nil, nil)
	mdb.On("GetFFIEvents", mock.Anything, mock.Anything).Return([]*fftypes.FFIEvent{
		{ID: fftypes.NewUUID(), FFIEventDefinition: fftypes.FFIEventDefinition{Name: "event1"}},
	}, nil, nil)
	mbi.On("GenerateEventSignature", mock.Anything, mock.MatchedBy(func(ev *fftypes.FFIEventDefinition) bool {
		return ev.Name == "event1"
	})).Return("event1Sig")

	_, err := cm.GetFFIWithChildren(context.Background(), "ns1", "ffi", "v1.0.0")
	assert.NoError(t, err)

	mdb.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestGetFFIByID(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	cid := fftypes.NewUUID()
	mdb.On("GetFFIByID", mock.Anything, cid).Return(&fftypes.FFI{}, nil)
	_, err := cm.GetFFIByID(context.Background(), cid)
	assert.NoError(t, err)
}

func TestGetFFIByIDWithChildren(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	cid := fftypes.NewUUID()
	mdb.On("GetFFIByID", mock.Anything, cid).Return(&fftypes.FFI{
		ID: cid,
	}, nil)
	mdb.On("GetFFIMethods", mock.Anything, mock.Anything).Return([]*fftypes.FFIMethod{
		{ID: fftypes.NewUUID(), Name: "method1"},
	}, nil, nil)
	mdb.On("GetFFIEvents", mock.Anything, mock.Anything).Return([]*fftypes.FFIEvent{
		{ID: fftypes.NewUUID(), FFIEventDefinition: fftypes.FFIEventDefinition{Name: "event1"}},
	}, nil, nil)
	mbi.On("GenerateEventSignature", mock.Anything, mock.MatchedBy(func(ev *fftypes.FFIEventDefinition) bool {
		return ev.Name == "event1"
	})).Return("event1Sig")

	ffi, err := cm.GetFFIByIDWithChildren(context.Background(), cid)

	assert.NoError(t, err)
	mdb.AssertExpectations(t)
	mbi.AssertExpectations(t)

	assert.Equal(t, "method1", ffi.Methods[0].Name)
	assert.Equal(t, "event1", ffi.Events[0].Name)
}

func TestGetFFIByIDWithChildrenEventsFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	cid := fftypes.NewUUID()
	mdb.On("GetFFIByID", mock.Anything, cid).Return(&fftypes.FFI{
		ID: cid,
	}, nil)
	mdb.On("GetFFIMethods", mock.Anything, mock.Anything).Return([]*fftypes.FFIMethod{
		{ID: fftypes.NewUUID(), Name: "method1"},
	}, nil, nil)
	mdb.On("GetFFIEvents", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	_, err := cm.GetFFIByIDWithChildren(context.Background(), cid)

	assert.EqualError(t, err, "pop")
	mdb.AssertExpectations(t)
}

func TestGetFFIByIDWithChildrenMethodsFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	cid := fftypes.NewUUID()
	mdb.On("GetFFIByID", mock.Anything, cid).Return(&fftypes.FFI{
		ID: cid,
	}, nil)
	mdb.On("GetFFIMethods", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	_, err := cm.GetFFIByIDWithChildren(context.Background(), cid)

	assert.EqualError(t, err, "pop")
	mdb.AssertExpectations(t)
}

func TestGetFFIByIDWithChildrenFFILookupFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	cid := fftypes.NewUUID()
	mdb.On("GetFFIByID", mock.Anything, cid).Return(nil, fmt.Errorf("pop"))

	_, err := cm.GetFFIByIDWithChildren(context.Background(), cid)

	assert.EqualError(t, err, "pop")
	mdb.AssertExpectations(t)
}

func TestGetFFIByIDWithChildrenFFINotFound(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	cid := fftypes.NewUUID()
	mdb.On("GetFFIByID", mock.Anything, cid).Return(nil, nil)

	ffi, err := cm.GetFFIByIDWithChildren(context.Background(), cid)

	assert.NoError(t, err)
	assert.Nil(t, ffi)
	mdb.AssertExpectations(t)
}

func TestGetFFIs(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	filter := database.FFIQueryFactory.NewFilter(context.Background()).And()
	mdb.On("GetFFIs", mock.Anything, "ns1", filter).Return([]*fftypes.FFI{}, &database.FilterResult{}, nil)
	_, _, err := cm.GetFFIs(context.Background(), "ns1", filter)
	assert.NoError(t, err)
}

func TestInvokeContract(t *testing.T) {
	cm := newTestContractManager()
	mim := cm.identity.(*identitymanagermocks.Manager)
	mdi := cm.database.(*databasemocks.Plugin)
	mth := cm.txHelper.(*txcommonmocks.Helper)
	mom := cm.operations.(*operationmocks.Manager)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			Name:    "doStuff",
			ID:      fftypes.NewUUID(),
			Params:  fftypes.FFIParams{},
			Returns: fftypes.FFIParams{},
		},
	}

	mth.On("SubmitNewTransaction", mock.Anything, "ns1", fftypes.TransactionTypeContractInvoke).Return(fftypes.NewUUID(), nil)
	mim.On("NormalizeSigningKey", mock.Anything, "", identity.KeyNormalizationBlockchainPlugin).Return("key-resolved", nil)
	mdi.On("InsertOperation", mock.Anything, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Namespace == "ns1" && op.Type == fftypes.OpTypeBlockchainInvoke && op.Plugin == "mockblockchain"
	})).Return(nil)
	mom.On("RunOperation", mock.Anything, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(blockchainInvokeData)
		return op.Type == fftypes.OpTypeBlockchainInvoke && data.Request == req
	})).Return(nil, nil)

	_, err := cm.InvokeContract(context.Background(), "ns1", req, false)

	assert.NoError(t, err)

	mth.AssertExpectations(t)
	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestInvokeContractConfirm(t *testing.T) {
	cm := newTestContractManager()
	mim := cm.identity.(*identitymanagermocks.Manager)
	mdi := cm.database.(*databasemocks.Plugin)
	mth := cm.txHelper.(*txcommonmocks.Helper)
	mom := cm.operations.(*operationmocks.Manager)
	msa := cm.syncasync.(*syncasyncmocks.Bridge)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			Name:    "doStuff",
			ID:      fftypes.NewUUID(),
			Params:  fftypes.FFIParams{},
			Returns: fftypes.FFIParams{},
		},
	}

	mth.On("SubmitNewTransaction", mock.Anything, "ns1", fftypes.TransactionTypeContractInvoke).Return(fftypes.NewUUID(), nil)
	mim.On("NormalizeSigningKey", mock.Anything, "", identity.KeyNormalizationBlockchainPlugin).Return("key-resolved", nil)
	mdi.On("InsertOperation", mock.Anything, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Namespace == "ns1" && op.Type == fftypes.OpTypeBlockchainInvoke && op.Plugin == "mockblockchain"
	})).Return(nil)
	mom.On("RunOperation", mock.Anything, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(blockchainInvokeData)
		return op.Type == fftypes.OpTypeBlockchainInvoke && data.Request == req
	})).Return(nil, nil)
	msa.On("WaitForInvokeOperation", mock.Anything, "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(context.Background())
		}).
		Return(&fftypes.Operation{}, nil)

	_, err := cm.InvokeContract(context.Background(), "ns1", req, true)

	assert.NoError(t, err)

	mth.AssertExpectations(t)
	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mom.AssertExpectations(t)
	msa.AssertExpectations(t)
}

func TestInvokeContractFail(t *testing.T) {
	cm := newTestContractManager()
	mim := cm.identity.(*identitymanagermocks.Manager)
	mdi := cm.database.(*databasemocks.Plugin)
	mth := cm.txHelper.(*txcommonmocks.Helper)
	mom := cm.operations.(*operationmocks.Manager)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			Name:    "doStuff",
			ID:      fftypes.NewUUID(),
			Params:  fftypes.FFIParams{},
			Returns: fftypes.FFIParams{},
		},
	}

	mth.On("SubmitNewTransaction", mock.Anything, "ns1", fftypes.TransactionTypeContractInvoke).Return(fftypes.NewUUID(), nil)
	mim.On("NormalizeSigningKey", mock.Anything, "", identity.KeyNormalizationBlockchainPlugin).Return("key-resolved", nil)
	mdi.On("InsertOperation", mock.Anything, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Namespace == "ns1" && op.Type == fftypes.OpTypeBlockchainInvoke && op.Plugin == "mockblockchain"
	})).Return(nil)
	mom.On("RunOperation", mock.Anything, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(blockchainInvokeData)
		return op.Type == fftypes.OpTypeBlockchainInvoke && data.Request == req
	})).Return(nil, fmt.Errorf("pop"))

	_, err := cm.InvokeContract(context.Background(), "ns1", req, false)

	assert.EqualError(t, err, "pop")

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestInvokeContractFailNormalizeSigningKey(t *testing.T) {
	cm := newTestContractManager()
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Location:  fftypes.JSONAnyPtr(""),
	}

	mim.On("NormalizeSigningKey", mock.Anything, "", identity.KeyNormalizationBlockchainPlugin).Return("", fmt.Errorf("pop"))

	_, err := cm.InvokeContract(context.Background(), "ns1", req, false)

	assert.Regexp(t, "pop", err)
}

func TestInvokeContractFailResolve(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Location:  fftypes.JSONAnyPtr(""),
	}

	mim.On("NormalizeSigningKey", mock.Anything, "", identity.KeyNormalizationBlockchainPlugin).Return("key-resolved", nil)
	mbi.On("InvokeContract", mock.Anything, mock.AnythingOfType("*fftypes.UUID"), "key-resolved", req.Location, req.Method, req.Input).Return(nil)

	_, err := cm.InvokeContract(context.Background(), "ns1", req, false)

	assert.Regexp(t, "FF10313", err)
}

func TestInvokeContractTXFail(t *testing.T) {
	cm := newTestContractManager()
	mim := cm.identity.(*identitymanagermocks.Manager)
	mth := cm.txHelper.(*txcommonmocks.Helper)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			Name:    "doStuff",
			ID:      fftypes.NewUUID(),
			Params:  fftypes.FFIParams{},
			Returns: fftypes.FFIParams{},
		},
	}

	mim.On("NormalizeSigningKey", mock.Anything, "", identity.KeyNormalizationBlockchainPlugin).Return("key-resolved", nil)
	mth.On("SubmitNewTransaction", mock.Anything, "ns1", fftypes.TransactionTypeContractInvoke).Return(nil, fmt.Errorf("pop"))

	_, err := cm.InvokeContract(context.Background(), "ns1", req, false)

	assert.EqualError(t, err, "pop")
}

func TestInvokeContractMethodNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.ContractCallRequest{
		Type:       fftypes.CallTypeInvoke,
		Interface:  fftypes.NewUUID(),
		Location:   fftypes.JSONAnyPtr(""),
		MethodPath: "set",
	}

	mim.On("NormalizeSigningKey", mock.Anything, "", identity.KeyNormalizationBlockchainPlugin).Return("key-resolved", nil)
	mdb.On("GetFFIMethod", mock.Anything, "ns1", req.Interface, req.MethodPath).Return(nil, fmt.Errorf("pop"))

	_, err := cm.InvokeContract(context.Background(), "ns1", req, false)

	assert.Regexp(t, "FF10315", err)
}

func TestInvokeContractMethodBadInput(t *testing.T) {
	cm := newTestContractManager()
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			Name: "sum",
			Params: fftypes.FFIParams{
				{
					Name:   "x",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
				},
				{
					Name:   "y",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
				},
			},
			Returns: fftypes.FFIParams{

				{
					Name:   "sum",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
				},
			},
		},
	}
	mim.On("NormalizeSigningKey", mock.Anything, "", identity.KeyNormalizationBlockchainPlugin).Return("key-resolved", nil)

	_, err := cm.InvokeContract(context.Background(), "ns1", req, false)
	assert.Regexp(t, "FF10304", err)
}

func TestQueryContract(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	mdi := cm.database.(*databasemocks.Plugin)
	mth := cm.txHelper.(*txcommonmocks.Helper)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeQuery,
		Interface: fftypes.NewUUID(),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			Name:    "doStuff",
			ID:      fftypes.NewUUID(),
			Params:  fftypes.FFIParams{},
			Returns: fftypes.FFIParams{},
		},
	}

	mim.On("NormalizeSigningKey", mock.Anything, "", identity.KeyNormalizationBlockchainPlugin).Return("key-resolved", nil)
	mth.On("SubmitNewTransaction", mock.Anything, "ns1", fftypes.TransactionTypeContractInvoke).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", mock.Anything, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Namespace == "ns1" && op.Type == fftypes.OpTypeBlockchainInvoke && op.Plugin == "mockblockchain"
	})).Return(nil)
	mbi.On("QueryContract", mock.Anything, req.Location, req.Method, req.Input).Return(struct{}{}, nil)

	_, err := cm.InvokeContract(context.Background(), "ns1", req, false)

	assert.NoError(t, err)
}

func TestCallContractInvalidType(t *testing.T) {
	cm := newTestContractManager()
	mim := cm.identity.(*identitymanagermocks.Manager)
	mdi := cm.database.(*databasemocks.Plugin)
	mth := cm.txHelper.(*txcommonmocks.Helper)

	req := &fftypes.ContractCallRequest{
		Interface: fftypes.NewUUID(),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			Name:    "doStuff",
			ID:      fftypes.NewUUID(),
			Params:  fftypes.FFIParams{},
			Returns: fftypes.FFIParams{},
		},
	}

	mim.On("NormalizeSigningKey", mock.Anything, "", identity.KeyNormalizationBlockchainPlugin).Return("key-resolved", nil)
	mth.On("SubmitNewTransaction", mock.Anything, "ns1", fftypes.TransactionTypeContractInvoke).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", mock.Anything, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Namespace == "ns1" && op.Type == fftypes.OpTypeBlockchainInvoke && op.Plugin == "mockblockchain"
	})).Return(nil)

	assert.PanicsWithValue(t, "unknown call type: ", func() {
		cm.InvokeContract(context.Background(), "ns1", req, false)
	})
}

func TestGetContractListenerByID(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	id := fftypes.NewUUID()
	mdi.On("GetContractListenerByID", context.Background(), id).Return(&fftypes.ContractListener{}, nil)

	_, err := cm.GetContractListenerByNameOrID(context.Background(), "ns", id.String())
	assert.NoError(t, err)
}

func TestGetContractListenerByIDFail(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	id := fftypes.NewUUID()
	mdi.On("GetContractListenerByID", context.Background(), id).Return(nil, fmt.Errorf("pop"))

	_, err := cm.GetContractListenerByNameOrID(context.Background(), "ns", id.String())
	assert.EqualError(t, err, "pop")
}

func TestGetContractListenerByName(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractListener", context.Background(), "ns", "sub1").Return(&fftypes.ContractListener{}, nil)

	_, err := cm.GetContractListenerByNameOrID(context.Background(), "ns", "sub1")
	assert.NoError(t, err)
}

func TestGetContractListenerBadName(t *testing.T) {
	cm := newTestContractManager()

	_, err := cm.GetContractListenerByNameOrID(context.Background(), "ns", "!bad")
	assert.Regexp(t, "FF00140", err)
}

func TestGetContractListenerByNameFail(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractListener", context.Background(), "ns", "sub1").Return(nil, fmt.Errorf("pop"))

	_, err := cm.GetContractListenerByNameOrID(context.Background(), "ns", "sub1")
	assert.EqualError(t, err, "pop")
}

func TestGetContractListenerNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractListener", context.Background(), "ns", "sub1").Return(nil, nil)

	_, err := cm.GetContractListenerByNameOrID(context.Background(), "ns", "sub1")
	assert.Regexp(t, "FF10109", err)
}

func TestGetContractListeners(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractListeners", context.Background(), mock.Anything).Return(nil, nil, nil)

	f := database.ContractListenerQueryFactory.NewFilter(context.Background())
	_, _, err := cm.GetContractListeners(context.Background(), "ns", f.And())
	assert.NoError(t, err)
}

func TestGetContractAPIListeners(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	interfaceID := fftypes.NewUUID()
	api := &fftypes.ContractAPI{
		Interface: &fftypes.FFIReference{
			ID: interfaceID,
		},
		Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
			"address": "0x123",
		}.String()),
	}
	event := &fftypes.FFIEvent{
		FFIEventDefinition: fftypes.FFIEventDefinition{
			Name: "changed",
		},
	}

	mdi.On("GetContractAPIByName", context.Background(), "ns", "simple").Return(api, nil)
	mdi.On("GetFFIByID", context.Background(), interfaceID).Return(&fftypes.FFI{}, nil)
	mdi.On("GetFFIEvent", context.Background(), "ns", interfaceID, "changed").Return(event, nil)
	mbi.On("GenerateEventSignature", context.Background(), mock.Anything).Return("changed")
	mdi.On("GetContractListeners", context.Background(), mock.Anything).Return(nil, nil, nil)

	f := database.ContractListenerQueryFactory.NewFilter(context.Background())
	_, _, err := cm.GetContractAPIListeners(context.Background(), "ns", "simple", "changed", f.And())
	assert.NoError(t, err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestGetContractAPIListenersNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractAPIByName", context.Background(), "ns", "simple").Return(nil, nil)

	f := database.ContractListenerQueryFactory.NewFilter(context.Background())
	_, _, err := cm.GetContractAPIListeners(context.Background(), "ns", "simple", "changed", f.And())
	assert.Regexp(t, "FF10109", err)

	mdi.AssertExpectations(t)
}

func TestGetContractAPIListenersFail(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractAPIByName", context.Background(), "ns", "simple").Return(nil, fmt.Errorf("pop"))

	f := database.ContractListenerQueryFactory.NewFilter(context.Background())
	_, _, err := cm.GetContractAPIListeners(context.Background(), "ns", "simple", "changed", f.And())
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestGetContractAPIListenersEventNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	interfaceID := fftypes.NewUUID()
	api := &fftypes.ContractAPI{
		Interface: &fftypes.FFIReference{
			ID: interfaceID,
		},
		Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
			"address": "0x123",
		}.String()),
	}

	mdi.On("GetContractAPIByName", context.Background(), "ns", "simple").Return(api, nil)
	mdi.On("GetFFIByID", context.Background(), interfaceID).Return(&fftypes.FFI{}, nil)
	mdi.On("GetFFIEvent", context.Background(), "ns", interfaceID, "changed").Return(nil, nil)

	f := database.ContractListenerQueryFactory.NewFilter(context.Background())
	_, _, err := cm.GetContractAPIListeners(context.Background(), "ns", "simple", "changed", f.And())
	assert.Regexp(t, "FF10370", err)

	mdi.AssertExpectations(t)
}

func TestDeleteContractListener(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractListener{
		ID: fftypes.NewUUID(),
	}

	mdi.On("GetContractListener", context.Background(), "ns", "sub1").Return(sub, nil)
	mbi.On("DeleteContractListener", context.Background(), sub).Return(nil)
	mdi.On("DeleteContractListenerByID", context.Background(), sub.ID).Return(nil)

	err := cm.DeleteContractListenerByNameOrID(context.Background(), "ns", "sub1")
	assert.NoError(t, err)
}

func TestDeleteContractListenerBlockchainFail(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractListener{
		ID: fftypes.NewUUID(),
	}

	mdi.On("GetContractListener", context.Background(), "ns", "sub1").Return(sub, nil)
	mbi.On("DeleteContractListener", context.Background(), sub).Return(fmt.Errorf("pop"))
	mdi.On("DeleteContractListenerByID", context.Background(), sub.ID).Return(nil)

	err := cm.DeleteContractListenerByNameOrID(context.Background(), "ns", "sub1")
	assert.EqualError(t, err, "pop")
}

func TestDeleteContractListenerNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractListener", context.Background(), "ns", "sub1").Return(nil, nil)

	err := cm.DeleteContractListenerByNameOrID(context.Background(), "ns", "sub1")
	assert.Regexp(t, "FF10109", err)
}

func TestInvokeContractAPI(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	mdi := cm.database.(*databasemocks.Plugin)
	mth := cm.txHelper.(*txcommonmocks.Helper)
	mom := cm.operations.(*operationmocks.Manager)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			ID:   fftypes.NewUUID(),
			Name: "peel",
		},
	}

	api := &fftypes.ContractAPI{
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
		Location: fftypes.JSONAnyPtr(""),
	}

	mim.On("NormalizeSigningKey", mock.Anything, "", identity.KeyNormalizationBlockchainPlugin).Return("key-resolved", nil)
	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(api, nil)
	mth.On("SubmitNewTransaction", mock.Anything, "ns1", fftypes.TransactionTypeContractInvoke).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", mock.Anything, mock.MatchedBy(func(op *fftypes.Operation) bool {
		return op.Namespace == "ns1" && op.Type == fftypes.OpTypeBlockchainInvoke && op.Plugin == "mockblockchain"
	})).Return(nil)
	mom.On("RunOperation", mock.Anything, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(blockchainInvokeData)
		return op.Type == fftypes.OpTypeBlockchainInvoke && data.Request == req
	})).Return(nil, nil)

	_, err := cm.InvokeContractAPI(context.Background(), "ns1", "banana", "peel", req, false)

	assert.NoError(t, err)

	mdb.AssertExpectations(t)
	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestInvokeContractAPIFailContractLookup(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			ID: fftypes.NewUUID(),
		},
	}

	mim.On("NormalizeSigningKey", mock.Anything, "", identity.KeyNormalizationBlockchainPlugin).Return("key-resolved", nil)
	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(nil, fmt.Errorf("pop"))

	_, err := cm.InvokeContractAPI(context.Background(), "ns1", "banana", "peel", req, false)

	assert.Regexp(t, "pop", err)
}

func TestInvokeContractAPIContractNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			ID: fftypes.NewUUID(),
		},
	}

	mim.On("NormalizeSigningKey", mock.Anything, "", identity.KeyNormalizationBlockchainPlugin).Return("key-resolved", nil)
	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(nil, nil)

	_, err := cm.InvokeContractAPI(context.Background(), "ns1", "banana", "peel", req, false)

	assert.Regexp(t, "FF10109", err)
}

func TestGetContractAPI(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	api := &fftypes.ContractAPI{
		Namespace: "ns1",
		Name:      "banana",
	}
	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(api, nil)

	result, err := cm.GetContractAPI(context.Background(), "http://localhost/api", "ns1", "banana")

	assert.NoError(t, err)
	assert.Equal(t, "http://localhost/api/namespaces/ns1/apis/banana/api/swagger.json", result.URLs.OpenAPI)
	assert.Equal(t, "http://localhost/api/namespaces/ns1/apis/banana/api", result.URLs.UI)
}

func TestGetContractAPIs(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	apis := []*fftypes.ContractAPI{
		{
			Namespace: "ns1",
			Name:      "banana",
		},
	}
	filter := database.ContractAPIQueryFactory.NewFilter(context.Background()).And()
	mdb.On("GetContractAPIs", mock.Anything, "ns1", filter).Return(apis, &database.FilterResult{}, nil)

	results, _, err := cm.GetContractAPIs(context.Background(), "http://localhost/api", "ns1", filter)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "http://localhost/api/namespaces/ns1/apis/banana/api/swagger.json", results[0].URLs.OpenAPI)
	assert.Equal(t, "http://localhost/api/namespaces/ns1/apis/banana/api", results[0].URLs.UI)
}

func TestGetContractAPIInterface(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	interfaceID := fftypes.NewUUID()
	api := &fftypes.ContractAPI{
		Namespace: "ns1",
		Name:      "banana",
		Interface: &fftypes.FFIReference{ID: interfaceID},
	}

	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(api, nil)
	mdb.On("GetFFIByID", mock.Anything, interfaceID).Return(&fftypes.FFI{}, nil)
	mdb.On("GetFFIMethods", mock.Anything, mock.Anything).Return([]*fftypes.FFIMethod{
		{ID: fftypes.NewUUID(), Name: "method1"},
	}, nil, nil)
	mdb.On("GetFFIEvents", mock.Anything, mock.Anything).Return([]*fftypes.FFIEvent{
		{ID: fftypes.NewUUID(), FFIEventDefinition: fftypes.FFIEventDefinition{Name: "event1"}},
	}, nil, nil)
	mbi.On("GenerateEventSignature", mock.Anything, mock.MatchedBy(func(ev *fftypes.FFIEventDefinition) bool {
		return ev.Name == "event1"
	})).Return("event1Sig")

	result, err := cm.GetContractAPIInterface(context.Background(), "ns1", "banana")

	assert.NoError(t, err)
	assert.NotNil(t, result)

	mdb.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestGetContractAPIInterfaceFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(nil, fmt.Errorf("pop"))

	_, err := cm.GetContractAPIInterface(context.Background(), "ns1", "banana")

	assert.EqualError(t, err, "pop")

	mdb.AssertExpectations(t)
}

func TestBroadcastContractAPI(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdb := cm.database.(*databasemocks.Plugin)
	mbm := cm.broadcast.(*broadcastmocks.Manager)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), api.Location).Return(api.Location, nil)
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(&fftypes.FFI{}, nil)
	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.ContractAPI"), fftypes.SystemTagDefineContractAPI, false).Return(msg, nil)

	api, err := cm.BroadcastContractAPI(context.Background(), "http://localhost/api", "ns1", api, false)

	assert.NoError(t, err)
	assert.NotNil(t, api)
	assert.NotEmpty(t, api.URLs.OpenAPI)
	assert.NotEmpty(t, api.URLs.UI)

	mbi.AssertExpectations(t)
	mdb.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestBroadcastContractAPIBadLocation(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), api.Location).Return(nil, fmt.Errorf("pop"))

	_, err := cm.BroadcastContractAPI(context.Background(), "http://localhost/api", "ns1", api, false)

	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
}

func TestBroadcastContractAPIExisting(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdb := cm.database.(*databasemocks.Plugin)
	mbm := cm.broadcast.(*broadcastmocks.Manager)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	apiID := fftypes.NewUUID()
	existing := &fftypes.ContractAPI{
		ID:        apiID,
		Namespace: "ns1",
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}
	api := &fftypes.ContractAPI{
		ID:        apiID,
		Namespace: "ns1",
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), api.Location).Return(api.Location, nil)
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(existing, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(&fftypes.FFI{}, nil)
	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.ContractAPI"), fftypes.SystemTagDefineContractAPI, false).Return(msg, nil)

	_, err := cm.BroadcastContractAPI(context.Background(), "http://localhost/api", "ns1", api, false)

	assert.NoError(t, err)

	mbi.AssertExpectations(t)
	mdb.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestBroadcastContractAPICannotChangeLocation(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdb := cm.database.(*databasemocks.Plugin)

	apiID := fftypes.NewUUID()
	existing := &fftypes.ContractAPI{
		ID:        apiID,
		Namespace: "ns1",
		Location:  fftypes.JSONAnyPtr(`"old"`),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}
	api := &fftypes.ContractAPI{
		ID:        apiID,
		Namespace: "ns1",
		Location:  fftypes.JSONAnyPtr(`"new"`),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), api.Location).Return(api.Location, nil)
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(existing, nil)

	_, err := cm.BroadcastContractAPI(context.Background(), "http://localhost/api", "ns1", api, false)

	assert.Regexp(t, "FF10316", err)

	mbi.AssertExpectations(t)
	mdb.AssertExpectations(t)
}

func TestBroadcastContractAPIInterfaceName(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdb := cm.database.(*databasemocks.Plugin)
	mbm := cm.broadcast.(*broadcastmocks.Manager)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			Name:    "my-ffi",
			Version: "1",
		},
	}
	interfaceID := fftypes.NewUUID()

	mbi.On("NormalizeContractLocation", context.Background(), api.Location).Return(api.Location, nil)
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFI", mock.Anything, "ns1", "my-ffi", "1").Return(&fftypes.FFI{ID: interfaceID}, nil)
	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.ContractAPI"), fftypes.SystemTagDefineContractAPI, false).Return(msg, nil)

	_, err := cm.BroadcastContractAPI(context.Background(), "http://localhost/api", "ns1", api, false)

	assert.NoError(t, err)
	assert.Equal(t, *interfaceID, *api.Interface.ID)

	mbi.AssertExpectations(t)
	mdb.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestBroadcastContractAPIFail(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdb := cm.database.(*databasemocks.Plugin)
	mbm := cm.broadcast.(*broadcastmocks.Manager)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), api.Location).Return(api.Location, nil)
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(&fftypes.FFI{}, nil)
	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.ContractAPI"), fftypes.SystemTagDefineContractAPI, false).Return(nil, fmt.Errorf("pop"))

	_, err := cm.BroadcastContractAPI(context.Background(), "http://localhost/api", "ns1", api, false)

	assert.Regexp(t, "pop", err)

	mbi.AssertExpectations(t)
	mdb.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestBroadcastContractAPINoInterface(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdb := cm.database.(*databasemocks.Plugin)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
	}

	mbi.On("NormalizeContractLocation", context.Background(), api.Location).Return(api.Location, nil)
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)

	_, err := cm.BroadcastContractAPI(context.Background(), "http://localhost/api", "ns1", api, false)

	assert.Regexp(t, "FF10303", err)

	mbi.AssertExpectations(t)
	mdb.AssertExpectations(t)
}

func TestBroadcastContractAPIInterfaceIDFail(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdb := cm.database.(*databasemocks.Plugin)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), api.Location).Return(api.Location, nil)
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(nil, fmt.Errorf("pop"))

	_, err := cm.BroadcastContractAPI(context.Background(), "http://localhost/api", "ns1", api, false)

	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdb.AssertExpectations(t)
}

func TestBroadcastContractAPIInterfaceIDNotFound(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdb := cm.database.(*databasemocks.Plugin)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), api.Location).Return(api.Location, nil)
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(nil, nil)

	_, err := cm.BroadcastContractAPI(context.Background(), "http://localhost/api", "ns1", api, false)

	assert.Regexp(t, "FF10303.*"+api.Interface.ID.String(), err)

	mbi.AssertExpectations(t)
	mdb.AssertExpectations(t)
}

func TestBroadcastContractAPIInterfaceNameFail(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdb := cm.database.(*databasemocks.Plugin)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			Name:    "my-ffi",
			Version: "1",
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), api.Location).Return(api.Location, nil)
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFI", mock.Anything, "ns1", "my-ffi", "1").Return(nil, fmt.Errorf("pop"))

	_, err := cm.BroadcastContractAPI(context.Background(), "http://localhost/api", "ns1", api, false)

	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdb.AssertExpectations(t)
}

func TestBroadcastContractAPIInterfaceNameNotFound(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdb := cm.database.(*databasemocks.Plugin)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			Name:    "my-ffi",
			Version: "1",
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), api.Location).Return(api.Location, nil)
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFI", mock.Anything, "ns1", "my-ffi", "1").Return(nil, nil)

	_, err := cm.BroadcastContractAPI(context.Background(), "http://localhost/api", "ns1", api, false)

	assert.Regexp(t, "FF10303.*my-ffi", err)

	mbi.AssertExpectations(t)
	mdb.AssertExpectations(t)
}

func TestBroadcastContractAPIInterfaceNoVersion(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdb := cm.database.(*databasemocks.Plugin)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			Name: "my-ffi",
		},
	}

	mbi.On("NormalizeContractLocation", context.Background(), api.Location).Return(api.Location, nil)
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)

	_, err := cm.BroadcastContractAPI(context.Background(), "http://localhost/api", "ns1", api, false)

	assert.Regexp(t, "FF10303.*my-ffi", err)

	mbi.AssertExpectations(t)
	mdb.AssertExpectations(t)
}

func TestValidateFFIParamBadSchemaJSON(t *testing.T) {
	cm := newTestContractManager()
	param := &fftypes.FFIParam{
		Name:   "x",
		Schema: fftypes.JSONAnyPtr(`{"type": "integer"`),
	}
	err := cm.validateFFIParam(context.Background(), param)
	assert.Regexp(t, "unexpected EOF", err)
}

func TestCheckParamSchemaBadSchema(t *testing.T) {
	cm := newTestContractManager()
	param := &fftypes.FFIParam{
		Name:   "x",
		Schema: fftypes.JSONAnyPtr(`{"type": "integer"`),
	}
	err := cm.checkParamSchema(context.Background(), 1, param)
	assert.Regexp(t, "unexpected EOF", err)
}

func TestCheckParamSchemaCompileFail(t *testing.T) {
	cm := newTestContractManager()
	param := &fftypes.FFIParam{
		Name:   "x",
		Schema: fftypes.JSONAnyPtr(``),
	}
	err := cm.checkParamSchema(context.Background(), 1, param)
	assert.Regexp(t, "compilation failed", err)
}

func TestAddJSONSchemaExtension(t *testing.T) {
	cm := &contractManager{
		database:          &databasemocks.Plugin{},
		broadcast:         &broadcastmocks.Manager{},
		identity:          &identitymanagermocks.Manager{},
		blockchain:        &blockchainmocks.Plugin{},
		ffiParamValidator: &MockFFIParamValidator{},
	}
	c := cm.newFFISchemaCompiler()
	assert.NotNil(t, c)
}

func TestGenerateFFI(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mbi.On("GenerateFFI", mock.Anything, mock.Anything).Return(&fftypes.FFI{
		Name: "generated",
	}, nil)
	ffi, err := cm.GenerateFFI(context.Background(), "default", &fftypes.FFIGenerationRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, ffi)
	assert.Equal(t, "generated", ffi.Name)
}

type MockFFIParamValidator struct{}

func (v MockFFIParamValidator) Compile(ctx jsonschema.CompilerContext, m map[string]interface{}) (jsonschema.ExtSchema, error) {
	return nil, nil
}

func (v *MockFFIParamValidator) GetMetaSchema() *jsonschema.Schema {
	return nil
}

func (v *MockFFIParamValidator) GetExtensionName() string {
	return "ffi"
}
