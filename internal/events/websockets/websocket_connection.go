// Copyright © 2022 Kaleido, Inc.
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

package websockets

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type websocketStartedSub struct {
	ephemeral bool
	name      string
	namespace string
}

type websocketConnection struct {
	ctx                context.Context
	ws                 *WebSockets
	wsConn             *websocket.Conn
	cancelCtx          func()
	connID             string
	sendMessages       chan interface{}
	senderDone         chan struct{}
	receiverDone       chan struct{}
	autoAck            bool
	started            []*websocketStartedSub
	inflight           []*fftypes.EventDeliveryResponse
	mux                sync.Mutex
	closed             bool
	changeEventMatcher *regexp.Regexp
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
		sendMessages: make(chan interface{}),
		senderDone:   make(chan struct{}),
		receiverDone: make(chan struct{}),
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
	autoAck, hasAutoack := req.URL.Query()["autoack"]
	isAutoack := hasAutoack && (len(autoAck) == 0 || autoAck[0] != "false")
	if hasEphemeral || hasName {
		err := wc.handleStart(&fftypes.WSClientActionStartPayload{
			AutoAck:   &isAutoack,
			Ephemeral: isEphemeral,
			Namespace: query.Get("namespace"),
			Name:      query.Get("name"),
			Filter: fftypes.SubscriptionFilter{
				Events: query.Get("filter.events"),
				Topics: query.Get("filter.topics"),
				Group:  query.Get("filter.group"),
				Tag:    query.Get("filter.tag"),
			},
			ChangeEvents: query.Get("changeevents"),
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
		case msg := <-wc.sendMessages:
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
		case <-wc.receiverDone:
			l.Debugf("Sender closing - receiver completed")
			return
		case <-wc.ctx.Done():
			l.Debugf("Sender closing - context cancelled")
			return
		}
	}
}

func (wc *websocketConnection) receiveLoop() {
	l := log.L(wc.ctx)
	defer close(wc.receiverDone)
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

func (wc *websocketConnection) dispatchChangeEvent(ce *fftypes.ChangeEvent) error {
	if wc.changeEventMatcher == nil || !wc.changeEventMatcher.MatchString(ce.Collection) {
		return nil
	}
	// Change events do *NOT* require an ack
	return wc.send(&fftypes.WSChangeNotification{
		WSClientActionBase: fftypes.WSClientActionBase{
			Type: fftypes.WSClientActionChangeNotifcation,
		},
		ChangeEvent: ce,
	})
}

func (wc *websocketConnection) dispatch(event *fftypes.EventDelivery) error {
	inflight := &fftypes.EventDeliveryResponse{
		ID:           event.ID,
		Subscription: event.Subscription,
	}

	var autoAck bool
	wc.mux.Lock()
	autoAck = wc.autoAck
	if !autoAck {
		wc.inflight = append(wc.inflight, inflight)
	}
	wc.mux.Unlock()

	err := wc.send(event)
	if err != nil {
		return err
	}

	if autoAck {
		wc.ws.ack(wc.connID, inflight)
	}

	return nil
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
	if wc.closed {
		return i18n.NewError(wc.ctx, i18n.MsgWSClosed)
	}
	select {
	case wc.sendMessages <- msg:
		return nil
	case <-wc.ctx.Done():
		return i18n.NewError(wc.ctx, i18n.MsgWSClosing)
	}
}

func (wc *websocketConnection) handleStart(start *fftypes.WSClientActionStartPayload) (err error) {
	wc.mux.Lock()
	if start.AutoAck != nil {
		if *start.AutoAck != wc.autoAck && len(wc.started) > 0 {
			return i18n.NewError(wc.ctx, i18n.MsgWSAutoAckChanged)
		}
		wc.autoAck = *start.AutoAck
	}
	wc.mux.Unlock()

	if start.ChangeEvents != "" {
		wc.changeEventMatcher, err = regexp.Compile(start.ChangeEvents)
		if err != nil {
			log.L(wc.ctx).Errorf("Unable to compile change events regular expression '%s': %s", start.ChangeEvents, err)
		} else {
			start.Options.ChangeEvents = true
		}
	}

	wc.mux.Lock()
	wc.started = append(wc.started, &websocketStartedSub{
		ephemeral: start.Ephemeral,
		namespace: start.Namespace,
		name:      start.Name,
	})
	wc.mux.Unlock()
	err = wc.ws.start(wc, start)
	if err != nil {
		return err
	}
	return nil
}

func (wc *websocketConnection) durableSubMatcher(sr fftypes.SubscriptionRef) bool {
	wc.mux.Lock()
	defer wc.mux.Unlock()
	for _, startedSub := range wc.started {
		if !startedSub.ephemeral && startedSub.namespace == sr.Namespace && startedSub.name == sr.Name {
			return true
		}
	}
	return false
}

func (wc *websocketConnection) checkAck(ack *fftypes.WSClientActionAckPayload) (*fftypes.EventDeliveryResponse, error) {
	l := log.L(wc.ctx)
	var inflight *fftypes.EventDeliveryResponse
	wc.mux.Lock()
	defer wc.mux.Unlock()

	if wc.autoAck {
		return nil, i18n.NewError(wc.ctx, i18n.MsgWSAutoAckEnabled)
	}

	if ack.ID != nil {
		newInflight := make([]*fftypes.EventDeliveryResponse, 0, len(wc.inflight))
		for _, candidate := range wc.inflight {
			var match bool
			if *candidate.ID == *ack.ID {
				if ack.Subscription != nil {
					// A subscription has been explicitly specified, so it must match
					if (ack.Subscription.ID != nil && *ack.Subscription.ID == *candidate.Subscription.ID) ||
						(ack.Subscription.Name == candidate.Subscription.Name && ack.Subscription.Namespace == candidate.Subscription.Namespace) {
						match = true
					}
				} else {
					// If there's more than one started subscription, that's a problem
					if len(wc.started) != 1 {
						l.Errorf("No subscription specified on ack, and there is not exactly one started subscription")
						return nil, i18n.NewError(wc.ctx, i18n.MsgWSMsgSubNotMatched)
					}
					match = true
				}
			}
			// Remove from the inflight list
			if match {
				inflight = candidate
			} else {
				newInflight = append(newInflight, candidate)
			}
		}
		wc.inflight = newInflight
	} else {
		// Just ack the front of the queue
		if len(wc.inflight) == 0 {
			l.Errorf("Ack received, but no messages in flight")
		} else {
			inflight = wc.inflight[0]
			wc.inflight = wc.inflight[1:]
		}
	}
	if inflight == nil {
		return nil, i18n.NewError(wc.ctx, i18n.MsgWSMsgSubNotMatched)
	}
	return inflight, nil
}

func (wc *websocketConnection) handleAck(ack *fftypes.WSClientActionAckPayload) error {
	// Perform a locked set of check
	inflight, err := wc.checkAck(ack)
	if err != nil {
		return err
	}

	// Deliver the ack to the core, now we're unlocked
	wc.ws.ack(wc.connID, inflight)
	return nil
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
	<-wc.receiverDone
}
