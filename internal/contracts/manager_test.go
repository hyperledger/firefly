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
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/publicstoragemocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestContractManager() *contractManager {
	mdb := &databasemocks.Plugin{}
	mps := &publicstoragemocks.Plugin{}
	mbm := &broadcastmocks.Manager{}
	mim := &identitymanagermocks.Manager{}
	mbi := &blockchainmocks.Plugin{}
	mbi.On("GetFFIParamValidator", mock.Anything).Return(nil, nil)

	rag := mdb.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}
	cm, _ := NewContractManager(context.Background(), mdb, mps, mbm, mim, mbi)
	return cm.(*contractManager)
}

func TestNewContractManagerFail(t *testing.T) {
	_, err := NewContractManager(context.Background(), nil, nil, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestNewContractManagerFFISchemaLoaderFail(t *testing.T) {
	mdb := &databasemocks.Plugin{}
	mps := &publicstoragemocks.Plugin{}
	mbm := &broadcastmocks.Manager{}
	mim := &identitymanagermocks.Manager{}
	mbi := &blockchainmocks.Plugin{}
	mbi.On("GetFFIParamValidator", mock.Anything).Return(nil, fmt.Errorf("pop"))
	_, err := NewContractManager(context.Background(), mdb, mps, mbm, mim, mbi)
	assert.Regexp(t, "pop", err)
}

func TestNewContractManagerFFISchemaLoader(t *testing.T) {
	mdb := &databasemocks.Plugin{}
	mps := &publicstoragemocks.Plugin{}
	mbm := &broadcastmocks.Manager{}
	mim := &identitymanagermocks.Manager{}
	mbi := &blockchainmocks.Plugin{}
	mbi.On("GetFFIParamValidator", mock.Anything).Return(&ethereum.FFIParamValidator{}, nil)
	_, err := NewContractManager(context.Background(), mdb, mps, mbm, mim, mbi)
	assert.NoError(t, err)
}

func TestBroadcastFFI(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	mbm := cm.broadcast.(*broadcastmocks.Manager)

	mdb.On("GetFFI", mock.Anything, "ns1", "", "").Return(nil, nil)
	mim.On("GetOrgKey", mock.Anything).Return("key", nil)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.FFI"), fftypes.SystemTagDefineFFI, false).Return(msg, nil)
	ffi := &fftypes.FFI{
		ID: fftypes.NewUUID(),
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

	mdb.On("GetFFI", mock.Anything, "ns1", "", "").Return(nil, nil)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.FFI"), fftypes.SystemTagDefineFFI, false).Return(msg, nil)
	ffi := &fftypes.FFI{
		ID: fftypes.NewUUID(),
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

	mdb.On("GetFFI", mock.Anything, "ns1", "", "").Return(&fftypes.FFI{}, nil)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.FFI"), fftypes.SystemTagDefineFFI, false).Return(msg, nil)
	ffi := &fftypes.FFI{
		ID: fftypes.NewUUID(),
	}
	_, err := cm.BroadcastFFI(context.Background(), "ns1", ffi, false)
	assert.Regexp(t, "FF10302", err)
}

func TestBroadcastFFIFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mbm := cm.broadcast.(*broadcastmocks.Manager)
	mim := cm.identity.(*identitymanagermocks.Manager)

	mdb.On("GetFFI", mock.Anything, "ns1", "", "").Return(nil, nil)
	mim.On("GetOrgKey", mock.Anything).Return("key", nil)

	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.FFI"), fftypes.SystemTagDefineFFI, false).Return(nil, fmt.Errorf("pop"))
	ffi := &fftypes.FFI{
		ID: fftypes.NewUUID(),
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

func TestValidateFFIBadMethod(t *testing.T) {
	cm := newTestContractManager()
	ffi := &fftypes.FFI{
		Name:      "math",
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

func TestAddContractSubscriptionInline(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
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
		},
	}

	mbi.On("AddSubscription", context.Background(), sub).Return(nil)
	mdi.On("UpsertContractSubscription", context.Background(), &sub.ContractSubscription).Return(nil)

	result, err := cm.AddContractSubscription(context.Background(), "ns", sub)
	assert.NoError(t, err)
	assert.NotNil(t, result.ID)
	assert.NotNil(t, result.Event)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionByRef(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

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

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
		},
		EventID: event.ID,
	}

	mbi.On("AddSubscription", context.Background(), sub).Return(nil)
	mdi.On("GetFFIEventByID", context.Background(), event.ID).Return(event, nil)
	mdi.On("UpsertContractSubscription", context.Background(), &sub.ContractSubscription).Return(nil)

	result, err := cm.AddContractSubscription(context.Background(), "ns1", sub)
	assert.NoError(t, err)
	assert.NotNil(t, result.ID)
	assert.NotNil(t, result.Event)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionByRefLookupFail(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
		},
		EventID: fftypes.NewUUID(),
	}

	mdi.On("GetFFIEventByID", context.Background(), mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, err := cm.AddContractSubscription(context.Background(), "ns1", sub)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionMissingEventOrID(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
		},
	}

	_, err := cm.AddContractSubscription(context.Background(), "ns2", sub)
	assert.Regexp(t, "FF10317", err)

	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionByRefLookupWrongNS(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	eventID := fftypes.NewUUID()
	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
		},
		EventID: eventID,
	}

	mdi.On("GetFFIEventByID", context.Background(), mock.Anything).Return(&fftypes.FFIEvent{
		ID:        eventID,
		Namespace: "ns1",
	}, nil)

	_, err := cm.AddContractSubscription(context.Background(), "ns2", sub)
	assert.Regexp(t, "FF10318", err)

	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionBadNamespace(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{}

	_, err := cm.AddContractSubscription(context.Background(), "!bad", sub)
	assert.Regexp(t, "FF10131.*'namespace'", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionBadName(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Name: "!bad",
		},
	}

	_, err := cm.AddContractSubscription(context.Background(), "ns", sub)
	assert.Regexp(t, "FF10131.*'name'", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionNameConflict(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Name: "sub1",
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
		},
		EventID: fftypes.NewUUID(),
	}

	mdi.On("GetContractSubscription", context.Background(), "ns", "sub1").Return(&fftypes.ContractSubscription{}, nil)

	_, err := cm.AddContractSubscription(context.Background(), "ns", sub)
	assert.Regexp(t, "FF10312", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionNameError(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Name: "sub1",
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
		},
		EventID: fftypes.NewUUID(),
	}

	mdi.On("GetContractSubscription", context.Background(), "ns", "sub1").Return(nil, fmt.Errorf("pop"))

	_, err := cm.AddContractSubscription(context.Background(), "ns", sub)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionValidateFail(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
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
		},
	}

	_, err := cm.AddContractSubscription(context.Background(), "ns", sub)
	assert.Regexp(t, "does not validate", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionBlockchainFail(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
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
		},
	}

	mbi.On("AddSubscription", context.Background(), sub).Return(fmt.Errorf("pop"))

	_, err := cm.AddContractSubscription(context.Background(), "ns", sub)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionUpsertSubFail(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
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
		},
	}

	mbi.On("AddSubscription", context.Background(), sub).Return(nil)
	mdi.On("UpsertContractSubscription", context.Background(), &sub.ContractSubscription).Return(fmt.Errorf("pop"))

	_, err := cm.AddContractSubscription(context.Background(), "ns", sub)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestGetFFI(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mdb.On("GetFFI", mock.Anything, "ns1", "ffi", "v1.0.0").Return(&fftypes.FFI{}, nil)
	_, err := cm.GetFFI(context.Background(), "ns1", "ffi", "v1.0.0")
	assert.NoError(t, err)
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

	ffi, err := cm.GetFFIByIDWithChildren(context.Background(), cid)

	assert.NoError(t, err)
	mdb.AssertExpectations(t)

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

func TestGetFFIByIDWithChildrenFFINotFoundl(t *testing.T) {
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
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			Name:    "doStuff",
			ID:      fftypes.NewUUID(),
			Params:  fftypes.FFIParams{},
			Returns: fftypes.FFIParams{},
		},
	}

	mim.On("ResolveSigningKey", mock.Anything, "").Return("key-resolved", nil)
	mbi.On("InvokeContract", mock.Anything, mock.AnythingOfType("*fftypes.UUID"), "key-resolved", req.Location, req.Method, req.Input).Return(struct{}{}, nil)

	_, err := cm.InvokeContract(context.Background(), "ns1", req)

	assert.NoError(t, err)
}

func TestInvokeContractFailResolveSigningKey(t *testing.T) {
	cm := newTestContractManager()
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
	}

	mim.On("ResolveSigningKey", mock.Anything, "").Return("", fmt.Errorf("pop"))

	_, err := cm.InvokeContract(context.Background(), "ns1", req)

	assert.Regexp(t, "pop", err)
}

func TestInvokeContractFailResolve(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
	}

	mim.On("ResolveSigningKey", mock.Anything, "").Return("key-resolved", nil)
	mbi.On("InvokeContract", mock.Anything, mock.AnythingOfType("*fftypes.UUID"), "key-resolved", req.Location, req.Method, req.Input).Return(struct{}{}, nil)

	_, err := cm.InvokeContract(context.Background(), "ns1", req)

	assert.Regexp(t, "FF10313", err)
}

func TestInvokeContractNoMethodSignature(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.ContractCallRequest{
		Type:     fftypes.CallTypeInvoke,
		Ledger:   fftypes.JSONAnyPtr(""),
		Location: fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			Name: "sum",
		},
	}

	mim.On("ResolveSigningKey", mock.Anything, "").Return("key-resolved", nil)
	mbi.On("InvokeContract", mock.Anything, mock.AnythingOfType("*fftypes.UUID"), "key-resolved", mock.Anything, mock.AnythingOfType("*fftypes.FFIMethod"), mock.Anything).Return(struct{}{}, nil)

	_, err := cm.InvokeContract(context.Background(), "ns1", req)

	assert.Regexp(t, "FF10314", err)
}

func TestInvokeContractMethodNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			Name: "sum",
		},
	}

	mim.On("ResolveSigningKey", mock.Anything, "").Return("key-resolved", nil)
	mdb.On("GetFFIMethod", mock.Anything, "ns1", req.Interface, req.Method.Name).Return(nil, fmt.Errorf("pop"))

	_, err := cm.InvokeContract(context.Background(), "ns1", req)

	assert.Regexp(t, "FF10315", err)
}

func TestInvokeContractMethodBadInput(t *testing.T) {
	cm := newTestContractManager()
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Ledger:    fftypes.JSONAnyPtr(""),
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
	mim.On("ResolveSigningKey", mock.Anything, "").Return("key-resolved", nil)

	_, err := cm.InvokeContract(context.Background(), "ns1", req)
	assert.Regexp(t, "FF10304", err)
}

func TestQueryContract(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeQuery,
		Interface: fftypes.NewUUID(),
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			Name:    "doStuff",
			ID:      fftypes.NewUUID(),
			Params:  fftypes.FFIParams{},
			Returns: fftypes.FFIParams{},
		},
	}

	mim.On("ResolveSigningKey", mock.Anything, "").Return("key-resolved", nil)
	mbi.On("QueryContract", mock.Anything, req.Location, req.Method, req.Input).Return(struct{}{}, nil)

	_, err := cm.InvokeContract(context.Background(), "ns1", req)

	assert.NoError(t, err)
}

func TestCallContractInvalidType(t *testing.T) {
	cm := newTestContractManager()
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.ContractCallRequest{
		Interface: fftypes.NewUUID(),
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			Name:    "doStuff",
			ID:      fftypes.NewUUID(),
			Params:  fftypes.FFIParams{},
			Returns: fftypes.FFIParams{},
		},
	}

	mim.On("ResolveSigningKey", mock.Anything, "").Return("key-resolved", nil)

	assert.PanicsWithValue(t, "unknown call type: ", func() {
		cm.InvokeContract(context.Background(), "ns1", req)
	})
}

func TestGetContractSubscriptionByID(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	id := fftypes.NewUUID()
	mdi.On("GetContractSubscriptionByID", context.Background(), id).Return(&fftypes.ContractSubscription{}, nil)

	_, err := cm.GetContractSubscriptionByNameOrID(context.Background(), "ns", id.String())
	assert.NoError(t, err)
}

func TestGetContractSubscriptionByIDFail(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	id := fftypes.NewUUID()
	mdi.On("GetContractSubscriptionByID", context.Background(), id).Return(nil, fmt.Errorf("pop"))

	_, err := cm.GetContractSubscriptionByNameOrID(context.Background(), "ns", id.String())
	assert.EqualError(t, err, "pop")
}

func TestGetContractSubscriptionByName(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractSubscription", context.Background(), "ns", "sub1").Return(&fftypes.ContractSubscription{}, nil)

	_, err := cm.GetContractSubscriptionByNameOrID(context.Background(), "ns", "sub1")
	assert.NoError(t, err)
}

func TestGetContractSubscriptionBadName(t *testing.T) {
	cm := newTestContractManager()

	_, err := cm.GetContractSubscriptionByNameOrID(context.Background(), "ns", "!bad")
	assert.Regexp(t, "FF10131", err)
}

func TestGetContractSubscriptionByNameFail(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractSubscription", context.Background(), "ns", "sub1").Return(nil, fmt.Errorf("pop"))

	_, err := cm.GetContractSubscriptionByNameOrID(context.Background(), "ns", "sub1")
	assert.EqualError(t, err, "pop")
}

func TestGetContractSubscriptionNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractSubscription", context.Background(), "ns", "sub1").Return(nil, nil)

	_, err := cm.GetContractSubscriptionByNameOrID(context.Background(), "ns", "sub1")
	assert.Regexp(t, "FF10109", err)
}

func TestGetContractSubscriptions(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractSubscriptions", context.Background(), mock.Anything).Return(nil, nil, nil)

	f := database.ContractSubscriptionQueryFactory.NewFilter(context.Background())
	_, _, err := cm.GetContractSubscriptions(context.Background(), "ns", f.And())
	assert.NoError(t, err)
}

func TestDeleteContractSubscription(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscription{
		ID: fftypes.NewUUID(),
	}

	mdi.On("GetContractSubscription", context.Background(), "ns", "sub1").Return(sub, nil)
	mbi.On("DeleteSubscription", context.Background(), sub).Return(nil)
	mdi.On("DeleteContractSubscriptionByID", context.Background(), sub.ID).Return(nil)

	err := cm.DeleteContractSubscriptionByNameOrID(context.Background(), "ns", "sub1")
	assert.NoError(t, err)
}

func TestDeleteContractSubscriptionBlockchainFail(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscription{
		ID: fftypes.NewUUID(),
	}

	mdi.On("GetContractSubscription", context.Background(), "ns", "sub1").Return(sub, nil)
	mbi.On("DeleteSubscription", context.Background(), sub).Return(fmt.Errorf("pop"))
	mdi.On("DeleteContractSubscriptionByID", context.Background(), sub.ID).Return(nil)

	err := cm.DeleteContractSubscriptionByNameOrID(context.Background(), "ns", "sub1")
	assert.EqualError(t, err, "pop")
}

func TestDeleteContractSubscriptionNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractSubscription", context.Background(), "ns", "sub1").Return(nil, nil)

	err := cm.DeleteContractSubscriptionByNameOrID(context.Background(), "ns", "sub1")
	assert.Regexp(t, "FF10109", err)
}

func TestGetContractEventByID(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	id := fftypes.NewUUID()
	mdi.On("GetBlockchainEventByID", context.Background(), id).Return(&fftypes.BlockchainEvent{}, nil)

	_, err := cm.GetContractEventByID(context.Background(), id)
	assert.NoError(t, err)
}

func TestGetContractEvents(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetBlockchainEvents", context.Background(), mock.Anything).Return(nil, nil, nil)

	f := database.ContractSubscriptionQueryFactory.NewFilter(context.Background())
	_, _, err := cm.GetContractEvents(context.Background(), "ns", f.And())
	assert.NoError(t, err)
}

func TestInvokeContractAPI(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			ID: fftypes.NewUUID(),
		},
	}

	api := &fftypes.ContractAPI{
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
		Location: fftypes.JSONAnyPtr(""),
	}

	mim.On("ResolveSigningKey", mock.Anything, "").Return("key-resolved", nil)
	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(api, nil)
	mdb.On("GetFFIMethod", mock.Anything, "ns1", mock.Anything, mock.Anything).Return(&fftypes.FFIMethod{Name: "peel"}, nil)
	mbi.On("InvokeContract", mock.Anything, mock.AnythingOfType("*fftypes.UUID"), "key-resolved", req.Location, mock.AnythingOfType("*fftypes.FFIMethod"), req.Input).Return(struct{}{}, nil)

	_, err := cm.InvokeContractAPI(context.Background(), "ns1", "banana", "peel", req)

	assert.NoError(t, err)
}

func TestInvokeContractAPIFailContractLookup(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			ID: fftypes.NewUUID(),
		},
	}

	mim.On("ResolveSigningKey", mock.Anything, "").Return("key-resolved", nil)
	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(nil, fmt.Errorf("pop"))

	_, err := cm.InvokeContractAPI(context.Background(), "ns1", "banana", "peel", req)

	assert.Regexp(t, "pop", err)
}

func TestInvokeContractAPIContractNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	req := &fftypes.ContractCallRequest{
		Type:      fftypes.CallTypeInvoke,
		Interface: fftypes.NewUUID(),
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Method: &fftypes.FFIMethod{
			ID: fftypes.NewUUID(),
		},
	}

	mim.On("ResolveSigningKey", mock.Anything, "").Return("key-resolved", nil)
	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(nil, nil)

	_, err := cm.InvokeContractAPI(context.Background(), "ns1", "banana", "peel", req)

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

func TestBroadcastContractAPI(t *testing.T) {
	cm := newTestContractManager()
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
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(&fftypes.FFI{}, nil)
	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.ContractAPI"), fftypes.SystemTagDefineContractAPI, false).Return(msg, nil)
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.NoError(t, err)
}

func TestBroadcastContractAPIExisting(t *testing.T) {
	cm := newTestContractManager()
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
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}
	api := &fftypes.ContractAPI{
		ID:        apiID,
		Namespace: "ns1",
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(existing, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(&fftypes.FFI{}, nil)
	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.ContractAPI"), fftypes.SystemTagDefineContractAPI, false).Return(msg, nil)
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.NoError(t, err)
}

func TestBroadcastContractAPICannotChangeLocation(t *testing.T) {
	cm := newTestContractManager()
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
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(`"old"`),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}
	api := &fftypes.ContractAPI{
		ID:        apiID,
		Namespace: "ns1",
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(`"new"`),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(existing, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(&fftypes.FFI{}, nil)
	mbm.On("BroadcastDefinition", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.ContractAPI"), fftypes.SystemTagDefineContractAPI, false).Return(msg, nil)
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.Regexp(t, "FF10316", err)
}

func TestBroadcastContractAPIInterfaceName(t *testing.T) {
	cm := newTestContractManager()
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
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			Name:    "my-ffi",
			Version: "1",
		},
	}
	interfaceID := fftypes.NewUUID()
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFI", mock.Anything, "ns1", "my-ffi", "1").Return(&fftypes.FFI{ID: interfaceID}, nil)
	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.ContractAPI"), fftypes.SystemTagDefineContractAPI, false).Return(msg, nil)
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.NoError(t, err)
	assert.Equal(t, *interfaceID, *api.Interface.ID)
}

func TestBroadcastContractAPIFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mbm := cm.broadcast.(*broadcastmocks.Manager)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(&fftypes.FFI{}, nil)
	mbm.On("BroadcastDefinitionAsNode", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.ContractAPI"), fftypes.SystemTagDefineContractAPI, false).Return(nil, fmt.Errorf("pop"))
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.Regexp(t, "pop", err)
}

func TestBroadcastContractAPINoInterface(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
	}
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.Regexp(t, "FF10303", err)
}

func TestBroadcastContractAPIInterfaceIDFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(nil, fmt.Errorf("pop"))
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.EqualError(t, err, "pop")
}

func TestBroadcastContractAPIInterfaceIDNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(nil, nil)
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.Regexp(t, "FF10303.*"+api.Interface.ID.String(), err)
}

func TestBroadcastContractAPIInterfaceNameFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			Name:    "my-ffi",
			Version: "1",
		},
	}
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFI", mock.Anything, "ns1", "my-ffi", "1").Return(nil, fmt.Errorf("pop"))
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.EqualError(t, err, "pop")
}

func TestBroadcastContractAPIInterfaceNameNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			Name:    "my-ffi",
			Version: "1",
		},
	}
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFI", mock.Anything, "ns1", "my-ffi", "1").Return(nil, nil)
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.Regexp(t, "FF10303.*my-ffi", err)
}

func TestBroadcastContractAPIInterfaceNoVersion(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Ledger:    fftypes.JSONAnyPtr(""),
		Location:  fftypes.JSONAnyPtr(""),
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			Name: "my-ffi",
		},
	}
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.Regexp(t, "FF10303.*my-ffi", err)
}

func TestSubscribeContractAPI(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	req := &fftypes.ContractSubscribeRequest{}
	event := &fftypes.FFIEvent{
		FFIEventDefinition: fftypes.FFIEventDefinition{
			Name: "peeled",
		},
	}
	api := &fftypes.ContractAPI{
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
		Location: fftypes.JSONAnyPtr(`"abc"`),
	}

	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(api, nil)
	mdb.On("GetFFIEvent", mock.Anything, "ns1", api.Interface.ID, "peeled").Return(event, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(&fftypes.FFI{}, nil)
	mbi.On("AddSubscription", mock.Anything, mock.MatchedBy(func(sub *fftypes.ContractSubscriptionInput) bool {
		return sub.Event.Name == "peeled" && *sub.Interface.ID == *api.Interface.ID && sub.Location.String() == api.Location.String()
	})).Return(nil)
	mdb.On("UpsertContractSubscription", mock.Anything, mock.MatchedBy(func(sub *fftypes.ContractSubscription) bool {
		return sub.Event.Name == "peeled" && *sub.Interface.ID == *api.Interface.ID && sub.Location.String() == api.Location.String()
	})).Return(nil)

	_, err := cm.SubscribeContractAPI(context.Background(), "ns1", "banana", "peeled", req)

	assert.NoError(t, err)
}

func TestSubscribeContractAPIContractLookupFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	req := &fftypes.ContractSubscribeRequest{}

	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(nil, fmt.Errorf("pop"))

	_, err := cm.SubscribeContractAPI(context.Background(), "ns1", "banana", "peeled", req)

	assert.EqualError(t, err, "pop")
}

func TestSubscribeContractAPIContractNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	req := &fftypes.ContractSubscribeRequest{}

	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(nil, nil)

	_, err := cm.SubscribeContractAPI(context.Background(), "ns1", "banana", "peeled", req)

	assert.Regexp(t, "FF10109", err)
}

func TestSubscribeContractAPIInterfaceNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	req := &fftypes.ContractSubscribeRequest{}
	event := &fftypes.FFIEvent{
		FFIEventDefinition: fftypes.FFIEventDefinition{
			Name: "peeled",
		},
	}
	api := &fftypes.ContractAPI{
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
		Location: fftypes.JSONAnyPtr(`"abc"`),
	}

	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(api, nil)
	mdb.On("GetFFIEvent", mock.Anything, "ns1", api.Interface.ID, "peeled").Return(event, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(nil, nil)

	_, err := cm.SubscribeContractAPI(context.Background(), "ns1", "banana", "peeled", req)

	assert.Regexp(t, "FF10303.*"+api.Interface.ID.String(), err)
}

func TestSubscribeContractAPIEventLookupFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	req := &fftypes.ContractSubscribeRequest{}
	api := &fftypes.ContractAPI{
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
		Location: fftypes.JSONAnyPtr(`"abc"`),
	}

	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(api, nil)
	mdb.On("GetFFIEvent", mock.Anything, "ns1", api.Interface.ID, "peeled").Return(nil, fmt.Errorf("pop"))

	_, err := cm.SubscribeContractAPI(context.Background(), "ns1", "banana", "peeled", req)

	assert.Regexp(t, "FF10321", err)
}

func TestValidateFFIParamBadSchemaJSON(t *testing.T) {
	cm := newTestContractManager()
	param := &fftypes.FFIParam{
		Name:   "x",
		Schema: fftypes.JSONAnyPtr(`{"type": "integer"`),
	}
	err := cm.validateFFIParam(param)
	assert.Regexp(t, "unexpected EOF", err)
}

func TestCheckParamSchemaBadSchema(t *testing.T) {
	cm := newTestContractManager()
	param := &fftypes.FFIParam{
		Name:   "x",
		Schema: fftypes.JSONAnyPtr(`{"type": "integer"`),
	}
	err := cm.checkParamSchema(1, param)
	assert.Regexp(t, "unexpected EOF", err)
}

func TestCheckParamSchemaCompileFail(t *testing.T) {
	cm := newTestContractManager()
	param := &fftypes.FFIParam{
		Name:   "x",
		Schema: fftypes.JSONAnyPtr(``),
	}
	err := cm.checkParamSchema(1, param)
	assert.Regexp(t, "compilation failed", err)
}
