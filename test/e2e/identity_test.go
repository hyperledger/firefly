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
	// defer suite.testState.done()

	// received1, changes1 := wsReader(suite.testState.ws1)
	// received2, changes2 := wsReader(suite.testState.ws2)

	// // Broadcast some messages, that should get batched, across two topics
	// totalMessages := 10
	// topics := []string{"topicA", "topicB"}
	// expectedData := make(map[string][]*fftypes.DataRefOrValue)
	// for i := 0; i < 10; i++ {
	// 	value := fftypes.JSONAnyPtr(fmt.Sprintf(`"Hello number %d"`, i))
	// 	data := &fftypes.DataRefOrValue{
	// 		Value: value,
	// 	}
	// 	topic := pickTopic(i, topics)

	// 	expectedData[topic] = append(expectedData[topic], data)

	// 	resp, err := BroadcastMessage(suite.testState.client1, topic, data, false)
	// 	require.NoError(suite.T(), err)
	// 	assert.Equal(suite.T(), 202, resp.StatusCode())
	// }

	// for i := 0; i < totalMessages; i++ {
	// 	// Wait for all thel message-confirmed events, from both participants
	// 	waitForMessageConfirmed(suite.T(), received1, fftypes.MessageTypeBroadcast)
	// 	waitForMessageConfirmed(suite.T(), received2, fftypes.MessageTypeBroadcast)
	// 	<-changes1 // also expect database change events
	// 	<-changes2 // also expect database change events
	// }

	// for topic, dataArray := range expectedData {
	// 	receiver1data := validateReceivedMessages(suite.testState, suite.testState.client1, topic, fftypes.MessageTypeBroadcast, fftypes.TransactionTypeBatchPin, len(dataArray))
	// 	receiver2data := validateReceivedMessages(suite.testState, suite.testState.client2, topic, fftypes.MessageTypeBroadcast, fftypes.TransactionTypeBatchPin, len(dataArray))
	// 	// Messages should be returned in exactly reverse send order (newest first)
	// 	for i := (len(dataArray) - 1); i >= 0; i-- {
	// 		assert.Equal(suite.T(), dataArray[i].Value, receiver1data[i].Value)
	// 		assert.Equal(suite.T(), dataArray[i].Value, receiver2data[i].Value)
	// 	}
	// }

}
