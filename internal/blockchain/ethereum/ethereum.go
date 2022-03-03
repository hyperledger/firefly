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
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/config/wsconfig"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/restclient"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/wsclient"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

const (
	broadcastBatchEventSignature = "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])"
	booleanType                  = "boolean"
	integerType                  = "integer"
	stringType                   = "string"
	arrayType                    = "array"
	objectType                   = "object"
)

type Ethereum struct {
	ctx          context.Context
	topic        string
	instancePath string
	prefixShort  string
	prefixLong   string
	capabilities *blockchain.Capabilities
	callbacks    blockchain.Callbacks
	client       *resty.Client
	streams      *streamManager
	initInfo     struct {
		stream *eventStream
		sub    *subscription
	}
	wsconn          wsclient.WSClient
	closed          chan struct{}
	addressResolver *addressResolver
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
	Type       string             `json:"type"`
	Details    *paramDetails      `json:"details,omitempty"`
	Properties map[string]*Schema `json:"properties,omitempty"`
	Items      *Schema            `json:"items,omitempty"`
}

func (s *Schema) ToJSON() string {
	b, _ := json.Marshal(s)
	return string(b)
}

// ABIArgumentMarshaling is abi.ArgumentMarshaling
type ABIArgumentMarshaling struct {
	Name         string                  `json:"name"`
	Type         string                  `json:"type"`
	InternalType string                  `json:"internalType,omitempty"`
	Components   []ABIArgumentMarshaling `json:"components,omitempty"`
	Indexed      bool                    `json:"indexed,omitempty"`
}

// ABIElementMarshaling is the serialized representation of a method or event in an ABI
type ABIElementMarshaling struct {
	Type            string                  `json:"type,omitempty"`
	Name            string                  `json:"name,omitempty"`
	Payable         bool                    `json:"payable,omitempty"`
	Constant        bool                    `json:"constant,omitempty"`
	Anonymous       bool                    `json:"anonymous,omitempty"`
	StateMutability string                  `json:"stateMutability,omitempty"`
	Inputs          []ABIArgumentMarshaling `json:"inputs"`
	Outputs         []ABIArgumentMarshaling `json:"outputs"`
}

type EthconnectMessageRequest struct {
	Headers EthconnectMessageHeaders `json:"headers,omitempty"`
	To      string                   `json:"to"`
	From    string                   `json:"from,omitempty"`
	Method  ABIElementMarshaling     `json:"method"`
	Params  []interface{}            `json:"params"`
}

type EthconnectMessageHeaders struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
}

// var batchPinEvent = "BatchPin"
var addressVerify = regexp.MustCompile("^[0-9a-f]{40}$")

func (e *Ethereum) Name() string {
	return "ethereum"
}

func (e *Ethereum) VerifierType() fftypes.VerifierType {
	return fftypes.VerifierTypeEthAddress
}

func (e *Ethereum) Init(ctx context.Context, prefix config.Prefix, callbacks blockchain.Callbacks) (err error) {

	ethconnectConf := prefix.SubPrefix(EthconnectConfigKey)
	addressResolverConf := prefix.SubPrefix(AddressResolverConfigKey)

	e.ctx = log.WithLogField(ctx, "proto", "ethereum")
	e.callbacks = callbacks

	if addressResolverConf.GetString(AddressResolverURLTemplate) != "" {
		if e.addressResolver, err = newAddressResolver(ctx, addressResolverConf); err != nil {
			return err
		}
	}

	if ethconnectConf.GetString(restclient.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "url", "blockchain.ethconnect")
	}

	e.client = restclient.New(e.ctx, ethconnectConf)
	e.capabilities = &blockchain.Capabilities{
		GlobalSequencer: true,
	}

	e.instancePath = ethconnectConf.GetString(EthconnectConfigInstancePath)
	if e.instancePath == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "instance", "blockchain.ethconnect")
	}

	// Backwards compatibility from when instance path was not a contract address
	if strings.HasPrefix(strings.ToLower(e.instancePath), "/contracts/") {
		address, err := e.getContractAddress(ctx, e.instancePath)
		if err != nil {
			return err
		}
		e.instancePath = address
	} else if strings.HasPrefix(e.instancePath, "/instances/") {
		e.instancePath = strings.Replace(e.instancePath, "/instances/", "", 1)
	}

	// Ethconnect needs the "0x" prefix in some cases
	if !strings.HasPrefix(e.instancePath, "0x") {
		e.instancePath = fmt.Sprintf("0x%s", e.instancePath)
	}

	e.topic = ethconnectConf.GetString(EthconnectConfigTopic)
	if e.topic == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "topic", "blockchain.ethconnect")
	}

	e.prefixShort = ethconnectConf.GetString(EthconnectPrefixShort)
	e.prefixLong = ethconnectConf.GetString(EthconnectPrefixLong)

	wsConfig := wsconfig.GenerateConfigFromPrefix(ethconnectConf)

	if wsConfig.WSKeyPath == "" {
		wsConfig.WSKeyPath = "/ws"
	}

	e.wsconn, err = wsclient.New(ctx, wsConfig, nil, e.afterConnect)
	if err != nil {
		return err
	}

	e.streams = &streamManager{
		client: e.client,
	}
	batchSize := ethconnectConf.GetUint(EthconnectConfigBatchSize)
	batchTimeout := uint(ethconnectConf.GetDuration(EthconnectConfigBatchTimeout).Milliseconds())
	if e.initInfo.stream, err = e.streams.ensureEventStream(e.ctx, e.topic, batchSize, batchTimeout); err != nil {
		return err
	}
	log.L(e.ctx).Infof("Event stream: %s (topic=%s)", e.initInfo.stream.ID, e.topic)
	if e.initInfo.sub, err = e.streams.ensureSubscription(e.ctx, e.instancePath, e.initInfo.stream.ID, batchPinEventABI); err != nil {
		return err
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

func (e *Ethereum) handleBatchPinEvent(ctx context.Context, msgJSON fftypes.JSONObject) (err error) {
	sBlockNumber := msgJSON.GetString("blockNumber")
	sTransactionHash := msgJSON.GetString("transactionHash")
	blockNumber := msgJSON.GetInt64("blockNumber")
	txIndex := msgJSON.GetInt64("transactionIndex")
	logIndex := msgJSON.GetInt64("logIndex")
	dataJSON := msgJSON.GetObject("data")
	authorAddress := dataJSON.GetString("author")
	ns := dataJSON.GetString("namespace")
	sUUIDs := dataJSON.GetString("uuids")
	sBatchHash := dataJSON.GetString("batchHash")
	sPayloadRef := dataJSON.GetString("payloadRef")
	sContexts := dataJSON.GetStringArray("contexts")
	timestampStr := msgJSON.GetString("timestamp")
	timestamp, err := fftypes.ParseTimeString(timestampStr)
	if err != nil {
		log.L(ctx).Errorf("BatchPin event is not valid - missing timestamp: %+v", msgJSON)
		return nil // move on
	}

	if sBlockNumber == "" ||
		sTransactionHash == "" ||
		authorAddress == "" ||
		sUUIDs == "" ||
		sBatchHash == "" {
		log.L(ctx).Errorf("BatchPin event is not valid - missing data: %+v", msgJSON)
		return nil // move on
	}

	authorAddress, err = e.NormalizeSigningKey(ctx, authorAddress)
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

	delete(msgJSON, "data")
	batch := &blockchain.BatchPin{
		Namespace:       ns,
		TransactionID:   &txnID,
		BatchID:         &batchID,
		BatchHash:       &batchHash,
		BatchPayloadRef: sPayloadRef,
		Contexts:        contexts,
		Event: blockchain.Event{
			BlockchainTXID: sTransactionHash,
			Source:         e.Name(),
			Name:           "BatchPin",
			ProtocolID:     fmt.Sprintf("%.12d/%.6d/%.6d", blockNumber, txIndex, logIndex),
			Output:         dataJSON,
			Info:           msgJSON,
			Timestamp:      timestamp,
		},
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return e.callbacks.BatchPinComplete(batch, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeEthAddress,
		Value: authorAddress,
	})
}

func (e *Ethereum) handleContractEvent(ctx context.Context, msgJSON fftypes.JSONObject) (err error) {
	sTransactionHash := msgJSON.GetString("transactionHash")
	blockNumber := msgJSON.GetInt64("blockNumber")
	txIndex := msgJSON.GetInt64("transactionIndex")
	logIndex := msgJSON.GetInt64("logIndex")
	sub := msgJSON.GetString("subId")
	signature := msgJSON.GetString("signature")
	dataJSON := msgJSON.GetObject("data")
	name := strings.SplitN(signature, "(", 2)[0]
	timestampStr := msgJSON.GetString("timestamp")
	timestamp, err := fftypes.ParseTimeString(timestampStr)
	if err != nil {
		log.L(ctx).Errorf("Contract event is not valid - missing timestamp: %+v", msgJSON)
		return err // move on
	}
	delete(msgJSON, "data")

	event := &blockchain.EventWithSubscription{
		Subscription: sub,
		Event: blockchain.Event{
			BlockchainTXID: sTransactionHash,
			Source:         e.Name(),
			Name:           name,
			ProtocolID:     fmt.Sprintf("%.12d/%.6d/%.6d", blockNumber, txIndex, logIndex),
			Output:         dataJSON,
			Info:           msgJSON,
			Timestamp:      timestamp,
		},
	}

	return e.callbacks.BlockchainEvent(event)
}

func (e *Ethereum) handleReceipt(ctx context.Context, reply fftypes.JSONObject) error {
	l := log.L(ctx)

	headers := reply.GetObject("headers")
	requestID := headers.GetString("requestId")
	replyType := headers.GetString("type")
	txHash := reply.GetString("transactionHash")
	message := reply.GetString("errorMessage")
	if requestID == "" || replyType == "" {
		l.Errorf("Reply cannot be processed - missing fields: %+v", reply)
		return nil // Swallow this and move on
	}
	operationID, err := fftypes.ParseUUID(ctx, requestID)
	if err != nil {
		l.Errorf("Reply cannot be processed - bad ID: %+v", reply)
		return nil // Swallow this and move on
	}
	updateType := fftypes.OpStatusSucceeded
	if replyType != "TransactionSuccess" {
		updateType = fftypes.OpStatusFailed
	}
	l.Infof("Ethconnect '%s' reply: request=%s tx=%s message=%s", replyType, requestID, txHash, message)
	return e.callbacks.BlockchainOpUpdate(operationID, updateType, txHash, message, reply)
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

		if sub == e.initInfo.sub.ID {
			switch signature {
			case broadcastBatchEventSignature:
				if err := e.handleBatchPinEvent(ctx1, msgJSON); err != nil {
					return err
				}
			default:
				l.Infof("Ignoring event with unknown signature: %s", signature)
			}
		} else if err := e.handleContractEvent(ctx1, msgJSON); err != nil {
			return err
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

func validateEthAddress(ctx context.Context, key string) (string, error) {
	keyLower := strings.ToLower(key)
	keyNoHexPrefix := strings.TrimPrefix(keyLower, "0x")
	if addressVerify.MatchString(keyNoHexPrefix) {
		return "0x" + keyNoHexPrefix, nil
	}
	return "", i18n.NewError(ctx, i18n.MsgInvalidEthAddress)
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

func (e *Ethereum) invokeContractMethod(ctx context.Context, address, signingKey string, abi ABIElementMarshaling, requestID string, input []interface{}) (*resty.Response, error) {
	body := EthconnectMessageRequest{
		Headers: EthconnectMessageHeaders{
			Type: "SendTransaction",
			ID:   requestID,
		},
		From:   signingKey,
		To:     address,
		Method: abi,
		Params: input,
	}
	return e.client.R().
		SetContext(ctx).
		SetBody(body).
		Post("/")
}

func (e *Ethereum) queryContractMethod(ctx context.Context, address string, abi ABIElementMarshaling, input []interface{}) (*resty.Response, error) {
	body := EthconnectMessageRequest{
		Headers: EthconnectMessageHeaders{
			Type: "Query",
		},
		To:     address,
		Method: abi,
		Params: input,
	}
	return e.client.R().
		SetContext(ctx).
		SetBody(body).
		Post("/")
}

func (e *Ethereum) SubmitBatchPin(ctx context.Context, operationID *fftypes.UUID, ledgerID *fftypes.UUID, signingKey string, batch *blockchain.BatchPin) error {
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
	res, err := e.invokeContractMethod(ctx, e.instancePath, signingKey, batchPinMethodABI, operationID.String(), input)
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return nil
}

func (e *Ethereum) InvokeContract(ctx context.Context, operationID *fftypes.UUID, signingKey string, location *fftypes.JSONAny, method *fftypes.FFIMethod, input map[string]interface{}) error {
	ethereumLocation, err := parseContractLocation(ctx, location)
	if err != nil {
		return err
	}
	abi, orderedInput, err := e.prepareRequest(ctx, method, input)
	if err != nil {
		return err
	}
	res, err := e.invokeContractMethod(ctx, ethereumLocation.Address, signingKey, abi, operationID.String(), orderedInput)
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return nil
}

func (e *Ethereum) QueryContract(ctx context.Context, location *fftypes.JSONAny, method *fftypes.FFIMethod, input map[string]interface{}) (interface{}, error) {
	ethereumLocation, err := parseContractLocation(ctx, location)
	if err != nil {
		return nil, err
	}
	abi, orderedInput, err := e.prepareRequest(ctx, method, input)
	if err != nil {
		return nil, err
	}
	res, err := e.queryContractMethod(ctx, ethereumLocation.Address, abi, orderedInput)
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	output := &queryOutput{}
	if err = json.Unmarshal(res.Body(), output); err != nil {
		return nil, err
	}
	return output, nil
}

func (e *Ethereum) ValidateContractLocation(ctx context.Context, location *fftypes.JSONAny) (err error) {
	_, err = parseContractLocation(ctx, location)
	return
}

func parseContractLocation(ctx context.Context, location *fftypes.JSONAny) (*Location, error) {
	ethLocation := Location{}
	if err := json.Unmarshal(location.Bytes(), &ethLocation); err != nil {
		return nil, i18n.NewError(ctx, i18n.MsgContractLocationInvalid, err)
	}
	if ethLocation.Address == "" {
		return nil, i18n.NewError(ctx, i18n.MsgContractLocationInvalid, "'address' not set")
	}
	return &ethLocation, nil
}

func (e *Ethereum) AddSubscription(ctx context.Context, subscription *fftypes.ContractSubscriptionInput) error {
	location, err := parseContractLocation(ctx, subscription.Location)
	if err != nil {
		return err
	}
	abi, err := e.FFIEventDefinitionToABI(ctx, &subscription.Event.FFIEventDefinition)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgContractParamInvalid)
	}

	subName := fmt.Sprintf("ff-sub-%s", subscription.ID)
	result, err := e.streams.createSubscription(ctx, location, e.initInfo.stream.ID, subName, abi)
	if err != nil {
		return err
	}
	subscription.ProtocolID = result.ID
	return nil
}

func (e *Ethereum) DeleteSubscription(ctx context.Context, subscription *fftypes.ContractSubscription) error {
	return e.streams.deleteSubscription(ctx, subscription.ProtocolID)
}

func (e *Ethereum) GetFFIParamValidator(ctx context.Context) (fftypes.FFIParamValidator, error) {
	return &FFIParamValidator{}, nil
}

func (e *Ethereum) FFIEventDefinitionToABI(ctx context.Context, event *fftypes.FFIEventDefinition) (ABIElementMarshaling, error) {
	abiElement := ABIElementMarshaling{
		Name:   event.Name,
		Type:   "event",
		Inputs: make([]ABIArgumentMarshaling, len(event.Params)),
	}

	if err := e.addParamsToList(ctx, abiElement.Inputs, event.Params); err != nil {
		return abiElement, err
	}
	return abiElement, nil
}

func (e *Ethereum) FFIMethodToABI(ctx context.Context, method *fftypes.FFIMethod) (ABIElementMarshaling, error) {
	abiElement := ABIElementMarshaling{
		Name:    method.Name,
		Type:    "function",
		Inputs:  make([]ABIArgumentMarshaling, len(method.Params)),
		Outputs: make([]ABIArgumentMarshaling, len(method.Returns)),
	}

	if err := e.addParamsToList(ctx, abiElement.Inputs, method.Params); err != nil {
		return abiElement, err
	}
	if err := e.addParamsToList(ctx, abiElement.Outputs, method.Returns); err != nil {
		return abiElement, err
	}

	return abiElement, nil
}

func (e *Ethereum) addParamsToList(ctx context.Context, abiParamList []ABIArgumentMarshaling, params fftypes.FFIParams) error {
	for i, param := range params {
		c := fftypes.NewFFISchemaCompiler()
		v, _ := e.GetFFIParamValidator(ctx)
		c.RegisterExtension(v.GetExtensionName(), v.GetMetaSchema(), v)
		err := c.AddResource(param.Name, strings.NewReader(param.Schema.String()))
		if err != nil {
			return err
		}
		s, err := c.Compile(param.Name)
		if err != nil {
			return err
		}
		abiParamList[i] = processField(param.Name, s)
	}
	return nil
}

func processField(name string, schema *jsonschema.Schema) ABIArgumentMarshaling {
	details := getParamDetails(schema)
	arg := ABIArgumentMarshaling{
		Name:         name,
		Type:         details.Type,
		InternalType: details.InternalType,
		Indexed:      details.Indexed,
	}
	if schema.Types[0] == objectType {
		arg.Components = buildABIArgumentArray(schema.Properties)
	}
	return arg
}

func buildABIArgumentArray(properties map[string]*jsonschema.Schema) []ABIArgumentMarshaling {
	args := make([]ABIArgumentMarshaling, len(properties))
	for propertyName, propertySchema := range properties {
		details := getParamDetails(propertySchema)
		arg := processField(propertyName, propertySchema)
		args[*details.Index] = arg
	}
	return args
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

func (e *Ethereum) prepareRequest(ctx context.Context, method *fftypes.FFIMethod, input map[string]interface{}) (ABIElementMarshaling, []interface{}, error) {
	orderedInput := make([]interface{}, len(method.Params))
	abi, err := e.FFIMethodToABI(ctx, method)
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
		return "", restclient.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	var output map[string]string
	if err = json.Unmarshal(res.Body(), &output); err != nil {
		return "", err
	}
	return output["address"], nil
}

func (e *Ethereum) GenerateFFI(ctx context.Context, generationRequest *fftypes.FFIGenerationRequest) (*fftypes.FFI, error) {
	var abi []ABIElementMarshaling
	err := json.Unmarshal(generationRequest.Input.Bytes(), &abi)
	if err != nil {
		return nil, i18n.NewError(ctx, i18n.MsgFFIGenerationFailed, "unable to deserialize JSON as ABI")
	}
	ffi := e.convertABIToFFI(generationRequest.Namespace, generationRequest.Name, generationRequest.Version, generationRequest.Description, abi)
	return ffi, nil
}

func (e *Ethereum) convertABIToFFI(ns, name, version, description string, abi []ABIElementMarshaling) *fftypes.FFI {
	ffi := &fftypes.FFI{
		Namespace:   ns,
		Name:        name,
		Version:     version,
		Description: description,
		Methods:     []*fftypes.FFIMethod{},
		Events:      []*fftypes.FFIEvent{},
	}

	for _, element := range abi {
		switch element.Type {
		case "event":
			event := &fftypes.FFIEvent{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name:   element.Name,
					Params: e.convertABIArgumentsToFFI(element.Inputs),
				},
			}
			ffi.Events = append(ffi.Events, event)
		case "function":
			method := &fftypes.FFIMethod{
				Name:    element.Name,
				Params:  e.convertABIArgumentsToFFI(element.Inputs),
				Returns: e.convertABIArgumentsToFFI(element.Outputs),
			}
			ffi.Methods = append(ffi.Methods, method)
		}
	}
	return ffi
}

func (e *Ethereum) convertABIArgumentsToFFI(args []ABIArgumentMarshaling) fftypes.FFIParams {
	ffiParams := fftypes.FFIParams{}
	for _, arg := range args {
		param := &fftypes.FFIParam{
			Name: arg.Name,
		}
		s := e.getSchema(arg)
		param.Schema = fftypes.JSONAnyPtr(s.ToJSON())
		ffiParams = append(ffiParams, param)
	}
	return ffiParams
}

func (e *Ethereum) getSchema(arg ABIArgumentMarshaling) *Schema {
	s := &Schema{
		Type: e.getFFIType(arg.Type),
		Details: &paramDetails{
			Type:         arg.Type,
			InternalType: arg.InternalType,
			Indexed:      arg.Indexed,
		},
	}
	var properties map[string]*Schema
	if len(arg.Components) > 0 {
		properties = e.getSchemaForObjectComponents(arg)
	}
	if s.Type == arrayType {
		levels := strings.Count(arg.Type, "[]")
		innerType := e.getFFIType(strings.ReplaceAll(arg.Type, "[]", ""))
		innerSchema := &Schema{
			Type: innerType,
		}
		if len(arg.Components) > 0 {
			innerSchema.Properties = e.getSchemaForObjectComponents(arg)
		}
		for i := 1; i < levels; i++ {
			innerSchema = &Schema{
				Type:  arrayType,
				Items: innerSchema,
			}
		}
		s.Items = innerSchema
	} else {
		s.Properties = properties
	}
	return s
}

func (e *Ethereum) getSchemaForObjectComponents(arg ABIArgumentMarshaling) map[string]*Schema {
	m := make(map[string]*Schema, len(arg.Components))
	for i, component := range arg.Components {
		componentSchema := e.getSchema(component)
		componentSchema.Details.Index = new(int)
		*componentSchema.Details.Index = i
		m[component.Name] = componentSchema
	}
	return m
}

func (e *Ethereum) getFFIType(solitidyType string) string {

	switch solitidyType {
	case stringType, "address":
		return stringType
	case "bool":
		return booleanType
	case "tuple":
		return objectType
	default:
		switch {
		case strings.HasSuffix(solitidyType, "[]"):
			return arrayType
		case strings.Contains(solitidyType, "byte"):
			return stringType
		case strings.Contains(solitidyType, "int"):
			return integerType
		}
	}
	return ""
}
