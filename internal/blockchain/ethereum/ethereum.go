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
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/wsclient"
	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

const (
	broadcastBatchEventSignature = "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])"
	addressType                  = "address"
	boolType                     = "bool"
	booleanType                  = "boolean"
	integerType                  = "integer"
	tupleType                    = "tuple"
	stringType                   = "string"
	arrayType                    = "array"
	objectType                   = "object"
)

type Ethereum struct {
	ctx             context.Context
	topic           string
	prefixShort     string
	prefixLong      string
	capabilities    *blockchain.Capabilities
	callbacks       blockchain.Callbacks
	client          *resty.Client
	fftmClient      *resty.Client
	streams         *streamManager
	streamID        string
	fireflyContract struct {
		mux            sync.Mutex
		address        string
		fromBlock      string
		networkVersion int
		subscription   string
	}
	wsconn           wsclient.WSClient
	closed           chan struct{}
	addressResolver  *addressResolver
	metrics          metrics.Manager
	ethconnectConf   config.Section
	contractConf     config.ArraySection
	contractConfSize int
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

type paramDetails struct {
	Type         string `json:"type"`
	InternalType string `json:"internalType,omitempty"`
	Indexed      bool   `json:"indexed,omitempty"`
	Index        *int   `json:"index,omitempty"`
}

type Schema struct {
	OneOf       []SchemaType       `json:"oneOf,omitempty"`
	Type        string             `json:"type,omitempty"`
	Details     *paramDetails      `json:"details,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Description string             `json:"description,omitempty"`
}

type SchemaType struct {
	Type string `json:"type"`
}

func (s *Schema) ToJSON() string {
	b, _ := json.Marshal(s)
	return string(b)
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

func (e *Ethereum) Init(ctx context.Context, config config.Section, callbacks blockchain.Callbacks, metrics metrics.Manager) (err error) {
	e.InitConfig(config)
	ethconnectConf := e.ethconnectConf
	addressResolverConf := config.SubSection(AddressResolverConfigKey)
	fftmConf := config.SubSection(FFTMConfigKey)

	e.ctx = log.WithLogField(ctx, "proto", "ethereum")
	e.callbacks = callbacks
	e.metrics = metrics
	e.capabilities = &blockchain.Capabilities{}

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

	e.streams = &streamManager{client: e.client}
	batchSize := ethconnectConf.GetUint(EthconnectConfigBatchSize)
	batchTimeout := uint(ethconnectConf.GetDuration(EthconnectConfigBatchTimeout).Milliseconds())
	stream, err := e.streams.ensureEventStream(e.ctx, e.topic, batchSize, batchTimeout)
	if err != nil {
		return err
	}
	e.streamID = stream.ID
	log.L(e.ctx).Infof("Event stream: %s (topic=%s)", e.streamID, e.topic)

	e.closed = make(chan struct{})
	go e.eventLoop()

	return nil
}

func (e *Ethereum) Start() (err error) {
	return e.wsconn.Connect()
}

func (e *Ethereum) resolveFireFlyContract(ctx context.Context, contractIndex int) (address, fromBlock string, err error) {

	if e.contractConfSize > 0 || contractIndex > 0 {
		// New config (array of objects under "fireflyContract")
		if contractIndex >= e.contractConfSize {
			return "", "", i18n.NewError(ctx, coremsgs.MsgInvalidFireFlyContractIndex, fmt.Sprintf("blockchain.ethereum.fireflyContract[%d]", contractIndex))
		}
		entry := e.contractConf.ArrayEntry(contractIndex)
		address = entry.GetString(FireFlyContractAddress)
		if address == "" {
			return "", "", i18n.NewError(ctx, coremsgs.MsgMissingPluginConfig, "address", "blockchain.ethereum.fireflyContract")
		}
		fromBlock = entry.GetString(FireFlyContractFromBlock)
	} else {
		// Old config (attributes under "ethconnect")
		address = e.ethconnectConf.GetString(EthconnectConfigInstanceDeprecated)
		if address != "" {
			log.L(ctx).Warnf("The %s.%s config key has been deprecated. Please use %s.%s instead",
				EthconnectConfigKey, EthconnectConfigInstanceDeprecated,
				FireFlyContractConfigKey, FireFlyContractAddress)
		} else {
			return "", "", i18n.NewError(ctx, coremsgs.MsgMissingPluginConfig, "instance", "blockchain.ethereum.ethconnect")
		}
		fromBlock = e.ethconnectConf.GetString(EthconnectConfigFromBlockDeprecated)
		if fromBlock != "" {
			log.L(ctx).Warnf("The %s.%s config key has been deprecated. Please use %s.%s instead",
				EthconnectConfigKey, EthconnectConfigFromBlockDeprecated,
				FireFlyContractConfigKey, FireFlyContractFromBlock)
		}
	}

	// Backwards compatibility from when instance path was not a contract address
	if strings.HasPrefix(strings.ToLower(address), "/contracts/") {
		address, err = e.getContractAddress(ctx, address)
		if err != nil {
			return "", "", err
		}
	} else if strings.HasPrefix(address, "/instances/") {
		address = strings.Replace(address, "/instances/", "", 1)
	}

	address, err = validateEthAddress(ctx, address)
	return address, fromBlock, err
}

func (e *Ethereum) ConfigureContract(ctx context.Context, contracts *core.FireFlyContracts) (err error) {

	log.L(ctx).Infof("Resolving FireFly contract at index %d", contracts.Active.Index)
	address, fromBlock, err := e.resolveFireFlyContract(ctx, contracts.Active.Index)
	if err != nil {
		return err
	}

	sub, err := e.streams.ensureFireFlySubscription(ctx, address, fromBlock, e.streamID, batchPinEventABI)
	if err == nil {
		var version int
		version, err = e.getNetworkVersion(ctx, address)
		if err == nil {
			e.fireflyContract.mux.Lock()
			e.fireflyContract.address = address
			e.fireflyContract.fromBlock = fromBlock
			e.fireflyContract.networkVersion = version
			e.fireflyContract.subscription = sub.ID
			e.fireflyContract.mux.Unlock()
			contracts.Active.Info = fftypes.JSONObject{
				"address":      address,
				"fromBlock":    fromBlock,
				"subscription": sub.ID,
			}
		}
	}
	return err
}

func (e *Ethereum) TerminateContract(ctx context.Context, contracts *core.FireFlyContracts, termination *blockchain.Event) (err error) {

	address, err := validateEthAddress(ctx, termination.Info.GetString("address"))
	if err != nil {
		return err
	}
	e.fireflyContract.mux.Lock()
	fireflyAddress := e.fireflyContract.address
	e.fireflyContract.mux.Unlock()
	if address != fireflyAddress {
		log.L(ctx).Warnf("Ignoring termination request from address %s, which differs from active address %s", address, fireflyAddress)
		return nil
	}

	log.L(ctx).Infof("Processing termination request from address %s", address)
	contracts.Active.FinalEvent = termination.ProtocolID
	contracts.Terminated = append(contracts.Terminated, contracts.Active)
	contracts.Active = core.FireFlyContractInfo{Index: contracts.Active.Index + 1}
	return e.ConfigureContract(ctx, contracts)
}

func (e *Ethereum) Capabilities() *blockchain.Capabilities {
	return e.capabilities
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

func (e *Ethereum) handleBatchPinEvent(ctx context.Context, msgJSON fftypes.JSONObject) (err error) {
	event := e.parseBlockchainEvent(ctx, msgJSON)
	if event == nil {
		return nil // move on
	}

	authorAddress := event.Output.GetString("author")
	nsOrAction := event.Output.GetString("namespace")
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
		return e.callbacks.BlockchainNetworkAction(action, event, verifier)
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
		Namespace:       nsOrAction,
		TransactionID:   &txnID,
		BatchID:         &batchID,
		BatchHash:       &batchHash,
		BatchPayloadRef: sPayloadRef,
		Contexts:        contexts,
		Event:           *event,
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return e.callbacks.BatchPinComplete(batch, verifier)
}

func (e *Ethereum) handleContractEvent(ctx context.Context, msgJSON fftypes.JSONObject) (err error) {
	event := e.parseBlockchainEvent(ctx, msgJSON)
	if event != nil {
		err = e.callbacks.BlockchainEvent(&blockchain.EventWithSubscription{
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
	e.callbacks.BlockchainOpUpdate(e, requestID, updateType, txHash, message, reply)
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

		e.fireflyContract.mux.Lock()
		fireflySub := e.fireflyContract.subscription
		e.fireflyContract.mux.Unlock()

		if sub == fireflySub {
			// Matches the active FireFly BatchPin subscription
			switch signature {
			case broadcastBatchEventSignature:
				if err := e.handleBatchPinEvent(ctx1, msgJSON); err != nil {
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

func (e *Ethereum) SubmitBatchPin(ctx context.Context, nsOpID string, signingKey string, batch *blockchain.BatchPin) error {
	ethHashes := make([]string, len(batch.Contexts))
	for i, v := range batch.Contexts {
		ethHashes[i] = ethHexFormatB32(v)
	}
	var uuids fftypes.Bytes32
	copy(uuids[0:16], (*batch.TransactionID)[:])
	copy(uuids[16:32], (*batch.BatchID)[:])
	input := []interface{}{
		batch.Namespace,
		ethHexFormatB32(&uuids),
		ethHexFormatB32(batch.BatchHash),
		batch.BatchPayloadRef,
		ethHashes,
	}
	e.fireflyContract.mux.Lock()
	address := e.fireflyContract.address
	e.fireflyContract.mux.Unlock()
	return e.invokeContractMethod(ctx, address, signingKey, batchPinMethodABI, nsOpID, input, nil)
}

func (e *Ethereum) SubmitNetworkAction(ctx context.Context, nsOpID string, signingKey string, action core.NetworkActionType) error {
	input := []interface{}{
		blockchain.FireFlyActionPrefix + action,
		ethHexFormatB32(nil),
		ethHexFormatB32(nil),
		"",
		[]string{},
	}
	e.fireflyContract.mux.Lock()
	address := e.fireflyContract.address
	e.fireflyContract.mux.Unlock()
	return e.invokeContractMethod(ctx, address, signingKey, batchPinMethodABI, nsOpID, input, nil)
}

func (e *Ethereum) InvokeContract(ctx context.Context, nsOpID string, signingKey string, location *fftypes.JSONAny, method *core.FFIMethod, input map[string]interface{}, options map[string]interface{}) error {
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

func (e *Ethereum) QueryContract(ctx context.Context, location *fftypes.JSONAny, method *core.FFIMethod, input map[string]interface{}, options map[string]interface{}) (interface{}, error) {
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
	parsed.Address, err = validateEthAddress(ctx, parsed.Address)
	if err != nil {
		return nil, err
	}
	normalized, err := json.Marshal(parsed)
	if err == nil {
		result = fftypes.JSONAnyPtrBytes(normalized)
	}
	return result, err
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

func (e *Ethereum) AddContractListener(ctx context.Context, listener *core.ContractListenerInput) error {
	location, err := parseContractLocation(ctx, listener.Location)
	if err != nil {
		return err
	}
	abi, err := e.FFIEventDefinitionToABI(ctx, &listener.Event.FFIEventDefinition)
	if err != nil {
		return i18n.WrapError(ctx, err, coremsgs.MsgContractParamInvalid)
	}

	subName := fmt.Sprintf("ff-sub-%s", listener.ID)
	result, err := e.streams.createSubscription(ctx, location, e.streamID, subName, listener.Options.FirstEvent, abi)
	if err != nil {
		return err
	}
	listener.BackendID = result.ID
	return nil
}

func (e *Ethereum) DeleteContractListener(ctx context.Context, subscription *core.ContractListener) error {
	return e.streams.deleteSubscription(ctx, subscription.BackendID)
}

func (e *Ethereum) GetFFIParamValidator(ctx context.Context) (core.FFIParamValidator, error) {
	return &FFIParamValidator{}, nil
}

func (e *Ethereum) FFIEventDefinitionToABI(ctx context.Context, event *core.FFIEventDefinition) (*abi.Entry, error) {
	abiInputs, err := e.convertFFIParamsToABIParameters(ctx, event.Params)
	if err != nil {
		return nil, err
	}
	abiEntry := &abi.Entry{
		Name:   event.Name,
		Type:   "event",
		Inputs: abiInputs,
	}
	if event.Details != nil {
		abiEntry.Anonymous = event.Details.GetBool("anonymous")
	}
	return abiEntry, nil
}

func (e *Ethereum) FFIMethodToABI(ctx context.Context, method *core.FFIMethod, input map[string]interface{}) (*abi.Entry, error) {
	abiInputs, err := e.convertFFIParamsToABIParameters(ctx, method.Params)
	if err != nil {
		return nil, err
	}

	abiOutputs, err := e.convertFFIParamsToABIParameters(ctx, method.Returns)
	if err != nil {
		return nil, err
	}
	abiEntry := &abi.Entry{
		Name:    method.Name,
		Type:    "function",
		Inputs:  abiInputs,
		Outputs: abiOutputs,
	}
	if method.Details != nil {
		if stateMutability, ok := method.Details.GetStringOk("stateMutability"); ok {
			abiEntry.StateMutability = abi.StateMutability(stateMutability)
		}
		abiEntry.Payable = method.Details.GetBool("payable")
		abiEntry.Constant = method.Details.GetBool("constant")
	}
	return abiEntry, nil
}

func ABIArgumentToTypeString(typeName string, components abi.ParameterArray) string {
	if strings.HasPrefix(typeName, "tuple") {
		suffix := typeName[5:]
		children := make([]string, len(components))
		for i, component := range components {
			children[i] = ABIArgumentToTypeString(component.Type, nil)
		}
		return "(" + strings.Join(children, ",") + ")" + suffix
	}
	return typeName
}

func ABIMethodToSignature(abi *abi.Entry) string {
	result := abi.Name + "("
	if len(abi.Inputs) > 0 {
		types := make([]string, len(abi.Inputs))
		for i, param := range abi.Inputs {
			types[i] = ABIArgumentToTypeString(param.Type, param.Components)
		}
		result += strings.Join(types, ",")
	}
	result += ")"
	return result
}

func (e *Ethereum) GenerateEventSignature(ctx context.Context, event *core.FFIEventDefinition) string {
	abi, err := e.FFIEventDefinitionToABI(ctx, event)
	if err != nil {
		return ""
	}
	return ABIMethodToSignature(abi)
}

func (e *Ethereum) convertFFIParamsToABIParameters(ctx context.Context, params core.FFIParams) (abi.ParameterArray, error) {
	abiParamList := make(abi.ParameterArray, len(params))
	for i, param := range params {
		c := core.NewFFISchemaCompiler()
		v, _ := e.GetFFIParamValidator(ctx)
		c.RegisterExtension(v.GetExtensionName(), v.GetMetaSchema(), v)
		err := c.AddResource(param.Name, strings.NewReader(param.Schema.String()))
		if err != nil {
			return nil, err
		}
		s, err := c.Compile(param.Name)
		if err != nil {
			return nil, err
		}
		abiParamList[i] = processField(param.Name, s)
	}
	return abiParamList, nil
}

func processField(name string, schema *jsonschema.Schema) *abi.Parameter {
	details := getParamDetails(schema)
	parameter := &abi.Parameter{
		Name:         name,
		Type:         details.Type,
		InternalType: details.InternalType,
		Indexed:      details.Indexed,
	}
	if len(schema.Types) > 0 {
		switch schema.Types[0] {
		case objectType:
			parameter.Components = buildABIParameterArrayForObject(schema.Properties)
		case arrayType:
			parameter.Components = buildABIParameterArrayForObject(schema.Items2020.Properties)
		}
	}
	return parameter
}

func buildABIParameterArrayForObject(properties map[string]*jsonschema.Schema) abi.ParameterArray {
	parameters := make(abi.ParameterArray, len(properties))
	for propertyName, propertySchema := range properties {
		details := getParamDetails(propertySchema)
		parameter := processField(propertyName, propertySchema)
		parameters[*details.Index] = parameter
	}
	return parameters
}

func getParamDetails(schema *jsonschema.Schema) *paramDetails {
	ext := schema.Extensions["details"]
	details := ext.(detailsSchema)
	blockchainType := details["type"].(string)
	paramDetails := &paramDetails{
		Type: blockchainType,
	}
	if i, ok := details["index"]; ok {
		index, _ := i.(json.Number).Int64()
		paramDetails.Index = new(int)
		*paramDetails.Index = int(index)
	}
	if i, ok := details["indexed"]; ok {
		paramDetails.Indexed = i.(bool)
	}
	if i, ok := details["internalType"]; ok {
		paramDetails.InternalType = i.(string)
	}
	return paramDetails
}

func (e *Ethereum) prepareRequest(ctx context.Context, method *core.FFIMethod, input map[string]interface{}) (*abi.Entry, []interface{}, error) {
	orderedInput := make([]interface{}, len(method.Params))
	abi, err := e.FFIMethodToABI(ctx, method, input)
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

func (e *Ethereum) GenerateFFI(ctx context.Context, generationRequest *core.FFIGenerationRequest) (*core.FFI, error) {
	var input FFIGenerationInput
	err := json.Unmarshal(generationRequest.Input.Bytes(), &input)
	if err != nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgFFIGenerationFailed, "unable to deserialize JSON as ABI")
	}
	if len(*input.ABI) == 0 {
		return nil, i18n.NewError(ctx, coremsgs.MsgFFIGenerationFailed, "ABI is empty")
	}
	return e.convertABIToFFI(ctx, generationRequest.Namespace, generationRequest.Name, generationRequest.Version, generationRequest.Description, input.ABI)
}

func (e *Ethereum) convertABIToFFI(ctx context.Context, ns, name, version, description string, abi *abi.ABI) (*core.FFI, error) {
	ffi := &core.FFI{
		Namespace:   ns,
		Name:        name,
		Version:     version,
		Description: description,
		Methods:     make([]*core.FFIMethod, len(abi.Functions())),
		Events:      make([]*core.FFIEvent, len(abi.Events())),
	}
	i := 0
	for _, f := range abi.Functions() {
		method, err := e.convertABIFunctionToFFIMethod(ctx, f)
		if err != nil {
			return nil, err
		}
		ffi.Methods[i] = method
		i++
	}
	i = 0
	for _, f := range abi.Events() {
		event, err := e.convertABIEventToFFIEvent(ctx, f)
		if err != nil {
			return nil, err
		}
		ffi.Events[i] = event
		i++
	}
	return ffi, nil
}

func (e *Ethereum) convertABIFunctionToFFIMethod(ctx context.Context, abiFunction *abi.Entry) (*core.FFIMethod, error) {
	params := make([]*core.FFIParam, len(abiFunction.Inputs))
	returns := make([]*core.FFIParam, len(abiFunction.Outputs))
	details := map[string]interface{}{}
	for i, input := range abiFunction.Inputs {
		typeComponent, err := input.TypeComponentTreeCtx(ctx)
		if err != nil {
			return nil, err
		}
		schema := e.getSchemaForABIInput(ctx, typeComponent)
		param := &core.FFIParam{
			Name:   input.Name,
			Schema: fftypes.JSONAnyPtr(schema.ToJSON()),
		}
		params[i] = param
	}
	for i, output := range abiFunction.Outputs {
		typeComponent, err := output.TypeComponentTreeCtx(ctx)
		if err != nil {
			return nil, err
		}
		schema := e.getSchemaForABIInput(ctx, typeComponent)
		param := &core.FFIParam{
			Name:   output.Name,
			Schema: fftypes.JSONAnyPtr(schema.ToJSON()),
		}
		returns[i] = param
	}
	if abiFunction.StateMutability != "" {
		details["stateMutability"] = string(abiFunction.StateMutability)
	}
	if abiFunction.Payable {
		details["payable"] = true
	}
	if abiFunction.Constant {
		details["constant"] = true
	}
	return &core.FFIMethod{
		Name:    abiFunction.Name,
		Params:  params,
		Returns: returns,
		Details: details,
	}, nil
}

func (e *Ethereum) convertABIEventToFFIEvent(ctx context.Context, abiEvent *abi.Entry) (*core.FFIEvent, error) {
	params := make([]*core.FFIParam, len(abiEvent.Inputs))
	details := map[string]interface{}{}
	for i, output := range abiEvent.Inputs {
		typeComponent, err := output.TypeComponentTreeCtx(ctx)
		if err != nil {
			return nil, err
		}
		schema := e.getSchemaForABIInput(ctx, typeComponent)
		param := &core.FFIParam{
			Name:   output.Name,
			Schema: fftypes.JSONAnyPtr(schema.ToJSON()),
		}
		params[i] = param
	}
	if abiEvent.Anonymous {
		details["anonymous"] = true
	}
	return &core.FFIEvent{
		FFIEventDefinition: core.FFIEventDefinition{
			Name:    abiEvent.Name,
			Params:  params,
			Details: details,
		},
	}, nil
}

func (e *Ethereum) getSchemaForABIInput(ctx context.Context, typeComponent abi.TypeComponent) *Schema {
	schema := &Schema{
		Details: &paramDetails{
			Type:         typeComponent.Parameter().Type,
			InternalType: typeComponent.Parameter().InternalType,
			Indexed:      typeComponent.Parameter().Indexed,
		},
	}
	switch typeComponent.ComponentType() {
	case abi.ElementaryComponent:
		t := e.getFFIType(typeComponent.ElementaryType().String())
		if t == core.FFIInputTypeInteger {
			schema.OneOf = []SchemaType{
				{Type: "string"},
				{Type: "integer"},
			}
			schema.Description = i18n.Expand(ctx, coremsgs.APIIntegerDescription)
		} else {
			schema.Type = t.String()
		}
	case abi.FixedArrayComponent, abi.DynamicArrayComponent:
		schema.Type = arrayType
		childSchema := e.getSchemaForABIInput(ctx, typeComponent.ArrayChild())
		schema.Items = childSchema
		schema.Details = childSchema.Details
		childSchema.Details = nil
	case abi.TupleComponent:
		schema.Type = objectType
		schema.Properties = make(map[string]*Schema, len(typeComponent.TupleChildren()))
		for i, tupleChild := range typeComponent.TupleChildren() {
			childSchema := e.getSchemaForABIInput(ctx, tupleChild)
			childSchema.Details.Index = new(int)
			*childSchema.Details.Index = i
			schema.Properties[tupleChild.KeyName()] = childSchema
		}
	}
	return schema
}

func (e *Ethereum) getFFIType(solidityType string) core.FFIInputType {
	switch solidityType {
	case stringType, "address":
		return core.FFIInputTypeString
	case boolType:
		return core.FFIInputTypeBoolean
	case tupleType:
		return core.FFIInputTypeObject
	default:
		switch {
		case strings.Contains(solidityType, "byte"):
			return core.FFIInputTypeString
		case strings.Contains(solidityType, "int"):
			return core.FFIInputTypeInteger
		}
	}
	return ""
}

func (e *Ethereum) getNetworkVersion(ctx context.Context, address string) (int, error) {
	res, err := e.queryContractMethod(ctx, address, networkVersionMethodABI, []interface{}{}, nil)
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

func (e *Ethereum) NetworkVersion(ctx context.Context) int {
	e.fireflyContract.mux.Lock()
	defer e.fireflyContract.mux.Unlock()
	return e.fireflyContract.networkVersion
}
