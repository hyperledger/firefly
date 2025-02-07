// Copyright © 2024 Kaleido, Inc.
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

package cardano

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/wsclient"
	"github.com/hyperledger/firefly/internal/blockchain/common"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
)

const (
	cardanoTxStatusPending string = "Pending"
)

const (
	ReceiptTransactionSuccess string = "TransactionSuccess"
	ReceiptTransactionFailed  string = "TransactionFailed"
)

type Cardano struct {
	ctx                context.Context
	cancelCtx          context.CancelFunc
	pluginTopic        string
	metrics            metrics.Manager
	capabilities       *blockchain.Capabilities
	callbacks          common.BlockchainCallbacks
	client             *resty.Client
	streams            *streamManager
	streamID           string
	wsconn             wsclient.WSClient
	cardanoconnectConf config.Section
	subs               common.FireflySubscriptions
}

type ffiMethodAndErrors struct {
	method *fftypes.FFIMethod
	errors []*fftypes.FFIError
}

type cardanoWSCommandPayload struct {
	Type        string `json:"type"`
	Topic       string `json:"topic,omitempty"`
	BatchNumber int64  `json:"batchNumber,omitempty"`
	Message     string `json:"message,omitempty"`
}

type Location struct {
	Address string `json:"address"`
}

type cardanoInvokeContractPayload struct {
	ID      string             `json:"id"`
	From    string             `json:"from"`
	Address string             `json:"address"`
	Method  *fftypes.FFIMethod `json:"method"`
	Params  []interface{}      `json:"params"`
}

func (c *Cardano) Name() string {
	return "cardano"
}

func (c *Cardano) VerifierType() core.VerifierType {
	return core.VerifierTypeCardanoAddress
}

func (c *Cardano) Init(ctx context.Context, cancelCtx context.CancelFunc, conf config.Section, metrics metrics.Manager, cacheManager cache.Manager) (err error) {
	c.InitConfig(conf)
	cardanoconnectConf := c.cardanoconnectConf

	c.ctx = log.WithLogField(ctx, "proto", "cardano")
	c.cancelCtx = cancelCtx
	c.metrics = metrics
	c.capabilities = &blockchain.Capabilities{}
	c.callbacks = common.NewBlockchainCallbacks()
	c.subs = common.NewFireflySubscriptions()

	if cardanoconnectConf.GetString(ffresty.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, coremsgs.MsgMissingPluginConfig, "url", cardanoconnectConf)
	}

	wsConfig, err := wsclient.GenerateConfig(ctx, cardanoconnectConf)
	if err == nil {
		c.client, err = ffresty.New(c.ctx, cardanoconnectConf)
	}

	if err != nil {
		return err
	}

	c.pluginTopic = cardanoconnectConf.GetString(CardanoconnectConfigTopic)
	if c.pluginTopic == "" {
		return i18n.NewError(ctx, coremsgs.MsgMissingPluginConfig, "topic", "blockchain.cardano.cardanoconnect")
	}

	if wsConfig.WSKeyPath == "" {
		wsConfig.WSKeyPath = "/ws"
	}
	c.wsconn, err = wsclient.New(ctx, wsConfig, nil, c.afterConnect)
	if err != nil {
		return err
	}

	c.streams = newStreamManager(c.client, c.cardanoconnectConf.GetUint(CardanoconnectConfigBatchSize), uint(c.cardanoconnectConf.GetDuration(CardanoconnectConfigBatchTimeout).Milliseconds()))

	stream, err := c.streams.ensureEventStream(c.ctx, c.pluginTopic)
	if err != nil {
		return err
	}

	log.L(c.ctx).Infof("Event stream: %s (topic=%s)", stream.ID, c.pluginTopic)
	c.streamID = stream.ID

	go c.eventLoop()

	return c.wsconn.Connect()
}

func (c *Cardano) StartNamespace(ctx context.Context, namespace string) (err error) {
	// TODO: Implement
	return nil
}

func (c *Cardano) StopNamespace(ctx context.Context, namespace string) (err error) {
	// TODO: Implement
	return nil
}

func (c *Cardano) SetHandler(namespace string, handler blockchain.Callbacks) {
	c.callbacks.SetHandler(namespace, handler)
}

func (c *Cardano) SetOperationHandler(namespace string, handler core.OperationCallbacks) {
	c.callbacks.SetOperationalHandler(namespace, handler)
}

func (c *Cardano) Capabilities() *blockchain.Capabilities {
	return c.capabilities
}

func (c *Cardano) AddFireflySubscription(ctx context.Context, namespace *core.Namespace, contract *blockchain.MultipartyContract, lastProtocolID string) (string, error) {
	return "", errors.New("AddFireflySubscription not supported")
}

func (c *Cardano) RemoveFireflySubscription(ctx context.Context, subID string) {
	c.subs.RemoveSubscription(ctx, subID)
}

func (c *Cardano) ResolveSigningKey(ctx context.Context, key string, intent blockchain.ResolveKeyIntent) (resolved string, err error) {
	if key == "" {
		return "", i18n.NewError(ctx, coremsgs.MsgNodeMissingBlockchainKey)
	}
	resolved, err = formatCardanoAddress(ctx, key)
	return resolved, err
}

func (c *Cardano) SubmitBatchPin(ctx context.Context, nsOpID, networkNamespace, signingKey string, batch *blockchain.BatchPin, location *fftypes.JSONAny) error {
	return errors.New("SubmitBatchPin not supported")
}

func (c *Cardano) SubmitNetworkAction(ctx context.Context, nsOpID string, signingKey string, action core.NetworkActionType, location *fftypes.JSONAny) error {
	return errors.New("SubmitNetworkAction not supported")
}

func (c *Cardano) DeployContract(ctx context.Context, nsOpID, signingKey string, definition, contract *fftypes.JSONAny, input []interface{}, options map[string]interface{}) (submissionRejected bool, err error) {
	body := map[string]interface{}{
		"id":         nsOpID,
		"contract":   contract,
		"definition": definition,
	}
	var resErr common.BlockchainRESTError
	res, err := c.client.R().
		SetContext(ctx).
		SetBody(body).
		SetError(&resErr).
		Post("/contracts/deploy")
	if err != nil || !res.IsSuccess() {
		return resErr.SubmissionRejected, common.WrapRESTError(ctx, &resErr, res, err, coremsgs.MsgCardanoconnectRESTErr)
	}
	return false, nil
}

func (c *Cardano) ValidateInvokeRequest(ctx context.Context, parsedMethod interface{}, input map[string]interface{}, hasMessage bool) error {
	// No additional validation beyond what is enforced by Contract Manager
	_, _, err := c.recoverFFI(ctx, parsedMethod)
	return err
}

func (c *Cardano) InvokeContract(ctx context.Context, nsOpID string, signingKey string, location *fftypes.JSONAny, parsedMethod interface{}, input map[string]interface{}, options map[string]interface{}, batch *blockchain.BatchPin) (bool, error) {
	cardanoLocation, err := c.parseContractLocation(ctx, location)
	if err != nil {
		return true, err
	}

	methodInfo, ok := parsedMethod.(*ffiMethodAndErrors)
	if !ok || methodInfo.method == nil {
		return true, i18n.NewError(ctx, coremsgs.MsgUnexpectedInterfaceType, parsedMethod)
	}
	method := methodInfo.method
	params := make([]interface{}, 0)
	for _, param := range method.Params {
		params = append(params, input[param.Name])
	}

	body := map[string]interface{}{
		"id":      nsOpID,
		"address": cardanoLocation.Address,
		"method":  method,
		"params":  params,
	}
	if signingKey != "" {
		body["from"] = signingKey
	}

	var resErr common.BlockchainRESTError
	res, err := c.client.R().
		SetContext(ctx).
		SetBody(body).
		SetError(&resErr).
		Post("/contracts/invoke")
	if err != nil || !res.IsSuccess() {
		return resErr.SubmissionRejected, common.WrapRESTError(ctx, &resErr, res, err, coremsgs.MsgCardanoconnectRESTErr)
	}
	return false, nil
}

func (c *Cardano) QueryContract(ctx context.Context, signingKey string, location *fftypes.JSONAny, parsedMethod interface{}, input map[string]interface{}, options map[string]interface{}) (interface{}, error) {
	return nil, errors.New("QueryContract not supported")
}

func (c *Cardano) ParseInterface(ctx context.Context, method *fftypes.FFIMethod, errors []*fftypes.FFIError) (interface{}, error) {
	return &ffiMethodAndErrors{
		method: method,
		errors: errors,
	}, nil
}

func (c *Cardano) NormalizeContractLocation(ctx context.Context, ntype blockchain.NormalizeType, location *fftypes.JSONAny) (result *fftypes.JSONAny, err error) {
	parsed, err := c.parseContractLocation(ctx, location)
	if err != nil {
		return nil, err
	}
	return c.encodeContractLocation(ctx, parsed)
}

func (c *Cardano) CheckOverlappingLocations(ctx context.Context, left *fftypes.JSONAny, right *fftypes.JSONAny) (bool, error) {
	if left == nil || right == nil {
		// No location on either side so overlapping
		return true, nil
	}

	parsedLeft, err := c.parseContractLocation(ctx, left)
	if err != nil {
		return false, err
	}

	parsedRight, err := c.parseContractLocation(ctx, right)
	if err != nil {
		return false, err
	}

	// For cardano just compare addresses
	return strings.EqualFold(parsedLeft.Address, parsedRight.Address), nil
}

func (c *Cardano) parseContractLocation(ctx context.Context, location *fftypes.JSONAny) (*Location, error) {
	cardanoLocation := Location{}
	if err := json.Unmarshal(location.Bytes(), &cardanoLocation); err != nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgContractLocationInvalid, err)
	}
	if cardanoLocation.Address == "" {
		return nil, i18n.NewError(ctx, coremsgs.MsgContractLocationInvalid, "'address' not set")
	}
	return &cardanoLocation, nil
}

func (c *Cardano) encodeContractLocation(ctx context.Context, location *Location) (result *fftypes.JSONAny, err error) {
	location.Address, err = formatCardanoAddress(ctx, location.Address)
	if err != nil {
		return nil, err
	}
	normalized, err := json.Marshal(location)
	if err == nil {
		result = fftypes.JSONAnyPtrBytes(normalized)
	}
	return result, err
}

func (c *Cardano) AddContractListener(ctx context.Context, listener *core.ContractListener, lastProtocolID string) (err error) {
	subName := fmt.Sprintf("ff-sub-%s-%s", listener.Namespace, listener.ID)
	firstEvent := string(core.SubOptsFirstEventNewest)
	if listener.Options != nil {
		firstEvent = listener.Options.FirstEvent
	}

	result, err := c.streams.createListener(ctx, c.streamID, subName, firstEvent, &listener.Filters)
	listener.BackendID = result.ID
	return err
}

func (c *Cardano) DeleteContractListener(ctx context.Context, subscription *core.ContractListener, okNotFound bool) error {
	return c.streams.deleteListener(ctx, c.streamID, subscription.BackendID)
}

func (c *Cardano) GetContractListenerStatus(ctx context.Context, namespace, subID string, okNotFound bool) (found bool, detail interface{}, status core.ContractListenerStatus, err error) {
	l, err := c.streams.getListener(ctx, c.streamID, subID)
	if err != nil || l == nil {
		return false, nil, core.ContractListenerStatusUnknown, err
	}
	return true, nil, core.ContractListenerStatusUnknown, nil
}

func (c *Cardano) GetFFIParamValidator(ctx context.Context) (fftypes.FFIParamValidator, error) {
	// Cardanoconnect does not require any additional validation beyond "JSON Schema correctness" at this time
	return nil, nil
}

func (c *Cardano) GenerateEventSignature(ctx context.Context, event *fftypes.FFIEventDefinition) (string, error) {
	params := []string{}
	for _, param := range event.Params {
		params = append(params, param.Schema.JSONObject().GetString("type"))
	}
	return fmt.Sprintf("%s(%s)", event.Name, strings.Join(params, ",")), nil
}

func (c *Cardano) GenerateEventSignatureWithLocation(ctx context.Context, event *fftypes.FFIEventDefinition, location *fftypes.JSONAny) (string, error) {
	eventSignature, _ := c.GenerateEventSignature(ctx, event)

	if location == nil {
		return fmt.Sprintf("*:%s", eventSignature), nil
	}

	parsed, err := c.parseContractLocation(ctx, location)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", parsed.Address, eventSignature), nil
}

func (c *Cardano) GenerateErrorSignature(ctx context.Context, event *fftypes.FFIErrorDefinition) string {
	// TODO: impl
	return ""
}

func (c *Cardano) GenerateFFI(ctx context.Context, generationRequest *fftypes.FFIGenerationRequest) (*fftypes.FFI, error) {
	return nil, i18n.NewError(ctx, coremsgs.MsgFFIGenerationUnsupported)
}

func (c *Cardano) GetNetworkVersion(ctx context.Context, location *fftypes.JSONAny) (version int, err error) {
	// Part of the FIR-12. https://github.com/hyperledger/firefly-fir/pull/12
	// Cardano doesn't support any of this yet, so just pretend we're on the new hotness
	return 2, nil
}

func (c *Cardano) GetAndConvertDeprecatedContractConfig(ctx context.Context) (location *fftypes.JSONAny, fromBlock string, err error) {
	return nil, "", nil
}

func (c *Cardano) GetTransactionStatus(ctx context.Context, operation *core.Operation) (interface{}, error) {
	txnID := (&core.PreparedOperation{ID: operation.ID, Namespace: operation.Namespace}).NamespacedIDString()

	transactionRequestPath := fmt.Sprintf("/transactions/%s", txnID)
	client := c.client
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
		return nil, common.WrapRESTError(ctx, &resErr, res, err, coremsgs.MsgCardanoconnectRESTErr)
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
		if (operation.Status == core.OpStatusPending || operation.Status == core.OpStatusInitialized) && txStatus != cardanoTxStatusPending {
			receipt := &common.BlockchainReceiptNotification{
				Headers: common.BlockchainReceiptHeaders{
					ReceiptID: statusResponse.GetString("id"),
					ReplyType: replyType,
				},
				TxHash:     statusResponse.GetString("transactionHash"),
				Message:    statusResponse.GetString("errorMessage"),
				ProtocolID: receiptInfo.GetString("protocolId")}
			err := common.HandleReceipt(ctx, operation.Namespace, c, receipt, c.callbacks)
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

func (c *Cardano) afterConnect(ctx context.Context, w wsclient.WSClient) error {
	// Send a subscribe to our topic after each connect/reconnect
	b, _ := json.Marshal(&cardanoWSCommandPayload{
		Type:  "listen",
		Topic: c.pluginTopic,
	})
	err := w.Send(ctx, b)
	if err == nil {
		b, _ = json.Marshal(&cardanoWSCommandPayload{
			Type: "listenreplies",
		})
		err = w.Send(ctx, b)
	}
	return err
}

func (c *Cardano) recoverFFI(ctx context.Context, parsedMethod interface{}) (*fftypes.FFIMethod, []*fftypes.FFIError, error) {
	methodInfo, ok := parsedMethod.(*ffiMethodAndErrors)
	if !ok || methodInfo.method == nil {
		return nil, nil, i18n.NewError(ctx, coremsgs.MsgUnexpectedInterfaceType, parsedMethod)
	}
	return methodInfo.method, methodInfo.errors, nil
}

func (c *Cardano) eventLoop() {
	defer c.wsconn.Close()
	l := log.L(c.ctx).WithField("role", "event-loop")
	ctx := log.WithLogger(c.ctx, l)
	for {
		select {
		case <-ctx.Done():
			l.Debugf("Event loop exiting (context cancelled)")
			return
		case msgBytes, ok := <-c.wsconn.Receive():
			if !ok {
				l.Debugf("Event loop exiting (receive channel closed). Terminating server!")
				c.cancelCtx()
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
				err = c.handleMessageBatch(ctx, 0, msgTyped)
				if err == nil {
					ack, _ := json.Marshal(&cardanoWSCommandPayload{
						Type:  "ack",
						Topic: c.pluginTopic,
					})
					err = c.wsconn.Send(ctx, ack)
				}
			case map[string]interface{}:
				isBatch := false
				if batchNumber, ok := msgTyped["batchNumber"].(float64); ok {
					if events, ok := msgTyped["events"].([]interface{}); ok {
						// FFTM delivery with a batch number to use in the ack
						isBatch = true
						err = c.handleMessageBatch(ctx, (int64)(batchNumber), events)
						// Errors processing messages are converted into nacks
						ackOrNack := &cardanoWSCommandPayload{
							Topic:       c.pluginTopic,
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
						err = c.wsconn.Send(ctx, b)
					}
				}
				if !isBatch {
					var receipt common.BlockchainReceiptNotification
					_ = json.Unmarshal(msgBytes, &receipt)

					err := common.HandleReceipt(ctx, "", c, &receipt, c.callbacks)
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
				c.cancelCtx()
				return
			}
		}
	}
}

func (c *Cardano) handleMessageBatch(ctx context.Context, batchID int64, messages []interface{}) error {
	events := make(common.EventsToDispatch)
	count := len(messages)
	for i, msgI := range messages {
		msgMap, ok := msgI.(map[string]interface{})
		if !ok {
			log.L(ctx).Errorf("Message cannot be parsed as JSON: %+v", msgI)
			return nil
		}
		msgJSON := fftypes.JSONObject(msgMap)

		signature := msgJSON.GetString("signature")

		logger := log.L(ctx)
		logger.Infof("[Cardano:%d:%d/%d]: '%s'", batchID, i+1, count, signature)
		logger.Tracef("Message: %+v", msgJSON)
		if err := c.processContractEvent(ctx, events, msgJSON); err != nil {
			return err
		}
	}

	// Dispatch all the events from this patch that were successfully parsed and routed to namespaces
	// (could be zero - that's ok)
	return c.callbacks.DispatchBlockchainEvents(ctx, events)
}

func (c *Cardano) processContractEvent(ctx context.Context, events common.EventsToDispatch, msgJSON fftypes.JSONObject) error {
	listenerID := msgJSON.GetString("listenerId")
	listener, err := c.streams.getListener(ctx, c.streamID, listenerID)
	if err != nil {
		return err
	}
	namespace := common.GetNamespaceFromSubName(listener.Name)
	event := c.parseBlockchainEvent(ctx, msgJSON)
	if event != nil {
		c.callbacks.PrepareBlockchainEvent(ctx, events, namespace, &blockchain.EventForListener{
			Event:      event,
			ListenerID: listenerID,
		})
	}
	return nil
}

func (c *Cardano) parseBlockchainEvent(ctx context.Context, msgJSON fftypes.JSONObject) *blockchain.Event {
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
		Source:         c.Name(),
		Name:           name,
		ProtocolID:     fmt.Sprintf("%.12d/%.6d/%.6d", blockNumber, txIndex, logIndex),
		Output:         dataJSON,
		Info:           msgJSON,
		Timestamp:      timestamp,
		Location:       c.buildEventLocationString(msgJSON),
		Signature:      signature,
	}
}

func (c *Cardano) buildEventLocationString(msgJSON fftypes.JSONObject) string {
	return fmt.Sprintf("address=%s", msgJSON.GetString("address"))
}

func formatCardanoAddress(ctx context.Context, key string) (string, error) {
	// TODO: this could be much stricter validation
	if key != "" {
		return key, nil
	}
	return "", i18n.NewError(ctx, coremsgs.MsgInvalidCardanoAddress)
}
