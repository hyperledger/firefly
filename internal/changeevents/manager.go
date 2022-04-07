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
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type ChangeEventManager interface {
	Dispatch(changeEvent *fftypes.ChangeEvent)
	ServeHTTPWebSocketListener(res http.ResponseWriter, req *http.Request)
	WaitClose()
}

type changeEventManager struct {
	ctx              context.Context
	cancelCtx        func()
	activeWebsockets map[string]*webSocket
	dirtyReadList    []*webSocket
	mux              sync.Mutex
	upgrader         websocket.Upgrader

	queueLength         int
	blockedWarnInterval time.Duration
}

func NewChangeEventManager(ctx context.Context) ChangeEventManager {
	ce := &changeEventManager{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  int(config.GetByteSize(config.AdminWebSocketReadBufferSize)),
			WriteBufferSize: int(config.GetByteSize(config.AdminWebSocketWriteBufferSize)),
			CheckOrigin: func(r *http.Request) bool {
				// Cors is handled by the API server that wraps this handler
				return true
			},
		},
		queueLength:         config.GetInt(config.AdminWebSocketEventQueueLength),
		blockedWarnInterval: config.GetDuration(config.AdminWebSocketBlockedWarnInterval),
	}
	ce.ctx, ce.cancelCtx = context.WithCancel(
		log.WithLogField(ctx, "role", "change-event-manager"),
	)
	return ce
}

func (ce *changeEventManager) ServeHTTPWebSocketListener(res http.ResponseWriter, req *http.Request) {
	wsConn, err := ce.upgrader.Upgrade(res, req, nil)
	if err != nil {
		log.L(ce.ctx).Errorf("WebSocket upgrade failed: %s", err)
		return
	}

	ce.mux.Lock()
	wc := newWebSocket(ce, wsConn)
	ce.activeWebsockets[wc.connID] = wc
	ce.makeDirtyReadList()
	ce.mux.Unlock()
}

func (ce *changeEventManager) wsClosed(connID string) {
	ce.mux.Lock()
	delete(ce.activeWebsockets, connID)
	ce.makeDirtyReadList()
	ce.mux.Unlock()
}

func (ce *changeEventManager) WaitClose() {
	ce.cancelCtx()

	ce.mux.Lock()
	activeWebsockets := make([]*webSocket, 0, len(ce.activeWebsockets))
	for _, ws := range ce.activeWebsockets {
		activeWebsockets = append(activeWebsockets, ws)
	}
	ce.mux.Unlock()

	for _, ws := range activeWebsockets {
		ws.waitClose()
	}
}

func (ce *changeEventManager) makeDirtyReadList() {
	ce.dirtyReadList = make([]*webSocket, len(ce.activeWebsockets))
	for _, ws := range ce.activeWebsockets {
		ce.dirtyReadList = append(ce.dirtyReadList, ws)
	}
}

func (ce *changeEventManager) Dispatch(changeEvent *fftypes.ChangeEvent) {
	// We don't lock here, as this is a critical path function.
	// We use a dirty copy of the connection list, updated on add/remove
	for _, ws := range ce.dirtyReadList {
		ws.dispatch(changeEvent)
	}
}
