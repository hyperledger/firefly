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

package e2e

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/aidarkhanov/nanoid"
	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var contractVersion, _ = nanoid.Generate(nanoid.DefaultAlphabet, nanoid.DefaultSize)

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

func newTestFFIEvent() *fftypes.FFIEvent {
	return &fftypes.FFIEvent{
		FFIEventDefinition: fftypes.FFIEventDefinition{
			Name: "Changed",
			Params: fftypes.FFIParams{
				{
					Name: "_from",
					Type: "string",
					Details: fftypes.Byteable(fftypes.JSONObject{
						"type":    "address",
						"indexed": true,
					}.String()),
				},
				{
					Name: "_value",
					Type: "integer",
					Details: fftypes.Byteable(fftypes.JSONObject{
						"type": "uint256",
					}.String()),
				},
			},
		},
	}
}

func newTestFFI() *fftypes.FFI {
	return &fftypes.FFI{
		Name:    "SimpleStorage",
		Version: contractVersion,
		Methods: []*fftypes.FFIMethod{
			newTestFFIMethod(),
		},
		Events: []*fftypes.FFIEvent{
			newTestFFIEvent(),
		},
	}
}

func newTestFFIMethod() *fftypes.FFIMethod {
	return &fftypes.FFIMethod{
		Name: "set",
		Params: fftypes.FFIParams{
			{
				Name:    "newValue",
				Type:    "integer",
				Details: []byte(`{"type": "uint256"}`),
			},
		},
		Returns: fftypes.FFIParams{},
	}
}

func loadSimpleStorageABI(t *testing.T) map[string]string {
	abi, err := ioutil.ReadFile("../data/simplestorage/simplestorage.abi.json")
	require.NoError(t, err)
	bytecode, err := ioutil.ReadFile("../data/simplestorage/simplestorage.bin")
	require.NoError(t, err)
	return map[string]string{
		"abi":      string(abi),
		"bytecode": "0x" + hex.EncodeToString(bytecode),
	}
}

func uploadABI(t *testing.T, client *resty.Client, abi map[string]string) (result uploadABIResult) {
	path := "/abis"
	resp, err := client.R().
		SetMultipartFormData(abi).
		SetResult(&result).
		Post(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return result
}

func deployABI(t *testing.T, client *resty.Client, identity, abiID string) (result deployABIResult) {
	path := "/abis/" + abiID
	resp, err := client.R().
		SetHeader("x-firefly-from", identity).
		SetHeader("x-firefly-sync", "true").
		SetResult(&result).
		Post(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return result
}

func queryContract(t *testing.T, client *resty.Client, contractAddress, method string) string {
	path := "/contracts/" + contractAddress + "/" + method
	var result ethconnectOutput
	resp, err := client.R().
		SetResult(&result).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return result.Output
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

	abi := loadSimpleStorageABI(suite.T())

	suite.ethClient = NewResty(suite.T())
	suite.ethClient.SetBaseURL(fmt.Sprintf("http://localhost:%d", stack.Members[0].ExposedConnectorPort))
	suite.ethIdentity = suite.testState.org1.Identity

	abiResult := uploadABI(suite.T(), suite.ethClient, abi)
	contractResult := deployABI(suite.T(), suite.ethClient, suite.ethIdentity, abiResult.ID)

	suite.contractAddress = contractResult.ContractAddress

	suite.T().Logf("contractAddress: %s", suite.contractAddress)

	res, err := CreateFFI(suite.T(), suite.testState.client1, newTestFFI())
	suite.interfaceID = res.(map[string]interface{})["id"].(string)
	suite.T().Logf("interfaceID: %s", suite.interfaceID)
	assert.NoError(suite.T(), err)
}

func (suite *EthereumContractTestSuite) BeforeTest(suiteName, testName string) {
	suite.testState = beforeE2ETest(suite.T())
}

func (suite *EthereumContractTestSuite) TestE2EContractEvents() {
	defer suite.testState.done()

	received1, changes1 := wsReader(suite.T(), suite.testState.ws1)

	sub := CreateContractSubscription(suite.T(), suite.testState.client1, newTestFFIEvent(), &fftypes.JSONObject{
		"address": suite.contractAddress,
	})

	<-changes1 // only expect database change events

	subs := GetContractSubscriptions(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(subs))
	assert.Equal(suite.T(), sub.ProtocolID, subs[0].ProtocolID)

	invokeEthContract(suite.T(), suite.ethClient, suite.ethIdentity, suite.contractAddress, "set", &simpleStorageBody{
		NewValue: "1",
	})

	<-received1
	<-changes1 // also expect database change events

	events := GetContractEvents(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(events))
	assert.Equal(suite.T(), "Changed", events[0].Name)
	assert.Equal(suite.T(), "1", events[0].Outputs.GetString("_value"))
	assert.Equal(suite.T(), suite.ethIdentity, events[0].Outputs.GetString("_from"))

	DeleteContractSubscription(suite.T(), suite.testState.client1, subs[0].ID)
	subs = GetContractSubscriptions(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 0, len(subs))

	<-changes1 // only expect database change events
}

func (suite *EthereumContractTestSuite) TestDirectInvokeMethod() {
	defer suite.testState.done()

	received1, changes1 := wsReader(suite.T(), suite.testState.ws1)

	sub := CreateContractSubscription(suite.T(), suite.testState.client1, newTestFFIEvent(), &fftypes.JSONObject{
		"address": suite.contractAddress,
	})

	<-changes1 // only expect database change events

	subs := GetContractSubscriptions(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(subs))
	assert.Equal(suite.T(), sub.ProtocolID, subs[0].ProtocolID)

	location := map[string]interface{}{
		"address": suite.contractAddress,
	}
	locationBytes, _ := json.Marshal(location)
	invokeContractRequest := &fftypes.InvokeContractRequest{
		Location: locationBytes,
		Method:   newTestFFIMethod(),
	}
	res, err := InvokeContractMethod(suite.testState.t, suite.testState.client1, invokeContractRequest)
	assert.NoError(suite.testState.t, err)
	assert.NotNil(suite.testState.t, res)

	<-received1
	<-changes1 // also expect database change events

	events := GetContractEvents(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(events))
	assert.Equal(suite.T(), "Changed", events[0].Name)
	assert.Equal(suite.T(), "42", events[0].Outputs.GetString("_value"))
	assert.Equal(suite.T(), suite.ethIdentity, events[0].Outputs.GetString("_from"))
}

func (suite *EthereumContractTestSuite) TestFFIInvokeMethod() {
	defer suite.testState.done()

	received1, changes1 := wsReader(suite.T(), suite.testState.ws1)

	sub := CreateContractSubscription(suite.T(), suite.testState.client1, newTestFFIEvent(), &fftypes.JSONObject{
		"address": suite.contractAddress,
	})

	<-changes1 // only expect database change events

	subs := GetContractSubscriptions(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(subs))
	assert.Equal(suite.T(), sub.ProtocolID, subs[0].ProtocolID)

	ffi := newTestFFI()
	location := map[string]interface{}{
		"address": suite.contractAddress,
	}
	locationBytes, _ := json.Marshal(location)
	invokeContractRequest := &fftypes.InvokeContractRequest{
		Location: locationBytes,
		Input: map[string]interface{}{
			"newValue": float64(42),
		},
	}

	<-received1
	<-changes1 // also expect database change events

	res, err := InvokeFFIMethod(suite.testState.t, suite.testState.client1, suite.interfaceID, ffi.Methods[0].Name, invokeContractRequest)
	assert.NoError(suite.testState.t, err)
	assert.NotNil(suite.testState.t, res)

	<-received1
	<-received1

	events := GetContractEvents(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(events))
	assert.Equal(suite.T(), "Changed", events[0].Name)
	assert.Equal(suite.T(), "42", events[0].Outputs.GetString("_value"))
	assert.Equal(suite.T(), suite.ethIdentity, events[0].Outputs.GetString("_from"))
}
