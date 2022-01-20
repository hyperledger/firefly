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
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type WSConfig struct {
	HTTPURL                string             `json:"httpUrl,omitempty"`
	WSKeyPath              string             `json:"wsKeyPath,omitempty"`
	ReadBufferSize         int                `json:"readBufferSize,omitempty"`
	WriteBufferSize        int                `json:"writeBufferSize,omitempty"`
	InitialDelay           time.Duration      `json:"initialDelay,omitempty"`
	MaximumDelay           time.Duration      `json:"maximumDelay,omitempty"`
	InitialConnectAttempts int                `json:"initialConnectAttempts,omitempty"`
	AuthUsername           string             `json:"authUsername,omitempty"`
	AuthPassword           string             `json:"authPassword,omitempty"`
	HTTPHeaders            fftypes.JSONObject `json:"headers,omitempty"`
	HeartbeatInterval      time.Duration      `json:"heartbeatInterval,omitempty"`
}

type WSClient interface {
	Connect() error
	Receive() <-chan []byte
	URL() string
	SetURL(url string)
	Send(ctx context.Context, message []byte) error
	Close()
}

type wsClient struct {
	ctx                  context.Context
	headers              http.Header
	url                  string
	initialRetryAttempts int
	wsdialer             *websocket.Dialer
	wsconn               *websocket.Conn
	retry                retry.Retry
	closed               bool
	receive              chan []byte
	send                 chan []byte
	sendDone             chan []byte
	closing              chan struct{}
	afterConnect         WSPostConnectHandler
	heartbeatInterval    time.Duration
	heartbeathMux        sync.Mutex
	activePingSent       *time.Time
	lastPingCompleted    time.Time
}

// WSPostConnectHandler will be called after every connect/reconnect. Can send data over ws, but must not block listening for data on the ws.
type WSPostConnectHandler func(ctx context.Context, w WSClient) error

func New(ctx context.Context, config *WSConfig, afterConnect WSPostConnectHandler) (WSClient, error) {

	wsURL, err := buildWSUrl(ctx, config)
	if err != nil {
		return nil, err
	}

	w := &wsClient{
		ctx: ctx,
		url: wsURL,
		wsdialer: &websocket.Dialer{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
		},
		retry: retry.Retry{
			InitialDelay: config.InitialDelay,
			MaximumDelay: config.MaximumDelay,
		},
		initialRetryAttempts: config.InitialConnectAttempts,
		headers:              make(http.Header),
		receive:              make(chan []byte),
		send:                 make(chan []byte),
		closing:              make(chan struct{}),
		afterConnect:         afterConnect,
		heartbeatInterval:    config.HeartbeatInterval,
	}
	for k, v := range config.HTTPHeaders {
		if vs, ok := v.(string); ok {
			w.headers.Set(k, vs)
		}
	}
	authUsername := config.AuthUsername
	authPassword := config.AuthPassword
	if authUsername != "" && authPassword != "" {
		w.headers.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", authUsername, authPassword)))))
	}

	return w, nil
}

func (w *wsClient) Connect() error {

	if err := w.connect(true); err != nil {
		return err
	}

	go w.receiveReconnectLoop()

	return nil
}

func (w *wsClient) Close() {
	if !w.closed {
		w.closed = true
		close(w.closing)
		c := w.wsconn
		if c != nil {
			_ = c.Close()
		}
	}
}

// Receive returns
func (w *wsClient) Receive() <-chan []byte {
	return w.receive
}

func (w *wsClient) URL() string {
	return w.url
}

func (w *wsClient) SetURL(url string) {
	w.url = url
}

func (w *wsClient) Send(ctx context.Context, message []byte) error {
	// Send
	select {
	case w.send <- message:
		return nil
	case <-ctx.Done():
		return i18n.NewError(ctx, i18n.MsgWSSendTimedOut)
	case <-w.closing:
		return i18n.NewError(ctx, i18n.MsgWSClosing)
	}
}

func (w *wsClient) heartbeatTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if w.heartbeatInterval > 0 {
		w.heartbeathMux.Lock()
		baseTime := w.lastPingCompleted
		if w.activePingSent != nil {
			// We're waiting for a pong
			baseTime = *w.activePingSent
		}
		waitTime := w.heartbeatInterval - time.Since(baseTime) // if negative, will pop immediately
		w.heartbeathMux.Unlock()
		return context.WithTimeout(ctx, waitTime)
	}
	return context.WithCancel(ctx)
}

func buildWSUrl(ctx context.Context, config *WSConfig) (string, error) {
	u, err := url.Parse(config.HTTPURL)
	if err != nil {
		return "", i18n.WrapError(ctx, err, i18n.MsgInvalidURL, config.HTTPURL)
	}
	if config.WSKeyPath != "" {
		u.Path = config.WSKeyPath
	}
	if u.Scheme == "http" {
		u.Scheme = "ws"
	}
	if u.Scheme == "https" {
		u.Scheme = "wss"
	}
	return u.String(), nil
}

func (w *wsClient) connect(initial bool) error {
	l := log.L(w.ctx)
	return w.retry.DoCustomLog(w.ctx, func(attempt int) (retry bool, err error) {
		if w.closed {
			return false, i18n.NewError(w.ctx, i18n.MsgWSClosing)
		}
		var res *http.Response
		w.wsconn, res, err = w.wsdialer.Dial(w.url, w.headers)
		if err != nil {
			var b []byte
			var status = -1
			if res != nil {
				b, _ = ioutil.ReadAll(res.Body)
				res.Body.Close()
				status = res.StatusCode
			}
			l.Warnf("WS %s connect attempt %d failed [%d]: %s", w.url, attempt, status, string(b))
			return !initial || attempt > w.initialRetryAttempts, i18n.WrapError(w.ctx, err, i18n.MsgWSConnectFailed)
		}
		w.pongReceivedOrReset(false)
		w.wsconn.SetPongHandler(w.pongHandler)
		l.Infof("WS %s connected", w.url)
		return false, nil
	})
}

func (w *wsClient) readLoop() {
	l := log.L(w.ctx)
	for {
		mt, message, err := w.wsconn.ReadMessage()
		if err != nil {
			l.Errorf("WS %s closed: %s", w.url, err)
			return
		}

		// Pass the message to the consumer
		l.Tracef("WS %s read (mt=%d): %s", w.url, mt, message)
		select {
		case <-w.sendDone:
			l.Debugf("WS %s closing reader after send error", w.url)
			return
		case w.receive <- message:
		}
	}
}

func (w *wsClient) pongHandler(appData string) error {
	w.pongReceivedOrReset(true)
	return nil
}

func (w *wsClient) pongReceivedOrReset(isPong bool) {
	w.heartbeathMux.Lock()
	defer w.heartbeathMux.Unlock()

	if isPong && w.activePingSent != nil {
		log.L(w.ctx).Debugf("WS %s heartbeat completed (pong) after %.2fms", w.url, float64(time.Since(*w.activePingSent))/float64(time.Millisecond))
	}
	w.lastPingCompleted = time.Now() // in new connection case we still want to consider now the time we completed the ping
	w.activePingSent = nil
}

func (w *wsClient) heartbeatCheck() error {
	w.heartbeathMux.Lock()
	defer w.heartbeathMux.Unlock()

	if w.activePingSent != nil {
		return i18n.NewError(w.ctx, i18n.MsgWSHeartbeatTimeout, float64(time.Since(*w.activePingSent))/float64(time.Millisecond))
	}
	log.L(w.ctx).Debugf("WS %s heartbeat timer popped (ping) after %.2fms", w.url, float64(time.Since(w.lastPingCompleted))/float64(time.Millisecond))
	now := time.Now()
	w.activePingSent = &now
	return nil
}

func (w *wsClient) sendLoop(receiverDone chan struct{}) {
	l := log.L(w.ctx)
	defer close(w.sendDone)

	disconnecting := false
	for !disconnecting {
		timeoutContext, timeoutCancel := w.heartbeatTimeout(w.ctx)

		select {
		case message := <-w.send:
			l.Tracef("WS sending: %s", message)
			if err := w.wsconn.WriteMessage(websocket.TextMessage, message); err != nil {
				l.Errorf("WS %s send failed: %s", w.url, err)
				disconnecting = true
			}
		case <-timeoutContext.Done():
			if err := w.heartbeatCheck(); err != nil {
				l.Errorf("WS %s closing: %s", w.url, err)
				disconnecting = true
			} else if err := w.wsconn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				l.Errorf("WS %s heartbeat send failed: %s", w.url, err)
				disconnecting = true
			}
		case <-receiverDone:
			l.Debugf("WS %s send loop exiting", w.url)
			disconnecting = true
		}

		timeoutCancel()
	}
}

func (w *wsClient) receiveReconnectLoop() {
	l := log.L(w.ctx)
	defer close(w.receive)
	for !w.closed {
		// Start the sender, letting it close without blocking sending a notifiation on the sendDone
		w.sendDone = make(chan []byte, 1)
		receiverDone := make(chan struct{})
		go w.sendLoop(receiverDone)

		// Call the reconnect processor
		var err error
		if w.afterConnect != nil {
			err = w.afterConnect(w.ctx, w)
		}

		if err == nil {
			// Synchronously invoke the reader, as it's important we react immediately to any error there.
			w.readLoop()
			close(receiverDone)

			// Ensure the connection is closed after the receiver exits
			err = w.wsconn.Close()
			if err != nil {
				l.Debugf("WS %s close failed: %s", w.url, err)
			}
			<-w.sendDone
			w.sendDone = nil
			w.wsconn = nil
		}

		// Go into reconnect
		if !w.closed {
			err = w.connect(false)
			if err != nil {
				l.Debugf("WS %s exiting: %s", w.url, err)
				return
			}
		}
	}
}
