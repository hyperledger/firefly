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
	"github.com/kaleido-io/firefly/internal/ffresty"
	"github.com/kaleido-io/firefly/internal/log"
)

type Ethereum struct {
	ctx    context.Context
	conf   *Config
	events blockchain.Events
	client *resty.Client
}

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

// var subscriptionInfo = map[string]string{
// 	"AssetInstanceBatchCreated": "Asset instance batch created",
// }

func (e *Ethereum) ensureEventStreams() error {
	// TODO: complete setup
	return nil
}

func (e *Ethereum) SubmitBroadcastBatch(identity string, broadcast blockchain.BroadcastBatch) (txTrackingID string, err error) {
	return "", nil
}
