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
	capabilities *tokens.Capabilities
	callbacks    tokens.Callbacks
	client       *resty.Client
	wsconn       wsclient.WSClient
	closed       chan struct{}
}

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
	return "https"
}

func (h *HTTPS) Init(ctx context.Context, prefix config.Prefix, callbacks tokens.Callbacks) (err error) {

	h.ctx = log.WithLogField(ctx, "proto", "https")
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

			var msgParsed interface{}
			err := json.Unmarshal(msgBytes, &msgParsed)
			if err != nil {
				l.Errorf("Message cannot be parsed as JSON: %s\n%s", err, string(msgBytes))
				continue // Swallow this and move on
			}
			l.Infof("Received an event: %s", string(msgBytes))
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
