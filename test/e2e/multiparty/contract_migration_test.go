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
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/test/e2e"
	"github.com/hyperledger/firefly/test/e2e/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
)

func randomName(t *testing.T) string {
	b := make([]byte, 5)
	_, err := rand.Read(b)
	assert.NoError(t, err)
	return fmt.Sprintf("e2e_%x", b)

}

func readConfig(t *testing.T, configFile string) map[string]interface{} {
	yfile, err := ioutil.ReadFile(configFile)
	assert.NoError(t, err)
	data := make(map[string]interface{})
	err = yaml.Unmarshal(yfile, &data)
	assert.NoError(t, err)
	return data
}

func writeConfig(t *testing.T, configFile string, data map[string]interface{}) {
	out, err := yaml.Marshal(data)
	assert.NoError(t, err)
	f, err := os.Create(configFile)
	assert.NoError(t, err)
	f.Write(out)
	f.Close()
}

func addNamespace(data map[string]interface{}, ns map[string]interface{}) {
	namespaces := data["namespaces"].(map[interface{}]interface{})
	predefined := namespaces["predefined"].([]interface{})
	namespaces["predefined"] = append(predefined, ns)
}

func resetFireFly(t *testing.T, client *resty.Client) {
	resp, err := client.R().
		SetBody(map[string]interface{}{}).
		Post("/reset")
	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode())
}

type ContractMigrationTestSuite struct {
	suite.Suite
	testState   *testState
	stackName   string
	adminHost1  string
	adminHost2  string
	configFile1 string
	configFile2 string
}

func (suite *ContractMigrationTestSuite) SetupSuite() {
	suite.testState = beforeE2ETest(suite.T())
	stack := e2e.ReadStack(suite.T())
	suite.stackName = stack.Name

	adminProtocol1 := "http"
	if stack.Members[0].UseHTTPS {
		adminProtocol1 = "https"
	}
	adminProtocol2 := "http"
	if stack.Members[1].UseHTTPS {
		adminProtocol2 = "https"
	}
	suite.adminHost1 = fmt.Sprintf("%s://%s:%d", adminProtocol1, stack.Members[0].FireflyHostname, stack.Members[0].ExposedAdminPort)
	suite.adminHost2 = fmt.Sprintf("%s://%s:%d", adminProtocol2, stack.Members[1].FireflyHostname, stack.Members[1].ExposedAdminPort)

	stackDir := os.Getenv("STACK_DIR")
	if stackDir == "" {
		suite.T().Fatal("STACK_DIR must be set")
	}
	suite.configFile1 = filepath.Join(stackDir, "runtime", "config", "firefly_core_0.yml")
	suite.configFile2 = filepath.Join(stackDir, "runtime", "config", "firefly_core_1.yml")
}

func (suite *ContractMigrationTestSuite) BeforeTest(suiteName, testName string) {
	suite.testState = beforeE2ETest(suite.T())
}

func (suite *ContractMigrationTestSuite) AfterTest(suiteName, testName string) {
	e2e.VerifyAllOperationsSucceeded(suite.T(), []*client.FireFlyClient{suite.testState.client1, suite.testState.client2}, suite.testState.startTime)
}

func (suite *ContractMigrationTestSuite) TestContractMigration() {
	defer suite.testState.Done()

	address1 := deployContract(suite.T(), suite.stackName, "firefly/FireflyV1.json")
	address2 := deployContract(suite.T(), suite.stackName, "firefly/Firefly.json")
	testNamespace := randomName(suite.T())
	suite.T().Logf("Test namespace: %s", testNamespace)

	org := map[string]interface{}{}
	namespaceInfo := map[string]interface{}{
		"name": testNamespace,
		"multiparty": map[string]interface{}{
			"enabled": true,
			"org":     org,
			"contract": []map[string]interface{}{
				{
					"location": map[string]interface{}{"address": address1},
				},
				{
					"location": map[string]interface{}{"address": address2},
				},
			},
		},
	}

	// Add the new namespace to both config files
	data1 := readConfig(suite.T(), suite.configFile1)
	org["name"] = suite.testState.org1.Name
	org["key"] = suite.testState.org1key.Value
	addNamespace(data1, namespaceInfo)
	writeConfig(suite.T(), suite.configFile1, data1)

	data2 := readConfig(suite.T(), suite.configFile2)
	org["name"] = suite.testState.org2.Name
	org["key"] = suite.testState.org2key.Value
	addNamespace(data2, namespaceInfo)
	writeConfig(suite.T(), suite.configFile2, data2)

	admin1 := client.NewResty(suite.T())
	admin2 := client.NewResty(suite.T())
	admin1.SetBaseURL(suite.adminHost1 + "/spi/v1")
	admin2.SetBaseURL(suite.adminHost2 + "/spi/v1")

	// Reset both nodes to pick up the new namespace
	resetFireFly(suite.T(), admin1)
	resetFireFly(suite.T(), admin2)
	e2e.PollForUp(suite.T(), suite.testState.client1)
	e2e.PollForUp(suite.T(), suite.testState.client2)

	systemClient1 := client.NewFireFly(suite.T(), suite.testState.client1.Hostname, "ff_system")
	systemClient2 := client.NewFireFly(suite.T(), suite.testState.client2.Hostname, "ff_system")
	client1 := client.NewFireFly(suite.T(), suite.testState.client1.Hostname, testNamespace)
	client2 := client.NewFireFly(suite.T(), suite.testState.client2.Hostname, testNamespace)

	// Register org/node identities on ff_system if not registered (but not on the new namespace)
	if systemClient1.GetOrganization(suite.T(), suite.testState.org1.Name) == nil {
		systemClient1.RegisterSelfOrg(suite.T(), true)
		systemClient1.RegisterSelfNode(suite.T(), true)
	}
	if systemClient2.GetOrganization(suite.T(), suite.testState.org2.Name) == nil {
		systemClient2.RegisterSelfOrg(suite.T(), true)
		systemClient2.RegisterSelfNode(suite.T(), true)
	}

	eventNames := "message_confirmed"
	queryString := fmt.Sprintf("namespace=%s&ephemeral&autoack&filter.events=%s&changeevents=.*", testNamespace, eventNames)
	ws1 := client1.WebSocket(suite.T(), queryString, nil)
	ws2 := client2.WebSocket(suite.T(), queryString, nil)
	received1 := e2e.WsReader(ws1, true)
	received2 := e2e.WsReader(ws2, true)

	// Verify that a broadcast on the new namespace succeeds under the V1 contract
	data := &core.DataRefOrValue{Value: fftypes.JSONAnyPtr(`"test"`)}
	resp, err := client1.BroadcastMessage(suite.T(), "topic", data, false)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 202, resp.StatusCode())

	e2e.WaitForMessageConfirmed(suite.T(), received1, core.MessageTypeBroadcast)
	e2e.WaitForMessageConfirmed(suite.T(), received2, core.MessageTypeBroadcast)

	// Register org/node identities on the new namespace and then migrate to the V2 contract
	client1.RegisterSelfOrg(suite.T(), true)
	client1.RegisterSelfNode(suite.T(), true)
	client2.RegisterSelfOrg(suite.T(), true)
	client2.RegisterSelfNode(suite.T(), true)
	client1.NetworkAction(suite.T(), core.NetworkActionTerminate)

	// TODO: remove
	// We should actually change the ws listener to listen for blockchain events too, and wait for the "terminate" events on ff_system
	time.Sleep(5 * time.Second)

	// Verify that a broadcast on the new namespace succeeds under the V2 contract
	resp, err = client1.BroadcastMessage(suite.T(), "topic", data, false)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 202, resp.StatusCode())

	e2e.WaitForMessageConfirmed(suite.T(), received1, core.MessageTypeBroadcast)
	e2e.WaitForMessageConfirmed(suite.T(), received2, core.MessageTypeBroadcast)

	// Verify the contract addresses for both blockchain events
	events := client1.GetBlockchainEvents(suite.T(), suite.testState.startTime)
	assert.Equal(suite.T(), address2, strings.ToLower(events[0].Info["address"].(string)))
	assert.Equal(suite.T(), address1, strings.ToLower(events[1].Info["address"].(string)))
}
