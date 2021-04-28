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

	"github.com/go-resty/resty/v2"
	"github.com/kaleido-io/firefly/internal/blockchain"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/ffresty"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
)

type Ethereum struct {
	ctx    context.Context
	conf   *Config
	events blockchain.Events
	client *resty.Client
}

const (
	defaultBatchSize    = 50
	defaultBatchTimeout = 500
)

func (e *Ethereum) ConfigInterface() interface{} { return &Ethereum{} }

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

var requiredSubscriptions = map[string]string{
	"AssetInstanceBatchCreated": "Asset instance batch created",
}

type eventStream struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ErrorHandling  string `json:"errorHandling"`
	BatchSize      uint   `json:"batchSize"`
	BatchTimeoutMS uint   `json:"batchTimeoutMS"`
	Type           string `json:"type"`
	WebSocket      struct {
		Topic string `json:"topic"`
	} `json:"websocket"`
}

type subscription struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Name        string `json:"name"`
	Stream      string `json:"streamId"`
	FromBlock   string `json:"fromBlock"`
}

func (e *Ethereum) ensureEventStreams() error {

	var existingStreams []eventStream
	res, err := e.client.R().SetResult(&existingStreams).Get("/eventstreams")
	if err != nil || !res.IsSuccess() {
		return ffresty.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
	}

	var streamID = ""
	for _, stream := range existingStreams {
		if stream.WebSocket.Topic == e.conf.Ethconnect.Topic {
			streamID = stream.ID
		}
	}

	if streamID == "" {
		newStream := eventStream{
			Name:           e.conf.Ethconnect.Topic,
			ErrorHandling:  "block",
			BatchSize:      config.UintWithDefault(e.conf.Ethconnect.BatchSize, defaultBatchSize),
			BatchTimeoutMS: config.UintWithDefault(e.conf.Ethconnect.BatchTimeoutMS, defaultBatchTimeout),
			Type:           "websocket",
		}
		newStream.WebSocket.Topic = e.conf.Ethconnect.Topic
		res, err = e.client.R().SetResult(&newStream).SetBody(&newStream).Post("/eventstreams")
		if err != nil || !res.IsSuccess() {
			return ffresty.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
		}
		streamID = newStream.ID
	}

	log.L(e.ctx).Infof("Event stream: %s", streamID)

	return e.ensureSusbscriptions(streamID)
}

func (e *Ethereum) ensureSusbscriptions(streamID string) error {

	for eventType, subDesc := range requiredSubscriptions {

		var existingSubs []subscription
		res, err := e.client.R().SetResult(&existingSubs).Get("/subscriptions")
		if err != nil || !res.IsSuccess() {
			return ffresty.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
		}

		var subID = ""
		for _, sub := range existingSubs {
			if sub.Name == eventType {
				subID = sub.ID
			}
		}

		if subID == "" {
			newSub := subscription{
				Name:        e.conf.Ethconnect.Topic,
				Description: subDesc,
				Stream:      streamID,
				FromBlock:   "0",
			}
			res, err = e.client.R().SetResult(&newSub).SetBody(&newSub).Post("/subscriptions")
			if err != nil || !res.IsSuccess() {
				return ffresty.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
			}
			subID = newSub.ID
		}

		log.L(e.ctx).Infof("%s subscription: %s", eventType, subID)

	}
	return nil
}

func (e *Ethereum) SubmitBroadcastBatch(identity string, broadcast blockchain.BroadcastBatch) (txTrackingID string, err error) {
	return "", nil
}
