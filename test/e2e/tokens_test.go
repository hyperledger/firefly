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

func (suite *TokensTestSuite) TestE2EFungibleTokensAsync() {
	defer suite.testState.done()

	received1, _ := wsReader(suite.testState.ws1, false)
	received2, _ := wsReader(suite.testState.ws2, false)

	pools := GetTokenPools(suite.T(), suite.testState.client1, time.Unix(0, 0))
	poolName := fmt.Sprintf("pool%d", len(pools))
	suite.T().Logf("Pool name: %s", poolName)

	pool := &fftypes.TokenPool{
		Name: poolName,
		Type: fftypes.TokenTypeFungible,
	}
	poolResp := CreateTokenPool(suite.T(), suite.testState.client1, pool, false)
	poolID := poolResp.ID

	waitForEvent(suite.T(), received1, fftypes.EventTypePoolConfirmed, poolID)
	pools = GetTokenPools(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(pools))
	assert.Equal(suite.T(), "default", pools[0].Namespace)
	assert.Equal(suite.T(), "erc1155", pools[0].Connector)
	assert.Equal(suite.T(), poolName, pools[0].Name)
	assert.Equal(suite.T(), fftypes.TokenTypeFungible, pools[0].Type)
	assert.NotEmpty(suite.T(), pools[0].ProtocolID)

	waitForEvent(suite.T(), received2, fftypes.EventTypePoolConfirmed, poolID)
	pools = GetTokenPools(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(pools))
	assert.Equal(suite.T(), "default", pools[0].Namespace)
	assert.Equal(suite.T(), "erc1155", pools[0].Connector)
	assert.Equal(suite.T(), poolName, pools[0].Name)
	assert.Equal(suite.T(), fftypes.TokenTypeFungible, pools[0].Type)
	assert.NotEmpty(suite.T(), pools[0].ProtocolID)

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Key:      suite.testState.org1key.Value,
			Operator: suite.testState.org2key.Value,
			Approved: true,
		},
	}
	approvalOut := TokenApproval(suite.T(), suite.testState.client1, approval, false)

	waitForEvent(suite.T(), received1, fftypes.EventTypeApprovalConfirmed, approvalOut.LocalID)
	approvals := GetTokenApprovals(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 1, len(approvals))
	assert.Equal(suite.T(), "erc1155", approvals[0].Connector)
	assert.Equal(suite.T(), true, approvals[0].Approved)

	transfer := &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{Amount: *fftypes.NewFFBigInt(1)},
		Pool:          poolName,
	}
	transferOut := MintTokens(suite.T(), suite.testState.client1, transfer, false)

	waitForEvent(suite.T(), received1, fftypes.EventTypeTransferConfirmed, transferOut.LocalID)
	transfers := GetTokenTransfers(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 1, len(transfers))
	assert.Equal(suite.T(), "erc1155", transfers[0].Connector)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeMint, transfers[0].Type)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client1, poolID, "", map[string]int64{
		suite.testState.org1key.Value: 1,
	})

	waitForEvent(suite.T(), received2, fftypes.EventTypeTransferConfirmed, nil)
	transfers = GetTokenTransfers(suite.T(), suite.testState.client2, poolID)
	assert.Equal(suite.T(), 1, len(transfers))
	assert.Equal(suite.T(), "erc1155", transfers[0].Connector)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeMint, transfers[0].Type)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client2, poolID, "", map[string]int64{
		suite.testState.org1key.Value: 1,
	})

	transfer = &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			To:     suite.testState.org2key.Value,
			Amount: *fftypes.NewFFBigInt(1),
			From:   suite.testState.org1key.Value,
			Key:    suite.testState.org2key.Value,
		},
		Pool: poolName,
		Message: &fftypes.MessageInOut{
			InlineData: fftypes.InlineData{
				{
					Value: fftypes.JSONAnyPtr(`"token approval - payment for data"`),
				},
			},
		},
	}
	transferOut = TransferTokens(suite.T(), suite.testState.client1, transfer, false)

	waitForEvent(suite.T(), received1, fftypes.EventTypeMessageConfirmed, transferOut.Message)
	transfers = GetTokenTransfers(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 2, len(transfers))
	assert.Equal(suite.T(), "erc1155", transfers[0].Connector)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeTransfer, transfers[0].Type)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	data := GetDataForMessage(suite.T(), suite.testState.client1, suite.testState.startTime, transfers[0].Message)
	assert.Equal(suite.T(), 1, len(data))
	assert.Equal(suite.T(), `"token approval - payment for data"`, data[0].Value.String())
	validateAccountBalances(suite.T(), suite.testState.client1, poolID, "", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 1,
	})

	waitForEvent(suite.T(), received2, fftypes.EventTypeMessageConfirmed, transferOut.Message)
	transfers = GetTokenTransfers(suite.T(), suite.testState.client2, poolID)
	assert.Equal(suite.T(), 2, len(transfers))
	assert.Equal(suite.T(), "erc1155", transfers[0].Connector)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeTransfer, transfers[0].Type)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client2, poolID, "", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 1,
	})

	transfer = &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{Amount: *fftypes.NewFFBigInt(1)},
		Pool:          poolName,
	}
	transferOut = BurnTokens(suite.T(), suite.testState.client2, transfer, false)

	waitForEvent(suite.T(), received2, fftypes.EventTypeTransferConfirmed, transferOut.LocalID)
	transfers = GetTokenTransfers(suite.T(), suite.testState.client2, poolID)
	assert.Equal(suite.T(), 3, len(transfers))
	assert.Equal(suite.T(), "erc1155", transfers[0].Connector)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeBurn, transfers[0].Type)
	assert.Equal(suite.T(), "", transfers[0].TokenIndex)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client2, poolID, "", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 0,
	})

	waitForEvent(suite.T(), received1, fftypes.EventTypeTransferConfirmed, nil)
	transfers = GetTokenTransfers(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 3, len(transfers))
	assert.Equal(suite.T(), "erc1155", transfers[0].Connector)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeBurn, transfers[0].Type)
	assert.Equal(suite.T(), "", transfers[0].TokenIndex)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client1, poolID, "", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 0,
	})

	accounts := GetTokenAccounts(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 2, len(accounts))
	assert.Equal(suite.T(), suite.testState.org2key.Value, accounts[0].Key)
	assert.Equal(suite.T(), suite.testState.org1key.Value, accounts[1].Key)
	accounts = GetTokenAccounts(suite.T(), suite.testState.client2, poolID)
	assert.Equal(suite.T(), 2, len(accounts))
	assert.Equal(suite.T(), suite.testState.org2key.Value, accounts[0].Key)
	assert.Equal(suite.T(), suite.testState.org1key.Value, accounts[1].Key)

	accountPools := GetTokenAccountPools(suite.T(), suite.testState.client1, suite.testState.org1key.Value)
	assert.Equal(suite.T(), *poolID, *accountPools[0].Pool)
	accountPools = GetTokenAccountPools(suite.T(), suite.testState.client2, suite.testState.org2key.Value)
	assert.Equal(suite.T(), *poolID, *accountPools[0].Pool)
}

func (suite *TokensTestSuite) TestE2ENonFungibleTokensSync() {
	defer suite.testState.done()

	received1, _ := wsReader(suite.testState.ws1, false)
	received2, _ := wsReader(suite.testState.ws2, false)

	pools := GetTokenPools(suite.T(), suite.testState.client1, time.Unix(0, 0))
	poolName := fmt.Sprintf("pool%d", len(pools))
	suite.T().Logf("Pool name: %s", poolName)

	pool := &fftypes.TokenPool{
		Name: poolName,
		Type: fftypes.TokenTypeNonFungible,
	}
	poolOut := CreateTokenPool(suite.T(), suite.testState.client1, pool, true)
	assert.Equal(suite.T(), "default", poolOut.Namespace)
	assert.Equal(suite.T(), poolName, poolOut.Name)
	assert.Equal(suite.T(), fftypes.TokenTypeNonFungible, poolOut.Type)
	assert.NotEmpty(suite.T(), poolOut.ProtocolID)

	poolID := poolOut.ID

	waitForEvent(suite.T(), received1, fftypes.EventTypePoolConfirmed, poolID)
	waitForEvent(suite.T(), received2, fftypes.EventTypePoolConfirmed, poolID)
	pools = GetTokenPools(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(pools))
	assert.Equal(suite.T(), "default", pools[0].Namespace)
	assert.Equal(suite.T(), poolName, pools[0].Name)
	assert.Equal(suite.T(), fftypes.TokenTypeNonFungible, pools[0].Type)
	assert.NotEmpty(suite.T(), pools[0].ProtocolID)

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Key:      suite.testState.org1key.Value,
			Operator: suite.testState.org2key.Value,
			Approved: true,
		},
		Pool: poolName,
	}
	approvalOut := TokenApproval(suite.T(), suite.testState.client1, approval, true)

	waitForEvent(suite.T(), received1, fftypes.EventTypeApprovalConfirmed, approvalOut.LocalID)
	approvals := GetTokenApprovals(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 1, len(approvals))
	assert.Equal(suite.T(), "erc1155", approvals[0].Connector)
	assert.Equal(suite.T(), true, approvals[0].Approved)

	transfer := &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{Amount: *fftypes.NewFFBigInt(1)},
		Pool:          poolName,
	}
	transferOut := MintTokens(suite.T(), suite.testState.client1, transfer, true)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeMint, transferOut.Type)
	assert.Equal(suite.T(), "1", transferOut.TokenIndex)
	assert.Equal(suite.T(), int64(1), transferOut.Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client1, poolID, "1", map[string]int64{
		suite.testState.org1key.Value: 1,
	})

	waitForEvent(suite.T(), received1, fftypes.EventTypeTransferConfirmed, transferOut.LocalID)
	waitForEvent(suite.T(), received2, fftypes.EventTypeTransferConfirmed, nil)
	transfers := GetTokenTransfers(suite.T(), suite.testState.client2, poolID)
	assert.Equal(suite.T(), 1, len(transfers))
	assert.Equal(suite.T(), fftypes.TokenTransferTypeMint, transfers[0].Type)
	assert.Equal(suite.T(), "1", transfers[0].TokenIndex)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client2, poolID, "1", map[string]int64{
		suite.testState.org1key.Value: 1,
	})

	transfer = &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			TokenIndex: "1",
			To:         suite.testState.org2key.Value,
			Amount:     *fftypes.NewFFBigInt(1),
			From:       suite.testState.org1key.Value,
			Key:        suite.testState.org2key.Value,
		},
		Pool: poolName,
		Message: &fftypes.MessageInOut{
			InlineData: fftypes.InlineData{
				{
					Value: fftypes.JSONAnyPtr(`"ownership change"`),
				},
			},
		},
	}
	transferOut = TransferTokens(suite.T(), suite.testState.client1, transfer, true)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeTransfer, transferOut.Type)
	assert.Equal(suite.T(), "1", transferOut.TokenIndex)
	assert.Equal(suite.T(), int64(1), transferOut.Amount.Int().Int64())
	data := GetDataForMessage(suite.T(), suite.testState.client1, suite.testState.startTime, transferOut.Message)
	assert.Equal(suite.T(), 1, len(data))
	assert.Equal(suite.T(), `"ownership change"`, data[0].Value.String())
	validateAccountBalances(suite.T(), suite.testState.client1, poolID, "1", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 1,
	})

	waitForEvent(suite.T(), received1, fftypes.EventTypeMessageConfirmed, transferOut.Message)
	waitForEvent(suite.T(), received2, fftypes.EventTypeMessageConfirmed, transferOut.Message)
	transfers = GetTokenTransfers(suite.T(), suite.testState.client2, poolID)
	assert.Equal(suite.T(), 2, len(transfers))
	assert.Equal(suite.T(), fftypes.TokenTransferTypeTransfer, transfers[0].Type)
	assert.Equal(suite.T(), "1", transfers[0].TokenIndex)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client2, poolID, "1", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 1,
	})

	transfer = &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			TokenIndex: "1",
			Amount:     *fftypes.NewFFBigInt(1),
		},
		Pool: poolName,
	}
	transferOut = BurnTokens(suite.T(), suite.testState.client2, transfer, true)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeBurn, transferOut.Type)
	assert.Equal(suite.T(), "1", transferOut.TokenIndex)
	assert.Equal(suite.T(), int64(1), transferOut.Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client2, poolID, "1", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 0,
	})

	waitForEvent(suite.T(), received2, fftypes.EventTypeTransferConfirmed, transferOut.LocalID)
	waitForEvent(suite.T(), received1, fftypes.EventTypeTransferConfirmed, nil)
	transfers = GetTokenTransfers(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 3, len(transfers))
	assert.Equal(suite.T(), fftypes.TokenTransferTypeBurn, transfers[0].Type)
	assert.Equal(suite.T(), "1", transfers[0].TokenIndex)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client1, poolID, "1", map[string]int64{
		suite.testState.org1key.Value: 0,
		suite.testState.org2key.Value: 0,
	})

	accounts := GetTokenAccounts(suite.T(), suite.testState.client1, poolID)
	assert.Equal(suite.T(), 2, len(accounts))
	assert.Equal(suite.T(), suite.testState.org2key.Value, accounts[0].Key)
	assert.Equal(suite.T(), suite.testState.org1key.Value, accounts[1].Key)
	accounts = GetTokenAccounts(suite.T(), suite.testState.client2, poolID)
	assert.Equal(suite.T(), 2, len(accounts))
	assert.Equal(suite.T(), suite.testState.org2key.Value, accounts[0].Key)
	assert.Equal(suite.T(), suite.testState.org1key.Value, accounts[1].Key)

	accountPools := GetTokenAccountPools(suite.T(), suite.testState.client1, suite.testState.org1key.Value)
	assert.Equal(suite.T(), *poolID, *accountPools[0].Pool)
	accountPools = GetTokenAccountPools(suite.T(), suite.testState.client2, suite.testState.org2key.Value)
	assert.Equal(suite.T(), *poolID, *accountPools[0].Pool)
}
