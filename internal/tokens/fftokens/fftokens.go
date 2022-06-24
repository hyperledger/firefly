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
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/wsclient"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/tokens"
)

type FFTokens struct {
	ctx            context.Context
	capabilities   *tokens.Capabilities
	callbacks      callbacks
	configuredName string
	client         *resty.Client
	wsconn         wsclient.WSClient
}

type callbacks struct {
	handlers map[string]tokens.Callbacks
}

func (cb *callbacks) TokenOpUpdate(plugin tokens.Plugin, nsOpID string, txState core.OpStatus, blockchainTXID, errorMessage string, opOutput fftypes.JSONObject) {
	for _, cb := range cb.handlers {
		cb.TokenOpUpdate(plugin, nsOpID, txState, blockchainTXID, errorMessage, opOutput)
	}
}

func (cb *callbacks) TokenPoolCreated(namespace string, plugin tokens.Plugin, pool *tokens.TokenPool) error {
	if namespace == "" {
		// Older token subscriptions don't populate namespace, so deliver the event to every listener
		for _, cb := range cb.handlers {
			if err := cb.TokenPoolCreated(plugin, pool); err != nil {
				return err
			}
		}
	} else {
		if listener, ok := cb.handlers[namespace]; ok {
			return listener.TokenPoolCreated(plugin, pool)
		}
	}
	return nil
}

func (cb *callbacks) TokensTransferred(namespace string, plugin tokens.Plugin, transfer *tokens.TokenTransfer) error {
	if namespace == "" {
		// Older token subscriptions don't populate namespace, so deliver the event to every listener
		for _, cb := range cb.handlers {
			if err := cb.TokensTransferred(plugin, transfer); err != nil {
				return err
			}
		}
	} else {
		if listener, ok := cb.handlers[namespace]; ok {
			return listener.TokensTransferred(plugin, transfer)
		}
	}
	return nil
}

func (cb *callbacks) TokensApproved(namespace string, plugin tokens.Plugin, approval *tokens.TokenApproval) error {
	if namespace == "" {
		// Older token subscriptions don't populate namespace, so deliver the event to every listener
		for _, cb := range cb.handlers {
			if err := cb.TokensApproved(plugin, approval); err != nil {
				return err
			}
		}
	} else {
		if listener, ok := cb.handlers[namespace]; ok {
			return listener.TokensApproved(plugin, approval)
		}
	}
	return nil
}

type wsEvent struct {
	Event msgType            `json:"event"`
	ID    string             `json:"id"`
	Data  fftypes.JSONObject `json:"data"`
}

type msgType string

const (
	messageReceipt       msgType = "receipt"
	messageBatch         msgType = "batch"
	messageTokenPool     msgType = "token-pool"
	messageTokenMint     msgType = "token-mint"
	messageTokenBurn     msgType = "token-burn"
	messageTokenTransfer msgType = "token-transfer"
	messageTokenApproval msgType = "token-approval"
)

type tokenData struct {
	TX          *fftypes.UUID        `json:"tx,omitempty"`
	TXType      core.TransactionType `json:"txtype,omitempty"`
	Message     *fftypes.UUID        `json:"message,omitempty"`
	MessageHash *fftypes.Bytes32     `json:"messageHash,omitempty"`
}

type tokenInit struct {
	Namespace string `json:"namespace"`
}

type createPool struct {
	Type      core.TokenType     `json:"type"`
	RequestID string             `json:"requestId"`
	Signer    string             `json:"signer"`
	Data      string             `json:"data,omitempty"`
	Config    fftypes.JSONObject `json:"config"`
	Name      string             `json:"name"`
	Symbol    string             `json:"symbol"`
}

type activatePool struct {
	Namespace   string             `json:"namespace"`
	PoolLocator string             `json:"poolLocator"`
	Config      fftypes.JSONObject `json:"config"`
	RequestID   string             `json:"requestId,omitempty"`
}

type mintTokens struct {
	PoolLocator string `json:"poolLocator"`
	TokenIndex  string `json:"tokenIndex,omitempty"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	RequestID   string `json:"requestId,omitempty"`
	Signer      string `json:"signer"`
	Data        string `json:"data,omitempty"`
}

type burnTokens struct {
	PoolLocator string `json:"poolLocator"`
	TokenIndex  string `json:"tokenIndex,omitempty"`
	From        string `json:"from"`
	Amount      string `json:"amount"`
	RequestID   string `json:"requestId,omitempty"`
	Signer      string `json:"signer"`
	Data        string `json:"data,omitempty"`
}

type transferTokens struct {
	PoolLocator string `json:"poolLocator"`
	TokenIndex  string `json:"tokenIndex,omitempty"`
	From        string `json:"from"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	RequestID   string `json:"requestId,omitempty"`
	Signer      string `json:"signer"`
	Data        string `json:"data,omitempty"`
}

type tokenApproval struct {
	Signer      string             `json:"signer"`
	Operator    string             `json:"operator"`
	Approved    bool               `json:"approved"`
	PoolLocator string             `json:"poolLocator"`
	RequestID   string             `json:"requestId,omitempty"`
	Data        string             `json:"data,omitempty"`
	Config      fftypes.JSONObject `json:"config"`
}

type tokenError struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

func (ft *FFTokens) Name() string {
	return "fftokens"
}

func (ft *FFTokens) Init(ctx context.Context, name string, config config.Section) (err error) {
	ft.ctx = log.WithLogField(ctx, "proto", "fftokens")
	ft.configuredName = name
	ft.capabilities = &tokens.Capabilities{}
	ft.callbacks.handlers = make(map[string]tokens.Callbacks)

	if config.GetString(ffresty.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, coremsgs.MsgMissingPluginConfig, "url", "tokens.fftokens")
	}
	ft.client = ffresty.New(ft.ctx, config)

	wsConfig := wsclient.GenerateConfig(config)
	if wsConfig.WSKeyPath == "" {
		wsConfig.WSKeyPath = "/api/ws"
	}

	ft.wsconn, err = wsclient.New(ctx, wsConfig, nil, nil)
	if err != nil {
		return err
	}

	go ft.eventLoop()

	return nil
}

func (ft *FFTokens) SetHandler(namespace string, handler tokens.Callbacks) error {
	ft.callbacks.handlers[namespace] = handler

	res, err := ft.client.R().SetContext(ft.ctx).
		SetBody(&tokenInit{
			Namespace: namespace,
		}).
		Post("/api/v1/init")
	if err != nil || !res.IsSuccess() {
		return wrapError(ft.ctx, nil, res, err)
	}
	return nil
}

func (ft *FFTokens) Start() error {
	return ft.wsconn.Connect()
}

func (ft *FFTokens) Capabilities() *tokens.Capabilities {
	return ft.capabilities
}

func (ft *FFTokens) handleReceipt(ctx context.Context, data fftypes.JSONObject) {
	l := log.L(ctx)

	requestID := data.GetString("id")
	success := data.GetBool("success")
	message := data.GetString("message")
	transactionHash := data.GetString("transactionHash")
	if requestID == "" {
		l.Errorf("Reply cannot be processed - missing fields: %+v", data)
		return
	}
	replyType := core.OpStatusSucceeded
	if !success {
		replyType = core.OpStatusFailed
	}
	l.Infof("Tokens '%s' reply: request=%s message=%s", replyType, requestID, message)
	ft.callbacks.TokenOpUpdate(ft, requestID, replyType, transactionHash, message, data)
}

func (ft *FFTokens) handleTokenPoolCreate(ctx context.Context, data fftypes.JSONObject) (err error) {
	tokenType := data.GetString("type")
	poolLocator := data.GetString("poolLocator")
	standard := data.GetString("standard")   // optional
	symbol := data.GetString("symbol")       // optional
	decimals := data.GetInt64("decimals")    // optional
	info := data.GetObject("info")           // optional
	namespace := data.GetString("namespace") // optional

	// All blockchain items below are optional
	blockchainEvent := data.GetObject("blockchain")
	blockchainID := blockchainEvent.GetString("id")
	blockchainInfo := blockchainEvent.GetObject("info")
	txHash := blockchainInfo.GetString("transactionHash")
	timestampStr := blockchainEvent.GetString("timestamp")

	timestamp, err := fftypes.ParseTimeString(timestampStr)
	if err != nil {
		timestamp = fftypes.Now()
	}

	if tokenType == "" || poolLocator == "" {
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

	txType := poolData.TXType
	if txType == "" {
		txType = core.TransactionTypeTokenPool
	}

	pool := &tokens.TokenPool{
		Type:        fftypes.FFEnum(tokenType),
		PoolLocator: poolLocator,
		TX: core.TransactionRef{
			ID:   poolData.TX,
			Type: txType,
		},
		Connector: ft.configuredName,
		Standard:  standard,
		Symbol:    symbol,
		Decimals:  int(decimals),
		Info:      info,
	}

	// Only include a blockchain event if there was some significant blockchain info
	if blockchainID != "" || txHash != "" {
		pool.Event = blockchain.Event{
			ProtocolID:     blockchainID,
			BlockchainTXID: txHash,
			Source:         ft.Name() + ":" + ft.configuredName,
			Name:           blockchainEvent.GetString("name"),
			Output:         blockchainEvent.GetObject("output"),
			Location:       blockchainEvent.GetString("location"),
			Signature:      blockchainEvent.GetString("signature"),
			Info:           blockchainInfo,
			Timestamp:      timestamp,
		}
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return ft.callbacks.TokenPoolCreated(namespace, ft, pool)
}

func (ft *FFTokens) handleTokenTransfer(ctx context.Context, t core.TokenTransferType, data fftypes.JSONObject) (err error) {
	protocolID := data.GetString("id")
	poolLocator := data.GetString("poolLocator")
	signerAddress := data.GetString("signer")
	fromAddress := data.GetString("from")
	toAddress := data.GetString("to")
	value := data.GetString("amount")
	tokenIndex := data.GetString("tokenIndex") // optional
	uri := data.GetString("uri")               // optional
	namespace := data.GetString("namespace")   // optional

	blockchainEvent := data.GetObject("blockchain")
	blockchainID := blockchainEvent.GetString("id")
	blockchainInfo := blockchainEvent.GetObject("info")
	txHash := blockchainInfo.GetString("transactionHash")  // optional
	timestampStr := blockchainEvent.GetString("timestamp") // optional

	timestamp, err := fftypes.ParseTimeString(timestampStr)
	if err != nil {
		timestamp = fftypes.Now()
	}

	if protocolID == "" ||
		poolLocator == "" ||
		signerAddress == "" ||
		value == "" ||
		(t != core.TokenTransferTypeMint && fromAddress == "") ||
		(t != core.TokenTransferTypeBurn && toAddress == "") {
		log.L(ctx).Errorf("%s event is not valid - missing data: %+v", t, data)
		return nil // move on
	}

	// We want to process all events, even those not initiated by FireFly.
	// The "data" argument is optional, so it's important not to fail if it's missing or malformed.
	transferDataString := data.GetString("data")
	var transferData tokenData
	if err = json.Unmarshal([]byte(transferDataString), &transferData); err != nil {
		log.L(ctx).Infof("%s event data could not be parsed - continuing anyway (%s): %+v", t, err, data)
		transferData = tokenData{}
	}

	var amount fftypes.FFBigInt
	_, ok := amount.Int().SetString(value, 10)
	if !ok {
		log.L(ctx).Errorf("%s event is not valid - invalid amount: %+v", t, data)
		return nil // move on
	}

	txType := transferData.TXType
	if txType == "" {
		txType = core.TransactionTypeTokenTransfer
	}

	transfer := &tokens.TokenTransfer{
		PoolLocator: poolLocator,
		TokenTransfer: core.TokenTransfer{
			Type:        t,
			TokenIndex:  tokenIndex,
			URI:         uri,
			Connector:   ft.configuredName,
			From:        fromAddress,
			To:          toAddress,
			Amount:      amount,
			ProtocolID:  protocolID,
			Key:         signerAddress,
			Message:     transferData.Message,
			MessageHash: transferData.MessageHash,
			TX: core.TransactionRef{
				ID:   transferData.TX,
				Type: txType,
			},
		},
		Event: blockchain.Event{
			ProtocolID:     blockchainID,
			BlockchainTXID: txHash,
			Source:         ft.Name() + ":" + ft.configuredName,
			Name:           blockchainEvent.GetString("name"),
			Output:         blockchainEvent.GetObject("output"),
			Location:       blockchainEvent.GetString("location"),
			Signature:      blockchainEvent.GetString("signature"),
			Info:           blockchainInfo,
			Timestamp:      timestamp,
		},
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return ft.callbacks.TokensTransferred(namespace, ft, transfer)
}

func (ft *FFTokens) handleTokenApproval(ctx context.Context, data fftypes.JSONObject) (err error) {
	protocolID := data.GetString("id")
	subject := data.GetString("subject")
	signerAddress := data.GetString("signer")
	poolLocator := data.GetString("poolLocator")
	operatorAddress := data.GetString("operator")
	approved := data.GetBool("approved")
	info := data.GetObject("info")           // optional
	namespace := data.GetString("namespace") // optional

	blockchainEvent := data.GetObject("blockchain")
	blockchainID := blockchainEvent.GetString("id")
	blockchainInfo := blockchainEvent.GetObject("info")
	txHash := blockchainInfo.GetString("transactionHash")  // optional
	timestampStr := blockchainEvent.GetString("timestamp") // optional

	timestamp, err := fftypes.ParseTimeString(timestampStr)
	if err != nil {
		timestamp = fftypes.Now()
	}

	if protocolID == "" ||
		subject == "" ||
		poolLocator == "" ||
		signerAddress == "" ||
		operatorAddress == "" {
		log.L(ctx).Errorf("Approval event is not valid - missing data: %+v", data)
		return nil // move on
	}

	// We want to process all events, even those not initiated by FireFly.
	// The "data" argument is optional, so it's important not to fail if it's missing or malformed.
	approvalDataString := data.GetString("data")
	var transferData tokenData
	if err = json.Unmarshal([]byte(approvalDataString), &transferData); err != nil {
		log.L(ctx).Infof("TokenApproval event data could not be parsed - continuing anyway (%s): %+v", err, data)
		transferData = tokenData{}
	}

	txType := transferData.TXType
	if txType == "" {
		txType = core.TransactionTypeTokenApproval
	}

	approval := &tokens.TokenApproval{
		PoolLocator: poolLocator,
		TokenApproval: core.TokenApproval{
			Connector:  ft.configuredName,
			Key:        signerAddress,
			Operator:   operatorAddress,
			Approved:   approved,
			ProtocolID: protocolID,
			Subject:    subject,
			Info:       info,
			TX: core.TransactionRef{
				ID:   transferData.TX,
				Type: txType,
			},
		},
		Event: blockchain.Event{
			ProtocolID:     blockchainID,
			BlockchainTXID: txHash,
			Source:         ft.Name() + ":" + ft.configuredName,
			Name:           blockchainEvent.GetString("name"),
			Output:         blockchainEvent.GetObject("output"),
			Location:       blockchainEvent.GetString("location"),
			Signature:      blockchainEvent.GetString("signature"),
			Info:           blockchainInfo,
			Timestamp:      timestamp,
		},
	}

	return ft.callbacks.TokensApproved(namespace, ft, approval)
}

func (ft *FFTokens) handleMessage(ctx context.Context, msgBytes []byte) (err error) {
	l := log.L(ctx)

	var msg wsEvent
	if err = json.Unmarshal(msgBytes, &msg); err != nil {
		l.Errorf("Message cannot be parsed as JSON: %s\n%s", err, string(msgBytes))
		return nil // Swallow this and move on
	}

	l.Debugf("Received %s event %s", msg.Event, msg.ID)
	switch msg.Event {
	case messageReceipt:
		ft.handleReceipt(ctx, msg.Data)
	case messageBatch:
		for _, msg := range msg.Data.GetObjectArray("events") {
			if err = ft.handleMessage(ctx, []byte(msg.String())); err != nil {
				break
			}
		}
	case messageTokenPool:
		err = ft.handleTokenPoolCreate(ctx, msg.Data)
	case messageTokenMint:
		err = ft.handleTokenTransfer(ctx, core.TokenTransferTypeMint, msg.Data)
	case messageTokenBurn:
		err = ft.handleTokenTransfer(ctx, core.TokenTransferTypeBurn, msg.Data)
	case messageTokenTransfer:
		err = ft.handleTokenTransfer(ctx, core.TokenTransferTypeTransfer, msg.Data)
	case messageTokenApproval:
		err = ft.handleTokenApproval(ctx, msg.Data)
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

	return err
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
			if err := ft.handleMessage(ctx, msgBytes); err != nil {
				l.Errorf("Event loop exiting: %s", err)
				return
			}
		}
	}
}

// Parse a JSON error of the form:
//   {"error": "Bad Request", "message": "Field 'x' is required"}
// into a message of the form:
//   "Bad Request: Field 'x' is required"
func wrapError(ctx context.Context, errRes *tokenError, res *resty.Response, err error) error {
	if errRes != nil && errRes.Message != "" {
		if errRes.Error != "" {
			return i18n.WrapError(ctx, err, coremsgs.MsgTokensRESTErr, errRes.Error+": "+errRes.Message)
		}
		return i18n.WrapError(ctx, err, coremsgs.MsgTokensRESTErr, errRes.Message)
	}
	return ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgTokensRESTErr)
}

func (ft *FFTokens) CreateTokenPool(ctx context.Context, nsOpID string, pool *core.TokenPool) (complete bool, err error) {
	data, _ := json.Marshal(tokenData{
		TX:     pool.TX.ID,
		TXType: pool.TX.Type,
	})
	var errRes tokenError
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&createPool{
			Type:      pool.Type,
			RequestID: nsOpID,
			Signer:    pool.Key,
			Data:      string(data),
			Config:    pool.Config,
			Name:      pool.Name,
			Symbol:    pool.Symbol,
		}).
		SetError(&errRes).
		Post("/api/v1/createpool")
	if err != nil || !res.IsSuccess() {
		return false, wrapError(ctx, &errRes, res, err)
	}
	if res.StatusCode() == 200 {
		// HTTP 200: Creation was successful, and pool details are in response body
		var obj fftypes.JSONObject
		if err := json.Unmarshal(res.Body(), &obj); err != nil {
			return false, i18n.WrapError(ctx, err, i18n.MsgJSONObjectParseFailed, res.Body())
		}
		return true, ft.handleTokenPoolCreate(ctx, obj)
	}
	// Default (HTTP 202): Request was accepted, and success/failure status will be delivered via websocket
	return false, nil
}

func (ft *FFTokens) ActivateTokenPool(ctx context.Context, nsOpID string, pool *core.TokenPool) (complete bool, err error) {
	var errRes tokenError
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&activatePool{
			RequestID:   nsOpID,
			Namespace:   pool.Namespace,
			PoolLocator: pool.Locator,
			Config:      pool.Config,
		}).
		SetError(&errRes).
		Post("/api/v1/activatepool")
	if err != nil || !res.IsSuccess() {
		return false, wrapError(ctx, &errRes, res, err)
	}
	if res.StatusCode() == 200 {
		// HTTP 200: Activation was successful, and pool details are in response body
		var obj fftypes.JSONObject
		if err := json.Unmarshal(res.Body(), &obj); err != nil {
			return false, i18n.WrapError(ctx, err, i18n.MsgJSONObjectParseFailed, res.Body())
		}
		return true, ft.handleTokenPoolCreate(ctx, obj)
	} else if res.StatusCode() == 204 {
		// HTTP 204: Activation was successful, but pool details are not available
		// This will resolve the operation, but connector is responsible for re-delivering pool details on the websocket.
		return true, nil
	}
	// Default (HTTP 202): Request was accepted, and success/failure status will be delivered via websocket
	return false, nil
}

func (ft *FFTokens) MintTokens(ctx context.Context, nsOpID string, poolLocator string, mint *core.TokenTransfer) error {
	data, _ := json.Marshal(tokenData{
		TX:          mint.TX.ID,
		TXType:      mint.TX.Type,
		Message:     mint.Message,
		MessageHash: mint.MessageHash,
	})
	var errRes tokenError
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&mintTokens{
			PoolLocator: poolLocator,
			TokenIndex:  mint.TokenIndex,
			To:          mint.To,
			Amount:      mint.Amount.Int().String(),
			RequestID:   nsOpID,
			Signer:      mint.Key,
			Data:        string(data),
		}).
		SetError(&errRes).
		Post("/api/v1/mint")
	if err != nil || !res.IsSuccess() {
		return wrapError(ctx, &errRes, res, err)
	}
	return nil
}

func (ft *FFTokens) BurnTokens(ctx context.Context, nsOpID string, poolLocator string, burn *core.TokenTransfer) error {
	data, _ := json.Marshal(tokenData{
		TX:          burn.TX.ID,
		TXType:      burn.TX.Type,
		Message:     burn.Message,
		MessageHash: burn.MessageHash,
	})
	var errRes tokenError
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&burnTokens{
			PoolLocator: poolLocator,
			TokenIndex:  burn.TokenIndex,
			From:        burn.From,
			Amount:      burn.Amount.Int().String(),
			RequestID:   nsOpID,
			Signer:      burn.Key,
			Data:        string(data),
		}).
		SetError(&errRes).
		Post("/api/v1/burn")
	if err != nil || !res.IsSuccess() {
		return wrapError(ctx, &errRes, res, err)
	}
	return nil
}

func (ft *FFTokens) TransferTokens(ctx context.Context, nsOpID string, poolLocator string, transfer *core.TokenTransfer) error {
	data, _ := json.Marshal(tokenData{
		TX:          transfer.TX.ID,
		TXType:      transfer.TX.Type,
		Message:     transfer.Message,
		MessageHash: transfer.MessageHash,
	})
	var errRes tokenError
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&transferTokens{
			PoolLocator: poolLocator,
			TokenIndex:  transfer.TokenIndex,
			From:        transfer.From,
			To:          transfer.To,
			Amount:      transfer.Amount.Int().String(),
			RequestID:   nsOpID,
			Signer:      transfer.Key,
			Data:        string(data),
		}).
		SetError(&errRes).
		Post("/api/v1/transfer")
	if err != nil || !res.IsSuccess() {
		return wrapError(ctx, &errRes, res, err)
	}
	return nil
}

func (ft *FFTokens) TokensApproval(ctx context.Context, nsOpID string, poolLocator string, approval *core.TokenApproval) error {
	data, _ := json.Marshal(tokenData{
		TX:     approval.TX.ID,
		TXType: approval.TX.Type,
	})
	var errRes tokenError
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&tokenApproval{
			PoolLocator: poolLocator,
			Signer:      approval.Key,
			Operator:    approval.Operator,
			Approved:    approval.Approved,
			RequestID:   nsOpID,
			Data:        string(data),
			Config:      approval.Config,
		}).
		SetError(&errRes).
		Post("/api/v1/approval")
	if err != nil || !res.IsSuccess() {
		return wrapError(ctx, &errRes, res, err)
	}
	return nil
}
