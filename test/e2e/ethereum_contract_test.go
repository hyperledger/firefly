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
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func simpleStorageFFIChanged() *fftypes.FFIEvent {
	return &fftypes.FFIEvent{
		FFIEventDefinition: fftypes.FFIEventDefinition{
			Name: "Changed",
			Params: fftypes.FFIParams{
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

func simpleStorageFFI() *fftypes.FFI {
	return &fftypes.FFI{
		Name:    "SimpleStorage",
		Version: contractVersion,
		Methods: []*fftypes.FFIMethod{
			simpleStorageFFISet(),
			simpleStorageFFIGet(),
		},
		Events: []*fftypes.FFIEvent{
			simpleStorageFFIChanged(),
		},
	}
}

func simpleStorageFFISet() *fftypes.FFIMethod {
	return &fftypes.FFIMethod{
		Name: "set",
		Params: fftypes.FFIParams{
			{
				Name:   "newValue",
				Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
			},
		},
		Returns: fftypes.FFIParams{},
	}
}

func simpleStorageFFIGet() *fftypes.FFIMethod {
	return &fftypes.FFIMethod{
		Name:   "get",
		Params: fftypes.FFIParams{},
		Returns: fftypes.FFIParams{
			{
				Name:   "output",
				Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
			},
		},
	}
}

func invokeEthContract(t *testing.T, client *resty.Client, identity, contractAddress, method string, body interface{}) {
	path := "/contracts/" + contractAddress + "/" + method
	resp, err := client.R().
		SetHeader("x-firefly-from", identity).
		SetHeader("x-firefly-sync", "true").
		SetBody(body).
		Post(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
}

type EthereumContractTestSuite struct {
	suite.Suite
	testState       *testState
	contractAddress string
	interfaceID     string
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
	suite.interfaceID = res.(map[string]interface{})["id"].(string)
	suite.T().Logf("interfaceID: %s", suite.interfaceID)
	assert.NoError(suite.T(), err)
}

func (suite *EthereumContractTestSuite) BeforeTest(suiteName, testName string) {
	suite.testState = beforeE2ETest(suite.T())
}

func (suite *EthereumContractTestSuite) TestE2EContractEvents() {
	defer suite.testState.done()

	received1, changes1 := wsReader(suite.testState.ws1, true)

	listener := CreateContractListener(suite.T(), suite.testState.client1, simpleStorageFFIChanged(), &fftypes.JSONObject{
		"address": suite.contractAddress,
	})

	<-received1
	<-changes1 // only expect database change events

	listeners := GetContractListeners(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(listeners))
	assert.Equal(suite.T(), listener.ProtocolID, listeners[0].ProtocolID)

	startTime := time.Now()
	suite.T().Log(startTime.UTC().UnixNano())

	invokeEthContract(suite.T(), suite.ethClient, suite.ethIdentity, suite.contractAddress, "set", &simpleStorageBody{
		NewValue: "1",
	})

	match := map[string]interface{}{
		"info": map[string]interface{}{
			"address": suite.contractAddress,
		},
		"output": map[string]interface{}{
			"_value": "1",
			"_from":  suite.testState.org1key.Value,
		},
		"listener": listener.ID.String(),
	}

	event := waitForContractEvent(suite.T(), suite.testState.client1, received1, match)
	assert.NotNil(suite.T(), event)

	DeleteContractListener(suite.T(), suite.testState.client1, listener.ID)
}

func (suite *EthereumContractTestSuite) TestDirectInvokeMethod() {
	defer suite.testState.done()

	received1, changes1 := wsReader(suite.testState.ws1, true)

	listener := CreateContractListener(suite.T(), suite.testState.client1, simpleStorageFFIChanged(), &fftypes.JSONObject{
		"address": suite.contractAddress,
	})

	<-changes1 // only expect database change events

	listeners := GetContractListeners(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(listeners))
	assert.Equal(suite.T(), listener.ProtocolID, listeners[0].ProtocolID)

	location := map[string]interface{}{
		"address": suite.contractAddress,
	}
	locationBytes, _ := json.Marshal(location)
	invokeContractRequest := &fftypes.ContractCallRequest{
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
		"subscription": listener.ID.String(),
	}

	event := waitForContractEvent(suite.T(), suite.testState.client1, received1, match)
	assert.NotNil(suite.T(), event)

	queryContractRequest := &fftypes.ContractCallRequest{
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

	received1, changes1 := wsReader(suite.testState.ws1, true)

	ffiReference := &fftypes.FFIReference{
		ID: fftypes.MustParseUUID(suite.interfaceID),
	}

	listener := CreateFFIContractListener(suite.T(), suite.testState.client1, ffiReference, "Changed", &fftypes.JSONObject{
		"address": suite.contractAddress,
	})

	<-changes1 // only expect database change events

	listeners := GetContractListeners(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(listeners))
	assert.Equal(suite.T(), listener.ProtocolID, listeners[0].ProtocolID)

	location := map[string]interface{}{
		"address": suite.contractAddress,
	}
	locationBytes, _ := json.Marshal(location)
	invokeContractRequest := &fftypes.ContractCallRequest{
		Location: fftypes.JSONAnyPtrBytes(locationBytes),
		Input: map[string]interface{}{
			"newValue": float64(42),
		},
	}

	<-received1

	res, err := InvokeFFIMethod(suite.testState.t, suite.testState.client1, suite.interfaceID, "set", invokeContractRequest)
	assert.NoError(suite.testState.t, err)
	assert.NotNil(suite.testState.t, res)

	match := map[string]interface{}{
		"info": map[string]interface{}{
			"address": suite.contractAddress,
		},
		"output": map[string]interface{}{
			"_value": "3",
			"_from":  suite.testState.org1key.Value,
		},
		"subscription": listener.ID.String(),
	}

	event := waitForContractEvent(suite.T(), suite.testState.client1, received1, match)
	assert.NotNil(suite.T(), event)

	queryContractRequest := &fftypes.ContractCallRequest{
		Location: fftypes.JSONAnyPtrBytes(locationBytes),
		Method:   simpleStorageFFIGet(),
	}
	res, err = QueryFFIMethod(suite.testState.t, suite.testState.client1, suite.interfaceID, "get", queryContractRequest)
	assert.NoError(suite.testState.t, err)
	resJSON, err := json.Marshal(res)
	assert.NoError(suite.testState.t, err)
	assert.Equal(suite.testState.t, `{"output":"42"}`, string(resJSON))
	DeleteContractListener(suite.T(), suite.testState.client1, listener.ID)
}
