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
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/restclient"
)

type streamManager struct {
	client *resty.Client
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
	ID        string               `json:"id"`
	Name      string               `json:"name,omitempty"`
	Stream    string               `json:"stream"`
	FromBlock string               `json:"fromBlock"`
	Address   string               `json:"address"`
	Event     ABIElementMarshaling `json:"event"`
}

func (s *streamManager) getEventStreams(ctx context.Context) (streams []*eventStream, err error) {
	res, err := s.client.R().
		SetContext(ctx).
		SetResult(&streams).
		Get("/eventstreams")
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
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
		return nil, restclient.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return &stream, nil
}

func (s *streamManager) updateEventStream(ctx context.Context, topic string, batchSize, batchTimeout uint, eventStreamID string) (*eventStream, error) {
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
		Patch("/eventstreams/" + eventStreamID)
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
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
			stream, err = s.updateEventStream(ctx, topic, batchSize, batchTimeout, stream.ID)
			if err != nil {
				return nil, err
			}
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
		return nil, restclient.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return subs, nil
}

func (s *streamManager) createSubscription(ctx context.Context, location *Location, stream, subName string, abi ABIElementMarshaling) (*subscription, error) {
	sub := subscription{
		Name:      subName,
		Stream:    stream,
		FromBlock: "0",
		Address:   location.Address,
		Event:     abi,
	}
	res, err := s.client.R().
		SetContext(ctx).
		SetBody(&sub).
		SetResult(&sub).
		Post("/subscriptions")
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return &sub, nil
}

func (s *streamManager) deleteSubscription(ctx context.Context, subID string) error {
	res, err := s.client.R().
		SetContext(ctx).
		Delete("/subscriptions/" + subID)
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return nil
}

func (s *streamManager) ensureSubscription(ctx context.Context, instancePath, stream string, abi ABIElementMarshaling) (sub *subscription, err error) {
	// Include a hash of the instance path in the subscription, so if we ever point at a different
	// contract configuration, we re-subscribe from block 0.
	// We don't need full strength hashing, so just use the first 16 chars for readability.
	instanceUniqueHash := hex.EncodeToString(sha256.New().Sum([]byte(instancePath)))[0:16]

	existingSubs, err := s.getSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	subName := fmt.Sprintf("%s_%s", abi.Name, instanceUniqueHash)

	for _, s := range existingSubs {
		if s.Name == subName ||
			/* Check for the plain name we used to use originally, before adding uniqueness qualifier.
			   If one of these very early environments needed a new subscription, the existing one would need to
				 be deleted manually. */
			s.Name == abi.Name {
			sub = s
		}
	}

	location := &Location{
		Address: instancePath,
	}

	if sub == nil {
		if sub, err = s.createSubscription(ctx, location, stream, subName, abi); err != nil {
			return nil, err
		}
	}

	log.L(ctx).Infof("%s subscription: %s", abi.Name, sub.ID)
	return sub, nil
}
