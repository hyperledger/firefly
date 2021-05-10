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
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kaleido-io/firefly/internal/log"
)

// WebSocketChannels is provided to allow us to do a blocking send to a namespace that will complete once a client connects on it
// We also provide a channel to listen on for closing of the connection, to allow a select to wake on a blocking send
type WebSocketChannels interface {
	GetChannels(topic string) (sender chan<- interface{}, receiver <-chan error, closing <-chan struct{})
}

// WebSocketServer is the full server interface with the init call
type WebSocketServer interface {
	WebSocketChannels
	Handler() http.HandlerFunc
	Close()
}

type webSocketServer struct {
	ctx               context.Context
	processingTimeout time.Duration
	mux               sync.Mutex
	topics            map[string]*webSocketTopic
	upgrader          *websocket.Upgrader
	connections       map[string]*webSocketConnection
}

type webSocketTopic struct {
	topic           string
	senderChannel   chan interface{}
	receiverChannel chan error
	closingChannel  chan struct{}
}

// NewWebSocketServer create a new server with a simplified interface
func NewWebSocketServer(ctx context.Context) WebSocketServer {
	return &webSocketServer{
		ctx:               ctx,
		connections:       make(map[string]*webSocketConnection),
		topics:            make(map[string]*webSocketTopic),
		processingTimeout: 30 * time.Second,
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

func (s *webSocketServer) handler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.L(s.ctx).Errorf("WebSocket upgrade failed: %s", err)
		return
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	c := newConnection(s, conn)
	s.connections[c.id] = c
}

func (s *webSocketServer) cycleTopic(t *webSocketTopic) {
	s.mux.Lock()
	defer s.mux.Unlock()

	// When a connection that was listening on a topic closes, we need to wake anyone
	// that was listening for a response
	close(t.closingChannel)
	t.closingChannel = make(chan struct{})
}

func (s *webSocketServer) connectionClosed(c *webSocketConnection) {
	s.mux.Lock()
	defer s.mux.Unlock()
	delete(s.connections, c.id)
}

func (s *webSocketServer) Handler() http.HandlerFunc {
	return s.handler
}

func (s *webSocketServer) Close() {
	for _, c := range s.connections {
		c.close()
	}
}

func (s *webSocketServer) getTopic(topic string) *webSocketTopic {
	s.mux.Lock()
	defer s.mux.Unlock()
	t, exists := s.topics[topic]
	if !exists {
		t = &webSocketTopic{
			topic:           topic,
			senderChannel:   make(chan interface{}),
			receiverChannel: make(chan error),
			closingChannel:  make(chan struct{}),
		}
		s.topics[topic] = t
	}
	return t
}

func (s *webSocketServer) GetChannels(topic string) (sender chan<- interface{}, receiver <-chan error, closing <-chan struct{}) {
	t := s.getTopic(topic)
	return t.senderChannel, t.receiverChannel, t.closingChannel
}
