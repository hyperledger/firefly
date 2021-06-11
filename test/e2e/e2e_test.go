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

type e2eTestState struct {
	startTime time.Time
	t         *testing.T
	client1   *resty.Client
	client2   *resty.Client
	ws1       *websocket.Conn
	ws2       *websocket.Conn
	org1      *fftypes.Organization
	org2      *fftypes.Organization
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

func validateReceivedMessages(ts *e2eTestState, client *resty.Client, value fftypes.Byteable, msgType fftypes.MessageType) {
	messages := GetMessages(ts, client, msgType, 200)
	assert.Equal(ts.t, 1, len(messages))
	assert.Equal(ts.t, fftypes.LowerCasedType("batch_pin"), (messages)[0].Header.TxType)
	assert.Equal(ts.t, "default", (messages)[0].Header.Namespace)
	assert.Equal(ts.t, fftypes.FFNameArray{"default"}, (messages)[0].Header.Topics)

	data := GetData(ts, client, 200)
	assert.Equal(ts.t, 1, len(data))
	assert.Equal(ts.t, "default", (data)[0].Namespace)
	assert.Equal(ts.t, value, (data)[0].Value)
}

func beforeE2ETest(t *testing.T) *e2eTestState {
	stackFile := os.Getenv("STACK_FILE")
	if stackFile == "" {
		t.Fatal("STACK_FILE must be set")
	}

	port1, err := GetMemberPort(stackFile, 0)
	require.NoError(t, err)
	port2, err := GetMemberPort(stackFile, 1)
	require.NoError(t, err)

	ts := &e2eTestState{
		t:         t,
		startTime: time.Now(),
		client1:   resty.New(),
		client2:   resty.New(),
	}

	ts.client1.SetHostURL(fmt.Sprintf("http://localhost:%d/api/v1", port1))
	ts.client2.SetHostURL(fmt.Sprintf("http://localhost:%d/api/v1", port2))

	t.Logf("Client 1: " + ts.client1.HostURL)
	t.Logf("Client 2: " + ts.client2.HostURL)
	pollForUp(t, ts.client1)
	pollForUp(t, ts.client2)

	orgs := GetOrgs(ts, ts.client1, 200)
	require.GreaterOrEqual(t, 2, len(orgs), "Must have two registered orgs: %v", orgs)
	ts.org1 = orgs[0]
	ts.org2 = orgs[1]

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

	return ts
}

func TestE2EBroadcast(t *testing.T) {

	ts := beforeE2ETest(t)

	received1 := make(chan bool)
	go func() {
		for {
			_, _, err := ts.ws1.ReadMessage()
			require.NoError(t, err)
			received1 <- true
		}
	}()

	received2 := make(chan bool)
	go func() {
		for {
			_, _, err := ts.ws2.ReadMessage()
			require.NoError(t, err)
			received2 <- true
		}
	}()

	var resp *resty.Response
	value := fftypes.Byteable(`"Hello"`)
	data := fftypes.DataRefOrValue{
		Value: value,
	}

	resp, err := BroadcastMessage(ts.client1, &data)
	require.NoError(t, err)
	assert.Equal(t, 202, resp.StatusCode())

	<-received1
	validateReceivedMessages(ts, ts.client1, value, fftypes.MessageTypeBroadcast)

	<-received2
	validateReceivedMessages(ts, ts.client2, value, fftypes.MessageTypeBroadcast)
}

func TestE2EPrivate(t *testing.T) {

	ts := beforeE2ETest(t)

	received1 := make(chan bool)
	go func() {
		for {
			_, _, err := ts.ws1.ReadMessage()
			require.NoError(t, err)
			received1 <- true
		}
	}()

	received2 := make(chan bool)
	go func() {
		for {
			_, _, err := ts.ws2.ReadMessage()
			require.NoError(t, err)
			received2 <- true
		}
	}()

	var resp *resty.Response
	value := fftypes.Byteable(`"Hello"`)
	data := fftypes.DataRefOrValue{
		Value: value,
	}

	resp, err := PrivateMessage(ts, ts.client1, &data, []string{
		ts.org1.Name,
		ts.org2.Name,
	})
	require.NoError(t, err)
	assert.Equal(t, 202, resp.StatusCode())

	<-received1
	validateReceivedMessages(ts, ts.client1, value, fftypes.MessageTypePrivate)

	<-received2
	validateReceivedMessages(ts, ts.client2, value, fftypes.MessageTypePrivate)
}
