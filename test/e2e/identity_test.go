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

func (suite *IdentityTestSuite) TestCustomChildIdentities() {
	defer suite.testState.done()

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
		resp, err := ClaimCustomIdentity(suite.testState.client1,
			keys[i],
			fmt.Sprintf("custom_%d_%d", ts, i),
			fmt.Sprintf("Description %d", i),
			fftypes.JSONObject{"profile": i},
			suite.testState.org1.ID,
			false)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), 202, resp.StatusCode())
	}

	identities := make(map[fftypes.UUID]bool)
	for i := 0; i < totalIdentities; i++ {
		ed := waitForIdentityConfirmed(suite.T(), received1)
		identities[*ed.Reference] = true
		ed = waitForIdentityConfirmed(suite.T(), received2)
		identities[*ed.Reference] = true
	}
	assert.Len(suite.T(), identities, totalIdentities)

	// for identityID := range identities {
	// 	identity :=
	// }

}
