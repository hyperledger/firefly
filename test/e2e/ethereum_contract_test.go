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

package e2e

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aidarkhanov/nanoid"
	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var contractVersion, _ = nanoid.Generate("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", nanoid.DefaultSize)

type uploadABIResult struct {
	ID string `json:"id"`
}

type deployABIResult struct {
	ContractAddress string `json:"contractAddress"`
}

type ethconnectOutput struct {
	Output string `json:"output"`
}

type simpleStorageBody struct {
	NewValue string `json:"newValue"`
}

func simpleStorageFFIChanged() *core.FFIEvent {
	return &core.FFIEvent{
		FFIEventDefinition: core.FFIEventDefinition{
			Name: "Changed",
			Params: core.FFIParams{
				{
					Name:   "_from",
					Schema: fftypes.JSONAnyPtr(`{"type": "string", "details": {"type": "address", "indexed": true}}`),
				},
				{
					Name:   "_value",
					Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
				},
			},
		},
	}
}

func simpleStorageFFI() *core.FFI {
	return &core.FFI{
		Name:    "SimpleStorage",
		Version: contractVersion,
		Methods: []*core.FFIMethod{
			simpleStorageFFISet(),
			simpleStorageFFIGet(),
		},
		Events: []*core.FFIEvent{
			simpleStorageFFIChanged(),
		},
	}
}

func simpleStorageFFISet() *core.FFIMethod {
	return &core.FFIMethod{
		Name: "set",
		Params: core.FFIParams{
			{
				Name:   "newValue",
				Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
			},
		},
		Returns: core.FFIParams{},
	}
}

func simpleStorageFFIGet() *core.FFIMethod {
	return &core.FFIMethod{
		Name:   "get",
		Params: core.FFIParams{},
		Returns: core.FFIParams{
			{
				Name:   "output",
				Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
			},
		},
	}
}

type EthereumContractTestSuite struct {
	suite.Suite
	testState       *testState
	contractAddress string
	interfaceID     *fftypes.UUID
	ethClient       *resty.Client
	ethIdentity     string
}

func (suite *EthereumContractTestSuite) SetupSuite() {
	suite.testState = beforeE2ETest(suite.T())
	stack := readStackFile(suite.T())
	suite.ethClient = NewResty(suite.T())
	suite.ethClient.SetBaseURL(fmt.Sprintf("http://localhost:%d", stack.Members[0].ExposedConnectorPort))
	suite.ethIdentity = suite.testState.org1key.Value
	suite.contractAddress = os.Getenv("CONTRACT_ADDRESS")
	if suite.contractAddress == "" {
		suite.T().Fatal("CONTRACT_ADDRESS must be set")
	}
	suite.T().Logf("contractAddress: %s", suite.contractAddress)

	res, err := CreateFFI(suite.T(), suite.testState.client1, simpleStorageFFI())
	suite.interfaceID = fftypes.MustParseUUID(res.(map[string]interface{})["id"].(string))
	suite.T().Logf("interfaceID: %s", suite.interfaceID)
	assert.NoError(suite.T(), err)
}

func (suite *EthereumContractTestSuite) BeforeTest(suiteName, testName string) {
	suite.testState = beforeE2ETest(suite.T())
}

func (suite *EthereumContractTestSuite) AfterTest(suiteName, testName string) {
	verifyAllOperationsSucceeded(suite.T(), []*resty.Client{suite.testState.client1, suite.testState.client2}, suite.testState.startTime)
}

func (suite *EthereumContractTestSuite) TestDirectInvokeMethod() {
	defer suite.testState.done()

	received1 := wsReader(suite.testState.ws1, true)
	listener := CreateContractListener(suite.T(), suite.testState.client1, simpleStorageFFIChanged(), &fftypes.JSONObject{
		"address": suite.contractAddress,
	})

	listeners := GetContractListeners(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(listeners))
	assert.Equal(suite.T(), listener.BackendID, listeners[0].BackendID)

	location := map[string]interface{}{
		"address": suite.contractAddress,
	}
	locationBytes, _ := json.Marshal(location)
	invokeContractRequest := &core.ContractCallRequest{
		Location: fftypes.JSONAnyPtrBytes(locationBytes),
		Method:   simpleStorageFFISet(),
		Input: map[string]interface{}{
			"newValue": float64(2),
		},
	}

	res, err := InvokeContractMethod(suite.testState.t, suite.testState.client1, invokeContractRequest)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), res)

	match := map[string]interface{}{
		"info": map[string]interface{}{
			"address": suite.contractAddress,
		},
		"output": map[string]interface{}{
			"_value": "2",
			"_from":  suite.testState.org1key.Value,
		},
		"listener": listener.ID.String(),
	}

	event := waitForContractEvent(suite.T(), suite.testState.client1, received1, match)
	assert.NotNil(suite.T(), event)

	queryContractRequest := &core.ContractCallRequest{
		Location: fftypes.JSONAnyPtrBytes(locationBytes),
		Method:   simpleStorageFFIGet(),
	}
	res, err = QueryContractMethod(suite.testState.t, suite.testState.client1, queryContractRequest)
	assert.NoError(suite.testState.t, err)
	resJSON, err := json.Marshal(res)
	assert.NoError(suite.testState.t, err)
	assert.Equal(suite.testState.t, `{"output":"2"}`, string(resJSON))
	DeleteContractListener(suite.T(), suite.testState.client1, listener.ID)
}

func (suite *EthereumContractTestSuite) TestFFIInvokeMethod() {
	defer suite.testState.done()

	received1 := wsReader(suite.testState.ws1, true)

	ffiReference := &core.FFIReference{
		ID: suite.interfaceID,
	}
	listener := CreateFFIContractListener(suite.T(), suite.testState.client1, ffiReference, "Changed", &fftypes.JSONObject{
		"address": suite.contractAddress,
	})

	listeners := GetContractListeners(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(listeners))
	assert.Equal(suite.T(), listener.BackendID, listeners[0].BackendID)

	location := map[string]interface{}{
		"address": suite.contractAddress,
	}
	locationBytes, _ := json.Marshal(location)
	invokeContractRequest := &core.ContractCallRequest{
		Location: fftypes.JSONAnyPtrBytes(locationBytes),
		Input: map[string]interface{}{
			"newValue": float64(42),
		},
		Interface:  suite.interfaceID,
		MethodPath: "set",
	}

	res, err := InvokeContractMethod(suite.testState.t, suite.testState.client1, invokeContractRequest)
	assert.NoError(suite.testState.t, err)
	assert.NotNil(suite.testState.t, res)

	match := map[string]interface{}{
		"info": map[string]interface{}{
			"address": suite.contractAddress,
		},
		"output": map[string]interface{}{
			"_value": "42",
			"_from":  suite.testState.org1key.Value,
		},
		"listener": listener.ID.String(),
	}
	event := waitForContractEvent(suite.T(), suite.testState.client1, received1, match)
	assert.NotNil(suite.T(), event)

	queryContractRequest := &core.ContractCallRequest{
		Location:   fftypes.JSONAnyPtrBytes(locationBytes),
		Interface:  suite.interfaceID,
		MethodPath: "get",
	}
	res, err = QueryContractMethod(suite.testState.t, suite.testState.client1, queryContractRequest)
	assert.NoError(suite.testState.t, err)
	resJSON, err := json.Marshal(res)
	assert.NoError(suite.testState.t, err)
	assert.Equal(suite.testState.t, `{"output":"42"}`, string(resJSON))
	DeleteContractListener(suite.T(), suite.testState.client1, listener.ID)
}

func (suite *EthereumContractTestSuite) TestContractAPIMethod() {
	defer suite.testState.done()

	received1 := wsReader(suite.testState.ws1, true)
	APIName := fftypes.NewUUID().String()

	ffiReference := &core.FFIReference{
		ID: suite.interfaceID,
	}

	location := map[string]interface{}{
		"address": suite.contractAddress,
	}
	locationBytes, _ := json.Marshal(location)

	createContractAPIResult, err := CreateContractAPI(suite.T(), suite.testState.client1, APIName, ffiReference, fftypes.JSONAnyPtr(string(locationBytes)))
	assert.NotNil(suite.T(), createContractAPIResult)
	assert.NoError(suite.T(), err)

	listener, err := CreateContractAPIListener(suite.T(), suite.testState.client1, APIName, "Changed", "firefly_e2e")
	assert.NoError(suite.T(), err)

	listeners := GetContractListeners(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(listeners))
	assert.Equal(suite.T(), listener.BackendID, listeners[0].BackendID)

	input := fftypes.JSONAny(`{"newValue": 42}`)
	invokeResult, err := InvokeContractAPIMethod(suite.T(), suite.testState.client1, APIName, "set", &input)
	assert.NoError(suite.testState.t, err)
	assert.NotNil(suite.T(), invokeResult)

	match := map[string]interface{}{
		"info": map[string]interface{}{
			"address": suite.contractAddress,
		},
		"output": map[string]interface{}{
			"_value": "42",
			"_from":  suite.testState.org1key.Value,
		},
		"listener": listener.ID.String(),
	}
	event := waitForContractEvent(suite.T(), suite.testState.client1, received1, match)
	assert.NotNil(suite.T(), event)
	assert.NoError(suite.testState.t, err)

	res, err := QueryContractAPIMethod(suite.T(), suite.testState.client1, APIName, "get", fftypes.JSONAnyPtr("{}"))
	assert.NoError(suite.testState.t, err)
	resJSON, err := json.Marshal(res)
	assert.NoError(suite.testState.t, err)
	assert.Equal(suite.testState.t, `{"output":"42"}`, string(resJSON))

	DeleteContractListener(suite.T(), suite.testState.client1, listener.ID)
}
