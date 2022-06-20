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

package multiparty

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testMultipartyManager struct {
	multipartyManager
	mbi *blockchainmocks.Plugin
}

func newTestMultipartyManager() *testMultipartyManager {
	nm := &testMultipartyManager{
		mbi: &blockchainmocks.Plugin{},
		multipartyManager: multipartyManager{
			ctx: context.Background(),
		},
	}

	nm.multipartyManager.blockchain = nm.mbi
	nm.mbi.On("Name").Return("ethereum")
	nm.mbi.On("SubmitBatchPin", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	nm.mbi.On("SubmitNetworkAction", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	return nm
}

func TestNewMultipartyManager(t *testing.T) {
	mbi := &blockchainmocks.Plugin{}
	contracts := make([]Contract, 0)
	nm := NewMultipartyManager(context.Background(), "namespace", contracts, mbi)
	assert.NotNil(t, nm)
}

func TestConfigureContract(t *testing.T) {
	contracts := make([]Contract, 1)
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())
	contract := Contract{
		FirstEvent: "oldest",
		Location:   location,
	}

	contracts[0] = contract
	mp := newTestMultipartyManager()
	mp.mbi.On("GetNetworkVersion", mock.Anything, mock.Anything).Return(1, nil)
	mp.mbi.On("AddFireflySubscription", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("test", nil)
	mp.multipartyManager.contracts = contracts

	cf := &core.FireFlyContracts{
		Active: core.FireFlyContractInfo{Index: 0},
	}

	err := mp.ConfigureContract(context.Background(), cf)
	assert.NoError(t, err)
}

func TestConfigureContractOldestBlock(t *testing.T) {
	contracts := make([]Contract, 1)
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())
	contract := Contract{
		FirstEvent: "oldest",
		Location:   location,
	}

	contracts[0] = contract
	mp := newTestMultipartyManager()
	mp.mbi.On("GetNetworkVersion", mock.Anything, mock.Anything).Return(1, nil)
	mp.mbi.On("AddFireflySubscription", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("test", nil)
	mp.multipartyManager.contracts = contracts

	cf := &core.FireFlyContracts{
		Active: core.FireFlyContractInfo{Index: 0},
	}

	err := mp.ConfigureContract(context.Background(), cf)
	assert.NoError(t, err)

	assert.Equal(t, "0", mp.activeContract.firstEvent)
}

func TestConfigureContractNewestBlock(t *testing.T) {
	contracts := make([]Contract, 1)
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())
	contract := Contract{
		FirstEvent: "newest",
		Location:   location,
	}

	contracts[0] = contract
	mp := newTestMultipartyManager()
	mp.mbi.On("GetNetworkVersion", mock.Anything, mock.Anything).Return(1, nil)
	mp.mbi.On("AddFireflySubscription", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("test", nil)
	mp.multipartyManager.contracts = contracts

	cf := &core.FireFlyContracts{
		Active: core.FireFlyContractInfo{Index: 0},
	}

	err := mp.ConfigureContract(context.Background(), cf)
	assert.NoError(t, err)

	assert.Equal(t, "latest", mp.activeContract.firstEvent)
}

func TestResolveContractDeprecatedConfig(t *testing.T) {
	mp := newTestMultipartyManager()
	mp.contracts = []Contract{}

	mp.mbi.On("GetAndConvertDeprecatedContractConfig", context.Background()).Return(fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String()), "0", nil)

	loc, firstBlock, err := mp.resolveFireFlyContract(context.Background(), 0)

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())
	assert.Equal(t, location, loc)
	assert.Equal(t, "0", firstBlock)
	assert.NoError(t, err)
}

func TestResolveContractDeprecatedConfigError(t *testing.T) {
	mp := newTestMultipartyManager()
	mp.contracts = []Contract{}

	mp.mbi.On("GetAndConvertDeprecatedContractConfig", context.Background()).Return(nil, "", fmt.Errorf("pop"))

	_, _, err := mp.resolveFireFlyContract(context.Background(), 0)
	assert.Regexp(t, "pop", err)
}

func TestResolveContractDeprecatedConfigNewestBlock(t *testing.T) {
	mp := newTestMultipartyManager()
	mp.contracts = []Contract{}

	mp.mbi.On("GetAndConvertDeprecatedContractConfig", context.Background()).Return(fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String()), "newst", nil)

	_, _, err := mp.resolveFireFlyContract(context.Background(), 0)
	assert.NoError(t, err)
}

func TestConfigureContractBadIndex(t *testing.T) {
	contracts := make([]Contract, 1)
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())
	contract := Contract{
		FirstEvent: "oldest",
		Location:   location,
	}

	contracts[0] = contract
	mp := newTestMultipartyManager()
	mp.mbi.On("GetNetworkVersion", mock.Anything, mock.Anything).Return(1, nil)
	mp.mbi.On("AddFireflySubscription", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("test", nil)
	mp.multipartyManager.contracts = contracts

	cf := &core.FireFlyContracts{
		Active: core.FireFlyContractInfo{Index: 1},
	}

	err := mp.ConfigureContract(context.Background(), cf)
	assert.Regexp(t, "FF10396", err)
}

func TestConfigureContractNetworkVersionFail(t *testing.T) {
	contracts := make([]Contract, 1)
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())
	contract := Contract{
		FirstEvent: "oldest",
		Location:   location,
	}

	contracts[0] = contract
	mp := newTestMultipartyManager()
	mp.mbi.On("GetNetworkVersion", mock.Anything, mock.Anything).Return(0, fmt.Errorf("pop"))
	mp.mbi.On("AddFireflySubscription", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("test", nil)
	mp.multipartyManager.contracts = contracts

	cf := &core.FireFlyContracts{
		Active: core.FireFlyContractInfo{Index: 0},
	}

	err := mp.ConfigureContract(context.Background(), cf)
	assert.Regexp(t, "pop", err)
}

func TestGetBlockchainName(t *testing.T) {
	mp := newTestMultipartyManager()
	name := mp.Name()
	assert.Equal(t, "ethereum", name)
}

func TestSubmitNetworkAction(t *testing.T) {
	contracts := make([]Contract, 1)
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())
	contract := Contract{
		FirstEvent: "oldest",
		Location:   location,
	}

	contracts[0] = contract
	mp := newTestMultipartyManager()
	mp.mbi.On("GetNetworkVersion", mock.Anything, mock.Anything).Return(1, nil)
	mp.mbi.On("AddFireflySubscription", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("test", nil)
	mp.multipartyManager.contracts = contracts

	cf := &core.FireFlyContracts{
		Active: core.FireFlyContractInfo{Index: 0},
	}

	err := mp.ConfigureContract(context.Background(), cf)
	assert.NoError(t, err)
	err = mp.SubmitNetworkAction(context.Background(), "test", "0x123", core.NetworkActionTerminate)
	assert.Nil(t, err)
}

func TestSubmitBatchPin(t *testing.T) {
	contracts := make([]Contract, 1)
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())
	contract := Contract{
		FirstEvent: "oldest",
		Location:   location,
	}

	contracts[0] = contract
	mp := newTestMultipartyManager()
	mp.mbi.On("GetNetworkVersion", mock.Anything, mock.Anything).Return(1, nil)
	mp.mbi.On("AddFireflySubscription", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("test", nil)
	mp.multipartyManager.contracts = contracts

	cf := &core.FireFlyContracts{
		Active: core.FireFlyContractInfo{Index: 0},
	}

	batchPin := &blockchain.BatchPin{}

	err := mp.ConfigureContract(context.Background(), cf)
	assert.NoError(t, err)
	err = mp.SubmitBatchPin(context.Background(), "ns1", "0x123", batchPin)
	assert.Nil(t, err)
}

func TestGetNetworkVersion(t *testing.T) {
	contracts := make([]Contract, 1)
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())
	contract := Contract{
		FirstEvent: "oldest",
		Location:   location,
	}

	contracts[0] = contract
	mp := newTestMultipartyManager()
	mp.mbi.On("GetNetworkVersion", mock.Anything, mock.Anything).Return(1, nil)
	mp.mbi.On("AddFireflySubscription", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("test", nil)
	mp.multipartyManager.contracts = contracts

	cf := &core.FireFlyContracts{
		Active: core.FireFlyContractInfo{Index: 0},
	}

	err := mp.ConfigureContract(context.Background(), cf)
	assert.NoError(t, err)

	version := mp.GetNetworkVersion()
	assert.Equal(t, 1, version)
}

func TestConfgureAndTerminateContract(t *testing.T) {
	contracts := make([]Contract, 2)
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())
	contract := Contract{
		FirstEvent: "oldest",
		Location:   location,
	}
	contract2 := Contract{
		FirstEvent: "oldest",
		Location:   location,
	}

	contracts[0] = contract
	contracts[1] = contract2
	mp := newTestMultipartyManager()
	mp.mbi.On("GetNetworkVersion", mock.Anything, mock.Anything).Return(1, nil)
	mp.mbi.On("AddFireflySubscription", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("test", nil)
	mp.mbi.On("RemoveFireflySubscription", mock.Anything, mock.Anything).Return(nil)
	mp.multipartyManager.contracts = contracts

	cf := &core.FireFlyContracts{
		Active: core.FireFlyContractInfo{Index: 0},
	}
	err := mp.ConfigureContract(context.Background(), cf)
	assert.NoError(t, err)

	err = mp.TerminateContract(context.Background(), cf, &blockchain.Event{})
	assert.NoError(t, err)
}

func TestTerminateContractError(t *testing.T) {
	mp := newTestMultipartyManager()
	mp.mbi.On("GetNetworkVersion", mock.Anything, mock.Anything).Return(1, nil)
	mp.mbi.On("RemoveFireflySubscription", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	cf := &core.FireFlyContracts{
		Active: core.FireFlyContractInfo{Index: 0},
	}

	err := mp.TerminateContract(context.Background(), cf, &blockchain.Event{})
	assert.Regexp(t, "pop", err)
}
