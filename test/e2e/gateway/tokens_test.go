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
	"fmt"
	"math/rand"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/test/e2e"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TokensTestSuite struct {
	suite.Suite
	testState *testState
	connector string
	key       string
}

func (suite *TokensTestSuite) SetupSuite() {
	suite.testState = beforeE2ETest(suite.T())
	stack := e2e.ReadStack(suite.T())
	stackState := e2e.ReadStackState(suite.T())
	suite.connector = stack.TokenProviders[0]
	account := stackState.Accounts[0].(map[string]interface{})
	suite.key = account["address"].(string)
}

func (suite *TokensTestSuite) BeforeTest(suiteName, testName string) {
	suite.testState = beforeE2ETest(suite.T())
}

func (suite *TokensTestSuite) AfterTest(suiteName, testName string) {
	e2e.VerifyAllOperationsSucceeded(suite.T(), []*resty.Client{suite.testState.client1}, suite.testState.startTime)
}

func (suite *TokensTestSuite) TestE2EFungibleTokensAsync() {
	defer suite.testState.done()

	received1 := e2e.WsReader(suite.testState.ws1, false)

	pools := e2e.GetTokenPools(suite.T(), suite.testState.client1, time.Unix(0, 0))
	rand.Seed(time.Now().UnixNano())
	poolName := fmt.Sprintf("pool%d", rand.Intn(10000))
	suite.T().Logf("Pool name: %s", poolName)

	pool := &core.TokenPool{
		Name:   poolName,
		Type:   core.TokenTypeFungible,
		Config: fftypes.JSONObject{},
	}

	poolResp := e2e.CreateTokenPool(suite.T(), suite.testState.client1, pool, false)
	poolID := poolResp.ID

	e2e.WaitForEvent(suite.T(), received1, core.EventTypePoolConfirmed, poolID)
	pools = e2e.GetTokenPools(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(pools))
	assert.Equal(suite.T(), suite.testState.namespace, pools[0].Namespace)
	assert.Equal(suite.T(), suite.connector, pools[0].Connector)
	assert.Equal(suite.T(), poolName, pools[0].Name)
	assert.Equal(suite.T(), core.TokenTypeFungible, pools[0].Type)
	assert.NotEmpty(suite.T(), pools[0].Locator)

	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{Amount: *fftypes.NewFFBigInt(1)},
		Pool:          poolName,
	}
	transferOut := e2e.MintTokens(suite.T(), suite.testState.client1, transfer, false)
	e2e.WaitForEvent(suite.T(), received1, core.EventTypeTransferConfirmed, transferOut.LocalID)
	e2e.ValidateAccountBalances(suite.T(), suite.testState.client1, poolID, "", map[string]int64{
		suite.key: 1,
	})

	transfer = &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{Amount: *fftypes.NewFFBigInt(1)},
		Pool:          poolName,
	}
	transferOut = e2e.BurnTokens(suite.T(), suite.testState.client1, transfer, false)
	e2e.WaitForEvent(suite.T(), received1, core.EventTypeTransferConfirmed, transferOut.LocalID)
	e2e.ValidateAccountBalances(suite.T(), suite.testState.client1, poolID, "", map[string]int64{
		suite.key: 0,
	})
}
