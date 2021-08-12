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

package https

import (
	"context"
	"encoding/json"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/restclient"
	"github.com/hyperledger-labs/firefly/internal/wsclient"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/tokens"
)

type HTTPS struct {
	ctx          context.Context
	name         string
	capabilities *tokens.Capabilities
	callbacks    tokens.Callbacks
	client       *resty.Client
	wsconn       wsclient.WSClient
	closed       chan struct{}
}

type wsEvent struct {
	Event msgType            `json:"event"`
	ID    string             `json:"id"`
	Data  fftypes.JSONObject `json:"data"`
}

type msgType string

const (
	messageReceipt   msgType = "receipt"
	messageTokenPool msgType = "token-pool"
)

type responseData struct {
	RequestID string `json:"id"`
}

type createPool struct {
	ID        *fftypes.UUID     `json:"client_id"`
	Type      fftypes.TokenType `json:"type"`
	Namespace string            `json:"namespace"`
	Name      string            `json:"name"`
}

func (h *HTTPS) Name() string {
	return h.name
}

func (h *HTTPS) ConnectorName() string {
	return "https"
}

func (h *HTTPS) Init(ctx context.Context, name string, prefix config.Prefix, callbacks tokens.Callbacks) (err error) {

	h.ctx = log.WithLogField(ctx, "proto", "https")
	h.name = name
	h.callbacks = callbacks

	if prefix.GetString(restclient.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "url", "tokens.https")
	}

	h.client = restclient.New(h.ctx, prefix)
	h.capabilities = &tokens.Capabilities{}

	if prefix.GetString(wsclient.WSConfigKeyPath) == "" {
		prefix.Set(wsclient.WSConfigKeyPath, "/api/ws")
	}
	h.wsconn, err = wsclient.New(ctx, prefix, nil)
	if err != nil {
		return err
	}

	h.closed = make(chan struct{})
	go h.eventLoop()

	return nil
}

func (h *HTTPS) Start() error {
	return h.wsconn.Connect()
}

func (h *HTTPS) Capabilities() *tokens.Capabilities {
	return h.capabilities
}

func (h *HTTPS) handleReceipt(ctx context.Context, data fftypes.JSONObject) error {
	l := log.L(ctx)

	requestID := data.GetString("id")
	success := data.GetBool("success")
	message := data.GetString("message")
	if requestID == "" {
		l.Errorf("Reply cannot be processed: %+v", data)
		return nil // Swallow this and move on
	}
	updateType := fftypes.OpStatusSucceeded
	if !success {
		updateType = fftypes.OpStatusFailed
	}
	l.Infof("Tokens '%s' reply (request=%s) %s", updateType, requestID, message)
	return h.callbacks.TokensTxUpdate(h, requestID, updateType, message, data)
}

func (h *HTTPS) handleTokenPoolCreate(ctx context.Context, data fftypes.JSONObject) (err error) {
	ns := data.GetString("namespace")
	name := data.GetString("name")
	clientID := data.GetString("client_id")
	tokenType := data.GetString("type")
	poolID := data.GetString("pool_id")
	authorAddress := data.GetString("author")

	if ns == "" ||
		name == "" ||
		clientID == "" ||
		tokenType == "" ||
		poolID == "" ||
		authorAddress == "" {
		log.L(ctx).Errorf("TokenPool event is not valid - missing data: %+v", data)
		return nil // move on
	}

	uuid, err := fftypes.ParseUUID(ctx, clientID)
	if err != nil {
		log.L(ctx).Errorf("TokenPool event is not valid - bad uuid (%s): %+v", err, data)
		return nil // move on
	}

	pool := &fftypes.TokenPool{
		ID:        uuid,
		Namespace: ns,
		Name:      name,
		Type:      fftypes.LowerCasedType(tokenType),
		PoolID:    poolID,
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return h.callbacks.TokenPoolCreated(h, pool, authorAddress, data)
}

func (h *HTTPS) eventLoop() {
	defer close(h.closed)
	l := log.L(h.ctx).WithField("role", "event-loop")
	ctx := log.WithLogger(h.ctx, l)
	for {
		select {
		case <-ctx.Done():
			l.Debugf("Event loop exiting (context cancelled)")
			return
		case msgBytes, ok := <-h.wsconn.Receive():
			if !ok {
				l.Debugf("Event loop exiting (receive channel closed)")
				return
			}

			var msg wsEvent
			err := json.Unmarshal(msgBytes, &msg)
			if err != nil {
				l.Errorf("Message cannot be parsed as JSON: %s\n%s", err, string(msgBytes))
				continue // Swallow this and move on
			}
			l.Debugf("Received %s event %s", msg.Event, msg.ID)
			switch msg.Event {
			case messageReceipt:
				err = h.handleReceipt(ctx, msg.Data)
			case messageTokenPool:
				err = h.handleTokenPoolCreate(ctx, msg.Data)
			default:
				l.Errorf("Message unexpected: %s", msg.Event)
			}

			if err == nil {
				l.Debugf("Sending ack %s", msg.ID)
				ack, _ := json.Marshal(fftypes.JSONObject{
					"event": "ack",
					"data": fftypes.JSONObject{
						"id": msg.ID,
					},
				})
				err = h.wsconn.Send(ctx, ack)
			}

			if err != nil {
				l.Errorf("Event loop exiting: %s", err)
				return
			}
		}
	}
}

func (h *HTTPS) CreateTokenPool(ctx context.Context, identity *fftypes.Identity, pool *fftypes.TokenPool) (txTrackingID string, err error) {
	var response responseData
	res, err := h.client.R().SetContext(ctx).
		SetBody(&createPool{
			ID:        pool.ID,
			Type:      pool.Type,
			Namespace: pool.Namespace,
			Name:      pool.Name,
		}).
		SetResult(&response).
		Post("/api/v1/pool")
	if err != nil || !res.IsSuccess() {
		return "", restclient.WrapRestErr(ctx, res, err, i18n.MsgTokensRESTErr)
	}
	return response.RequestID, nil
}
