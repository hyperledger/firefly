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
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"image/png"
	"math/big"
	"strings"

	image2ascii "github.com/qeesung/image2ascii/convert"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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

	received1, changes1 := wsReader(suite.testState.ws1)
	received2, changes2 := wsReader(suite.testState.ws2)

	// Broadcast some messages, that should get batched, across two topics
	totalMessages := 10
	topics := []string{"topicA", "topicB"}
	expectedData := make(map[string][]*fftypes.DataRefOrValue)
	for i := 0; i < 10; i++ {
		value := fftypes.JSONAnyPtr(fmt.Sprintf(`"Hello number %d"`, i))
		data := &fftypes.DataRefOrValue{
			Value: value,
		}
		topic := pickTopic(i, topics)

		expectedData[topic] = append(expectedData[topic], data)

		resp, err := BroadcastMessage(suite.testState.client1, topic, data, false)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), 202, resp.StatusCode())
	}

	for i := 0; i < totalMessages; i++ {
		// Wait for all thel message-confirmed events, from both participants
		waitForMessageConfirmed(suite.T(), received1, fftypes.MessageTypeBroadcast)
		waitForMessageConfirmed(suite.T(), received2, fftypes.MessageTypeBroadcast)
		<-changes1 // also expect database change events
		<-changes2 // also expect database change events
	}

	for topic, dataArray := range expectedData {
		receiver1data := validateReceivedMessages(suite.testState, suite.testState.client1, topic, fftypes.MessageTypeBroadcast, fftypes.TransactionTypeBatchPin, len(dataArray))
		receiver2data := validateReceivedMessages(suite.testState, suite.testState.client2, topic, fftypes.MessageTypeBroadcast, fftypes.TransactionTypeBatchPin, len(dataArray))
		// Messages should be returned in exactly reverse send order (newest first)
		for i := (len(dataArray) - 1); i >= 0; i-- {
			assert.Equal(suite.T(), dataArray[i].Value, receiver1data[i].Value)
			assert.Equal(suite.T(), dataArray[i].Value, receiver2data[i].Value)
		}
	}

}

func (suite *OnChainOffChainTestSuite) TestStrongDatatypesBroadcast() {
	defer suite.testState.done()

	received1, changes1 := wsReader(suite.testState.ws1)
	received2, changes2 := wsReader(suite.testState.ws2)

	var resp *resty.Response
	value := fftypes.JSONAnyPtr(`"Hello"`)
	randVer, _ := rand.Int(rand.Reader, big.NewInt(100000000))
	version := fmt.Sprintf("0.0.%d", randVer.Int64())
	data := fftypes.DataRefOrValue{
		Value: value,
		Datatype: &fftypes.DatatypeRef{
			Name:    "widget",
			Version: version,
		},
	}

	// Should be rejected as datatype not known
	resp, err := BroadcastMessage(suite.testState.client1, "topic1", &data, true)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 400, resp.StatusCode())
	assert.Contains(suite.T(), resp.String(), "FF10195") // datatype not found

	dt := &fftypes.Datatype{
		Name:    "widget",
		Version: version,
		Value:   fftypes.JSONAnyPtrBytes(widgetSchemaJSON),
	}
	dt = CreateDatatype(suite.T(), suite.testState.client1, dt, true)

	resp, err = BroadcastMessage(suite.testState.client1, "topic1", &data, true)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 400, resp.StatusCode())
	assert.Contains(suite.T(), resp.String(), "FF10198") // does not conform

	data.Value = fftypes.JSONAnyPtr(`{
		"id": "widget12345",
		"name": "mywidget"
	}`)

	resp, err = BroadcastMessage(suite.testState.client1, "topic1", &data, true)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 200, resp.StatusCode())

	waitForMessageConfirmed(suite.T(), received1, fftypes.MessageTypeBroadcast)
	<-changes1 // also expect database change events
	waitForMessageConfirmed(suite.T(), received2, fftypes.MessageTypeBroadcast)
	<-changes2 // also expect database change events
}

func (suite *OnChainOffChainTestSuite) TestStrongDatatypesPrivate() {
	defer suite.testState.done()

	received1, changes1 := wsReader(suite.testState.ws1)
	received2, changes2 := wsReader(suite.testState.ws2)

	var resp *resty.Response
	value := fftypes.JSONAnyPtr(`{"foo":"bar"}`)
	randVer, _ := rand.Int(rand.Reader, big.NewInt(100000000))
	version := fmt.Sprintf("0.0.%d", randVer.Int64())
	data := fftypes.DataRefOrValue{
		Value: value,
		Datatype: &fftypes.DatatypeRef{
			Name:    "widget",
			Version: version,
		},
	}

	// Should be rejected as datatype not known
	resp, err := PrivateMessage(suite.testState, suite.testState.client1, "topic1", &data, []string{
		suite.testState.org1.Name,
		suite.testState.org2.Name,
	}, "", fftypes.TransactionTypeBatchPin, true)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 400, resp.StatusCode())
	assert.Contains(suite.T(), resp.String(), "FF10195") // datatype not found

	dt := &fftypes.Datatype{
		Name:    "widget",
		Version: version,
		Value:   fftypes.JSONAnyPtrBytes(widgetSchemaJSON),
	}
	dt = CreateDatatype(suite.T(), suite.testState.client1, dt, true)

	resp, err = PrivateMessage(suite.testState, suite.testState.client1, "topic1", &data, []string{
		suite.testState.org1.Name,
		suite.testState.org2.Name,
	}, "", fftypes.TransactionTypeBatchPin, false)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 400, resp.StatusCode())
	assert.Contains(suite.T(), resp.String(), "FF10198") // does not conform

	data.Value = fftypes.JSONAnyPtr(`{
		"id": "widget12345",
		"name": "mywidget"
	}`)

	resp, err = PrivateMessage(suite.testState, suite.testState.client1, "topic1", &data, []string{
		suite.testState.org1.Name,
		suite.testState.org2.Name,
	}, "", fftypes.TransactionTypeBatchPin, true)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 200, resp.StatusCode())

	waitForMessageConfirmed(suite.T(), received1, fftypes.MessageTypePrivate)
	<-changes1 // also expect database change events
	waitForMessageConfirmed(suite.T(), received2, fftypes.MessageTypePrivate)
	<-changes2 // also expect database change events
}

func (suite *OnChainOffChainTestSuite) TestE2EPrivate() {

	defer suite.testState.done()

	received1, _ := wsReader(suite.testState.ws1)
	received2, _ := wsReader(suite.testState.ws2)

	// Send 10 messages, that should get batched, across two topics
	totalMessages := 10
	topics := []string{"topicA", "topicB"}
	expectedData := make(map[string][]*fftypes.DataRefOrValue)
	for i := 0; i < 10; i++ {
		value := fftypes.JSONAnyPtr(fmt.Sprintf(`"Hello number %d"`, i))
		data := &fftypes.DataRefOrValue{
			Value: value,
		}
		topic := pickTopic(i, topics)

		expectedData[topic] = append(expectedData[topic], data)

		resp, err := PrivateMessage(suite.testState, suite.testState.client1, topic, data, []string{
			suite.testState.org1.Name,
			suite.testState.org2.Name,
		}, "", fftypes.TransactionTypeBatchPin, false)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), 202, resp.StatusCode())
	}

	for i := 0; i < totalMessages; i++ {
		// Wait for all thel message-confirmed events, from both participants
		waitForMessageConfirmed(suite.T(), received1, fftypes.MessageTypePrivate)
		waitForMessageConfirmed(suite.T(), received2, fftypes.MessageTypePrivate)
	}

	for topic, dataArray := range expectedData {
		receiver1data := validateReceivedMessages(suite.testState, suite.testState.client1, topic, fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, len(dataArray))
		receiver2data := validateReceivedMessages(suite.testState, suite.testState.client2, topic, fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, len(dataArray))
		// Messages should be returned in exactly reverse send order (newest first)
		for i := (len(dataArray) - 1); i >= 0; i-- {
			assert.Equal(suite.T(), dataArray[i].Value, receiver1data[i].Value)
			assert.Equal(suite.T(), dataArray[i].Value, receiver2data[i].Value)
		}
	}

}

func (suite *OnChainOffChainTestSuite) TestE2EBroadcastBlob() {
	defer suite.testState.done()

	received1, _ := wsReader(suite.testState.ws1)
	received2, _ := wsReader(suite.testState.ws2)

	var resp *resty.Response

	data, resp, err := BroadcastBlobMessage(suite.T(), suite.testState.client1, "topic1")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 202, resp.StatusCode())

	waitForMessageConfirmed(suite.T(), received1, fftypes.MessageTypeBroadcast)
	val1 := validateReceivedMessages(suite.testState, suite.testState.client1, "topic1", fftypes.MessageTypeBroadcast, fftypes.TransactionTypeBatchPin, 1)
	assert.Regexp(suite.T(), "myfile.txt", val1[0].Value.String())
	assert.Equal(suite.T(), "myfile.txt", val1[0].Blob.Name)
	assert.Equal(suite.T(), data.Blob.Size, val1[0].Blob.Size)

	waitForMessageConfirmed(suite.T(), received2, fftypes.MessageTypeBroadcast)
	val2 := validateReceivedMessages(suite.testState, suite.testState.client2, "topic1", fftypes.MessageTypeBroadcast, fftypes.TransactionTypeBatchPin, 1)
	assert.Regexp(suite.T(), "myfile.txt", val2[0].Value.String())
	assert.Equal(suite.T(), "myfile.txt", val2[0].Blob.Name)
	assert.Equal(suite.T(), data.Blob.Size, val2[0].Blob.Size)

}

func (suite *OnChainOffChainTestSuite) TestE2EPrivateBlobDatatypeTagged() {
	defer suite.testState.done()

	received1, _ := wsReader(suite.testState.ws1)
	received2, _ := wsReader(suite.testState.ws2)

	var resp *resty.Response

	data, resp, err := PrivateBlobMessageDatatypeTagged(suite.testState, suite.testState.client1, "topic1", []string{
		suite.testState.org1.Name,
		suite.testState.org2.Name,
	})
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 202, resp.StatusCode())
	assert.Empty(suite.T(), data.Blob.Name)
	assert.NotNil(suite.T(), data.Blob.Hash)

	waitForMessageConfirmed(suite.T(), received1, fftypes.MessageTypePrivate)
	res1 := validateReceivedMessages(suite.testState, suite.testState.client1, "topic1", fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 1)
	assert.Equal(suite.T(), data.Blob.Hash.String(), res1[0].Blob.Hash.String())
	assert.Empty(suite.T(), res1[0].Blob.Name)
	assert.Equal(suite.T(), data.Blob.Size, res1[0].Blob.Size)

	waitForMessageConfirmed(suite.T(), received2, fftypes.MessageTypePrivate)
	res2 := validateReceivedMessages(suite.testState, suite.testState.client2, "topic1", fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 1)
	assert.Equal(suite.T(), data.Blob.Hash.String(), res2[0].Blob.Hash.String())
	assert.Empty(suite.T(), res2[0].Blob.Name)
	assert.Equal(suite.T(), data.Blob.Size, res2[0].Blob.Size)
}

func (suite *OnChainOffChainTestSuite) TestE2EWebhookExchange() {
	defer suite.testState.done()

	received1, _ := wsReader(suite.testState.ws1)
	received2, _ := wsReader(suite.testState.ws2)

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
		Value: fftypes.JSONAnyPtr(`{}`),
	}

	var resp *resty.Response
	resp, err := PrivateMessage(suite.testState, suite.testState.client1, "topic1", &data, []string{
		suite.testState.org1.Name,
		suite.testState.org2.Name,
	}, "myrequest", fftypes.TransactionTypeBatchPin, false)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 202, resp.StatusCode())

	waitForMessageConfirmed(suite.T(), received1, fftypes.MessageTypePrivate) // request 1
	waitForMessageConfirmed(suite.T(), received2, fftypes.MessageTypePrivate) // request 2
	waitForMessageConfirmed(suite.T(), received1, fftypes.MessageTypePrivate) // response 1
	waitForMessageConfirmed(suite.T(), received2, fftypes.MessageTypePrivate) // response 2

	// When we query the confirmed messages for each receiver, we will see the requests and responses.
	// We just check the reponses (index 1)

	receiver1vals := validateReceivedMessages(suite.testState, suite.testState.client1, "topic1", fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 2)
	assert.Equal(suite.T(), float64(200), receiver1vals[1].Value.JSONObject()["status"])
	decoded1, err := base64.StdEncoding.DecodeString(receiver1vals[1].Value.JSONObject().GetString("body"))
	assert.NoError(suite.T(), err)
	assert.Regexp(suite.T(), "Example YAML", string(decoded1))

	receiver2vals := validateReceivedMessages(suite.testState, suite.testState.client2, "topic1", fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 2)
	assert.Equal(suite.T(), float64(200), receiver2vals[1].Value.JSONObject()["status"])
	decoded2, err := base64.StdEncoding.DecodeString(receiver2vals[1].Value.JSONObject().GetString("body"))
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
		Value: fftypes.JSONAnyPtr(`{}`),
	}

	reply := RequestReply(suite.testState, suite.testState.client1, &data, []string{
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
