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
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/kaleido-io/firefly/internal/blockchain"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/ffresty"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
)

type Ethereum struct {
	ctx      context.Context
	conf     *Config
	events   blockchain.Events
	client   *resty.Client
	initInfo struct {
		stream *eventStream
		subs   []*subscription
	}
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
	FromBlock   string `json:"fromBlock"`
}

type asyncTXSubmission struct {
	ID string `json:"id"`
}

type ethBroadcastBatchInput struct {
	BatchID    string `json:"batchId"`
	PayloadRef string `json:"payloadRef"`
}

var requiredSubscriptions = map[string]string{
	"AssetInstanceBatchCreated": "Asset instance batch created",
}

const (
	defaultBatchSize    = 50
	defaultBatchTimeout = 500
)

func (e *Ethereum) ConfigInterface() interface{} { return &Config{} }

func (e *Ethereum) Init(ctx context.Context, conf interface{}, events blockchain.Events) (*blockchain.Capabilities, error) {
	e.ctx = log.WithLogField(ctx, "proto", "ethereum")
	e.conf = conf.(*Config)
	e.events = events
	e.client = ffresty.New(e.ctx, &e.conf.Ethconnect.HTTPConfig)

	log.L(e.ctx).Debugf("Config: %+v", e.conf)

	if err := e.ensureEventStreams(); err != nil {
		return nil, err
	}

	return &blockchain.Capabilities{
		GlobalSequencer: true,
	}, nil
}

func (e *Ethereum) ensureEventStreams() error {

	var existingStreams []eventStream
	res, err := e.client.R().SetContext(e.ctx).SetResult(&existingStreams).Get("/eventstreams")
	if err != nil || !res.IsSuccess() {
		return ffresty.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
	}

	for _, stream := range existingStreams {
		if stream.WebSocket.Topic == e.conf.Ethconnect.Topic {
			e.initInfo.stream = &stream
		}
	}

	if e.initInfo.stream == nil {
		newStream := eventStream{
			Name:           e.conf.Ethconnect.Topic,
			ErrorHandling:  "block",
			BatchSize:      config.UintWithDefault(e.conf.Ethconnect.BatchSize, defaultBatchSize),
			BatchTimeoutMS: config.UintWithDefault(e.conf.Ethconnect.BatchTimeoutMS, defaultBatchTimeout),
			Type:           "websocket",
		}
		newStream.WebSocket.Topic = e.conf.Ethconnect.Topic
		res, err = e.client.R().SetBody(&newStream).SetResult(&newStream).Post("/eventstreams")
		if err != nil || !res.IsSuccess() {
			return ffresty.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
		}
		e.initInfo.stream = &newStream
	}

	log.L(e.ctx).Infof("Event stream: %s", e.initInfo.stream.ID)

	return e.ensureSusbscriptions(e.initInfo.stream.ID)
}

func (e *Ethereum) ensureSusbscriptions(streamID string) error {

	for eventType, subDesc := range requiredSubscriptions {

		var existingSubs []subscription
		res, err := e.client.R().SetResult(&existingSubs).Get("/subscriptions")
		if err != nil || !res.IsSuccess() {
			return ffresty.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
		}

		var sub *subscription
		for _, s := range existingSubs {
			if s.Name == eventType {
				sub = &s
			}
		}

		if sub == nil {
			newSub := subscription{
				Name:        e.conf.Ethconnect.Topic,
				Description: subDesc,
				StreamID:    streamID,
				FromBlock:   "0",
			}
			res, err = e.client.R().SetContext(e.ctx).SetBody(&newSub).SetResult(&newSub).Post("/subscriptions")
			if err != nil || !res.IsSuccess() {
				return ffresty.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
			}
			sub = &newSub
		}

		log.L(e.ctx).Infof("%s subscription: %s", eventType, sub.ID)
		e.initInfo.subs = append(e.initInfo.subs, sub)

	}
	return nil
}

func ethHexFormatB32(b fftypes.Bytes32) string {
	return "0x" + hex.EncodeToString(b[0:32])
}

func (e *Ethereum) SubmitBroadcastBatch(ctx context.Context, identity string, broadcast *blockchain.BroadcastBatch) (txTrackingID string, err error) {
	tx := &asyncTXSubmission{}
	input := &ethBroadcastBatchInput{
		BatchID:    ethHexFormatB32(broadcast.BatchID),
		PayloadRef: ethHexFormatB32(broadcast.BatchPaylodRef),
	}
	path := fmt.Sprintf("%s/broadcastBatch", e.conf.Ethconnect.InstancePath)
	res, err := e.client.R().
		SetContext(ctx).
		SetQueryParam("kld-from", identity).
		SetQueryParam("kld-sync", "false").
		SetBody(input).
		SetResult(tx).
		Post(path)
	if err != nil || !res.IsSuccess() {
		return "", ffresty.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return tx.ID, nil
}
