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
	"regexp"
	"sync"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/events"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

type subscription struct {
	definition *fftypes.Subscription

	dispatcherElection chan bool
	eventMatcher       *regexp.Regexp
	topicFilter        *regexp.Regexp
	groupFilter        *regexp.Regexp
	contextFilter      *regexp.Regexp
}

type subscriptionManager struct {
	ctx         context.Context
	database    database.Plugin
	transport   events.Plugin
	dispatchers map[string]map[uuid.UUID]*eventDispatcher
	mux         sync.Mutex
	maxSubs     uint64
	durableSubs []*subscription
	cancelCtx   func()
}

func newSubscriptionManager(ctx context.Context, di database.Plugin, et events.Plugin) (*subscriptionManager, error) {
	ctx, cancelCtx := context.WithCancel(ctx)
	sm := &subscriptionManager{
		ctx:         ctx,
		database:    di,
		transport:   et,
		dispatchers: make(map[string]map[uuid.UUID]*eventDispatcher),
		maxSubs:     uint64(config.GetUint(config.SubscriptionMaxPerTransport)),
		cancelCtx:   cancelCtx,
	}

	// Initialize the transport
	prefix := config.NewPluginConfig("events").SubPrefix(et.Name())
	et.InitConfigPrefix(prefix)
	err := et.Init(ctx, prefix, sm)
	if err != nil {
		return nil, err
	}

	return sm, nil
}

func (sm *subscriptionManager) start() error {
	fb := database.SubscriptionQueryFactory.NewFilter(sm.ctx)
	filter := fb.And(
		fb.Eq("transport", sm.transport),
	).Limit(sm.maxSubs)
	persistedSubs, err := sm.database.GetSubscriptions(sm.ctx, filter)
	if err != nil {
		return err
	}
	sm.mux.Lock()
	defer sm.mux.Unlock()
	for _, subDef := range persistedSubs {
		newSub, err := sm.createSubscription(sm.ctx, subDef)
		if err != nil {
			// Warn and continue startup
			log.L(sm.ctx).Warnf("Failed to reload subscription %s:%s [%s]: %s", subDef.Namespace, subDef.Name, subDef.ID, err)
		}
		sm.durableSubs = append(sm.durableSubs, newSub)
	}
	return nil
}

func (sm *subscriptionManager) createSubscription(ctx context.Context, subDefinition *fftypes.Subscription) (sub *subscription, err error) {
	filter := subDefinition.Filter

	var eventFilter *regexp.Regexp
	if filter.Events != "" {
		eventFilter, err = regexp.Compile(filter.Events)
		if err != nil {
			return nil, i18n.WrapError(ctx, err, i18n.MsgRegexpCompileFailed, "filter.events", filter.Events)
		}
	}

	var topicFilter *regexp.Regexp
	if filter.Topic != "" {
		topicFilter, err = regexp.Compile(filter.Topic)
		if err != nil {
			return nil, i18n.WrapError(ctx, err, i18n.MsgRegexpCompileFailed, "filter.topic", filter.Topic)
		}
	}

	var groupFilter *regexp.Regexp
	if filter.Topic != "" {
		groupFilter, err = regexp.Compile(filter.Group)
		if err != nil {
			return nil, i18n.WrapError(ctx, err, i18n.MsgRegexpCompileFailed, "filter.group", filter.Group)
		}
	}

	var contextFilter *regexp.Regexp
	if filter.Topic != "" {
		contextFilter, err = regexp.Compile(filter.Context)
		if err != nil {
			return nil, i18n.WrapError(ctx, err, i18n.MsgRegexpCompileFailed, "filter.context", filter.Context)
		}
	}

	sub = &subscription{
		dispatcherElection: make(chan bool, 1),
		definition:         subDefinition,
		eventMatcher:       eventFilter,
		topicFilter:        topicFilter,
		groupFilter:        groupFilter,
		contextFilter:      contextFilter,
	}
	return sub, err
}

func (sm *subscriptionManager) close() {
	connIDs := []string{}
	sm.mux.Lock()
	for connID := range sm.dispatchers {
		connIDs = append(connIDs, connID)
	}
	sm.mux.Unlock()
	for _, connID := range connIDs {
		sm.ConnnectionClosed(connID)
	}
}

func (sm *subscriptionManager) RegisterConnection(connID string, matcher events.SubscriptionMatcher) {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	// Check if there are existing dispatchers
	dispatchersForConn, ok := sm.dispatchers[connID]
	if !ok {
		dispatchersForConn = make(map[uuid.UUID]*eventDispatcher)
	}
	// Make sure we don't have dispatchers now for any that don't match
	for subID, d := range dispatchersForConn {
		if !d.subscription.definition.Ephemeral && !matcher(d.subscription.definition.SubscriptionRef) {
			d.close()
			delete(dispatchersForConn, subID)
		}
	}
	// Make new dispatchers for all durable subscriptions that match
	for _, sub := range sm.durableSubs {
		if matcher(sub.definition.SubscriptionRef) {
			if _, ok := dispatchersForConn[*sub.definition.ID]; !ok {
				dispatchersForConn[*sub.definition.ID] = newEventDispatcher(sm.ctx, sm.database, connID, sub)
			}
		}
	}
	sm.dispatchers[connID] = dispatchersForConn

}

func (sm *subscriptionManager) EphemeralSubscription(connID, namespace string, filter fftypes.SubscriptionFilter, options fftypes.SubscriptionOptions) error {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	dispatchersForConn, ok := sm.dispatchers[connID]
	if !ok {
		dispatchersForConn = make(map[uuid.UUID]*eventDispatcher)
		sm.dispatchers[connID] = dispatchersForConn
	}

	subID := fftypes.NewUUID()
	subDefinition := &fftypes.Subscription{
		SubscriptionRef: fftypes.SubscriptionRef{
			ID:        subID,
			Name:      subID.String(),
			Namespace: namespace,
		},
		Transport: sm.transport.Name(),
		Ephemeral: true,
		Filter:    filter,
		Options:   options,
		Created:   fftypes.Now(),
	}

	newSub, err := sm.createSubscription(sm.ctx, subDefinition)
	if err != nil {
		return err
	}

	// Create the dispatcher, and start immediately
	dispatcher := newEventDispatcher(sm.ctx, sm.database, connID, newSub)
	dispatcher.start()

	dispatchersForConn[*subID] = dispatcher
	return nil
}

func (sm *subscriptionManager) ConnnectionClosed(connID string) {
	sm.mux.Lock()
	dispatchersForConn, ok := sm.dispatchers[connID]
	delete(sm.dispatchers, connID)
	sm.mux.Unlock()
	if !ok {
		log.L(sm.ctx).Debugf("Dispatchers already disposed: %s", connID)
	}
	log.L(sm.ctx).Debugf("Closing %d dispatcher(s) for connection '%s'", len(dispatchersForConn), connID)
	for _, d := range dispatchersForConn {
		d.close()
	}
}

func (sm *subscriptionManager) DeliveryResponse(connID string, inflight fftypes.EventDeliveryResponse) error {
	sm.mux.Lock()
	var dispatcher *eventDispatcher
	dispatchersForConn, ok := sm.dispatchers[connID]
	if ok && inflight.Subscription.ID != nil {
		dispatcher = dispatchersForConn[*inflight.Subscription.ID]
	}
	sm.mux.Unlock()

	if dispatcher == nil {
		return i18n.NewError(sm.ctx, i18n.MsgConnSubscriptionNotStarted, inflight.Subscription.ID)
	}

	return dispatcher.deliveryResponse(&inflight)
}
