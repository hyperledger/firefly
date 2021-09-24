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

package fftokens

import (
	"context"
	"encoding/json"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/restclient"
	"github.com/hyperledger/firefly/internal/wsclient"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

type FFTokens struct {
	ctx            context.Context
	capabilities   *tokens.Capabilities
	callbacks      tokens.Callbacks
	configuredName string
	client         *resty.Client
	wsconn         wsclient.WSClient
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

type createPool struct {
	Type      fftypes.TokenType `json:"type"`
	RequestID string            `json:"requestId"`
	Data      string            `json:"data"`
}

type createPoolData struct {
	Namespace     string        `json:"namespace"`
	Name          string        `json:"name"`
	ID            *fftypes.UUID `json:"id"`
	TransactionID *fftypes.UUID `json:"transactionId"`
}

func (h *FFTokens) Name() string {
	return "fftokens"
}

func (h *FFTokens) Init(ctx context.Context, name string, prefix config.Prefix, callbacks tokens.Callbacks) (err error) {
	h.ctx = log.WithLogField(ctx, "proto", "fftokens")
	h.callbacks = callbacks
	h.configuredName = name

	if prefix.GetString(restclient.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "url", "tokens.fftokens")
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

	go h.eventLoop()

	return nil
}

func (h *FFTokens) Start() error {
	return h.wsconn.Connect()
}

func (h *FFTokens) Capabilities() *tokens.Capabilities {
	return h.capabilities
}

func (h *FFTokens) handleReceipt(ctx context.Context, data fftypes.JSONObject) error {
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

func (h *FFTokens) handleTokenPoolCreate(ctx context.Context, data fftypes.JSONObject) (err error) {
	packedData := data.GetString("data")
	tokenType := data.GetString("type")
	protocolID := data.GetString("poolId")
	operatorAddress := data.GetString("operator")
	tx := data.GetObject("transaction")
	txHash := tx.GetString("transactionHash")

	if packedData == "" ||
		tokenType == "" ||
		protocolID == "" ||
		operatorAddress == "" ||
		txHash == "" {
		log.L(ctx).Errorf("TokenPool event is not valid - missing data: %+v", data)
		return nil // move on
	}

	unpackedData := createPoolData{}
	err = json.Unmarshal([]byte(packedData), &unpackedData)
	if err != nil {
		log.L(ctx).Errorf("TokenPool event is not valid - could not unpack data (%s): %+v", err, data)
		return nil // move on
	}
	if unpackedData.Namespace == "" ||
		unpackedData.Name == "" ||
		unpackedData.ID == nil ||
		unpackedData.TransactionID == nil {
		log.L(ctx).Errorf("TokenPool event is not valid - missing packed data: %+v", unpackedData)
		return nil // move on
	}

	pool := &fftypes.TokenPool{
		ID: unpackedData.ID,
		TX: fftypes.TransactionRef{
			ID:   unpackedData.TransactionID,
			Type: fftypes.TransactionTypeTokenPool,
		},
		Namespace:  unpackedData.Namespace,
		Name:       unpackedData.Name,
		Type:       fftypes.FFEnum(tokenType),
		Connector:  h.configuredName,
		ProtocolID: protocolID,
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return h.callbacks.TokenPoolCreated(h, pool, operatorAddress, txHash, tx)
}

func (h *FFTokens) eventLoop() {
	defer h.wsconn.Close()
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

			if err == nil && msg.Event != messageReceipt && msg.ID != "" {
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

func (h *FFTokens) CreateTokenPool(ctx context.Context, identity *fftypes.Identity, pool *fftypes.TokenPool) error {
	data := createPoolData{
		Namespace:     pool.Namespace,
		Name:          pool.Name,
		ID:            pool.ID,
		TransactionID: pool.TX.ID,
	}
	packedData, err := json.Marshal(data)
	var res *resty.Response
	if err == nil {
		res, err = h.client.R().SetContext(ctx).
			SetBody(&createPool{
				Type:      pool.Type,
				RequestID: pool.TX.ID.String(),
				Data:      string(packedData),
			}).
			Post("/api/v1/pool")
	}
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgTokensRESTErr)
	}
	return nil
}
