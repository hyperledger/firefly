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
	"github.com/kaleido-io/firefly/pkg/eventtransport"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

type subscription struct {
	definition *fftypes.Subscription

	eventMatcher  *regexp.Regexp
	topicFilter   *regexp.Regexp
	groupFilter   *regexp.Regexp
	contextFilter *regexp.Regexp
}

type subscriptionManager struct {
	ctx         context.Context
	database    database.Plugin
	transport   eventtransport.Plugin
	dispatchers map[uuid.UUID]*eventDispatcher
	dispatchMux sync.Mutex
	maxSubs     uint64
	subs        []*subscription
	subsLock    sync.Mutex
	cancelCtx   func()
}

func newSubscriptionManager(ctx context.Context, di database.Plugin, et eventtransport.Plugin) (*subscriptionManager, error) {
	ctx, cancelCtx := context.WithCancel(ctx)
	sm := &subscriptionManager{
		ctx:         ctx,
		database:    di,
		transport:   et,
		dispatchers: make(map[uuid.UUID]*eventDispatcher),
		maxSubs:     uint64(config.GetUint(config.SubscriptionMaxPerTransport)),
		cancelCtx:   cancelCtx,
	}

	// Initialize the transport
	prefix := config.NewPluginConfig("eventtransports").SubPrefix(et.Name())
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
	for _, s := range persistedSubs {
		err := sm.addSubscription(sm.ctx, s)
		if err != nil {
			// Warn and continue startup
			log.L(sm.ctx).Warnf("Failed to reload subscription %s:%s [%s]: %s", s.Namespace, s.Name, s.ID, err)
		}
	}
	return nil
}

func (sm *subscriptionManager) addSubscription(ctx context.Context, subDefinition *fftypes.Subscription) (err error) {
	var eventFilter *regexp.Regexp
	if subDefinition.Filter.Events != "" {
		eventFilter, err = regexp.Compile(subDefinition.Filter.Events)
		if err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgRegexpCompileFailed, "filter.events", subDefinition.Filter.Events)
		}
	}

	var topicFilter *regexp.Regexp
	if subDefinition.Filter.Topic != "" {
		topicFilter, err = regexp.Compile(subDefinition.Filter.Topic)
		if err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgRegexpCompileFailed, "filter.topic", subDefinition.Filter.Topic)
		}
	}

	var groupFilter *regexp.Regexp
	if subDefinition.Filter.Topic != "" {
		groupFilter, err = regexp.Compile(subDefinition.Filter.Group)
		if err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgRegexpCompileFailed, "filter.group", subDefinition.Filter.Group)
		}
	}

	var contextFilter *regexp.Regexp
	if subDefinition.Filter.Topic != "" {
		contextFilter, err = regexp.Compile(subDefinition.Filter.Context)
		if err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgRegexpCompileFailed, "filter.context", subDefinition.Filter.Context)
		}
	}

	sm.subsLock.Lock()
	defer sm.subsLock.Unlock()
	sm.subs = append(sm.subs, &subscription{
		definition:    subDefinition,
		eventMatcher:  eventFilter,
		topicFilter:   topicFilter,
		groupFilter:   groupFilter,
		contextFilter: contextFilter,
	})

	return nil
}

func (sm *subscriptionManager) close() {}

func (sm *subscriptionManager) RegisterDispatcher(namespace string, name string) (dispatcherID uuid.UUID, err error) {
	return uuid.UUID{}, nil
}

// EphemeralSubscription creates a subscription
func (sm *subscriptionManager) EphemeralSubscription(subscription fftypes.Subscription) (dispatcherID uuid.UUID, err error) {
	return uuid.UUID{}, nil
}

// DispatcherClosed is a notification that a dispatcher has closed.
// Note the plugin must not crash if it receives PublishEvent calls after the DispatcherClosed event is fired
func (sm *subscriptionManager) DispatcherClosed(dispatcherID uuid.UUID) {
	sm.dispatchMux.Lock()
	dispatcher, ok := sm.dispatchers[dispatcherID]
	sm.dispatchMux.Unlock()
	if !ok {
		log.L(sm.ctx).Debugf("Closed callback for dispatcher already disposed: %s", dispatcherID)
	}
	dispatcher.close()
}
