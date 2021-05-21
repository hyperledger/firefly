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

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/events/eifactory"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/retry"
	"github.com/kaleido-io/firefly/pkg/blockchain"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/publicstorage"
)

type EventManager interface {
	blockchain.Callbacks

	NewEvents() chan<- int64
	Start() error
	WaitStop()
}

type eventManager struct {
	ctx           context.Context
	publicstorage publicstorage.Plugin
	database      database.Plugin
	subManagers   map[string]*subscriptionManager
	retry         retry.Retry
	aggregator    *aggregator
	eventNotifier *eventNotifier
}

func NewEventManager(ctx context.Context, pi publicstorage.Plugin, di database.Plugin) (EventManager, error) {
	if pi == nil || di == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	en := newEventNotifier(ctx)
	em := &eventManager{
		ctx:           log.WithLogField(ctx, "role", "event-manager"),
		publicstorage: pi,
		database:      di,
		subManagers:   make(map[string]*subscriptionManager),
		retry: retry.Retry{
			InitialDelay: config.GetDuration(config.EventAggregatorRetryInitDelay),
			MaximumDelay: config.GetDuration(config.EventAggregatorRetryMaxDelay),
			Factor:       config.GetFloat64(config.EventAggregatorRetryFactor),
		},
		eventNotifier: en,
		aggregator:    newAggregator(ctx, di, en),
	}

	enabledTransports := config.GetStringSlice(config.EventTransportsEnabled)
	for _, transport := range enabledTransports {
		et, err := eifactory.GetPlugin(ctx, transport)
		if err == nil {
			em.subManagers[transport], err = newSubscriptionManager(ctx, di, et, en)
		}
		if err != nil {
			return nil, err
		}
	}

	return em, nil
}

func (em *eventManager) Start() (err error) {
	for _, sm := range em.subManagers {
		err = sm.start()
	}
	if err == nil {
		err = em.aggregator.start()
	}
	return err
}

func (em *eventManager) NewEvents() chan<- int64 {
	return em.eventNotifier.newEvents
}

func (em *eventManager) WaitStop() {
	for _, sm := range em.subManagers {
		sm.close()
	}
	<-em.aggregator.eventPoller.closed
}
