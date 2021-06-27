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

package events

import (
	"context"
	"database/sql/driver"
	"fmt"
	"sync"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/data"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/retry"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/events"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

type ackNack struct {
	id     fftypes.UUID
	isNack bool
	offset int64
}

type eventDispatcher struct {
	acksNacks     chan ackNack
	cancelCtx     func()
	closed        chan struct{}
	connID        string
	ctx           context.Context
	data          data.Manager
	database      database.Plugin
	transport     events.Plugin
	rs            *replySender
	elected       bool
	eventPoller   *eventPoller
	inflight      map[fftypes.UUID]*fftypes.Event
	eventDelivery chan *fftypes.EventDelivery
	mux           sync.Mutex
	namespace     string
	readAhead     int
	subscription  *subscription
}

func newEventDispatcher(ctx context.Context, ei events.Plugin, di database.Plugin, dm data.Manager, rs *replySender, connID string, sub *subscription, en *eventNotifier) *eventDispatcher {
	ctx, cancelCtx := context.WithCancel(ctx)
	readAhead := int(config.GetUint(config.SubscriptionDefaultsReadAhead))
	ed := &eventDispatcher{
		ctx: log.WithLogField(log.WithLogField(ctx,
			"role", fmt.Sprintf("ed[%s]", connID)),
			"sub", fmt.Sprintf("%s/%s:%s", sub.definition.ID, sub.definition.Namespace, sub.definition.Name)),
		database:      di,
		transport:     ei,
		rs:            rs,
		data:          dm,
		connID:        connID,
		cancelCtx:     cancelCtx,
		subscription:  sub,
		namespace:     sub.definition.Namespace,
		inflight:      make(map[fftypes.UUID]*fftypes.Event),
		eventDelivery: make(chan *fftypes.EventDelivery, readAhead+1),
		readAhead:     readAhead,
		acksNacks:     make(chan ackNack),
		closed:        make(chan struct{}),
	}

	pollerConf := &eventPollerConf{
		eventBatchSize:             config.GetInt(config.EventDispatcherBufferLength),
		eventBatchTimeout:          config.GetDuration(config.EventDispatcherBatchTimeout),
		eventPollTimeout:           config.GetDuration(config.EventDispatcherPollTimeout),
		startupOffsetRetryAttempts: 0, // We need to keep trying to start indefinitely
		retry: retry.Retry{
			InitialDelay: config.GetDuration(config.EventDispatcherRetryInitDelay),
			MaximumDelay: config.GetDuration(config.EventDispatcherRetryMaxDelay),
			Factor:       config.GetFloat64(config.EventDispatcherRetryFactor),
		},
		offsetType:      fftypes.OffsetTypeSubscription,
		offsetNamespace: sub.definition.Namespace,
		offsetName:      sub.definition.Name,
		addCriteria: func(af database.AndFilter) database.AndFilter {
			return af.Condition(af.Builder().Eq("namespace", sub.definition.Namespace))
		},
		queryFactory:     database.EventQueryFactory,
		getItems:         ed.getEvents,
		newEventsHandler: ed.bufferedDelivery,
		ephemeral:        sub.definition.Ephemeral,
		firstEvent:       sub.definition.Options.FirstEvent,
	}
	if sub.definition.Options.ReadAhead != nil {
		ed.readAhead = int(*sub.definition.Options.ReadAhead)
	}

	ed.eventPoller = newEventPoller(ctx, di, en, pollerConf)
	return ed
}

func (ed *eventDispatcher) start() {
	go ed.electAndStart()
}

func (ed *eventDispatcher) electAndStart() {
	defer close(ed.closed)
	l := log.L(ed.ctx)
	l.Debugf("Dispatcher attempting to become leader")
	select {
	case ed.subscription.dispatcherElection <- true:
		l.Debugf("Dispatcher became leader")
	case <-ed.ctx.Done():
		l.Debugf("Closed before we became leader")
		return
	}
	// We're ready to go - not
	ed.elected = true
	go ed.deliverEvents()
	go func() {
		err := ed.eventPoller.start()
		l.Debugf("Event dispatcher completed: %v", err)
	}()
	// Wait until we close
	<-ed.eventPoller.closed
	// Unelect ourselves on close, to let another dispatcher in
	<-ed.subscription.dispatcherElection
}

func (ed *eventDispatcher) getEvents(ctx context.Context, filter database.Filter) ([]fftypes.LocallySequenced, error) {
	events, err := ed.database.GetEvents(ctx, filter)
	ls := make([]fftypes.LocallySequenced, len(events))
	for i, e := range events {
		ls[i] = e
	}
	return ls, err
}

func (ed *eventDispatcher) enrichEvents(events []fftypes.LocallySequenced) ([]*fftypes.EventDelivery, error) {
	// We need all the messages that match event references
	refIDs := make([]driver.Value, len(events))
	for i, ls := range events {
		e := ls.(*fftypes.Event)
		if e.Reference != nil {
			refIDs[i] = *e.Reference
		}
	}

	mfb := database.MessageQueryFactory.NewFilter(ed.ctx)
	msgFilter := mfb.And(
		mfb.In("id", refIDs),
		mfb.Eq("namespace", ed.namespace),
	)
	msgs, err := ed.database.GetMessages(ed.ctx, msgFilter)
	if err != nil {
		return nil, err
	}

	enriched := make([]*fftypes.EventDelivery, len(events))
	for i, ls := range events {
		e := ls.(*fftypes.Event)
		enriched[i] = &fftypes.EventDelivery{
			Event:        *e,
			Subscription: ed.subscription.definition.SubscriptionRef,
		}
		for _, msg := range msgs {
			if *e.Reference == *msg.Header.ID {
				enriched[i].Message = msg
				break
			}
		}
	}

	return enriched, nil

}

func (ed *eventDispatcher) filterEvents(candidates []*fftypes.EventDelivery) []*fftypes.EventDelivery {
	matchingEvents := make([]*fftypes.EventDelivery, 0, len(candidates))
	for _, event := range candidates {
		filter := ed.subscription
		if filter.eventMatcher != nil && !filter.eventMatcher.MatchString(string(event.Type)) {
			continue
		}
		msg := event.Message
		tag := ""
		group := ""
		author := ""
		var topics []string
		if msg != nil {
			tag = msg.Header.Tag
			topics = msg.Header.Topics
			author = msg.Header.Author
			if msg.Header.Group != nil {
				group = msg.Header.Group.String()
			}
		}
		if filter.tagFilter != nil && !filter.tagFilter.MatchString(tag) {
			continue
		}
		if filter.authorFilter != nil && !filter.authorFilter.MatchString(author) {
			continue
		}
		if filter.topicsFilter != nil {
			topicsMatch := false
			for _, topic := range topics {
				if filter.topicsFilter.MatchString(topic) {
					topicsMatch = true
					break
				}
			}
			if !topicsMatch {
				continue
			}
		}
		if filter.groupFilter != nil && !filter.groupFilter.MatchString(group) {
			continue
		}
		matchingEvents = append(matchingEvents, event)
	}
	return matchingEvents
}

func (ed *eventDispatcher) bufferedDelivery(events []fftypes.LocallySequenced) (bool, error) {
	// At this point, the page of messages we've been given are loaded from the DB into memory,
	// but we can only make them in-flight and push them to the client up to the maximum
	// readahead (which is likely lower than our page size - 1 by default)

	if len(events) == 0 {
		return false, nil
	}
	highestOffset := events[len(events)-1].LocalSequence()
	var lastAck int64
	var nacks int

	l := log.L(ed.ctx)
	candidates, err := ed.enrichEvents(events)
	if err != nil {
		return false, err
	}

	matching := ed.filterEvents(candidates)
	matchCount := len(matching)
	dispatched := 0

	// We stay here blocked until we've consumed all the messages in the buffer,
	// or a reset event happens
	for {
		ed.mux.Lock()
		var disapatchable []*fftypes.EventDelivery
		inflightCount := len(ed.inflight)
		maxDispatch := 1 + ed.readAhead - inflightCount
		if maxDispatch >= len(matching) {
			disapatchable = matching
			matching = nil
		} else if maxDispatch > 0 {
			disapatchable = matching[0:maxDispatch]
			matching = matching[maxDispatch:]
		}
		ed.mux.Unlock()

		l.Debugf("Dispatcher event state: candidates=%d matched=%d inflight=%d queued=%d dispatched=%d dispatchable=%d lastAck=%d nacks=%d highest=%d",
			len(candidates), matchCount, inflightCount, len(matching), dispatched, len(disapatchable), lastAck, nacks, highestOffset)

		for _, event := range disapatchable {
			ed.mux.Lock()
			ed.inflight[*event.ID] = &event.Event
			inflightCount = len(ed.inflight)
			ed.mux.Unlock()

			dispatched++
			ed.eventDelivery <- event
		}

		if inflightCount == 0 {
			// We've cleared the decks. Time to look for more messages
			break
		}

		// Block until we're closed, or woken due to a delivery response
		select {
		case <-ed.ctx.Done():
			return false, i18n.NewError(ed.ctx, i18n.MsgDispatcherClosing)
		case an := <-ed.acksNacks:
			if an.isNack {
				nacks++
				ed.handleNackOffsetUpdate(an)
			} else if nacks == 0 {
				err := ed.handleAckOffsetUpdate(an)
				if err != nil {
					return false, err
				}
				lastAck = an.offset
			}
		}
	}
	if nacks == 0 && lastAck != highestOffset {
		err := ed.eventPoller.commitOffset(ed.ctx, highestOffset)
		if err != nil {
			return false, err
		}
	}
	return true, nil // poll again straight away for more messages
}

func (ed *eventDispatcher) handleNackOffsetUpdate(nack ackNack) {
	ed.mux.Lock()
	defer ed.mux.Unlock()
	// If we're rejected, we need to redeliver all messages from this offset onwards,
	// even if we've delivered messages after that.
	// That means resetting the polling offest, and clearing out all our state
	delete(ed.inflight, nack.id)
	if ed.eventPoller.pollingOffset > nack.offset {
		ed.eventPoller.rewindPollingOffset(nack.offset)
	}
	ed.inflight = map[fftypes.UUID]*fftypes.Event{}
}

func (ed *eventDispatcher) handleAckOffsetUpdate(ack ackNack) error {
	oldOffset := ed.eventPoller.getPollingOffset()
	ed.mux.Lock()
	delete(ed.inflight, ack.id)
	lowestInflight := int64(-1)
	for _, inflight := range ed.inflight {
		if lowestInflight < 0 || inflight.Sequence < lowestInflight {
			lowestInflight = inflight.Sequence
		}
	}
	ed.mux.Unlock()
	if (lowestInflight == -1 || lowestInflight > ack.offset) && ack.offset > oldOffset {
		// This was the lowest in flight, and we can move the offset forwards
		return ed.eventPoller.commitOffset(ed.ctx, ack.offset)
	}
	return nil
}

func (ed *eventDispatcher) deliverEvents() {
	withData := ed.subscription.definition.Options.WithData != nil && *ed.subscription.definition.Options.WithData
	for {
		select {
		case event, ok := <-ed.eventDelivery:
			if !ok {
				return
			}
			log.L(ed.ctx).Debugf("Dispatching event: %.10d/%s [%s]: ref=%s/%s", event.Sequence, event.ID, event.Type, event.Namespace, event.Reference)
			var data []*fftypes.Data
			var err error
			if withData && event.Message != nil {
				data, _, err = ed.data.GetMessageData(ed.ctx, event.Message, true)
			}
			if err == nil {
				err = ed.transport.DeliveryRequest(ed.connID, ed.subscription.definition, event, data)
			}
			if err != nil {
				ed.deliveryResponse(&fftypes.EventDeliveryResponse{ID: event.ID, Rejected: true})
			}
		case <-ed.ctx.Done():
			return
		}
	}
}
func (ed *eventDispatcher) deliveryResponse(response *fftypes.EventDeliveryResponse) {
	l := log.L(ed.ctx)

	ed.mux.Lock()
	var an ackNack
	event, found := ed.inflight[*response.ID]
	if found {
		an.id = *response.ID
		an.offset = event.Sequence
		an.isNack = response.Rejected
	}
	ed.mux.Unlock()

	// Do some extra logging and persistent actions now we're out of lock
	if !found {
		l.Warnf("Response for event not in flight: %s rejected=%t info='%s' (likely previous reject)", response.ID, response.Rejected, response.Info)
		return
	}

	// We might have a message to send, do that before we dispatch the ack
	// Note a failure to send the reply does not invalidate the ack
	if response.Reply != nil {
		ed.rs.sendReply(ed.ctx, event, response.Reply)
	}

	l.Debugf("Response for event: %.10d/%s [%s]: ref=%s/%s rejected=%t info='%s'", event.Sequence, event.ID, event.Type, event.Namespace, event.Reference, response.Rejected, response.Info)
	// We don't do any meaningful work in this call, we just set things up so the right thing
	// will happen when the poller wakes up. So we need to pass it over
	select {
	case ed.acksNacks <- an:
	case <-ed.ctx.Done():
		l.Debugf("Delivery response will not be delivered: closing")
		return
	}
}

func (ed *eventDispatcher) close() {
	log.L(ed.ctx).Infof("Dispatcher closing for conn=%s subscription=%s", ed.connID, ed.subscription.definition.ID)
	ed.cancelCtx()
	<-ed.closed
	if ed.elected {
		<-ed.eventPoller.closed
		close(ed.eventDelivery)
		ed.elected = false
	}
}
