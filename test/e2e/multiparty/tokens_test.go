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
}

func (suite *TokensTestSuite) SetupSuite() {
	suite.testState = beforeE2ETest(suite.T())
	stack := e2e.ReadStack(suite.T())
	suite.connector = stack.TokenProviders[0]
}

func (suite *TokensTestSuite) BeforeTest(suiteName, testName string) {
	suite.testState = beforeE2ETest(suite.T())
}

func (suite *TokensTestSuite) AfterTest(suiteName, testName string) {
	e2e.VerifyAllOperationsSucceeded(suite.T(), []*resty.Client{suite.testState.client1, suite.testState.client2}, suite.testState.startTime)
}

func (suite *TokensTestSuite) TestE2EFungibleTokensAsync() {
	defer suite.testState.done()

	received1 := e2e.WsReader(suite.testState.ws1, false)
	received2 := e2e.WsReader(suite.testState.ws2, false)

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

	e2e.WaitForEvent(suite.T(), received2, core.EventTypePoolConfirmed, poolID)
	pools = e2e.GetTokenPools(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(pools))
	assert.Equal(suite.T(), suite.testState.namespace, pools[0].Namespace)
	assert.Equal(suite.T(), suite.connector, pools[0].Connector)
	assert.Equal(suite.T(), poolName, pools[0].Name)
	assert.Equal(suite.T(), core.TokenTypeFungible, pools[0].Type)
	assert.NotEmpty(suite.T(), pools[0].Locator)

	approval := &core.TokenApprovalInput{
		TokenApproval: core.TokenApproval{
			Key:      suite.testState.org1key.Value,
			Operator: suite.testState.org2key.Value,
			Approved: true,
		},
		Pool: poolName,
	}
	approvalOut := e2e.TokenApproval(suite.T(), suite.testState.client1, approval, false)

	e2e.WaitForEvent(suite.T(), received1, core.EventTypeApprovalConfirmed, approvalOut.LocalID)
	approvals := e2e.GetTokenApprovals(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 1, len(approvals))
	assert.Equal(suite.T(), suite.connector, approvals[0].Connector)
	assert.Equal(suite.T(), true, approvals[0].Approved)

	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{Amount: *fftypes.NewFFBigInt(1)},
		Pool:          poolName,
	}
	transferOut := e2e.MintTokens(suite.T(), suite.testState.client1, transfer, false)

	e2e.WaitForEvent(suite.T(), received1, core.EventTypeTransferConfirmed, transferOut.LocalID)
	transfers := e2e.GetTokenTransfers(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 1, len(transfers))
	assert.Equal(suite.T(), suite.connector, transfers[0].Connector)
	assert.Equal(suite.T(), core.TokenTransferTypeMint, transfers[0].Type)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	e2e.ValidateAccountBalances(suite.T(), suite.testState.client1, poolID, "", map[string]int64{
		suite.testState.org1key.Value: 1,
	})

	e2e.WaitForEvent(suite.T(), received2, core.EventTypeTransferConfirmed, nil)
	transfers = e2e.GetTokenTransfers(suite.T(), suite.testState.client2, poolID)
	assert.Equal(suite.T(), 1, len(transfers))
	assert.Equal(suite.T(), suite.connector, transfers[0].Connector)
	assert.Equal(suite.T(), core.TokenTransferTypeMint, transfers[0].Type)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	e2e.ValidateAccountBalances(suite.T(), suite.testState.client2, poolID, "", map[string]int64{
		suite.testState.org1key.Value: 1,
	})

	transfer = &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			To:     suite.testState.org2key.Value,
			Amount: *fftypes.NewFFBigInt(1),
			From:   suite.testState.org1key.Value,
			Key:    suite.testState.org2key.Value,
		},
		Pool: poolName,
		Message: &core.MessageInOut{
			InlineData: core.InlineData{
				{
					Value: fftypes.JSONAnyPtr(`"token approval - payment for data"`),
				},
			},
		},
	}
	transferOut = e2e.TransferTokens(suite.T(), suite.testState.client2, transfer, false)

	e2e.WaitForEvent(suite.T(), received1, core.EventTypeMessageConfirmed, transferOut.Message)
	transfers = e2e.GetTokenTransfers(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 2, len(transfers))
	assert.Equal(suite.T(), suite.connector, transfers[0].Connector)
	assert.Equal(suite.T(), core.TokenTransferTypeTransfer, transfers[0].Type)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	data := e2e.GetDataForMessage(suite.T(), suite.testState.client1, suite.testState.startTime, transfers[0].Message)
	assert.Equal(suite.T(), 1, len(data))
	assert.Equal(suite.T(), `"token approval - payment for data"`, data[0].Value.String())
	e2e.ValidateAccountBalances(suite.T(), suite.testState.client1, poolID, "", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 1,
	})

	e2e.WaitForEvent(suite.T(), received2, core.EventTypeMessageConfirmed, transferOut.Message)
	transfers = e2e.GetTokenTransfers(suite.T(), suite.testState.client2, poolID)
	assert.Equal(suite.T(), 2, len(transfers))
	assert.Equal(suite.T(), suite.connector, transfers[0].Connector)
	assert.Equal(suite.T(), core.TokenTransferTypeTransfer, transfers[0].Type)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	e2e.ValidateAccountBalances(suite.T(), suite.testState.client2, poolID, "", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 1,
	})

	transfer = &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{Amount: *fftypes.NewFFBigInt(1)},
		Pool:          poolName,
	}
	transferOut = e2e.BurnTokens(suite.T(), suite.testState.client2, transfer, false)

	e2e.WaitForEvent(suite.T(), received2, core.EventTypeTransferConfirmed, transferOut.LocalID)
	transfers = e2e.GetTokenTransfers(suite.T(), suite.testState.client2, poolID)
	assert.Equal(suite.T(), 3, len(transfers))
	assert.Equal(suite.T(), suite.connector, transfers[0].Connector)
	assert.Equal(suite.T(), core.TokenTransferTypeBurn, transfers[0].Type)
	assert.Equal(suite.T(), "", transfers[0].TokenIndex)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	e2e.ValidateAccountBalances(suite.T(), suite.testState.client2, poolID, "", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 0,
	})

	e2e.WaitForEvent(suite.T(), received1, core.EventTypeTransferConfirmed, nil)
	transfers = e2e.GetTokenTransfers(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 3, len(transfers))
	assert.Equal(suite.T(), suite.connector, transfers[0].Connector)
	assert.Equal(suite.T(), core.TokenTransferTypeBurn, transfers[0].Type)
	assert.Equal(suite.T(), "", transfers[0].TokenIndex)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	e2e.ValidateAccountBalances(suite.T(), suite.testState.client1, poolID, "", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 0,
	})

	accounts := e2e.GetTokenAccounts(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 2, len(accounts))
	assert.Equal(suite.T(), suite.testState.org2key.Value, accounts[0].Key)
	assert.Equal(suite.T(), suite.testState.org1key.Value, accounts[1].Key)
	accounts = e2e.GetTokenAccounts(suite.T(), suite.testState.client2, poolID)
	assert.Equal(suite.T(), 2, len(accounts))
	assert.Equal(suite.T(), suite.testState.org2key.Value, accounts[0].Key)
	assert.Equal(suite.T(), suite.testState.org1key.Value, accounts[1].Key)

	accountPools := e2e.GetTokenAccountPools(suite.T(), suite.testState.client1, suite.testState.org1key.Value)
	assert.Equal(suite.T(), *poolID, *accountPools[0].Pool)
	accountPools = e2e.GetTokenAccountPools(suite.T(), suite.testState.client2, suite.testState.org2key.Value)
	assert.Equal(suite.T(), *poolID, *accountPools[0].Pool)
}

func (suite *TokensTestSuite) TestE2ENonFungibleTokensSync() {
	defer suite.testState.done()

	received1 := e2e.WsReader(suite.testState.ws1, false)
	received2 := e2e.WsReader(suite.testState.ws2, false)

	pools := e2e.GetTokenPools(suite.T(), suite.testState.client1, time.Unix(0, 0))
	rand.Seed(time.Now().UnixNano())
	poolName := fmt.Sprintf("pool%d", rand.Intn(10000))
	suite.T().Logf("Pool name: %s", poolName)

	pool := &core.TokenPool{
		Name:   poolName,
		Type:   core.TokenTypeNonFungible,
		Config: fftypes.JSONObject{},
	}

	poolOut := e2e.CreateTokenPool(suite.T(), suite.testState.client1, pool, true)
	assert.Equal(suite.T(), suite.testState.namespace, poolOut.Namespace)
	assert.Equal(suite.T(), poolName, poolOut.Name)
	assert.Equal(suite.T(), core.TokenTypeNonFungible, poolOut.Type)
	assert.NotEmpty(suite.T(), poolOut.Locator)

	poolID := poolOut.ID

	e2e.WaitForEvent(suite.T(), received1, core.EventTypePoolConfirmed, poolID)
	e2e.WaitForEvent(suite.T(), received2, core.EventTypePoolConfirmed, poolID)
	pools = e2e.GetTokenPools(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(pools))
	assert.Equal(suite.T(), suite.testState.namespace, pools[0].Namespace)
	assert.Equal(suite.T(), poolName, pools[0].Name)
	assert.Equal(suite.T(), core.TokenTypeNonFungible, pools[0].Type)
	assert.NotEmpty(suite.T(), pools[0].Locator)

	approval := &core.TokenApprovalInput{
		TokenApproval: core.TokenApproval{
			Key:      suite.testState.org1key.Value,
			Operator: suite.testState.org2key.Value,
			Approved: true,
		},
		Pool: poolName,
	}
	approvalOut := e2e.TokenApproval(suite.T(), suite.testState.client1, approval, true)

	e2e.WaitForEvent(suite.T(), received1, core.EventTypeApprovalConfirmed, approvalOut.LocalID)
	approvals := e2e.GetTokenApprovals(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 1, len(approvals))
	assert.Equal(suite.T(), suite.connector, approvals[0].Connector)
	assert.Equal(suite.T(), true, approvals[0].Approved)

	transfer := &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			TokenIndex: "1",
			Amount:     *fftypes.NewFFBigInt(1),
		},
		Pool: poolName,
	}
	transferOut := e2e.MintTokens(suite.T(), suite.testState.client1, transfer, true)
	assert.Equal(suite.T(), core.TokenTransferTypeMint, transferOut.Type)
	assert.Equal(suite.T(), "1", transferOut.TokenIndex)
	assert.Equal(suite.T(), int64(1), transferOut.Amount.Int().Int64())
	e2e.ValidateAccountBalances(suite.T(), suite.testState.client1, poolID, "1", map[string]int64{
		suite.testState.org1key.Value: 1,
	})

	e2e.WaitForEvent(suite.T(), received1, core.EventTypeTransferConfirmed, transferOut.LocalID)
	e2e.WaitForEvent(suite.T(), received2, core.EventTypeTransferConfirmed, nil)
	transfers := e2e.GetTokenTransfers(suite.T(), suite.testState.client2, poolID)
	assert.Equal(suite.T(), 1, len(transfers))
	assert.Equal(suite.T(), core.TokenTransferTypeMint, transfers[0].Type)
	assert.Equal(suite.T(), "1", transfers[0].TokenIndex)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	e2e.ValidateAccountBalances(suite.T(), suite.testState.client2, poolID, "1", map[string]int64{
		suite.testState.org1key.Value: 1,
	})

	transfer = &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			TokenIndex: "1",
			To:         suite.testState.org2key.Value,
			Amount:     *fftypes.NewFFBigInt(1),
			From:       suite.testState.org1key.Value,
			Key:        suite.testState.org1key.Value,
		},
		Pool: poolName,
		Message: &core.MessageInOut{
			InlineData: core.InlineData{
				{
					Value: fftypes.JSONAnyPtr(`"ownership change"`),
				},
			},
		},
	}
	transferOut = e2e.TransferTokens(suite.T(), suite.testState.client1, transfer, true)
	assert.Equal(suite.T(), core.TokenTransferTypeTransfer, transferOut.Type)
	assert.Equal(suite.T(), "1", transferOut.TokenIndex)
	assert.Equal(suite.T(), int64(1), transferOut.Amount.Int().Int64())
	data := e2e.GetDataForMessage(suite.T(), suite.testState.client1, suite.testState.startTime, transferOut.Message)
	assert.Equal(suite.T(), 1, len(data))
	assert.Equal(suite.T(), `"ownership change"`, data[0].Value.String())
	e2e.ValidateAccountBalances(suite.T(), suite.testState.client1, poolID, "1", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 1,
	})

	e2e.WaitForEvent(suite.T(), received1, core.EventTypeMessageConfirmed, transferOut.Message)
	e2e.WaitForEvent(suite.T(), received2, core.EventTypeMessageConfirmed, transferOut.Message)
	transfers = e2e.GetTokenTransfers(suite.T(), suite.testState.client2, poolID)
	assert.Equal(suite.T(), 2, len(transfers))
	assert.Equal(suite.T(), core.TokenTransferTypeTransfer, transfers[0].Type)
	assert.Equal(suite.T(), "1", transfers[0].TokenIndex)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	e2e.ValidateAccountBalances(suite.T(), suite.testState.client2, poolID, "1", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 1,
	})

	transfer = &core.TokenTransferInput{
		TokenTransfer: core.TokenTransfer{
			TokenIndex: "1",
			Amount:     *fftypes.NewFFBigInt(1),
		},
		Pool: poolName,
	}
	transferOut = e2e.BurnTokens(suite.T(), suite.testState.client2, transfer, true)
	assert.Equal(suite.T(), core.TokenTransferTypeBurn, transferOut.Type)
	assert.Equal(suite.T(), "1", transferOut.TokenIndex)
	assert.Equal(suite.T(), int64(1), transferOut.Amount.Int().Int64())
	e2e.ValidateAccountBalances(suite.T(), suite.testState.client2, poolID, "1", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 0,
	})

	e2e.WaitForEvent(suite.T(), received2, core.EventTypeTransferConfirmed, transferOut.LocalID)
	e2e.WaitForEvent(suite.T(), received1, core.EventTypeTransferConfirmed, nil)
	transfers = e2e.GetTokenTransfers(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 3, len(transfers))
	assert.Equal(suite.T(), core.TokenTransferTypeBurn, transfers[0].Type)
	assert.Equal(suite.T(), "1", transfers[0].TokenIndex)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	e2e.ValidateAccountBalances(suite.T(), suite.testState.client1, poolID, "1", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 0,
	})

	accounts := e2e.GetTokenAccounts(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 2, len(accounts))
	assert.Equal(suite.T(), suite.testState.org2key.Value, accounts[0].Key)
	assert.Equal(suite.T(), suite.testState.org1key.Value, accounts[1].Key)
	accounts = e2e.GetTokenAccounts(suite.T(), suite.testState.client2, poolID)
	assert.Equal(suite.T(), 2, len(accounts))
	assert.Equal(suite.T(), suite.testState.org2key.Value, accounts[0].Key)
	assert.Equal(suite.T(), suite.testState.org1key.Value, accounts[1].Key)

	accountPools := e2e.GetTokenAccountPools(suite.T(), suite.testState.client1, suite.testState.org1key.Value)
	assert.Equal(suite.T(), *poolID, *accountPools[0].Pool)
	accountPools = e2e.GetTokenAccountPools(suite.T(), suite.testState.client2, suite.testState.org2key.Value)
	assert.Equal(suite.T(), *poolID, *accountPools[0].Pool)
}
