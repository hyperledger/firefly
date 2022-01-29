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

package wsclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func generateConfig() *WSConfig {
	return &WSConfig{}
}

func TestWSClientE2E(t *testing.T) {

	toServer, fromServer, url, close := NewTestWSServer(func(req *http.Request) {
		assert.Equal(t, "/test/updated", req.URL.Path)
	})
	defer close()

	afterConnect := func(ctx context.Context, w WSClient) error {
		return w.Send(ctx, []byte(`after connect message`))
	}

	// Init clean config
	wsConfig := generateConfig()

	wsConfig.HTTPURL = url
	wsConfig.WSKeyPath = "/test"
	wsConfig.HeartbeatInterval = 50 * time.Millisecond

	wsc, err := New(context.Background(), wsConfig, afterConnect)
	assert.NoError(t, err)

	//  Change the settings and connect
	wsc.SetURL(wsc.URL() + "/updated")
	err = wsc.Connect()
	assert.NoError(t, err)

	// Receive the message automatically sent in afterConnect
	message1 := <-toServer
	assert.Equal(t, `after connect message`, message1)

	// Tell the unit test server to send us a reply, and confirm it
	fromServer <- `some data from server`
	reply := <-wsc.Receive()
	assert.Equal(t, `some data from server`, string(reply))

	// Send some data back
	err = wsc.Send(context.Background(), []byte(`some data to server`))
	assert.NoError(t, err)

	// Check the sevrer got it
	message2 := <-toServer
	assert.Equal(t, `some data to server`, message2)

	// Check heartbeating works
	beforePing := time.Now()
	for wsc.(*wsClient).lastPingCompleted.Before(beforePing) {
		time.Sleep(10 * time.Millisecond)
	}

	// Close the client
	wsc.Close()

}

func TestWSClientBadURL(t *testing.T) {
	wsConfig := generateConfig()
	wsConfig.HTTPURL = ":::"

	_, err := New(context.Background(), wsConfig, nil)
	assert.Regexp(t, "FF10162", err)
}

func TestHTTPToWSURLRemap(t *testing.T) {
	wsConfig := generateConfig()
	wsConfig.HTTPURL = "http://test:12345"
	wsConfig.WSKeyPath = "/websocket"

	url, err := buildWSUrl(context.Background(), wsConfig)
	assert.NoError(t, err)
	assert.Equal(t, "ws://test:12345/websocket", url)
}

func TestHTTPSToWSSURLRemap(t *testing.T) {
	wsConfig := generateConfig()
	wsConfig.HTTPURL = "https://test:12345"

	url, err := buildWSUrl(context.Background(), wsConfig)
	assert.NoError(t, err)
	assert.Equal(t, "wss://test:12345", url)
}

func TestWSFailStartupHttp500(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "custom value", r.Header.Get("Custom-Header"))
			assert.Equal(t, "Basic dXNlcjpwYXNz", r.Header.Get("Authorization"))
			rw.WriteHeader(500)
			rw.Write([]byte(`{"error": "pop"}`))
		},
	))
	defer svr.Close()

	wsConfig := generateConfig()
	wsConfig.HTTPURL = fmt.Sprintf("ws://%s", svr.Listener.Addr())
	wsConfig.HTTPHeaders = map[string]interface{}{
		"custom-header": "custom value",
	}
	wsConfig.AuthUsername = "user"
	wsConfig.AuthPassword = "pass"
	wsConfig.InitialDelay = 1
	wsConfig.InitialConnectAttempts = 1

	w, _ := New(context.Background(), wsConfig, nil)
	err := w.Connect()
	assert.Regexp(t, "FF10161", err)
}

func TestWSFailStartupConnect(t *testing.T) {

	svr := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(500)
		},
	))
	svr.Close()

	wsConfig := generateConfig()
	wsConfig.HTTPURL = fmt.Sprintf("ws://%s", svr.Listener.Addr())
	wsConfig.InitialDelay = 1
	wsConfig.InitialConnectAttempts = 1

	w, _ := New(context.Background(), wsConfig, nil)
	err := w.Connect()
	assert.Regexp(t, "FF10161", err)
}

func TestWSSendClosed(t *testing.T) {

	wsConfig := generateConfig()
	wsConfig.HTTPURL = "http://test:12345"

	w, err := New(context.Background(), wsConfig, nil)
	assert.NoError(t, err)
	w.Close()

	err = w.Send(context.Background(), []byte(`sent after close`))
	assert.Regexp(t, "FF10160", err)
}

func TestWSSendCancelledContext(t *testing.T) {

	w := &wsClient{
		send:    make(chan []byte),
		closing: make(chan struct{}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := w.Send(ctx, []byte(`sent after close`))
	assert.Regexp(t, "FF10159", err)
}

func TestWSConnectClosed(t *testing.T) {

	w := &wsClient{
		ctx:    context.Background(),
		closed: true,
	}

	err := w.connect(false)
	assert.Regexp(t, "FF10160", err)
}

func TestWSReadLoopSendFailure(t *testing.T) {

	toServer, fromServer, url, done := NewTestWSServer(nil)
	defer done()

	wsconn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	wsconn.WriteJSON(map[string]string{"type": "listen", "topic": "topic1"})
	assert.NoError(t, err)
	<-toServer
	w := &wsClient{
		ctx:      context.Background(),
		sendDone: make(chan []byte, 1),
		wsconn:   wsconn,
	}

	// Queue a message for the receiver, then immediately close the sender channel
	fromServer <- `some data from server`
	close(w.sendDone)

	// Ensure the readLoop exits immediately
	w.readLoop()

	// Try reconnect, should fail here
	_, _, err = websocket.DefaultDialer.Dial(url, nil)
	assert.Error(t, err)

}

func TestWSReconnectFail(t *testing.T) {

	_, _, url, done := NewTestWSServer(nil)
	defer done()

	wsconn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	wsconn.Close()
	ctxCancelled, cancel := context.WithCancel(context.Background())
	cancel()
	w := &wsClient{
		ctx:     ctxCancelled,
		receive: make(chan []byte),
		send:    make(chan []byte),
		closing: make(chan struct{}),
		wsconn:  wsconn,
	}
	close(w.send) // will mean sender exits immediately

	w.receiveReconnectLoop()
}

func TestWSSendFail(t *testing.T) {

	_, _, url, done := NewTestWSServer(nil)
	defer done()

	wsconn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	wsconn.Close()
	w := &wsClient{
		ctx:      context.Background(),
		receive:  make(chan []byte),
		send:     make(chan []byte, 1),
		closing:  make(chan struct{}),
		sendDone: make(chan []byte, 1),
		wsconn:   wsconn,
	}
	w.send <- []byte(`wakes sender`)
	w.sendLoop(make(chan struct{}))
	<-w.sendDone
}

func TestWSSendInstructClose(t *testing.T) {

	_, _, url, done := NewTestWSServer(nil)
	defer done()

	wsconn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	wsconn.Close()
	w := &wsClient{
		ctx:      context.Background(),
		receive:  make(chan []byte),
		send:     make(chan []byte, 1),
		closing:  make(chan struct{}),
		sendDone: make(chan []byte, 1),
		wsconn:   wsconn,
	}
	receiverClosed := make(chan struct{})
	close(receiverClosed)
	w.sendLoop(receiverClosed)
	<-w.sendDone
}

func TestHeartbeatTimedout(t *testing.T) {

	now := time.Now()
	w := &wsClient{
		ctx:               context.Background(),
		sendDone:          make(chan []byte),
		heartbeatInterval: 1 * time.Microsecond,
		activePingSent:    &now,
	}

	w.sendLoop(make(chan struct{}))

}

func TestHeartbeatSendFailed(t *testing.T) {

	_, _, url, close := NewTestWSServer(func(req *http.Request) {})
	defer close()

	wsc, err := New(context.Background(), &WSConfig{HTTPURL: url}, func(ctx context.Context, w WSClient) error { return nil })
	assert.NoError(t, err)
	defer wsc.Close()

	err = wsc.Connect()
	assert.NoError(t, err)

	// Close and use the underlying wsconn to drive a failure to send a heartbeat
	wsc.(*wsClient).wsconn.Close()
	w := &wsClient{
		ctx:               context.Background(),
		sendDone:          make(chan []byte),
		heartbeatInterval: 1 * time.Microsecond,
		wsconn:            wsc.(*wsClient).wsconn,
	}

	w.sendLoop(make(chan struct{}))

}
