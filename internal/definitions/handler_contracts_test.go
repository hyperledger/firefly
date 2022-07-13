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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/contractmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
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

func testContractAPI() *core.ContractAPI {
	return &core.ContractAPI{
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
	data := &core.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("UpsertFFI", mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertFFIMethod", mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertFFIEvent", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ResolveFFI", mock.Anything, mock.Anything).Return(nil)

	action, err := dh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, &core.Message{
		Header: core.MessageHeader{
			Tag: core.SystemTagDefineFFI,
		},
	}, core.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionConfirm}, action)
	assert.NoError(t, err)
	err = bs.RunFinalize(context.Background())
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestPersistFFIValidateFFIFail(t *testing.T) {
	dh, _ := newTestDefinitionHandler(t)
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ResolveFFI", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	retry, err := dh.persistFFI(context.Background(), testFFI())
	assert.Regexp(t, "FF10403", err)
	assert.False(t, retry)
	mcm.AssertExpectations(t)
}

func TestHandleFFIBroadcastReject(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	mdi := dh.database.(*databasemocks.Plugin)
	mcm := dh.contracts.(*contractmocks.Manager)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	mcm.On("ResolveFFI", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	action, err := dh.handleFFIBroadcast(context.Background(), &bs.BatchState, &core.Message{
		Header: core.MessageHeader{
			Tag: core.SystemTagDefineFFI,
		},
	}, core.DataArray{}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.Error(t, err)
	bs.assertNoFinalizers()
}

func TestPersistFFIUpsertFFIFail(t *testing.T) {
	dh, _ := newTestDefinitionHandler(t)
	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("UpsertFFI", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ResolveFFI", mock.Anything, mock.Anything).Return(nil)
	retry, err := dh.persistFFI(context.Background(), testFFI())
	assert.Regexp(t, "pop", err)
	assert.True(t, retry)
	mdi.AssertExpectations(t)
	mcm.AssertExpectations(t)
}

func TestPersistFFIUpsertFFIMethodFail(t *testing.T) {
	dh, _ := newTestDefinitionHandler(t)
	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("UpsertFFI", mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertFFIMethod", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ResolveFFI", mock.Anything, mock.Anything).Return(nil)
	retry, err := dh.persistFFI(context.Background(), testFFI())
	assert.Regexp(t, "pop", err)
	assert.True(t, retry)
	mdi.AssertExpectations(t)
	mcm.AssertExpectations(t)
}

func TestPersistFFIUpsertFFIEventFail(t *testing.T) {
	dh, _ := newTestDefinitionHandler(t)
	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("UpsertFFI", mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertFFIMethod", mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertFFIEvent", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ResolveFFI", mock.Anything, mock.Anything).Return(nil)
	retry, err := dh.persistFFI(context.Background(), testFFI())
	assert.Regexp(t, "pop", err)
	assert.True(t, retry)
	mdi.AssertExpectations(t)
	mcm.AssertExpectations(t)
}

func TestHandleFFIBroadcastValidateFail(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ffi := testFFI()
	ffi.Name = "*%^!$%^&*"
	b, err := json.Marshal(ffi)
	assert.NoError(t, err)
	data := &core.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}
	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, &core.Message{
		Header: core.MessageHeader{
			Tag: core.SystemTagDefineFFI,
		},
	}, core.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.Error(t, err)
	bs.assertNoFinalizers()
}

func TestHandleFFIBroadcastPersistFail(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ffi := testFFI()
	b, err := json.Marshal(ffi)
	assert.NoError(t, err)
	data := &core.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}
	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("UpsertFFI", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ResolveFFI", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, &core.Message{
		Header: core.MessageHeader{
			Tag: core.SystemTagDefineFFI,
		},
	}, core.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)
	bs.assertNoFinalizers()

	mdi.AssertExpectations(t)
	mcm.AssertExpectations(t)
}

func TestHandleFFIBroadcastResolveFail(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ffi := testFFI()
	b, err := json.Marshal(ffi)
	assert.NoError(t, err)
	data := &core.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ResolveFFI", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	action, err := dh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, &core.Message{
		Header: core.MessageHeader{
			Tag: core.SystemTagDefineFFI,
		},
	}, core.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.Regexp(t, "pop", err)
	bs.assertNoFinalizers()

	mcm.AssertExpectations(t)
}

func TestHandleContractAPIBroadcastOk(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)

	b, err := json.Marshal(testFFI())
	assert.NoError(t, err)
	data := &core.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("UpsertContractAPI", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ResolveContractAPI", context.Background(), "", mock.Anything).Return(nil)

	action, err := dh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, &core.Message{
		Header: core.MessageHeader{
			Tag: core.SystemTagDefineContractAPI,
		},
	}, core.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionConfirm}, action)
	assert.NoError(t, err)
	err = bs.RunFinalize(context.Background())
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mcm.AssertExpectations(t)
}

func TestHandleContractAPIBadPayload(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	data := &core.Data{
		Value: fftypes.JSONAnyPtr("bad"),
	}

	action, err := dh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, &core.Message{
		Header: core.MessageHeader{
			Tag: core.SystemTagDefineContractAPI,
		},
	}, core.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.Regexp(t, "FF10400", err)
}

func TestHandleContractAPIIDMismatch(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)

	b, err := json.Marshal(testFFI())
	assert.NoError(t, err)
	data := &core.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("UpsertContractAPI", mock.Anything, mock.Anything, mock.Anything).Return(database.IDMismatch)
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ResolveContractAPI", context.Background(), "", mock.Anything).Return(nil)

	action, err := dh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, &core.Message{
		Header: core.MessageHeader{
			Tag: core.SystemTagDefineContractAPI,
		},
	}, core.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.Regexp(t, "FF10404", err)

	mdi.AssertExpectations(t)
	mcm.AssertExpectations(t)
}

func TestPersistContractAPIUpsertFail(t *testing.T) {
	dh, _ := newTestDefinitionHandler(t)
	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("UpsertContractAPI", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ResolveContractAPI", context.Background(), "http://test", mock.Anything).Return(nil)

	_, err := dh.persistContractAPI(context.Background(), "http://test", testContractAPI())
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
	mcm.AssertExpectations(t)
}

func TestHandleContractAPIBroadcastValidateFail(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	api := testContractAPI()
	api.Name = "*%^!$%^&*"
	b, err := json.Marshal(api)
	assert.NoError(t, err)
	data := &core.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}
	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, &core.Message{
		Header: core.MessageHeader{
			Tag: core.SystemTagDefineContractAPI,
		},
	}, core.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.Error(t, err)
	bs.assertNoFinalizers()
}

func TestHandleContractAPIBroadcastPersistFail(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ffi := testFFI()
	b, err := json.Marshal(ffi)
	assert.NoError(t, err)
	data := &core.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}
	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("UpsertContractAPI", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ResolveContractAPI", context.Background(), "", mock.Anything).Return(nil)

	action, err := dh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, &core.Message{
		Header: core.MessageHeader{
			Tag: core.SystemTagDefineContractAPI,
		},
	}, core.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.Regexp(t, "pop", err)

	bs.assertNoFinalizers()

	mdi.AssertExpectations(t)
	mcm.AssertExpectations(t)
}

func TestHandleContractAPIBroadcastResolveFail(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ffi := testFFI()
	b, err := json.Marshal(ffi)
	assert.NoError(t, err)
	data := &core.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}
	mcm := dh.contracts.(*contractmocks.Manager)
	mcm.On("ResolveContractAPI", context.Background(), "", mock.Anything).Return(fmt.Errorf("pop"))

	action, err := dh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, &core.Message{
		Header: core.MessageHeader{
			Tag: core.SystemTagDefineContractAPI,
		},
	}, core.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.Regexp(t, "pop", err)

	bs.assertNoFinalizers()

	mcm.AssertExpectations(t)
}
