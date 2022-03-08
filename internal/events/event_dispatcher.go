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
	"database/sql/driver"
	"fmt"
	"sync"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/definitions"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/events"
	"github.com/hyperledger/firefly/pkg/fftypes"
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
	definitions   definitions.DefinitionHandlers
	elected       bool
	eventPoller   *eventPoller
	inflight      map[fftypes.UUID]*fftypes.Event
	eventDelivery chan *fftypes.EventDelivery
	mux           sync.Mutex
	namespace     string
	readAhead     int
	subscription  *subscription
	cel           *changeEventListener
	changeEvents  chan *fftypes.ChangeEvent
}

func newEventDispatcher(ctx context.Context, ei events.Plugin, di database.Plugin, dm data.Manager, sh definitions.DefinitionHandlers, connID string, sub *subscription, en *eventNotifier, cel *changeEventListener) *eventDispatcher {
	ctx, cancelCtx := context.WithCancel(ctx)
	readAhead := config.GetUint(config.SubscriptionDefaultsReadAhead)
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
		definitions:   sh,
		data:          dm,
		connID:        connID,
		cancelCtx:     cancelCtx,
		subscription:  sub,
		namespace:     sub.definition.Namespace,
		inflight:      make(map[fftypes.UUID]*fftypes.Event),
		eventDelivery: make(chan *fftypes.EventDelivery, readAhead+1),
		changeEvents:  make(chan *fftypes.ChangeEvent),
		readAhead:     int(readAhead),
		acksNacks:     make(chan ackNack),
		closed:        make(chan struct{}),
		cel:           cel,
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

func (ed *eventDispatcher) getEvents(ctx context.Context, filter database.Filter) ([]fftypes.LocallySequenced, error) {
	events, _, err := ed.database.GetEvents(ctx, filter)
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
	msgs, _, err := ed.database.GetMessages(ed.ctx, msgFilter)
	if err != nil {
		return nil, err
	}

	tfb := database.TransactionQueryFactory.NewFilter(ed.ctx)
	txFilter := tfb.And(
		tfb.In("id", refIDs),
		tfb.Eq("namespace", ed.namespace),
	)
	txs, _, err := ed.database.GetTransactions(ed.ctx, txFilter)
	if err != nil {
		return nil, err
	}

	bfb := database.BlockchainEventQueryFactory.NewFilter(ed.ctx)
	beFilter := bfb.And(
		tfb.In("id", refIDs),
		tfb.Eq("namespace", ed.namespace),
	)
	bEvents, _, err := ed.database.GetBlockchainEvents(ed.ctx, beFilter)
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
		for _, tx := range txs {
			if *e.Reference == *tx.ID {
				enriched[i].Transaction = tx
				break
			}
		}
		for _, be := range bEvents {
			if *e.Reference == *be.ID {
				enriched[i].BlockchainEvent = be
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
		tx := event.Transaction
		be := event.BlockchainEvent
		tag := ""
		group := ""
		author := ""
		txType := ""
		beName := ""
		beListener := ""
		var topics []string

		if msg != nil {
			tag = msg.Header.Tag
			topics = msg.Header.Topics
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

		if filter.messageFilter != nil {
			if filter.messageFilter.tagFilter != nil && !filter.messageFilter.tagFilter.MatchString(tag) {
				continue
			}
			if filter.messageFilter.authorFilter != nil && !filter.messageFilter.authorFilter.MatchString(author) {
				continue
			}
			if filter.messageFilter.topicsFilter != nil {
				topicsMatch := false
				for _, topic := range topics {
					if filter.messageFilter.topicsFilter.MatchString(topic) {
						topicsMatch = true
						break
					}
				}
				if !topicsMatch {
					continue
				}
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

func (ed *eventDispatcher) dispatchChangeEvent(ce *fftypes.ChangeEvent) {
	select {
	case ed.changeEvents <- ce:
		log.L(ed.ctx).Tracef("Dispatched change event %+v", ce)
		break
	case <-ed.eventPoller.closed:
		log.L(ed.ctx).Warnf("Dispatcher closed before dispatching change event")
	}
}

func (ed *eventDispatcher) deliverEvents() {
	if ed.transport.Capabilities().ChangeEvents && ed.subscription.definition.Options.ChangeEvents {
		ed.cel.addDispatcher(*ed.subscription.definition.ID, ed)
		defer ed.cel.removeDispatcher(*ed.subscription.definition.ID)
	}
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
				data, _, err = ed.data.GetMessageData(ed.ctx, event.Message, true)
			}
			if err == nil {
				err = ed.transport.DeliveryRequest(ed.connID, ed.subscription.definition, event, data)
			}
			if err != nil {
				ed.deliveryResponse(&fftypes.EventDeliveryResponse{ID: event.ID, Rejected: true})
			}
		case changeEvent := <-ed.changeEvents:
			ws, ok := ed.transport.(events.ChangeEventListener)
			if !ok {
				log.L(ed.ctx).Warnf("Change event received for transport that does not support change events '%s'", ed.transport.Name())
				break
			}
			ws.ChangeEvent(ed.connID, changeEvent)
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
		ed.definitions.SendReply(ed.ctx, event, response.Reply)
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
