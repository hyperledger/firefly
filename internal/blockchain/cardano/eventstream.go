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

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly/internal/coremsgs"
)

type streamManager struct {
	client       *resty.Client
	batchSize    uint
	batchTimeout uint
}

type eventStream struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ErrorHandling  string `json:"errorHandling"`
	BatchSize      uint   `json:"batchSize"`
	BatchTimeoutMS uint   `json:"batchTimeoutMS"`
	Type           string `json:"type"`
	Timestamps     bool   `json:"timestamps"`
}

func newStreamManager(client *resty.Client, batchSize, batchTimeout uint) *streamManager {
	return &streamManager{
		client:       client,
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
		return nil, ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgCardanoconnectRESTErr)
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
		Timestamps:     true,
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
		return nil, ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgCardanoconnectRESTErr)
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
		return nil, ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgCardanoconnectRESTErr)
	}
	return stream, nil
}

func (s *streamManager) ensureEventStream(ctx context.Context, topic string) (*eventStream, error) {
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
	}
	return s.createEventStream(ctx, topic)
}
