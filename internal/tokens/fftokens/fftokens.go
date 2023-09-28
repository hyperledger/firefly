// Copyright © 2023 Kaleido, Inc.
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
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/retry"
	"github.com/hyperledger/firefly-common/pkg/wsclient"
	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/hyperledger/firefly-signer/pkg/ffi2abi"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/tokens"
)

type ConflictError struct {
	err error
}

func (ie *ConflictError) Error() string {
	return ie.err.Error()
}

func (ie *ConflictError) IsConflictError() bool {
	return true
}

type FFTokens struct {
	ctx             context.Context
	cancelCtx       context.CancelFunc
	capabilities    *tokens.Capabilities
	callbacks       callbacks
	configuredName  string
	client          *resty.Client
	wsconn          wsclient.WSClient
	retry           *retry.Retry
	backgroundRetry *retry.Retry
	backgroundStart bool
}

type callbacks struct {
	plugin     tokens.Plugin
	writeLock  sync.Mutex
	handlers   map[string]tokens.Callbacks
	opHandlers map[string]core.OperationCallbacks
}

func (cb *callbacks) OperationUpdate(ctx context.Context, nsOpID string, status core.OpStatus, blockchainTXID, errorMessage string, opOutput fftypes.JSONObject) {
	namespace, _, _ := core.ParseNamespacedOpID(ctx, nsOpID)
	if handler, ok := cb.opHandlers[namespace]; ok {
		handler.OperationUpdate(&core.OperationUpdate{
			Plugin:         cb.plugin.Name(),
			NamespacedOpID: nsOpID,
			Status:         status,
			BlockchainTXID: blockchainTXID,
			ErrorMessage:   errorMessage,
			Output:         opOutput,
		})
	} else {
		log.L(ctx).Errorf("No handler found for token operation '%s'", nsOpID)
	}
}

func (cb *callbacks) TokenPoolCreated(ctx context.Context, namespace string, pool *tokens.TokenPool) error {
	if namespace == "" {
		// Some pool creation subscriptions don't populate namespace, so deliver the event to every handler
		for _, handler := range cb.handlers {
			if err := handler.TokenPoolCreated(ctx, cb.plugin, pool); err != nil {
				return err
			}
		}
	} else {
		if handler, ok := cb.handlers[namespace]; ok {
			return handler.TokenPoolCreated(ctx, cb.plugin, pool)
		}
		log.L(ctx).Errorf("No handler found for token pool event on namespace '%s'", namespace)
	}
	return nil
}

func (cb *callbacks) TokensTransferred(ctx context.Context, namespace string, transfer *tokens.TokenTransfer) error {
	if namespace == "" {
		// Older token subscriptions don't populate namespace, so deliver the event to every handler
		for _, handler := range cb.handlers {
			if err := handler.TokensTransferred(cb.plugin, transfer); err != nil {
				return err
			}
		}
	} else {
		if handler, ok := cb.handlers[namespace]; ok {
			return handler.TokensTransferred(cb.plugin, transfer)
		}
		log.L(ctx).Errorf("No handler found for token transfer event on namespace '%s'", namespace)
	}
	return nil
}

func (cb *callbacks) TokensApproved(ctx context.Context, namespace string, approval *tokens.TokenApproval) error {
	if namespace == "" {
		// Older token subscriptions don't populate namespace, so deliver the event to every handler
		for _, handler := range cb.handlers {
			if err := handler.TokensApproved(cb.plugin, approval); err != nil {
				return err
			}
		}
	} else {
		if handler, ok := cb.handlers[namespace]; ok {
			return handler.TokensApproved(cb.plugin, approval)
		}
		log.L(ctx).Errorf("No handler found for token approval event on namespace '%s'", namespace)
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
	PoolData    string             `json:"poolData"`
	PoolLocator string             `json:"poolLocator"`
	Config      fftypes.JSONObject `json:"config"`
}

type deactivatePool struct {
	PoolData    string             `json:"poolData"`
	PoolLocator string             `json:"poolLocator"`
	Config      fftypes.JSONObject `json:"config"`
}

type tokenInterface struct {
	Format  core.TokenInterfaceFormat `json:"format"`
	Methods interface{}               `json:"methods,omitempty"`
}

type checkInterface struct {
	tokenInterface
	PoolLocator string `json:"poolLocator"`
}

type mintTokens struct {
	PoolLocator string             `json:"poolLocator"`
	TokenIndex  string             `json:"tokenIndex,omitempty"`
	To          string             `json:"to"`
	Amount      string             `json:"amount"`
	RequestID   string             `json:"requestId,omitempty"`
	Signer      string             `json:"signer"`
	Data        string             `json:"data,omitempty"`
	URI         string             `json:"uri,omitempty"`
	Config      fftypes.JSONObject `json:"config"`
	Interface   interface{}        `json:"interface,omitempty"`
}

type burnTokens struct {
	PoolLocator string             `json:"poolLocator"`
	TokenIndex  string             `json:"tokenIndex,omitempty"`
	From        string             `json:"from"`
	Amount      string             `json:"amount"`
	RequestID   string             `json:"requestId,omitempty"`
	Signer      string             `json:"signer"`
	Data        string             `json:"data,omitempty"`
	Config      fftypes.JSONObject `json:"config"`
	Interface   interface{}        `json:"interface,omitempty"`
}

type transferTokens struct {
	PoolLocator string             `json:"poolLocator"`
	TokenIndex  string             `json:"tokenIndex,omitempty"`
	From        string             `json:"from"`
	To          string             `json:"to"`
	Amount      string             `json:"amount"`
	RequestID   string             `json:"requestId,omitempty"`
	Signer      string             `json:"signer"`
	Data        string             `json:"data,omitempty"`
	Config      fftypes.JSONObject `json:"config"`
	Interface   interface{}        `json:"interface,omitempty"`
}

type tokenApproval struct {
	Signer      string             `json:"signer"`
	Operator    string             `json:"operator"`
	Approved    bool               `json:"approved"`
	PoolLocator string             `json:"poolLocator"`
	RequestID   string             `json:"requestId,omitempty"`
	Data        string             `json:"data,omitempty"`
	Config      fftypes.JSONObject `json:"config"`
	Interface   interface{}        `json:"interface,omitempty"`
}

type tokenError struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

func packPoolData(namespace string, id *fftypes.UUID) string {
	if id == nil {
		return namespace
	}
	return namespace + "|" + id.String()
}

func unpackPoolData(ctx context.Context, data string) (namespace string, id *fftypes.UUID) {
	pieces := strings.Split(data, "|")
	if len(pieces) > 1 {
		if id, err := fftypes.ParseUUID(ctx, pieces[1]); err == nil {
			return pieces[0], id
		}
	}
	return pieces[0], nil
}

func (ft *FFTokens) Name() string {
	return "fftokens"
}

func (ft *FFTokens) Init(ctx context.Context, cancelCtx context.CancelFunc, name string, config config.Section) (err error) {
	ft.ctx = log.WithLogField(ctx, "proto", "fftokens")
	ft.cancelCtx = cancelCtx
	ft.configuredName = name
	ft.capabilities = &tokens.Capabilities{}
	ft.callbacks = callbacks{
		plugin:     ft,
		handlers:   make(map[string]tokens.Callbacks),
		opHandlers: make(map[string]core.OperationCallbacks),
	}

	if config.GetString(ffresty.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, coremsgs.MsgMissingPluginConfig, "url", "tokens.fftokens")
	}

	wsConfig, err := wsclient.GenerateConfig(ctx, config)
	if err == nil {
		ft.client, err = ffresty.New(ft.ctx, config)
	}

	if err != nil {
		return err
	}

	if wsConfig.WSKeyPath == "" {
		wsConfig.WSKeyPath = "/api/ws"
	}

	ft.wsconn, err = wsclient.New(ctx, wsConfig, nil, nil)
	if err != nil {
		return err
	}

	ft.retry = &retry.Retry{
		InitialDelay: config.GetDuration(FFTEventRetryInitialDelay),
		MaximumDelay: config.GetDuration(FFTEventRetryMaxDelay),
		Factor:       config.GetFloat64(FFTEventRetryFactor),
	}

	ft.backgroundStart = config.GetBool(FFTBackgroundStart)

	if ft.backgroundStart {
		ft.backgroundRetry = &retry.Retry{
			InitialDelay: config.GetDuration(FFTBackgroundStartInitialDelay),
			MaximumDelay: config.GetDuration(FFTBackgroundStartMaxDelay),
			Factor:       config.GetFloat64(FFTBackgroundStartFactor),
		}
		return nil
	}

	go ft.eventLoop()

	return nil
}

func (ft *FFTokens) SetHandler(namespace string, handler tokens.Callbacks) {
	ft.callbacks.writeLock.Lock()
	defer ft.callbacks.writeLock.Unlock()
	if handler == nil {
		delete(ft.callbacks.handlers, namespace)
	} else {
		ft.callbacks.handlers[namespace] = handler
	}
}

func (ft *FFTokens) SetOperationHandler(namespace string, handler core.OperationCallbacks) {
	ft.callbacks.writeLock.Lock()
	defer ft.callbacks.writeLock.Unlock()
	if handler == nil {
		delete(ft.callbacks.opHandlers, namespace)
	} else {
		ft.callbacks.opHandlers[namespace] = handler
	}
}

func (ft *FFTokens) backgroundStartLoop() {
	_ = ft.backgroundRetry.Do(ft.ctx, fmt.Sprintf("Background start %s", ft.Name()), func(attempt int) (retry bool, err error) {
		err = ft.wsconn.Connect()
		if err != nil {
			return true, err
		}

		go ft.eventLoop()

		return false, nil
	})
}

func (ft *FFTokens) Start() error {
	if ft.backgroundStart {
		go ft.backgroundStartLoop()
		return nil
	}
	return ft.wsconn.Connect()
}

func (ft *FFTokens) Capabilities() *tokens.Capabilities {
	return ft.capabilities
}

func (ft *FFTokens) handleReceipt(ctx context.Context, data fftypes.JSONObject) {
	l := log.L(ctx)

	headers := data.GetObject("headers")
	requestID := headers.GetString("requestId")
	replyType := headers.GetString("type")
	txHash := data.GetString("transactionHash")
	message := data.GetString("errorMessage")
	if requestID == "" || replyType == "" {
		l.Errorf("Reply cannot be processed - missing fields: %+v", data)
		return
	}
	var updateType core.OpStatus
	switch replyType {
	case "TransactionSuccess":
		updateType = core.OpStatusSucceeded
	case "TransactionUpdate":
		updateType = core.OpStatusPending
	default:
		updateType = core.OpStatusFailed
	}
	l.Infof("Received operation update: status=%s request=%s message=%s", updateType, requestID, message)
	ft.callbacks.OperationUpdate(ctx, requestID, updateType, txHash, message, data)
}

func (ft *FFTokens) buildBlockchainEvent(eventData fftypes.JSONObject) *blockchain.Event {
	blockchainID := eventData.GetString("id")
	blockchainInfo := eventData.GetObject("info")
	txHash := blockchainInfo.GetString("transactionHash")

	// Only include a blockchain event if there was some significant blockchain info
	if blockchainID != "" || txHash != "" {
		timestampStr := eventData.GetString("timestamp")
		timestamp, err := fftypes.ParseTimeString(timestampStr)
		if err != nil {
			timestamp = fftypes.Now()
		}

		return &blockchain.Event{
			ProtocolID:     blockchainID,
			BlockchainTXID: txHash,
			Source:         ft.Name() + ":" + ft.configuredName,
			Name:           eventData.GetString("name"),
			Output:         eventData.GetObject("output"),
			Location:       eventData.GetString("location"),
			Signature:      eventData.GetString("signature"),
			Info:           blockchainInfo,
			Timestamp:      timestamp,
		}
	}
	return nil
}

func (ft *FFTokens) handleTokenPoolCreate(ctx context.Context, eventData fftypes.JSONObject, txData *tokenData) (err error) {

	tokenType := eventData.GetString("type")
	poolLocator := eventData.GetString("poolLocator")

	if tokenType == "" || poolLocator == "" {
		log.L(ctx).Errorf("TokenPool event is not valid - missing data: %+v", eventData)
		return nil // move on
	}

	// These fields are optional
	standard := eventData.GetString("standard")
	interfaceFormat := eventData.GetString("interfaceFormat")
	symbol := eventData.GetString("symbol")
	decimals := eventData.GetInt64("decimals")
	info := eventData.GetObject("info")
	blockchainEvent := eventData.GetObject("blockchain")
	poolData := eventData.GetString("poolData")
	namespace, poolID := unpackPoolData(ctx, poolData)

	dataString := eventData.GetString("data")
	if txData == nil {
		txData = &tokenData{}
		if dataString != "" {
			// We want to process all events, even those not initiated by FireFly.
			// The "data" argument is optional, so it's important not to fail if it's missing or malformed.
			if err = json.Unmarshal([]byte(dataString), &txData); err != nil {
				log.L(ctx).Warnf("TokenPool event data could not be parsed - continuing anyway (%s): %+v", err, eventData)
				txData = &tokenData{}
			}
		}
	}

	txType := txData.TXType
	if txType == "" {
		txType = core.TransactionTypeTokenPool
	}

	pool := &tokens.TokenPool{
		ID:          poolID,
		Type:        fftypes.FFEnum(tokenType),
		PoolLocator: poolLocator,
		PluginData:  poolData,
		TX: core.TransactionRef{
			ID:   txData.TX,
			Type: txType,
		},
		Connector:       ft.configuredName,
		Standard:        standard,
		InterfaceFormat: interfaceFormat,
		Symbol:          symbol,
		Decimals:        int(decimals),
		Info:            info,
		Event:           ft.buildBlockchainEvent(blockchainEvent),
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	log.L(ctx).Debugf("Calling TokenPoolCreated callback. Locator='%s' TX=%s/%s", pool.PoolLocator, txType, txData.TX)
	return ft.callbacks.TokenPoolCreated(ctx, namespace, pool)
}

func (ft *FFTokens) handleTokenTransfer(ctx context.Context, t core.TokenTransferType, eventData fftypes.JSONObject) (err error) {
	protocolID := eventData.GetString("id")
	poolLocator := eventData.GetString("poolLocator")
	signerAddress := eventData.GetString("signer")
	fromAddress := eventData.GetString("from")
	toAddress := eventData.GetString("to")
	value := eventData.GetString("amount")
	blockchainEvent := ft.buildBlockchainEvent(eventData.GetObject("blockchain"))

	if protocolID == "" ||
		poolLocator == "" ||
		value == "" ||
		(t != core.TokenTransferTypeMint && fromAddress == "") ||
		(t != core.TokenTransferTypeBurn && toAddress == "") ||
		blockchainEvent == nil {
		log.L(ctx).Errorf("%s event is not valid - missing data: %+v", t, eventData)
		return nil // move on
	}

	// These fields are optional
	tokenIndex := eventData.GetString("tokenIndex")
	uri := eventData.GetString("uri")
	namespace, poolID := unpackPoolData(ctx, eventData.GetString("poolData"))

	// We want to process all events, even those not initiated by FireFly.
	// The "data" argument is optional, so it's important not to fail if it's missing or malformed.
	transferDataString := eventData.GetString("data")
	var transferData tokenData
	if err = json.Unmarshal([]byte(transferDataString), &transferData); err != nil {
		log.L(ctx).Infof("%s event data could not be parsed - continuing anyway (%s): %+v", t, err, eventData)
		transferData = tokenData{}
	}

	var amount fftypes.FFBigInt
	_, ok := amount.Int().SetString(value, 10)
	if !ok {
		log.L(ctx).Errorf("%s event is not valid - invalid amount: %+v", t, eventData)
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
			Pool:        poolID,
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
		Event: blockchainEvent,
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return ft.callbacks.TokensTransferred(ctx, namespace, transfer)
}

func (ft *FFTokens) handleTokenApproval(ctx context.Context, eventData fftypes.JSONObject) (err error) {
	protocolID := eventData.GetString("id")
	subject := eventData.GetString("subject")
	signerAddress := eventData.GetString("signer")
	poolLocator := eventData.GetString("poolLocator")
	operatorAddress := eventData.GetString("operator")
	approved := eventData.GetBool("approved")
	blockchainEvent := ft.buildBlockchainEvent(eventData.GetObject("blockchain"))

	if protocolID == "" ||
		subject == "" ||
		poolLocator == "" ||
		operatorAddress == "" ||
		blockchainEvent == nil {
		log.L(ctx).Errorf("Approval event is not valid - missing data: %+v", eventData)
		return nil // move on
	}

	// These fields are optional
	info := eventData.GetObject("info")
	namespace, poolID := unpackPoolData(ctx, eventData.GetString("poolData"))

	// We want to process all events, even those not initiated by FireFly.
	// The "data" argument is optional, so it's important not to fail if it's missing or malformed.
	approvalDataString := eventData.GetString("data")
	var approvalData tokenData
	if err = json.Unmarshal([]byte(approvalDataString), &approvalData); err != nil {
		log.L(ctx).Infof("TokenApproval event data could not be parsed - continuing anyway (%s): %+v", err, eventData)
		approvalData = tokenData{}
	}

	txType := approvalData.TXType
	if txType == "" {
		txType = core.TransactionTypeTokenApproval
	}

	approval := &tokens.TokenApproval{
		PoolLocator: poolLocator,
		TokenApproval: core.TokenApproval{
			Connector:   ft.configuredName,
			Pool:        poolID,
			Key:         signerAddress,
			Operator:    operatorAddress,
			Approved:    approved,
			ProtocolID:  protocolID,
			Subject:     subject,
			Info:        info,
			Message:     approvalData.Message,
			MessageHash: approvalData.MessageHash,
			TX: core.TransactionRef{
				ID:   approvalData.TX,
				Type: txType,
			},
		},
		Event: blockchainEvent,
	}

	return ft.callbacks.TokensApproved(ctx, namespace, approval)
}

func (ft *FFTokens) handleMessage(ctx context.Context, msgBytes []byte) (retry bool, err error) {
	var msg *wsEvent
	if err = json.Unmarshal(msgBytes, &msg); err != nil {
		log.L(ctx).Errorf("Message cannot be parsed as JSON: %s\n%s", err, string(msgBytes))
		return false, nil // Swallow this and move on
	}
	log.L(ctx).Debugf("Received %s event %s", msg.Event, msg.ID)
	switch msg.Event {
	case messageReceipt:
		ft.handleReceipt(ctx, msg.Data)
	case messageBatch:
		for _, msg := range msg.Data.GetObjectArray("events") {
			if retry, err = ft.handleMessage(ctx, []byte(msg.String())); err != nil {
				return retry, err
			}
		}
	case messageTokenPool:
		err = ft.handleTokenPoolCreate(ctx, msg.Data, nil /* need to extract poolData from event */)
	case messageTokenMint:
		err = ft.handleTokenTransfer(ctx, core.TokenTransferTypeMint, msg.Data)
	case messageTokenBurn:
		err = ft.handleTokenTransfer(ctx, core.TokenTransferTypeBurn, msg.Data)
	case messageTokenTransfer:
		err = ft.handleTokenTransfer(ctx, core.TokenTransferTypeTransfer, msg.Data)
	case messageTokenApproval:
		err = ft.handleTokenApproval(ctx, msg.Data)
	default:
		log.L(ctx).Errorf("Message unexpected: %s", msg.Event)
		// do not set error here - we will never be able to process this message so log+swallow it.
	}
	if err != nil {
		// All errors above are retryable
		return true, err
	}
	if msg.Event != messageReceipt && msg.ID != "" {
		log.L(ctx).Debugf("Sending ack %s", msg.ID)
		ack, _ := json.Marshal(fftypes.JSONObject{
			"event": "ack",
			"data": fftypes.JSONObject{
				"id": msg.ID,
			},
		})
		// Do not retry this
		return false, ft.wsconn.Send(ctx, ack)
	}
	return false, nil
}

func (ft *FFTokens) handleMessageRetry(ctx context.Context, msgBytes []byte) (err error) {
	eventCtx, done := context.WithCancel(ctx)
	defer done()
	return ft.retry.Do(eventCtx, "fftokens event", func(attempt int) (retry bool, err error) {
		return ft.handleMessage(eventCtx, msgBytes) // We keep retrying on error until the context ends
	})
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
				l.Debugf("Event loop exiting (receive channel closed). Terminating server!")
				ft.cancelCtx()
				return
			}
			if err := ft.handleMessageRetry(ctx, msgBytes); err != nil {
				l.Errorf("Event loop exiting (%s). Terminating server!", err)
				ft.cancelCtx()
				return
			}
		}
	}
}

// Parse a JSON error of the form:
// {"error": "Bad Request", "message": "Field 'x' is required"}
// into a message of the form:
//
//	"Bad Request: Field 'x' is required"
func wrapError(ctx context.Context, errRes *tokenError, res *resty.Response, err error) error {
	if errRes != nil && (errRes.Message != "" || errRes.Error != "") {
		errMsgFromBody := errRes.Message
		if errRes.Error != "" {
			errMsgFromBody = errRes.Error + ": " + errRes.Message
		}
		if res != nil && res.StatusCode() == http.StatusConflict {
			return &ConflictError{err: i18n.WrapError(ctx, err, coremsgs.MsgTokensRESTErrConflict, errMsgFromBody)}
		}
		err = i18n.WrapError(ctx, err, coremsgs.MsgTokensRESTErr, errMsgFromBody)
	}
	if res != nil && res.StatusCode() == http.StatusConflict {
		return &ConflictError{err: ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgTokensRESTErrConflict)}
	}
	return ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgTokensRESTErr)
}

func (ft *FFTokens) CreateTokenPool(ctx context.Context, nsOpID string, pool *core.TokenPool) (phase core.OpPhase, err error) {
	tokenData := &tokenData{
		TX:     pool.TX.ID,
		TXType: pool.TX.Type,
	}
	data, _ := json.Marshal(tokenData)
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
		return core.OpPhaseInitializing, wrapError(ctx, &errRes, res, err)
	}
	if res.StatusCode() == 200 {
		// HTTP 200: Creation was successful, and pool details are in response body
		var obj fftypes.JSONObject
		if err := json.Unmarshal(res.Body(), &obj); err != nil {
			return core.OpPhaseComplete, i18n.WrapError(ctx, err, i18n.MsgJSONObjectParseFailed, res.Body())
		}
		obj["poolData"] = packPoolData(pool.Namespace, pool.ID)
		return core.OpPhaseComplete, ft.handleTokenPoolCreate(ctx, obj, tokenData)
	}
	// Default (HTTP 202): Request was accepted, and success/failure status will be delivered via websocket
	return core.OpPhasePending, nil
}

func (ft *FFTokens) ActivateTokenPool(ctx context.Context, pool *core.TokenPool) (phase core.OpPhase, err error) {
	var errRes tokenError
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&activatePool{
			PoolData:    packPoolData(pool.Namespace, pool.ID),
			PoolLocator: pool.Locator,
			Config:      pool.Config,
		}).
		SetError(&errRes).
		Post("/api/v1/activatepool")
	if err != nil || !res.IsSuccess() {
		return core.OpPhaseInitializing, wrapError(ctx, &errRes, res, err)
	}
	if res.StatusCode() == 200 {
		// HTTP 200: Activation was successful, and pool details are in response body
		var obj fftypes.JSONObject
		if err := json.Unmarshal(res.Body(), &obj); err != nil {
			return core.OpPhaseComplete, i18n.WrapError(ctx, err, i18n.MsgJSONObjectParseFailed, res.Body())
		}
		return core.OpPhaseComplete, ft.handleTokenPoolCreate(ctx, obj, &tokenData{
			TX:     pool.TX.ID,
			TXType: pool.TX.Type,
		})
	} else if res.StatusCode() == 204 {
		// HTTP 204: Activation was successful, but pool details are not available
		// This will resolve the operation, but connector is responsible for re-delivering pool details on the websocket.
		return core.OpPhaseComplete, nil
	}
	// Default (HTTP 202): Request was accepted, and success/failure status will be delivered via websocket
	return core.OpPhasePending, nil
}

func (ft *FFTokens) DeactivateTokenPool(ctx context.Context, pool *core.TokenPool) error {
	var errRes tokenError
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&deactivatePool{
			PoolData:    pool.PluginData,
			PoolLocator: pool.Locator,
			Config:      pool.Config,
		}).
		SetError(&errRes).
		Post("/api/v1/deactivatepool")
	if err == nil && (res.IsSuccess() || res.StatusCode() == 404) {
		return nil
	}
	return wrapError(ctx, &errRes, res, err)
}

func (ft *FFTokens) prepareABI(ctx context.Context, methods []*fftypes.FFIMethod) ([]*abi.Entry, error) {
	abiMethods := make([]*abi.Entry, len(methods))
	for i, method := range methods {
		abi, err := ffi2abi.ConvertFFIMethodToABI(ctx, method)
		if err != nil {
			return nil, err
		}
		abiMethods[i] = abi
	}
	return abiMethods, nil
}

func (ft *FFTokens) CheckInterface(ctx context.Context, pool *core.TokenPool, methods []*fftypes.FFIMethod) (*fftypes.JSONAny, error) {
	body := checkInterface{
		PoolLocator: pool.Locator,
		tokenInterface: tokenInterface{
			Format: pool.InterfaceFormat,
		},
	}

	switch pool.InterfaceFormat {
	case core.TokenInterfaceFormatFFI:
		body.tokenInterface.Methods = methods
	case core.TokenInterfaceFormatABI:
		abi, err := ft.prepareABI(ctx, methods)
		if err != nil {
			return nil, err
		}
		body.tokenInterface.Methods = abi
	default:
		return nil, i18n.NewError(ctx, coremsgs.MsgUnknownInterfaceFormat, pool.InterfaceFormat)
	}

	var errRes tokenError
	res, err := ft.client.R().SetContext(ctx).
		SetBody(&body).
		SetError(&errRes).
		Post("/api/v1/checkinterface")
	if err != nil || !res.IsSuccess() {
		return nil, wrapError(ctx, &errRes, res, err)
	}

	var obj tokens.TokenPoolMethods
	if err := json.Unmarshal(res.Body(), &obj); err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgJSONObjectParseFailed, res.Body())
	}
	return fftypes.JSONAnyPtrBytes(res.Body()), nil
}

func (ft *FFTokens) MintTokens(ctx context.Context, nsOpID string, poolLocator string, mint *core.TokenTransfer, methods *fftypes.JSONAny) error {
	data, _ := json.Marshal(tokenData{
		TX:          mint.TX.ID,
		TXType:      mint.TX.Type,
		Message:     mint.Message,
		MessageHash: mint.MessageHash,
	})

	var iface interface{}
	if methods != nil {
		iface = methods.JSONObject()["mint"]
	}

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
			URI:         mint.URI,
			Config:      mint.Config,
			Interface:   iface,
		}).
		SetError(&errRes).
		Post("/api/v1/mint")
	if err != nil || !res.IsSuccess() {
		return wrapError(ctx, &errRes, res, err)
	}
	return nil
}

func (ft *FFTokens) BurnTokens(ctx context.Context, nsOpID string, poolLocator string, burn *core.TokenTransfer, methods *fftypes.JSONAny) error {
	data, _ := json.Marshal(tokenData{
		TX:          burn.TX.ID,
		TXType:      burn.TX.Type,
		Message:     burn.Message,
		MessageHash: burn.MessageHash,
	})

	var iface interface{}
	if methods != nil {
		iface = methods.JSONObject()["burn"]
	}

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
			Config:      burn.Config,
			Interface:   iface,
		}).
		SetError(&errRes).
		Post("/api/v1/burn")
	if err != nil || !res.IsSuccess() {
		return wrapError(ctx, &errRes, res, err)
	}
	return nil
}

func (ft *FFTokens) TransferTokens(ctx context.Context, nsOpID string, poolLocator string, transfer *core.TokenTransfer, methods *fftypes.JSONAny) error {
	data, _ := json.Marshal(tokenData{
		TX:          transfer.TX.ID,
		TXType:      transfer.TX.Type,
		Message:     transfer.Message,
		MessageHash: transfer.MessageHash,
	})

	var iface interface{}
	if methods != nil {
		iface = methods.JSONObject()["transfer"]
	}

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
			Config:      transfer.Config,
			Interface:   iface,
		}).
		SetError(&errRes).
		Post("/api/v1/transfer")
	if err != nil || !res.IsSuccess() {
		return wrapError(ctx, &errRes, res, err)
	}
	return nil
}

func (ft *FFTokens) TokensApproval(ctx context.Context, nsOpID string, poolLocator string, approval *core.TokenApproval, methods *fftypes.JSONAny) error {
	data, _ := json.Marshal(tokenData{
		TX:          approval.TX.ID,
		TXType:      approval.TX.Type,
		Message:     approval.Message,
		MessageHash: approval.MessageHash,
	})

	var iface interface{}
	if methods != nil {
		iface = methods.JSONObject()["approval"]
	}

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
			Interface:   iface,
		}).
		SetError(&errRes).
		Post("/api/v1/approval")
	if err != nil || !res.IsSuccess() {
		return wrapError(ctx, &errRes, res, err)
	}
	return nil
}
