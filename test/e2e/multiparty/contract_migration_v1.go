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

	"github.com/hyperledger/firefly/test/e2e"
	"github.com/hyperledger/firefly/test/e2e/client"
)

type ContractMigrationV1TestSuite struct {
	ContractMigrationTestSuite
}

func (suite *ContractMigrationV1TestSuite) SetupSuite() {
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

func (suite *ContractMigrationV1TestSuite) BeforeTest(suiteName, testName string) {
	suite.testState = beforeE2ETest(suite.T())
}

func (suite *ContractMigrationV1TestSuite) AfterTest(suiteName, testName string) {
	e2e.VerifyAllOperationsSucceeded(suite.T(), []*client.FireFlyClient{suite.testState.client1, suite.testState.client2}, suite.testState.startTime)
}

func (suite *ContractMigrationV1TestSuite) TestContractMigration() {
	defer suite.testState.Done()

	address1 := deployContract(suite.T(), suite.stackName, "firefly/FireflyV1.json")
	address2 := deployContract(suite.T(), suite.stackName, "firefly/Firefly.json")
	runMigrationTest(&suite.ContractMigrationTestSuite, address1, address2, true)
}
