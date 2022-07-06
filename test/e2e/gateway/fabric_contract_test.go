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

package gateway

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/aidarkhanov/nanoid"
	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/test/e2e"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type fabConnectHeaders struct {
	Type      string `json:"type"`
	Channel   string `json:"channel"`
	Signer    string `json:"signer"`
	Chaincode string `json:"chaincode"`
}

type createAssetBody struct {
	Headers fabConnectHeaders `json:"headers"`
	Func    string            `json:"func"`
	Args    []string          `json:"args"`
}

var assetCreatedEvent = &core.FFIEvent{
	FFIEventDefinition: core.FFIEventDefinition{
		Name: "AssetCreated",
	},
}

func assetManagerCreateAsset() *core.FFIMethod {
	return &core.FFIMethod{
		Name: "CreateAsset",
		Params: core.FFIParams{
			{
				Name:   "name",
				Schema: fftypes.JSONAnyPtr(`{"type": "string"}`),
			},
		},
		Returns: core.FFIParams{},
	}
}

func assetManagerGetAsset() *core.FFIMethod {
	return &core.FFIMethod{
		Name: "GetAsset",
		Params: core.FFIParams{
			{
				Name:   "name",
				Schema: fftypes.JSONAnyPtr(`{"type": "string"}`),
			},
		},
		Returns: core.FFIParams{
			{
				Name:   "name",
				Schema: fftypes.JSONAnyPtr(`{"type": "string"}`),
			},
		},
	}
}

func deployChaincode(t *testing.T, stackName string) string {
	id, err := nanoid.Generate("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", nanoid.DefaultSize)
	assert.NoError(t, err)
	chaincodeName := "e2e_" + id

	cmd := exec.Command("bash", "../deploy_chaincode.sh")
	cmd.Env = append(cmd.Env, "STACK_NAME="+stackName)
	cmd.Env = append(cmd.Env, "CHAINCODE_NAME="+chaincodeName)
	cmd.Env = append(cmd.Env, "PATH="+os.Getenv("PATH"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	require.NoError(t, err)

	return chaincodeName
}

func invokeFabContract(t *testing.T, client *resty.Client, channel, chaincode, signer, method string, args []string) {
	path := "/transactions"
	body := &createAssetBody{
		Headers: fabConnectHeaders{
			Type:      "SendTransaction",
			Channel:   channel,
			Signer:    signer,
			Chaincode: chaincode,
		},
		Func: method,
		Args: args,
	}
	resp, err := client.R().
		SetHeader("x-firefly-sync", "true").
		SetBody(body).
		Post(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
}

type FabricContractTestSuite struct {
	suite.Suite
	testState     *testState
	chaincodeName string
	fabClient     *resty.Client
}

func (suite *FabricContractTestSuite) SetupSuite() {
	stack := e2e.ReadStack(suite.T())
	suite.chaincodeName = deployChaincode(suite.T(), stack.Name)

	suite.fabClient = e2e.NewResty(suite.T())
	suite.fabClient.SetBaseURL(fmt.Sprintf("http://localhost:%d", stack.Members[0].ExposedConnectorPort))
}

func (suite *FabricContractTestSuite) BeforeTest(suiteName, testName string) {
	suite.testState = beforeE2ETest(suite.T())
}

func (suite *FabricContractTestSuite) AfterTest(suiteName, testName string) {
	e2e.VerifyAllOperationsSucceeded(suite.T(), []*resty.Client{suite.testState.client1}, suite.testState.startTime)
}

func (suite *FabricContractTestSuite) TestE2EContractEvents() {
	defer suite.testState.done()

	received1 := e2e.WsReader(suite.testState.ws1, true)

	sub := e2e.CreateContractListener(suite.T(), suite.testState.client1, assetCreatedEvent, &fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": suite.chaincodeName,
	})

	subs := e2e.GetContractListeners(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(subs))
	assert.Equal(suite.T(), sub.BackendID, subs[0].BackendID)

	assetName := nanoid.New()
	location := map[string]interface{}{
		"chaincode": suite.chaincodeName,
		"channel":   "firefly",
	}
	locationBytes, _ := json.Marshal(location)
	invokeContractRequest := &core.ContractCallRequest{
		Location: fftypes.JSONAnyPtrBytes(locationBytes),
		Method:   assetManagerCreateAsset(),
		Input: map[string]interface{}{
			"name": assetName,
		},
	}

	res, err := e2e.InvokeContractMethod(suite.testState.t, suite.testState.client1, invokeContractRequest)
	suite.T().Log(res)
	assert.NoError(suite.T(), err)

	<-received1

	events := e2e.GetContractEvents(suite.T(), suite.testState.client1, suite.testState.startTime, sub.ID)
	assert.Equal(suite.T(), 1, len(events))
	assert.Equal(suite.T(), "AssetCreated", events[0].Name)
	assert.Equal(suite.T(), assetName, events[0].Output.GetString("name"))

	queryContractRequest := &core.ContractCallRequest{
		Location: fftypes.JSONAnyPtrBytes(locationBytes),
		Method:   assetManagerGetAsset(),
		Input: map[string]interface{}{
			"name": assetName,
		},
	}

	res, err = e2e.QueryContractMethod(suite.testState.t, suite.testState.client1, queryContractRequest)
	suite.T().Log(res)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), assetName, res.(map[string]interface{})["name"])

	e2e.DeleteContractListener(suite.T(), suite.testState.client1, subs[0].ID)
	subs = e2e.GetContractListeners(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 0, len(subs))

}
