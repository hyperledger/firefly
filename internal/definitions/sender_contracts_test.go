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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/contractmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDefineFFIResolveFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	ffi := &fftypes.FFI{
		Name:      "ffi1",
		Version:   "1.0",
		Methods:   []*fftypes.FFIMethod{{}},
		Events:    []*fftypes.FFIEvent{{}},
		Errors:    []*fftypes.FFIError{{}},
		Published: true,
	}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("GetFFI", context.Background(), "ffi1", "", "1.0").Return(nil, nil)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(fmt.Errorf("pop"))

	err := ds.DefineFFI(context.Background(), ffi, false)
	assert.EqualError(t, err, "pop")

	mcm.AssertExpectations(t)
}

func TestDefineFFIFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	ffi := &fftypes.FFI{
		Name:      "ffi1",
		Version:   "1.0",
		Published: true,
	}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("GetFFI", context.Background(), "ffi1", "", "1.0").Return(nil, nil)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(nil)

	mim := ds.identity.(*identitymanagermocks.Manager)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(nil, fmt.Errorf("pop"))

	err := ds.DefineFFI(context.Background(), ffi, false)
	assert.EqualError(t, err, "pop")

	mcm.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestDefineFFIExists(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	ffi := &fftypes.FFI{
		Name:      "ffi1",
		Version:   "1.0",
		Published: true,
	}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("GetFFI", context.Background(), "ffi1", "", "1.0").Return(&fftypes.FFI{}, nil)

	err := ds.DefineFFI(context.Background(), ffi, false)
	assert.Regexp(t, "FF10302", err)

	mcm.AssertExpectations(t)
}

func TestDefineFFIOk(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	ffi := &fftypes.FFI{
		Name:      "ffi1",
		Version:   "1.0",
		Published: true,
	}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("GetFFI", context.Background(), "ffi1", "", "1.0").Return(nil, nil)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(nil)

	mim := ds.identity.(*identitymanagermocks.Manager)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)
	mim.On("ResolveInputSigningIdentity", context.Background(), mock.Anything).Return(nil)

	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &syncasyncmocks.Sender{}
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("Send", context.Background()).Return(nil)

	err := ds.DefineFFI(context.Background(), ffi, false)
	assert.NoError(t, err)

	mcm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestDefineFFIConfirm(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	ffi := &fftypes.FFI{
		Name:      "ffi1",
		Version:   "1.0",
		Published: true,
	}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("GetFFI", context.Background(), "ffi1", "", "1.0").Return(nil, nil)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(nil)

	mim := ds.identity.(*identitymanagermocks.Manager)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)
	mim.On("ResolveInputSigningIdentity", context.Background(), mock.Anything).Return(nil)

	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &syncasyncmocks.Sender{}
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("SendAndWait", context.Background()).Return(nil)

	err := ds.DefineFFI(context.Background(), ffi, true)
	assert.NoError(t, err)

	mcm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestDefineFFIPublishNonMultiparty(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = false

	ffi := &fftypes.FFI{
		Name:      "ffi1",
		Version:   "1.0",
		Published: true,
	}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("GetFFI", context.Background(), "ffi1", "", "1.0").Return(nil, nil)

	err := ds.DefineFFI(context.Background(), ffi, false)
	assert.Regexp(t, "FF10414", err)

	mcm.AssertExpectations(t)
}

func TestDefineFFINonMultiparty(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	ffi := &fftypes.FFI{
		Name:    "ffi1",
		Version: "1.0",
	}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("GetFFI", context.Background(), "ffi1", "", "1.0").Return(nil, nil)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(nil)

	mdi := ds.database.(*databasemocks.Plugin)
	mdi.On("InsertOrGetFFI", context.Background(), ffi).Return(nil, nil)
	mdi.On("InsertEvent", context.Background(), mock.Anything).Return(nil)

	err := ds.DefineFFI(context.Background(), ffi, false)
	assert.NoError(t, err)

	mcm.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestDefineFFINonMultipartyFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	ffi := &fftypes.FFI{
		Name:    "ffi1",
		Version: "1.0",
	}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("GetFFI", context.Background(), "ffi1", "", "1.0").Return(nil, nil)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(fmt.Errorf("pop"))

	err := ds.DefineFFI(context.Background(), ffi, false)
	assert.Regexp(t, "FF10403", err)

	mcm.AssertExpectations(t)
}

func TestDefineContractAPIResolveFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	url := "http://firefly"
	api := &core.ContractAPI{}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("ResolveContractAPI", context.Background(), url, api).Return(fmt.Errorf("pop"))

	err := ds.DefineContractAPI(context.Background(), url, api, false)
	assert.EqualError(t, err, "pop")

	mcm.AssertExpectations(t)
}

func TestDefineContractAPIFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	url := "http://firefly"
	api := &core.ContractAPI{}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("ResolveContractAPI", context.Background(), url, api).Return(nil)

	mim := ds.identity.(*identitymanagermocks.Manager)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(nil, fmt.Errorf("pop"))

	err := ds.DefineContractAPI(context.Background(), url, api, false)
	assert.EqualError(t, err, "pop")

	mcm.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestDefineContractAPIOk(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	url := "http://firefly"
	api := &core.ContractAPI{}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("ResolveContractAPI", context.Background(), url, api).Return(nil)

	mim := ds.identity.(*identitymanagermocks.Manager)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)
	mim.On("ResolveInputSigningIdentity", context.Background(), mock.Anything).Return(nil)

	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &syncasyncmocks.Sender{}
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("Send", context.Background()).Return(nil)

	err := ds.DefineContractAPI(context.Background(), url, api, false)
	assert.NoError(t, err)

	mcm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestDefineContractAPINonMultiparty(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	url := "http://firefly"
	api := &core.ContractAPI{}

	err := ds.DefineContractAPI(context.Background(), url, api, false)
	assert.Regexp(t, "FF10403", err)
}

func TestPublishFFI(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mdi := ds.database.(*databasemocks.Plugin)
	mcm := ds.contracts.(*contractmocks.Manager)
	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &syncasyncmocks.Sender{}

	ffi := &fftypes.FFI{
		Name:      "ffi1",
		Version:   "1.0",
		Namespace: "ns1",
		Published: false,
	}

	mcm.On("GetFFI", context.Background(), "ffi1", "", "1.0").Return(ffi, nil)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(nil)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)
	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(nil)
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("Prepare", context.Background()).Return(nil)
	mms.On("Send", context.Background()).Return(nil)
	mdi.On("UpsertFFI", context.Background(), ffi, database.UpsertOptimizationExisting).Return(nil)
	mockRunAsGroupPassthrough(mdi)

	result, err := ds.PublishFFI(context.Background(), "ffi1", "1.0", "ffi1-shared", false)
	assert.NoError(t, err)
	assert.Equal(t, ffi, result)
	assert.True(t, ffi.Published)

	mdi.AssertExpectations(t)
	mcm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestPublishFFIQueryFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mdi := ds.database.(*databasemocks.Plugin)
	mcm := ds.contracts.(*contractmocks.Manager)

	mcm.On("GetFFI", context.Background(), "ffi1", "", "1.0").Return(nil, fmt.Errorf("pop"))
	mockRunAsGroupPassthrough(mdi)

	_, err := ds.PublishFFI(context.Background(), "ffi1", "1.0", "ffi1-shared", false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mcm.AssertExpectations(t)
}

func TestPublishFFIResolveFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mdi := ds.database.(*databasemocks.Plugin)
	mcm := ds.contracts.(*contractmocks.Manager)

	ffi := &fftypes.FFI{
		Name:      "ffi1",
		Version:   "1.0",
		Namespace: "ns1",
		Published: false,
	}

	mcm.On("GetFFI", context.Background(), "ffi1", "", "1.0").Return(ffi, nil)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(fmt.Errorf("pop"))
	mockRunAsGroupPassthrough(mdi)

	_, err := ds.PublishFFI(context.Background(), "ffi1", "1.0", "ffi1-shared", false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mcm.AssertExpectations(t)
}

func TestPublishFFIPrepareFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mdi := ds.database.(*databasemocks.Plugin)
	mcm := ds.contracts.(*contractmocks.Manager)
	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &syncasyncmocks.Sender{}

	ffi := &fftypes.FFI{
		Name:      "ffi1",
		Version:   "1.0",
		Namespace: "ns1",
		Published: false,
	}

	mcm.On("GetFFI", context.Background(), "ffi1", "", "1.0").Return(ffi, nil)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(nil)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)
	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(nil)
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("Prepare", context.Background()).Return(fmt.Errorf("pop"))
	mockRunAsGroupPassthrough(mdi)

	_, err := ds.PublishFFI(context.Background(), "ffi1", "1.0", "ffi1-shared", false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mcm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestPublishFFIUpsertFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	mdi := ds.database.(*databasemocks.Plugin)
	mcm := ds.contracts.(*contractmocks.Manager)
	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &syncasyncmocks.Sender{}

	ffi := &fftypes.FFI{
		Name:      "ffi1",
		Version:   "1.0",
		Namespace: "ns1",
		Published: false,
	}

	mcm.On("GetFFI", context.Background(), "ffi1", "", "1.0").Return(ffi, nil)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(nil)
	mim.On("GetMultipartyRootOrg", context.Background()).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			DID: "firefly:org1",
		},
	}, nil)
	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(nil)
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("Prepare", context.Background()).Return(nil)
	mdi.On("UpsertFFI", context.Background(), ffi, database.UpsertOptimizationExisting).Return(fmt.Errorf("pop"))
	mockRunAsGroupPassthrough(mdi)

	_, err := ds.PublishFFI(context.Background(), "ffi1", "1.0", "ffi1-shared", false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mcm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestPublishFFINonMultiparty(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = false

	_, err := ds.PublishFFI(context.Background(), "ffi1", "1.0", "ffi1-shared", false)
	assert.Regexp(t, "FF10414", err)
}
