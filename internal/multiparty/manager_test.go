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
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testMultipartyManager struct {
	multipartyManager
	mdi *databasemocks.Plugin
	mbi *blockchainmocks.Plugin
	mom *operationmocks.Manager
	mmi *metricsmocks.Manager
}

func (mp *testMultipartyManager) cleanup(t *testing.T) {
	mp.mdi.AssertExpectations(t)
	mp.mbi.AssertExpectations(t)
	mp.mom.AssertExpectations(t)
	mp.mmi.AssertExpectations(t)
}

func newTestMultipartyManager() *testMultipartyManager {
	nm := &testMultipartyManager{
		mdi: &databasemocks.Plugin{},
		mbi: &blockchainmocks.Plugin{},
		mom: &operationmocks.Manager{},
		mmi: &metricsmocks.Manager{},
		multipartyManager: multipartyManager{
			ctx:       context.Background(),
			namespace: "ns1",
		},
	}

	nm.multipartyManager.database = nm.mdi
	nm.multipartyManager.blockchain = nm.mbi
	nm.multipartyManager.operations = nm.mom
	nm.multipartyManager.metrics = nm.mmi
	return nm
}

func TestNewMultipartyManager(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mbi := &blockchainmocks.Plugin{}
	mom := &operationmocks.Manager{}
	mmi := &metricsmocks.Manager{}
	contracts := make([]Contract, 0)
	mom.On("RegisterHandler", mock.Anything, mock.Anything, []core.OpType{
		core.OpTypeBlockchainPinBatch,
	}).Return()
	nm, err := NewMultipartyManager(context.Background(), "namespace", contracts, mdi, mbi, mom, mmi)
	assert.NotNil(t, nm)
	assert.NoError(t, err)
	assert.Equal(t, "MultipartyManager", nm.Name())
}

func TestInitFail(t *testing.T) {
	contracts := make([]Contract, 0)
	_, err := NewMultipartyManager(context.Background(), "namespace", contracts, nil, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestConfigureContract(t *testing.T) {
	contracts := make([]Contract, 1)
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())
	contract := Contract{
		FirstEvent: "0",
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
		FirstEvent: "0",
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
		FirstEvent: "latest",
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
		FirstEvent: "0",
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
		FirstEvent: "0",
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

func TestSubmitNetworkAction(t *testing.T) {
	contracts := make([]Contract, 1)
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())
	contract := Contract{
		FirstEvent: "0",
		Location:   location,
	}

	contracts[0] = contract
	mp := newTestMultipartyManager()
	mp.mbi.On("GetNetworkVersion", mock.Anything, mock.Anything).Return(1, nil)
	mp.mbi.On("AddFireflySubscription", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("test", nil)
	mp.mbi.On("SubmitNetworkAction", mock.Anything, "test", "0x123", core.NetworkActionTerminate, mock.Anything).Return(nil)
	mp.multipartyManager.contracts = contracts

	cf := &core.FireFlyContracts{
		Active: core.FireFlyContractInfo{Index: 0},
	}

	err := mp.ConfigureContract(context.Background(), cf)
	assert.NoError(t, err)
	err = mp.SubmitNetworkAction(context.Background(), "test", "0x123", core.NetworkActionTerminate)
	assert.Nil(t, err)
}

func TestSubmitBatchPinOk(t *testing.T) {
	mp := newTestMultipartyManager()
	defer mp.cleanup(t)
	ctx := context.Background()

	batch := &core.BatchPersisted{
		BatchHeader: core.BatchHeader{
			ID: fftypes.NewUUID(),
			SignerRef: core.SignerRef{
				Author: "id1",
				Key:    "0x12345",
			},
		},
		TX: core.TransactionRef{
			ID: fftypes.NewUUID(),
		},
	}
	contexts := []*fftypes.Bytes32{}

	mp.mbi.On("Name").Return("ut")
	mp.mom.On("AddOrReuseOperation", ctx, mock.MatchedBy(func(op *core.Operation) bool {
		assert.Equal(t, core.OpTypeBlockchainPinBatch, op.Type)
		assert.Equal(t, "ut", op.Plugin)
		assert.Equal(t, *batch.TX.ID, *op.Transaction)
		assert.Equal(t, "payload1", op.Input.GetString("payloadRef"))
		return true
	})).Return(nil)
	mp.mmi.On("IsMetricsEnabled").Return(false)
	mp.mom.On("RunOperation", mock.Anything, mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(batchPinData)
		return op.Type == core.OpTypeBlockchainPinBatch && data.Batch == batch
	})).Return(nil, nil)

	err := mp.SubmitBatchPin(ctx, batch, contexts, "payload1")
	assert.NoError(t, err)
}

func TestSubmitPinnedBatchWithMetricsOk(t *testing.T) {
	mp := newTestMultipartyManager()
	defer mp.cleanup(t)
	ctx := context.Background()

	batch := &core.BatchPersisted{
		BatchHeader: core.BatchHeader{
			ID: fftypes.NewUUID(),
			SignerRef: core.SignerRef{
				Author: "id1",
				Key:    "0x12345",
			},
		},
		TX: core.TransactionRef{
			ID: fftypes.NewUUID(),
		},
	}
	contexts := []*fftypes.Bytes32{}

	mp.mbi.On("Name").Return("ut")
	mp.mom.On("AddOrReuseOperation", ctx, mock.MatchedBy(func(op *core.Operation) bool {
		assert.Equal(t, core.OpTypeBlockchainPinBatch, op.Type)
		assert.Equal(t, "ut", op.Plugin)
		assert.Equal(t, *batch.TX.ID, *op.Transaction)
		assert.Equal(t, "payload1", op.Input.GetString("payloadRef"))
		return true
	})).Return(nil)
	mp.mmi.On("IsMetricsEnabled").Return(true)
	mp.mmi.On("CountBatchPin").Return()
	mp.mom.On("RunOperation", mock.Anything, mock.MatchedBy(func(op *core.PreparedOperation) bool {
		data := op.Data.(batchPinData)
		return op.Type == core.OpTypeBlockchainPinBatch && data.Batch == batch
	})).Return(nil, nil)

	err := mp.SubmitBatchPin(ctx, batch, contexts, "payload1")
	assert.NoError(t, err)
}

func TestSubmitPinnedBatchOpFail(t *testing.T) {
	mp := newTestMultipartyManager()
	defer mp.cleanup(t)
	ctx := context.Background()

	batch := &core.BatchPersisted{
		BatchHeader: core.BatchHeader{
			ID: fftypes.NewUUID(),
			SignerRef: core.SignerRef{
				Author: "id1",
				Key:    "0x12345",
			},
		},
		TX: core.TransactionRef{
			ID: fftypes.NewUUID(),
		},
	}
	contexts := []*fftypes.Bytes32{}

	mp.mbi.On("Name").Return("ut")
	mp.mom.On("AddOrReuseOperation", ctx, mock.Anything).Return(fmt.Errorf("pop"))
	err := mp.SubmitBatchPin(ctx, batch, contexts, "payload1")
	assert.Regexp(t, "pop", err)
}

func TestGetNetworkVersion(t *testing.T) {
	contracts := make([]Contract, 1)
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())
	contract := Contract{
		FirstEvent: "0",
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
		FirstEvent: "0",
		Location:   location,
	}
	contract2 := Contract{
		FirstEvent: "0",
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
