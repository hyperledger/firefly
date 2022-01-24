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

package wsclient

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/websocket"
)

// NewTestWSServer creates a little test server for packages (including wsclient itself) to use in unit tests
func NewTestWSServer(testReq func(req *http.Request)) (toServer, fromServer chan string, url string, done func()) {
	upgrader := &websocket.Upgrader{WriteBufferSize: 1024, ReadBufferSize: 1024}
	toServer = make(chan string, 1)
	fromServer = make(chan string, 1)
	sendDone := make(chan struct{})
	receiveDone := make(chan struct{})
	connected := false
	svr := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if testReq != nil {
			testReq(req)
		}
		ws, _ := upgrader.Upgrade(res, req, http.Header{})
		go func() {
			defer close(receiveDone)
			for {
				_, data, err := ws.ReadMessage()
				if err != nil {
					return
				}
				toServer <- string(data)
			}
		}()
		go func() {
			defer close(sendDone)
			defer ws.Close()
			for data := range fromServer {
				_ = ws.WriteMessage(websocket.TextMessage, []byte(data))
			}
		}()
		connected = true
	}))
	return toServer, fromServer, fmt.Sprintf("ws://%s", svr.Listener.Addr()), func() {
		close(fromServer)
		svr.Close()
		if connected {
			<-sendDone
			<-receiveDone
		}
	}
}
