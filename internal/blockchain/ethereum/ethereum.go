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

package ethereum

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/wsclient"
	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/hyperledger/firefly-signer/pkg/ffi2abi"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/karlseguin/ccache"
)

const (
	broadcastBatchEventSignature = "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])"
)

type Ethereum struct {
	ctx             context.Context
	topic           string
	prefixShort     string
	prefixLong      string
	capabilities    *blockchain.Capabilities
	callbacks       callbacks
	client          *resty.Client
	fftmClient      *resty.Client
	streams         *streamManager
	streamID        string
	wsconn          wsclient.WSClient
	closed          chan struct{}
	addressResolver *addressResolver
	metrics         metrics.Manager
	ethconnectConf  config.Section
	subs            map[string]subscriptionInfo
	cache           *ccache.Cache
	cacheTTL        time.Duration
}

type subscriptionInfo struct {
	namespace string
	version   int
}

type callbacks struct {
	handlers map[string]blockchain.Callbacks
}

func (cb *callbacks) BlockchainOpUpdate(ctx context.Context, plugin blockchain.Plugin, nsOpID string, txState blockchain.TransactionStatus, blockchainTXID, errorMessage string, opOutput fftypes.JSONObject) {
	namespace, _, _ := core.ParseNamespacedOpID(ctx, nsOpID)
	if handler, ok := cb.handlers[namespace]; ok {
		handler.BlockchainOpUpdate(plugin, nsOpID, txState, blockchainTXID, errorMessage, opOutput)
		return
	}
	log.L(ctx).Errorf("No handler found for blockchain operation '%s'", nsOpID)
}

func (cb *callbacks) BatchPinComplete(ctx context.Context, batch *blockchain.BatchPin, signingKey *core.VerifierRef) error {
	if handler, ok := cb.handlers[batch.Namespace]; ok {
		return handler.BatchPinComplete(batch, signingKey)
	}
	log.L(ctx).Errorf("No handler found for blockchain batch pin on namespace '%s'", batch.Namespace)
	return nil
}

func (cb *callbacks) BlockchainNetworkAction(ctx context.Context, namespace, action string, location *fftypes.JSONAny, event *blockchain.Event, signingKey *core.VerifierRef) error {
	if namespace == "" {
		// V1 networks don't populate namespace, so deliver the event to every handler
		for _, handler := range cb.handlers {
			if err := handler.BlockchainNetworkAction(action, location, event, signingKey); err != nil {
				return err
			}
		}
	} else {
		if handler, ok := cb.handlers[namespace]; ok {
			return handler.BlockchainNetworkAction(action, location, event, signingKey)
		}
		log.L(ctx).Errorf("No handler found for blockchain network action on namespace '%s'", namespace)
	}
	return nil
}

func (cb *callbacks) BlockchainEvent(ctx context.Context, namespace string, event *blockchain.EventWithSubscription) error {
	if namespace == "" {
		// Older token subscriptions don't populate namespace, so deliver the event to every handler
		for _, cb := range cb.handlers {
			// Send the event to all handlers and let them match it to a contract listener
			if err := cb.BlockchainEvent(event); err != nil {
				return err
			}
		}
	} else {
		if handler, ok := cb.handlers[namespace]; ok {
			return handler.BlockchainEvent(event)
		}
		log.L(ctx).Errorf("No handler found for blockchain event on namespace '%s'", namespace)
	}
	return nil
}

type eventStreamWebsocket struct {
	Topic string `json:"topic"`
}

type queryOutput struct {
	Output interface{} `json:"output"`
}

type ethWSCommandPayload struct {
	Type  string `json:"type"`
	Topic string `json:"topic,omitempty"`
}

type ethError struct {
	Error string `json:"error,omitempty"`
}

type Location struct {
	Address string `json:"address"`
}

type EthconnectMessageRequest struct {
	Headers EthconnectMessageHeaders `json:"headers,omitempty"`
	To      string                   `json:"to"`
	From    string                   `json:"from,omitempty"`
	Method  *abi.Entry               `json:"method"`
	Params  []interface{}            `json:"params"`
}

type EthconnectMessageHeaders struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
}

type FFIGenerationInput struct {
	ABI *abi.ABI `json:"abi,omitempty"`
}

var addressVerify = regexp.MustCompile("^[0-9a-f]{40}$")

func (e *Ethereum) Name() string {
	return "ethereum"
}

func (e *Ethereum) VerifierType() core.VerifierType {
	return core.VerifierTypeEthAddress
}

func (e *Ethereum) Init(ctx context.Context, conf config.Section, metrics metrics.Manager) (err error) {
	e.InitConfig(conf)
	ethconnectConf := e.ethconnectConf
	addressResolverConf := conf.SubSection(AddressResolverConfigKey)
	fftmConf := conf.SubSection(FFTMConfigKey)

	e.ctx = log.WithLogField(ctx, "proto", "ethereum")
	e.metrics = metrics
	e.capabilities = &blockchain.Capabilities{}
	e.callbacks.handlers = make(map[string]blockchain.Callbacks)

	if addressResolverConf.GetString(AddressResolverURLTemplate) != "" {
		if e.addressResolver, err = newAddressResolver(ctx, addressResolverConf); err != nil {
			return err
		}
	}

	if ethconnectConf.GetString(ffresty.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, coremsgs.MsgMissingPluginConfig, "url", "blockchain.ethereum.ethconnect")
	}
	e.client = ffresty.New(e.ctx, ethconnectConf)

	if fftmConf.GetString(ffresty.HTTPConfigURL) != "" {
		e.fftmClient = ffresty.New(e.ctx, fftmConf)
	}

	e.topic = ethconnectConf.GetString(EthconnectConfigTopic)
	if e.topic == "" {
		return i18n.NewError(ctx, coremsgs.MsgMissingPluginConfig, "topic", "blockchain.ethereum.ethconnect")
	}
	e.prefixShort = ethconnectConf.GetString(EthconnectPrefixShort)
	e.prefixLong = ethconnectConf.GetString(EthconnectPrefixLong)

	wsConfig := wsclient.GenerateConfig(ethconnectConf)
	if wsConfig.WSKeyPath == "" {
		wsConfig.WSKeyPath = "/ws"
	}
	e.wsconn, err = wsclient.New(ctx, wsConfig, nil, e.afterConnect)
	if err != nil {
		return err
	}

	e.cacheTTL = config.GetDuration(coreconfig.CacheBlockchainTTL)
	e.cache = ccache.New(ccache.Configure().MaxSize(config.GetByteSize(coreconfig.CacheBlockchainSize)))

	e.streams = newStreamManager(e.client, e.cache, e.cacheTTL)
	batchSize := ethconnectConf.GetUint(EthconnectConfigBatchSize)
	batchTimeout := uint(ethconnectConf.GetDuration(EthconnectConfigBatchTimeout).Milliseconds())
	stream, err := e.streams.ensureEventStream(e.ctx, e.topic, batchSize, batchTimeout)
	if err != nil {
		return err
	}
	e.streamID = stream.ID
	e.subs = make(map[string]subscriptionInfo)
	log.L(e.ctx).Infof("Event stream: %s (topic=%s)", e.streamID, e.topic)

	e.closed = make(chan struct{})
	go e.eventLoop()

	return nil
}

func (e *Ethereum) SetHandler(namespace string, handler blockchain.Callbacks) {
	e.callbacks.handlers[namespace] = handler
}

func (e *Ethereum) Start() (err error) {
	return e.wsconn.Connect()
}

func (e *Ethereum) Capabilities() *blockchain.Capabilities {
	return e.capabilities
}

func (e *Ethereum) AddFireflySubscription(ctx context.Context, namespace string, location *fftypes.JSONAny, firstEvent string) (string, error) {
	ethLocation, err := parseContractLocation(ctx, location)
	if err != nil {
		return "", err
	}

	switch firstEvent {
	case string(core.SubOptsFirstEventOldest):
		firstEvent = "0"
	case string(core.SubOptsFirstEventNewest):
		firstEvent = "latest"
	}

	sub, subNS, err := e.streams.ensureFireFlySubscription(ctx, namespace, ethLocation.Address, firstEvent, e.streamID, batchPinEventABI)
	if err != nil {
		return "", err
	}

	version, err := e.GetNetworkVersion(ctx, location)
	if err != nil {
		return "", err
	}

	if version > 1 && subNS == "" {
		return "", i18n.NewError(ctx, coremsgs.MsgInvalidSubscriptionForNetwork, sub.Name, version)
	}

	e.subs[sub.ID] = subscriptionInfo{
		namespace: subNS,
		version:   version,
	}
	return sub.ID, nil
}

func (e *Ethereum) RemoveFireflySubscription(ctx context.Context, subID string) error {
	// Don't actually delete the subscription from ethconnect, as this may be called while processing
	// events from the subscription (and handling that scenario cleanly could be difficult for ethconnect).
	// TODO: can old subscriptions be somehow cleaned up later?
	if _, ok := e.subs[subID]; ok {
		delete(e.subs, subID)
		return nil
	}

	return i18n.NewError(ctx, coremsgs.MsgSubscriptionIDInvalid, subID)
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

func ethHexFormatB32(b *fftypes.Bytes32) string {
	if b == nil {
		return "0x0000000000000000000000000000000000000000000000000000000000000000"
	}
	return "0x" + hex.EncodeToString(b[0:32])
}

func (e *Ethereum) parseBlockchainEvent(ctx context.Context, msgJSON fftypes.JSONObject) *blockchain.Event {
	sBlockNumber := msgJSON.GetString("blockNumber")
	sTransactionHash := msgJSON.GetString("transactionHash")
	blockNumber := msgJSON.GetInt64("blockNumber")
	txIndex := msgJSON.GetInt64("transactionIndex")
	logIndex := msgJSON.GetInt64("logIndex")
	dataJSON := msgJSON.GetObject("data")
	signature := msgJSON.GetString("signature")
	name := strings.SplitN(signature, "(", 2)[0]
	timestampStr := msgJSON.GetString("timestamp")
	timestamp, err := fftypes.ParseTimeString(timestampStr)
	if err != nil {
		log.L(ctx).Errorf("Blockchain event is not valid - missing timestamp: %+v", msgJSON)
		return nil // move on
	}

	if sBlockNumber == "" || sTransactionHash == "" {
		log.L(ctx).Errorf("Blockchain event is not valid - missing data: %+v", msgJSON)
		return nil // move on
	}

	delete(msgJSON, "data")
	return &blockchain.Event{
		BlockchainTXID: sTransactionHash,
		Source:         e.Name(),
		Name:           name,
		ProtocolID:     fmt.Sprintf("%.12d/%.6d/%.6d", blockNumber, txIndex, logIndex),
		Output:         dataJSON,
		Info:           msgJSON,
		Timestamp:      timestamp,
		Location:       e.buildEventLocationString(msgJSON),
		Signature:      signature,
	}
}

func (e *Ethereum) handleBatchPinEvent(ctx context.Context, location *fftypes.JSONAny, subInfo *subscriptionInfo, msgJSON fftypes.JSONObject) (err error) {
	event := e.parseBlockchainEvent(ctx, msgJSON)
	if event == nil {
		return nil // move on
	}

	authorAddress := event.Output.GetString("author")
	nsOrAction := event.Output.GetString("action")
	if nsOrAction == "" {
		nsOrAction = event.Output.GetString("namespace")
	}
	sUUIDs := event.Output.GetString("uuids")
	sBatchHash := event.Output.GetString("batchHash")
	sPayloadRef := event.Output.GetString("payloadRef")
	sContexts := event.Output.GetStringArray("contexts")

	if authorAddress == "" || sUUIDs == "" || sBatchHash == "" {
		log.L(ctx).Errorf("BatchPin event is not valid - missing data: %+v", msgJSON)
		return nil // move on
	}

	authorAddress, err = e.NormalizeSigningKey(ctx, authorAddress)
	if err != nil {
		log.L(ctx).Errorf("BatchPin event is not valid - bad from address (%s): %+v", err, msgJSON)
		return nil // move on
	}
	verifier := &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: authorAddress,
	}

	// Check if this is actually an operator action
	if strings.HasPrefix(nsOrAction, blockchain.FireFlyActionPrefix) {
		action := nsOrAction[len(blockchain.FireFlyActionPrefix):]

		// For V1 of the FireFly contract, action is sent to all namespaces
		// For V2+, namespace is inferred from the subscription
		var namespace string
		if subInfo.version > 1 {
			namespace = subInfo.namespace
		}

		return e.callbacks.BlockchainNetworkAction(ctx, namespace, action, location, event, verifier)
	}

	// For V1 of the FireFly contract, namespace is passed explicitly
	// For V2+, namespace is inferred from the subscription
	var namespace string
	if subInfo.version == 1 {
		namespace = nsOrAction
	} else {
		namespace = subInfo.namespace
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
		Namespace:       namespace,
		TransactionID:   &txnID,
		BatchID:         &batchID,
		BatchHash:       &batchHash,
		BatchPayloadRef: sPayloadRef,
		Contexts:        contexts,
		Event:           *event,
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return e.callbacks.BatchPinComplete(ctx, batch, verifier)
}

func (e *Ethereum) handleContractEvent(ctx context.Context, msgJSON fftypes.JSONObject) (err error) {
	subName, err := e.streams.getSubscriptionName(ctx, msgJSON.GetString("subId"))
	if err != nil {
		return err
	}

	namespace := e.streams.getNamespaceFromSubName(subName)
	event := e.parseBlockchainEvent(ctx, msgJSON)
	if event != nil {
		err = e.callbacks.BlockchainEvent(ctx, namespace, &blockchain.EventWithSubscription{
			Event:        *event,
			Subscription: msgJSON.GetString("subId"),
		})
	}
	return err
}

func (e *Ethereum) handleReceipt(ctx context.Context, reply fftypes.JSONObject) {
	l := log.L(ctx)

	headers := reply.GetObject("headers")
	requestID := headers.GetString("requestId")
	replyType := headers.GetString("type")
	txHash := reply.GetString("transactionHash")
	message := reply.GetString("errorMessage")
	if requestID == "" || replyType == "" {
		l.Errorf("Reply cannot be processed - missing fields: %+v", reply)
		return
	}
	updateType := core.OpStatusSucceeded
	if replyType != "TransactionSuccess" {
		updateType = core.OpStatusFailed
	}
	l.Infof("Ethconnect '%s' reply: request=%s tx=%s message=%s", replyType, requestID, txHash, message)
	e.callbacks.BlockchainOpUpdate(ctx, e, requestID, updateType, txHash, message, reply)
}

func (e *Ethereum) buildEventLocationString(msgJSON fftypes.JSONObject) string {
	return fmt.Sprintf("address=%s", msgJSON.GetString("address"))
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
		sub := msgJSON.GetString("subId")
		l1.Infof("Received '%s' message", signature)
		l1.Tracef("Message: %+v", msgJSON)

		// Matches one of the active FireFly BatchPin subscriptions
		if subInfo, ok := e.subs[sub]; ok {
			location, err := encodeContractLocation(ctx, &Location{
				Address: msgJSON.GetString("address"),
			})
			if err != nil {
				return err
			}

			switch signature {
			case broadcastBatchEventSignature:
				if err := e.handleBatchPinEvent(ctx1, location, &subInfo, msgJSON); err != nil {
					return err
				}
			default:
				l.Infof("Ignoring event with unknown signature: %s", signature)
			}
		} else {
			// Subscription not recognized - assume it's from a custom contract listener
			// (event manager will reject it if it's not)
			if err := e.handleContractEvent(ctx1, msgJSON); err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *Ethereum) eventLoop() {
	defer e.wsconn.Close()
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
				e.handleReceipt(ctx, fftypes.JSONObject(msgTyped))
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

func validateEthAddress(ctx context.Context, key string) (string, error) {
	keyLower := strings.ToLower(key)
	keyNoHexPrefix := strings.TrimPrefix(keyLower, "0x")
	if addressVerify.MatchString(keyNoHexPrefix) {
		return "0x" + keyNoHexPrefix, nil
	}
	return "", i18n.NewError(ctx, coremsgs.MsgInvalidEthAddress)
}

func (e *Ethereum) NormalizeSigningKey(ctx context.Context, key string) (string, error) {
	resolved, err := validateEthAddress(ctx, key)
	if err != nil && e.addressResolver != nil {
		resolved, err := e.addressResolver.NormalizeSigningKey(ctx, key)
		if err == nil {
			log.L(ctx).Infof("Key '%s' resolved to '%s'", key, resolved)
		}
		return resolved, err
	}
	return resolved, err
}

func wrapError(ctx context.Context, errRes *ethError, res *resty.Response, err error) error {
	if errRes != nil && errRes.Error != "" {
		return i18n.WrapError(ctx, err, coremsgs.MsgEthconnectRESTErr, errRes.Error)
	}
	return ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgEthconnectRESTErr)
}

func (e *Ethereum) buildEthconnectRequestBody(ctx context.Context, messageType, address, signingKey string, abi *abi.Entry, requestID string, input []interface{}, options map[string]interface{}) (map[string]interface{}, error) {
	headers := EthconnectMessageHeaders{
		Type: messageType,
	}
	if requestID != "" {
		headers.ID = requestID
	}
	body := map[string]interface{}{
		"headers": headers,
		"to":      address,
		"method":  abi,
		"params":  input,
	}
	if signingKey != "" {
		body["from"] = signingKey
	}
	for k, v := range options {
		// Set the new field if it's not already set. Do not allow overriding of existing fields
		if _, ok := body[k]; !ok {
			body[k] = v
		} else {
			return nil, i18n.NewError(ctx, coremsgs.MsgOverrideExistingFieldCustomOption, k)
		}
	}
	return body, nil
}

func (e *Ethereum) invokeContractMethod(ctx context.Context, address, signingKey string, abi *abi.Entry, requestID string, input []interface{}, options map[string]interface{}) error {
	if e.metrics.IsMetricsEnabled() {
		e.metrics.BlockchainTransaction(address, abi.Name)
	}
	messageType := "SendTransaction"
	body, err := e.buildEthconnectRequestBody(ctx, messageType, address, signingKey, abi, requestID, input, options)
	if err != nil {
		return err
	}
	client := e.fftmClient
	if client == nil {
		client = e.client
	}
	var resErr ethError
	res, err := client.R().
		SetContext(ctx).
		SetBody(body).
		SetError(&resErr).
		Post("/")
	if err != nil || !res.IsSuccess() {
		return wrapError(ctx, &resErr, res, err)
	}
	return nil
}

func (e *Ethereum) queryContractMethod(ctx context.Context, address string, abi *abi.Entry, input []interface{}, options map[string]interface{}) (*resty.Response, error) {
	if e.metrics.IsMetricsEnabled() {
		e.metrics.BlockchainQuery(address, abi.Name)
	}
	messageType := "Query"
	body, err := e.buildEthconnectRequestBody(ctx, messageType, address, "", abi, "", input, options)
	if err != nil {
		return nil, err
	}
	var resErr ethError
	res, err := e.client.R().
		SetContext(ctx).
		SetBody(body).
		SetError(&resErr).
		Post("/")
	if err != nil || !res.IsSuccess() {
		return res, wrapError(ctx, &resErr, res, err)
	}
	return res, nil
}

func (e *Ethereum) SubmitBatchPin(ctx context.Context, nsOpID string, signingKey string, batch *blockchain.BatchPin, location *fftypes.JSONAny) error {
	ethLocation, err := parseContractLocation(ctx, location)
	if err != nil {
		return err
	}

	ethHashes := make([]string, len(batch.Contexts))
	for i, v := range batch.Contexts {
		ethHashes[i] = ethHexFormatB32(v)
	}
	var uuids fftypes.Bytes32
	copy(uuids[0:16], (*batch.TransactionID)[:])
	copy(uuids[16:32], (*batch.BatchID)[:])
	input := []interface{}{
		"",
		ethHexFormatB32(&uuids),
		ethHexFormatB32(batch.BatchHash),
		batch.BatchPayloadRef,
		ethHashes,
	}

	version, err := e.GetNetworkVersion(ctx, location)
	if err != nil {
		return err
	}
	if version == 1 {
		input[0] = batch.Namespace
	}

	return e.invokeContractMethod(ctx, ethLocation.Address, signingKey, batchPinMethodABI, nsOpID, input, nil)
}

func (e *Ethereum) SubmitNetworkAction(ctx context.Context, nsOpID string, signingKey string, action core.NetworkActionType, location *fftypes.JSONAny) error {
	ethLocation, err := parseContractLocation(ctx, location)
	if err != nil {
		return err
	}

	input := []interface{}{
		blockchain.FireFlyActionPrefix + action,
		ethHexFormatB32(nil),
		ethHexFormatB32(nil),
		"",
		[]string{},
	}
	return e.invokeContractMethod(ctx, ethLocation.Address, signingKey, batchPinMethodABI, nsOpID, input, nil)
}

func (e *Ethereum) InvokeContract(ctx context.Context, nsOpID string, signingKey string, location *fftypes.JSONAny, method *fftypes.FFIMethod, input map[string]interface{}, options map[string]interface{}) error {
	ethereumLocation, err := parseContractLocation(ctx, location)
	if err != nil {
		return err
	}
	abi, orderedInput, err := e.prepareRequest(ctx, method, input)
	if err != nil {
		return err
	}
	return e.invokeContractMethod(ctx, ethereumLocation.Address, signingKey, abi, nsOpID, orderedInput, options)
}

func (e *Ethereum) QueryContract(ctx context.Context, location *fftypes.JSONAny, method *fftypes.FFIMethod, input map[string]interface{}, options map[string]interface{}) (interface{}, error) {
	ethereumLocation, err := parseContractLocation(ctx, location)
	if err != nil {
		return nil, err
	}
	abi, orderedInput, err := e.prepareRequest(ctx, method, input)
	if err != nil {
		return nil, err
	}
	res, err := e.queryContractMethod(ctx, ethereumLocation.Address, abi, orderedInput, options)
	if err != nil || !res.IsSuccess() {
		return nil, err
	}
	output := &queryOutput{}
	if err = json.Unmarshal(res.Body(), output); err != nil {
		return nil, err
	}
	return output, nil
}

func (e *Ethereum) NormalizeContractLocation(ctx context.Context, location *fftypes.JSONAny) (result *fftypes.JSONAny, err error) {
	parsed, err := parseContractLocation(ctx, location)
	if err != nil {
		return nil, err
	}
	return encodeContractLocation(ctx, parsed)
}

func parseContractLocation(ctx context.Context, location *fftypes.JSONAny) (*Location, error) {
	ethLocation := Location{}
	if err := json.Unmarshal(location.Bytes(), &ethLocation); err != nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgContractLocationInvalid, err)
	}
	if ethLocation.Address == "" {
		return nil, i18n.NewError(ctx, coremsgs.MsgContractLocationInvalid, "'address' not set")
	}
	return &ethLocation, nil
}

func encodeContractLocation(ctx context.Context, location *Location) (result *fftypes.JSONAny, err error) {
	location.Address, err = validateEthAddress(ctx, location.Address)
	if err != nil {
		return nil, err
	}
	normalized, err := json.Marshal(location)
	if err == nil {
		result = fftypes.JSONAnyPtrBytes(normalized)
	}
	return result, err
}

func (e *Ethereum) AddContractListener(ctx context.Context, listener *core.ContractListenerInput) error {
	location, err := parseContractLocation(ctx, listener.Location)
	if err != nil {
		return err
	}
	abi, err := ffi2abi.ConvertFFIEventDefinitionToABI(ctx, &listener.Event.FFIEventDefinition)
	if err != nil {
		return i18n.WrapError(ctx, err, coremsgs.MsgContractParamInvalid)
	}

	subName := fmt.Sprintf("ff-sub-%s-%s", listener.Namespace, listener.ID)
	firstEvent := string(core.SubOptsFirstEventNewest)
	if listener.Options != nil {
		firstEvent = listener.Options.FirstEvent
	}
	result, err := e.streams.createSubscription(ctx, location, e.streamID, subName, firstEvent, abi)
	if err != nil {
		return err
	}
	listener.BackendID = result.ID
	return nil
}

func (e *Ethereum) DeleteContractListener(ctx context.Context, subscription *core.ContractListener) error {
	return e.streams.deleteSubscription(ctx, subscription.BackendID)
}

func (e *Ethereum) GetFFIParamValidator(ctx context.Context) (fftypes.FFIParamValidator, error) {
	return &ffi2abi.ParamValidator{}, nil
}

func (e *Ethereum) GenerateEventSignature(ctx context.Context, event *fftypes.FFIEventDefinition) string {
	abi, err := ffi2abi.ConvertFFIEventDefinitionToABI(ctx, event)
	if err != nil {
		return ""
	}
	return ffi2abi.ABIMethodToSignature(abi)
}

func (e *Ethereum) prepareRequest(ctx context.Context, method *fftypes.FFIMethod, input map[string]interface{}) (*abi.Entry, []interface{}, error) {
	orderedInput := make([]interface{}, len(method.Params))
	abi, err := ffi2abi.ConvertFFIMethodToABI(ctx, method)
	if err != nil {
		return abi, orderedInput, err
	}
	for i, ffiParam := range method.Params {

		orderedInput[i] = input[ffiParam.Name]
	}
	return abi, orderedInput, nil
}

func (e *Ethereum) getContractAddress(ctx context.Context, instancePath string) (string, error) {
	res, err := e.client.R().
		SetContext(ctx).
		Get(instancePath)
	if err != nil || !res.IsSuccess() {
		return "", ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgEthconnectRESTErr)
	}
	var output map[string]string
	if err = json.Unmarshal(res.Body(), &output); err != nil {
		return "", err
	}
	return output["address"], nil
}

func (e *Ethereum) GenerateFFI(ctx context.Context, generationRequest *fftypes.FFIGenerationRequest) (*fftypes.FFI, error) {
	var input FFIGenerationInput
	err := json.Unmarshal(generationRequest.Input.Bytes(), &input)
	if err != nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgFFIGenerationFailed, "unable to deserialize JSON as ABI")
	}
	if len(*input.ABI) == 0 {
		return nil, i18n.NewError(ctx, coremsgs.MsgFFIGenerationFailed, "ABI is empty")
	}
	return ffi2abi.ConvertABIToFFI(ctx, generationRequest.Namespace, generationRequest.Name, generationRequest.Version, generationRequest.Description, input.ABI)
}

func (e *Ethereum) GetNetworkVersion(ctx context.Context, location *fftypes.JSONAny) (int, error) {
	ethLocation, err := parseContractLocation(ctx, location)
	if err != nil {
		return 0, err
	}

	res, err := e.queryContractMethod(ctx, ethLocation.Address, networkVersionMethodABI, []interface{}{}, nil)
	if err != nil || !res.IsSuccess() {
		// "Call failed" is interpreted as "method does not exist, default to version 1"
		if strings.Contains(err.Error(), "FFEC100148") {
			return 1, nil
		}
		return 0, err
	}
	output := &queryOutput{}
	if err = json.Unmarshal(res.Body(), output); err != nil {
		return 0, err
	}
	return strconv.Atoi(output.Output.(string))
}

func (e *Ethereum) GetAndConvertDeprecatedContractConfig(ctx context.Context) (location *fftypes.JSONAny, fromBlock string, err error) {
	// Old config (attributes under "ethconnect")
	address := e.ethconnectConf.GetString(EthconnectConfigInstanceDeprecated)
	if address != "" {
		log.L(ctx).Warnf("The %s.%s config key has been deprecated. Please use namespaces.predefined[].multiparty.contract[].location.address instead",
			EthconnectConfigKey, EthconnectConfigInstanceDeprecated)
	} else {
		return nil, "", i18n.NewError(ctx, coremsgs.MsgMissingPluginConfig, "instance", "blockchain.ethereum.ethconnect")
	}

	fromBlock = e.ethconnectConf.GetString(EthconnectConfigFromBlockDeprecated)
	if fromBlock != "" {
		log.L(ctx).Warnf("The %s.%s config key has been deprecated. Please use namespaces.predefined[].multiparty.contract[].location.firstEvent instead",
			EthconnectConfigKey, EthconnectConfigFromBlockDeprecated)
	}

	// Backwards compatibility from when instance path was not a contract address
	if strings.HasPrefix(strings.ToLower(address), "/contracts/") {
		address, err = e.getContractAddress(ctx, address)
		if err != nil {
			return nil, "", err
		}
	} else if strings.HasPrefix(address, "/instances/") {
		address = strings.Replace(address, "/instances/", "", 1)
	}

	location, err = encodeContractLocation(ctx, &Location{
		Address: address,
	})
	return location, fromBlock, err
}
