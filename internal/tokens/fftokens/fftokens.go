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
	"github.com/hyperledger/firefly/internal/config/wsconfig"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/restclient"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/hyperledger/firefly/pkg/wsclient"
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
	messageReceipt       msgType = "receipt"
	messageTokenPool     msgType = "token-pool"
	messageTokenMint     msgType = "token-mint"
	messageTokenBurn     msgType = "token-burn"
	messageTokenTransfer msgType = "token-transfer"
)

type createPool struct {
	Type       fftypes.TokenType  `json:"type"`
	RequestID  string             `json:"requestId"`
	TrackingID string             `json:"trackingId"`
	Config     fftypes.JSONObject `json:"config"`
}

type mintTokens struct {
	PoolID     string `json:"poolId"`
	To         string `json:"to"`
	Amount     string `json:"amount"`
	RequestID  string `json:"requestId,omitempty"`
	TrackingID string `json:"trackingId"`
}

type burnTokens struct {
	PoolID     string `json:"poolId"`
	TokenIndex string `json:"tokenIndex,omitempty"`
	From       string `json:"from"`
	Amount     string `json:"amount"`
	RequestID  string `json:"requestId,omitempty"`
	TrackingID string `json:"trackingId"`
}

type transferTokens struct {
	PoolID     string `json:"poolId"`
	TokenIndex string `json:"tokenIndex,omitempty"`
	From       string `json:"from"`
	To         string `json:"to"`
	Amount     string `json:"amount"`
	RequestID  string `json:"requestId,omitempty"`
	TrackingID string `json:"trackingId"`
	Data       string `json:"data,omitempty"`
}

func (ft *FFTokens) Name() string {
	return "fftokens"
}

func (ft *FFTokens) Init(ctx context.Context, name string, prefix config.Prefix, callbacks tokens.Callbacks) (err error) {
	ft.ctx = log.WithLogField(ctx, "proto", "fftokens")
	ft.callbacks = callbacks
	ft.configuredName = name

	if prefix.GetString(restclient.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "url", "tokens.fftokens")
	}

	ft.client = restclient.New(ft.ctx, prefix)
	ft.capabilities = &tokens.Capabilities{}

	wsConfig := wsconfig.GenerateConfigFromPrefix(prefix)

	if wsConfig.WSKeyPath == "" {
		wsConfig.WSKeyPath = "/api/ws"
	}

	ft.wsconn, err = wsclient.New(ctx, wsConfig, nil)
	if err != nil {
		return err
	}

	go ft.eventLoop()

	return nil
}

func (ft *FFTokens) Start() error {
	return ft.wsconn.Connect()
}

func (ft *FFTokens) Capabilities() *tokens.Capabilities {
	return ft.capabilities
}

func (ft *FFTokens) handleReceipt(ctx context.Context, data fftypes.JSONObject) error {
	l := log.L(ctx)

	requestID := data.GetString("id")
	success := data.GetBool("success")
	message := data.GetString("message")
	if requestID == "" {
		l.Errorf("Reply cannot be processed - missing fields: %+v", data)
		return nil // Swallow this and move on
	}
	operationID, err := fftypes.ParseUUID(ctx, requestID)
	if err != nil {
		l.Errorf("Reply cannot be processed - bad ID: %+v", data)
		return nil // Swallow this and move on
	}
	replyType := fftypes.OpStatusSucceeded
	if !success {
		replyType = fftypes.OpStatusFailed
	}
	l.Infof("Tokens '%s' reply: request=%s message=%s", replyType, requestID, message)
	return ft.callbacks.TokenOpUpdate(ft, operationID, replyType, message, data)
}

func (ft *FFTokens) handleTokenPoolCreate(ctx context.Context, data fftypes.JSONObject) (err error) {
	tokenType := data.GetString("type")
	protocolID := data.GetString("poolId")
	standard := data.GetString("standard") // this is optional
	trackingID := data.GetString("trackingId")
	operatorAddress := data.GetString("operator")
	tx := data.GetObject("transaction")
	txHash := tx.GetString("transactionHash")

	if tokenType == "" ||
		protocolID == "" ||
		trackingID == "" ||
		operatorAddress == "" ||
		txHash == "" {
		log.L(ctx).Errorf("TokenPool event is not valid - missing data: %+v", data)
		return nil // move on
	}

	txID, err := fftypes.ParseUUID(ctx, trackingID)
	if err != nil {
		log.L(ctx).Errorf("TokenPool event is not valid - invalid transaction ID (%s): %+v", err, data)
		return nil // move on
	}

	pool := &fftypes.TokenPool{
		Type:       fftypes.FFEnum(tokenType),
		ProtocolID: protocolID,
		Standard:   standard,
		Connector:  ft.configuredName,
		Key:        operatorAddress,
		TX: fftypes.TransactionRef{
			ID:   txID,
			Type: fftypes.TransactionTypeTokenPool,
		},
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return ft.callbacks.TokenPoolCreated(ft, pool, txHash, tx)
}

func (ft *FFTokens) handleTokenTransfer(ctx context.Context, t fftypes.TokenTransferType, data fftypes.JSONObject) (err error) {
	tokenIndex := data.GetString("tokenIndex")
	poolProtocolID := data.GetString("poolId")
	operatorAddress := data.GetString("operator")
	fromAddress := data.GetString("from")
	toAddress := data.GetString("to")
	value := data.GetString("amount")
	tx := data.GetObject("transaction")
	txHash := tx.GetString("transactionHash")

	var eventName string
	switch t {
	case fftypes.TokenTransferTypeMint:
		eventName = "Mint"
	case fftypes.TokenTransferTypeBurn:
		eventName = "Burn"
	default:
		eventName = "Transfer"
	}

	if poolProtocolID == "" ||
		operatorAddress == "" ||
		value == "" ||
		txHash == "" ||
		(t != fftypes.TokenTransferTypeMint && fromAddress == "") ||
		(t != fftypes.TokenTransferTypeBurn && toAddress == "") {
		log.L(ctx).Errorf("%s event is not valid - missing data: %+v", eventName, data)
		return nil // move on
	}

	// We want to process all transfers, even those not initiated by FireFly.
	// The following are optional arguments from the connector, so it's important not to
	// fail if they're missing or malformed.
	trackingID := data.GetString("trackingId")
	txID, err := fftypes.ParseUUID(ctx, trackingID)
	if err != nil {
		log.L(ctx).Infof("%s event contains invalid ID - continuing anyway (%s): %+v", eventName, err, data)
		txID = fftypes.NewUUID()
	}
	transferData := data.GetString("data")
	var messageHash fftypes.Bytes32
	if transferData != "" {
		err = messageHash.UnmarshalText([]byte(transferData))
		if err != nil {
			log.L(ctx).Errorf("%s event contains invalid message hash - continuing anyway (%s): %+v", eventName, err, data)
		}
	}

	transfer := &fftypes.TokenTransfer{
		Type:           t,
		PoolProtocolID: poolProtocolID,
		TokenIndex:     tokenIndex,
		Connector:      ft.configuredName,
		From:           fromAddress,
		To:             toAddress,
		ProtocolID:     txHash,
		Key:            operatorAddress,
		MessageHash:    &messageHash,
		TX: fftypes.TransactionRef{
			ID:   txID,
			Type: fftypes.TransactionTypeTokenTransfer,
		},
	}

	_, ok := transfer.Amount.Int().SetString(value, 10)
	if !ok {
		log.L(ctx).Errorf("%s event is not valid - invalid amount: %+v", eventName, data)
		return nil // move on
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return ft.callbacks.TokensTransferred(ft, transfer, txHash, tx)
}

func (ft *FFTokens) eventLoop() {
	defer ft.wsconn.Close()
	l := log.L(ft.ctx).WithField("role", "event-loop")
	ctx := log.WithLogger(ft.ctx, l)
	for {
		select {
		case <-ctx.Done():
			l.Debugf("Event loop exiting (context cancelled)")
			return
		case msgBytes, ok := <-ft.wsconn.Receive():
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
				err = ft.handleReceipt(ctx, msg.Data)
			case messageTokenPool:
				err = ft.handleTokenPoolCreate(ctx, msg.Data)
			case messageTokenMint:
				err = ft.handleTokenTransfer(ctx, fftypes.TokenTransferTypeMint, msg.Data)
			case messageTokenBurn:
				err = ft.handleTokenTransfer(ctx, fftypes.TokenTransferTypeBurn, msg.Data)
			case messageTokenTransfer:
				err = ft.handleTokenTransfer(ctx, fftypes.TokenTransferTypeTransfer, msg.Data)
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
				err = ft.wsconn.Send(ctx, ack)
			}

			if err != nil {
				l.Errorf("Event loop exiting: %s", err)
				return
			}
		}
	}
}

func (ft *FFTokens) CreateTokenPool(ctx context.Context, operationID *fftypes.UUID, pool *fftypes.TokenPool) error {
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&createPool{
			Type:       pool.Type,
			RequestID:  operationID.String(),
			TrackingID: pool.TX.ID.String(),
			Config:     pool.Config,
		}).
		Post("/api/v1/pool")
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgTokensRESTErr)
	}
	return nil
}

func (ft *FFTokens) MintTokens(ctx context.Context, operationID *fftypes.UUID, mint *fftypes.TokenTransfer) error {
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&mintTokens{
			PoolID:     mint.PoolProtocolID,
			To:         mint.To,
			Amount:     mint.Amount.Int().String(),
			RequestID:  operationID.String(),
			TrackingID: mint.TX.ID.String(),
		}).
		Post("/api/v1/mint")
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgTokensRESTErr)
	}
	return nil
}

func (ft *FFTokens) BurnTokens(ctx context.Context, operationID *fftypes.UUID, burn *fftypes.TokenTransfer) error {
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&burnTokens{
			PoolID:     burn.PoolProtocolID,
			TokenIndex: burn.TokenIndex,
			From:       burn.From,
			Amount:     burn.Amount.Int().String(),
			RequestID:  operationID.String(),
			TrackingID: burn.TX.ID.String(),
		}).
		Post("/api/v1/burn")
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgTokensRESTErr)
	}
	return nil
}

func (ft *FFTokens) TransferTokens(ctx context.Context, operationID *fftypes.UUID, transfer *fftypes.TokenTransfer) error {
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&transferTokens{
			PoolID:     transfer.PoolProtocolID,
			TokenIndex: transfer.TokenIndex,
			From:       transfer.From,
			To:         transfer.To,
			Amount:     transfer.Amount.Int().String(),
			RequestID:  operationID.String(),
			TrackingID: transfer.TX.ID.String(),
			Data:       transfer.MessageHash.String(),
		}).
		Post("/api/v1/transfer")
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgTokensRESTErr)
	}
	return nil
}
