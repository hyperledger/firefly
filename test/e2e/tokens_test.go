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

	received1, _ := wsReader(suite.T(), suite.testState.ws1)
	received2, _ := wsReader(suite.T(), suite.testState.ws2)

	pools := GetTokenPools(suite.T(), suite.testState.client1, time.Unix(0, 0))
	poolName := fmt.Sprintf("pool%d", len(pools))
	suite.T().Logf("Pool name: %s", poolName)

	pool := &fftypes.TokenPool{
		Name: poolName,
		Type: fftypes.TokenTypeFungible,
	}
	CreateTokenPool(suite.T(), suite.testState.client1, pool, false)

	<-received1 // event for token pool creation
	<-received1 // event for token pool announcement
	pools = GetTokenPools(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(pools))
	assert.Equal(suite.T(), "default", pools[0].Namespace)
	assert.Equal(suite.T(), "erc1155", pools[0].Connector)
	assert.Equal(suite.T(), poolName, pools[0].Name)
	assert.Equal(suite.T(), fftypes.TokenTypeFungible, pools[0].Type)
	assert.NotEmpty(suite.T(), pools[0].ProtocolID)

	<-received2 // event for token pool creation
	<-received2 // event for token pool announcement
	pools = GetTokenPools(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(pools))
	assert.Equal(suite.T(), "default", pools[0].Namespace)
	assert.Equal(suite.T(), "erc1155", pools[0].Connector)
	assert.Equal(suite.T(), poolName, pools[0].Name)
	assert.Equal(suite.T(), fftypes.TokenTypeFungible, pools[0].Type)
	assert.NotEmpty(suite.T(), pools[0].ProtocolID)

	transfer := &fftypes.TokenTransferInput{}
	transfer.Amount.Int().SetInt64(1)
	MintTokens(suite.T(), suite.testState.client1, poolName, transfer, false)

	<-received1
	transfers := GetTokenTransfers(suite.T(), suite.testState.client1, poolName)
	assert.Equal(suite.T(), 1, len(transfers))
	assert.Equal(suite.T(), "erc1155", transfers[0].Connector)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeMint, transfers[0].Type)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client1, poolName, "", map[string]int64{
		suite.testState.org1.Identity: 1,
	})

	<-received2
	transfers = GetTokenTransfers(suite.T(), suite.testState.client2, poolName)
	assert.Equal(suite.T(), 1, len(transfers))
	assert.Equal(suite.T(), "erc1155", transfers[0].Connector)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeMint, transfers[0].Type)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client2, poolName, "", map[string]int64{
		suite.testState.org1.Identity: 1,
	})

	transfer = &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			To: suite.testState.org2.Identity,
		},
		Message: &fftypes.MessageInOut{
			InlineData: fftypes.InlineData{
				{
					Value: fftypes.Byteable(`"payment for data"`),
				},
			},
		},
	}
	transfer.Amount.Int().SetInt64(1)
	TransferTokens(suite.T(), suite.testState.client1, poolName, transfer, false)

	<-received1 // one event for transfer
	<-received1 // one event for message
	transfers = GetTokenTransfers(suite.T(), suite.testState.client1, poolName)
	assert.Equal(suite.T(), 2, len(transfers))
	assert.Equal(suite.T(), "erc1155", transfers[0].Connector)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeTransfer, transfers[0].Type)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	data := GetDataForMessage(suite.T(), suite.testState.client1, suite.testState.startTime, transfers[0].MessageHash)
	assert.Equal(suite.T(), 1, len(data))
	assert.Equal(suite.T(), `"payment for data"`, data[0].Value.String())
	validateAccountBalances(suite.T(), suite.testState.client1, poolName, "", map[string]int64{
		suite.testState.org1.Identity: 0,
		suite.testState.org2.Identity: 1,
	})

	<-received2 // one event for transfer
	<-received2 // one event for message
	transfers = GetTokenTransfers(suite.T(), suite.testState.client2, poolName)
	assert.Equal(suite.T(), 2, len(transfers))
	assert.Equal(suite.T(), "erc1155", transfers[0].Connector)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeTransfer, transfers[0].Type)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client2, poolName, "", map[string]int64{
		suite.testState.org1.Identity: 0,
		suite.testState.org2.Identity: 1,
	})

	transfer = &fftypes.TokenTransferInput{}
	transfer.Amount.Int().SetInt64(1)
	BurnTokens(suite.T(), suite.testState.client2, poolName, transfer, false)

	<-received2
	transfers = GetTokenTransfers(suite.T(), suite.testState.client2, poolName)
	assert.Equal(suite.T(), 3, len(transfers))
	assert.Equal(suite.T(), "erc1155", transfers[0].Connector)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeBurn, transfers[0].Type)
	assert.Equal(suite.T(), "", transfers[0].TokenIndex)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client2, poolName, "", map[string]int64{
		suite.testState.org1.Identity: 0,
		suite.testState.org2.Identity: 0,
	})

	<-received1
	transfers = GetTokenTransfers(suite.T(), suite.testState.client1, poolName)
	assert.Equal(suite.T(), 3, len(transfers))
	assert.Equal(suite.T(), "erc1155", transfers[0].Connector)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeBurn, transfers[0].Type)
	assert.Equal(suite.T(), "", transfers[0].TokenIndex)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client1, poolName, "", map[string]int64{
		suite.testState.org1.Identity: 0,
		suite.testState.org2.Identity: 0,
	})
}

func (suite *TokensTestSuite) TestE2ENonFungibleTokensSync() {
	defer suite.testState.done()

	received1, _ := wsReader(suite.T(), suite.testState.ws1)
	received2, _ := wsReader(suite.T(), suite.testState.ws2)

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

	<-received1 // event for token pool creation
	<-received2 // event for token pool announcement
	<-received1 // event for token pool creation
	<-received2 // event for token pool announcement
	pools = GetTokenPools(suite.T(), suite.testState.client1, suite.testState.startTime)
	assert.Equal(suite.T(), 1, len(pools))
	assert.Equal(suite.T(), "default", pools[0].Namespace)
	assert.Equal(suite.T(), poolName, pools[0].Name)
	assert.Equal(suite.T(), fftypes.TokenTypeNonFungible, pools[0].Type)
	assert.NotEmpty(suite.T(), pools[0].ProtocolID)

	transfer := &fftypes.TokenTransferInput{}
	transfer.Amount.Int().SetInt64(1)
	transferOut := MintTokens(suite.T(), suite.testState.client1, poolName, transfer, true)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeMint, transferOut.Type)
	assert.Equal(suite.T(), "1", transferOut.TokenIndex)
	assert.Equal(suite.T(), int64(1), transferOut.Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client1, poolName, "1", map[string]int64{
		suite.testState.org1.Identity: 1,
	})

	<-received1
	<-received2
	transfers := GetTokenTransfers(suite.T(), suite.testState.client2, poolName)
	assert.Equal(suite.T(), 1, len(transfers))
	assert.Equal(suite.T(), fftypes.TokenTransferTypeMint, transfers[0].Type)
	assert.Equal(suite.T(), "1", transfers[0].TokenIndex)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client2, poolName, "1", map[string]int64{
		suite.testState.org1.Identity: 1,
	})

	transfer = &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			TokenIndex: "1",
			To:         suite.testState.org2.Identity,
		},
		Message: &fftypes.MessageInOut{
			InlineData: fftypes.InlineData{
				{
					Value: fftypes.Byteable(`"ownership change"`),
				},
			},
		},
	}
	transfer.Amount.Int().SetInt64(1)
	transferOut = TransferTokens(suite.T(), suite.testState.client1, poolName, transfer, true)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeTransfer, transferOut.Type)
	assert.Equal(suite.T(), "1", transferOut.TokenIndex)
	assert.Equal(suite.T(), int64(1), transferOut.Amount.Int().Int64())
	data := GetDataForMessage(suite.T(), suite.testState.client1, suite.testState.startTime, transferOut.MessageHash)
	assert.Equal(suite.T(), 1, len(data))
	assert.Equal(suite.T(), `"ownership change"`, data[0].Value.String())
	validateAccountBalances(suite.T(), suite.testState.client1, poolName, "1", map[string]int64{
		suite.testState.org1.Identity: 0,
		suite.testState.org2.Identity: 1,
	})

	<-received1 // one event for transfer
	<-received1 // one event for message
	<-received2 // one event for transfer
	<-received2 // one event for message
	transfers = GetTokenTransfers(suite.T(), suite.testState.client2, poolName)
	assert.Equal(suite.T(), 2, len(transfers))
	assert.Equal(suite.T(), fftypes.TokenTransferTypeTransfer, transfers[0].Type)
	assert.Equal(suite.T(), "1", transfers[0].TokenIndex)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client2, poolName, "1", map[string]int64{
		suite.testState.org1.Identity: 0,
		suite.testState.org2.Identity: 1,
	})

	transfer = &fftypes.TokenTransferInput{
		TokenTransfer: fftypes.TokenTransfer{
			TokenIndex: "1",
		},
	}
	transfer.Amount.Int().SetInt64(1)
	transferOut = BurnTokens(suite.T(), suite.testState.client2, poolName, transfer, true)
	assert.Equal(suite.T(), fftypes.TokenTransferTypeBurn, transferOut.Type)
	assert.Equal(suite.T(), "1", transferOut.TokenIndex)
	assert.Equal(suite.T(), int64(1), transferOut.Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client2, poolName, "1", map[string]int64{
		suite.testState.org1.Identity: 0,
		suite.testState.org2.Identity: 0,
	})

	<-received2
	<-received1
	transfers = GetTokenTransfers(suite.T(), suite.testState.client1, poolName)
	assert.Equal(suite.T(), 3, len(transfers))
	assert.Equal(suite.T(), fftypes.TokenTransferTypeBurn, transfers[0].Type)
	assert.Equal(suite.T(), "1", transfers[0].TokenIndex)
	assert.Equal(suite.T(), int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(suite.T(), suite.testState.client1, poolName, "1", map[string]int64{
		suite.testState.org1.Identity: 0,
		suite.testState.org2.Identity: 0,
	})
}
