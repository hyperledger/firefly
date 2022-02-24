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

package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gorilla/websocket"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testState struct {
	startTime time.Time
	t         *testing.T
	client1   *resty.Client
	client2   *resty.Client
	ws1       *websocket.Conn
	ws2       *websocket.Conn
	org1      *fftypes.Identity
	org1key   *fftypes.Verifier
	org2      *fftypes.Identity
	org2key   *fftypes.Verifier
	done      func()
}

var widgetSchemaJSON = []byte(`{
	"$id": "https://example.com/widget.schema.json",
	"$schema": "https://json-schema.org/draft/2020-12/schema",
	"title": "Widget",
	"type": "object",
	"properties": {
		"id": {
			"type": "string",
			"description": "The unique identifier for the widget."
		},
		"name": {
			"type": "string",
			"description": "The person's last name."
		}
	},
	"additionalProperties": false
}`)

func pollForUp(t *testing.T, client *resty.Client) {
	var resp *resty.Response
	var err error
	for i := 0; i < 3; i++ {
		resp, err = GetNamespaces(client)
		if err == nil && resp.StatusCode() == 200 {
			break
		}
		time.Sleep(5 * time.Second)
	}
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode())
}

func validateReceivedMessages(ts *testState, client *resty.Client, topic string, msgType fftypes.MessageType, txtype fftypes.TransactionType, count int) (data []*fftypes.Data) {
	var group *fftypes.Bytes32
	messages := GetMessages(ts.t, client, ts.startTime, msgType, topic, 200)
	for i, message := range messages {
		ts.t.Logf("Message %d: %+v", i, *message)
		if group != nil {
			assert.Equal(ts.t, group.String(), message.Header.Group.String(), "All messages must be same group")
		}
		group = message.Header.Group
	}
	assert.Equal(ts.t, count, len(messages))

	var returnData []*fftypes.Data
	for idx := 0; idx < len(messages); idx++ {
		assert.Equal(ts.t, txtype, (messages)[idx].Header.TxType)
		assert.Equal(ts.t, fftypes.FFStringArray{topic}, (messages)[idx].Header.Topics)
		assert.Equal(ts.t, topic, (messages)[idx].Header.Topics[0])

		data := GetDataForMessage(ts.t, client, ts.startTime, (messages)[idx].Header.ID)
		var msgData *fftypes.Data
		for i, d := range data {
			ts.t.Logf("Data %d: %+v", i, *d)
			if *d.ID == *messages[idx].Data[0].ID {
				msgData = d
			}
		}
		assert.NotNil(ts.t, msgData, "Found data with ID '%s'", messages[idx].Data[0].ID)
		if group == nil {
			assert.Equal(ts.t, 1, len(data))
		}

		returnData = append(returnData, msgData)

		assert.Equal(ts.t, "default", msgData.Namespace)
		expectedHash, err := msgData.CalcHash(context.Background())
		assert.NoError(ts.t, err)
		assert.Equal(ts.t, *expectedHash, *msgData.Hash)

		if msgData.Blob != nil {
			blob := GetBlob(ts.t, client, msgData, 200)
			assert.NotNil(ts.t, blob)
			var hash fftypes.Bytes32 = sha256.Sum256(blob)
			assert.Equal(ts.t, *msgData.Blob.Hash, hash)
		}

	}

	// Flip data (returned in most recent order) into delivery order
	return returnData
}

func validateAccountBalances(t *testing.T, client *resty.Client, poolID *fftypes.UUID, tokenIndex string, balances map[string]int64) {
	for key, balance := range balances {
		account := GetTokenBalance(t, client, poolID, tokenIndex, key)
		assert.Equal(t, "erc1155", account.Connector)
		assert.Equal(t, balance, account.Balance.Int().Int64())
	}
}

func pickTopic(i int, options []string) string {
	return options[i%len(options)]
}

func readStackFile(t *testing.T) *Stack {
	stackFile := os.Getenv("STACK_FILE")
	if stackFile == "" {
		t.Fatal("STACK_FILE must be set")
	}
	stack, err := ReadStack(stackFile)
	assert.NoError(t, err)
	return stack
}

func beforeE2ETest(t *testing.T) *testState {
	stack := readStackFile(t)

	var authHeader1 http.Header
	var authHeader2 http.Header

	ts := &testState{
		t:         t,
		startTime: time.Now(),
		client1:   NewResty(t),
		client2:   NewResty(t),
	}

	httpProtocolClient1 := "http"
	websocketProtocolClient1 := "ws"
	httpProtocolClient2 := "http"
	websocketProtocolClient2 := "ws"
	if stack.Members[0].UseHTTPS {
		httpProtocolClient1 = "https"
		websocketProtocolClient1 = "wss"
	}
	if stack.Members[1].UseHTTPS {
		httpProtocolClient2 = "https"
		websocketProtocolClient2 = "wss"
	}

	member0WithPort := ""
	if stack.Members[0].ExposedFireflyPort != 0 {
		member0WithPort = fmt.Sprintf(":%d", stack.Members[0].ExposedFireflyPort)
	}
	member1WithPort := ""
	if stack.Members[1].ExposedFireflyPort != 0 {
		member1WithPort = fmt.Sprintf(":%d", stack.Members[1].ExposedFireflyPort)
	}

	ts.client1.SetBaseURL(fmt.Sprintf("%s://%s%s/api/v1", httpProtocolClient1, stack.Members[0].FireflyHostname, member0WithPort))
	ts.client2.SetBaseURL(fmt.Sprintf("%s://%s%s/api/v1", httpProtocolClient2, stack.Members[1].FireflyHostname, member1WithPort))

	if stack.Members[0].Username != "" && stack.Members[0].Password != "" {
		t.Log("Setting auth for user 1")
		ts.client1.SetBasicAuth(stack.Members[0].Username, stack.Members[0].Password)
		authHeader1 = http.Header{
			"Authorization": []string{fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", stack.Members[0].Username, stack.Members[0].Password))))},
		}
	}

	if stack.Members[1].Username != "" && stack.Members[1].Password != "" {
		t.Log("Setting auth for user 2")
		ts.client2.SetBasicAuth(stack.Members[1].Username, stack.Members[1].Password)
		authHeader2 = http.Header{
			"Authorization": []string{fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", stack.Members[1].Username, stack.Members[1].Password))))},
		}
	}

	t.Logf("Client 1: " + ts.client1.HostURL)
	t.Logf("Client 2: " + ts.client2.HostURL)
	pollForUp(t, ts.client1)
	pollForUp(t, ts.client2)

	for {
		orgsC1 := GetOrgs(t, ts.client1, 200)
		orgsC2 := GetOrgs(t, ts.client2, 200)
		if len(orgsC1) >= 2 && len(orgsC2) >= 2 {
			// in case there are more than two orgs in the network we need to ensure
			// we select the same two that were provided in the first two elements
			// of the stack file
			for _, org := range orgsC1 {
				if org.Name == stack.Members[0].OrgName {
					ts.org1 = org
				} else if org.Name == stack.Members[1].OrgName {
					ts.org2 = org
				}
			}
			if ts.org1 != nil && ts.org2 != nil {
				break
			}
		}
		t.Logf("Waiting for 2 orgs to appear. Currently have: node1=%d node2=%d", len(orgsC1), len(orgsC2))
		time.Sleep(3 * time.Second)
	}
	ts.org1key = GetIdentityBlockchainKeys(t, ts.client1, ts.org1.ID, 200)[0]
	ts.org2key = GetIdentityBlockchainKeys(t, ts.client2, ts.org2.ID, 200)[0]

	eventNames := "message_confirmed|token_pool_confirmed|token_transfer_confirmed|blockchain_event|token_approval_confirmed"
	queryString := fmt.Sprintf("namespace=default&ephemeral&autoack&filter.events=%s&changeevents=.*", eventNames)

	wsUrl1 := url.URL{
		Scheme:   websocketProtocolClient1,
		Host:     fmt.Sprintf("%s%s", stack.Members[0].FireflyHostname, member0WithPort),
		Path:     "/ws",
		RawQuery: queryString,
	}
	wsUrl2 := url.URL{
		Scheme:   websocketProtocolClient2,
		Host:     fmt.Sprintf("%s%s", stack.Members[1].FireflyHostname, member1WithPort),
		Path:     "/ws",
		RawQuery: queryString,
	}

	t.Logf("Websocket 1: " + wsUrl1.String())
	t.Logf("Websocket 2: " + wsUrl2.String())

	var err error
	ts.ws1, _, err = websocket.DefaultDialer.Dial(wsUrl1.String(), authHeader1)
	if err != nil {
		t.Logf(err.Error())
	}
	require.NoError(t, err)

	ts.ws2, _, err = websocket.DefaultDialer.Dial(wsUrl2.String(), authHeader2)
	require.NoError(t, err)

	ts.done = func() {
		ts.ws1.Close()
		ts.ws2.Close()
		t.Log("WebSockets closed")
	}
	return ts
}

func wsReader(conn *websocket.Conn) (chan *fftypes.EventDelivery, chan *fftypes.ChangeEvent) {
	events := make(chan *fftypes.EventDelivery, 100)
	changeEvents := make(chan *fftypes.ChangeEvent, 100)
	go func() {
		for {
			_, b, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("Websocket %s closing, error: %s", conn.RemoteAddr(), err)
				return
			}
			fmt.Printf("Websocket %s receive: %s", conn.RemoteAddr(), b)
			var wsa fftypes.WSClientActionBase
			err = json.Unmarshal(b, &wsa)
			if err != nil {
				panic(fmt.Errorf("Invalid JSON received on WebSocket: %s", err))
			}
			switch wsa.Type {
			case fftypes.WSClientActionChangeNotifcation:
				var wscn fftypes.WSChangeNotification
				err = json.Unmarshal(b, &wscn)
				if err != nil {
					panic(fmt.Errorf("Invalid JSON received on WebSocket: %s", err))
				}
				if err == nil {
					changeEvents <- wscn.ChangeEvent
				}
			default:
				var ed fftypes.EventDelivery
				err = json.Unmarshal(b, &ed)
				if err != nil {
					panic(fmt.Errorf("Invalid JSON received on WebSocket: %s", err))
				}
				if err == nil {
					events <- &ed
				}
			}
		}
	}()
	return events, changeEvents
}

func waitForEvent(t *testing.T, c chan *fftypes.EventDelivery, eventType fftypes.EventType, ref *fftypes.UUID) {
	for {
		eventDelivery := <-c
		if eventDelivery.Type == eventType && (ref == nil || *ref == *eventDelivery.Reference) {
			return
		}
	}
}

func waitForMessageConfirmed(t *testing.T, c chan *fftypes.EventDelivery, msgType fftypes.MessageType) *fftypes.EventDelivery {
	for {
		ed := <-c
		if ed.Type == fftypes.EventTypeMessageConfirmed && ed.Message != nil && ed.Message.Header.Type == msgType {
			t.Logf("Detected '%s' event for message '%s' of type '%s'", ed.Type, ed.Message.Header.ID, msgType)
			return ed
		}
		t.Logf("Ignored event '%s'", ed.ID)
	}
}

func waitForContractEvent(t *testing.T, client *resty.Client, c chan *fftypes.EventDelivery, match map[string]interface{}) map[string]interface{} {
	for {
		eventDelivery := <-c
		if eventDelivery.Type == fftypes.EventTypeBlockchainEvent {
			event, err := GetBlockchainEvent(t, client, eventDelivery.Event.Reference.String())
			if err != nil {
				t.Logf("WARN: unable to get event: %v", err.Error())
				continue
			}
			eventJSON, ok := event.(map[string]interface{})
			if !ok {
				t.Logf("WARN: unable to parse changeEvent: %v", event)
				continue
			}
			if checkObject(t, match, eventJSON) {
				return eventJSON
			}
		}
	}
}

func checkObject(t *testing.T, expected interface{}, actual interface{}) bool {
	match := true

	// check if this is a nested object
	expectedObject, expectedIsObject := expected.(map[string]interface{})
	actualObject, actualIsObject := actual.(map[string]interface{})

	t.Logf("Matching blockchain event: %s", fftypes.JSONObject(actualObject).String())

	// check if this is an array
	expectedArray, expectedIsArray := expected.([]interface{})
	actualArray, actualIsArray := actual.([]interface{})
	switch {
	case expectedIsObject && actualIsObject:
		for expectedKey, expectedValue := range expectedObject {
			if !checkObject(t, expectedValue, actualObject[expectedKey]) {
				return false
			}
		}
	case expectedIsArray && actualIsArray:
		for _, expectedItem := range expectedArray {
			for j, actualItem := range actualArray {
				if checkObject(t, expectedItem, actualItem) {
					break
				}
				if j == len(actualArray)-1 {
					return false
				}
			}
		}
	default:
		expectedString, expectedIsString := expected.(string)
		actualString, actualIsString := expected.(string)
		if expectedIsString && actualIsString {
			return strings.ToLower(expectedString) == strings.ToLower(actualString)
		}
		return expected == actual
	}
	return match
}
