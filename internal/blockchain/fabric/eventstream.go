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

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/restclient"
)

type streamManager struct {
	client *resty.Client
	signer string
}

type eventStream struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	ErrorHandling  string               `json:"errorHandling"`
	BatchSize      uint                 `json:"batchSize"`
	BatchTimeoutMS uint                 `json:"batchTimeoutMS"`
	Type           string               `json:"type"`
	WebSocket      eventStreamWebsocket `json:"websocket"`
	Timestamps     bool                 `json:"timestamps"`
}

type subscription struct {
	ID        string      `json:"id"`
	Name      string      `json:"name,omitempty"`
	Channel   string      `json:"channel"`
	Signer    string      `json:"signer"`
	Stream    string      `json:"stream"`
	FromBlock string      `json:"fromBlock"`
	Filter    eventFilter `json:"filter"`
}

type eventFilter struct {
	ChaincodeID string `json:"chaincodeId"`
	EventFilter string `json:"eventFilter"`
}

func (s *streamManager) getEventStreams(ctx context.Context) (streams []*eventStream, err error) {
	res, err := s.client.R().
		SetContext(ctx).
		SetResult(&streams).
		Get("/eventstreams")
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(ctx, res, err, i18n.MsgFabconnectRESTErr)
	}
	return streams, nil
}

func (s *streamManager) createEventStream(ctx context.Context, topic string, batchSize, batchTimeout uint) (*eventStream, error) {
	stream := eventStream{
		Name:           topic,
		ErrorHandling:  "block",
		BatchSize:      batchSize,
		BatchTimeoutMS: batchTimeout,
		Type:           "websocket",
		WebSocket:      eventStreamWebsocket{Topic: topic},
		Timestamps:     true,
	}
	res, err := s.client.R().
		SetContext(ctx).
		SetBody(&stream).
		SetResult(&stream).
		Post("/eventstreams")
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(ctx, res, err, i18n.MsgFabconnectRESTErr)
	}
	return &stream, nil
}

func (s *streamManager) ensureEventStream(ctx context.Context, topic string, batchSize, batchTimeout uint) (*eventStream, error) {
	existingStreams, err := s.getEventStreams(ctx)
	if err != nil {
		return nil, err
	}
	for _, stream := range existingStreams {
		if stream.WebSocket.Topic == topic {
			return stream, nil
		}
	}
	return s.createEventStream(ctx, topic, batchSize, batchTimeout)
}

func (s *streamManager) getSubscriptions(ctx context.Context) (subs []*subscription, err error) {
	res, err := s.client.R().
		SetContext(ctx).
		SetResult(&subs).
		Get("/subscriptions")
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(ctx, res, err, i18n.MsgFabconnectRESTErr)
	}
	return subs, nil
}

func (s *streamManager) createSubscription(ctx context.Context, location *Location, stream, name, event string) (*subscription, error) {
	sub := subscription{
		Name:    name,
		Channel: location.Channel,
		Signer:  s.signer,
		Stream:  stream,
		Filter: eventFilter{
			ChaincodeID: location.Chaincode,
			EventFilter: event,
		},
		FromBlock: "0",
	}
	res, err := s.client.R().
		SetContext(ctx).
		SetBody(&sub).
		SetResult(&sub).
		Post("/subscriptions")
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(ctx, res, err, i18n.MsgFabconnectRESTErr)
	}
	return &sub, nil
}

func (s *streamManager) deleteSubscription(ctx context.Context, subID string) error {
	res, err := s.client.R().
		SetContext(ctx).
		Delete("/subscriptions/" + subID)
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgFabconnectRESTErr)
	}
	return nil
}

func (s *streamManager) ensureSubscription(ctx context.Context, location *Location, stream, event string) (sub *subscription, err error) {
	existingSubs, err := s.getSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	subName := event
	for _, s := range existingSubs {
		if s.Stream == stream && s.Name == subName {
			sub = s
		}
	}

	if sub == nil {
		if sub, err = s.createSubscription(ctx, location, stream, subName, event); err != nil {
			return nil, err
		}
	}

	log.L(ctx).Infof("%s subscription: %s", event, sub.ID)
	return sub, nil
}
