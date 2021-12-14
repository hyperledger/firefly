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

	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/publicstoragemocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestContractManager(t *testing.T) *contractManager {
	mdi := &databasemocks.Plugin{}
	mps := &publicstoragemocks.Plugin{}
	mbm := &broadcastmocks.Manager{}
	mbi := &blockchainmocks.Plugin{}

	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}

	return NewContractManager(mdi, mps, mbm, nil, mbi).(*contractManager)
}

func TestValidateInvokeContractRequest(t *testing.T) {
	cm := newTestContractManager(t)
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
					Details: []byte("\"type\":\"uint256\"}"),
				},
				{
					Name:    "y",
					Type:    "integer",
					Details: []byte("\"type\":\"uint256\"}"),
				},
			},
			Returns: []*fftypes.FFIParam{
				{
					Name:    "z",
					Type:    "integer",
					Details: []byte("\"type\":\"uint256\"}"),
				},
			},
		},
		Params: map[string]interface{}{
			"x": float64(1),
			"y": float64(2),
		},
	}
	err := cm.ValidateInvokeContractRequest(context.Background(), req)
	assert.NoError(t, err)
}

func TestValidateInvokeContractRequestMissingInput(t *testing.T) {
	cm := newTestContractManager(t)
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
					Details: []byte("\"type\":\"uint256\"}"),
				},
				{
					Name:    "y",
					Type:    "integer",
					Details: []byte("\"type\":\"uint256\"}"),
				},
			},
			Returns: []*fftypes.FFIParam{
				{
					Name:    "z",
					Type:    "integer",
					Details: []byte("\"type\":\"uint256\"}"),
				},
			},
		},
		Params: map[string]interface{}{
			"x": float64(1),
		},
	}
	err := cm.ValidateInvokeContractRequest(context.Background(), req)
	assert.Regexp(t, "Missing required input argument 'y'", err)
}

func TestValidateInvokeContractRequestInputWrongType(t *testing.T) {
	cm := newTestContractManager(t)
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
					Details: []byte("\"type\":\"uint256\"}"),
				},
				{
					Name:    "y",
					Type:    "integer",
					Details: []byte("\"type\":\"uint256\"}"),
				},
			},
			Returns: []*fftypes.FFIParam{
				{
					Name:    "z",
					Type:    "integer",
					Details: []byte("\"type\":\"uint256\"}"),
				},
			},
		},
		Params: map[string]interface{}{
			"x": float64(1),
			"y": "two",
		},
	}
	err := cm.ValidateInvokeContractRequest(context.Background(), req)
	assert.Regexp(t, "Input.*not expected.*integer", err)
}

func TestValidateInvokeContractRequestInvalidParam(t *testing.T) {
	cm := newTestContractManager(t)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	mbi.On("ValidateFFIParam", mock.Anything, mock.Anything).Return(errors.New("pop"))

	req := &fftypes.InvokeContractRequest{
		Method: &fftypes.FFIMethod{
			Name: "sum",
			Params: []*fftypes.FFIParam{
				{
					Name:    "x",
					Type:    "integer",
					Details: []byte("\"type\":\"uint256\"}"),
				},
				{
					Name:    "y",
					Type:    "integer",
					Details: []byte("\"type\":\"uint256\"}"),
				},
			},
			Returns: []*fftypes.FFIParam{
				{
					Name:    "z",
					Type:    "integer",
					Details: []byte("\"type\":\"uint256\"}"),
				},
			},
		},
		Params: map[string]interface{}{
			"x": float64(1),
			"y": float64(2),
		},
	}

	err := cm.ValidateInvokeContractRequest(context.Background(), req)
	assert.Regexp(t, err, "pop")
}

func TestValidateInvokeContractRequestInvalidMethod(t *testing.T) {
	cm := newTestContractManager(t)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	mbi.On("ValidateFFIParam", mock.Anything, mock.Anything).Return(errors.New("pop"))

	method := &fftypes.FFIMethod{
		Name: "sum",
		Params: []*fftypes.FFIParam{
			{
				Name:    "x",
				Type:    "integer",
				Details: []byte("\"type\":\"uint256\"}"),
			},
			{
				Name:    "y",
				Type:    "integer",
				Details: []byte("\"type\":\"uint256\"}"),
			},
		},
		Returns: []*fftypes.FFIParam{
			{
				Name:    "z",
				Type:    "integer",
				Details: []byte("\"type\":\"uint256\"}"),
			},
		},
	}

	err := cm.validateFFIMethod(context.Background(), method)
	assert.Regexp(t, err, "pop")
}

func TestValidateInvokeContractRequestInvalidEvent(t *testing.T) {
	cm := newTestContractManager(t)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)

	mbi.On("ValidateFFIParam", mock.Anything, mock.Anything).Return(errors.New("pop"))

	method := &fftypes.FFIEvent{
		Name: "sum",
		Params: []*fftypes.FFIParam{
			{
				Name:    "x",
				Type:    "integer",
				Details: []byte("\"type\":\"uint256\"}"),
			},
			{
				Name:    "y",
				Type:    "integer",
				Details: []byte("\"type\":\"uint256\"}"),
			},
		},
	}

	err := cm.validateFFIEvent(context.Background(), method)
	assert.Regexp(t, err, "pop")
}

func TestValidateFFI(t *testing.T) {
	cm := newTestContractManager(t)
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
						Details: []byte("\"type\":\"uint256\"}"),
					},
					{
						Name:    "y",
						Type:    "integer",
						Details: []byte("\"type\":\"uint256\"}"),
					},
				},
				Returns: []*fftypes.FFIParam{
					{
						Name:    "z",
						Type:    "integer",
						Details: []byte("\"type\":\"uint256\"}"),
					},
				},
			},
		},
		Events: []*fftypes.FFIEvent{
			{
				Name: "sum",
				Params: []*fftypes.FFIParam{
					{
						Name:    "z",
						Type:    "integer",
						Details: []byte("\"type\":\"uint256\"}"),
					},
				},
			},
		},
	}

	err := cm.ValidateFFI(context.Background(), ffi)
	assert.NoError(t, err)
}

func TestValidateFFIBadMethodParam(t *testing.T) {
	cm := newTestContractManager(t)
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
						Details: []byte("\"type\":\"uint256\"}"),
					},
					{
						Name:    "y",
						Type:    "integer",
						Details: []byte("\"type\":\"uint256\"}"),
					},
				},
				Returns: []*fftypes.FFIParam{
					{
						Name:    "z",
						Type:    "integer",
						Details: []byte("\"type\":\"uint256\"}"),
					},
				},
			},
		},
		Events: []*fftypes.FFIEvent{
			{
				Name: "sum",
				Params: []*fftypes.FFIParam{
					{
						Name:    "z",
						Type:    "integer",
						Details: []byte("\"type\":\"uint256\"}"),
					},
				},
			},
		},
	}

	err := cm.ValidateFFI(context.Background(), ffi)
	assert.Regexp(t, err, "pop")
}

func TestValidateFFIBadMethodReturnParam(t *testing.T) {
	cm := newTestContractManager(t)
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
						Details: []byte("\"type\":\"uint256\"}"),
					},
					{
						Name:    "y",
						Type:    "integer",
						Details: []byte("\"type\":\"uint256\"}"),
					},
				},
				Returns: []*fftypes.FFIParam{
					{
						Name:    "z",
						Type:    "integer",
						Details: []byte("\"type\":\"uint256\"}"),
					},
				},
			},
		},
		Events: []*fftypes.FFIEvent{
			{
				Name: "sum",
				Params: []*fftypes.FFIParam{
					{
						Name:    "z",
						Type:    "integer",
						Details: []byte("\"type\":\"uint256\"}"),
					},
				},
			},
		},
	}

	err := cm.ValidateFFI(context.Background(), ffi)
	assert.Regexp(t, err, "pop")
}

func TestValidateFFIBadEventParam(t *testing.T) {
	cm := newTestContractManager(t)
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
						Details: []byte("\"type\":\"uint256\"}"),
					},
					{
						Name:    "y",
						Type:    "integer",
						Details: []byte("\"type\":\"uint256\"}"),
					},
				},
				Returns: []*fftypes.FFIParam{
					{
						Name:    "z",
						Type:    "integer",
						Details: []byte("\"type\":\"uint256\"}"),
					},
				},
			},
		},
		Events: []*fftypes.FFIEvent{
			{
				Name: "sum",
				Params: []*fftypes.FFIParam{
					{
						Name:    "z",
						Type:    "integer",
						Details: []byte("\"type\":\"uint256\"}"),
					},
				},
			},
		},
	}

	err := cm.ValidateFFI(context.Background(), ffi)
	assert.Regexp(t, err, "pop")
}

func TestAddContractSubscription(t *testing.T) {
	cm := newTestContractManager(t)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Location: fftypes.Byteable(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
		},
		Event: fftypes.FFIEvent{
			Name: "changed",
			Params: fftypes.FFIParams{
				{
					Name: "value",
					Type: "integer",
				},
			},
		},
	}

	mbi.On("ValidateFFIParam", context.Background(), sub.Event.Params[0]).Return(nil)
	mbi.On("AddSubscription", context.Background(), sub).Return(nil)
	mdi.On("UpsertFFIEvent", context.Background(), "ns", (*fftypes.UUID)(nil), &sub.Event).Return(nil)
	mdi.On("UpsertContractSubscription", context.Background(), &sub.ContractSubscription).Return(nil)

	result, err := cm.AddContractSubscription(context.Background(), "ns", sub)
	assert.NoError(t, err)
	assert.NotNil(t, result.ID)
	assert.NotNil(t, result.Event)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionBadNamespace(t *testing.T) {
	cm := newTestContractManager(t)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{}

	_, err := cm.AddContractSubscription(context.Background(), "!bad", sub)
	assert.Regexp(t, "FF10131.*'namespace'", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionBadName(t *testing.T) {
	cm := newTestContractManager(t)
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
	cm := newTestContractManager(t)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Name: "sub1",
			Location: fftypes.Byteable(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
		},
		Event: fftypes.FFIEvent{
			Name: "changed",
			Params: fftypes.FFIParams{
				{
					Name: "value",
					Type: "integer",
				},
			},
		},
	}

	mdi.On("GetContractSubscription", context.Background(), "ns", "sub1").Return(&fftypes.ContractSubscription{}, nil)

	_, err := cm.AddContractSubscription(context.Background(), "ns", sub)
	assert.Regexp(t, "FF10304", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionNameError(t *testing.T) {
	cm := newTestContractManager(t)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Name: "sub1",
			Location: fftypes.Byteable(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
		},
		Event: fftypes.FFIEvent{
			Name: "changed",
			Params: fftypes.FFIParams{
				{
					Name: "value",
					Type: "integer",
				},
			},
		},
	}

	mdi.On("GetContractSubscription", context.Background(), "ns", "sub1").Return(nil, fmt.Errorf("pop"))

	_, err := cm.AddContractSubscription(context.Background(), "ns", sub)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionValidateFail(t *testing.T) {
	cm := newTestContractManager(t)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Location: fftypes.Byteable(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
		},
		Event: fftypes.FFIEvent{
			Name: "changed",
			Params: fftypes.FFIParams{
				{
					Name: "value",
					Type: "integer",
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
	cm := newTestContractManager(t)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Location: fftypes.Byteable(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
		},
		Event: fftypes.FFIEvent{
			Name: "changed",
			Params: fftypes.FFIParams{
				{
					Name: "value",
					Type: "integer",
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

func TestAddContractSubscriptionUpsertEventFail(t *testing.T) {
	cm := newTestContractManager(t)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Location: fftypes.Byteable(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
		},
		Event: fftypes.FFIEvent{
			Name: "changed",
			Params: fftypes.FFIParams{
				{
					Name: "value",
					Type: "integer",
				},
			},
		},
	}

	mbi.On("ValidateFFIParam", context.Background(), sub.Event.Params[0]).Return(nil)
	mbi.On("AddSubscription", context.Background(), sub).Return(nil)
	mdi.On("UpsertFFIEvent", context.Background(), "ns", (*fftypes.UUID)(nil), &sub.Event).Return(fmt.Errorf("pop"))

	_, err := cm.AddContractSubscription(context.Background(), "ns", sub)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestAddContractSubscriptionUpsertSubFail(t *testing.T) {
	cm := newTestContractManager(t)
	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Location: fftypes.Byteable(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
		},
		Event: fftypes.FFIEvent{
			Name: "changed",
			Params: fftypes.FFIParams{
				{
					Name: "value",
					Type: "integer",
				},
			},
		},
	}

	mbi.On("ValidateFFIParam", context.Background(), sub.Event.Params[0]).Return(nil)
	mbi.On("AddSubscription", context.Background(), sub).Return(nil)
	mdi.On("UpsertFFIEvent", context.Background(), "ns", (*fftypes.UUID)(nil), &sub.Event).Return(nil)
	mdi.On("UpsertContractSubscription", context.Background(), &sub.ContractSubscription).Return(fmt.Errorf("pop"))

	_, err := cm.AddContractSubscription(context.Background(), "ns", sub)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestGetContractSubscriptionByID(t *testing.T) {
	cm := newTestContractManager(t)
	mdi := cm.database.(*databasemocks.Plugin)

	id := fftypes.NewUUID()
	mdi.On("GetContractSubscriptionByID", context.Background(), id).Return(&fftypes.ContractSubscription{}, nil)

	_, err := cm.GetContractSubscriptionByNameOrID(context.Background(), "ns", id.String())
	assert.NoError(t, err)
}

func TestGetContractSubscriptionByIDFail(t *testing.T) {
	cm := newTestContractManager(t)
	mdi := cm.database.(*databasemocks.Plugin)

	id := fftypes.NewUUID()
	mdi.On("GetContractSubscriptionByID", context.Background(), id).Return(nil, fmt.Errorf("pop"))

	_, err := cm.GetContractSubscriptionByNameOrID(context.Background(), "ns", id.String())
	assert.EqualError(t, err, "pop")
}

func TestGetContractSubscriptionByName(t *testing.T) {
	cm := newTestContractManager(t)
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractSubscription", context.Background(), "ns", "sub1").Return(&fftypes.ContractSubscription{}, nil)

	_, err := cm.GetContractSubscriptionByNameOrID(context.Background(), "ns", "sub1")
	assert.NoError(t, err)
}

func TestGetContractSubscriptionBadName(t *testing.T) {
	cm := newTestContractManager(t)

	_, err := cm.GetContractSubscriptionByNameOrID(context.Background(), "ns", "!bad")
	assert.Regexp(t, "FF10131", err)
}

func TestGetContractSubscriptionByNameFail(t *testing.T) {
	cm := newTestContractManager(t)
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractSubscription", context.Background(), "ns", "sub1").Return(nil, fmt.Errorf("pop"))

	_, err := cm.GetContractSubscriptionByNameOrID(context.Background(), "ns", "sub1")
	assert.EqualError(t, err, "pop")
}

func TestGetContractSubscriptionNotFound(t *testing.T) {
	cm := newTestContractManager(t)
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractSubscription", context.Background(), "ns", "sub1").Return(nil, nil)

	_, err := cm.GetContractSubscriptionByNameOrID(context.Background(), "ns", "sub1")
	assert.Regexp(t, "FF10109", err)
}

func TestGetContractSubscriptions(t *testing.T) {
	cm := newTestContractManager(t)
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractSubscriptions", context.Background(), mock.Anything).Return(nil, nil, nil)

	f := database.ContractSubscriptionQueryFactory.NewFilter(context.Background())
	_, _, err := cm.GetContractSubscriptions(context.Background(), "ns", f.And())
	assert.NoError(t, err)
}

func TestDeleteContractSubscription(t *testing.T) {
	cm := newTestContractManager(t)
	mdi := cm.database.(*databasemocks.Plugin)

	sub := &fftypes.ContractSubscription{
		ID: fftypes.NewUUID(),
	}

	mdi.On("GetContractSubscription", context.Background(), "ns", "sub1").Return(sub, nil)
	mdi.On("DeleteContractSubscriptionByID", context.Background(), sub.ID).Return(nil)

	err := cm.DeleteContractSubscriptionByNameOrID(context.Background(), "ns", "sub1")
	assert.NoError(t, err)
}

func TestDeleteContractSubscriptionNotFound(t *testing.T) {
	cm := newTestContractManager(t)
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractSubscription", context.Background(), "ns", "sub1").Return(nil, nil)

	err := cm.DeleteContractSubscriptionByNameOrID(context.Background(), "ns", "sub1")
	assert.Regexp(t, "FF10109", err)
}

func TestGetContractEventByID(t *testing.T) {
	cm := newTestContractManager(t)
	mdi := cm.database.(*databasemocks.Plugin)

	id := fftypes.NewUUID()
	mdi.On("GetContractEventByID", context.Background(), id).Return(&fftypes.ContractEvent{}, nil)

	_, err := cm.GetContractEventByID(context.Background(), id)
	assert.NoError(t, err)
}

func TestGetContractEvents(t *testing.T) {
	cm := newTestContractManager(t)
	mdi := cm.database.(*databasemocks.Plugin)

	mdi.On("GetContractEvents", context.Background(), mock.Anything).Return(nil, nil, nil)

	f := database.ContractSubscriptionQueryFactory.NewFilter(context.Background())
	_, _, err := cm.GetContractEvents(context.Background(), "ns", f.And())
	assert.NoError(t, err)
}
