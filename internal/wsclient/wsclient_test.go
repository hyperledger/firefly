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

	"github.com/gorilla/websocket"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/restclient"
	"github.com/stretchr/testify/assert"
)

var utConfPrefix = config.NewPluginConfig("ws_unit_tests")

func resetConf() {
	config.Reset()
	InitPrefix(utConfPrefix)
}

func TestWSClientE2E(t *testing.T) {

	toServer, fromServer, url, close := NewTestWSServer(func(req *http.Request) {
		assert.Equal(t, "/test/updated", req.URL.Path)
	})
	defer close()

	afterConnect := func(ctx context.Context, w WSClient) error {
		return w.Send(ctx, []byte(`after connect message`))
	}

	// Init from config
	resetConf()
	utConfPrefix.Set(restclient.HTTPConfigURL, url)
	utConfPrefix.Set(WSConfigKeyPath, "/test")
	wsClient, err := New(context.Background(), utConfPrefix, afterConnect)
	assert.NoError(t, err)

	//  Change the settings and connect
	wsClient.SetURL(wsClient.URL() + "/updated")
	err = wsClient.Connect()
	assert.NoError(t, err)

	// Receive the message automatically sent in afterConnect
	message1 := <-toServer
	assert.Equal(t, `after connect message`, message1)

	// Tell the unit test server to send us a reply, and confirm it
	fromServer <- `some data from server`
	reply := <-wsClient.Receive()
	assert.Equal(t, `some data from server`, string(reply))

	// Send some data back
	err = wsClient.Send(context.Background(), []byte(`some data to server`))
	assert.NoError(t, err)

	// Check the sevrer got it
	message2 := <-toServer
	assert.Equal(t, `some data to server`, message2)

	// Close the client
	wsClient.Close()

}

func TestWSClientBadURL(t *testing.T) {
	resetConf()
	utConfPrefix.Set(restclient.HTTPConfigURL, ":::")

	_, err := New(context.Background(), utConfPrefix, nil)
	assert.Regexp(t, "FF10162", err)
}

func TestHTTPToWSURLRemap(t *testing.T) {
	resetConf()
	utConfPrefix.Set(restclient.HTTPConfigURL, "http://test:12345")
	utConfPrefix.Set(WSConfigKeyPath, "/websocket")

	url, err := buildWSUrl(context.Background(), utConfPrefix)
	assert.NoError(t, err)
	assert.Equal(t, "ws://test:12345/websocket", url)
}

func TestHTTPSToWSSURLRemap(t *testing.T) {
	resetConf()
	utConfPrefix.Set(restclient.HTTPConfigURL, "https://test:12345")

	url, err := buildWSUrl(context.Background(), utConfPrefix)
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

	resetConf()
	utConfPrefix.Set(restclient.HTTPConfigURL, fmt.Sprintf("ws://%s", svr.Listener.Addr()))
	utConfPrefix.Set(restclient.HTTPConfigHeaders, map[string]interface{}{
		"custom-header": "custom value",
	})
	utConfPrefix.Set(restclient.HTTPConfigAuthUsername, "user")
	utConfPrefix.Set(restclient.HTTPConfigAuthPassword, "pass")
	utConfPrefix.Set(restclient.HTTPConfigRetryInitDelay, 1)
	utConfPrefix.Set(WSConfigKeyInitialConnectAttempts, 1)

	w, _ := New(context.Background(), utConfPrefix, nil)
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

	resetConf()
	utConfPrefix.Set(restclient.HTTPConfigURL, fmt.Sprintf("ws://%s", svr.Listener.Addr()))
	utConfPrefix.Set(restclient.HTTPConfigRetryInitDelay, 1)
	utConfPrefix.Set(WSConfigKeyInitialConnectAttempts, 1)

	w, _ := New(context.Background(), utConfPrefix, nil)
	err := w.Connect()
	assert.Regexp(t, "FF10161", err)
}

func TestWSSendClosed(t *testing.T) {

	resetConf()
	utConfPrefix.Set(restclient.HTTPConfigURL, "ws://localhost:12345")

	w, err := New(context.Background(), utConfPrefix, nil)
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

	toServer, _, url, done := NewTestWSServer(nil)
	defer done()

	wsconn, _, err := websocket.DefaultDialer.Dial(url, nil)
	wsconn.WriteJSON(map[string]string{"type": "listen", "topic": "topic1"})
	assert.NoError(t, err)
	<-toServer
	wsconn.Close()
	w := &wsClient{
		ctx:      context.Background(),
		closed:   true,
		sendDone: make(chan []byte, 1),
		wsconn:   wsconn,
	}

	// Close the sender channel
	close(w.sendDone)

	// Ensure the readLoop exits immediately
	w.readLoop()

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
