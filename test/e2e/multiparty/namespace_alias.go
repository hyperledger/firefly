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
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/test/e2e"
	"github.com/hyperledger/firefly/test/e2e/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type NamespaceAliasSuite struct {
	suite.Suite
	testState   *testState
	stackName   string
	adminHost1  string
	adminHost2  string
	configFile1 string
	configFile2 string
}

func (suite *NamespaceAliasSuite) SetupSuite() {
	suite.testState = beforeE2ETest(suite.T())
	stack := e2e.ReadStack(suite.T())
	suite.stackName = stack.Name

	adminProtocol1 := schemeHTTP
	if stack.Members[0].UseHTTPS {
		adminProtocol1 = schemeHTTPS
	}
	adminProtocol2 := schemeHTTP
	if stack.Members[1].UseHTTPS {
		adminProtocol2 = schemeHTTPS
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

func (suite *NamespaceAliasSuite) BeforeTest(suiteName, testName string) {
	suite.testState = beforeE2ETest(suite.T())
}

func (suite *NamespaceAliasSuite) AfterTest(suiteName, testName string) {
	e2e.VerifyAllOperationsSucceeded(suite.T(), []*client.FireFlyClient{suite.testState.client1, suite.testState.client2}, suite.testState.startTime)
}

func (suite *NamespaceAliasSuite) TestNamespaceMapping() {
	defer suite.testState.Done()

	address := deployContract(suite.T(), suite.stackName, "firefly/Firefly.json")
	localNamespace1 := randomName(suite.T())
	localNamespace2 := randomName(suite.T())
	remoteNamespace := randomName(suite.T())
	suite.T().Logf("Test namespace: local1=%s, local2=%s, remote=%s", localNamespace1, localNamespace2, remoteNamespace)

	org := map[string]interface{}{}
	namespaceInfo := map[string]interface{}{
		"remotename": remoteNamespace,
		"multiparty": map[string]interface{}{
			"enabled": true,
			"org":     org,
			"contract": []map[string]interface{}{
				{
					"location": map[string]interface{}{"address": address},
				},
			},
		},
	}
	data := &core.DataRefOrValue{Value: fftypes.JSONAnyPtr(`"test"`)}

	// Add the new namespace to both config files
	data1 := readConfig(suite.T(), suite.configFile1)
	namespaceInfo["name"] = localNamespace1
	org["name"] = suite.testState.org1.Name
	org["key"] = suite.testState.org1key.Value
	addNamespace(data1, namespaceInfo)
	writeConfig(suite.T(), suite.configFile1, data1)

	data2 := readConfig(suite.T(), suite.configFile2)
	namespaceInfo["name"] = localNamespace2
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

	client1 := client.NewFireFly(suite.T(), suite.testState.client1.Hostname, localNamespace1)
	client2 := client.NewFireFly(suite.T(), suite.testState.client2.Hostname, localNamespace2)

	eventNames := "message_confirmed|blockchain_event_received"
	queryString := fmt.Sprintf("namespace=%s&ephemeral&autoack&filter.events=%s", localNamespace1, eventNames)
	received1 := e2e.WsReader(client1.WebSocket(suite.T(), queryString, nil))
	queryString = fmt.Sprintf("namespace=%s&ephemeral&autoack&filter.events=%s", localNamespace2, eventNames)
	received2 := e2e.WsReader(client2.WebSocket(suite.T(), queryString, nil))

	// Register org/node identities on the new namespace
	client1.RegisterSelfOrg(suite.T(), true)
	client1.RegisterSelfNode(suite.T(), true)
	client2.RegisterSelfOrg(suite.T(), true)
	client2.RegisterSelfNode(suite.T(), true)

	// Verify that a broadcast on the new namespace succeeds
	resp, err := client1.BroadcastMessage(suite.T(), "topic", data, false)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 202, resp.StatusCode())
	e2e.WaitForMessageConfirmed(suite.T(), received1, core.MessageTypeBroadcast)
	e2e.WaitForMessageConfirmed(suite.T(), received2, core.MessageTypeBroadcast)
}
