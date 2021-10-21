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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
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
	org1      *fftypes.Organization
	org2      *fftypes.Organization
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

func validateReceivedMessages(ts *testState, client *resty.Client, msgType fftypes.MessageType, txtype fftypes.TransactionType, count, idx int) fftypes.Byteable {
	var group *fftypes.Bytes32
	messages := GetMessages(ts.t, client, ts.startTime, msgType, 200)
	for i, message := range messages {
		ts.t.Logf("Message %d: %+v", i, *message)
		if group != nil {
			assert.Equal(ts.t, group.String(), message.Header.Group.String(), "All messages must be same group")
		}
		group = message.Header.Group
	}
	assert.Equal(ts.t, count, len(messages))
	assert.Equal(ts.t, txtype, (messages)[idx].Header.TxType)
	assert.Equal(ts.t, "default", (messages)[idx].Header.Namespace)
	assert.Equal(ts.t, fftypes.FFNameArray{"default"}, (messages)[idx].Header.Topics)

	data := GetData(ts.t, client, ts.startTime, 200)
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

	return msgData.Value
}

func validateAccountBalances(t *testing.T, client *resty.Client, poolName, tokenIndex string, balances map[string]int64) {
	for key, balance := range balances {
		account := GetTokenAccount(t, client, poolName, tokenIndex, key)
		assert.Equal(t, "erc1155", account.Connector)
		assert.Equal(t, balance, account.Balance.Int().Int64())
	}
}

func beforeE2ETest(t *testing.T) *testState {
	stackFile := os.Getenv("STACK_FILE")
	if stackFile == "" {
		t.Fatal("STACK_FILE must be set")
	}

	stack, err := ReadStack(stackFile)
	assert.NoError(t, err)

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
	ts.client1.SetHostURL(fmt.Sprintf("%s://%s:%d/api/v1", httpProtocolClient1, stack.Members[0].FireflyHostname, stack.Members[0].ExposedFireflyPort))
	ts.client2.SetHostURL(fmt.Sprintf("%s://%s:%d/api/v1", httpProtocolClient2, stack.Members[1].FireflyHostname, stack.Members[1].ExposedFireflyPort))

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
			ts.org1 = orgsC1[0]
			ts.org2 = orgsC1[1]
			break
		}
		t.Logf("Waiting for 2 orgs to appear. Currently have: node1=%d node2=%d", len(orgsC1), len(orgsC2))
		time.Sleep(3 * time.Second)
	}

	eventNames := "message_confirmed|token_pool_confirmed|token_transfer_confirmed"
	queryString := fmt.Sprintf("namespace=default&ephemeral&autoack&filter.events=%s&changeevents=.*", eventNames)

	wsUrl1 := url.URL{
		Scheme:   websocketProtocolClient1,
		Host:     fmt.Sprintf("%s:%d", stack.Members[0].FireflyHostname, stack.Members[0].ExposedFireflyPort),
		Path:     "/ws",
		RawQuery: queryString,
	}
	wsUrl2 := url.URL{
		Scheme:   websocketProtocolClient2,
		Host:     fmt.Sprintf("%s:%d", stack.Members[1].FireflyHostname, stack.Members[1].ExposedFireflyPort),
		Path:     "/ws",
		RawQuery: queryString,
	}

	t.Logf("Websocket 1: " + wsUrl1.String())
	t.Logf("Websocket 2: " + wsUrl2.String())

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

func wsReader(t *testing.T, conn *websocket.Conn) (chan *fftypes.EventDelivery, chan *fftypes.ChangeEvent) {
	events := make(chan *fftypes.EventDelivery, 100)
	changeEvents := make(chan *fftypes.ChangeEvent, 100)
	go func() {
		for {
			_, b, err := conn.ReadMessage()
			if err != nil {
				t.Logf("Websocket %s closing, error: %s", conn.RemoteAddr(), err)
				return
			}
			t.Logf("Websocket %s receive: %s", conn.RemoteAddr(), b)
			var wsa fftypes.WSClientActionBase
			err = json.Unmarshal(b, &wsa)
			assert.NoError(t, err)
			switch wsa.Type {
			case fftypes.WSClientActionChangeNotifcation:
				var wscn fftypes.WSChangeNotification
				err = json.Unmarshal(b, &wscn)
				assert.NoError(t, err)
				if err == nil {
					changeEvents <- wscn.ChangeEvent
				}
			default:
				var ed fftypes.EventDelivery
				err = json.Unmarshal(b, &ed)
				assert.NoError(t, err)
				if err == nil {
					events <- &ed
				}
			}
		}
	}()
	return events, changeEvents
}
