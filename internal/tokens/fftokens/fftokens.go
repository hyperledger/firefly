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
	"github.com/hyperledger/firefly/pkg/blockchain"
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

type tokenData struct {
	TX          *fftypes.UUID    `json:"tx,omitempty"`
	Message     *fftypes.UUID    `json:"message,omitempty"`
	MessageHash *fftypes.Bytes32 `json:"messageHash,omitempty"`
}

type createPool struct {
	Type      fftypes.TokenType  `json:"type"`
	RequestID string             `json:"requestId"`
	Operator  string             `json:"operator"`
	Data      string             `json:"data,omitempty"`
	Config    fftypes.JSONObject `json:"config"`
	Name      string             `json:"name"`
	Symbol    string             `json:"symbol"`
}

type activatePool struct {
	PoolID      string             `json:"poolId"`
	Transaction fftypes.JSONObject `json:"transaction"`
	RequestID   string             `json:"requestId,omitempty"`
}

type mintTokens struct {
	PoolID    string `json:"poolId"`
	To        string `json:"to"`
	Amount    string `json:"amount"`
	RequestID string `json:"requestId,omitempty"`
	Operator  string `json:"operator"`
	Data      string `json:"data,omitempty"`
}

type burnTokens struct {
	PoolID     string `json:"poolId"`
	TokenIndex string `json:"tokenIndex,omitempty"`
	From       string `json:"from"`
	Amount     string `json:"amount"`
	RequestID  string `json:"requestId,omitempty"`
	Operator   string `json:"operator"`
	Data       string `json:"data,omitempty"`
}

type transferTokens struct {
	PoolID     string `json:"poolId"`
	TokenIndex string `json:"tokenIndex,omitempty"`
	From       string `json:"from"`
	To         string `json:"to"`
	Amount     string `json:"amount"`
	RequestID  string `json:"requestId,omitempty"`
	Operator   string `json:"operator"`
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
	transactionHash := data.GetString("transactionHash")
	if requestID == "" {
		l.Errorf("Reply cannot be processed - missing fields: %+v", data)
		return nil // Swallow this and move on
	}
	opID, err := fftypes.ParseUUID(ctx, requestID)
	if err != nil {
		l.Errorf("Reply cannot be processed - bad ID: %+v", data)
		return nil // Swallow this and move on
	}
	replyType := fftypes.OpStatusSucceeded
	if !success {
		replyType = fftypes.OpStatusFailed
	}
	l.Infof("Tokens '%s' reply: request=%s message=%s", replyType, requestID, message)
	return ft.callbacks.TokenOpUpdate(ft, opID, replyType, transactionHash, message, data)
}

func (ft *FFTokens) handleTokenPoolCreate(ctx context.Context, data fftypes.JSONObject) (err error) {
	eventProtocolID := data.GetString("id")
	tokenType := data.GetString("type")
	protocolID := data.GetString("poolId")
	standard := data.GetString("standard") // this is optional
	operatorAddress := data.GetString("operator")
	tx := data.GetObject("transaction")
	txHash := tx.GetString("transactionHash")
	rawOutput := data.GetObject("rawOutput") // optional

	timestampStr := data.GetString("timestamp")
	timestamp, err := fftypes.ParseTimeString(timestampStr)
	if err != nil {
		timestamp = fftypes.Now()
	}

	if tokenType == "" ||
		protocolID == "" ||
		operatorAddress == "" ||
		txHash == "" {
		log.L(ctx).Errorf("TokenPool event is not valid - missing data: %+v", data)
		return nil // move on
	}

	// We want to process all events, even those not initiated by FireFly.
	// The "data" argument is optional, so it's important not to fail if it's missing or malformed.
	poolDataString := data.GetString("data")
	var poolData tokenData
	if err = json.Unmarshal([]byte(poolDataString), &poolData); err != nil {
		log.L(ctx).Infof("TokenPool event data could not be parsed - continuing anyway (%s): %+v", err, data)
		poolData = tokenData{}
	}

	pool := &tokens.TokenPool{
		Type:          fftypes.FFEnum(tokenType),
		ProtocolID:    protocolID,
		TransactionID: poolData.TX,
		Key:           operatorAddress,
		Connector:     ft.configuredName,
		Standard:      standard,
		Event: blockchain.Event{
			BlockchainTXID: txHash,
			Source:         ft.Name() + ":" + ft.configuredName,
			Name:           "TokenPool",
			ProtocolID:     eventProtocolID,
			Output:         rawOutput,
			Info:           tx,
			Timestamp:      timestamp,
		},
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return ft.callbacks.TokenPoolCreated(ft, pool)
}

func (ft *FFTokens) handleTokenTransfer(ctx context.Context, t fftypes.TokenTransferType, data fftypes.JSONObject) (err error) {
	eventProtocolID := data.GetString("id")
	poolProtocolID := data.GetString("poolId")
	operatorAddress := data.GetString("operator")
	fromAddress := data.GetString("from")
	toAddress := data.GetString("to")
	value := data.GetString("amount")
	tx := data.GetObject("transaction")
	txHash := tx.GetString("transactionHash")
	tokenIndex := data.GetString("tokenIndex") // optional
	uri := data.GetString("uri")               // optional
	rawOutput := data.GetObject("rawOutput")   // optional

	timestampStr := data.GetString("timestamp")
	timestamp, err := fftypes.ParseTimeString(timestampStr)
	if err != nil {
		timestamp = fftypes.Now()
	}

	var eventName string
	switch t {
	case fftypes.TokenTransferTypeMint:
		eventName = "Mint"
	case fftypes.TokenTransferTypeBurn:
		eventName = "Burn"
	default:
		eventName = "Transfer"
	}

	if eventProtocolID == "" ||
		poolProtocolID == "" ||
		operatorAddress == "" ||
		value == "" ||
		txHash == "" ||
		(t != fftypes.TokenTransferTypeMint && fromAddress == "") ||
		(t != fftypes.TokenTransferTypeBurn && toAddress == "") {
		log.L(ctx).Errorf("%s event is not valid - missing data: %+v", eventName, data)
		return nil // move on
	}

	// We want to process all events, even those not initiated by FireFly.
	// The "data" argument is optional, so it's important not to fail if it's missing or malformed.
	transferDataString := data.GetString("data")
	var transferData tokenData
	if err = json.Unmarshal([]byte(transferDataString), &transferData); err != nil {
		log.L(ctx).Infof("%s event data could not be parsed - continuing anyway (%s): %+v", eventName, err, data)
		transferData = tokenData{}
	}

	var amount fftypes.FFBigInt
	_, ok := amount.Int().SetString(value, 10)
	if !ok {
		log.L(ctx).Errorf("%s event is not valid - invalid amount: %+v", eventName, data)
		return nil // move on
	}

	transfer := &tokens.TokenTransfer{
		PoolProtocolID: poolProtocolID,
		TokenTransfer: fftypes.TokenTransfer{
			Type:        t,
			TokenIndex:  tokenIndex,
			URI:         uri,
			Connector:   ft.configuredName,
			From:        fromAddress,
			To:          toAddress,
			Amount:      amount,
			ProtocolID:  eventProtocolID,
			Key:         operatorAddress,
			Message:     transferData.Message,
			MessageHash: transferData.MessageHash,
			TX: fftypes.TransactionRef{
				ID:   transferData.TX,
				Type: fftypes.TransactionTypeTokenTransfer,
			},
		},
		Event: blockchain.Event{
			BlockchainTXID: txHash,
			Source:         ft.Name() + ":" + ft.configuredName,
			Name:           eventName,
			ProtocolID:     eventProtocolID,
			Output:         rawOutput,
			Info:           tx,
			Timestamp:      timestamp,
		},
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return ft.callbacks.TokensTransferred(ft, transfer)
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

func (ft *FFTokens) CreateTokenPool(ctx context.Context, opID *fftypes.UUID, pool *fftypes.TokenPool) error {
	data, _ := json.Marshal(tokenData{
		TX: pool.TX.ID,
	})
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&createPool{
			Type:      pool.Type,
			RequestID: opID.String(),
			Operator:  pool.Key,
			Data:      string(data),
			Config:    pool.Config,
			Name:      pool.Name,
			Symbol:    pool.Symbol,
		}).
		Post("/api/v1/createpool")
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgTokensRESTErr)
	}
	return nil
}

func (ft *FFTokens) ActivateTokenPool(ctx context.Context, opID *fftypes.UUID, pool *fftypes.TokenPool, event *fftypes.BlockchainEvent) error {
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&activatePool{
			RequestID:   opID.String(),
			PoolID:      pool.ProtocolID,
			Transaction: event.Info,
		}).
		Post("/api/v1/activatepool")
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgTokensRESTErr)
	}
	return nil
}

func (ft *FFTokens) MintTokens(ctx context.Context, opID *fftypes.UUID, poolProtocolID string, mint *fftypes.TokenTransfer) error {
	data, _ := json.Marshal(tokenData{
		TX:          mint.TX.ID,
		Message:     mint.Message,
		MessageHash: mint.MessageHash,
	})
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&mintTokens{
			PoolID:    poolProtocolID,
			To:        mint.To,
			Amount:    mint.Amount.Int().String(),
			RequestID: opID.String(),
			Operator:  mint.Key,
			Data:      string(data),
		}).
		Post("/api/v1/mint")
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgTokensRESTErr)
	}
	return nil
}

func (ft *FFTokens) BurnTokens(ctx context.Context, opID *fftypes.UUID, poolProtocolID string, burn *fftypes.TokenTransfer) error {
	data, _ := json.Marshal(tokenData{
		TX:          burn.TX.ID,
		Message:     burn.Message,
		MessageHash: burn.MessageHash,
	})
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&burnTokens{
			PoolID:     poolProtocolID,
			TokenIndex: burn.TokenIndex,
			From:       burn.From,
			Amount:     burn.Amount.Int().String(),
			RequestID:  opID.String(),
			Operator:   burn.Key,
			Data:       string(data),
		}).
		Post("/api/v1/burn")
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgTokensRESTErr)
	}
	return nil
}

func (ft *FFTokens) TransferTokens(ctx context.Context, opID *fftypes.UUID, poolProtocolID string, transfer *fftypes.TokenTransfer) error {
	data, _ := json.Marshal(tokenData{
		TX:          transfer.TX.ID,
		Message:     transfer.Message,
		MessageHash: transfer.MessageHash,
	})
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&transferTokens{
			PoolID:     poolProtocolID,
			TokenIndex: transfer.TokenIndex,
			From:       transfer.From,
			To:         transfer.To,
			Amount:     transfer.Amount.Int().String(),
			RequestID:  opID.String(),
			Operator:   transfer.Key,
			Data:       string(data),
		}).
		Post("/api/v1/transfer")
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgTokensRESTErr)
	}
	return nil
}
