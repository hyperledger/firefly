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

package fabric

import (
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/restclient"
)

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
	ID          string      `json:"id"`
	Description string      `json:"description"`
	Name        string      `json:"name"`
	Channel     string      `json:"channel"`
	Signer      string      `json:"signer"`
	Stream      string      `json:"stream"`
	FromBlock   string      `json:"fromBlock"`
	Filter      eventFilter `json:"filter"`
}

func (f *Fabric) ensureEventStreams(fabconnectConf config.Prefix) error {

	var existingStreams []*eventStream
	res, err := f.client.R().SetContext(f.ctx).SetResult(&existingStreams).Get("/eventstreams")
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(f.ctx, res, err, i18n.MsgFabconnectRESTErr)
	}

	for _, stream := range existingStreams {
		if stream.WebSocket.Topic == f.topic {
			f.initInfo.stream = stream
		}
	}

	if f.initInfo.stream == nil {
		newStream := eventStream{
			Name:           f.topic,
			ErrorHandling:  "block",
			BatchSize:      fabconnectConf.GetUint(FabconnectConfigBatchSize),
			BatchTimeoutMS: uint(fabconnectConf.GetDuration(FabconnectConfigBatchTimeout).Milliseconds()),
			Type:           "websocket",
		}
		newStream.WebSocket.Topic = f.topic
		res, err = f.client.R().SetBody(&newStream).SetResult(&newStream).Post("/eventstreams")
		if err != nil || !res.IsSuccess() {
			return restclient.WrapRestErr(f.ctx, res, err, i18n.MsgFabconnectRESTErr)
		}
		f.initInfo.stream = &newStream
	}

	log.L(f.ctx).Infof("Event stream: %s", f.initInfo.stream.ID)

	return f.ensureSusbscriptions(f.initInfo.stream.ID)
}

func (f *Fabric) ensureSusbscriptions(streamID string) error {
	for eventType, subDesc := range requiredSubscriptions {

		var existingSubs []*subscription
		res, err := f.client.R().SetResult(&existingSubs).Get("/subscriptions")
		if err != nil || !res.IsSuccess() {
			return restclient.WrapRestErr(f.ctx, res, err, i18n.MsgFabconnectRESTErr)
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
				Channel:     f.defaultChannel,
				Signer:      f.signer,
				Stream:      streamID,
			}
			newSub.Filter.ChaincodeID = f.chaincode
			newSub.Filter.EventFilter = "BatchPin"

			res, err = f.client.R().
				SetContext(f.ctx).
				SetBody(&newSub).
				SetResult(&newSub).
				Post("/subscriptions")
			if err != nil || !res.IsSuccess() {
				return restclient.WrapRestErr(f.ctx, res, err, i18n.MsgFabconnectRESTErr)
			}
			sub = &newSub
		}

		log.L(f.ctx).Infof("%s subscription: %s", eventType, sub.ID)
		f.initInfo.subs = append(f.initInfo.subs, sub)

	}
	return nil
}
