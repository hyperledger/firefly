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
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gorilla/websocket"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
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

func validateReceivedMessages(ts *testState, client *resty.Client, value fftypes.Byteable, msgType fftypes.MessageType) {
	var group *fftypes.Bytes32
	messages := GetMessages(ts.t, client, ts.startTime, msgType, 200)
	for i, message := range messages {
		ts.t.Logf("Message %d: %+v", i, *message)
		if group != nil {
			assert.Equal(ts.t, group.String(), message.Header.Group.String(), "All messages must be same group")
		}
		group = message.Header.Group
	}
	assert.Equal(ts.t, 1, len(messages))
	assert.Equal(ts.t, fftypes.LowerCasedType("batch_pin"), (messages)[0].Header.TxType)
	assert.Equal(ts.t, "default", (messages)[0].Header.Namespace)
	assert.Equal(ts.t, fftypes.FFNameArray{"default"}, (messages)[0].Header.Topics)

	data := GetData(ts.t, client, ts.startTime, 200)
	var msgData *fftypes.Data
	for i, d := range data {
		ts.t.Logf("Data %d: %+v", i, *d)
		if *d.ID == *messages[0].Data[0].ID {
			msgData = d
		}
	}
	assert.NotNil(ts.t, msgData, "Found data with ID '%s'", messages[0].Data[0].ID)
	if group == nil {
		assert.Equal(ts.t, 1, len(data))
	}

	assert.Equal(ts.t, "default", msgData.Namespace)
	expectedHash, err := msgData.CalcHash(context.Background())
	assert.NoError(ts.t, err)
	assert.Equal(ts.t, *expectedHash, *msgData.Hash)

	if value != nil {
		assert.Equal(ts.t, value, (data)[0].Value)
	} else {
		blob := GetBlob(ts.t, client, msgData, 200)
		assert.NotNil(ts.t, blob)
		var hash fftypes.Bytes32 = sha256.Sum256(blob)
		assert.Equal(ts.t, *msgData.Blob.Hash, hash)
	}
}

func beforeE2ETest(t *testing.T) *testState {
	stackFile := os.Getenv("STACK_FILE")
	if stackFile == "" {
		t.Fatal("STACK_FILE must be set")
	}

	port1, err := GetMemberPort(stackFile, 0)
	require.NoError(t, err)
	port2, err := GetMemberPort(stackFile, 1)
	require.NoError(t, err)

	ts := &testState{
		t:         t,
		startTime: time.Now(),
		client1:   NewResty(t),
		client2:   NewResty(t),
	}

	ts.client1.SetHostURL(fmt.Sprintf("http://localhost:%d/api/v1", port1))
	ts.client2.SetHostURL(fmt.Sprintf("http://localhost:%d/api/v1", port2))

	t.Logf("Client 1: " + ts.client1.HostURL)
	t.Logf("Client 2: " + ts.client2.HostURL)
	pollForUp(t, ts.client1)
	pollForUp(t, ts.client2)

	for {
		orgs := GetOrgs(t, ts.client1, 200)
		if len(orgs) >= 2 {
			ts.org1 = orgs[0]
			ts.org2 = orgs[1]
			break
		}
		t.Logf("Waiting for 2 orgs to appear. Currently have: %d", len(orgs))
		time.Sleep(3 * time.Second)
	}

	wsUrl1 := url.URL{
		Scheme:   "ws",
		Host:     fmt.Sprintf("localhost:%d", port1),
		Path:     "/ws",
		RawQuery: "namespace=default&ephemeral&autoack",
	}
	wsUrl2 := url.URL{
		Scheme:   "ws",
		Host:     fmt.Sprintf("localhost:%d", port2),
		Path:     "/ws",
		RawQuery: "namespace=default&ephemeral&autoack",
	}

	t.Logf("Websocket 1: " + wsUrl1.String())
	t.Logf("Websocket 2: " + wsUrl2.String())

	ts.ws1, _, err = websocket.DefaultDialer.Dial(wsUrl1.String(), nil)
	require.NoError(t, err)
	ts.ws2, _, err = websocket.DefaultDialer.Dial(wsUrl2.String(), nil)
	require.NoError(t, err)

	ts.done = func() {
		ts.ws1.Close()
		ts.ws2.Close()
	}
	return ts
}

func wsReader(t *testing.T, conn *websocket.Conn) chan []byte {
	receiver := make(chan []byte)
	go func() {
		for {
			_, b, err := conn.ReadMessage()
			if err != nil {
				t.Logf("Websocket closing (%s)", err)
				return
			}
			t.Logf("WS Recevied: %s", b)
			receiver <- b
		}
	}()
	return receiver
}

// func TestE2EBroadcast(t *testing.T) {

// 	ts := beforeE2ETest(t)
// 	defer ts.done()

// 	received1 := wsReader(t, ts.ws1)
// 	received2 := wsReader(t, ts.ws2)

// 	var resp *resty.Response
// 	value := fftypes.Byteable(`"Hello"`)
// 	data := fftypes.DataRefOrValue{
// 		Value: value,
// 	}

// 	resp, err := BroadcastMessage(ts.client1, &data)
// 	require.NoError(t, err)
// 	assert.Equal(t, 202, resp.StatusCode())

// 	<-received1
// 	validateReceivedMessages(ts, ts.client1, value, fftypes.MessageTypeBroadcast)

// 	<-received2
// 	validateReceivedMessages(ts, ts.client2, value, fftypes.MessageTypeBroadcast)
// }

// func TestE2EPrivate(t *testing.T) {

// 	ts := beforeE2ETest(t)
// 	defer ts.done()

// 	received1 := wsReader(t, ts.ws1)
// 	received2 := wsReader(t, ts.ws2)

// 	var resp *resty.Response
// 	value := fftypes.Byteable(`"Hello"`)
// 	data := fftypes.DataRefOrValue{
// 		Value: value,
// 	}

// 	resp, err := PrivateMessage(t, ts.client1, &data, []string{
// 		ts.org1.Name,
// 		ts.org2.Name,
// 	})
// 	require.NoError(t, err)
// 	assert.Equal(t, 202, resp.StatusCode())

// 	<-received1
// 	validateReceivedMessages(ts, ts.client1, value, fftypes.MessageTypePrivate)

// 	<-received2
// 	validateReceivedMessages(ts, ts.client2, value, fftypes.MessageTypePrivate)
// }

// func TestE2EBroadcastBlob(t *testing.T) {

// 	ts := beforeE2ETest(t)
// 	defer ts.done()

// 	received1 := wsReader(t, ts.ws1)
// 	received2 := wsReader(t, ts.ws2)

// 	var resp *resty.Response

// 	resp, err := BroadcastBlobMessage(t, ts.client1)
// 	require.NoError(t, err)
// 	assert.Equal(t, 202, resp.StatusCode())

// 	<-received1
// 	validateReceivedMessages(ts, ts.client1, nil, fftypes.MessageTypeBroadcast)

// 	<-received2
// 	validateReceivedMessages(ts, ts.client2, nil, fftypes.MessageTypeBroadcast)
// }

// func TestE2EPrivateBlob(t *testing.T) {

// 	ts := beforeE2ETest(t)
// 	defer ts.done()

// 	received1 := wsReader(t, ts.ws1)
// 	received2 := wsReader(t, ts.ws2)

// 	var resp *resty.Response

// 	resp, err := PrivateBlobMessage(t, ts.client1, []string{
// 		ts.org1.Name,
// 		ts.org2.Name,
// 	})
// 	require.NoError(t, err)
// 	assert.Equal(t, 202, resp.StatusCode())

// 	<-received1
// 	validateReceivedMessages(ts, ts.client1, nil, fftypes.MessageTypePrivate)

// 	<-received2
// 	validateReceivedMessages(ts, ts.client2, nil, fftypes.MessageTypePrivate)
// }

func TestE2EWebhookExchange(t *testing.T) {

	ts := beforeE2ETest(t)
	defer ts.done()

	received1 := wsReader(t, ts.ws1)
	received2 := wsReader(t, ts.ws2)

	subJSON := `{
		"transport": "webhooks",
		"namespace": "default",
		"name": "myhook",
		"options": {
			"withData": true,
			"url": "https://raw.githubusercontent.com/hyperledger-labs/firefly/main/test/data/config/firefly.core.yaml",
			"reply": true,
			"method": "GET"
		}
	}`
	CleanupExistingSubscription(t, ts.client2, "default", "myhook")
	sub := CreateSubscription(t, ts.client2, subJSON, 201)
	assert.NotNil(t, sub.ID)

	var resp *resty.Response
	resp, err := PrivateBlobMessage(t, ts.client1, []string{
		ts.org1.Name,
		ts.org2.Name,
	})
	require.NoError(t, err)
	assert.Equal(t, 202, resp.StatusCode())

	<-received1
	<-received2

	<-received1 // reply
	validateReceivedMessages(ts, ts.client1, nil, fftypes.MessageTypePrivate)

	<-received2 // reply
	validateReceivedMessages(ts, ts.client2, nil, fftypes.MessageTypePrivate)
}
