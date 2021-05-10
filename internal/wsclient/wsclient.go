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

package wsclient

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/retry"
)

const (
	defaultRetryWaitTimeMillis    = 100
	defaultRetryMaxWaitTimeMillis = 1000
	defaultIntialConnectAttempts  = 5
	defaultBufferSizeKB           = 1024
)

type WSConfig struct {
	URL               string            `json:"url"`
	Headers           map[string]string `json:"headers,omitempty"`
	Auth              *WSAuthConfig     `json:"auth,omitempty"`
	WriteBufferSizeKB *uint             `json:"writeBufferSizeKB"`
	ReadBufferSizeKB  *uint             `json:"readBufferSizeKB"`
	WSRetryConfig
}

type WSRetryConfig struct {
	InitialConnectAttempts *uint `json:"intialConnectAttempts,omitempty"`
	WaitTimeMS             *uint `json:"waitTimeMS,omitempty"`
	MaxWaitTimeMS          *uint `json:"maxWaitTimeMS,omitempty"`
}

type WSAuthConfig struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type WSClient struct {
	ctx                  context.Context
	headers              http.Header
	url                  string
	initialRetryAttempts int
	wsdialer             *websocket.Dialer
	wsconn               *websocket.Conn
	retry                *retry.Retry
	closed               bool
	receive              chan []byte
	send                 chan []byte
	sendDone             chan []byte
	closing              chan struct{}
	sendOnConnect        [][]byte
}

func NewWSClient(ctx context.Context, conf *WSConfig, sendOnConnect ...[]byte) (*WSClient, error) {

	w := &WSClient{
		ctx: ctx,
		url: conf.URL,
		wsdialer: &websocket.Dialer{
			ReadBufferSize:  int(config.UintWithDefault(conf.WriteBufferSizeKB, defaultBufferSizeKB) * 1024),
			WriteBufferSize: int(config.UintWithDefault(conf.ReadBufferSizeKB, defaultBufferSizeKB) * 1024),
		},
		retry: &retry.Retry{
			InitialDelay: time.Duration(config.UintWithDefault(conf.WSRetryConfig.WaitTimeMS, defaultRetryWaitTimeMillis)) * time.Millisecond,
			MaximumDelay: time.Duration(config.UintWithDefault(conf.WSRetryConfig.MaxWaitTimeMS, defaultRetryMaxWaitTimeMillis)) * time.Millisecond,
		},
		initialRetryAttempts: int(config.UintWithDefault(conf.InitialConnectAttempts, defaultIntialConnectAttempts)),
		headers:              make(http.Header),
		receive:              make(chan []byte),
		send:                 make(chan []byte),
		closing:              make(chan struct{}),
		sendOnConnect:        sendOnConnect,
	}
	for k, v := range conf.Headers {
		w.headers.Set(k, v)
	}
	if conf.Auth != nil && conf.Auth.Username != "" && conf.Auth.Password != "" {
		w.headers.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", conf.Auth.Username, conf.Auth.Password)))))
	}

	if err := w.connect(true); err != nil {
		return nil, err
	}

	go w.receiveReconnectLoop()

	return w, nil
}

func (w *WSClient) Close() {
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
func (w *WSClient) Receive() <-chan []byte {
	return w.receive
}

func (w *WSClient) Send(ctx context.Context, message []byte) error {
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

func (w *WSClient) connect(initial bool) error {
	return w.retry.Do(w.ctx, func(attempt int) (retry bool, err error) {
		l := log.L(w.ctx)
		if w.closed {
			return false, i18n.NewError(w.ctx, i18n.MsgWSClosing)
		}
		var res *http.Response
		w.wsconn, res, err = w.wsdialer.Dial(w.url, w.headers)
		for i := 0; err == nil && i < len(w.sendOnConnect); i++ {
			err = w.wsconn.WriteMessage(websocket.TextMessage, w.sendOnConnect[i])
		}
		if err != nil {
			var b []byte
			var status = -1
			if res != nil {
				b, _ = ioutil.ReadAll(res.Body)
				status = res.StatusCode
			}
			l.Warnf("WS %s connect attempt %d failed [%d]: %s", w.url, attempt, status, string(b))
			return !initial || attempt > w.initialRetryAttempts, i18n.WrapError(w.ctx, err, i18n.MsgWSConnectFailed)
		}
		l.Infof("WS %s connected", w.url)
		return false, nil
	})
}

func (w *WSClient) readLoop() []byte {
	l := log.L(w.ctx)
	for {
		mt, message, err := w.wsconn.ReadMessage()

		// Check there's not a pending send message we need to return
		// before returning any error (do not block)
		select {
		case pendingMsg := <-w.sendDone:
			l.Debugf("WS %s closing reader after send error", w.url)
			return pendingMsg
		default:
		}

		// return any error
		if err != nil {
			l.Errorf("WS %s closed: %s", w.url, err)
			return nil
		}

		// Pass the message to the consumer
		l.Tracef("WS %s read (mt=%d): %s", w.url, mt, message)
		w.receive <- message
	}
}

func (w *WSClient) sendLoop(message []byte) {
	l := log.L(w.ctx)
	defer close(w.sendDone)
	for {
		if message != nil {
			if err := w.wsconn.WriteMessage(websocket.TextMessage, message); err != nil {
				l.Errorf("WS %s send failed: %s", w.url, err)
				// Keep the message for when we reconnect
				w.sendDone <- message
				return
			}
		}

		var ok bool
		message, ok = <-w.send
		if !ok {
			l.Debugf("WS %s send loop exiting", w.url)
			return
		}

	}
}

func (w *WSClient) receiveReconnectLoop() {
	l := log.L(w.ctx)
	defer close(w.receive)
	var pendingSend []byte
	for !w.closed {
		// Start the sender, letting it close without blocking sending a notifiation on the sendDone
		w.sendDone = make(chan []byte, 1)
		go w.sendLoop(pendingSend)

		// Synchronously invoke the reader, as it's important we react immediately
		// to any error there.
		pendingSend = w.readLoop()

		// Ensure the connection is closed after the sender exits
		err := w.wsconn.Close()
		if err != nil {
			l.Warnf("WS %s close failed: %s", w.url, err)
		}
		<-w.sendDone
		w.sendDone = nil
		w.wsconn = nil

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
