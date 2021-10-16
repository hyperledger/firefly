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
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"math/big"
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

	image2ascii "github.com/qeesung/image2ascii/convert"
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
	for identity, balance := range balances {
		account := GetTokenAccount(t, client, poolName, tokenIndex, identity)
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

func TestE2EBroadcast(t *testing.T) {

	ts := beforeE2ETest(t)
	defer ts.done()

	received1, changes1 := wsReader(t, ts.ws1)
	received2, changes2 := wsReader(t, ts.ws2)

	var resp *resty.Response
	value := fftypes.Byteable(`"Hello"`)
	data := fftypes.DataRefOrValue{
		Value: value,
	}

	resp, err := BroadcastMessage(ts.client1, &data, false)
	require.NoError(t, err)
	assert.Equal(t, 202, resp.StatusCode())

	<-received1
	<-changes1 // also expect database change events
	val1 := validateReceivedMessages(ts, ts.client1, fftypes.MessageTypeBroadcast, fftypes.TransactionTypeBatchPin, 1, 0)
	assert.Equal(t, data.Value, val1)

	<-received2
	<-changes2 // also expect database change events
	val2 := validateReceivedMessages(ts, ts.client2, fftypes.MessageTypeBroadcast, fftypes.TransactionTypeBatchPin, 1, 0)
	assert.Equal(t, data.Value, val2)

}

func TestE2EPrivate(t *testing.T) {

	ts := beforeE2ETest(t)
	defer ts.done()

	received1, _ := wsReader(t, ts.ws1)
	received2, _ := wsReader(t, ts.ws2)

	var resp *resty.Response
	value := fftypes.Byteable(`"Hello"`)
	data := fftypes.DataRefOrValue{
		Value: value,
	}

	resp, err := PrivateMessage(t, ts.client1, &data, []string{
		ts.org1.Name,
		ts.org2.Name,
	}, "", fftypes.TransactionTypeBatchPin, false)
	require.NoError(t, err)
	assert.Equal(t, 202, resp.StatusCode())

	<-received1
	val1 := validateReceivedMessages(ts, ts.client1, fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 1, 0)
	assert.Equal(t, data.Value, val1)

	<-received2
	val2 := validateReceivedMessages(ts, ts.client2, fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 1, 0)
	assert.Equal(t, data.Value, val2)
}

func TestStrongDatatypesBroadcast(t *testing.T) {

	ts := beforeE2ETest(t)
	defer ts.done()

	var resp *resty.Response
	value := fftypes.Byteable(`"Hello"`)
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
	resp, err := BroadcastMessage(ts.client1, &data, true)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode())
	assert.Contains(t, resp.String(), "FF10195") // datatype not found

	dt := &fftypes.Datatype{
		Name:    "widget",
		Version: version,
		Value:   widgetSchemaJSON,
	}
	dt = CreateDatatype(t, ts.client1, dt, true)

	resp, err = BroadcastMessage(ts.client1, &data, true)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode())
	assert.Contains(t, resp.String(), "FF10198") // does not conform

	data.Value = fftypes.Byteable(`{
		"id": "widget12345",
		"name": "mywidget"
	}`)

	resp, err = BroadcastMessage(ts.client1, &data, true)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode())

}

func TestStrongDatatypesPrivate(t *testing.T) {

	ts := beforeE2ETest(t)
	defer ts.done()

	var resp *resty.Response
	value := fftypes.Byteable(`{"foo":"bar"}`)
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
	resp, err := PrivateMessage(t, ts.client1, &data, []string{
		ts.org1.Name,
		ts.org2.Name,
	}, "", fftypes.TransactionTypeBatchPin, true)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode())
	assert.Contains(t, resp.String(), "FF10195") // datatype not found

	dt := &fftypes.Datatype{
		Name:    "widget",
		Version: version,
		Value:   widgetSchemaJSON,
	}
	dt = CreateDatatype(t, ts.client1, dt, true)

	resp, err = PrivateMessage(t, ts.client1, &data, []string{
		ts.org1.Name,
		ts.org2.Name,
	}, "", fftypes.TransactionTypeBatchPin, false)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode())
	assert.Contains(t, resp.String(), "FF10198") // does not conform

	data.Value = fftypes.Byteable(`{
		"id": "widget12345",
		"name": "mywidget"
	}`)

	resp, err = PrivateMessage(t, ts.client1, &data, []string{
		ts.org1.Name,
		ts.org2.Name,
	}, "", fftypes.TransactionTypeBatchPin, true)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode())

}

func TestE2EBroadcastBlob(t *testing.T) {

	ts := beforeE2ETest(t)
	defer ts.done()

	received1, _ := wsReader(t, ts.ws1)
	received2, _ := wsReader(t, ts.ws2)

	var resp *resty.Response

	resp, err := BroadcastBlobMessage(t, ts.client1)
	require.NoError(t, err)
	assert.Equal(t, 202, resp.StatusCode())

	<-received1
	val1 := validateReceivedMessages(ts, ts.client1, fftypes.MessageTypeBroadcast, fftypes.TransactionTypeBatchPin, 1, 0)
	assert.Regexp(t, "myfile.txt", string(val1))

	<-received2
	val2 := validateReceivedMessages(ts, ts.client2, fftypes.MessageTypeBroadcast, fftypes.TransactionTypeBatchPin, 1, 0)
	assert.Regexp(t, "myfile.txt", string(val2))

}

func TestE2EPrivateBlobDatatypeTagged(t *testing.T) {

	ts := beforeE2ETest(t)
	defer ts.done()

	received1, _ := wsReader(t, ts.ws1)
	received2, _ := wsReader(t, ts.ws2)

	var resp *resty.Response

	resp, err := PrivateBlobMessageDatatypeTagged(t, ts.client1, []string{
		ts.org1.Name,
		ts.org2.Name,
	})
	require.NoError(t, err)
	assert.Equal(t, 202, resp.StatusCode())

	<-received1
	_ = validateReceivedMessages(ts, ts.client1, fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 1, 0)

	<-received2
	_ = validateReceivedMessages(ts, ts.client2, fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 1, 0)
}

func TestE2ETokenPool(t *testing.T) {

	ts := beforeE2ETest(t)
	defer ts.done()

	received1, _ := wsReader(t, ts.ws1)
	received2, _ := wsReader(t, ts.ws2)

	pools := GetTokenPools(t, ts.client1, time.Unix(0, 0))
	poolName := fmt.Sprintf("pool%d", len(pools))
	t.Logf("Pool name: %s", poolName)

	pool := &fftypes.TokenPool{
		Name: poolName,
		Type: fftypes.TokenTypeFungible,
	}
	CreateTokenPool(t, ts.client1, pool)

	<-received1
	pools = GetTokenPools(t, ts.client1, ts.startTime)
	assert.Equal(t, 1, len(pools))
	assert.Equal(t, "default", pools[0].Namespace)
	assert.Equal(t, poolName, pools[0].Name)
	assert.Equal(t, fftypes.TokenTypeFungible, pools[0].Type)

	<-received2
	pools = GetTokenPools(t, ts.client1, ts.startTime)
	assert.Equal(t, 1, len(pools))
	assert.Equal(t, "default", pools[0].Namespace)
	assert.Equal(t, poolName, pools[0].Name)
	assert.Equal(t, fftypes.TokenTypeFungible, pools[0].Type)

	transfer := &fftypes.TokenTransfer{}
	transfer.Amount.Int().SetInt64(1)
	MintTokens(t, ts.client1, poolName, transfer)

	<-received1
	transfers := GetTokenTransfers(t, ts.client1, poolName)
	assert.Equal(t, 1, len(transfers))
	assert.Equal(t, fftypes.TokenTransferTypeMint, transfers[0].Type)
	assert.Equal(t, "0", transfers[0].TokenIndex)
	assert.Equal(t, int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(t, ts.client1, poolName, "0", map[string]int64{
		ts.org1.Identity: 1,
	})

	<-received2
	transfers = GetTokenTransfers(t, ts.client2, poolName)
	assert.Equal(t, 1, len(transfers))
	assert.Equal(t, fftypes.TokenTransferTypeMint, transfers[0].Type)
	assert.Equal(t, int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(t, ts.client2, poolName, "0", map[string]int64{
		ts.org1.Identity: 1,
	})

	transfer = &fftypes.TokenTransfer{
		TokenIndex: "0",
		To:         ts.org2.Identity,
	}
	transfer.Amount.Int().SetInt64(1)
	TransferTokens(t, ts.client1, poolName, transfer)

	<-received1
	transfers = GetTokenTransfers(t, ts.client1, poolName)
	assert.Equal(t, 2, len(transfers))
	assert.Equal(t, fftypes.TokenTransferTypeTransfer, transfers[0].Type)
	assert.Equal(t, "0", transfers[0].TokenIndex)
	assert.Equal(t, int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(t, ts.client1, poolName, "0", map[string]int64{
		ts.org1.Identity: 0,
		ts.org2.Identity: 1,
	})

	<-received2
	transfers = GetTokenTransfers(t, ts.client2, poolName)
	assert.Equal(t, 2, len(transfers))
	assert.Equal(t, fftypes.TokenTransferTypeTransfer, transfers[0].Type)
	assert.Equal(t, "0", transfers[0].TokenIndex)
	assert.Equal(t, int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(t, ts.client2, poolName, "0", map[string]int64{
		ts.org1.Identity: 0,
		ts.org2.Identity: 1,
	})

	transfer = &fftypes.TokenTransfer{
		TokenIndex: "0",
	}
	transfer.Amount.Int().SetInt64(1)
	BurnTokens(t, ts.client2, poolName, transfer)

	<-received2
	transfers = GetTokenTransfers(t, ts.client2, poolName)
	assert.Equal(t, 3, len(transfers))
	assert.Equal(t, fftypes.TokenTransferTypeBurn, transfers[0].Type)
	assert.Equal(t, "0", transfers[0].TokenIndex)
	assert.Equal(t, int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(t, ts.client2, poolName, "0", map[string]int64{
		ts.org1.Identity: 0,
		ts.org2.Identity: 0,
	})

	<-received1
	transfers = GetTokenTransfers(t, ts.client1, poolName)
	assert.Equal(t, 3, len(transfers))
	assert.Equal(t, fftypes.TokenTransferTypeBurn, transfers[0].Type)
	assert.Equal(t, "0", transfers[0].TokenIndex)
	assert.Equal(t, int64(1), transfers[0].Amount.Int().Int64())
	validateAccountBalances(t, ts.client1, poolName, "0", map[string]int64{
		ts.org1.Identity: 0,
		ts.org2.Identity: 0,
	})
}

func TestE2EWebhookExchange(t *testing.T) {

	ts := beforeE2ETest(t)
	defer ts.done()

	received1, _ := wsReader(t, ts.ws1)
	received2, _ := wsReader(t, ts.ws2)

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
	CleanupExistingSubscription(t, ts.client2, "default", "myhook")
	sub := CreateSubscription(t, ts.client2, subJSON, 201)
	assert.NotNil(t, sub.ID)

	data := fftypes.DataRefOrValue{
		Value: fftypes.Byteable(`{}`),
	}

	var resp *resty.Response
	resp, err := PrivateMessage(t, ts.client1, &data, []string{
		ts.org1.Name,
		ts.org2.Name,
	}, "myrequest", fftypes.TransactionTypeBatchPin, false)
	require.NoError(t, err)
	assert.Equal(t, 202, resp.StatusCode())

	<-received1 // request
	<-received2 // request

	<-received1 // reply
	val1 := validateReceivedMessages(ts, ts.client1, fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 2, 0)
	assert.Equal(t, float64(200), val1.JSONObject()["status"])
	decoded1, err := base64.StdEncoding.DecodeString(val1.JSONObject().GetString("body"))
	assert.NoError(t, err)
	assert.Regexp(t, "Example YAML", string(decoded1))

	<-received2 // reply
	val2 := validateReceivedMessages(ts, ts.client1, fftypes.MessageTypePrivate, fftypes.TransactionTypeBatchPin, 2, 0)
	assert.Equal(t, float64(200), val2.JSONObject()["status"])
	decoded2, err := base64.StdEncoding.DecodeString(val2.JSONObject().GetString("body"))
	assert.NoError(t, err)
	assert.Regexp(t, "Example YAML", string(decoded2))
}

func TestE2EWebhookRequestReplyNoTx(t *testing.T) {

	ts := beforeE2ETest(t)
	defer ts.done()

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
	CleanupExistingSubscription(t, ts.client2, "default", "myhook")
	sub := CreateSubscription(t, ts.client2, subJSON, 201)
	assert.NotNil(t, sub.ID)

	data := fftypes.DataRefOrValue{
		Value: fftypes.Byteable(`{}`),
	}

	reply := RequestReply(t, ts.client1, &data, []string{
		ts.org1.Name,
		ts.org2.Name,
	}, "myrequest", fftypes.TransactionTypeNone)
	assert.NotNil(t, reply)

	bodyData := reply.InlineData[0].Value.JSONObject().GetString("body")
	b, err := base64.StdEncoding.DecodeString(bodyData)
	assert.NoError(t, err)
	ffImg, err := png.Decode(bytes.NewReader(b))
	assert.NoError(t, err)

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
