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
	"fmt"
	"time"

	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TokensTestSuite struct {
	suite.Suite
	testState *testState
}

func (suite *TokensTestSuite) BeforeTest(suiteName, testName string) {
	suite.testState = beforeE2ETest(suite.T())
}
func (suite *TokensTestSuite) TestE2ETokenPool() {
	defer suite.testState.done()

	received1, changes1 := wsReader(suite.T(), suite.testState.ws1)
	received2, changes2 := wsReader(suite.T(), suite.testState.ws2)

	pools := GetTokenPools(suite.T(), suite.testState.client1, time.Unix(0, 0))
	poolName := fmt.Sprintf("pool%d", len(pools))
	suite.T().Logf("Pool name: %s", poolName)

	pool := &fftypes.TokenPool{
		Name: poolName,
		Type: fftypes.TokenTypeFungible,
	}
	CreateTokenPool(suite.T(), suite.testState.client1, pool)

	<-received1
	<-changes1 // also expect database change events

	pools1 := GetTokenPools(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(pools1))
	assert.Equal(suite.T(), "default", pools1[0].Namespace)
	assert.Equal(suite.T(), poolName, pools1[0].Name)
	assert.Equal(suite.T(), fftypes.TokenTypeFungible, pools1[0].Type)

	<-received2
	<-changes2 // also expect database change events

	pools2 := GetTokenPools(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(pools2))
	assert.Equal(suite.T(), "default", pools2[0].Namespace)
	assert.Equal(suite.T(), poolName, pools2[0].Name)
	assert.Equal(suite.T(), fftypes.TokenTypeFungible, pools2[0].Type)
}
