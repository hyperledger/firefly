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

package ethereum

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/restclient"
	"github.com/hyperledger-labs/firefly/internal/wsclient"
	"github.com/hyperledger-labs/firefly/pkg/blockchain"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

const (
	broadcastBatchEventSignature = "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])"
	uriEventSignture             = "URI(string,uint256)"
	transferSingleEventSignature = "TransferSingle(address,address,address,uint256,uint256)"
)

type Ethereum struct {
	ctx          context.Context
	topic        string
	instancePath string
	tokenPath    string
	prefixShort  string
	prefixLong   string
	capabilities *blockchain.Capabilities
	callbacks    blockchain.Callbacks
	client       *resty.Client
	initInfo     struct {
		stream *eventStream
		subs   []*subscription
	}
	wsconn wsclient.WSClient
	closed chan struct{}
}

type eventStream struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	ErrorHandling  string               `json:"errorHandling"`
	BatchSize      uint                 `json:"batchSize"`
	BatchTimeoutMS uint                 `json:"batchTimeoutMS"`
	Type           string               `json:"type"`
	WebSocket      eventStreamWebsocket `json:"websocket"`
}

type eventStreamWebsocket struct {
	Topic string `json:"topic"`
}

type subscription struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Name        string `json:"name"`
	StreamID    string `json:"streamID"`
	Stream      string `json:"stream"`
	FromBlock   string `json:"fromBlock"`
}

type asyncTXSubmission struct {
	ID string `json:"id"`
}

type ethBatchPinInput struct {
	Namespace  string   `json:"namespace"`
	UUIDs      string   `json:"uuids"`
	BatchHash  string   `json:"batchHash"`
	PayloadRef string   `json:"payloadRef"`
	Contexts   []string `json:"contexts"`
}

type ethWSCommandPayload struct {
	Type  string `json:"type"`
	Topic string `json:"topic,omitempty"`
}

type ethCreateTokenPoolInput struct {
	BaseURI    string `json:"uri"`
	IsFungible bool   `json:"is_fungible"`
}

type ethMintFungibleInput struct {
	PoolID     string   `json:"type_id"`
	Recipients []string `json:"to"`
	Amounts    []int    `json:"amounts"`
	Data       []byte   `json:"data"`
}

type ethMintNonFungibleInput struct {
	PoolID     string   `json:"type_id"`
	Recipients []string `json:"to"`
	Data       []byte   `json:"data"`
}

var fireflySubscriptions = []string{
	"BatchPin",
}

var tokenSubscriptions = []string{
	"URI",
	"TransferSingle",
}

var addressVerify = regexp.MustCompile("^[0-9a-f]{40}$")
var zeroAddress = "0x0000000000000000000000000000000000000000"

func (e *Ethereum) Name() string {
	return "ethereum"
}

func (e *Ethereum) Init(ctx context.Context, prefix config.Prefix, callbacks blockchain.Callbacks) (err error) {

	ethconnectConf := prefix.SubPrefix(EthconnectConfigKey)
	tokenConf := prefix.SubPrefix(TokenConfigKey)

	e.ctx = log.WithLogField(ctx, "proto", "ethereum")
	e.callbacks = callbacks

	if ethconnectConf.GetString(restclient.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "url", "blockchain.ethconnect")
	}
	e.instancePath = ethconnectConf.GetString(EthconnectConfigInstancePath)
	if e.instancePath == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "instance", "blockchain.ethconnect")
	}
	e.tokenPath = tokenConf.GetString(EthconnectConfigInstancePath)
	if e.tokenPath == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "instance", "blockchain.tokens")
	}
	e.topic = ethconnectConf.GetString(EthconnectConfigTopic)
	if e.topic == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "topic", "blockchain.ethconnect")
	}

	e.prefixShort = ethconnectConf.GetString(EthconnectPrefixShort)
	e.prefixLong = ethconnectConf.GetString(EthconnectPrefixLong)

	e.client = restclient.New(e.ctx, ethconnectConf)
	e.capabilities = &blockchain.Capabilities{
		GlobalSequencer: true,
	}

	if ethconnectConf.GetString(wsclient.WSConfigKeyPath) == "" {
		ethconnectConf.Set(wsclient.WSConfigKeyPath, "/ws")
	}
	e.wsconn, err = wsclient.New(ctx, ethconnectConf, e.afterConnect)
	if err != nil {
		return err
	}

	if !ethconnectConf.GetBool(EthconnectConfigSkipEventstreamInit) {
		if err = e.ensureEventStreams(ethconnectConf); err != nil {
			return err
		}
	}

	e.closed = make(chan struct{})
	go e.eventLoop()

	return nil
}

func (e *Ethereum) Start() error {
	return e.wsconn.Connect()
}

func (e *Ethereum) Capabilities() *blockchain.Capabilities {
	return e.capabilities
}

func (e *Ethereum) ensureEventStreams(ethconnectConf config.Prefix) error {

	var existingStreams []*eventStream
	res, err := e.client.R().SetContext(e.ctx).SetResult(&existingStreams).Get("/eventstreams")
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
	}

	for _, stream := range existingStreams {
		if stream.WebSocket.Topic == e.topic {
			e.initInfo.stream = stream
		}
	}

	if e.initInfo.stream == nil {
		newStream := eventStream{
			Name:           e.topic,
			ErrorHandling:  "block",
			BatchSize:      ethconnectConf.GetUint(EthconnectConfigBatchSize),
			BatchTimeoutMS: uint(ethconnectConf.GetDuration(EthconnectConfigBatchTimeout).Milliseconds()),
			Type:           "websocket",
		}
		newStream.WebSocket.Topic = e.topic
		res, err = e.client.R().SetBody(&newStream).SetResult(&newStream).Post("/eventstreams")
		if err != nil || !res.IsSuccess() {
			return restclient.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
		}
		e.initInfo.stream = &newStream
	}

	log.L(e.ctx).Infof("Event stream: %s", e.initInfo.stream.ID)

	err = e.ensureSubscriptions(e.instancePath, e.initInfo.stream.ID, fireflySubscriptions)
	if err != nil {
		return err
	}
	return e.ensureSubscriptions(e.tokenPath, e.initInfo.stream.ID, tokenSubscriptions)
}

func (e *Ethereum) afterConnect(ctx context.Context, w wsclient.WSClient) error {
	// Send a subscribe to our topic after each connect/reconnect
	b, _ := json.Marshal(&ethWSCommandPayload{
		Type:  "listen",
		Topic: e.topic,
	})
	err := w.Send(ctx, b)
	if err == nil {
		b, _ = json.Marshal(&ethWSCommandPayload{
			Type: "listenreplies",
		})
		err = w.Send(ctx, b)
	}
	return err
}

func (e *Ethereum) ensureSubscriptions(instancePath string, streamID string, requiredSubscriptions []string) error {
	for _, eventType := range requiredSubscriptions {

		var existingSubs []*subscription
		res, err := e.client.R().SetResult(&existingSubs).Get("/subscriptions")
		if err != nil || !res.IsSuccess() {
			return restclient.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
		}

		var sub *subscription
		for _, s := range existingSubs {
			if s.Name == eventType {
				sub = s
			}
		}

		if sub == nil {
			newSub := subscription{
				Name:      eventType,
				StreamID:  streamID,
				Stream:    e.initInfo.stream.ID,
				FromBlock: "0",
			}
			res, err = e.client.R().
				SetContext(e.ctx).
				SetBody(&newSub).
				SetResult(&newSub).
				Post(fmt.Sprintf("%s/%s", instancePath, eventType))
			if err != nil || !res.IsSuccess() {
				return restclient.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
			}
			sub = &newSub
		}

		log.L(e.ctx).Infof("%s subscription: %s", eventType, sub.ID)
		e.initInfo.subs = append(e.initInfo.subs, sub)

	}
	return nil
}

func ethHexFormatB32(b *fftypes.Bytes32) string {
	if b == nil {
		return "0x0000000000000000000000000000000000000000000000000000000000000000"
	}
	return "0x" + hex.EncodeToString(b[0:32])
}

func (e *Ethereum) handleBatchPinEvent(ctx context.Context, msgJSON fftypes.JSONObject) (err error) {
	sBlockNumber := msgJSON.GetString("blockNumber")
	sTransactionIndex := msgJSON.GetString("transactionIndex")
	sTransactionHash := msgJSON.GetString("transactionHash")
	dataJSON := msgJSON.GetObject("data")
	authorAddress := dataJSON.GetString("author")
	ns := dataJSON.GetString("namespace")
	sUUIDs := dataJSON.GetString("uuids")
	sBatchHash := dataJSON.GetString("batchHash")
	sPayloadRef := dataJSON.GetString("payloadRef")
	sContexts := dataJSON.GetStringArray("contexts")

	if sBlockNumber == "" ||
		sTransactionIndex == "" ||
		sTransactionHash == "" ||
		authorAddress == "" ||
		sUUIDs == "" ||
		sBatchHash == "" {
		log.L(ctx).Errorf("BatchPin event is not valid - missing data: %+v", msgJSON)
		return nil // move on
	}

	authorAddress, err = e.validateEthAddress(ctx, authorAddress)
	if err != nil {
		log.L(ctx).Errorf("BatchPin event is not valid - bad from address (%s): %+v", err, msgJSON)
		return nil // move on
	}

	hexUUIDs, err := hex.DecodeString(strings.TrimPrefix(sUUIDs, "0x"))
	if err != nil || len(hexUUIDs) != 32 {
		log.L(ctx).Errorf("BatchPin event is not valid - bad uuids (%s): %+v", err, msgJSON)
		return nil // move on
	}
	var txnID fftypes.UUID
	copy(txnID[:], hexUUIDs[0:16])
	var batchID fftypes.UUID
	copy(batchID[:], hexUUIDs[16:32])

	var batchHash fftypes.Bytes32
	err = batchHash.UnmarshalText([]byte(sBatchHash))
	if err != nil {
		log.L(ctx).Errorf("BatchPin event is not valid - bad batchHash (%s): %+v", err, msgJSON)
		return nil // move on
	}

	contexts := make([]*fftypes.Bytes32, len(sContexts))
	for i, sHash := range sContexts {
		var hash fftypes.Bytes32
		err = hash.UnmarshalText([]byte(sHash))
		if err != nil {
			log.L(ctx).Errorf("BatchPin event is not valid - bad pin %d (%s): %+v", i, err, msgJSON)
			return nil // move on
		}
		contexts[i] = &hash
	}

	batch := &blockchain.BatchPin{
		Namespace:      ns,
		TransactionID:  &txnID,
		BatchID:        &batchID,
		BatchHash:      &batchHash,
		BatchPaylodRef: sPayloadRef,
		Contexts:       contexts,
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return e.callbacks.BatchPinComplete(batch, authorAddress, sTransactionHash, msgJSON)
}

func (e *Ethereum) handleURIEvent(ctx context.Context, msgJSON fftypes.JSONObject) (err error) {
	sBlockNumber := msgJSON.GetString("blockNumber")
	sTransactionIndex := msgJSON.GetString("transactionIndex")
	sTransactionHash := msgJSON.GetString("transactionHash")
	dataJSON := msgJSON.GetObject("data")
	poolID := dataJSON.GetString("id")
	uri := dataJSON.GetString("value")

	if sBlockNumber == "" ||
		sTransactionIndex == "" ||
		sTransactionHash == "" ||
		poolID == "" ||
		uri == "" {
		log.L(ctx).Errorf("URI event is not valid - missing data: %+v", msgJSON)
		return nil // move on
	}

	id := new(big.Int)
	id.SetString(poolID, 10)
	poolType := fftypes.TokenTypeFungible
	if id.Bit(255) == 1 {
		poolType = fftypes.TokenTypeNonFungible
	}

	pool := &blockchain.TokenPool{
		PoolID:  poolID,
		Type:    poolType,
		BaseURI: uri,
	}
	return e.callbacks.TokenPoolCreated(pool)
}

func (e *Ethereum) handleTransferSingleEvent(ctx context.Context, msgJSON fftypes.JSONObject) (err error) {
	sBlockNumber := msgJSON.GetString("blockNumber")
	sTransactionIndex := msgJSON.GetString("transactionIndex")
	sTransactionHash := msgJSON.GetString("transactionHash")
	dataJSON := msgJSON.GetObject("data")
	fromAddress := dataJSON.GetString("from")
	toAddress := dataJSON.GetString("to")
	poolID := dataJSON.GetString("id")
	value, err := strconv.Atoi(dataJSON.GetString("value"))
	if err != nil {
		return err
	}

	if sBlockNumber == "" ||
		sTransactionIndex == "" ||
		sTransactionHash == "" ||
		fromAddress == "" ||
		toAddress == "" {
		log.L(ctx).Errorf("TransferSingle event is not valid - missing data: %+v", msgJSON)
		return nil // move on
	}

	fromAddress, err = e.validateEthAddress(ctx, fromAddress)
	if err != nil {
		log.L(ctx).Errorf("TransferSingle event is not valid - bad from address (%s): %+v", err, msgJSON)
		return nil // move on
	}

	toAddress, err = e.validateEthAddress(ctx, toAddress)
	if err != nil {
		log.L(ctx).Errorf("TransferSingle event is not valid - bad to address (%s): %+v", err, msgJSON)
		return nil // move on
	}

	switch {
	case fromAddress == zeroAddress && toAddress == zeroAddress:
		// pool creation - handled by URI event
	case fromAddress == zeroAddress:
		return e.callbacks.TokenBalanceChanged(poolID, toAddress, value)
	case toAddress == zeroAddress:
		// TODO: handle burn
	default:
		// TODO: handle transfer
	}

	return nil
}

func (e *Ethereum) handleReceipt(ctx context.Context, reply fftypes.JSONObject) error {
	l := log.L(ctx)

	headers := reply.GetObject("headers")
	requestID := headers.GetString("requestId")
	replyType := headers.GetString("type")
	txHash := reply.GetString("transactionHash")
	message := reply.GetString("errorMessage")
	if requestID == "" || replyType == "" {
		l.Errorf("Reply cannot be processed: %+v", reply)
		return nil // Swallow this and move on
	}
	updateType := fftypes.OpStatusSucceeded
	if replyType != "TransactionSuccess" {
		updateType = fftypes.OpStatusFailed
	}
	l.Infof("Ethconnect '%s' reply tx=%s (request=%s) %s", replyType, txHash, requestID, message)
	return e.callbacks.TxSubmissionUpdate(requestID, updateType, txHash, message, reply)
}

func (e *Ethereum) handleMessageBatch(ctx context.Context, messages []interface{}) error {
	l := log.L(ctx)

	for i, msgI := range messages {
		msgMap, ok := msgI.(map[string]interface{})
		if !ok {
			l.Errorf("Message cannot be parsed as JSON: %+v", msgI)
			return nil // Swallow this and move on
		}
		msgJSON := fftypes.JSONObject(msgMap)

		l1 := l.WithField("ethmsgidx", i)
		ctx1 := log.WithLogger(ctx, l1)
		signature := msgJSON.GetString("signature")
		l1.Infof("Received '%s' message", signature)
		l1.Tracef("Message: %+v", msgJSON)

		switch signature {
		case broadcastBatchEventSignature:
			if err := e.handleBatchPinEvent(ctx1, msgJSON); err != nil {
				return err
			}

		case uriEventSignture:
			if err := e.handleURIEvent(ctx1, msgJSON); err != nil {
				return err
			}

		case transferSingleEventSignature:
			if err := e.handleTransferSingleEvent(ctx1, msgJSON); err != nil {
				return err
			}

		default:
			l.Infof("Ignoring event with unknown signature: %s", signature)
		}
	}

	return nil
}

func (e *Ethereum) eventLoop() {
	defer close(e.closed)
	l := log.L(e.ctx).WithField("role", "event-loop")
	ctx := log.WithLogger(e.ctx, l)
	ack, _ := json.Marshal(map[string]string{"type": "ack", "topic": e.topic})
	for {
		select {
		case <-ctx.Done():
			l.Debugf("Event loop exiting (context cancelled)")
			return
		case msgBytes, ok := <-e.wsconn.Receive():
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
			switch msgTyped := msgParsed.(type) {
			case []interface{}:
				err = e.handleMessageBatch(ctx, msgTyped)
				if err == nil {
					err = e.wsconn.Send(ctx, ack)
				}
			case map[string]interface{}:
				err = e.handleReceipt(ctx, fftypes.JSONObject(msgTyped))
			default:
				l.Errorf("Message unexpected: %+v", msgTyped)
				continue
			}

			// Send the ack - only fails if shutting down
			if err != nil {
				l.Errorf("Event loop exiting: %s", err)
				return
			}
		}
	}
}

func (e *Ethereum) VerifyIdentitySyntax(ctx context.Context, identity *fftypes.Identity) (err error) {
	identity.OnChain, err = e.validateEthAddress(ctx, identity.OnChain)
	return
}

func (e *Ethereum) validateEthAddress(ctx context.Context, identity string) (string, error) {
	identity = strings.TrimPrefix(strings.ToLower(identity), "0x")
	if !addressVerify.MatchString(identity) {
		return "", i18n.NewError(ctx, i18n.MsgInvalidEthAddress)
	}
	return "0x" + identity, nil
}

func (e *Ethereum) invokeContractMethod(ctx context.Context, instancePath string, method string, identity *fftypes.Identity, input interface{}, output interface{}) (*resty.Response, error) {
	return e.client.R().
		SetContext(ctx).
		SetQueryParam(e.prefixShort+"-from", identity.OnChain).
		SetQueryParam(e.prefixShort+"-sync", "false").
		SetBody(input).
		SetResult(output).
		Post(instancePath + "/" + method)
}

func (e *Ethereum) SubmitBatchPin(ctx context.Context, ledgerID *fftypes.UUID, identity *fftypes.Identity, batch *blockchain.BatchPin) (txTrackingID string, err error) {
	tx := &asyncTXSubmission{}
	ethHashes := make([]string, len(batch.Contexts))
	for i, v := range batch.Contexts {
		ethHashes[i] = ethHexFormatB32(v)
	}
	var uuids fftypes.Bytes32
	copy(uuids[0:16], (*batch.TransactionID)[:])
	copy(uuids[16:32], (*batch.BatchID)[:])
	input := &ethBatchPinInput{
		Namespace:  batch.Namespace,
		UUIDs:      ethHexFormatB32(&uuids),
		BatchHash:  ethHexFormatB32(batch.BatchHash),
		PayloadRef: batch.BatchPaylodRef,
		Contexts:   ethHashes,
	}
	res, err := e.invokeContractMethod(ctx, e.instancePath, "pinBatch", identity, input, tx)
	if err != nil || !res.IsSuccess() {
		return "", restclient.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return tx.ID, nil
}

func (e *Ethereum) CreateTokenPool(ctx context.Context, identity *fftypes.Identity, pool *blockchain.TokenPool) (txTrackingID string, err error) {
	tx := &asyncTXSubmission{}
	input := &ethCreateTokenPoolInput{
		BaseURI:    pool.BaseURI,
		IsFungible: pool.Type.Equals(fftypes.TokenTypeFungible),
	}
	res, err := e.invokeContractMethod(ctx, e.tokenPath, "create", identity, input, tx)
	if err != nil || !res.IsSuccess() {
		return "", restclient.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return tx.ID, nil
}

func (e *Ethereum) MintTokens(ctx context.Context, identity *fftypes.Identity, mint *blockchain.TokenMint) (txTrackingID string, err error) {
	tx := &asyncTXSubmission{}
	var res *resty.Response

	if mint.Type == fftypes.TokenTypeFungible {
		input := &ethMintFungibleInput{
			PoolID:     mint.PoolID,
			Recipients: []string{mint.Recipient},
			Amounts:    []int{mint.Amount},
			Data:       []byte{0},
		}
		res, err = e.invokeContractMethod(ctx, e.tokenPath, "mintFungible", identity, input, tx)
	} else {
		input := &ethMintNonFungibleInput{
			PoolID:     mint.PoolID,
			Recipients: []string{mint.Recipient},
			Data:       []byte{0},
		}
		res, err = e.invokeContractMethod(ctx, e.tokenPath, "mintNonFungible", identity, input, tx)
	}

	if err != nil || !res.IsSuccess() {
		return "", restclient.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return tx.ID, nil
}
