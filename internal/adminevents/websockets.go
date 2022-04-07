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

package adminevents

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type webSocket struct {
	ctx          context.Context
	manager      *adminEventManager
	wsConn       *websocket.Conn
	cancelCtx    func()
	connID       string
	senderDone   chan struct{}
	receiverDone chan struct{}
	events       chan *fftypes.ChangeEvent
	collections  []string
	filter       fftypes.ChangeEventFilter
	mux          sync.Mutex
	closed       bool
	blockedTime  *fftypes.FFTime
	lastWarnTime *fftypes.FFTime
}

func newWebSocket(ae *adminEventManager, wsConn *websocket.Conn) *webSocket {
	connID := fftypes.NewUUID().String()
	ctx := log.WithLogField(ae.ctx, "admin-websocket", connID)
	ctx, cancelCtx := context.WithCancel(ctx)
	wc := &webSocket{
		ctx:          ctx,
		manager:      ae,
		wsConn:       wsConn,
		cancelCtx:    cancelCtx,
		connID:       connID,
		events:       make(chan *fftypes.ChangeEvent, ae.queueLength),
		senderDone:   make(chan struct{}),
		receiverDone: make(chan struct{}),
	}
	go wc.sendLoop()
	go wc.receiveLoop()
	return wc
}

func (wc *webSocket) eventMatches(changeEvent *fftypes.ChangeEvent) bool {
	collectionMatches := false
	for _, c := range wc.collections {
		if c == changeEvent.Collection {
			collectionMatches = true
			break
		}
	}
	if !collectionMatches {
		return false
	}
	if len(wc.filter.Namespaces) > 0 {
		namespaceMatches := false
		for _, ns := range wc.filter.Namespaces {
			if ns == changeEvent.Namespace {
				namespaceMatches = true
				break
			}
		}
		if !namespaceMatches {
			return false
		}
	}
	if len(wc.filter.Types) > 0 {
		typeMatches := false
		for _, t := range wc.filter.Types {
			if t == changeEvent.Type {
				typeMatches = true
				break
			}
		}
		if !typeMatches {
			return false
		}
	}
	return true
}

func (wc *webSocket) writeObject(obj interface{}) {
	writer, err := wc.wsConn.NextWriter(websocket.TextMessage)
	if err == nil {
		err = json.NewEncoder(writer).Encode(obj)
		_ = writer.Close()
	}
	if err != nil {
		// Log and continue - the receiver closing will be what ends our loop
		log.L(wc.ctx).Errorf("Write failed on socket: %s", err)
	}
}

func (wc *webSocket) sendLoop() {
	l := log.L(wc.ctx)
	defer close(wc.senderDone)
	defer wc.close()
	for {
		select {
		case changeEvent := <-wc.events:
			if !wc.eventMatches(changeEvent) {
				continue
			}
			l.Tracef("Sending: %+v", changeEvent)
			wc.writeObject(changeEvent)
		case <-wc.receiverDone:
			l.Debugf("Sender closing - receiver completed")
			return
		case <-wc.ctx.Done():
			l.Debugf("Sender closing - context cancelled")
			return
		}
	}
}

func (wc *webSocket) receiveLoop() {
	l := log.L(wc.ctx)
	defer close(wc.receiverDone)
	for {
		var msgData []byte
		var cmd fftypes.WSChangeEventCommand
		_, reader, err := wc.wsConn.NextReader()
		if err == nil {
			msgData, err = ioutil.ReadAll(reader)
			if err == nil {
				err = json.Unmarshal(msgData, &cmd)
			}
		}
		if err != nil {
			l.Errorf("Read failed: %s", err)
			return
		}
		l.Tracef("Received: %s", string(msgData))
		switch cmd.Type {
		case fftypes.WSChangeEventCommandTypeStart:
			wc.handleStart(&cmd)
		default:
			l.Errorf("Invalid request sent on socket: %+v", cmd)
		}
	}
}

func (wc *webSocket) dispatch(event *fftypes.ChangeEvent) {
	// We take as much as we possibly can off of this function, including string matching etc.
	// This function is called on the critical path of the commit for all database operations.
	select {
	case wc.events <- event:
		if wc.blockedTime != nil {
			wc.blockedTime = nil
		}
	default:
		if wc.blockedTime == nil {
			wc.blockedTime = fftypes.Now()
		}
		if wc.lastWarnTime == nil || time.Since(*wc.lastWarnTime.Time()) > wc.manager.blockedWarnInterval {
			log.L(wc.ctx).Warnf("Change event listener is blocked an missing events (since %s)", wc.blockedTime)
		}
	}
}

func (wc *webSocket) handleStart(start *fftypes.WSChangeEventCommand) {
	wc.mux.Lock()
	wc.collections = start.Collections
	wc.filter = start.Filter
	wc.mux.Unlock()
}

func (wc *webSocket) close() {
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
		wc.manager.wsClosed(wc.connID)
	}
}

func (wc *webSocket) waitClose() {
	<-wc.senderDone
	<-wc.receiverDone
}
