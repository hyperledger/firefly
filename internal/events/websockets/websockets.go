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

package websockets

import (
	"context"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/events"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

type WebSockets struct {
	ctx          context.Context
	capabilities *events.Capabilities
	callbacks    events.Callbacks
	connections  map[string]*websocketConnection
	connMux      sync.Mutex
	upgrader     websocket.Upgrader
}

func (ws *WebSockets) Name() string { return "websockets" }

func (ws *WebSockets) Init(ctx context.Context, prefix config.Prefix, callbacks events.Callbacks) error {
	*ws = WebSockets{
		ctx:          ctx,
		connections:  make(map[string]*websocketConnection),
		capabilities: &events.Capabilities{},
		callbacks:    callbacks,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  int(prefix.GetByteSize(ReadBufferSize)),
			WriteBufferSize: int(prefix.GetByteSize(WriteBufferSize)),
			CheckOrigin: func(r *http.Request) bool {
				// Cors is handled by the API server that wraps this handler
				return true
			},
		},
	}
	return nil
}

func (ws *WebSockets) Capabilities() *events.Capabilities {
	return ws.capabilities
}

func (ws *WebSockets) ValidateOptions(transportOptions fftypes.JSONObject) error {
	// We don't have any additional options currently
	return nil
}

func (ws *WebSockets) DeliveryRequest(connID string, sub *fftypes.Subscription, event *fftypes.EventDelivery) error {
	ws.connMux.Lock()
	conn, ok := ws.connections[connID]
	ws.connMux.Unlock()
	if !ok {
		return i18n.NewError(ws.ctx, i18n.MsgWSConnectionNotActive, connID)
	}
	return conn.dispatch(event)
}

func (ws *WebSockets) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	wsConn, err := ws.upgrader.Upgrade(res, req, nil)
	if err != nil {
		log.L(ws.ctx).Errorf("WebSocket upgrade failed: %s", err)
		return
	}

	ws.connMux.Lock()
	wc := newConnection(ws.ctx, ws, wsConn)
	ws.connections[wc.connID] = wc
	ws.connMux.Unlock()

	wc.processAutoStart(req)
}

func (ws *WebSockets) ack(connID string, inflight *fftypes.EventDeliveryResponse) error {
	return ws.callbacks.DeliveryResponse(connID, *inflight)
}

func (ws *WebSockets) start(connID string, start *fftypes.WSClientActionStartPayload) error {
	if start.Namespace == "" || (!start.Ephemeral && start.Name == "") {
		return i18n.NewError(ws.ctx, i18n.MsgWSInvalidStartAction)
	}
	if start.Ephemeral {
		return ws.callbacks.EphemeralSubscription(connID, start.Namespace, start.Filter, start.Options)
	}
	return ws.callbacks.RegisterConnection(connID, func(sr fftypes.SubscriptionRef) bool {
		return sr.Namespace == start.Namespace && sr.Name == start.Name
	})
}

func (ws *WebSockets) connClosed(connID string) {
	ws.connMux.Lock()
	delete(ws.connections, connID)
	ws.connMux.Unlock()
	// Drop lock before calling back
	ws.callbacks.ConnnectionClosed(connID)
}

func (ws *WebSockets) WaitClosed() {
	closedConnections := []*websocketConnection{}
	ws.connMux.Lock()
	for _, ws := range ws.connections {
		closedConnections = append(closedConnections, ws)
	}
	ws.connMux.Unlock()
	for _, ws := range closedConnections {
		ws.waitClose()
	}
}
