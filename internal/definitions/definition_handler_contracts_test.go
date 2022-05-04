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

package definitions

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/contractmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testFFI() *fftypes.FFI {
	return &fftypes.FFI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "math",
		Version:   "v1.0.0",
		Methods: []*fftypes.FFIMethod{
			{
				Name: "sum",
				Params: fftypes.FFIParams{
					{
						Name:   "x",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer"}`),
					},
					{
						Name:   "y",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer"}`),
					},
				},
				Returns: fftypes.FFIParams{
					{
						Name:   "result",
						Schema: fftypes.JSONAnyPtr(`{"type": "integer"}`),
					},
				},
			},
		},
		Events: []*fftypes.FFIEvent{
			{
				ID: fftypes.NewUUID(),
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "event1",
					Params: fftypes.FFIParams{
						{
							Name:   "result",
							Schema: fftypes.JSONAnyPtr(`{"type": "integer"}`),
						},
					},
				},
			},
		},
	}
}

func testContractAPI() *fftypes.ContractAPI {
	return &fftypes.ContractAPI{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "math",
		Interface: &fftypes.FFIReference{
			ID: fftypes.NewUUID(),
		},
		Location: fftypes.JSONAnyPtr(""),
	}
}

func TestHandleFFIBroadcastOk(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)

	b, err := json.Marshal(testFFI())
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("UpsertFFI", mock.Anything, mock.Anything).Return(nil)
	mbi.On("UpsertFFIMethod", mock.Anything, mock.Anything).Return(nil)
	mbi.On("UpsertFFIEvent", mock.Anything, mock.Anything).Return(nil)
	mbi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ValidateFFIAndSetPathnames", mock.Anything, mock.Anything).Return(nil)

	action, err := dh.HandleDefinitionBroadcast(context.Background(), bs, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: fftypes.SystemTagDefineFFI,
		},
	}, fftypes.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionConfirm}, action)
	assert.NoError(t, err)
	err = bs.finalizers[0](context.Background())
	assert.NoError(t, err)
	mbi.AssertExpectations(t)
}

func TestPersistFFIValidateFFIFail(t *testing.T) {
	dh, _ := newTestDefinitionHandler(t)
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ValidateFFIAndSetPathnames", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	valid, err := dh.persistFFI(context.Background(), testFFI())
	assert.NoError(t, err)
	assert.False(t, valid)
	mcm.AssertExpectations(t)
}

func TestHandleFFIBroadcastReject(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	mbi := dh.database.(*databasemocks.Plugin)
	mcm := dh.contracts.(*contractmocks.Manager)
	mbi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	mcm.On("ValidateFFIAndSetPathnames", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	action, err := dh.handleFFIBroadcast(context.Background(), bs, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: fftypes.SystemTagDefineFFI,
		},
	}, fftypes.DataArray{}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.NoError(t, err)
	bs.assertNoFinalizers()
}

func TestPersistFFIUpsertFFIFail(t *testing.T) {
	dh, _ := newTestDefinitionHandler(t)
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("UpsertFFI", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ValidateFFIAndSetPathnames", mock.Anything, mock.Anything).Return(nil)
	_, err := dh.persistFFI(context.Background(), testFFI())
	assert.Regexp(t, "pop", err)
	mbi.AssertExpectations(t)
	mcm.AssertExpectations(t)
}

func TestPersistFFIUpsertFFIMethodFail(t *testing.T) {
	dh, _ := newTestDefinitionHandler(t)
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("UpsertFFI", mock.Anything, mock.Anything).Return(nil)
	mbi.On("UpsertFFIMethod", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ValidateFFIAndSetPathnames", mock.Anything, mock.Anything).Return(nil)
	_, err := dh.persistFFI(context.Background(), testFFI())
	assert.Regexp(t, "pop", err)
	mbi.AssertExpectations(t)
	mcm.AssertExpectations(t)
}

func TestPersistFFIUpsertFFIEventFail(t *testing.T) {
	dh, _ := newTestDefinitionHandler(t)
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("UpsertFFI", mock.Anything, mock.Anything).Return(nil)
	mbi.On("UpsertFFIMethod", mock.Anything, mock.Anything).Return(nil)
	mbi.On("UpsertFFIEvent", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ValidateFFIAndSetPathnames", mock.Anything, mock.Anything).Return(nil)
	_, err := dh.persistFFI(context.Background(), testFFI())
	assert.Regexp(t, "pop", err)
	mbi.AssertExpectations(t)
	mcm.AssertExpectations(t)
}

func TestHandleFFIBroadcastValidateFail(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ffi := testFFI()
	ffi.Name = "*%^!$%^&*"
	b, err := json.Marshal(ffi)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleDefinitionBroadcast(context.Background(), bs, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: fftypes.SystemTagDefineFFI,
		},
	}, fftypes.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.NoError(t, err)
	bs.assertNoFinalizers()
}

func TestHandleFFIBroadcastPersistFail(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ffi := testFFI()
	b, err := json.Marshal(ffi)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}
	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("UpsertFFI", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ValidateFFIAndSetPathnames", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleDefinitionBroadcast(context.Background(), bs, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: fftypes.SystemTagDefineFFI,
		},
	}, fftypes.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)
	bs.assertNoFinalizers()
}

func TestHandleContractAPIBroadcastOk(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)

	b, err := json.Marshal(testFFI())
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("UpsertContractAPI", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mbi.On("GetContractAPIByName", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	mbi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleDefinitionBroadcast(context.Background(), bs, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: fftypes.SystemTagDefineContractAPI,
		},
	}, fftypes.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionConfirm}, action)
	assert.NoError(t, err)
	err = bs.finalizers[0](context.Background())
	assert.NoError(t, err)
	mbi.AssertExpectations(t)
}

func TestPersistContractAPIGetFail(t *testing.T) {
	dh, _ := newTestDefinitionHandler(t)
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("GetContractAPIByName", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	_, err := dh.persistContractAPI(context.Background(), testContractAPI())
	assert.Regexp(t, "pop", err)
	mbi.AssertExpectations(t)
}

func TestPersistContractAPIDifferentLocation(t *testing.T) {
	existing := testContractAPI()
	existing.Location = fftypes.JSONAnyPtr(`{"existing": true}`)
	dh, _ := newTestDefinitionHandler(t)
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("GetContractAPIByName", mock.Anything, mock.Anything, mock.Anything).Return(existing, nil)
	valid, err := dh.persistContractAPI(context.Background(), testContractAPI())
	assert.False(t, valid)
	assert.NoError(t, err)
	mbi.AssertExpectations(t)
}

func TestPersistContractAPIUpsertFail(t *testing.T) {
	dh, _ := newTestDefinitionHandler(t)
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("GetContractAPIByName", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	mbi.On("UpsertContractAPI", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	_, err := dh.persistContractAPI(context.Background(), testContractAPI())
	assert.Regexp(t, "pop", err)
	mbi.AssertExpectations(t)
}

func TestPersistContractAPIIDMismatch(t *testing.T) {
	dh, _ := newTestDefinitionHandler(t)
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("GetContractAPIByName", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	mbi.On("UpsertContractAPI", mock.Anything, mock.Anything, mock.Anything).Return(database.IDMismatch)
	valid, err := dh.persistContractAPI(context.Background(), testContractAPI())
	assert.False(t, valid)
	assert.NoError(t, err)
	mbi.AssertExpectations(t)
}

func TestHandleContractAPIBroadcastValidateFail(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	api := testContractAPI()
	api.Name = "*%^!$%^&*"
	b, err := json.Marshal(api)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleDefinitionBroadcast(context.Background(), bs, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: fftypes.SystemTagDefineContractAPI,
		},
	}, fftypes.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.NoError(t, err)
	bs.assertNoFinalizers()
}

func TestHandleContractAPIBroadcastPersistFail(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ffi := testFFI()
	b, err := json.Marshal(ffi)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("GetContractAPIByName", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	mbi.On("UpsertContractAPI", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mbi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleDefinitionBroadcast(context.Background(), bs, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: fftypes.SystemTagDefineContractAPI,
		},
	}, fftypes.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)
	bs.assertNoFinalizers()
}
