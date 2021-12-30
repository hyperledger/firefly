// Copyright © 2021 Kaleido, Inc.
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
	"errors"
	"fmt"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/oapiffimocks"
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
	mfg := &oapiffimocks.FFISwaggerGen{}

	rag := mdb.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}
	cm, _ := NewContractManager(context.Background(), mdb, mps, mbm, mim, mbi)
	cm.(*contractManager).swaggerGen = mfg
	return cm.(*contractManager)
}

func TestNewContractManagerFail(t *testing.T) {
	_, err := NewContractManager(context.Background(), nil, nil, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestBroadcastFFI(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	mbm := cm.broadcast.(*broadcastmocks.Manager)

	mdb.On("GetFFI", mock.Anything, "ns1", "", "").Return(nil, nil)
	mim.On("ResolveLocalOrgDID", mock.Anything).Return("firefly:org1/id", nil)
	mim.On("GetOrgKey", mock.Anything).Return("key", nil)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mbm.On("BroadcastDefinition", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.FFI"), mock.AnythingOfType("*fftypes.Identity"), fftypes.SystemTagDefineFFI, false).Return(msg, nil)
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
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mbm := cm.broadcast.(*broadcastmocks.Manager)

	mdb.On("GetFFI", mock.Anything, "ns1", "", "").Return(nil, nil)
	mbi.On("ValidateFFIParam", mock.Anything, mock.AnythingOfType("*fftypes.FFIParam")).Return(fmt.Errorf("pop"))

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mbm.On("BroadcastDefinition", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.FFI"), mock.AnythingOfType("*fftypes.Identity"), fftypes.SystemTagDefineFFI, false).Return(msg, nil)
	ffi := &fftypes.FFI{
		ID: fftypes.NewUUID(),
		Methods: []*fftypes.FFIMethod{
			{
				Name: "sum",
				Params: []*fftypes.FFIParam{
					{
						Name:    "x",
						Type:    "integer",
						Details: []byte(`{"type": "uint256"}`),
					},
				},
			},
		},
	}
	_, err := cm.BroadcastFFI(context.Background(), "ns1", ffi, false)
	assert.Regexp(t, "pop", err)
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
	mbm.On("BroadcastDefinition", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.FFI"), mock.AnythingOfType("*fftypes.Identity"), fftypes.SystemTagDefineFFI, false).Return(msg, nil)
	ffi := &fftypes.FFI{
		ID: fftypes.NewUUID(),
	}
	_, err := cm.BroadcastFFI(context.Background(), "ns1", ffi, false)
	assert.Regexp(t, "FF10302", err)
}

func TestBroadcastFFIResolveOrgFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mbm := cm.broadcast.(*broadcastmocks.Manager)
	mim := cm.identity.(*identitymanagermocks.Manager)

	mdb.On("GetFFI", mock.Anything, "ns1", "", "").Return(nil, nil)
	mim.On("ResolveLocalOrgDID", mock.Anything).Return("", fmt.Errorf("pop"))

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mbm.On("BroadcastDefinition", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.FFI"), mock.AnythingOfType("*fftypes.Identity"), fftypes.SystemTagDefineFFI, false).Return(msg, nil)
	ffi := &fftypes.FFI{
		ID: fftypes.NewUUID(),
	}
	_, err := cm.BroadcastFFI(context.Background(), "ns1", ffi, false)
	assert.Regexp(t, "pop", err)
}

func TestBroadcastFFIFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mbm := cm.broadcast.(*broadcastmocks.Manager)
	mim := cm.identity.(*identitymanagermocks.Manager)

	mdb.On("GetFFI", mock.Anything, "ns1", "", "").Return(nil, nil)
	mim.On("ResolveLocalOrgDID", mock.Anything).Return("firefly:org1/id", nil)
	mim.On("GetOrgKey", mock.Anything).Return("key", nil)

	mbm.On("BroadcastDefinition", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.FFI"), mock.AnythingOfType("*fftypes.Identity"), fftypes.SystemTagDefineFFI, false).Return(nil, fmt.Errorf("pop"))
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
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	mbi.On("ValidateFFIParam", mock.Anything, mock.Anything).Return(nil)
	mbi.On("ValidateInvokeContractRequest", mock.Anything, mock.Anything).Return(nil)

	req := &fftypes.InvokeContractRequest{
		Method: &fftypes.FFIMethod{
			Name: "sum",
			Params: []*fftypes.FFIParam{
				{
					Name:    "x",
					Type:    "integer",
					Details: []byte(`{"type": "uint256"}`),
				},
				{
					Name:    "y",
					Type:    "integer",
					Details: []byte(`{"type": "uint256"}`),
				},
			},
			Returns: []*fftypes.FFIParam{
				{
					Name:    "z",
					Type:    "integer",
					Details: []byte(`{"type": "uint256"}`),
				},
			},
		},
		Params: map[string]interface{}{
			"x": float64(1),
			"y": float64(2),
		},
	}
	err := cm.validateInvokeContractRequest(context.Background(), req)
	assert.NoError(t, err)
}

func TestValidateInvokeContractRequestMissingInput(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	mbi.On("ValidateFFIParam", mock.Anything, mock.Anything).Return(nil)
	mbi.On("ValidateInvokeContractRequest", mock.Anything, mock.Anything).Return(nil)

	req := &fftypes.InvokeContractRequest{
		Method: &fftypes.FFIMethod{
			Name: "sum",
			Params: []*fftypes.FFIParam{
				{
					Name:    "x",
					Type:    "integer",
					Details: []byte(`{"type": "uint256"}`),
				},
				{
					Name:    "y",
					Type:    "integer",
					Details: []byte(`{"type": "uint256"}`),
				},
			},
			Returns: []*fftypes.FFIParam{
				{
					Name:    "z",
					Type:    "integer",
					Details: []byte(`{"type": "uint256"}`),
				},
			},
		},
		Params: map[string]interface{}{
			"x": float64(1),
		},
	}
	err := cm.validateInvokeContractRequest(context.Background(), req)
	assert.Regexp(t, "Missing required input argument 'y'", err)
}

func TestValidateInvokeContractRequestInputWrongType(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	mbi.On("ValidateFFIParam", mock.Anything, mock.Anything).Return(nil)
	mbi.On("ValidateInvokeContractRequest", mock.Anything, mock.Anything).Return(nil)

	req := &fftypes.InvokeContractRequest{
		Method: &fftypes.FFIMethod{
			Name: "sum",
			Params: []*fftypes.FFIParam{
				{
					Name:    "x",
					Type:    "integer",
					Details: []byte(`{"type": "uint256"}`),
				},
				{
					Name:    "y",
					Type:    "integer",
					Details: []byte(`{"type": "uint256"}`),
				},
			},
			Returns: []*fftypes.FFIParam{
				{
					Name:    "z",
					Type:    "integer",
					Details: []byte(`{"type": "uint256"}`),
				},
			},
		},
		Params: map[string]interface{}{
			"x": float64(1),
			"y": "two",
		},
	}
	err := cm.validateInvokeContractRequest(context.Background(), req)
	assert.Regexp(t, "Input.*not expected.*integer", err)
}

func TestValidateInvokeContractRequestInvalidParam(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	mbi.On("ValidateFFIParam", mock.Anything, mock.Anything).Return(errors.New("pop"))

	req := &fftypes.InvokeContractRequest{
		Method: &fftypes.FFIMethod{
			Name: "sum",
			Params: []*fftypes.FFIParam{
				{
					Name:    "x",
					Type:    "integer",
					Details: []byte(`{"type": "uint256"}`),
				},
				{
					Name:    "y",
					Type:    "integer",
					Details: []byte(`{"type": "uint256"}`),
				},
			},
			Returns: []*fftypes.FFIParam{
				{
					Name:    "z",
					Type:    "integer",
					Details: []byte(`{"type": "uint256"}`),
				},
			},
		},
		Params: map[string]interface{}{
			"x": float64(1),
			"y": float64(2),
		},
	}

	err := cm.validateInvokeContractRequest(context.Background(), req)
	assert.Regexp(t, err, "pop")
}

func TestValidateInvokeContractRequestInvalidMethod(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	mbi.On("ValidateFFIParam", mock.Anything, mock.Anything).Return(errors.New("pop"))

	method := &fftypes.FFIMethod{
		Name: "sum",
		Params: []*fftypes.FFIParam{
			{
				Name:    "x",
				Type:    "integer",
				Details: []byte(`{"type": "uint256"}`),
			},
			{
				Name:    "y",
				Type:    "integer",
				Details: []byte(`{"type": "uint256"}`),
			},
		},
		Returns: []*fftypes.FFIParam{
			{
				Name:    "z",
				Type:    "integer",
				Details: []byte(`{"type": "uint256"}`),
			},
		},
	}

	err := cm.validateFFIMethod(context.Background(), method)
	assert.Regexp(t, err, "pop")
}

func TestValidateInvokeContractRequestInvalidEventName(t *testing.T) {
	cm := newTestContractManager()
	event := &fftypes.FFIEventDefinition{}
	err := cm.validateFFIEvent(context.Background(), event)
	assert.Regexp(t, "FF10319", err)
}

func TestValidateInvokeContractRequestInvalidMethodName(t *testing.T) {
	cm := newTestContractManager()
	method := &fftypes.FFIMethod{}
	err := cm.validateFFIMethod(context.Background(), method)
	assert.Regexp(t, "FF10320", err)
}

func TestValidateInvokeContractRequestInvalidEventParams(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	mbi.On("ValidateFFIParam", mock.Anything, mock.Anything).Return(errors.New("pop"))

	method := &fftypes.FFIEventDefinition{
		Name: "sum",
		Params: []*fftypes.FFIParam{
			{
				Name:    "x",
				Type:    "integer",
				Details: []byte(`{"type": "uint256"}`),
			},
			{
				Name:    "y",
				Type:    "integer",
				Details: []byte(`{"type": "uint256"}`),
			},
		},
	}

	err := cm.validateFFIEvent(context.Background(), method)
	assert.Regexp(t, err, "pop")
}

func TestValidateFFI(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	mbi.On("ValidateFFIParam", mock.Anything, mock.Anything).Return(nil)

	ffi := &fftypes.FFI{
		Name:      "math",
		Namespace: "default",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "sum",
				Params: []*fftypes.FFIParam{
					{
						Name:    "x",
						Type:    "integer",
						Details: []byte(`{"type": "uint256"}`),
					},
					{
						Name:    "y",
						Type:    "integer",
						Details: []byte(`{"type": "uint256"}`),
					},
				},
				Returns: []*fftypes.FFIParam{
					{
						Name:    "z",
						Type:    "integer",
						Details: []byte(`{"type": "uint256"}`),
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
							Name:    "z",
							Type:    "integer",
							Details: []byte(`{"type": "uint256"}`),
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

func TestValidateFFIBadMethodParam(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	mbi.On("ValidateFFIParam", mock.Anything, mock.Anything).Once().Return(errors.New("pop"))

	ffi := &fftypes.FFI{
		Name:      "math",
		Namespace: "default",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "sum",
				Params: []*fftypes.FFIParam{
					{
						Name:    "x",
						Type:    "integer",
						Details: []byte(`{"type": "uint256"}`),
					},
					{
						Name:    "y",
						Type:    "integer",
						Details: []byte(`{"type": "uint256"}`),
					},
				},
				Returns: []*fftypes.FFIParam{
					{
						Name:    "z",
						Type:    "integer",
						Details: []byte(`{"type": "uint256"}`),
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
							Name:    "z",
							Type:    "integer",
							Details: []byte(`{"type": "uint256"}`),
						},
					},
				},
			},
		},
	}

	err := cm.ValidateFFIAndSetPathnames(context.Background(), ffi)
	assert.Regexp(t, err, "pop")
}

func TestValidateFFIBadMethodReturnParam(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	mbi.On("ValidateFFIParam", mock.Anything, mock.Anything).Twice().Return(nil)
	mbi.On("ValidateFFIParam", mock.Anything, mock.Anything).Once().Return(errors.New("pop"))

	ffi := &fftypes.FFI{
		Name:      "math",
		Namespace: "default",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "sum",
				Params: []*fftypes.FFIParam{
					{
						Name:    "x",
						Type:    "integer",
						Details: []byte(`{"type": "uint256"}`),
					},
					{
						Name:    "y",
						Type:    "integer",
						Details: []byte(`{"type": "uint256"}`),
					},
				},
				Returns: []*fftypes.FFIParam{
					{
						Name:    "z",
						Type:    "integer",
						Details: []byte(`{"type": "uint256"}`),
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
							Name:    "z",
							Type:    "integer",
							Details: []byte(`{"type": "uint256"}`),
						},
					},
				},
			},
		},
	}

	err := cm.ValidateFFIAndSetPathnames(context.Background(), ffi)
	assert.Regexp(t, err, "pop")
}

func TestValidateFFIBadEventParam(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	mbi.On("ValidateFFIParam", mock.Anything, mock.Anything).Times(3).Return(nil)
	mbi.On("ValidateFFIParam", mock.Anything, mock.Anything).Once().Return(errors.New("pop"))

	ffi := &fftypes.FFI{
		Name:      "math",
		Namespace: "default",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "sum",
				Params: []*fftypes.FFIParam{
					{
						Name:    "x",
						Type:    "integer",
						Details: []byte(`{"type": "uint256"}`),
					},
					{
						Name:    "y",
						Type:    "integer",
						Details: []byte(`{"type": "uint256"}`),
					},
				},
				Returns: []*fftypes.FFIParam{
					{
						Name:    "z",
						Type:    "integer",
						Details: []byte(`{"type": "uint256"}`),
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
							Name:    "z",
							Type:    "integer",
							Details: []byte(`{"type": "uint256"}`),
						},
					},
				},
			},
		},
	}

	err := cm.ValidateFFIAndSetPathnames(context.Background(), ffi)
	assert.Regexp(t, err, "pop")
}

func TestAddContractSubscriptionInline(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Location: fftypes.Byteable(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Event: &fftypes.FFISerializedEvent{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "changed",
					Params: fftypes.FFIParams{
						{
							Name: "value",
							Type: "integer",
						},
					},
				},
			},
		},
	}

	mbi.On("ValidateFFIParam", context.Background(), sub.Event.Params[0]).Return(nil)
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
					Name: "value",
					Type: "integer",
				},
			},
		},
	}

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Location: fftypes.Byteable(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
		},
		EventID: event.ID,
	}

	mbi.On("ValidateFFIParam", context.Background(), mock.Anything).Return(nil)
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
			Location: fftypes.Byteable(fftypes.JSONObject{
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
			Location: fftypes.Byteable(fftypes.JSONObject{
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
			Location: fftypes.Byteable(fftypes.JSONObject{
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
			Location: fftypes.Byteable(fftypes.JSONObject{
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
			Location: fftypes.Byteable(fftypes.JSONObject{
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
			Location: fftypes.Byteable(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Event: &fftypes.FFISerializedEvent{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "changed",
					Params: fftypes.FFIParams{
						{
							Name: "value",
							Type: "integer",
						},
					},
				},
			},
		},
	}

	mbi.On("ValidateFFIParam", context.Background(), sub.Event.Params[0]).Return(fmt.Errorf("pop"))

	_, err := cm.AddContractSubscription(context.Background(), "ns", sub)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionBlockchainFail(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Location: fftypes.Byteable(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Event: &fftypes.FFISerializedEvent{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "changed",
					Params: fftypes.FFIParams{
						{
							Name: "value",
							Type: "integer",
						},
					},
				},
			},
		},
	}

	mbi.On("ValidateFFIParam", context.Background(), sub.Event.Params[0]).Return(nil)
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
			Location: fftypes.Byteable(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Event: &fftypes.FFISerializedEvent{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "changed",
					Params: fftypes.FFIParams{
						{
							Name: "value",
							Type: "integer",
						},
					},
				},
			},
		},
	}

	mbi.On("ValidateFFIParam", context.Background(), sub.Event.Params[0]).Return(nil)
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

	req := &fftypes.InvokeContractRequest{
		Interface: fftypes.NewUUID(),
		Ledger:    []byte{},
		Location:  []byte{},
		Method: &fftypes.FFIMethod{
			Name:    "doStuff",
			ID:      fftypes.NewUUID(),
			Params:  fftypes.FFIParams{},
			Returns: fftypes.FFIParams{},
		},
	}

	mim.On("GetOrgKey", mock.Anything).Return("key", nil)
	mbi.On("InvokeContract", mock.Anything, mock.AnythingOfType("*fftypes.UUID"), "key", mock.Anything, mock.AnythingOfType("*fftypes.FFIMethod"), mock.Anything).Return(struct{}{}, nil)

	_, err := cm.InvokeContract(context.Background(), "ns1", req)

	assert.NoError(t, err)
}

func TestInvokeContractFailResolve(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.InvokeContractRequest{
		Interface: fftypes.NewUUID(),
		Ledger:    []byte{},
		Location:  []byte{},
	}

	mim.On("GetOrgKey", mock.Anything).Return("key", nil)
	mbi.On("InvokeContract", mock.Anything, mock.AnythingOfType("*fftypes.UUID"), "key", mock.Anything, mock.AnythingOfType("*fftypes.FFIMethod"), mock.Anything).Return(struct{}{}, nil)

	_, err := cm.InvokeContract(context.Background(), "ns1", req)

	assert.Regexp(t, "FF10313", err)
}

func TestInvokeContractNoMethodSignature(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.InvokeContractRequest{
		Ledger:   []byte{},
		Location: []byte{},
		Method: &fftypes.FFIMethod{
			Name: "sum",
		},
	}

	mim.On("GetOrgKey", mock.Anything).Return("key", nil)
	mbi.On("InvokeContract", mock.Anything, mock.AnythingOfType("*fftypes.UUID"), "key", mock.Anything, mock.AnythingOfType("*fftypes.FFIMethod"), mock.Anything).Return(struct{}{}, nil)

	_, err := cm.InvokeContract(context.Background(), "ns1", req)

	assert.Regexp(t, "FF10314", err)
}

func TestInvokeContractMethodNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.InvokeContractRequest{
		Interface: fftypes.NewUUID(),
		Ledger:    []byte{},
		Location:  []byte{},
		Method: &fftypes.FFIMethod{
			Name: "sum",
		},
	}

	mim.On("GetOrgKey", mock.Anything).Return("key", nil)
	mdb.On("GetFFIMethod", mock.Anything, "ns1", req.Interface, req.Method.Name).Return(nil, fmt.Errorf("pop"))

	_, err := cm.InvokeContract(context.Background(), "ns1", req)

	assert.Regexp(t, "FF10315", err)
}

func TestInvokeContractMethodBadInput(t *testing.T) {
	cm := newTestContractManager()
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)

	req := &fftypes.InvokeContractRequest{
		Interface: fftypes.NewUUID(),
		Ledger:    []byte{},
		Location:  []byte{},
		Method: &fftypes.FFIMethod{
			Name: "sum",
			Params: fftypes.FFIParams{
				{
					Name:    "x",
					Type:    "integer",
					Details: []byte(`{"type": "uint256"}`),
				},
				{
					Name:    "y",
					Type:    "integer",
					Details: []byte(`{"type": "uint256"}`),
				},
			},
			Returns: fftypes.FFIParams{

				{
					Name:    "sum",
					Type:    "integer",
					Details: []byte(`{"type": "uint256"}`),
				},
			},
		},
	}
	mim.On("GetOrgKey", mock.Anything).Return("key", nil)
	mbi.On("ValidateFFIParam", mock.Anything, mock.AnythingOfType("*fftypes.FFIParam")).Return(nil)

	_, err := cm.InvokeContract(context.Background(), "ns1", req)
	assert.Regexp(t, "FF10304", err)
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
	mdi.On("GetContractEventByID", context.Background(), id).Return(&fftypes.ContractEvent{}, nil)

	_, err := cm.GetContractEventByID(context.Background(), id)
	assert.NoError(t, err)
}

func TestGetContractEvents(t *testing.T) {
	cm := newTestContractManager()
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractEvents", context.Background(), mock.Anything).Return(nil, nil, nil)

	f := database.ContractSubscriptionQueryFactory.NewFilter(context.Background())
	_, _, err := cm.GetContractEvents(context.Background(), "ns", f.And())
	assert.NoError(t, err)
}

func TestInvokeContractAPI(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	req := &fftypes.InvokeContractRequest{
		Interface: fftypes.NewUUID(),
		Ledger:    []byte{},
		Location:  []byte{},
		Method: &fftypes.FFIMethod{
			ID: fftypes.NewUUID(),
		},
	}

	api := &fftypes.ContractAPI{
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
		Location: []byte{},
	}

	mim.On("GetOrgKey", mock.Anything).Return("key", nil)
	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(api, nil)
	mdb.On("GetFFIMethod", mock.Anything, "ns1", mock.Anything, mock.Anything).Return(&fftypes.FFIMethod{Name: "peel"}, nil)
	mbi.On("InvokeContract", mock.Anything, mock.AnythingOfType("*fftypes.UUID"), "key", mock.Anything, mock.AnythingOfType("*fftypes.FFIMethod"), mock.Anything).Return(struct{}{}, nil)

	_, err := cm.InvokeContractAPI(context.Background(), "ns1", "banana", "peel", req)

	assert.NoError(t, err)
}

func TestInvokeContractAPIFailContractLookup(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	req := &fftypes.InvokeContractRequest{
		Interface: fftypes.NewUUID(),
		Ledger:    []byte{},
		Location:  []byte{},
		Method: &fftypes.FFIMethod{
			ID: fftypes.NewUUID(),
		},
	}

	mim.On("GetOrgKey", mock.Anything).Return("key", nil)
	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(nil, fmt.Errorf("pop"))

	_, err := cm.InvokeContractAPI(context.Background(), "ns1", "banana", "peel", req)

	assert.Regexp(t, "pop", err)
}

func TestInvokeContractAPIContractNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	req := &fftypes.InvokeContractRequest{
		Interface: fftypes.NewUUID(),
		Ledger:    []byte{},
		Location:  []byte{},
		Method: &fftypes.FFIMethod{
			ID: fftypes.NewUUID(),
		},
	}

	mim.On("GetOrgKey", mock.Anything).Return("key", nil)
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

func TestGetContractAPISwagger(t *testing.T) {
	cm := newTestContractManager()
	msg := cm.swaggerGen.(*oapiffimocks.FFISwaggerGen)
	mdb := cm.database.(*databasemocks.Plugin)

	cid := fftypes.NewUUID()
	ffi := &fftypes.FFI{
		ID: cid,
	}
	api := &fftypes.ContractAPI{
		ID: fftypes.NewUUID(),
		Interface: &fftypes.FFIReference{
			ID: cid,
		},
	}
	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(api, nil)
	mdb.On("GetFFIByID", mock.Anything, cid).Return(ffi, nil)
	mdb.On("GetFFIMethods", mock.Anything, mock.Anything).Return([]*fftypes.FFIMethod{
		{ID: fftypes.NewUUID(), Name: "method1"},
	}, nil, nil)
	mdb.On("GetFFIEvents", mock.Anything, mock.Anything).Return([]*fftypes.FFIEvent{
		{ID: fftypes.NewUUID(), FFIEventDefinition: fftypes.FFIEventDefinition{Name: "event1"}},
	}, nil, nil)
	msg.On("Generate", mock.Anything, "http://localhost:5000/api/v1/namespaces/ns1/apis/banana", api, ffi).Return(&openapi3.T{
		Info: &openapi3.Info{
			Title: "utapi",
		},
	}, nil)

	swagger, err := cm.GetContractAPISwagger(context.Background(), "http://localhost:5000/api/v1", "ns1", "banana")

	assert.NoError(t, err)
	assert.Equal(t, "utapi", swagger.Info.Title)
	mdb.AssertExpectations(t)
}

func TestGetContractAPISwaggerGenFail(t *testing.T) {
	cm := newTestContractManager()
	msg := cm.swaggerGen.(*oapiffimocks.FFISwaggerGen)
	mdb := cm.database.(*databasemocks.Plugin)

	cid := fftypes.NewUUID()
	ffi := &fftypes.FFI{
		ID: cid,
	}
	api := &fftypes.ContractAPI{
		ID: fftypes.NewUUID(),
		Interface: &fftypes.FFIReference{
			ID: cid,
		},
	}
	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(api, nil)
	mdb.On("GetFFIByID", mock.Anything, cid).Return(ffi, nil)
	mdb.On("GetFFIMethods", mock.Anything, mock.Anything).Return([]*fftypes.FFIMethod{}, nil, nil)
	mdb.On("GetFFIEvents", mock.Anything, mock.Anything).Return([]*fftypes.FFIEvent{}, nil, nil)
	msg.On("Generate", mock.Anything, "http://localhost:5000/api/v1/namespaces/ns1/apis/banana", api, ffi).Return(nil, fmt.Errorf("pop"))

	_, err := cm.GetContractAPISwagger(context.Background(), "http://localhost:5000/api/v1", "ns1", "banana")

	assert.EqualError(t, err, "pop")
	mdb.AssertExpectations(t)
}

func TestGetContractAPISwaggerNotFound(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(nil, nil)

	_, err := cm.GetContractAPISwagger(context.Background(), "http://localhost:5000/api/v1", "ns1", "banana")

	assert.Regexp(t, "FF10143", err)
	mdb.AssertExpectations(t)
}

func TestGetContractAPISwaggerAPIFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(nil, fmt.Errorf("pop"))

	_, err := cm.GetContractAPISwagger(context.Background(), "http://localhost:5000/api/v1", "ns1", "banana")

	assert.EqualError(t, err, "pop")
	mdb.AssertExpectations(t)
}

func TestGetContractAPISwaggerFFIFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	cid := fftypes.NewUUID()
	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(&fftypes.ContractAPI{
		ID: fftypes.NewUUID(),
		Interface: &fftypes.FFIReference{
			ID: cid,
		},
	}, nil)
	mdb.On("GetFFIByID", mock.Anything, cid).Return(nil, fmt.Errorf("pop"))

	_, err := cm.GetContractAPISwagger(context.Background(), "http://localhost:5000/api/v1", "ns1", "banana")

	assert.EqualError(t, err, "pop")
	mdb.AssertExpectations(t)
}

func TestBroadcastContractAPI(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	mbm := cm.broadcast.(*broadcastmocks.Manager)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Ledger:    []byte{},
		Location:  []byte{},
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(&fftypes.FFI{}, nil)
	mim.On("ResolveLocalOrgDID", mock.Anything).Return("firefly:org1/id", nil)
	mim.On("GetOrgKey", mock.Anything).Return("key", nil)
	mbm.On("BroadcastDefinition", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.ContractAPI"), mock.AnythingOfType("*fftypes.Identity"), fftypes.SystemTagDefineContractAPI, false).Return(msg, nil)
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.NoError(t, err)
}

func TestBroadcastContractAPIInterfaceName(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	mbm := cm.broadcast.(*broadcastmocks.Manager)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Ledger:    []byte{},
		Location:  []byte{},
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			Name:    "my-ffi",
			Version: "1",
		},
	}
	interfaceID := fftypes.NewUUID()
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFI", mock.Anything, "ns1", "my-ffi", "1").Return(&fftypes.FFI{ID: interfaceID}, nil)
	mim.On("ResolveLocalOrgDID", mock.Anything).Return("firefly:org1/id", nil)
	mim.On("GetOrgKey", mock.Anything).Return("key", nil)
	mbm.On("BroadcastDefinition", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.ContractAPI"), mock.AnythingOfType("*fftypes.Identity"), fftypes.SystemTagDefineContractAPI, false).Return(msg, nil)
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.NoError(t, err)
	assert.Equal(t, *interfaceID, *api.Interface.ID)
}

func TestBroadcastContractAPIExisting(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Ledger:    []byte{},
		Location:  []byte{},
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(&fftypes.ContractAPI{}, nil)
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.Regexp(t, "FF10316", err)
}

func TestBroadcastContractAPIResolveLocalOrgFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Ledger:    []byte{},
		Location:  []byte{},
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(&fftypes.FFI{}, nil)
	mim.On("ResolveLocalOrgDID", mock.Anything).Return("", fmt.Errorf("pop"))
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.Regexp(t, "pop", err)
}

func TestBroadcastContractAPIFail(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)
	mim := cm.identity.(*identitymanagermocks.Manager)
	mbm := cm.broadcast.(*broadcastmocks.Manager)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Ledger:    []byte{},
		Location:  []byte{},
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
	}
	mdb.On("GetContractAPIByName", mock.Anything, api.Namespace, api.Name).Return(nil, nil)
	mdb.On("GetFFIByID", mock.Anything, api.Interface.ID).Return(&fftypes.FFI{}, nil)
	mim.On("ResolveLocalOrgDID", mock.Anything).Return("firefly:org1/id", nil)
	mim.On("GetOrgKey", mock.Anything).Return("key", nil)
	mbm.On("BroadcastDefinition", mock.Anything, "ns1", mock.AnythingOfType("*fftypes.ContractAPI"), mock.AnythingOfType("*fftypes.Identity"), fftypes.SystemTagDefineContractAPI, false).Return(nil, fmt.Errorf("pop"))
	_, err := cm.BroadcastContractAPI(context.Background(), "ns1", api, false)
	assert.Regexp(t, "pop", err)
}

func TestBroadcastContractAPINoInterface(t *testing.T) {
	cm := newTestContractManager()
	mdb := cm.database.(*databasemocks.Plugin)

	api := &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Ledger:    []byte{},
		Location:  []byte{},
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
		Ledger:    []byte{},
		Location:  []byte{},
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
		Ledger:    []byte{},
		Location:  []byte{},
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
		Ledger:    []byte{},
		Location:  []byte{},
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
		Ledger:    []byte{},
		Location:  []byte{},
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
		Ledger:    []byte{},
		Location:  []byte{},
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
		Location: []byte{'a', 'b', 'c'},
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
		Location: []byte{'a', 'b', 'c'},
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
		Location: []byte{'a', 'b', 'c'},
	}

	mdb.On("GetContractAPIByName", mock.Anything, "ns1", "banana").Return(api, nil)
	mdb.On("GetFFIEvent", mock.Anything, "ns1", api.Interface.ID, "peeled").Return(nil, fmt.Errorf("pop"))

	_, err := cm.SubscribeContractAPI(context.Background(), "ns1", "banana", "peeled", req)

	assert.Regexp(t, "FF10321", err)
}
