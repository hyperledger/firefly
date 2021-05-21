// Copyright © 2021 Kaleido, Inc.
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
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/restclient"
	"github.com/kaleido-io/firefly/internal/wsclient"
	"github.com/kaleido-io/firefly/pkg/blockchain"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

const (
	broadcastBatchEventSignature = "BroadcastBatch(address,uint256,bytes32,bytes32,bytes32)"
)

type Ethereum struct {
	ctx          context.Context
	topic        string
	instancePath string
	capabilities *blockchain.Capabilities
	callbacks    blockchain.Callbacks
	client       *resty.Client
	initInfo     struct {
		stream *eventStream
		subs   []*subscription
	}
	wsconn wsclient.WSClient
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
	StreamID    string `json:"streamId"`
	Stream      string `json:"stream"`
	FromBlock   string `json:"fromBlock"`
}

type asyncTXSubmission struct {
	ID string `json:"id"`
}

type ethBroadcastBatchInput struct {
	TransactionID string `json:"txnId"`
	BatchID       string `json:"batchId"`
	PayloadRef    string `json:"payloadRef"`
}

type ethWSCommandPayload struct {
	Type  string `json:"type"`
	Topic string `json:"topic,omitempty"`
}

var requiredSubscriptions = map[string]string{
	"BroadcastBatch": "Batch broadcast",
}

var addressVerify = regexp.MustCompile("^[0-9a-f]{40}$")

func (e *Ethereum) Name() string {
	return "ethereum"
}

func (e *Ethereum) Init(ctx context.Context, prefix config.Prefix, callbacks blockchain.Callbacks) (err error) {

	ethconnectConf := prefix.SubPrefix(EthconnectConfigKey)

	e.ctx = log.WithLogField(ctx, "proto", "ethereum")
	e.callbacks = callbacks

	if ethconnectConf.GetString(restclient.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "url", "blockchain.ethconnect")
	}
	e.instancePath = ethconnectConf.GetString(EthconnectConfigInstancePath)
	if e.instancePath == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "instance", "blockchain.ethconnect")
	}
	e.topic = ethconnectConf.GetString(EthconnectConfigTopic)
	if e.topic == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "topic", "blockchain.ethconnect")
	}

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

	go e.eventLoop()

	log.L(e.ctx).Infof("Event stream: %s", e.initInfo.stream.ID)

	return e.ensureSusbscriptions(e.initInfo.stream.ID)
}

func (e *Ethereum) afterConnect(ctx context.Context, w wsclient.WSClient) error {
	// Send a subscribe to our topic after each connect/reconnect
	b, _ := json.Marshal(&ethWSCommandPayload{
		Type:  "listen",
		Topic: e.topic,
	})
	return w.Send(ctx, b)
}

func (e *Ethereum) ensureSusbscriptions(streamID string) error {
	for eventType, subDesc := range requiredSubscriptions {

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
				Name:        eventType,
				Description: subDesc,
				StreamID:    streamID,
				Stream:      e.initInfo.stream.ID,
				FromBlock:   "0",
			}
			res, err = e.client.R().
				SetContext(e.ctx).
				SetBody(&newSub).
				SetResult(&newSub).
				Post(fmt.Sprintf("%s/%s", e.instancePath, eventType))
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
	return "0x" + hex.EncodeToString(b[0:32])
}

func (e *Ethereum) handleBroadcastBatchEvent(ctx context.Context, msgJSON fftypes.JSONObject) error {
	sBlockNumber := msgJSON.GetString(ctx, "blockNumber")
	sTransactionIndex := msgJSON.GetString(ctx, "transactionIndex")
	sTransactionHash := msgJSON.GetString(ctx, "transactionHash")
	dataJSON := msgJSON.GetObject(ctx, "data")
	sAuthor := dataJSON.GetString(ctx, "author")
	sTxnId := dataJSON.GetString(ctx, "txnId")
	sBatchId := dataJSON.GetString(ctx, "batchId")
	sPayloadRef := dataJSON.GetString(ctx, "payloadRef")

	if sBlockNumber == "" ||
		sTransactionIndex == "" ||
		sTransactionHash == "" ||
		sAuthor == "" ||
		sTxnId == "" ||
		sBatchId == "" ||
		sPayloadRef == "" {
		log.L(ctx).Errorf("BroadcastBatch event is not valid - missing data: %+v", msgJSON)
		return nil // move on
	}

	author, err := e.VerifyIdentitySyntax(ctx, sAuthor)
	if err != nil {
		log.L(ctx).Errorf("BroadcastBatch event is not valid - bad author (%s): %+v", err, msgJSON)
		return nil // move on
	}

	var txnIDB32 fftypes.Bytes32
	err = txnIDB32.UnmarshalText([]byte(sTxnId))
	if err != nil {
		log.L(ctx).Errorf("BroadcastBatch event is not valid - bad txnId (%s): %+v", err, msgJSON)
		return nil // move on
	}
	var txnID fftypes.UUID
	copy(txnID[:], txnIDB32[0:16])

	var batchIDB32 fftypes.Bytes32
	err = batchIDB32.UnmarshalText([]byte(sBatchId))
	if err != nil {
		log.L(ctx).Errorf("BroadcastBatch event is not valid - bad batchId (%s): %+v", err, msgJSON)
		return nil // move on
	}
	var batchID fftypes.UUID
	copy(batchID[:], batchIDB32[0:16])

	var payloadRef fftypes.Bytes32
	err = payloadRef.UnmarshalText([]byte(sPayloadRef))
	if err != nil {
		log.L(ctx).Errorf("BroadcastBatch event is not valid - bad payloadRef (%s): %+v", err, msgJSON)
		return nil // move on
	}

	batch := &blockchain.BroadcastBatch{
		TransactionID:  &txnID,
		BatchID:        &batchID,
		BatchPaylodRef: &payloadRef,
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	return e.callbacks.SequencedBroadcastBatch(batch, author, sTransactionHash, msgJSON)
}

func (e *Ethereum) handleMessageBatch(ctx context.Context, message []byte) error {

	l := log.L(ctx)
	var msgsJSON []fftypes.JSONObject
	err := json.Unmarshal(message, &msgsJSON)
	if err != nil {
		l.Errorf("Message cannot be parsed as JSON: %s\n%s", err, string(message))
		return nil // Swallow this and move on
	}

	for i, msgJSON := range msgsJSON {
		l1 := l.WithField("ethmsgidx", i)
		ctx1 := log.WithLogger(ctx, l1)
		signature := msgJSON.GetString(ctx, "signature")
		l1.Infof("Received '%s' message", signature)
		l1.Tracef("Message: %+v", msgJSON)

		switch signature {
		case broadcastBatchEventSignature:
			if err = e.handleBroadcastBatchEvent(ctx1, msgJSON); err != nil {
				return err
			}
		default:
			l.Infof("Ignoring event with unknown signature: %s", signature)
		}
	}

	return nil
}

func (e *Ethereum) eventLoop() {
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
			err := e.handleMessageBatch(ctx, msgBytes)
			if err == nil {
				err = e.wsconn.Send(ctx, ack)
			}

			// Send the ack - only fails if shutting down
			if err != nil {
				l.Errorf("Event loop exiting: %s", err)
				return
			}
		}
	}
}

func (e *Ethereum) VerifyIdentitySyntax(ctx context.Context, identity string) (string, error) {
	identity = strings.TrimPrefix(strings.ToLower(identity), "0x")
	if !addressVerify.MatchString(identity) {
		return "", i18n.NewError(ctx, i18n.MsgInvalidEthAddress)
	}
	return "0x" + identity, nil
}

func (e *Ethereum) SubmitBroadcastBatch(ctx context.Context, identity string, batch *blockchain.BroadcastBatch) (txTrackingID string, err error) {
	tx := &asyncTXSubmission{}
	input := &ethBroadcastBatchInput{
		TransactionID: ethHexFormatB32(fftypes.UUIDBytes(batch.TransactionID)),
		BatchID:       ethHexFormatB32(fftypes.UUIDBytes(batch.BatchID)),
		PayloadRef:    ethHexFormatB32(batch.BatchPaylodRef),
	}
	path := fmt.Sprintf("%s/broadcastBatch", e.instancePath)
	res, err := e.client.R().
		SetContext(ctx).
		SetQueryParam("kld-from", identity).
		SetQueryParam("kld-sync", "false").
		SetBody(input).
		SetResult(tx).
		Post(path)
	if err != nil || !res.IsSuccess() {
		return "", restclient.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return tx.ID, nil
}
