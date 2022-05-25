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
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testState struct {
	startTime            time.Time
	t                    *testing.T
	client1              *resty.Client
	client2              *resty.Client
	ws1                  *websocket.Conn
	ws2                  *websocket.Conn
	org1                 *core.Identity
	org1key              *core.Verifier
	org2                 *core.Identity
	org2key              *core.Verifier
	done                 func()
	stackState           *StackState
	unregisteredAccounts []interface{}
	namespace            string
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

func validateReceivedMessages(ts *testState, client *resty.Client, topic string, msgType core.MessageType, txtype core.TransactionType, count int) (data core.DataArray) {
	var group *fftypes.Bytes32
	var messages []*core.Message
	events := GetMessageEvents(ts.t, client, ts.startTime, topic, 200)
	for i, event := range events {
		if event.Message != nil {
			message := event.Message
			ts.t.Logf("Message %d: %+v", i, *message)
			if message.Header.Type != msgType {
				continue
			}
			if group != nil {
				assert.Equal(ts.t, group.String(), message.Header.Group.String(), "All messages must be same group")
			}
			group = message.Header.Group
			messages = append(messages, message)
		}
	}
	assert.Equal(ts.t, count, len(messages))

	var returnData []*core.Data
	for idx := 0; idx < len(messages); idx++ {
		assert.Equal(ts.t, txtype, (messages)[idx].Header.TxType)
		assert.Equal(ts.t, core.FFStringArray{topic}, (messages)[idx].Header.Topics)
		assert.Equal(ts.t, topic, (messages)[idx].Header.Topics[0])

		data := GetDataForMessage(ts.t, client, ts.startTime, (messages)[idx].Header.ID)
		var msgData *core.Data
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
		assert.Equal(ts.t, ts.namespace, msgData.Namespace)
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

func readStackState(t *testing.T) *StackState {
	stackFile := os.Getenv("STACK_STATE")
	if stackFile == "" {
		t.Fatal("STACK_STATE must be set")
	}
	stackState, err := ReadStackState(stackFile)
	assert.NoError(t, err)
	return stackState
}

func beforeE2ETest(t *testing.T) *testState {
	stack := readStackFile(t)
	stackState := readStackState(t)
	namespace := "default"

	var authHeader1 http.Header
	var authHeader2 http.Header

	ts := &testState{
		t:                    t,
		startTime:            time.Now(),
		client1:              NewResty(t),
		client2:              NewResty(t),
		stackState:           stackState,
		unregisteredAccounts: stackState.Accounts[2:],
		namespace:            namespace,
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

	t.Logf("Blockchain provider: %s", stack.BlockchainProvider)

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

	// If no namespace is set to run tests against, use the default namespace
	if os.Getenv("NAMESPACE") != "" {
		namespace := os.Getenv("NAMESPACE")
		ts.namespace = namespace
		CreateNamespaces(ts.client1, namespace)
		CreateNamespaces(ts.client2, namespace)
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
	t.Logf("Org1: ID=%s DID=%s Key=%s", ts.org1.DID, ts.org1.ID, ts.org1key.Value)
	t.Logf("Org2: ID=%s DID=%s Key=%s", ts.org2.DID, ts.org2.ID, ts.org2key.Value)

	eventNames := "message_confirmed|token_pool_confirmed|token_transfer_confirmed|blockchain_event_received|token_approval_confirmed|identity_confirmed"
	queryString := fmt.Sprintf("namespace=%s&ephemeral&autoack&filter.events=%s&changeevents=.*", ts.namespace, eventNames)

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

func wsReader(conn *websocket.Conn, dbChanges bool) chan *core.EventDelivery {
	events := make(chan *core.EventDelivery, 100)
	go func() {
		for {
			_, b, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("Websocket %s closing, error: %s\n", conn.RemoteAddr(), err)
				return
			}
			var wsa core.WSActionBase
			err = json.Unmarshal(b, &wsa)
			if err != nil {
				panic(fmt.Errorf("Invalid JSON received on WebSocket: %s", err))
			}
			switch wsa.Type {
			default:
				var ed core.EventDelivery
				err = json.Unmarshal(b, &ed)
				if err != nil {
					panic(fmt.Errorf("Invalid JSON received on WebSocket: %s", err))
				}
				if err == nil {
					fmt.Printf("Websocket %s event: %s/%s/%s -> %s (tx=%s)\n", conn.RemoteAddr(), ed.Namespace, ed.Type, ed.ID, ed.Reference, ed.Transaction)
					events <- &ed
				}
			}
		}
	}()
	return events
}

func waitForEvent(t *testing.T, c chan *core.EventDelivery, eventType core.EventType, ref *fftypes.UUID) {
	for {
		ed := <-c
		if ed.Type == eventType && (ref == nil || *ref == *ed.Reference) {
			t.Logf("Detected '%s' event for ref '%s'", ed.Type, ed.Reference)
			return
		}
		t.Logf("Ignored event '%s'", ed.ID)
	}
}

func waitForMessageConfirmed(t *testing.T, c chan *core.EventDelivery, msgType core.MessageType) *core.EventDelivery {
	for {
		ed := <-c
		if ed.Type == core.EventTypeMessageConfirmed && ed.Message != nil && ed.Message.Header.Type == msgType {
			t.Logf("Detected '%s' event for message '%s' of type '%s'", ed.Type, ed.Message.Header.ID, msgType)
			return ed
		}
		t.Logf("Ignored event '%s'", ed.ID)
	}
}

func waitForIdentityConfirmed(t *testing.T, c chan *core.EventDelivery) *core.EventDelivery {
	for {
		ed := <-c
		if ed.Type == core.EventTypeIdentityConfirmed {
			t.Logf("Detected '%s' event for identity '%s'", ed.Type, ed.Reference)
			return ed
		}
		t.Logf("Ignored event '%s'", ed.ID)
	}
}

func waitForContractEvent(t *testing.T, client *resty.Client, c chan *core.EventDelivery, match map[string]interface{}) map[string]interface{} {
	for {
		eventDelivery := <-c
		if eventDelivery.Type == core.EventTypeBlockchainEventReceived {
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
		actualString, actualIsString := actual.(string)
		if expectedIsString && actualIsString {
			return strings.ToLower(expectedString) == strings.ToLower(actualString)
		}
		return expected == actual
	}
	return match
}
