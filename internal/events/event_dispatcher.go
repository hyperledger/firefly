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
	"fmt"
	"sync"

	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/privatemessaging"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/config"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/events"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/i18n"
	"github.com/hyperledger/firefly/pkg/log"
)

const (
	maxReadAhead = 65536
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
	broadcast     broadcast.Manager
	messaging     privatemessaging.Manager
	elected       bool
	eventPoller   *eventPoller
	inflight      map[fftypes.UUID]*fftypes.Event
	eventDelivery chan *fftypes.EventDelivery
	mux           sync.Mutex
	namespace     string
	readAhead     int
	subscription  *subscription
	txHelper      txcommon.Helper
}

func newEventDispatcher(ctx context.Context, ei events.Plugin, di database.Plugin, dm data.Manager, bm broadcast.Manager, pm privatemessaging.Manager, connID string, sub *subscription, en *eventNotifier, txHelper txcommon.Helper) *eventDispatcher {
	ctx, cancelCtx := context.WithCancel(ctx)
	readAhead := config.GetUint(coreconfig.SubscriptionDefaultsReadAhead)
	if sub.definition.Options.ReadAhead != nil {
		readAhead = uint(*sub.definition.Options.ReadAhead)
	}
	if readAhead > maxReadAhead {
		readAhead = maxReadAhead
	}
	ed := &eventDispatcher{
		ctx: log.WithLogField(log.WithLogField(ctx,
			"role", fmt.Sprintf("ed[%s]", connID)),
			"sub", fmt.Sprintf("%s/%s:%s", sub.definition.ID, sub.definition.Namespace, sub.definition.Name)),
		database:      di,
		transport:     ei,
		broadcast:     bm,
		messaging:     pm,
		data:          dm,
		connID:        connID,
		cancelCtx:     cancelCtx,
		subscription:  sub,
		namespace:     sub.definition.Namespace,
		inflight:      make(map[fftypes.UUID]*fftypes.Event),
		eventDelivery: make(chan *fftypes.EventDelivery, readAhead+1),
		readAhead:     int(readAhead),
		acksNacks:     make(chan ackNack),
		closed:        make(chan struct{}),
		txHelper:      txHelper,
	}

	pollerConf := &eventPollerConf{
		eventBatchSize:             config.GetInt(coreconfig.EventDispatcherBufferLength),
		eventBatchTimeout:          config.GetDuration(coreconfig.EventDispatcherBatchTimeout),
		eventPollTimeout:           config.GetDuration(coreconfig.EventDispatcherPollTimeout),
		startupOffsetRetryAttempts: 0, // We need to keep trying to start indefinitely
		retry: retry.Retry{
			InitialDelay: config.GetDuration(coreconfig.EventDispatcherRetryInitDelay),
			MaximumDelay: config.GetDuration(coreconfig.EventDispatcherRetryMaxDelay),
			Factor:       config.GetFloat64(coreconfig.EventDispatcherRetryFactor),
		},
		namespace:  sub.definition.Namespace,
		offsetType: fftypes.OffsetTypeSubscription,
		offsetName: sub.definition.ID.String(),
		addCriteria: func(af database.AndFilter) database.AndFilter {
			return af.Condition(af.Builder().Eq("namespace", sub.definition.Namespace))
		},
		queryFactory:     database.EventQueryFactory,
		getItems:         ed.getEvents,
		newEventsHandler: ed.bufferedDelivery,
		ephemeral:        sub.definition.Ephemeral,
		firstEvent:       sub.definition.Options.FirstEvent,
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
		defer func() {
			// Unelect ourselves on close, to let another dispatcher in
			<-ed.subscription.dispatcherElection
		}()
	case <-ed.ctx.Done():
		l.Debugf("Closed before we became leader")
		return
	}
	// We're ready to go - not
	ed.elected = true
	ed.eventPoller.start()
	go ed.deliverEvents()
	// Wait until the event poller closes
	<-ed.eventPoller.closed
}

func (ed *eventDispatcher) getEvents(ctx context.Context, filter database.Filter, offset int64) ([]fftypes.LocallySequenced, error) {
	log.L(ctx).Tracef("Reading page of events > %d (first events would be %d)", offset, offset+1)
	events, _, err := ed.database.GetEvents(ctx, filter)
	ls := make([]fftypes.LocallySequenced, len(events))
	for i, e := range events {
		ls[i] = e
	}
	return ls, err
}

func (ed *eventDispatcher) enrichEvents(events []fftypes.LocallySequenced) ([]*fftypes.EventDelivery, error) {
	enriched := make([]*fftypes.EventDelivery, len(events))
	for i, ls := range events {
		e := ls.(*fftypes.Event)
		enrichedEvent, err := ed.txHelper.EnrichEvent(ed.ctx, e)
		if err != nil {
			return nil, err
		}
		enriched[i] = &fftypes.EventDelivery{
			EnrichedEvent: *enrichedEvent,
			Subscription:  ed.subscription.definition.SubscriptionRef,
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
		tx := event.Transaction
		be := event.BlockchainEvent
		tag := ""
		topic := event.Topic
		group := ""
		author := ""
		txType := ""
		beName := ""
		beListener := ""

		if msg != nil {
			tag = msg.Header.Tag
			author = msg.Header.Author
			if msg.Header.Group != nil {
				group = msg.Header.Group.String()
			}
		}

		if tx != nil {
			txType = tx.Type.String()
		}

		if be != nil {
			beName = be.Name
			beListener = be.Listener.String()
		}

		if filter.topicFilter != nil {
			topicsMatch := false
			if filter.topicFilter.MatchString(topic) {
				topicsMatch = true
			}
			if !topicsMatch {
				continue
			}
		}

		if filter.messageFilter != nil {
			if filter.messageFilter.tagFilter != nil && !filter.messageFilter.tagFilter.MatchString(tag) {
				continue
			}
			if filter.messageFilter.authorFilter != nil && !filter.messageFilter.authorFilter.MatchString(author) {
				continue
			}
			if filter.messageFilter.groupFilter != nil && !filter.messageFilter.groupFilter.MatchString(group) {
				continue
			}
		}

		if filter.transactionFilter != nil {
			if filter.transactionFilter.typeFilter != nil && !filter.transactionFilter.typeFilter.MatchString(txType) {
				continue
			}
		}

		if filter.blockchainFilter != nil {
			if filter.blockchainFilter.nameFilter != nil && !filter.blockchainFilter.nameFilter.MatchString(beName) {
				continue
			}
			if filter.blockchainFilter.listenerFilter != nil && !filter.blockchainFilter.listenerFilter.MatchString(beListener) {
				continue
			}
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

		l.Debugf("Dispatcher event state: readahead=%d candidates=%d matched=%d inflight=%d queued=%d dispatched=%d dispatchable=%d lastAck=%d nacks=%d highest=%d",
			ed.readAhead, len(candidates), matchCount, inflightCount, len(matching), dispatched, len(disapatchable), lastAck, nacks, highestOffset)

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
			return false, i18n.NewError(ed.ctx, coremsgs.MsgDispatcherClosing)
		case an := <-ed.acksNacks:
			if an.isNack {
				nacks++
				ed.handleNackOffsetUpdate(an)
			} else if nacks == 0 {
				ed.handleAckOffsetUpdate(an)
				lastAck = an.offset
			}
		}
	}
	if nacks == 0 && lastAck != highestOffset {
		ed.eventPoller.commitOffset(highestOffset)
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
		ed.eventPoller.rewindPollingOffset(nack.offset - 1)
	}
	ed.inflight = map[fftypes.UUID]*fftypes.Event{}
}

func (ed *eventDispatcher) handleAckOffsetUpdate(ack ackNack) {
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
		ed.eventPoller.commitOffset(ack.offset)
	}
}

func (ed *eventDispatcher) deliverEvents() {
	withData := ed.subscription.definition.Options.WithData != nil && *ed.subscription.definition.Options.WithData
	for {
		select {
		case event, ok := <-ed.eventDelivery:
			if !ok {
				return
			}
			log.L(ed.ctx).Debugf("Dispatching %s event: %.10d/%s [%s]: ref=%s/%s", ed.transport.Name(), event.Sequence, event.ID, event.Type, event.Namespace, event.Reference)
			var data []*fftypes.Data
			var err error
			if withData && event.Message != nil {
				data, _, err = ed.data.GetMessageDataCached(ed.ctx, event.Message)
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
		ed.sendReply(ed.ctx, event, response.Reply)
	}

	l.Debugf("Response for %s event: %.10d/%s [%s]: ref=%s/%s rejected=%t info='%s'", ed.transport.Name(), event.Sequence, event.ID, event.Type, event.Namespace, event.Reference, response.Rejected, response.Info)
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
		close(ed.eventDelivery)
		ed.elected = false
	}
}
