// Copyright © 2021 Kaleido, Inc.
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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/kaleido-io/firefly/internal/retry"
	"github.com/kaleido-io/firefly/internal/wsserver"
	"github.com/stretchr/testify/assert"
)

func TestWSClientE2E(t *testing.T) {

	wsServer := wsserver.NewWebSocketServer(context.Background())
	svr := httptest.NewServer(wsServer.Handler())
	defer svr.Close()

	acks := make(chan bool)
	sender, receiver, _ := wsServer.GetChannels("topic1")
	go func() {
		sender <- map[string]string{"test": "message"}
		err := <-receiver
		assert.Nil(t, err)
		acks <- true
	}()

	// Send a listen on topic1 in the connect options
	b, _ := json.Marshal(map[string]string{"type": "listen", "topic": "topic1"})

	wsClient, err := NewWSClient(context.Background(), &WSConfig{
		URL: fmt.Sprintf("ws://%s", svr.Listener.Addr()),
	}, b)
	assert.NoError(t, err)

	// Receive the message sent by the server
	b = <-wsClient.Receive()
	var msg map[string]string
	err = json.Unmarshal(b, &msg)
	assert.NoError(t, err)
	assert.Equal(t, "message", msg["test"])

	// Ack it
	b, _ = json.Marshal(map[string]string{"type": "ack", "topic": "topic1"})
	err = wsClient.Send(context.Background(), b)
	assert.NoError(t, err)

	// Wait for server to process our ack
	<-acks

	// Close out
	wsServer.Close()
	wsClient.Close()

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

	var one uint = 1
	_, err := NewWSClient(context.Background(), &WSConfig{
		URL: fmt.Sprintf("ws://%s", svr.Listener.Addr()),
		Headers: map[string]string{
			"custom-header": "custom value",
		},
		Auth: &WSAuthConfig{
			Username: "user",
			Password: "pass",
		},
		WSRetryConfig: WSRetryConfig{
			InitialConnectAttempts: &one,
			MaxWaitTimeMS:          &one,
		},
	}, nil)
	assert.Regexp(t, "FF10161", err.Error())
}

func TestWSFailStartupConnect(t *testing.T) {

	svr := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(500)
		},
	))
	svr.Close()

	var one uint = 1
	_, err := NewWSClient(context.Background(), &WSConfig{
		URL: fmt.Sprintf("ws://%s", svr.Listener.Addr()),
		WSRetryConfig: WSRetryConfig{
			InitialConnectAttempts: &one,
			MaxWaitTimeMS:          &one,
		},
	}, nil)
	assert.Regexp(t, "FF10161", err.Error())
}

func TestWSSendClosed(t *testing.T) {

	wsServer := wsserver.NewWebSocketServer(context.Background())
	svr := httptest.NewServer(wsServer.Handler())
	defer svr.Close()

	w, err := NewWSClient(context.Background(), &WSConfig{
		URL: fmt.Sprintf("ws://%s", svr.Listener.Addr()),
	}, nil)
	assert.NoError(t, err)
	w.Close()

	err = w.Send(context.Background(), []byte(`sent after close`))
	assert.Regexp(t, "FF10160", err.Error())
}

func TestWSSendCancelledContext(t *testing.T) {

	w := &WSClient{
		send:    make(chan []byte),
		closing: make(chan struct{}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := w.Send(ctx, []byte(`sent after close`))
	assert.Regexp(t, "FF10159", err.Error())
}

func TestWSConnectClosed(t *testing.T) {

	w := &WSClient{
		ctx:    context.Background(),
		closed: true,
		retry:  &retry.Retry{},
	}

	err := w.connect(false)
	assert.Regexp(t, "FF10160", err.Error())
}

func TestWSReadLoopCapturePending(t *testing.T) {

	wsServer := wsserver.NewWebSocketServer(context.Background())
	svr := httptest.NewServer(wsServer.Handler())
	defer svr.Close()
	defer wsServer.Close()

	sender, _, _ := wsServer.GetChannels("topic1")
	go func() {
		sender <- map[string]string{"test": "message"}
	}()

	wsconn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s", svr.Listener.Addr()), nil)
	wsconn.WriteJSON(map[string]string{"type": "listen", "topic": "topic1"})
	assert.NoError(t, err)
	defer wsconn.Close()
	w := &WSClient{
		ctx:      context.Background(),
		closed:   true,
		sendDone: make(chan []byte, 1),
		wsconn:   wsconn,
	}

	// Close the sender channel
	w.sendDone <- []byte(`message pending`)
	close(w.sendDone)

	// Go direct into the receive loop
	pendingMsg := w.readLoop()
	assert.Equal(t, `message pending`, string(pendingMsg))

}

func TestWSReconnect(t *testing.T) {

	wsServer := wsserver.NewWebSocketServer(context.Background())
	svr := httptest.NewServer(wsServer.Handler())
	defer svr.Close()
	defer wsServer.Close()

	wsconn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s", svr.Listener.Addr()), nil)
	assert.NoError(t, err)
	wsconn.Close()
	ctxCancelled, cancel := context.WithCancel(context.Background())
	cancel()
	w := &WSClient{
		ctx:     ctxCancelled,
		receive: make(chan []byte),
		send:    make(chan []byte),
		closing: make(chan struct{}),
		wsconn:  wsconn,
		retry:   &retry.Retry{},
	}
	close(w.send) // will mean sender exits immediately

	w.receiveReconnectLoop()
}

func TestWSSendFailPendingMessage(t *testing.T) {

	wsServer := wsserver.NewWebSocketServer(context.Background())
	svr := httptest.NewServer(wsServer.Handler())
	defer svr.Close()
	defer wsServer.Close()

	wsconn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s", svr.Listener.Addr()), nil)
	assert.NoError(t, err)
	wsconn.Close()
	w := &WSClient{
		ctx:      context.Background(),
		receive:  make(chan []byte),
		send:     make(chan []byte),
		closing:  make(chan struct{}),
		sendDone: make(chan []byte, 1),
		wsconn:   wsconn,
		retry:    &retry.Retry{},
	}
	close(w.send) // will mean sender exits immediately

	w.sendLoop([]byte(`pending message`))
	msg := <-w.sendDone
	assert.Equal(t, `pending message`, string(msg))
}
