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
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	image2ascii "github.com/qeesung/image2ascii/convert"
)

type OnChainOffChainTestSuite struct {
	suite.Suite
	testState *testState
}

func (suite *OnChainOffChainTestSuite) BeforeTest(suiteName, testName string) {
	suite.testState = beforeE2ETest(suite.T())
}

func (suite *OnChainOffChainTestSuite) TestE2EBroadcast() {
	defer suite.testState.done()

	received1, changes1 := wsReader(suite.T(), suite.testState.ws1)
	received2, changes2 := wsReader(suite.T(), suite.testState.ws2)

	var resp *resty.Response
	value := fftypes.Byteable(`"Hello"`)
	data := fftypes.DataRefOrValue{
		Value: value,
	}

	resp, err := BroadcastMessage(suite.testState.client1, &data)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 202, resp.StatusCode())

	<-received1
	<-changes1 // also expect database change events
	val1 := validateReceivedMessages(suite.testState, suite.testState.client1, fftypes.MessageTypeBroadcast, fftypes.TransactionTypeBatchPin, 1, 0)
	assert.Equal(suite.T(), data.Value, val1)

	<-received2
	<-changes2 // also expect database change events
	val2 := validateReceivedMessages(suite.testState, suite.testState.client2, fftypes.MessageTypeBroadcast, fftypes.TransactionTypeBatchPin, 1, 0)
	assert.Equal(suite.T(), data.Value, val2)

}

func (suite *OnChainOffChainTestSuite) TestE2EPrivate() {
	defer suite.testState.done()

	received1, _ := wsReader(suite.T(), suite.testState.ws1)
	received2, _ := wsReader(suite.T(), suite.testState.ws2)

	var resp *resty.Response
	value := fftypes.Byteable(`"Hello"`)
	data := fftypes.DataRefOrValue{
		Value: value,
	}

	resp, err := PrivateMessage(suite.T(), suite.testState.client1, &data, []string{
		suite.testState.org1.Name,
		suite.testState.org2.Name,
	}, "", fftypes.TransactionTypeBatchPin)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 202, resp.StatusCode())

	<-received1
	val1 := validateReceivedMessages(suite.testState, suite.testState.client1, fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 1, 0)
	assert.Equal(suite.T(), data.Value, val1)

	<-received2
	val2 := validateReceivedMessages(suite.testState, suite.testState.client2, fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 1, 0)
	assert.Equal(suite.T(), data.Value, val2)
}

func (suite *OnChainOffChainTestSuite) TestE2EBroadcastBlob() {
	defer suite.testState.done()

	received1, _ := wsReader(suite.T(), suite.testState.ws1)
	received2, _ := wsReader(suite.T(), suite.testState.ws2)

	var resp *resty.Response

	resp, err := BroadcastBlobMessage(suite.T(), suite.testState.client1)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 202, resp.StatusCode())

	<-received1
	val1 := validateReceivedMessages(suite.testState, suite.testState.client1, fftypes.MessageTypeBroadcast, fftypes.TransactionTypeBatchPin, 1, 0)
	assert.Regexp(suite.T(), "myfile.txt", string(val1))

	<-received2
	val2 := validateReceivedMessages(suite.testState, suite.testState.client2, fftypes.MessageTypeBroadcast, fftypes.TransactionTypeBatchPin, 1, 0)
	assert.Regexp(suite.T(), "myfile.txt", string(val2))

}

func (suite *OnChainOffChainTestSuite) TestE2EPrivateBlobDatatypeTagged() {
	defer suite.testState.done()

	received1, _ := wsReader(suite.T(), suite.testState.ws1)
	received2, _ := wsReader(suite.T(), suite.testState.ws2)

	var resp *resty.Response

	resp, err := PrivateBlobMessageDatatypeTagged(suite.T(), suite.testState.client1, []string{
		suite.testState.org1.Name,
		suite.testState.org2.Name,
	})
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 202, resp.StatusCode())

	<-received1
	_ = validateReceivedMessages(suite.testState, suite.testState.client1, fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 1, 0)

	<-received2
	_ = validateReceivedMessages(suite.testState, suite.testState.client2, fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 1, 0)
}

func (suite *OnChainOffChainTestSuite) TestE2EWebhookExchange() {
	defer suite.testState.done()

	received1, _ := wsReader(suite.T(), suite.testState.ws1)
	received2, _ := wsReader(suite.T(), suite.testState.ws2)

	subJSON := `{
		"transport": "webhooks",
		"namespace": "default",
		"name": "myhook",
		"options": {
			"withData": true,
			"url": "https://raw.githubusercontent.com/hyperledger/firefly/main/test/data/config/firefly.core.yaml",
			"reply": true,
			"replytag": "myreply",
			"method": "GET"
		},
		"filter": {
			"tag": "myrequest"
		}
	}`
	CleanupExistingSubscription(suite.T(), suite.testState.client2, "default", "myhook")
	sub := CreateSubscription(suite.T(), suite.testState.client2, subJSON, 201)
	assert.NotNil(suite.T(), sub.ID)

	data := fftypes.DataRefOrValue{
		Value: fftypes.Byteable(`{}`),
	}

	var resp *resty.Response
	resp, err := PrivateMessage(suite.T(), suite.testState.client1, &data, []string{
		suite.testState.org1.Name,
		suite.testState.org2.Name,
	}, "myrequest", fftypes.TransactionTypeBatchPin)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 202, resp.StatusCode())

	<-received1 // request
	<-received2 // request

	<-received1 // reply
	val1 := validateReceivedMessages(suite.testState, suite.testState.client1, fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 2, 0)
	assert.Equal(suite.T(), float64(200), val1.JSONObject()["status"])
	decoded1, err := base64.StdEncoding.DecodeString(val1.JSONObject().GetString("body"))
	assert.NoError(suite.T(), err)
	assert.Regexp(suite.T(), "Example YAML", string(decoded1))

	<-received2 // reply
	val2 := validateReceivedMessages(suite.testState, suite.testState.client1, fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 2, 0)
	assert.Equal(suite.T(), float64(200), val2.JSONObject()["status"])
	decoded2, err := base64.StdEncoding.DecodeString(val2.JSONObject().GetString("body"))
	assert.NoError(suite.T(), err)
	assert.Regexp(suite.T(), "Example YAML", string(decoded2))
}

func (suite *OnChainOffChainTestSuite) TestE2EWebhookRequestReplyNoTx() {
	defer suite.testState.done()

	subJSON := `{
		"transport": "webhooks",
		"namespace": "default",
		"name": "myhook",
		"options": {
			"withData": true,
			"url": "https://github.com/hyperledger/firefly/raw/main/resources/ff-logo-32.png",
			"reply": true,
			"replytag": "myreply",
			"replytx": "none",
			"method": "GET"
		},
		"filter": {
			"tag": "myrequest"
		}
	}`
	CleanupExistingSubscription(suite.T(), suite.testState.client2, "default", "myhook")
	sub := CreateSubscription(suite.T(), suite.testState.client2, subJSON, 201)
	assert.NotNil(suite.T(), sub.ID)

	data := fftypes.DataRefOrValue{
		Value: fftypes.Byteable(`{}`),
	}

	reply := RequestReply(suite.T(), suite.testState.client1, &data, []string{
		suite.testState.org1.Name,
		suite.testState.org2.Name,
	}, "myrequest", fftypes.TransactionTypeNone)
	assert.NotNil(suite.T(), reply)

	bodyData := reply.InlineData[0].Value.JSONObject().GetString("body")
	b, err := base64.StdEncoding.DecodeString(bodyData)
	assert.NoError(suite.T(), err)
	ffImg, err := png.Decode(bytes.NewReader(b))
	assert.NoError(suite.T(), err)

	// Verify we got the right data back by parsing it
	convertOptions := image2ascii.DefaultOptions
	convertOptions.FixedWidth = 100
	convertOptions.FixedHeight = 60
	convertOptions.Colored = false
	converter := image2ascii.NewImageConverter()
	str := converter.Image2ASCIIString(ffImg, &convertOptions)
	for _, s := range strings.Split(str, "\n") {
		if len(strings.TrimSpace(s)) > 0 {
			fmt.Println(s)
		}
	}

}
