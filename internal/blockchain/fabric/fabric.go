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

package fabric

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
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
)

const (
	broadcastBatchEventName = "BatchPin"
)

type Fabric struct {
	ctx            context.Context
	topic          string
	defaultChannel string
	chaincode      string
	signer         string
	prefixShort    string
	prefixLong     string
	capabilities   *blockchain.Capabilities
	callbacks      blockchain.Callbacks
	client         *resty.Client
	streams        *streamManager
	initInfo       struct {
		stream *eventStream
		sub    *subscription
	}
	idCache map[string]*fabIdentity
	wsconn  wsclient.WSClient
	closed  chan struct{}
}

type eventStreamWebsocket struct {
	Topic string `json:"topic"`
}

type fabTxInputHeaders struct {
	ID            string         `json:"id,omitempty"`
	Type          string         `json:"type"`
	PayloadSchema *PayloadSchema `json:"payloadSchema,omitempty"`
	Signer        string         `json:"signer,omitempty"`
	Channel       string         `json:"channel,omitempty"`
	Chaincode     string         `json:"chaincode,omitempty"`
}

type PayloadSchema struct {
	Type        string        `json:"type"`
	PrefixItems []*PrefixItem `json:"prefixItems"`
}

type PrefixItem struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type fabTxNamedInput struct {
	Headers *fabTxInputHeaders `json:"headers"`
	Func    string             `json:"func"`
	Args    map[string]string  `json:"args"`
}

type fabQueryNamedOutput struct {
	Headers *fabTxInputHeaders `json:"headers"`
	Result  interface{}        `json:"result"`
}

type ffiParamSchema struct {
	Type string `json:"type,omitempty"`
}

type fabWSCommandPayload struct {
	Type  string `json:"type"`
	Topic string `json:"topic,omitempty"`
}

type fabIdentity struct {
	MSPID  string `json:"mspId"`
	ECert  string `json:"enrollmentCert"`
	CACert string `json:"caCert"`
}

type Location struct {
	Channel   string `json:"channel"`
	Chaincode string `json:"chaincode"`
}

var batchPinEvent = "BatchPin"
var batchPinMethodName = "PinBatch"
var batchPinPrefixItems = []*PrefixItem{
	{
		Name: "namespace",
		Type: "string",
	},
	{
		Name: "uuids",
		Type: "string",
	},
	{
		Name: "batchHash",
		Type: "string",
	},
	{
		Name: "payloadRef",
		Type: "string",
	},
	{
		Name: "contexts",
		Type: "string",
	},
}

var fullIdentityPattern = regexp.MustCompile(".+::x509::(.+)::.+")

var cnPattern = regexp.MustCompile("CN=([^,]+)")

func (f *Fabric) Name() string {
	return "fabric"
}

func (f *Fabric) VerifierType() fftypes.VerifierType {
	return fftypes.VerifierTypeMSPIdentity
}

func (f *Fabric) Init(ctx context.Context, prefix config.Prefix, callbacks blockchain.Callbacks) (err error) {

	fabconnectConf := prefix.SubPrefix(FabconnectConfigKey)

	f.ctx = log.WithLogField(ctx, "proto", "fabric")
	f.callbacks = callbacks
	f.idCache = make(map[string]*fabIdentity)

	if fabconnectConf.GetString(restclient.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "url", "blockchain.fabconnect")
	}
	f.defaultChannel = fabconnectConf.GetString(FabconnectConfigDefaultChannel)
	f.chaincode = fabconnectConf.GetString(FabconnectConfigChaincode)
	if f.chaincode == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "chaincode", "blockchain.fabconnect")
	}
	// the org identity is guaranteed to be configured by the core
	f.signer = fabconnectConf.GetString(FabconnectConfigSigner)
	f.topic = fabconnectConf.GetString(FabconnectConfigTopic)
	if f.topic == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "topic", "blockchain.fabconnect")
	}

	f.prefixShort = fabconnectConf.GetString(FabconnectPrefixShort)
	f.prefixLong = fabconnectConf.GetString(FabconnectPrefixLong)

	f.client = restclient.New(f.ctx, fabconnectConf)
	f.capabilities = &blockchain.Capabilities{
		GlobalSequencer: true,
	}

	wsConfig := wsconfig.GenerateConfigFromPrefix(fabconnectConf)

	if wsConfig.WSKeyPath == "" {
		wsConfig.WSKeyPath = "/ws"
	}

	f.wsconn, err = wsclient.New(ctx, wsConfig, nil, f.afterConnect)
	if err != nil {
		return err
	}

	f.streams = &streamManager{
		client: f.client,
		signer: f.signer,
	}
	batchSize := fabconnectConf.GetUint(FabconnectConfigBatchSize)
	batchTimeout := uint(fabconnectConf.GetDuration(FabconnectConfigBatchTimeout).Milliseconds())
	if f.initInfo.stream, err = f.streams.ensureEventStream(f.ctx, f.topic, batchSize, batchTimeout); err != nil {
		return err
	}
	log.L(f.ctx).Infof("Event stream: %s", f.initInfo.stream.ID)
	location := &Location{
		Channel:   f.defaultChannel,
		Chaincode: f.chaincode,
	}
	if f.initInfo.sub, err = f.streams.ensureSubscription(f.ctx, location, f.initInfo.stream.ID, batchPinEvent); err != nil {
		return err
	}

	f.closed = make(chan struct{})
	go f.eventLoop()

	return nil
}

func (f *Fabric) Start() error {
	return f.wsconn.Connect()
}

func (f *Fabric) Capabilities() *blockchain.Capabilities {
	return f.capabilities
}

func (f *Fabric) afterConnect(ctx context.Context, w wsclient.WSClient) error {
	// Send a subscribe to our topic after each connect/reconnect
	b, _ := json.Marshal(&fabWSCommandPayload{
		Type:  "listen",
		Topic: f.topic,
	})
	err := w.Send(ctx, b)
	if err == nil {
		b, _ = json.Marshal(&fabWSCommandPayload{
			Type: "listenreplies",
		})
		err = w.Send(ctx, b)
	}
	return err
}

func decodeJSONPayload(ctx context.Context, payloadString string) *fftypes.JSONObject {
	bytes, err := base64.StdEncoding.DecodeString(payloadString)
	if err != nil {
		log.L(ctx).Errorf("BatchPin event is not valid - bad payload content: %s", payloadString)
		return nil
	}
	dataBytes := fftypes.JSONAnyPtrBytes(bytes)
	payload, ok := dataBytes.JSONObjectOk()
	if !ok {
		log.L(ctx).Errorf("BatchPin event is not valid - bad JSON payload: %s", bytes)
		return nil
	}
	return &payload
}

func (f *Fabric) handleBatchPinEvent(ctx context.Context, msgJSON fftypes.JSONObject) (err error) {
	payloadString := msgJSON.GetString("payload")
	payload := decodeJSONPayload(ctx, payloadString)
	if payload == nil {
		return nil // move on
	}

	sTransactionHash := msgJSON.GetString("transactionId")
	blockNumber := msgJSON.GetInt64("blockNumber")
	transactionIndex := msgJSON.GetInt64("transactionIndex")
	eventIndex := msgJSON.GetInt64("eventIndex")
	timestamp := msgJSON.GetInt64("timestamp")
	signer := payload.GetString("signer")
	ns := payload.GetString("namespace")
	sUUIDs := payload.GetString("uuids")
	sBatchHash := payload.GetString("batchHash")
	sPayloadRef := payload.GetString("payloadRef")
	sContexts := payload.GetStringArray("contexts")

	hexUUIDs, err := hex.DecodeString(strings.TrimPrefix(sUUIDs, "0x"))
	if err != nil || len(hexUUIDs) != 32 {
		log.L(ctx).Errorf("BatchPin event is not valid - bad uuids (%s): %s", sUUIDs, err)
		return nil // move on
	}
	var txnID fftypes.UUID
	copy(txnID[:], hexUUIDs[0:16])
	var batchID fftypes.UUID
	copy(batchID[:], hexUUIDs[16:32])

	var batchHash fftypes.Bytes32
	err = batchHash.UnmarshalText([]byte(sBatchHash))
	if err != nil {
		log.L(ctx).Errorf("BatchPin event is not valid - bad batchHash (%s): %s", sBatchHash, err)
		return nil // move on
	}

	contexts := make([]*fftypes.Bytes32, len(sContexts))
	for i, sHash := range sContexts {
		var hash fftypes.Bytes32
		err = hash.UnmarshalText([]byte(sHash))
		if err != nil {
			log.L(ctx).Errorf("BatchPin event is not valid - bad pin %d (%s): %s", i, sHash, err)
			return nil // move on
		}
		contexts[i] = &hash
	}

	delete(msgJSON, "payload")
	batch := &blockchain.BatchPin{
		Namespace:       ns,
		TransactionID:   &txnID,
		BatchID:         &batchID,
		BatchHash:       &batchHash,
		BatchPayloadRef: sPayloadRef,
		Contexts:        contexts,
		Event: blockchain.Event{
			BlockchainTXID: sTransactionHash,
			Source:         f.Name(),
			Name:           "BatchPin",
			ProtocolID:     fmt.Sprintf("%.12d/%.6d/%.6d", blockNumber, transactionIndex, eventIndex),
			Output:         *payload,
			Info:           msgJSON,
			Timestamp:      fftypes.UnixTime(timestamp),
		},
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return f.callbacks.BatchPinComplete(batch, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeMSPIdentity,
		Value: signer,
	})
}

func (f *Fabric) handleContractEvent(ctx context.Context, msgJSON fftypes.JSONObject) (err error) {
	payloadString := msgJSON.GetString("payload")
	payload := decodeJSONPayload(ctx, payloadString)
	if payload == nil {
		return nil // move on
	}
	delete(msgJSON, "payload")

	sTransactionHash := msgJSON.GetString("transactionId")
	blockNumber := msgJSON.GetInt64("blockNumber")
	transactionIndex := msgJSON.GetInt64("transactionIndex")
	eventIndex := msgJSON.GetInt64("eventIndex")
	sub := msgJSON.GetString("subId")
	name := msgJSON.GetString("eventName")
	sTimestamp := msgJSON.GetString("timestamp")
	timestamp, err := strconv.ParseInt(sTimestamp, 10, 64)
	if err != nil {
		log.L(ctx).Errorf("Blockchain event is not valid - bad timestamp (%s): %s", sTimestamp, err)
		// Continue with zero timestamp
	}

	event := &blockchain.EventWithSubscription{
		Subscription: sub,
		Event: blockchain.Event{
			BlockchainTXID: sTransactionHash,
			Source:         f.Name(),
			Name:           name,
			ProtocolID:     fmt.Sprintf("%.12d/%.6d/%.6d", blockNumber, transactionIndex, eventIndex),
			Output:         *payload,
			Info:           msgJSON,
			Timestamp:      fftypes.UnixTime(timestamp),
		},
	}

	return f.callbacks.BlockchainEvent(event)
}

func (f *Fabric) handleReceipt(ctx context.Context, reply fftypes.JSONObject) error {
	l := log.L(ctx)

	headers := reply.GetObject("headers")
	requestID := headers.GetString("requestId")
	replyType := headers.GetString("type")
	txHash := reply.GetString("transactionId")
	message := reply.GetString("errorMessage")
	if requestID == "" || replyType == "" {
		l.Errorf("Reply cannot be processed: %+v", reply)
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
	l.Infof("Fabconnect '%s' reply tx=%s (request=%s) %s", replyType, txHash, requestID, message)
	return f.callbacks.BlockchainOpUpdate(operationID, updateType, txHash, message, reply)
}

func (f *Fabric) handleMessageBatch(ctx context.Context, messages []interface{}) error {
	l := log.L(ctx)

	for i, msgI := range messages {
		msgMap, ok := msgI.(map[string]interface{})
		if !ok {
			l.Errorf("Message cannot be parsed as JSON: %+v", msgI)
			return nil // Swallow this and move on
		}
		msgJSON := fftypes.JSONObject(msgMap)

		l1 := l.WithField("fabmsgidx", i)
		ctx1 := log.WithLogger(ctx, l1)
		eventName := msgJSON.GetString("eventName")
		sub := msgJSON.GetString("subId")
		l1.Infof("Received '%s' message", eventName)
		l1.Tracef("Message: %+v", msgJSON)

		if sub == f.initInfo.sub.ID {
			switch eventName {
			case broadcastBatchEventName:
				if err := f.handleBatchPinEvent(ctx1, msgJSON); err != nil {
					return err
				}
			default:
				l.Infof("Ignoring event with unknown name: %s", eventName)
			}
		} else if err := f.handleContractEvent(ctx, msgJSON); err != nil {
			return err
		}
	}

	return nil
}

func (f *Fabric) eventLoop() {
	defer f.wsconn.Close()
	defer close(f.closed)
	l := log.L(f.ctx).WithField("role", "event-loop")
	ctx := log.WithLogger(f.ctx, l)
	ack, _ := json.Marshal(map[string]string{"type": "ack", "topic": f.topic})
	for {
		select {
		case <-ctx.Done():
			l.Debugf("Event loop exiting (context cancelled)")
			return
		case msgBytes, ok := <-f.wsconn.Receive():
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
				err = f.handleMessageBatch(ctx, msgTyped)
				if err == nil {
					err = f.wsconn.Send(ctx, ack)
				}
			case map[string]interface{}:
				err = f.handleReceipt(ctx, fftypes.JSONObject(msgTyped))
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

func (f *Fabric) NormalizeSigningKey(ctx context.Context, signingKeyInput string) (string, error) {
	// we expand the short user name into the fully qualified onchain identity:
	// mspid::x509::{ecert DN}::{CA DN}	return signingKeyInput, nil
	if !fullIdentityPattern.MatchString(signingKeyInput) {
		existingID := f.idCache[signingKeyInput]
		if existingID == nil {
			var idRes fabIdentity
			res, err := f.client.R().SetContext(f.ctx).SetResult(&idRes).Get(fmt.Sprintf("/identities/%s", signingKeyInput))
			if err != nil || !res.IsSuccess() {
				return "", i18n.NewError(f.ctx, i18n.MsgFabconnectRESTErr, err)
			}
			f.idCache[signingKeyInput] = &idRes
			existingID = &idRes
		}

		ecertDN, err := getDNFromCertString(existingID.ECert)
		if err != nil {
			return "", i18n.NewError(f.ctx, i18n.MsgFailedToDecodeCertificate, err)
		}
		cacertDN, err := getDNFromCertString(existingID.CACert)
		if err != nil {
			return "", i18n.NewError(f.ctx, i18n.MsgFailedToDecodeCertificate, err)
		}
		resolvedSigningKey := fmt.Sprintf("%s::x509::%s::%s", existingID.MSPID, ecertDN, cacertDN)
		log.L(f.ctx).Debugf("Resolved signing key: %s", resolvedSigningKey)
		return resolvedSigningKey, nil
	}
	return signingKeyInput, nil
}

func (f *Fabric) invokeContractMethod(ctx context.Context, channel, chaincode, methodName, signingKey, requestID string, prefixItems []*PrefixItem, input map[string]string) (*resty.Response, error) {
	in := &fabTxNamedInput{
		Headers: &fabTxInputHeaders{
			ID: requestID,
			PayloadSchema: &PayloadSchema{
				Type:        "array",
				PrefixItems: prefixItems,
			},
			Channel:   channel,
			Chaincode: chaincode,
			Signer:    getUserName(signingKey),
		},
		Func: methodName,
		Args: input,
	}

	return f.client.R().
		SetContext(ctx).
		SetBody(in).
		Post("/transactions")
}

func getUserName(fullIDString string) string {
	matches := fullIdentityPattern.FindStringSubmatch(fullIDString)
	if len(matches) == 0 {
		return fullIDString
	}
	matches = cnPattern.FindStringSubmatch(matches[1])
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func hexFormatB32(b *fftypes.Bytes32) string {
	if b == nil {
		return "0x0000000000000000000000000000000000000000000000000000000000000000"
	}
	return "0x" + hex.EncodeToString(b[0:32])
}

func (f *Fabric) SubmitBatchPin(ctx context.Context, operationID *fftypes.UUID, ledgerID *fftypes.UUID, signingKey string, batch *blockchain.BatchPin) error {
	hashes := make([]string, len(batch.Contexts))
	for i, v := range batch.Contexts {
		hashes[i] = hexFormatB32(v)
	}
	var uuids fftypes.Bytes32
	copy(uuids[0:16], (*batch.TransactionID)[:])
	copy(uuids[16:32], (*batch.BatchID)[:])
	pinInput := map[string]interface{}{
		"namespace":  batch.Namespace,
		"uuids":      hexFormatB32(&uuids),
		"batchHash":  hexFormatB32(batch.BatchHash),
		"payloadRef": batch.BatchPayloadRef,
		"contexts":   hashes,
	}

	input, _ := jsonEncodeInput(pinInput)

	res, err := f.invokeContractMethod(ctx, f.defaultChannel, f.chaincode, batchPinMethodName, signingKey, operationID.String(), batchPinPrefixItems, input)
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgFabconnectRESTErr)
	}
	return nil
}

func (f *Fabric) InvokeContract(ctx context.Context, operationID *fftypes.UUID, signingKey string, location *fftypes.JSONAny, method *fftypes.FFIMethod, input map[string]interface{}) error {
	// All arguments must be JSON serialized
	args, err := jsonEncodeInput(input)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgJSONObjectParseFailed, "params")
	}

	fabricOnChainLocation, err := parseContractLocation(ctx, location)
	if err != nil {
		return err
	}

	// Build the payload schema for the method parameters
	prefixItems := make([]*PrefixItem, len(method.Params))
	for i, param := range method.Params {
		var paramSchema ffiParamSchema
		if err := json.Unmarshal(param.Schema.Bytes(), &paramSchema); err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgJSONObjectParseFailed, fmt.Sprintf("%s.schema", param.Name))
		}

		prefixItems[i] = &PrefixItem{
			Name: param.Name,
			Type: paramSchema.Type,
		}
	}

	res, err := f.invokeContractMethod(ctx, fabricOnChainLocation.Channel, fabricOnChainLocation.Chaincode, method.Name, signingKey, operationID.String(), prefixItems, args)

	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgFabconnectRESTErr)
	}
	return nil
}

func (f *Fabric) QueryContract(ctx context.Context, location *fftypes.JSONAny, method *fftypes.FFIMethod, input map[string]interface{}) (interface{}, error) {
	// All arguments must be JSON serialized
	args, err := jsonEncodeInput(input)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgJSONObjectParseFailed, "params")
	}

	fabricOnChainLocation, err := parseContractLocation(ctx, location)
	if err != nil {
		return nil, err
	}

	// Build the payload schema for the method parameters
	prefixItems := make([]*PrefixItem, len(method.Params))
	for i, param := range method.Params {
		prefixItems[i] = &PrefixItem{
			Name: param.Name,
			Type: "string",
		}
	}

	in := &fabTxNamedInput{
		Headers: &fabTxInputHeaders{
			PayloadSchema: &PayloadSchema{
				Type:        "array",
				PrefixItems: prefixItems,
			},
			Channel:   fabricOnChainLocation.Channel,
			Chaincode: fabricOnChainLocation.Chaincode,
			Signer:    f.signer,
		},
		Func: method.Name,
		Args: args,
	}

	res, err := f.client.R().
		SetContext(ctx).
		SetBody(in).
		Post("/query")

	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(ctx, res, err, i18n.MsgFabconnectRESTErr)
	}
	output := &fabQueryNamedOutput{}
	if err = json.Unmarshal(res.Body(), output); err != nil {
		return nil, err
	}
	return output.Result, nil
}

func jsonEncodeInput(params map[string]interface{}) (output map[string]string, err error) {
	output = make(map[string]string, len(params))
	for field, value := range params {
		switch v := value.(type) {
		case string:
			output[field] = v
		default:
			encodedValue, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			output[field] = string(encodedValue)
		}

	}
	return
}

func (f *Fabric) ValidateContractLocation(ctx context.Context, location *fftypes.JSONAny) (err error) {
	_, err = parseContractLocation(ctx, location)
	return
}

func parseContractLocation(ctx context.Context, location *fftypes.JSONAny) (*Location, error) {
	fabricLocation := Location{}
	if err := json.Unmarshal(location.Bytes(), &fabricLocation); err != nil {
		return nil, i18n.NewError(ctx, i18n.MsgContractLocationInvalid, err)
	}
	if fabricLocation.Channel == "" {
		return nil, i18n.NewError(ctx, i18n.MsgContractLocationInvalid, "'channel' not set")
	}
	if fabricLocation.Chaincode == "" {
		return nil, i18n.NewError(ctx, i18n.MsgContractLocationInvalid, "'chaincode' not set")
	}
	return &fabricLocation, nil
}

func (f *Fabric) AddSubscription(ctx context.Context, subscription *fftypes.ContractListenerInput) error {
	location, err := parseContractLocation(ctx, subscription.Location)
	if err != nil {
		return err
	}
	result, err := f.streams.createSubscription(ctx, location, f.initInfo.stream.ID, "", subscription.Event.Name)
	if err != nil {
		return err
	}
	subscription.ProtocolID = result.ID
	return nil
}

func (f *Fabric) DeleteSubscription(ctx context.Context, subscription *fftypes.ContractListener) error {
	return f.streams.deleteSubscription(ctx, subscription.ProtocolID)
}

func (f *Fabric) GetFFIParamValidator(ctx context.Context) (fftypes.FFIParamValidator, error) {
	// Fabconnect does not require any additional validation beyond "JSON Schema correctness" at this time
	return nil, nil
}

func (f *Fabric) GenerateFFI(ctx context.Context, generationRequest *fftypes.FFIGenerationRequest) (*fftypes.FFI, error) {
	return nil, i18n.NewError(ctx, i18n.MsgFFIGenerationUnsupported)
}
