// Copyright © 2020, 2021 Kaleido, Inc.
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

package wsserver

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
)

type webSocketConnection struct {
	id       string
	ctx      context.Context
	server   *webSocketServer
	conn     *ws.Conn
	mux      sync.Mutex
	closed   bool
	topics   map[string]*webSocketTopic
	newTopic chan bool
	receive  chan error
	closing  chan struct{}
}

type webSocketCommandMessage struct {
	Type    string `json:"type,omitempty"`
	Topic   string `json:"topic,omitempty"`
	Message string `json:"message,omitempty"`
}

func newConnection(server *webSocketServer, conn *ws.Conn) *webSocketConnection {
	id := uuid.NewString()
	wsc := &webSocketConnection{
		id:       id,
		server:   server,
		conn:     conn,
		newTopic: make(chan bool),
		topics:   make(map[string]*webSocketTopic),
		receive:  make(chan error),
		closing:  make(chan struct{}),
		ctx:      log.WithLogField(server.ctx, "ws", id),
	}
	go wsc.listen()
	go wsc.sender()
	return wsc
}

func (c *webSocketConnection) close() {
	c.mux.Lock()
	if !c.closed {
		c.closed = true
		c.conn.Close()
		close(c.closing)
	}
	c.mux.Unlock()

	for _, t := range c.topics {
		c.server.cycleTopic(t)
	}
	c.server.connectionClosed(c)
	log.L(c.ctx).Infof("WS/%s: Disconnected", c.id)
}

func (c *webSocketConnection) sender() {
	defer c.close()
	buildCases := func() []reflect.SelectCase {
		c.mux.Lock()
		defer c.mux.Unlock()
		cases := make([]reflect.SelectCase, len(c.topics)+2)
		i := 0
		for _, t := range c.topics {
			cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(t.senderChannel)}
			i++
		}
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(c.closing)}
		i++
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(c.newTopic)}
		return cases
	}
	cases := buildCases()
	for {
		chosen, value, ok := reflect.Select(cases)
		if !ok {
			log.L(c.ctx).Infof("Websocket closing")
			return
		}

		if chosen == len(cases)-1 {
			// Addition of a new topic
			cases = buildCases()
		} else {
			// Message from one of the existing topics
			_ = c.conn.WriteJSON(value.Interface())
		}
	}
}

func (c *webSocketConnection) listenTopic(t *webSocketTopic) {
	c.mux.Lock()
	c.topics[t.topic] = t
	c.mux.Unlock()
	select {
	case c.newTopic <- true:
	case <-c.closing:
	}
}

func (c *webSocketConnection) listen() {
	defer c.close()
	log.L(c.ctx).Infof("Websocket connected")
	for {
		var msg webSocketCommandMessage
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			log.L(c.ctx).Infof("Websocket error: %s", err)
			return
		}
		log.L(c.ctx).Debugf("Websocket received: %+v", msg)

		t := c.server.getTopic(msg.Topic)
		switch msg.Type {
		case "listen":
			c.listenTopic(t)
		case "ack":
			c.handleAckOrError(t, nil)
		case "error":
			c.handleAckOrError(t, i18n.NewError(c.ctx, i18n.MsgWebsocketClientError, msg.Message))
		default:
			log.L(c.ctx).Errorf("Unexpected message type: %+v", msg)
		}
	}
}

func (c *webSocketConnection) handleAckOrError(t *webSocketTopic, err error) {
	isError := err != nil
	select {
	case <-time.After(c.server.processingTimeout):
		log.L(c.ctx).Errorf("Response (error='%t') on topic '%s'. We were not available to process it after %.2f seconds. Closing connection", isError, t.topic, c.server.processingTimeout.Seconds())
		c.close()
	case t.receiverChannel <- err:
		log.L(c.ctx).Debugf("Response (error='%t') on topic '%s' passed on for processing", isError, t.topic)
		break
	}
}
