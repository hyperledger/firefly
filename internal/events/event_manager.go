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

package events

import (
	"context"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/blockchain"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/publicstorage"
	"github.com/kaleido-io/firefly/internal/retry"
)

type EventManager interface {
	blockchain.Events

	NewEvents() chan<- *uuid.UUID
	Start() error
}

type eventManager struct {
	ctx           context.Context
	publicstorage publicstorage.Plugin
	database      database.Plugin
	retry         retry.Retry
	aggregator    *aggregator
}

func NewEventManager(ctx context.Context, pi publicstorage.Plugin, di database.Plugin) EventManager {
	return &eventManager{
		ctx:           log.WithLogField(ctx, "role", "event-manager"),
		publicstorage: pi,
		database:      di,
		retry: retry.Retry{
			InitialDelay: config.GetDuration(config.AggregatorDataReadRetryDelay),
			MaximumDelay: config.GetDuration(config.AggregatorDataReadRetryMaxDelay),
			Factor:       config.GetFloat64(config.AggregatorDataReadRetryFactor),
		},
		aggregator: newAggregator(ctx),
	}
}

func (em *eventManager) Start() error {
	em.aggregator.start()
	return nil
}

func (em *eventManager) NewEvents() chan<- *uuid.UUID {
	return em.aggregator.newEvents
}
