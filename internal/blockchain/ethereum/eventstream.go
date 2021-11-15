// Copyright © 2021 Kaleido, Inc.
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
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/restclient"
)

type streamManager struct {
	ctx          context.Context
	client       *resty.Client
	instancePath string
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

type subscription struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Stream    string `json:"stream"`
	FromBlock string `json:"fromBlock"`
}

func (s *streamManager) getEventStreams() (streams []*eventStream, err error) {
	res, err := s.client.R().
		SetContext(s.ctx).
		SetResult(&streams).
		Get("/eventstreams")
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(s.ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return streams, nil
}

func (s *streamManager) createEventStream(topic string, batchSize, batchTimeout uint) (*eventStream, error) {
	stream := eventStream{
		Name:           topic,
		ErrorHandling:  "block",
		BatchSize:      batchSize,
		BatchTimeoutMS: batchTimeout,
		Type:           "websocket",
		WebSocket:      eventStreamWebsocket{Topic: topic},
	}
	res, err := s.client.R().
		SetContext(s.ctx).
		SetBody(&stream).
		SetResult(&stream).
		Post("/eventstreams")
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(s.ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return &stream, nil
}

func (s *streamManager) ensureEventStream(topic string, batchSize, batchTimeout uint) (*eventStream, error) {
	existingStreams, err := s.getEventStreams()
	if err != nil {
		return nil, err
	}
	for _, stream := range existingStreams {
		if stream.WebSocket.Topic == topic {
			return stream, nil
		}
	}
	return s.createEventStream(topic, batchSize, batchTimeout)
}

func (s *streamManager) getSubscriptions() (subs []*subscription, err error) {
	res, err := s.client.R().
		SetContext(s.ctx).
		SetResult(&subs).
		Get("/subscriptions")
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(s.ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return subs, nil
}

func (s *streamManager) createSubscription(name, stream, event string) (*subscription, error) {
	sub := subscription{
		Name:      name,
		Stream:    stream,
		FromBlock: "0",
	}
	res, err := s.client.R().
		SetContext(s.ctx).
		SetBody(&sub).
		SetResult(&sub).
		Post(fmt.Sprintf("%s/%s", s.instancePath, event))
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(s.ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return &sub, nil
}

func (s *streamManager) ensureSubscriptions(stream string, subscriptions []string) (subs []*subscription, err error) {
	// Include a hash of the instance path in the subscription, so if we ever point at a different
	// contract configuration, we re-subscribe from block 0.
	// We don't need full strength hashing, so just use the first 16 chars for readability.
	instanceUniqueHash := hex.EncodeToString(sha256.New().Sum([]byte(s.instancePath)))[0:16]

	existingSubs, err := s.getSubscriptions()
	if err != nil {
		return nil, err
	}

	for _, eventType := range subscriptions {
		var sub *subscription
		subName := fmt.Sprintf("%s_%s", eventType, instanceUniqueHash)
		for _, s := range existingSubs {
			if s.Name == subName ||
				/* Check for the plain name we used to use originally, before adding uniqueness qualifier.
				   If one of these very early environments needed a new subscription, the existing one would need to
					 be deleted manually. */
				s.Name == eventType {
				sub = s
			}
		}

		if sub == nil {
			if sub, err = s.createSubscription(subName, stream, eventType); err != nil {
				return nil, err
			}
		}

		log.L(s.ctx).Infof("%s subscription: %s", eventType, sub.ID)
		subs = append(subs, sub)
	}
	return subs, nil
}
