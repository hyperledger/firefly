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

package definitions

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
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
						Name:    "x",
						Type:    "integer",
						Details: []byte(`{}`),
					},
					{
						Name:    "y",
						Type:    "integer",
						Details: []byte(`{}`),
					},
				},
				Returns: fftypes.FFIParams{
					{
						Name:    "result",
						Type:    "integer",
						Details: []byte(`{}`),
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
		Contract: &fftypes.ContractIdentifier{
			ID: fftypes.NewUUID(),
		},
		Ledger:   []byte{},
		Location: []byte{},
	}
}

func TestHandleFFIBroadcastOk(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	b, err := json.Marshal(testFFI())
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("UpsertFFI", mock.Anything, mock.Anything).Return(nil)
	mbi.On("UpsertFFIMethod", mock.Anything, "ns1", mock.Anything, mock.Anything).Return(nil)
	mbi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineFFI),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionConfirm, action)
	assert.NoError(t, err)
	mbi.AssertExpectations(t)
}

func TestPersistFFIUpsertFFIFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("UpsertFFI", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	_, err := dh.persistFFI(context.Background(), testFFI())
	assert.Regexp(t, "pop", err)
	mbi.AssertExpectations(t)
}

func TestPersistFFIUpsertFFIMethodFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("UpsertFFI", mock.Anything, mock.Anything).Return(nil)
	mbi.On("UpsertFFIMethod", mock.Anything, "ns1", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	_, err := dh.persistFFI(context.Background(), testFFI())
	assert.Regexp(t, "pop", err)
	mbi.AssertExpectations(t)
}

func TestHandleFFIBroadcastValidateFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)
	ffi := testFFI()
	ffi.Name = "*%^!$%^&*"
	b, err := json.Marshal(ffi)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineFFI),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)
}

func TestHandleFFIBroadcastPersistFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)
	ffi := testFFI()
	b, err := json.Marshal(ffi)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("UpsertFFI", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mbi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineFFI),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.Regexp(t, "pop", err)
}

func TestHandleContractAPIBroadcastOk(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	b, err := json.Marshal(testFFI())
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("UpsertContractAPI", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mbi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineContractAPI),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionConfirm, action)
	assert.NoError(t, err)
	mbi.AssertExpectations(t)
}

func TestPersistContractAPIUpsertFFIFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("UpsertContractAPI", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	_, err := dh.persistContractAPI(context.Background(), testContractAPI())
	assert.Regexp(t, "pop", err)
	mbi.AssertExpectations(t)
}

func TestHandleContractAPIBroadcastValidateFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)
	api := testContractAPI()
	api.Name = "*%^!$%^&*"
	b, err := json.Marshal(api)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineContractAPI),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)
}

func TestHandleContractAPIBroadcastPersistFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)
	ffi := testFFI()
	b, err := json.Marshal(ffi)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}
	mbi := dh.database.(*databasemocks.Plugin)
	mbi.On("UpsertContractAPI", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mbi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineContractAPI),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.Regexp(t, "pop", err)
}
