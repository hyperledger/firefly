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
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/restclient"
)

func (e *Ethereum) getEventStreams() (streams []*eventStream, err error) {
	res, err := e.client.R().
		SetContext(e.ctx).
		SetResult(&streams).
		Get("/eventstreams")
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return streams, nil
}

func (e *Ethereum) createEventStream(batchSize, batchTimeout uint) (*eventStream, error) {
	stream := eventStream{
		Name:           e.topic,
		ErrorHandling:  "block",
		BatchSize:      batchSize,
		BatchTimeoutMS: batchTimeout,
		Type:           "websocket",
	}
	stream.WebSocket.Topic = e.topic
	res, err := e.client.R().
		SetBody(&stream).
		SetResult(&stream).
		Post("/eventstreams")
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return &stream, nil
}

func (e *Ethereum) ensureEventStreams(ethconnectConf config.Prefix) error {
	existingStreams, err := e.getEventStreams()
	if err != nil {
		return err
	}

	for _, stream := range existingStreams {
		if stream.WebSocket.Topic == e.topic {
			e.initInfo.stream = stream
		}
	}

	if e.initInfo.stream == nil {
		e.initInfo.stream, err = e.createEventStream(
			ethconnectConf.GetUint(EthconnectConfigBatchSize),
			uint(ethconnectConf.GetDuration(EthconnectConfigBatchTimeout).Milliseconds()),
		)
		if err != nil {
			return err
		}
	}

	log.L(e.ctx).Infof("Event stream: %s", e.initInfo.stream.ID)

	return e.ensureSubscriptions()
}

func (e *Ethereum) getSubscriptions() (subs []*subscription, err error) {
	res, err := e.client.R().
		SetResult(&subs).
		Get("/subscriptions")
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return subs, nil
}

func (e *Ethereum) createSubscription(name, desc, event string) (*subscription, error) {
	sub := subscription{
		Name:        name,
		Description: desc,
		Stream:      e.initInfo.stream.ID,
		FromBlock:   "0",
	}
	res, err := e.client.R().
		SetContext(e.ctx).
		SetBody(&sub).
		SetResult(&sub).
		Post(fmt.Sprintf("%s/%s", e.instancePath, event))
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(e.ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return &sub, nil
}

func (e *Ethereum) ensureSubscriptions() error {
	// Include a hash of the instance path in the subscription, so if we ever point at a different
	// contract configuration, we re-subscribe from block 0.
	// We don't need full strength hashing, so just use the first 16 chars for readability.
	instanceUniqueHash := hex.EncodeToString(sha256.New().Sum([]byte(e.instancePath)))[0:16]

	existingSubs, err := e.getSubscriptions()
	if err != nil {
		return err
	}

	for eventType, subDesc := range requiredSubscriptions {
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
			if sub, err = e.createSubscription(subName, subDesc, eventType); err != nil {
				return err
			}
		}

		log.L(e.ctx).Infof("%s subscription: %s", eventType, sub.ID)
		e.initInfo.subs = append(e.initInfo.subs, sub)
	}
	return nil
}
