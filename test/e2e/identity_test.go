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
	"time"

	"github.com/hyperledger/firefly/pkg/fftypes"
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

func (suite *IdentityTestSuite) TestCustomChildIdentityBroadcasts() {
	defer suite.testState.done()

	ctx := context.Background()
	received1, _ := wsReader(suite.testState.ws1, false)
	received2, _ := wsReader(suite.testState.ws2, false)

	// Create some keys
	totalIdentities := 3
	keys := make([]string, totalIdentities)
	for i := 0; i < totalIdentities; i++ {
		keys[i] = CreateEthAccount(suite.T(), suite.testState.ethNode)
	}

	ts := time.Now().Unix()
	for i := 0; i < totalIdentities; i++ {
		ClaimCustomIdentity(suite.T(),
			suite.testState.client1,
			keys[i],
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
		ed = waitForIdentityConfirmed(suite.T(), received2)
		identityIDs[*ed.Reference] = true
	}
	assert.Len(suite.T(), identityIDs, totalIdentities)

	identities := make(map[string]*fftypes.Identity)
	for identityID := range identityIDs {
		identityNode1 := GetIdentity(suite.T(), suite.testState.client1, &identityID)
		identityNode2 := GetIdentity(suite.T(), suite.testState.client1, &identityID)
		assert.True(suite.T(), identityNode1.IdentityBase.Equals(ctx, &identityNode2.IdentityBase))
		identities[identityNode1.DID] = identityNode1
	}

	// Send a broadcast from each custom identity
	for did := range identities {
		resp, err := BroadcastMessageAsIdentity(suite.T(), suite.testState.client1, did, "identitytest", &fftypes.DataRefOrValue{
			Value: fftypes.JSONAnyPtr(`{"some": "data"}`),
		}, false)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), 202, resp.StatusCode())
	}
	for range identities {
		waitForMessageConfirmed(suite.T(), received1, fftypes.MessageTypeBroadcast)
		waitForMessageConfirmed(suite.T(), received2, fftypes.MessageTypeBroadcast)
	}

}

func (suite *IdentityTestSuite) TestCustomChildIdentityPrivate() {
	defer suite.testState.done()

	received1, _ := wsReader(suite.testState.ws1, false)
	received2, _ := wsReader(suite.testState.ws2, false)

	// Create an identity on both sides
	org1Key := CreateEthAccount(suite.T(), suite.testState.ethNode)
	org2Key := CreateEthAccount(suite.T(), suite.testState.ethNode)

	ts := time.Now().Unix()
	custom1 := ClaimCustomIdentity(suite.T(),
		suite.testState.client1,
		org1Key,
		fmt.Sprintf("custom_%d_org1priv", ts),
		fmt.Sprintf("Description org1priv"),
		nil,
		suite.testState.org1.ID,
		true)
	custom2 := ClaimCustomIdentity(suite.T(),
		suite.testState.client2,
		org2Key,
		fmt.Sprintf("custom_%d_org2priv", ts),
		fmt.Sprintf("Description org2priv"),
		nil,
		suite.testState.org2.ID,
		true)
	for i := 0; i < 2; i++ {
		waitForIdentityConfirmed(suite.T(), received1)
		waitForIdentityConfirmed(suite.T(), received2)
	}

	resp, err := PrivateMessageWithKey(suite.testState, suite.testState.client1, org1Key, "topic1", &fftypes.DataRefOrValue{
		Value: fftypes.JSONAnyPtr(`"test private custom identity"`),
	}, []string{custom1.DID, custom2.DID}, "tag1", fftypes.TransactionTypeBatchPin, true)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 200, resp.StatusCode())

	waitForMessageConfirmed(suite.T(), received1, fftypes.MessageTypePrivate)
	waitForMessageConfirmed(suite.T(), received2, fftypes.MessageTypePrivate)

}
