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
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDefineFFIResolveFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	ffi := &fftypes.FFI{
		Methods: []*fftypes.FFIMethod{{}},
		Events:  []*fftypes.FFIEvent{{}},
	}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(fmt.Errorf("pop"))

	err := ds.DefineFFI(context.Background(), ffi, false)
	assert.EqualError(t, err, "pop")

	mcm.AssertExpectations(t)
}

func TestDefineFFIFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	ffi := &fftypes.FFI{}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(nil)

	mim := ds.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", context.Background(), mock.Anything).Return(fmt.Errorf("pop"))

	err := ds.DefineFFI(context.Background(), ffi, false)
	assert.EqualError(t, err, "pop")

	mcm.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestDefineFFIOk(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	ffi := &fftypes.FFI{}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(nil)

	mim := ds.identity.(*identitymanagermocks.Manager)
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

func TestDefineFFINonMultiparty(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	ffi := &fftypes.FFI{}

	err := ds.DefineFFI(context.Background(), ffi, false)
	assert.Regexp(t, "FF10403", err)
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
	mim.On("ResolveInputSigningIdentity", context.Background(), mock.Anything).Return(fmt.Errorf("pop"))

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
