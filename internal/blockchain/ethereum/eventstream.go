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
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
)

type streamManager struct {
	client       *resty.Client
	cache        cache.CInterface
	batchSize    uint
	batchTimeout uint
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
	ID               string            `json:"id"`
	Name             string            `json:"name,omitempty"`
	Stream           string            `json:"stream"`
	FromBlock        string            `json:"fromBlock"`
	EthCompatAddress string            `json:"address,omitempty"`
	EthCompatEvent   *abi.Entry        `json:"event,omitempty"`
	Filters          []fftypes.JSONAny `json:"filters"`
	subscriptionCheckpoint
}

type subscriptionCheckpoint struct {
	Checkpoint ListenerCheckpoint `json:"checkpoint,omitempty"`
	Catchup    bool               `json:"catchup,omitempty"`
}

func newStreamManager(client *resty.Client, cache cache.CInterface, batchSize, batchTimeout uint) *streamManager {
	return &streamManager{
		client:       client,
		cache:        cache,
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
	}
}

func (s *streamManager) getEventStreams(ctx context.Context) (streams []*eventStream, err error) {
	res, err := s.client.R().
		SetContext(ctx).
		SetResult(&streams).
		Get("/eventstreams")
	if err != nil || !res.IsSuccess() {
		return nil, ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgEthConnectorRESTErr)
	}
	return streams, nil
}

func buildEventStream(topic string, batchSize, batchTimeout uint) *eventStream {
	return &eventStream{
		Name:           topic,
		ErrorHandling:  "block",
		BatchSize:      batchSize,
		BatchTimeoutMS: batchTimeout,
		Type:           "websocket",
		// Some implementations require a "topic" to be set separately, while others rely only on the name.
		// We set them to the same thing for cross compatibility.
		WebSocket:  eventStreamWebsocket{Topic: topic},
		Timestamps: true,
	}
}

func (s *streamManager) createEventStream(ctx context.Context, topic string) (*eventStream, error) {
	stream := buildEventStream(topic, s.batchSize, s.batchTimeout)
	res, err := s.client.R().
		SetContext(ctx).
		SetBody(stream).
		SetResult(stream).
		Post("/eventstreams")
	if err != nil || !res.IsSuccess() {
		return nil, ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgEthConnectorRESTErr)
	}
	return stream, nil
}

func (s *streamManager) updateEventStream(ctx context.Context, topic string, batchSize, batchTimeout uint, eventStreamID string) (*eventStream, error) {
	stream := buildEventStream(topic, batchSize, batchTimeout)
	res, err := s.client.R().
		SetContext(ctx).
		SetBody(stream).
		SetResult(stream).
		Patch("/eventstreams/" + eventStreamID)
	if err != nil || !res.IsSuccess() {
		return nil, ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgEthConnectorRESTErr)
	}
	return stream, nil
}

func (s *streamManager) ensureEventStream(ctx context.Context, topic, pluginTopic string) (*eventStream, error) {
	existingStreams, err := s.getEventStreams(ctx)
	if err != nil {
		return nil, err
	}
	for _, stream := range existingStreams {
		if stream.Name == topic {
			stream, err = s.updateEventStream(ctx, topic, s.batchSize, s.batchTimeout, stream.ID)
			if err != nil {
				return nil, err
			}
			return stream, nil
		}
		if stream.Name == pluginTopic {
			// We have an old event stream that needs to get deleted
			if err := s.deleteEventStream(ctx, stream.ID, false); err != nil {
				return nil, err
			}
		}
	}
	return s.createEventStream(ctx, topic)
}

func (s *streamManager) deleteEventStream(ctx context.Context, esID string, okNotFound bool) error {
	res, err := s.client.R().
		SetContext(ctx).
		Delete("/eventstreams/" + esID)
	if err != nil || !res.IsSuccess() {
		if okNotFound && res.StatusCode() == 404 {
			return nil
		}
		return ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgEthConnectorRESTErr)
	}
	return nil
}

func (s *streamManager) getSubscriptions(ctx context.Context) (subs []*subscription, err error) {
	res, err := s.client.R().
		SetContext(ctx).
		SetResult(&subs).
		Get("/subscriptions")
	if err != nil || !res.IsSuccess() {
		return nil, ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgEthConnectorRESTErr)
	}
	return subs, nil
}

func (s *streamManager) getSubscription(ctx context.Context, subID string, okNotFound bool) (sub *subscription, err error) {
	res, err := s.client.R().
		SetContext(ctx).
		SetResult(&sub).
		Get(fmt.Sprintf("/subscriptions/%s", subID))
	if err != nil || !res.IsSuccess() {
		if okNotFound && res.StatusCode() == 404 {
			return nil, nil
		}
		return nil, ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgEthConnectorRESTErr)
	}
	return sub, nil
}

func (s *streamManager) getSubscriptionName(ctx context.Context, subID string) (string, error) {
	if cachedValue := s.cache.GetString("sub:" + subID); cachedValue != "" {
		return cachedValue, nil
	}

	sub, err := s.getSubscription(ctx, subID, false)
	if err != nil {
		return "", err
	}
	s.cache.SetString("sub:"+subID, sub.Name)
	return sub.Name, nil
}

func (s *streamManager) createSubscription(ctx context.Context, location *Location, stream, subName, firstEvent string, abi *abi.Entry) (*subscription, error) {
	// Map FireFly "firstEvent" values to Ethereum "fromBlock" values
	switch firstEvent {
	case string(core.SubOptsFirstEventOldest):
		firstEvent = "0"
	case string(core.SubOptsFirstEventNewest):
		firstEvent = "latest"
	}
	sub := subscription{
		Name:           subName,
		Stream:         stream,
		FromBlock:      firstEvent,
		EthCompatEvent: abi,
	}

	if location != nil {
		sub.EthCompatAddress = location.Address
	}

	res, err := s.client.R().
		SetContext(ctx).
		SetBody(&sub).
		SetResult(&sub).
		Post("/subscriptions")
	if err != nil || !res.IsSuccess() {
		return nil, ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgEthConnectorRESTErr)
	}
	return &sub, nil
}

func (s *streamManager) deleteSubscription(ctx context.Context, subID string, okNotFound bool) error {
	res, err := s.client.R().
		SetContext(ctx).
		Delete("/subscriptions/" + subID)
	if err != nil || !res.IsSuccess() {
		if okNotFound && res.StatusCode() == 404 {
			return nil
		}
		return ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgEthConnectorRESTErr)
	}
	return nil
}

func (s *streamManager) ensureFireFlySubscription(ctx context.Context, namespace string, version int, instancePath, firstEvent, stream string, abi *abi.Entry) (sub *subscription, err error) {
	// Include a hash of the instance path in the subscription, so if we ever point at a different
	// contract configuration, we re-subscribe from block 0.
	// We don't need full strength hashing, so just use the first 16 chars for readability.
	instanceUniqueHash := hex.EncodeToString(sha256.New().Sum([]byte(instancePath)))[0:16]

	existingSubs, err := s.getSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	legacyName := abi.Name
	v1Name := fmt.Sprintf("%s_%s", abi.Name, instanceUniqueHash)
	v2Name := fmt.Sprintf("%s_%s_%s", namespace, abi.Name, instanceUniqueHash)

	for _, s := range existingSubs {
		if s.Stream == stream {
			/* Check for the deprecated names, before adding namespace uniqueness qualifier.
			   NOTE: If one of these early environments needed a new subscription, the existing one would need to
				 be deleted manually. */
			if version == 1 {
				if s.Name == legacyName {
					log.L(ctx).Warnf("Subscription %s uses a legacy name format '%s' - expected '%s' instead", s.ID, legacyName, v1Name)
					return s, nil
				} else if s.Name == v1Name {
					return s, nil
				}
			} else {
				if s.Name == legacyName || s.Name == v1Name {
					return nil, i18n.NewError(ctx, coremsgs.MsgInvalidSubscriptionForNetwork, s.Name, version)
				} else if s.Name == v2Name {
					return s, nil
				}
			}
		}
	}

	name := v2Name
	if version == 1 {
		name = v1Name
	}
	location := &Location{Address: instancePath}
	if sub, err = s.createSubscription(ctx, location, stream, name, firstEvent, abi); err != nil {
		return nil, err
	}
	log.L(ctx).Infof("%s subscription: %s", abi.Name, sub.ID)
	return sub, nil
}
