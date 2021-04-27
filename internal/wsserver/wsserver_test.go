// Copyright © 2020, 2021 Kaleido, Inc.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wsserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	ws "github.com/gorilla/websocket"

	"github.com/stretchr/testify/assert"
)

func newTestWebSocketServer() (*webSocketServer, *httptest.Server) {
	s := NewWebSocketServer(context.Background()).(*webSocketServer)
	ts := httptest.NewServer(s.Handler())
	return s, ts
}

func TestConnectSendReceiveCycle(t *testing.T) {
	assert := assert.New(t)

	w, ts := newTestWebSocketServer()
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	u.Scheme = "ws"
	u.Path = "/ws"
	c, _, err := ws.DefaultDialer.Dial(u.String(), nil)
	assert.NoError(err)

	c.WriteJSON(&webSocketCommandMessage{
		Type: "listen",
	})

	s, r, _ := w.GetChannels("")

	s <- "Hello World"

	var val string
	c.ReadJSON(&val)
	assert.Equal("Hello World", val)

	c.WriteJSON(&webSocketCommandMessage{
		Type: "ignoreme",
	})

	c.WriteJSON(&webSocketCommandMessage{
		Type: "ack",
	})
	err = <-r
	assert.NoError(err)

	s <- "Don't Panic!"

	c.ReadJSON(&val)
	assert.Equal("Don't Panic!", val)

	c.WriteJSON(&webSocketCommandMessage{
		Type:    "error",
		Message: "Panic!",
	})

	err = <-r
	assert.EqualError(err, "FF10108: Error received from WebSocket client: Panic!")

	w.Close()

}

func TestConnectTopicIsolation(t *testing.T) {
	assert := assert.New(t)

	w, ts := newTestWebSocketServer()
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	u.Scheme = "ws"
	u.Path = "/ws"
	c1, _, err := ws.DefaultDialer.Dial(u.String(), nil)
	assert.NoError(err)
	c2, _, err := ws.DefaultDialer.Dial(u.String(), nil)
	assert.NoError(err)

	c1.WriteJSON(&webSocketCommandMessage{
		Type:  "listen",
		Topic: "topic1",
	})

	c2.WriteJSON(&webSocketCommandMessage{
		Type:  "listen",
		Topic: "topic2",
	})

	s1, r1, _ := w.GetChannels("topic1")
	s2, r2, _ := w.GetChannels("topic2")

	s1 <- "Hello Number 1"
	s2 <- "Hello Number 2"

	var val string
	c1.ReadJSON(&val)
	assert.Equal("Hello Number 1", val)
	c1.WriteJSON(&webSocketCommandMessage{
		Type:  "ack",
		Topic: "topic1",
	})
	err = <-r1
	assert.NoError(err)

	c2.ReadJSON(&val)
	assert.Equal("Hello Number 2", val)
	c2.WriteJSON(&webSocketCommandMessage{
		Type:  "ack",
		Topic: "topic2",
	})
	err = <-r2
	assert.NoError(err)

	w.Close()

}

func TestConnectAbandonRequest(t *testing.T) {
	assert := assert.New(t)

	w, ts := newTestWebSocketServer()
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	u.Scheme = "ws"
	u.Path = "/ws"
	c, _, err := ws.DefaultDialer.Dial(u.String(), nil)
	assert.NoError(err)

	c.WriteJSON(&webSocketCommandMessage{
		Type: "listen",
	})
	_, r, closing := w.GetChannels("")

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		select {
		case <-r:
			break
		case <-closing:
			break
		}
		wg.Done()
	}()

	// Close the client while we've got an active read stream
	c.Close()

	// We whould find the read stream closes out
	wg.Wait()
	w.Close()

}

func TestTimeoutProcessing(t *testing.T) {
	assert := assert.New(t)

	w, ts := newTestWebSocketServer()
	defer ts.Close()
	w.processingTimeout = 1 * time.Millisecond

	u, err := url.Parse(ts.URL)
	u.Scheme = "ws"
	u.Path = "/ws"
	c, _, err := ws.DefaultDialer.Dial(u.String(), nil)
	assert.NoError(err)

	c.WriteJSON(&webSocketCommandMessage{
		Type: "ack",
	})

	// Confirm we close the connection after the timeout pops
	for len(w.connections) > 0 {
		time.Sleep(1 * time.Millisecond)
	}

	w.Close()

}

func TestConnectBadWebsocketHandshake(t *testing.T) {
	assert := assert.New(t)

	w, ts := newTestWebSocketServer()
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	u.Path = "/ws"

	res, err := http.Get(u.String())
	assert.NoError(err)
	assert.Equal(400, res.StatusCode)

	w.Close()

}
