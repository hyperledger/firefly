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

package ethereum

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

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
	"github.com/hyperledger/firefly/internal/blockchain/common"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/sirupsen/logrus"
)

const (
	broadcastBatchEventSignature = "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])"
)

const (
	ethTxStatusPending string = "Pending"
)

const (
	ReceiptTransactionSuccess string = "TransactionSuccess"
	ReceiptTransactionFailed  string = "TransactionFailed"
)

type Ethereum struct {
	ctx                  context.Context
	cancelCtx            context.CancelFunc
	topic                string
	prefixShort          string
	prefixLong           string
	capabilities         *blockchain.Capabilities
	callbacks            common.BlockchainCallbacks
	client               *resty.Client
	streams              *streamManager
	streamID             string
	wsconn               wsclient.WSClient
	closed               chan struct{}
	addressResolveAlways bool
	addressResolver      *addressResolver
	metrics              metrics.Manager
	ethconnectConf       config.Section
	subs                 common.FireflySubscriptions
	cache                cache.CInterface
	backgroundRetry      *retry.Retry
	backgroundStart      bool
}

type eventStreamWebsocket struct {
	Topic string `json:"topic"`
}

type queryOutput struct {
	Output interface{} `json:"output"`
}

type ethWSCommandPayload struct {
	Type        string `json:"type"`
	Topic       string `json:"topic,omitempty"`
	BatchNumber int64  `json:"batchNumber,omitempty"`
	Message     string `json:"message,omitempty"`
}

type Location struct {
	Address string `json:"address"`
}

type ListenerCheckpoint struct {
	Block            int64 `json:"block"`
	TransactionIndex int64 `json:"transactionIndex"`
	LogIndex         int64 `json:"logIndex"`
}

type ListenerStatus struct {
	Checkpoint ListenerCheckpoint `json:"checkpoint"`
	Catchup    bool               `json:"catchup"`
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

func (e *Ethereum) Init(ctx context.Context, cancelCtx context.CancelFunc, conf config.Section, metrics metrics.Manager, cacheManager cache.Manager) (err error) {
	e.InitConfig(conf)
	ethconnectConf := e.ethconnectConf
	addressResolverConf := conf.SubSection(AddressResolverConfigKey)

	e.ctx = log.WithLogField(ctx, "proto", "ethereum")
	e.cancelCtx = cancelCtx
	e.metrics = metrics
	e.capabilities = &blockchain.Capabilities{}
	e.callbacks = common.NewBlockchainCallbacks()
	e.subs = common.NewFireflySubscriptions()

	if addressResolverConf.GetString(AddressResolverURLTemplate) != "" {
		// Check if we need to invoke the address resolver (without caching) on every call
		e.addressResolveAlways = addressResolverConf.GetBool(AddressResolverAlwaysResolve)
		if e.addressResolver, err = newAddressResolver(ctx, addressResolverConf, cacheManager, !e.addressResolveAlways); err != nil {
			return err
		}
	}

	if ethconnectConf.GetString(ffresty.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, coremsgs.MsgMissingPluginConfig, "url", ethconnectConf)
	}

	wsConfig, err := wsclient.GenerateConfig(ctx, ethconnectConf)
	if err == nil {
		e.client, err = ffresty.New(e.ctx, ethconnectConf)
	}

	if err != nil {
		return err
	}

	e.topic = ethconnectConf.GetString(EthconnectConfigTopic)
	if e.topic == "" {
		return i18n.NewError(ctx, coremsgs.MsgMissingPluginConfig, "topic", ethconnectConf)
	}
	e.prefixShort = ethconnectConf.GetString(EthconnectPrefixShort)
	e.prefixLong = ethconnectConf.GetString(EthconnectPrefixLong)

	if wsConfig.WSKeyPath == "" {
		wsConfig.WSKeyPath = "/ws"
	}
	e.wsconn, err = wsclient.New(ctx, wsConfig, nil, e.afterConnect)
	if err != nil {
		return err
	}

	cache, err := cacheManager.GetCache(
		cache.NewCacheConfig(
			ctx,
			coreconfig.CacheBlockchainLimit,
			coreconfig.CacheBlockchainTTL,
			"",
		),
	)
	if err != nil {
		return err
	}
	e.cache = cache

	e.streams = newStreamManager(e.client, e.cache, e.ethconnectConf.GetUint(EthconnectConfigBatchSize), uint(e.ethconnectConf.GetDuration(EthconnectConfigBatchTimeout).Milliseconds()))

	e.backgroundStart = e.ethconnectConf.GetBool(EthconnectBackgroundStart)
	if e.backgroundStart {
		e.backgroundRetry = &retry.Retry{
			InitialDelay: e.ethconnectConf.GetDuration(EthconnectBackgroundStartInitialDelay),
			MaximumDelay: e.ethconnectConf.GetDuration(EthconnectBackgroundStartMaxDelay),
			Factor:       e.ethconnectConf.GetFloat64(EthconnectBackgroundStartFactor),
		}

		return nil
	}

	stream, err := e.streams.ensureEventStream(e.ctx, e.topic)
	if err != nil {
		return err
	}

	e.streamID = stream.ID
	log.L(e.ctx).Infof("Event stream: %s (topic=%s)", e.streamID, e.topic)

	e.closed = make(chan struct{})
	go e.eventLoop()

	return nil
}

func (e *Ethereum) SetHandler(namespace string, handler blockchain.Callbacks) {
	e.callbacks.SetHandler(namespace, handler)
}

func (e *Ethereum) SetOperationHandler(namespace string, handler core.OperationCallbacks) {
	e.callbacks.SetOperationalHandler(namespace, handler)
}

func (e *Ethereum) startBackgroundLoop() {
	_ = e.backgroundRetry.Do(e.ctx, fmt.Sprintf("ethereum connector %s", e.Name()), func(attempt int) (retry bool, err error) {
		stream, err := e.streams.ensureEventStream(e.ctx, e.topic)
		if err != nil {
			return true, err
		}

		e.streamID = stream.ID
		log.L(e.ctx).Infof("Event stream: %s (topic=%s)", e.streamID, e.topic)
		err = e.wsconn.Connect()
		if err != nil {
			return true, err
		}

		e.closed = make(chan struct{})
		go e.eventLoop()

		return false, nil
	})
}

func (e *Ethereum) Start() (err error) {
	if e.backgroundStart {
		go e.startBackgroundLoop()
		return nil
	}

	return e.wsconn.Connect()
}

func (e *Ethereum) Capabilities() *blockchain.Capabilities {
	return e.capabilities
}

func (e *Ethereum) AddFireflySubscription(ctx context.Context, namespace *core.Namespace, contract *blockchain.MultipartyContract) (string, error) {
	ethLocation, err := e.parseContractLocation(ctx, contract.Location)
	if err != nil {
		return "", err
	}

	version, err := e.GetNetworkVersion(ctx, contract.Location)
	if err != nil {
		return "", err
	}

	sub, err := e.streams.ensureFireFlySubscription(ctx, namespace.Name, version, ethLocation.Address, contract.FirstEvent, e.streamID, batchPinEventABI)
	if err != nil {
		return "", err
	}

	e.subs.AddSubscription(ctx, namespace, version, sub.ID, nil)
	return sub.ID, nil
}

func (e *Ethereum) RemoveFireflySubscription(ctx context.Context, subID string) {
	// Don't actually delete the subscription from ethconnect, as this may be called while processing
	// events from the subscription (and handling that scenario cleanly could be difficult for ethconnect).
	// TODO: can old subscriptions be somehow cleaned up later?
	e.subs.RemoveSubscription(ctx, subID)
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

func (e *Ethereum) processBatchPinEvent(ctx context.Context, events common.EventsToDispatch, location *fftypes.JSONAny, subInfo *common.SubscriptionInfo, msgJSON fftypes.JSONObject) {
	event := e.parseBlockchainEvent(ctx, msgJSON)
	if event == nil {
		return // move on
	}

	authorAddress := event.Output.GetString("author")
	nsOrAction := event.Output.GetString("action")
	if nsOrAction == "" {
		nsOrAction = event.Output.GetString("namespace")
	}

	params := &common.BatchPinParams{
		UUIDs:      event.Output.GetString("uuids"),
		BatchHash:  event.Output.GetString("batchHash"),
		PayloadRef: event.Output.GetString("payloadRef"),
		Contexts:   event.Output.GetStringArray("contexts"),
		NsOrAction: nsOrAction,
	}

	// Validate the ethereum address - it must already be a valid address, we do not
	// engage the address resolve on this blockchain-driven path.
	authorAddress, err := formatEthAddress(ctx, authorAddress)
	if err != nil {
		log.L(ctx).Errorf("BatchPin event is not valid - bad from address (%s): %+v", err, msgJSON)
		return // move on
	}
	verifier := &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: authorAddress,
	}

	e.callbacks.PrepareBatchPinOrNetworkAction(ctx, events, subInfo, location, event, verifier, params)
}

func (e *Ethereum) processContractEvent(ctx context.Context, events common.EventsToDispatch, msgJSON fftypes.JSONObject) error {
	subID := msgJSON.GetString("subId")
	subName, err := e.streams.getSubscriptionName(ctx, subID)
	if err != nil {
		return err // this is a problem - we should be able to find the listener that dispatched this to us
	}

	namespace := common.GetNamespaceFromSubName(subName)
	event := e.parseBlockchainEvent(ctx, msgJSON)
	if event != nil {
		e.callbacks.PrepareBlockchainEvent(ctx, events, namespace, &blockchain.EventForListener{
			Event:      event,
			ListenerID: subID,
		})
	}
	return nil
}

func (e *Ethereum) buildEventLocationString(msgJSON fftypes.JSONObject) string {
	return fmt.Sprintf("address=%s", msgJSON.GetString("address"))
}

func (e *Ethereum) handleMessageBatch(ctx context.Context, batchID int64, messages []interface{}) error {
	// Build the set of events that need handling
	events := make(common.EventsToDispatch)
	count := len(messages)
	for i, msgI := range messages {
		msgMap, ok := msgI.(map[string]interface{})
		if !ok {
			log.L(ctx).Errorf("Message cannot be parsed as JSON: %+v", msgI)
			return nil // Swallow this and move on
		}
		msgJSON := fftypes.JSONObject(msgMap)

		signature := msgJSON.GetString("signature")
		sub := msgJSON.GetString("subId")
		logger := log.L(ctx)
		logger.Infof("[EVM:%d:%d/%d]: '%s' on '%s'", batchID, i+1, count, signature, sub)
		logger.Tracef("Message: %+v", msgJSON)

		// Matches one of the active FireFly BatchPin subscriptions
		if subInfo := e.subs.GetSubscription(sub); subInfo != nil {
			location, err := e.encodeContractLocation(ctx, &Location{
				Address: msgJSON.GetString("address"),
			})
			if err != nil {
				return err
			}

			firstColon := strings.Index(signature, ":")
			if firstColon >= 0 {
				signature = signature[firstColon+1:]
			}
			switch signature {
			case broadcastBatchEventSignature:
				e.processBatchPinEvent(ctx, events, location, subInfo, msgJSON)
			default:
				log.L(ctx).Infof("Ignoring event with unknown signature: %s", signature)
			}
		} else {
			// Subscription not recognized - assume it's from a custom contract listener
			// (event manager will reject it if it's not)
			if err := e.processContractEvent(ctx, events, msgJSON); err != nil {
				return err
			}
		}
	}
	// Dispatch all the events from this patch that were successfully parsed and routed to namespaces
	// (could be zero - that's ok)
	return e.callbacks.DispatchBlockchainEvents(ctx, events)
}

func (e *Ethereum) eventLoop() {
	defer e.wsconn.Close()
	defer close(e.closed)
	l := log.L(e.ctx).WithField("role", "event-loop")
	ctx := log.WithLogger(e.ctx, l)
	for {
		select {
		case <-ctx.Done():
			l.Debugf("Event loop exiting (context cancelled)")
			return
		case msgBytes, ok := <-e.wsconn.Receive():
			if !ok {
				l.Debugf("Event loop exiting (receive channel closed). Terminating server!")
				e.cancelCtx()
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
				err = e.handleMessageBatch(ctx, 0, msgTyped)
				if err == nil {
					ack, _ := json.Marshal(&ethWSCommandPayload{
						Type:  "ack",
						Topic: e.topic,
					})
					err = e.wsconn.Send(ctx, ack)
				}
			case map[string]interface{}:
				isBatch := false
				if batchNumber, ok := msgTyped["batchNumber"].(float64); ok {
					if events, ok := msgTyped["events"].([]interface{}); ok {
						// FFTM delivery with a batch number to use in the ack
						isBatch = true
						err = e.handleMessageBatch(ctx, (int64)(batchNumber), events)
						// Errors processing messages are converted into nacks
						ackOrNack := &ethWSCommandPayload{
							Topic:       e.topic,
							BatchNumber: int64(batchNumber),
						}
						if err == nil {
							ackOrNack.Type = "ack"
						} else {
							log.L(ctx).Errorf("Rejecting batch due error: %s", err)
							ackOrNack.Type = "error"
							ackOrNack.Message = err.Error()
						}
						b, _ := json.Marshal(&ackOrNack)
						err = e.wsconn.Send(ctx, b)
					}
				}
				if !isBatch {
					var receipt common.BlockchainReceiptNotification
					_ = json.Unmarshal(msgBytes, &receipt)
					err := common.HandleReceipt(ctx, e, &receipt, e.callbacks)
					if err != nil {
						l.Errorf("Failed to process receipt: %+v", msgTyped)
					}
				}
			default:
				l.Errorf("Message unexpected: %+v", msgTyped)
				continue
			}

			if err != nil {
				l.Errorf("Event loop exiting (%s). Terminating server!", err)
				e.cancelCtx()
				return
			}
		}
	}
}

func formatEthAddress(ctx context.Context, key string) (string, error) {
	keyLower := strings.ToLower(key)
	keyNoHexPrefix := strings.TrimPrefix(keyLower, "0x")
	if addressVerify.MatchString(keyNoHexPrefix) {
		return "0x" + keyNoHexPrefix, nil
	}
	return "", i18n.NewError(ctx, coremsgs.MsgInvalidEthAddress)
}

func (e *Ethereum) ResolveSigningKey(ctx context.Context, key string, intent blockchain.ResolveKeyIntent) (resolved string, err error) {
	if !e.addressResolveAlways {
		// If there's no address resolver plugin, or addressResolveAlways is false,
		// we check if it's already an ethereum address - in which case we can just return it.
		resolved, err = formatEthAddress(ctx, key)
	}
	if e.addressResolveAlways || (err != nil && e.addressResolver != nil) {
		// Either it's not a valid ethereum address,
		// or we've been configured to invoke the address resolver on every call
		resolved, err = e.addressResolver.ResolveSigningKey(ctx, key, intent)
		if err == nil {
			log.L(ctx).Infof("Key '%s' resolved to '%s'", key, resolved)
			return resolved, nil
		}
	}
	return resolved, err
}

func (e *Ethereum) buildEthconnectRequestBody(ctx context.Context, messageType, address, signingKey string, abi *abi.Entry, requestID string, input []interface{}, errors []*abi.Entry, options map[string]interface{}) (map[string]interface{}, error) {
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
	if len(errors) > 0 {
		body["errors"] = errors
	}
	finalBody, err := e.applyOptions(ctx, body, options)
	if err != nil {
		return nil, err
	}
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		jsonBody, _ := json.Marshal(finalBody)
		log.L(ctx).Debugf("EVMConnectorBody: %s", string(jsonBody))
	}
	return finalBody, nil
}

func (e *Ethereum) applyOptions(ctx context.Context, body, options map[string]interface{}) (map[string]interface{}, error) {
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

func (e *Ethereum) invokeContractMethod(ctx context.Context, address, signingKey string, abi *abi.Entry, requestID string, input []interface{}, errors []*abi.Entry, options map[string]interface{}) (submissionRejected bool, err error) {
	if e.metrics.IsMetricsEnabled() {
		e.metrics.BlockchainTransaction(address, abi.Name)
	}
	messageType := "SendTransaction"
	body, err := e.buildEthconnectRequestBody(ctx, messageType, address, signingKey, abi, requestID, input, errors, options)
	if err != nil {
		return true, err
	}
	var resErr common.BlockchainRESTError
	res, err := e.client.R().
		SetContext(ctx).
		SetBody(body).
		SetError(&resErr).
		Post("/")
	if err != nil || !res.IsSuccess() {
		return resErr.SubmissionRejected, common.WrapRESTError(ctx, &resErr, res, err, coremsgs.MsgEthConnectorRESTErr)
	}
	return false, nil
}

func (e *Ethereum) queryContractMethod(ctx context.Context, address, signingKey string, abi *abi.Entry, input []interface{}, errors []*abi.Entry, options map[string]interface{}) (*resty.Response, error) {
	if e.metrics.IsMetricsEnabled() {
		e.metrics.BlockchainQuery(address, abi.Name)
	}
	messageType := "Query"
	body, err := e.buildEthconnectRequestBody(ctx, messageType, address, signingKey, abi, "", input, errors, options)
	if err != nil {
		return nil, err
	}
	var resErr common.BlockchainRESTError
	res, err := e.client.R().
		SetContext(ctx).
		SetBody(body).
		SetError(&resErr).
		Post("/")
	if err != nil || !res.IsSuccess() {
		return res, common.WrapRESTError(ctx, &resErr, res, err, coremsgs.MsgEthConnectorRESTErr)
	}
	return res, nil
}

func (e *Ethereum) buildBatchPinInput(ctx context.Context, version int, namespace string, batch *blockchain.BatchPin) (*abi.Entry, []interface{}) {
	ethHashes := make([]string, len(batch.Contexts))
	for i, v := range batch.Contexts {
		ethHashes[i] = ethHexFormatB32(v)
	}
	var uuids fftypes.Bytes32
	copy(uuids[0:16], (*batch.TransactionID)[:])
	copy(uuids[16:32], (*batch.BatchID)[:])

	var input []interface{}
	var method *abi.Entry

	if version == 1 {
		method = batchPinMethodABIV1
		input = []interface{}{
			namespace,
			ethHexFormatB32(&uuids),
			ethHexFormatB32(batch.BatchHash),
			batch.BatchPayloadRef,
			ethHashes,
		}
	} else {
		method = batchPinMethodABI
		input = []interface{}{
			ethHexFormatB32(&uuids),
			ethHexFormatB32(batch.BatchHash),
			batch.BatchPayloadRef,
			ethHashes,
		}
	}

	return method, input
}

func (e *Ethereum) SubmitBatchPin(ctx context.Context, nsOpID, networkNamespace, signingKey string, batch *blockchain.BatchPin, location *fftypes.JSONAny) error {
	ethLocation, err := e.parseContractLocation(ctx, location)
	if err != nil {
		return err
	}

	version, err := e.GetNetworkVersion(ctx, location)
	if err != nil {
		return err
	}

	method, input := e.buildBatchPinInput(ctx, version, networkNamespace, batch)

	var emptyErrors []*abi.Entry
	_, err = e.invokeContractMethod(ctx, ethLocation.Address, signingKey, method, nsOpID, input, emptyErrors, nil)
	return err
}

func (e *Ethereum) SubmitNetworkAction(ctx context.Context, nsOpID string, signingKey string, action core.NetworkActionType, location *fftypes.JSONAny) error {
	ethLocation, err := e.parseContractLocation(ctx, location)
	if err != nil {
		return err
	}

	version, err := e.GetNetworkVersion(ctx, location)
	if err != nil {
		return err
	}

	var input []interface{}
	var method *abi.Entry

	if version == 1 {
		method = batchPinMethodABIV1
		input = []interface{}{
			blockchain.FireFlyActionPrefix + action,
			ethHexFormatB32(nil),
			ethHexFormatB32(nil),
			"",
			[]string{},
		}
	} else {
		method = networkActionMethodABI
		input = []interface{}{
			blockchain.FireFlyActionPrefix + action,
			"",
		}
	}
	var emptyErrors []*abi.Entry
	_, err = e.invokeContractMethod(ctx, ethLocation.Address, signingKey, method, nsOpID, input, emptyErrors, nil)
	return err
}

func (e *Ethereum) DeployContract(ctx context.Context, nsOpID, signingKey string, definition, contract *fftypes.JSONAny, input []interface{}, options map[string]interface{}) (submissionRejected bool, err error) {
	if e.metrics.IsMetricsEnabled() {
		e.metrics.BlockchainContractDeployment()
	}
	headers := EthconnectMessageHeaders{
		Type: "DeployContract",
		ID:   nsOpID,
	}
	body := map[string]interface{}{
		"headers":    headers,
		"from":       signingKey,
		"params":     input,
		"definition": definition,
		"contract":   contract,
	}
	if signingKey != "" {
		body["from"] = signingKey
	}
	body, err = e.applyOptions(ctx, body, options)
	if err != nil {
		return true, err
	}

	var resErr common.BlockchainRESTError
	res, err := e.client.R().
		SetContext(ctx).
		SetBody(body).
		SetError(&resErr).
		Post("/")
	if err != nil || !res.IsSuccess() {
		if strings.Contains(string(res.Body()), "FFEC100130") {
			// This error is returned by ethconnect because it does not support deploying contracts with this syntax
			// Return a more helpful and clear error message
			return true, i18n.NewError(ctx, coremsgs.MsgNotSupportedByBlockchainPlugin)
		}
		return resErr.SubmissionRejected, common.WrapRESTError(ctx, &resErr, res, err, coremsgs.MsgEthConnectorRESTErr)
	}
	return false, nil
}

// Check if a method supports passing extra data via conformance to ERC5750.
// That is, check if the last method input is a "bytes" parameter.
func (e *Ethereum) checkDataSupport(ctx context.Context, method *abi.Entry) error {
	if len(method.Inputs) > 0 {
		lastParam := method.Inputs[len(method.Inputs)-1]
		if lastParam.Type == "bytes" {
			return nil
		}
	}
	return i18n.NewError(ctx, coremsgs.MsgMethodDoesNotSupportPinning)
}

func (e *Ethereum) ValidateInvokeRequest(ctx context.Context, parsedMethod interface{}, input map[string]interface{}, hasMessage bool) error {
	methodInfo, _, err := e.prepareRequest(ctx, parsedMethod, input)
	if err == nil && hasMessage {
		if err = e.checkDataSupport(ctx, methodInfo.methodABI); err != nil {
			return err
		}
	}
	return err
}

func (e *Ethereum) InvokeContract(ctx context.Context, nsOpID string, signingKey string, location *fftypes.JSONAny, parsedMethod interface{}, input map[string]interface{}, options map[string]interface{}, batch *blockchain.BatchPin) (bool, error) {
	ethereumLocation, err := e.parseContractLocation(ctx, location)
	if err != nil {
		return true, err
	}
	methodInfo, orderedInput, err := e.prepareRequest(ctx, parsedMethod, input)
	if err != nil {
		return true, err
	}
	if batch != nil {
		err := e.checkDataSupport(ctx, methodInfo.methodABI)
		if err == nil {
			method, batchPin := e.buildBatchPinInput(ctx, 2, "", batch)
			encoded, err := method.Inputs.EncodeABIDataValuesCtx(ctx, batchPin)
			if err == nil {
				orderedInput[len(orderedInput)-1] = hex.EncodeToString(encoded)
			}
		}
		if err != nil {
			return true, err
		}
	}
	return e.invokeContractMethod(ctx, ethereumLocation.Address, signingKey, methodInfo.methodABI, nsOpID, orderedInput, methodInfo.errorsABI, options)
}

func (e *Ethereum) QueryContract(ctx context.Context, signingKey string, location *fftypes.JSONAny, parsedMethod interface{}, input map[string]interface{}, options map[string]interface{}) (interface{}, error) {
	ethereumLocation, err := e.parseContractLocation(ctx, location)
	if err != nil {
		return nil, err
	}
	methodInfo, orderedInput, err := e.prepareRequest(ctx, parsedMethod, input)
	if err != nil {
		return nil, err
	}
	res, err := e.queryContractMethod(ctx, ethereumLocation.Address, signingKey, methodInfo.methodABI, orderedInput, methodInfo.errorsABI, options)
	if err != nil || !res.IsSuccess() {
		return nil, err
	}

	var output interface{}
	if err = json.Unmarshal(res.Body(), &output); err != nil {
		return nil, err
	}
	return output, nil // note UNLIKE fabric this is just `output`, not `output.Result` - but either way the top level of what we return to the end user, is whatever the Connector sent us
}

func (e *Ethereum) NormalizeContractLocation(ctx context.Context, ntype blockchain.NormalizeType, location *fftypes.JSONAny) (result *fftypes.JSONAny, err error) {
	parsed, err := e.parseContractLocation(ctx, location)
	if err != nil {
		return nil, err
	}
	return e.encodeContractLocation(ctx, parsed)
}

func (e *Ethereum) parseContractLocation(ctx context.Context, location *fftypes.JSONAny) (*Location, error) {
	ethLocation := Location{}
	if err := json.Unmarshal(location.Bytes(), &ethLocation); err != nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgContractLocationInvalid, err)
	}
	if ethLocation.Address == "" {
		return nil, i18n.NewError(ctx, coremsgs.MsgContractLocationInvalid, "'address' not set")
	}
	return &ethLocation, nil
}

func (e *Ethereum) encodeContractLocation(ctx context.Context, location *Location) (result *fftypes.JSONAny, err error) {
	location.Address, err = formatEthAddress(ctx, location.Address)
	if err != nil {
		return nil, err
	}
	normalized, err := json.Marshal(location)
	if err == nil {
		result = fftypes.JSONAnyPtrBytes(normalized)
	}
	return result, err
}

func (e *Ethereum) AddContractListener(ctx context.Context, listener *core.ContractListener) (err error) {
	var location *Location
	if listener.Location != nil {
		location, err = e.parseContractLocation(ctx, listener.Location)
		if err != nil {
			return err
		}
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

func (e *Ethereum) DeleteContractListener(ctx context.Context, subscription *core.ContractListener, okNotFound bool) error {
	return e.streams.deleteSubscription(ctx, subscription.BackendID, okNotFound)
}

func (e *Ethereum) GetContractListenerStatus(ctx context.Context, subID string, okNotFound bool) (found bool, status interface{}, err error) {
	sub, err := e.streams.getSubscription(ctx, subID, okNotFound)
	if err != nil || sub == nil {
		return false, nil, err
	}

	checkpoint := &ListenerStatus{
		Catchup: sub.Catchup,
		Checkpoint: ListenerCheckpoint{
			Block:            sub.Checkpoint.Block,
			TransactionIndex: sub.Checkpoint.TransactionIndex,
			LogIndex:         sub.Checkpoint.LogIndex,
		},
	}

	return true, checkpoint, nil
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

func (e *Ethereum) GenerateErrorSignature(ctx context.Context, errorDef *fftypes.FFIErrorDefinition) string {
	abi, err := ffi2abi.ConvertFFIErrorDefinitionToABI(ctx, errorDef)
	if err != nil {
		return ""
	}
	return ffi2abi.ABIMethodToSignature(abi)
}

type parsedFFIMethod struct {
	methodABI *abi.Entry
	errorsABI []*abi.Entry
}

func (e *Ethereum) ParseInterface(ctx context.Context, method *fftypes.FFIMethod, errors []*fftypes.FFIError) (interface{}, error) {
	methodABI, err := ffi2abi.ConvertFFIMethodToABI(ctx, method)
	if err != nil {
		return nil, err
	}
	methodInfo := &parsedFFIMethod{
		methodABI: methodABI,
		errorsABI: make([]*abi.Entry, len(errors)),
	}
	for i, ffiError := range errors {
		errorABI, err := ffi2abi.ConvertFFIErrorDefinitionToABI(ctx, &ffiError.FFIErrorDefinition)
		if err != nil {
			return nil, err
		}
		methodInfo.errorsABI[i] = errorABI
	}
	return methodInfo, nil
}

func (e *Ethereum) prepareRequest(ctx context.Context, parsedMethod interface{}, input map[string]interface{}) (*parsedFFIMethod, []interface{}, error) {
	methodInfo, ok := parsedMethod.(*parsedFFIMethod)
	if !ok || methodInfo.methodABI == nil {
		return nil, nil, i18n.NewError(ctx, coremsgs.MsgUnexpectedInterfaceType, parsedMethod)
	}
	inputs := methodInfo.methodABI.Inputs
	orderedInput := make([]interface{}, len(inputs))
	for i, param := range inputs {
		orderedInput[i] = input[param.Name]
	}
	return methodInfo, orderedInput, nil
}

func (e *Ethereum) getContractAddress(ctx context.Context, instancePath string) (string, error) {
	res, err := e.client.R().
		SetContext(ctx).
		Get(instancePath)
	if err != nil || !res.IsSuccess() {
		return "", ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgEthConnectorRESTErr)
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
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgFFIGenerationFailed, "unable to deserialize JSON as ABI")
	}
	if input.ABI == nil || len(*input.ABI) == 0 {
		return nil, i18n.NewError(ctx, coremsgs.MsgFFIGenerationFailed, "ABI is empty")
	}
	return ffi2abi.ConvertABIToFFI(ctx, generationRequest.Namespace, generationRequest.Name, generationRequest.Version, generationRequest.Description, input.ABI)
}

func (e *Ethereum) GetNetworkVersion(ctx context.Context, location *fftypes.JSONAny) (version int, err error) {
	ethLocation, err := e.parseContractLocation(ctx, location)
	if err != nil {
		return 0, err
	}

	cacheKey := "version:" + ethLocation.Address
	if cachedValue := e.cache.GetInt(cacheKey); cachedValue != 0 {
		return cachedValue, nil
	}

	version, err = e.queryNetworkVersion(ctx, ethLocation.Address)
	if err == nil {
		e.cache.SetInt(cacheKey, version)
	}
	return version, err
}

func (e *Ethereum) queryNetworkVersion(ctx context.Context, address string) (version int, err error) {
	var emptyErrors []*abi.Entry
	res, err := e.queryContractMethod(ctx, address, "", networkVersionMethodABI, []interface{}{}, emptyErrors, nil)
	if err != nil || !res.IsSuccess() {
		// "Call failed" is interpreted as "method does not exist, default to version 1"
		if strings.Contains(err.Error(), "FFEC100148") || strings.Contains(err.Error(), "FF23021") {
			return 1, nil
		}
		return 0, err
	}

	// Leave as queryOutput as it only has one value
	output := &queryOutput{}
	if err = json.Unmarshal(res.Body(), output); err != nil {
		return 0, err
	}

	switch result := output.Output.(type) {
	case string:
		version, err = strconv.Atoi(result)
	default:
		err = i18n.NewError(ctx, coremsgs.MsgBadNetworkVersion, output.Output)
	}
	return version, err
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

	location, err = e.encodeContractLocation(ctx, &Location{
		Address: address,
	})
	return location, fromBlock, err
}

func (e *Ethereum) GetTransactionStatus(ctx context.Context, operation *core.Operation) (interface{}, error) {
	txnID := (&core.PreparedOperation{ID: operation.ID, Namespace: operation.Namespace}).NamespacedIDString()

	transactionRequestPath := fmt.Sprintf("/transactions/%s", txnID)
	client := e.client
	var resErr common.BlockchainRESTError
	var statusResponse fftypes.JSONObject
	res, err := client.R().
		SetContext(ctx).
		SetError(&resErr).
		SetResult(&statusResponse).
		Get(transactionRequestPath)
	if err != nil || !res.IsSuccess() {
		if res.StatusCode() == 404 {
			return nil, nil
		}
		return nil, common.WrapRESTError(ctx, &resErr, res, err, coremsgs.MsgEthConnectorRESTErr)
	}

	receiptInfo := statusResponse.GetObject("receipt")
	txStatus := statusResponse.GetString("status")

	if txStatus != "" {
		var replyType string
		if txStatus == "Succeeded" {
			replyType = ReceiptTransactionSuccess
		} else {
			replyType = ReceiptTransactionFailed
		}
		// If the status has changed, mock up blockchain receipt as if we'd received it
		// as a web socket notification
		if (operation.Status == core.OpStatusPending || operation.Status == core.OpStatusInitialized) && txStatus != ethTxStatusPending {
			receipt := &common.BlockchainReceiptNotification{
				Headers: common.BlockchainReceiptHeaders{
					ReceiptID: statusResponse.GetString("id"),
					ReplyType: replyType},
				TxHash:     statusResponse.GetString("transactionHash"),
				Message:    statusResponse.GetString("errorMessage"),
				ProtocolID: receiptInfo.GetString("protocolId")}
			err := common.HandleReceipt(ctx, e, receipt, e.callbacks)
			if err != nil {
				log.L(ctx).Warnf("Failed to handle receipt")
			}
		}
	} else {
		// Don't expect to get here so issue a warning
		log.L(ctx).Warnf("Transaction status didn't include txStatus information")
	}

	return statusResponse, nil
}
