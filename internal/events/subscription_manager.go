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

package events

import (
	"context"
	"regexp"
	"sync"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/definitions"
	"github.com/hyperledger/firefly/internal/events/eifactory"
	"github.com/hyperledger/firefly/internal/events/system"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/events"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type subscription struct {
	definition *fftypes.Subscription

	dispatcherElection chan bool
	eventMatcher       *regexp.Regexp
	messageFilter      *messageFilter
	blockchainFilter   *blockchainFilter
	transactionFilter  *transactionFilter
}

type messageFilter struct {
	groupFilter  *regexp.Regexp
	tagFilter    *regexp.Regexp
	topicsFilter *regexp.Regexp
	authorFilter *regexp.Regexp
}

type blockchainFilter struct {
	nameFilter *regexp.Regexp
}

type transactionFilter struct {
	typeFilter *regexp.Regexp
}

type connection struct {
	id          string
	transport   string
	matcher     events.SubscriptionMatcher
	dispatchers map[fftypes.UUID]*eventDispatcher
	ei          events.Plugin
}

type subscriptionManager struct {
	ctx                       context.Context
	database                  database.Plugin
	data                      data.Manager
	eventNotifier             *eventNotifier
	definitions               definitions.DefinitionHandlers
	transports                map[string]events.Plugin
	connections               map[string]*connection
	mux                       sync.Mutex
	maxSubs                   uint64
	durableSubs               map[fftypes.UUID]*subscription
	cancelCtx                 func()
	newOrUpdatedSubscriptions chan *fftypes.UUID
	deletedSubscriptions      chan *fftypes.UUID
	cel                       *changeEventListener
	retry                     retry.Retry
}

func newSubscriptionManager(ctx context.Context, di database.Plugin, dm data.Manager, en *eventNotifier, sh definitions.DefinitionHandlers) (*subscriptionManager, error) {
	ctx, cancelCtx := context.WithCancel(ctx)
	sm := &subscriptionManager{
		ctx:                       ctx,
		database:                  di,
		data:                      dm,
		transports:                make(map[string]events.Plugin),
		connections:               make(map[string]*connection),
		durableSubs:               make(map[fftypes.UUID]*subscription),
		newOrUpdatedSubscriptions: make(chan *fftypes.UUID),
		deletedSubscriptions:      make(chan *fftypes.UUID),
		maxSubs:                   uint64(config.GetUint(config.SubscriptionMax)),
		cancelCtx:                 cancelCtx,
		eventNotifier:             en,
		definitions:               sh,
		retry: retry.Retry{
			InitialDelay: config.GetDuration(config.SubscriptionsRetryInitialDelay),
			MaximumDelay: config.GetDuration(config.SubscriptionsRetryMaxDelay),
			Factor:       config.GetFloat64(config.SubscriptionsRetryFactor),
		},
	}
	sm.cel = newChangeEventListener(ctx)

	err := sm.loadTransports()
	if err == nil {
		err = sm.initTransports()
	}
	return sm, err
}

func (sm *subscriptionManager) loadTransports() error {
	var err error
	enabledTransports := config.GetStringSlice(config.EventTransportsEnabled)
	uniqueTransports := make(map[string]bool)
	for _, transport := range enabledTransports {
		uniqueTransports[transport] = true
	}
	// Cannot disable the internal listener
	uniqueTransports[system.SystemEventsTransport] = true
	for transport := range uniqueTransports {
		sm.transports[transport], err = eifactory.GetPlugin(sm.ctx, transport)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sm *subscriptionManager) initTransports() error {
	var err error
	for _, ei := range sm.transports {
		prefix := config.NewPluginConfig("events").SubPrefix(ei.Name())
		ei.InitPrefix(prefix)
		err = ei.Init(sm.ctx, prefix, &boundCallbacks{sm: sm, ei: ei})
		if err != nil {
			return err
		}
	}
	return nil
}

func (sm *subscriptionManager) start() error {
	fb := database.SubscriptionQueryFactory.NewFilter(sm.ctx)
	filter := fb.And().Limit(sm.maxSubs)
	persistedSubs, _, err := sm.database.GetSubscriptions(sm.ctx, filter)
	if err != nil {
		return err
	}
	sm.mux.Lock()
	defer sm.mux.Unlock()
	for _, subDef := range persistedSubs {
		newSub, err := sm.parseSubscriptionDef(sm.ctx, subDef)
		if err != nil {
			// Warn and continue startup
			log.L(sm.ctx).Warnf("Failed to reload subscription %s:%s [%s]: %s", subDef.Namespace, subDef.Name, subDef.ID, err)
			continue
		}
		sm.durableSubs[*subDef.ID] = newSub
		for _, conn := range sm.connections {
			sm.matchSubToConnLocked(conn, newSub)
		}
	}
	log.L(sm.ctx).Infof("Subscription manager started - loaded %d durable subscriptions", len(sm.durableSubs))
	go sm.subscriptionEventListener()
	go sm.cel.changeEventListener()
	return nil
}

func (sm *subscriptionManager) subscriptionEventListener() {
	for {
		select {
		case id := <-sm.newOrUpdatedSubscriptions:
			go sm.newOrUpdatedDurableSubscription(id)
		case id := <-sm.deletedSubscriptions:
			go sm.deletedDurableSubscription(id)
		case <-sm.ctx.Done():
			return
		}
	}
}

func (sm *subscriptionManager) newOrUpdatedDurableSubscription(id *fftypes.UUID) {
	var subDef *fftypes.Subscription
	err := sm.retry.Do(sm.ctx, "retrieve subscription", func(attempt int) (retry bool, err error) {
		subDef, err = sm.database.GetSubscriptionByID(sm.ctx, id)
		return err != nil, err // indefinite retry
	})
	if err != nil || subDef == nil {
		// either the context was cancelled (so we're closing), or the subscription no longer exists
		log.L(sm.ctx).Infof("Unable to process new subscription event for id=%s (%v)", id, err)
		return
	}

	log.L(sm.ctx).Infof("Created subscription %s:%s [%s]", subDef.Namespace, subDef.Name, subDef.ID)

	newSub, err := sm.parseSubscriptionDef(sm.ctx, subDef)
	if err != nil {
		// Swallow this, as the subscription is simply invalid
		log.L(sm.ctx).Errorf("Subscription rejected by subscription manager: %s", err)
		return
	}

	// Now we're ready to update our locked state, adding this subscription to our
	// in-memory table, and creating any missing dispatchers
	sm.mux.Lock()
	defer sm.mux.Unlock()
	if existingSub, ok := sm.durableSubs[*subDef.ID]; ok {
		if existingSub.definition.Updated.Equal(newSub.definition.Updated) {
			log.L(sm.ctx).Infof("Subscription already active")
			return
		}
		// Need to close the old one
		loaded, dispatchers := sm.closeDurabeSubscriptionLocked(subDef.ID)
		if loaded {
			// Outside the lock, close out the active dispatchers
			sm.mux.Unlock()
			for _, dispatcher := range dispatchers {
				dispatcher.close()
			}
			sm.mux.Lock()
		}
	}
	sm.durableSubs[*subDef.ID] = newSub
	for _, conn := range sm.connections {
		sm.matchSubToConnLocked(conn, newSub)
	}
}

func (sm *subscriptionManager) closeDurabeSubscriptionLocked(id *fftypes.UUID) (bool, []*eventDispatcher) {
	var dispatchers []*eventDispatcher
	// Remove it from the list of durable subs (if there)
	_, loaded := sm.durableSubs[*id]
	if loaded {
		delete(sm.durableSubs, *id)
		// Find any active dispatchers, while we're in the lock, and remove them
		for _, conn := range sm.connections {
			dispatcher, ok := conn.dispatchers[*id]
			if ok {
				dispatchers = append(dispatchers, dispatcher)
				delete(conn.dispatchers, *id)
			}
		}
	}
	return loaded, dispatchers
}

func (sm *subscriptionManager) deletedDurableSubscription(id *fftypes.UUID) {
	sm.mux.Lock()
	loaded, dispatchers := sm.closeDurabeSubscriptionLocked(id)
	sm.mux.Unlock()

	log.L(sm.ctx).Infof("Cleaning up subscription %s loaded=%t dispatchers=%d", id, loaded, len(dispatchers))

	// Outside the lock, close out the active dispatchers
	for _, dispatcher := range dispatchers {
		dispatcher.close()
	}
	// Delete the offsets, as the durable subscriptions are gone
	err := sm.database.DeleteOffset(sm.ctx, fftypes.OffsetTypeSubscription, id.String())
	if err != nil {
		log.L(sm.ctx).Errorf("Failed to cleanup subscription offset: %s", err)
	}
}

func (sm *subscriptionManager) parseSubscriptionDef(ctx context.Context, subDef *fftypes.Subscription) (sub *subscription, err error) {
	filter := subDef.Filter

	transport, ok := sm.transports[subDef.Transport]
	if !ok {
		return nil, i18n.NewError(ctx, i18n.MsgUnknownEventTransportPlugin, subDef.Transport)
	}

	if err := transport.ValidateOptions(&subDef.Options); err != nil {
		return nil, err
	}

	var eventFilter *regexp.Regexp
	if filter.Events != "" {
		eventFilter, err = regexp.Compile(filter.Events)
		if err != nil {
			return nil, i18n.WrapError(ctx, err, i18n.MsgRegexpCompileFailed, "filter.events", filter.Events)
		}
	}

	sub = &subscription{
		dispatcherElection: make(chan bool, 1),
		definition:         subDef,
		eventMatcher:       eventFilter,
	}

	if (filter.Message != fftypes.MessageFilter{}) {
		var tagFilter *regexp.Regexp
		if filter.Message.Tag != "" {
			tagFilter, err = regexp.Compile(filter.Message.Tag)
			if err != nil {
				return nil, i18n.WrapError(ctx, err, i18n.MsgRegexpCompileFailed, "filter.Message.tag", filter.Message.Tag)
			}
		}

		var groupFilter *regexp.Regexp
		if filter.Message.Group != "" {
			groupFilter, err = regexp.Compile(filter.Message.Group)
			if err != nil {
				return nil, i18n.WrapError(ctx, err, i18n.MsgRegexpCompileFailed, "filter.Message.group", filter.Message.Group)
			}
		}

		var topicsFilter *regexp.Regexp
		if filter.Message.Topics != "" {
			topicsFilter, err = regexp.Compile(filter.Message.Topics)
			if err != nil {
				return nil, i18n.WrapError(ctx, err, i18n.MsgRegexpCompileFailed, "filter.Message.topics", filter.Message.Topics)
			}
		}

		var authorFilter *regexp.Regexp
		if filter.Message.Author != "" {
			authorFilter, err = regexp.Compile(filter.Message.Author)
			if err != nil {
				return nil, i18n.WrapError(ctx, err, i18n.MsgRegexpCompileFailed, "filter.Message.author", filter.Message.Author)
			}
		}

		msg := &messageFilter{
			tagFilter:    tagFilter,
			groupFilter:  groupFilter,
			topicsFilter: topicsFilter,
			authorFilter: authorFilter,
		}
		sub.messageFilter = msg
	}

	if (filter.BlockchainEvent != fftypes.BlockchainEventFilter{}) {
		var nameFilter *regexp.Regexp
		if filter.BlockchainEvent.Name != "" {
			nameFilter, err = regexp.Compile(filter.BlockchainEvent.Name)
			if err != nil {
				return nil, i18n.WrapError(ctx, err, i18n.MsgRegexpCompileFailed, "filter.BlockchainEvent.name", filter.BlockchainEvent.Name)
			}
		}

		bf := &blockchainFilter{
			nameFilter: nameFilter,
		}
		sub.blockchainFilter = bf
	}

	if (filter.Transaction != fftypes.TransactionFilter{}) {
		var typeFilter *regexp.Regexp
		if filter.Transaction.Type != "" {
			typeFilter, err = regexp.Compile(filter.Transaction.Type)
			if err != nil {
				return nil, i18n.WrapError(ctx, err, i18n.MsgRegexpCompileFailed, "filter.Transaction.type", filter.Transaction.Type)
			}
		}

		tf := &transactionFilter{
			typeFilter: typeFilter,
		}
		sub.transactionFilter = tf
	}

	return sub, err
}

func (sm *subscriptionManager) close() {
	sm.mux.Lock()
	conns := make([]*connection, 0, len(sm.connections))
	for _, conn := range sm.connections {
		conns = append(conns, conn)
	}
	sm.mux.Unlock()
	for _, conn := range conns {
		sm.connnectionClosed(conn.ei, conn.id)
	}
}

func (sm *subscriptionManager) getCreateConnLocked(ei events.Plugin, connID string) *connection {
	conn, ok := sm.connections[connID]
	if !ok {
		conn = &connection{
			id:          connID,
			transport:   ei.Name(),
			dispatchers: make(map[fftypes.UUID]*eventDispatcher),
			ei:          ei,
		}
		sm.connections[connID] = conn
		log.L(sm.ctx).Debugf("Registered connection %s for %s", conn.id, ei.Name())
	}
	return conn
}

func (sm *subscriptionManager) registerConnection(ei events.Plugin, connID string, matcher events.SubscriptionMatcher) error {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	// Check if there are existing dispatchers
	conn := sm.getCreateConnLocked(ei, connID)
	if conn.ei != ei {
		return i18n.NewError(sm.ctx, i18n.MsgMismatchedTransport, connID, ei.Name(), conn.ei.Name())
	}

	// Update the matcher for this connection ID
	conn.matcher = matcher

	// Make sure we don't have dispatchers now for any that don't match
	for subID, d := range conn.dispatchers {
		if !d.subscription.definition.Ephemeral && !conn.matcher(d.subscription.definition.SubscriptionRef) {
			d.close()
			delete(conn.dispatchers, subID)
		}
	}
	// Make new dispatchers for all durable subscriptions that match
	for _, sub := range sm.durableSubs {
		sm.matchSubToConnLocked(conn, sub)
	}

	return nil
}

func (sm *subscriptionManager) matchSubToConnLocked(conn *connection, sub *subscription) {
	if conn == nil || sub == nil || sub.definition == nil || conn.matcher == nil {
		log.L(sm.ctx).Warnf("Invalid connection/subscription registered: conn=%+v sub=%+v", conn, sub)
		return
	}
	if conn.transport == sub.definition.Transport && conn.matcher(sub.definition.SubscriptionRef) {
		if _, ok := conn.dispatchers[*sub.definition.ID]; !ok {
			dispatcher := newEventDispatcher(sm.ctx, conn.ei, sm.database, sm.data, sm.definitions, conn.id, sub, sm.eventNotifier, sm.cel)
			conn.dispatchers[*sub.definition.ID] = dispatcher
			dispatcher.start()
		}
	}
}

func (sm *subscriptionManager) ephemeralSubscription(ei events.Plugin, connID, namespace string, filter *fftypes.SubscriptionFilter, options *fftypes.SubscriptionOptions) error {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	conn := sm.getCreateConnLocked(ei, connID)

	if conn.ei != ei {
		return i18n.NewError(sm.ctx, i18n.MsgMismatchedTransport, connID, ei.Name(), conn.ei.Name())
	}

	subID := fftypes.NewUUID()
	subDefinition := &fftypes.Subscription{
		SubscriptionRef: fftypes.SubscriptionRef{
			ID:        subID,
			Name:      subID.String(),
			Namespace: namespace,
		},
		Transport: ei.Name(),
		Ephemeral: true,
		Filter:    *filter,
		Options:   *options,
		Created:   fftypes.Now(),
	}

	newSub, err := sm.parseSubscriptionDef(sm.ctx, subDefinition)
	if err != nil {
		return err
	}

	// Create the dispatcher, and start immediately
	dispatcher := newEventDispatcher(sm.ctx, ei, sm.database, sm.data, sm.definitions, connID, newSub, sm.eventNotifier, sm.cel)
	dispatcher.start()

	conn.dispatchers[*subID] = dispatcher

	log.L(sm.ctx).Infof("Created new %s ephemeral subscription %s:%s for connID=%s", ei.Name(), namespace, subID, connID)

	return nil
}

func (sm *subscriptionManager) connnectionClosed(ei events.Plugin, connID string) {
	sm.mux.Lock()
	conn, ok := sm.connections[connID]
	if ok && conn.ei != ei {
		log.L(sm.ctx).Warnf(i18n.ExpandWithCode(sm.ctx, i18n.MsgMismatchedTransport, connID, ei.Name(), conn.ei.Name()))
		sm.mux.Unlock()
		return
	}
	delete(sm.connections, connID)
	sm.mux.Unlock()

	if !ok {
		log.L(sm.ctx).Debugf("Connections already disposed: %s", connID)
		return
	}
	log.L(sm.ctx).Debugf("Closing %d dispatcher(s) for connection '%s'", len(conn.dispatchers), connID)
	for _, d := range conn.dispatchers {
		d.close()
	}
}

func (sm *subscriptionManager) deliveryResponse(ei events.Plugin, connID string, inflight *fftypes.EventDeliveryResponse) {
	sm.mux.Lock()
	var dispatcher *eventDispatcher
	conn, ok := sm.connections[connID]
	if ok && inflight.Subscription.ID != nil {
		dispatcher = conn.dispatchers[*inflight.Subscription.ID]
	}
	if ok && conn.ei != ei {
		err := i18n.NewError(sm.ctx, i18n.MsgMismatchedTransport, connID, ei.Name(), conn.ei.Name())
		log.L(sm.ctx).Errorf("Invalid DeliveryResponse callback from plugin: %s", err)
		sm.mux.Unlock()
		return
	}
	if dispatcher == nil {
		err := i18n.NewError(sm.ctx, i18n.MsgConnSubscriptionNotStarted, inflight.Subscription.ID)
		log.L(sm.ctx).Errorf("Invalid DeliveryResponse callback from plugin: %s", err)
		sm.mux.Unlock()
		return
	}
	sm.mux.Unlock()
	dispatcher.deliveryResponse(inflight)
}
