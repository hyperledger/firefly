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
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IdentityTestSuite struct {
	suite.Suite
	testState *testState
}

func (suite *IdentityTestSuite) BeforeTest(suiteName, testName string) {
	suite.testState = beforeE2ETest(suite.T())
}

func (suite *IdentityTestSuite) AfterTest(suiteName, testName string) {
	verifyAllOperationsSucceeded(suite.T(), []*resty.Client{suite.testState.client1, suite.testState.client2}, suite.testState.startTime)
}

func (suite *IdentityTestSuite) TestCustomChildIdentityBroadcasts() {
	defer suite.testState.done()

	ctx := context.Background()
	received1 := wsReader(suite.testState.ws1, false)
	received2 := wsReader(suite.testState.ws2, false)

	totalIdentities := 2
	ts := time.Now().Unix()
	for i := 0; i < totalIdentities; i++ {
		key := getUnregisteredAccount(suite, suite.testState.org1.Name)
		ClaimCustomIdentity(suite.T(),
			suite.testState.client1,
			key,
			fmt.Sprintf("custom_%d_%d", ts, i),
			fmt.Sprintf("Description %d", i),
			fftypes.JSONObject{"profile": i},
			suite.testState.org1.ID,
			false)
	}

	identityIDs := make(map[fftypes.UUID]bool)
	for i := 0; i < totalIdentities; i++ {
		ed := waitForIdentityConfirmed(suite.T(), received1)
		identityIDs[*ed.Reference] = true
		suite.T().Logf("Received node 1 confirmation of identity %s", ed.Reference)
		ed = waitForIdentityConfirmed(suite.T(), received2)
		identityIDs[*ed.Reference] = true
		suite.T().Logf("Received node 2 confirmation of identity %s", ed.Reference)
	}
	assert.Len(suite.T(), identityIDs, totalIdentities)

	identities := make(map[string]*core.Identity)
	for identityID := range identityIDs {
		identityNode1 := GetIdentity(suite.T(), suite.testState.client1, &identityID)
		identityNode2 := GetIdentity(suite.T(), suite.testState.client1, &identityID)
		assert.True(suite.T(), identityNode1.IdentityBase.Equals(ctx, &identityNode2.IdentityBase))
		identities[identityNode1.DID] = identityNode1
	}

	// Send a broadcast from each custom identity
	for did := range identities {
		resp, err := BroadcastMessageAsIdentity(suite.T(), suite.testState.client1, did, "identitytest", &core.DataRefOrValue{
			Value: fftypes.JSONAnyPtr(`{"some": "data"}`),
		}, false)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), 202, resp.StatusCode())
	}
	for range identities {
		waitForMessageConfirmed(suite.T(), received1, core.MessageTypeBroadcast)
		waitForMessageConfirmed(suite.T(), received2, core.MessageTypeBroadcast)
	}

}

func (suite *IdentityTestSuite) TestCustomChildIdentityPrivate() {
	defer suite.testState.done()

	received1 := wsReader(suite.testState.ws1, false)
	received2 := wsReader(suite.testState.ws2, false)

	org1key := getUnregisteredAccount(suite, suite.testState.org1.Name)
	org2key := getUnregisteredAccount(suite, suite.testState.org2.Name)

	ts := time.Now().Unix()
	custom1 := ClaimCustomIdentity(suite.T(),
		suite.testState.client1,
		org1key,
		fmt.Sprintf("custom_%d_org1priv", ts),
		fmt.Sprintf("Description org1priv"),
		nil,
		suite.testState.org1.ID,
		true)
	custom2 := ClaimCustomIdentity(suite.T(),
		suite.testState.client2,
		org2key,
		fmt.Sprintf("custom_%d_org2priv", ts),
		fmt.Sprintf("Description org2priv"),
		nil,
		suite.testState.org2.ID,
		true)
	for i := 0; i < 2; i++ {
		waitForIdentityConfirmed(suite.T(), received1)
		waitForIdentityConfirmed(suite.T(), received2)
	}

	resp, err := PrivateMessageWithKey(suite.testState, suite.testState.client1, org1key, "topic1", &core.DataRefOrValue{
		Value: fftypes.JSONAnyPtr(`"test private custom identity"`),
	}, []string{custom1.DID, custom2.DID}, "tag1", core.TransactionTypeBatchPin, true)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 200, resp.StatusCode())

	waitForMessageConfirmed(suite.T(), received1, core.MessageTypePrivate)
	waitForMessageConfirmed(suite.T(), received2, core.MessageTypePrivate)
}

func getUnregisteredAccount(suite *IdentityTestSuite, orgName string) string {
	verifiers := GetVerifiers(suite.T(), suite.testState.client1)
	suite.T().Logf("checking for account with orgName: %s", orgName)
	for i, account := range suite.testState.unregisteredAccounts {
		alreadyRegisted := false
		accountMap := account.(map[string]interface{})
		suite.T().Logf("account from stackState has orgName: %s", accountMap["orgName"])
		if accountOrgName, ok := accountMap["orgName"]; ok {
			if accountOrgName != orgName {
				// An orgName was present and it wasn't what we were looking for. Skip.
				continue
			}
		}
		var key string
		if k, ok := accountMap["address"]; ok {
			key = k.(string)
		}
		if k, ok := accountMap["name"]; ok {
			key = k.(string)
		}

		suite.T().Logf("checking to make sure key '%s' is not registered", key)
		if len(verifiers) > 0 {
			for _, verifier := range verifiers {

				suite.T().Logf("account name/address: %s", key)
				suite.T().Logf("verifier value: %s", verifier.Value)
				if strings.Contains(verifier.Value, key) {
					// Already registered. Look at the next account
					alreadyRegisted = true
					break
				}
			}
		}
		if !alreadyRegisted {
			suite.testState.unregisteredAccounts = append(suite.testState.unregisteredAccounts[:i], suite.testState.unregisteredAccounts[i+1:]...)
			return key
		}
	}
	return ""
}
