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

	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/contractmocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/sysmessagingmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateFFIResolveFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	ffi := &core.FFI{
		Methods: []*core.FFIMethod{{}},
		Events:  []*core.FFIEvent{{}},
	}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(fmt.Errorf("pop"))

	_, err := ds.CreateFFI(context.Background(), ffi, false)
	assert.EqualError(t, err, "pop")

	mcm.AssertExpectations(t)
}

func TestCreateFFIFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	ffi := &core.FFI{}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(nil)

	mim := ds.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", context.Background(), mock.Anything).Return(fmt.Errorf("pop"))

	_, err := ds.CreateFFI(context.Background(), ffi, false)
	assert.EqualError(t, err, "pop")

	mcm.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestCreateFFIOk(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	ffi := &core.FFI{}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("ResolveFFI", context.Background(), ffi).Return(nil)

	mim := ds.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", context.Background(), mock.Anything).Return(nil)

	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("Send", context.Background()).Return(nil)

	_, err := ds.CreateFFI(context.Background(), ffi, false)
	assert.NoError(t, err)

	mcm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestCreateContractAPIResolveFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	url := "http://firefly"
	api := &core.ContractAPI{}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("ResolveContractAPI", context.Background(), url, api).Return(fmt.Errorf("pop"))

	_, err := ds.CreateContractAPI(context.Background(), url, api, false)
	assert.EqualError(t, err, "pop")

	mcm.AssertExpectations(t)
}

func TestCreateContractAPIFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	ds.multiparty = true

	url := "http://firefly"
	api := &core.ContractAPI{}

	mcm := ds.contracts.(*contractmocks.Manager)
	mcm.On("ResolveContractAPI", context.Background(), url, api).Return(nil)

	mim := ds.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", context.Background(), mock.Anything).Return(fmt.Errorf("pop"))

	_, err := ds.CreateContractAPI(context.Background(), url, api, false)
	assert.EqualError(t, err, "pop")

	mcm.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestCreateContractAPIOk(t *testing.T) {
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
	mms := &sysmessagingmocks.MessageSender{}
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("Send", context.Background()).Return(nil)

	_, err := ds.CreateContractAPI(context.Background(), url, api, false)
	assert.NoError(t, err)

	mcm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}
