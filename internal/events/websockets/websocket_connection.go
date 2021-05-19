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

package websockets

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

type websocketConnection struct {
	ctx          context.Context
	ws           *WebSockets
	wsConn       *websocket.Conn
	cancelCtx    func()
	connID       string
	sendMessages chan interface{}
	senderDone   chan struct{}
	startedCount int
	inflight     []*fftypes.EventDeliveryResponse
	mux          sync.Mutex
	closed       bool
}

func newConnection(pCtx context.Context, ws *WebSockets, wsConn *websocket.Conn) *websocketConnection {
	connID := fftypes.NewUUID().String()
	ctx := log.WithLogField(pCtx, "websocket", connID)
	ctx, cancelCtx := context.WithCancel(ctx)
	wc := &websocketConnection{
		ctx:          ctx,
		ws:           ws,
		wsConn:       wsConn,
		cancelCtx:    cancelCtx,
		connID:       connID,
		sendMessages: make(chan interface{}, 1),
		senderDone:   make(chan struct{}),
	}
	go wc.sendLoop()
	go wc.receiveLoop()
	return wc
}

// processAutoStart gives a helper to specify query parameters to auto-start your subscription
func (wc *websocketConnection) processAutoStart(req *http.Request) {
	query := req.URL.Query()
	ephemeral, hasEphemeral := req.URL.Query()["ephemeral"]
	isEphemeral := hasEphemeral && (len(ephemeral) == 0 || ephemeral[0] != "false")
	_, hasName := query["name"]
	if hasEphemeral || hasName {
		err := wc.ws.start(wc.connID, &fftypes.WSClientActionStartPayload{
			Ephemeral: isEphemeral,
			Namespace: query.Get("namespace"),
			Name:      query.Get("name"),
			Filter: fftypes.SubscriptionFilter{
				Events:  query.Get("filter.events"),
				Topic:   query.Get("filter.topic"),
				Group:   query.Get("filter.group"),
				Context: query.Get("filter.context"),
			},
		})
		if err != nil {
			wc.protocolError(err)
		}
	}
}

func (wc *websocketConnection) sendLoop() {
	l := log.L(wc.ctx)
	defer close(wc.senderDone)
	defer wc.close()
	for {
		select {
		case msg, ok := <-wc.sendMessages:
			if !ok {
				l.Debugf("Sender closing")
				return
			}
			l.Tracef("Sending: %+v", msg)
			writer, err := wc.wsConn.NextWriter(websocket.TextMessage)
			if err == nil {
				err = json.NewEncoder(writer).Encode(msg)
				_ = writer.Close()
			}
			if err != nil {
				l.Errorf("Write failed on socket: %s", err)
				return
			}
		case <-wc.ctx.Done():
			l.Debugf("Sender closing - context cancelled")
			return
		}
	}
}

func (wc *websocketConnection) receiveLoop() {
	l := log.L(wc.ctx)
	defer close(wc.sendMessages)
	for {
		var msgData []byte
		var msgHeader fftypes.WSClientActionBase
		_, reader, err := wc.wsConn.NextReader()
		if err == nil {
			msgData, err = ioutil.ReadAll(reader)
			if err == nil {
				err = json.Unmarshal(msgData, &msgHeader)
				if err != nil {
					// We can notify the client on this one, before we bail
					wc.protocolError(i18n.WrapError(wc.ctx, err, i18n.MsgWSClientSentInvalidData))
				}
			}
		}
		if err != nil {
			l.Errorf("Read failed: %s", err)
			return
		}
		l.Tracef("Received: %s", string(msgData))
		switch msgHeader.Type {
		case fftypes.WSClientActionStart:
			var msg fftypes.WSClientActionStartPayload
			err = json.Unmarshal(msgData, &msg)
			if err == nil {
				err = wc.handleStart(&msg)
			}
		case fftypes.WSClientActionAck:
			var msg fftypes.WSClientActionAckPayload
			err = json.Unmarshal(msgData, &msg)
			if err == nil {
				err = wc.handleAck(&msg)
			}
		default:
			err = i18n.NewError(wc.ctx, i18n.MsgWSClientUnknownAction, msgHeader.Type)
		}
		if err != nil {
			wc.protocolError(i18n.WrapError(wc.ctx, err, i18n.MsgWSClientSentInvalidData))
			l.Errorf("Invalid request sent on socket: %s", err)
			return
		}
	}
}

func (wc *websocketConnection) dispatch(event *fftypes.EventDelivery) error {
	wc.mux.Lock()
	wc.inflight = append(wc.inflight, &fftypes.EventDeliveryResponse{
		ID:           event.ID,
		Subscription: event.Subscription,
	})
	wc.mux.Unlock()
	return wc.send(event)
}

func (wc *websocketConnection) protocolError(err error) {
	log.L(wc.ctx).Errorf("Sending protocol error to client: %s", err)
	sendErr := wc.send(&fftypes.WSProtocolErrorPayload{
		Type:  fftypes.WSProtocolErrorEventType,
		Error: err.Error(),
	})
	if sendErr != nil {
		log.L(wc.ctx).Errorf("Failed to send protocol error: %s", sendErr)
	}
}

func (wc *websocketConnection) send(msg interface{}) error {
	select {
	case wc.sendMessages <- msg:
		return nil
	case <-wc.ctx.Done():
		return i18n.NewError(wc.ctx, i18n.MsgWSClosing)
	}
}

func (wc *websocketConnection) handleStart(start *fftypes.WSClientActionStartPayload) (err error) {
	err = wc.ws.start(wc.connID, start)
	if err != nil {
		return err
	}
	wc.mux.Lock()
	wc.startedCount++
	wc.mux.Unlock()
	return nil
}

func (wc *websocketConnection) handleAck(ack *fftypes.WSClientActionAckPayload) error {
	l := log.L(wc.ctx)
	var inflight *fftypes.EventDeliveryResponse
	var subMismatch bool
	wc.mux.Lock()
	if ack.ID != nil {
		for _, candidate := range wc.inflight {
			if *candidate.ID == *ack.ID {
				if ack.Subscription != nil {
					// A subscription has been explicitly specified, so it must match
					if (ack.Subscription.ID != nil && *ack.Subscription.ID == *candidate.Subscription.ID) ||
						(ack.Subscription.Name == candidate.Subscription.Name && ack.Subscription.Namespace == candidate.Subscription.Namespace) {
						inflight = candidate
						break
					}
				} else {
					// If there's more than one started subscription, that's a problem
					if wc.startedCount != 1 {
						l.Errorf("No subscription specified on ack, and there is not exactly one started subscription")
						subMismatch = true
						break
					}
					inflight = candidate
				}
			}
		}
	} else {
		// Just ack the front of the queue
		if len(wc.inflight) == 0 {
			l.Errorf("Ack received, but no messages in flight")
		} else {
			inflight = wc.inflight[0]
			wc.inflight = wc.inflight[1:]
		}
	}
	wc.mux.Unlock()

	// Check we found a match, now we've dropped the lock
	if inflight == nil || subMismatch {
		err := i18n.NewError(wc.ctx, i18n.MsgWSMsgSubNotMatched)
		wc.protocolError(err)
		return err
	}

	// Deliver the ack to the core
	return wc.ws.ack(wc.connID, inflight)
}

func (wc *websocketConnection) close() {
	var didClosed bool
	wc.mux.Lock()
	if !wc.closed {
		didClosed = true
		wc.closed = true
		_ = wc.wsConn.Close()
		wc.cancelCtx()
	}
	wc.mux.Unlock()
	// Drop lock before callback
	if didClosed {
		wc.ws.connClosed(wc.connID)
	}
}

func (wc *websocketConnection) waitClose() {
	<-wc.senderDone
	<-wc.sendMessages
}
