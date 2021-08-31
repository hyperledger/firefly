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
	"encoding/hex"
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
	ID        string            `json:"clientId"`
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
	clientID := data.GetString("clientId")
	tokenType := data.GetString("type")
	protocolID := data.GetString("poolId")
	operatorAddress := data.GetString("operator")
	tx := data.GetObject("transaction")
	txHash := tx.GetString("transactionHash")

	if ns == "" ||
		name == "" ||
		clientID == "" ||
		tokenType == "" ||
		protocolID == "" ||
		operatorAddress == "" ||
		txHash == "" {
		log.L(ctx).Errorf("TokenPool event is not valid - missing data: %+v", data)
		return nil // move on
	}

	hexUUIDs, err := hex.DecodeString(clientID)
	if err != nil || len(hexUUIDs) != 32 {
		log.L(ctx).Errorf("TokenPool event is not valid - bad uuids (%s): %+v", err, data)
		return nil // move on
	}
	var txnID fftypes.UUID
	copy(txnID[:], hexUUIDs[0:16])
	var id fftypes.UUID
	copy(id[:], hexUUIDs[16:32])

	pool := &fftypes.TokenPool{
		ID: &id,
		TX: fftypes.TransactionRef{
			ID:   &txnID,
			Type: fftypes.TransactionTypeTokenPool,
		},
		Namespace:  ns,
		Name:       name,
		Type:       fftypes.LowerCasedType(tokenType),
		ProtocolID: protocolID,
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return h.callbacks.TokenPoolCreated(h, pool, operatorAddress, txHash, tx)
}

func (h *HTTPS) eventLoop() {
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

func (h *HTTPS) CreateTokenPool(ctx context.Context, identity *fftypes.Identity, pool *fftypes.TokenPool) (txTrackingID string, err error) {
	var uuids fftypes.Bytes32
	copy(uuids[0:16], (*pool.TX.ID)[:])
	copy(uuids[16:32], (*pool.ID)[:])

	var response responseData
	res, err := h.client.R().SetContext(ctx).
		SetBody(&createPool{
			ID:        hex.EncodeToString(uuids[0:32]),
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
