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
	"fmt"
	"regexp"
	"testing"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/mocks/datamocks"
	"github.com/hyperledger-labs/firefly/mocks/eventsmocks"
	"github.com/hyperledger-labs/firefly/mocks/syshandlersmocks"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/events"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestEventDispatcher(sub *subscription) (*eventDispatcher, func()) {
	mdi := &databasemocks.Plugin{}
	mei := &eventsmocks.PluginAll{}
	mei.On("Capabilities").Return(&events.Capabilities{ChangeEvents: true}).Maybe()
	mei.On("Name").Return("ut").Maybe()
	mdm := &datamocks.Manager{}
	msh := &syshandlersmocks.SystemHandlers{}
	ctx, cancel := context.WithCancel(context.Background())
	return newEventDispatcher(ctx, mei, mdi, mdm, msh, fftypes.NewUUID().String(), sub, newEventNotifier(ctx, "ut"), newChangeEventListener(ctx)), func() {
		cancel()
		config.Reset()
	}
}

func TestEventDispatcherStartStop(t *testing.T) {
	ten := uint16(10)
	oldest := fftypes.SubOptsFirstEventOldest
	ed, cancel := newTestEventDispatcher(&subscription{
		dispatcherElection: make(chan bool, 1),
		definition: &fftypes.Subscription{
			SubscriptionRef: fftypes.SubscriptionRef{Namespace: "ns1", Name: "sub1"},
			Ephemeral:       true,
			Options: fftypes.SubscriptionOptions{
				SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
					ReadAhead:  &ten,
					FirstEvent: &oldest,
				},
			},
		},
	})
	defer cancel()
	mdi := ed.database.(*databasemocks.Plugin)
	ge := mdi.On("GetEvents", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil, nil)
	confirmedElected := make(chan bool)
	ge.RunFn = func(a mock.Arguments) {
		<-confirmedElected
	}

	assert.Equal(t, int(10), ed.readAhead)
	ed.start()
	confirmedElected <- true
	close(confirmedElected)
	ed.eventPoller.eventNotifier.newEvents <- 12345
	ed.close()
}

func TestMaxReadAhead(t *testing.T) {
	config.Set(config.SubscriptionDefaultsReadAhead, 65537)
	ed, cancel := newTestEventDispatcher(&subscription{
		dispatcherElection: make(chan bool, 1),
		definition: &fftypes.Subscription{
			SubscriptionRef: fftypes.SubscriptionRef{Namespace: "ns1", Name: "sub1"},
			Ephemeral:       true,
			Options:         fftypes.SubscriptionOptions{},
		},
	})
	defer cancel()
	assert.Equal(t, int(65536), ed.readAhead)
}

func TestEventDispatcherLeaderElection(t *testing.T) {
	log.SetLevel("debug")

	subID := fftypes.NewUUID()
	sub := &subscription{
		dispatcherElection: make(chan bool, 1),
		definition: &fftypes.Subscription{
			SubscriptionRef: fftypes.SubscriptionRef{Namespace: "ns1", Name: "sub1", ID: subID},
		},
	}

	ed1, cancel1 := newTestEventDispatcher(sub)
	ed2, cancel2 := newTestEventDispatcher(sub /* same sub */)

	gev1Wait := make(chan bool)
	gev1Done := make(chan struct{})
	mdi1 := ed1.database.(*databasemocks.Plugin)
	gev1 := mdi1.On("GetEvents", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil, nil)
	mdi1.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, subID.String()).Return(&fftypes.Offset{
		Type:    fftypes.OffsetTypeSubscription,
		Name:    subID.String(),
		Current: 12345,
		RowID:   333333,
	}, nil)
	gev1.RunFn = func(a mock.Arguments) {
		gev1Wait <- true
		<-gev1Done
	}

	ed1.start()
	<-gev1Wait
	ed2.start()

	cancel2()
	ed2.close() // while ed1 is active
	close(gev1Done)
	cancel1()

}

func TestEventDispatcherReadAheadOutOfOrderAcks(t *testing.T) {
	log.SetLevel("debug")
	var five = uint16(5)
	subID := fftypes.NewUUID()
	sub := &subscription{
		dispatcherElection: make(chan bool, 1),
		definition: &fftypes.Subscription{
			SubscriptionRef: fftypes.SubscriptionRef{ID: subID, Namespace: "ns1", Name: "sub1"},
			Options: fftypes.SubscriptionOptions{
				SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
					ReadAhead: &five,
				},
			},
		},
		eventMatcher: regexp.MustCompile(fmt.Sprintf("^%s|%s$", fftypes.EventTypeMessageConfirmed, fftypes.EventTypeMessageConfirmed)),
	}

	ed, cancel := newTestEventDispatcher(sub)
	defer cancel()
	go ed.deliverEvents()
	mdi := ed.database.(*databasemocks.Plugin)
	mei := ed.transport.(*eventsmocks.PluginAll)

	eventDeliveries := make(chan *fftypes.EventDelivery)
	deliveryRequestMock := mei.On("DeliveryRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deliveryRequestMock.RunFn = func(a mock.Arguments) {
		eventDeliveries <- a.Get(2).(*fftypes.EventDelivery)
	}

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()
	ref2 := fftypes.NewUUID()
	ev2 := fftypes.NewUUID()
	ref3 := fftypes.NewUUID()
	ev3 := fftypes.NewUUID()
	ref4 := fftypes.NewUUID()
	ev4 := fftypes.NewUUID()

	// Capture offset commits
	offsetUpdates := make(chan int64)
	uof := mdi.On("UpdateOffset", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	uof.RunFn = func(a mock.Arguments) {
		f, err := a.Get(2).(database.Update).Finalize()
		assert.NoError(t, err)
		v, _ := f.SetOperations[0].Value.Value()
		offsetUpdates <- v.(int64)
	}
	// Setup enrichment
	mdi.On("GetMessages", mock.Anything, mock.MatchedBy(func(filter database.Filter) bool {
		fi, err := filter.Finalize()
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf(`( id IN ['%s','%s','%s','%s'] ) && ( namespace == 'ns1' )`, ref1, ref2, ref3, ref4), fi.String())
		return true
	})).Return([]*fftypes.Message{
		{Header: fftypes.MessageHeader{ID: ref1}},
		{Header: fftypes.MessageHeader{ID: ref2}},
		{Header: fftypes.MessageHeader{ID: ref3}},
		{Header: fftypes.MessageHeader{ID: ref4}},
	}, nil, nil)

	// Deliver a batch of messages
	batch1Done := make(chan struct{})
	go func() {
		repoll, err := ed.bufferedDelivery([]fftypes.LocallySequenced{
			&fftypes.Event{ID: ev1, Sequence: 10000001, Reference: ref1, Type: fftypes.EventTypeMessageConfirmed}, // match
			&fftypes.Event{ID: ev2, Sequence: 10000002, Reference: ref2, Type: fftypes.EventTypeMessageRejected},
			&fftypes.Event{ID: ev3, Sequence: 10000003, Reference: ref3, Type: fftypes.EventTypeMessageConfirmed}, // match
			&fftypes.Event{ID: ev4, Sequence: 10000004, Reference: ref4, Type: fftypes.EventTypeMessageConfirmed}, // match
		})
		assert.NoError(t, err)
		assert.True(t, repoll)
		close(batch1Done)
	}()

	// Wait for the two calls to deliver the matching messages to the client (read ahead allows this)
	event1 := <-eventDeliveries
	assert.Equal(t, *ev1, *event1.ID)
	assert.Equal(t, *ref1, *event1.Message.Header.ID)
	event3 := <-eventDeliveries
	assert.Equal(t, *ev3, *event3.ID)
	assert.Equal(t, *ref3, *event3.Message.Header.ID)
	event4 := <-eventDeliveries
	assert.Equal(t, *ev4, *event4.ID)
	assert.Equal(t, *ref4, *event4.Message.Header.ID)

	// Send back the two acks - out of order to validate the read-ahead logic
	go func() {
		ed.deliveryResponse(&fftypes.EventDeliveryResponse{ID: event4.ID})
		ed.deliveryResponse(&fftypes.EventDeliveryResponse{ID: event1.ID})
		ed.deliveryResponse(&fftypes.EventDeliveryResponse{ID: event3.ID})
	}()

	// Confirm we get the offset updates in the correct order, even though the confirmations
	// came in a different order from the app.
	assert.Equal(t, int64(10000001), <-offsetUpdates)
	assert.Equal(t, int64(10000003), <-offsetUpdates)
	assert.Equal(t, int64(10000004), <-offsetUpdates)

	// This should complete the batch
	<-batch1Done

	mdi.AssertExpectations(t)
	mei.AssertExpectations(t)
}

func TestEventDispatcherNoReadAheadInOrder(t *testing.T) {
	log.SetLevel("debug")
	sub := &subscription{
		dispatcherElection: make(chan bool, 1),
		definition: &fftypes.Subscription{
			SubscriptionRef: fftypes.SubscriptionRef{ID: fftypes.NewUUID(), Namespace: "ns1", Name: "sub1"},
			Ephemeral:       true,
			Options:         fftypes.SubscriptionOptions{},
		},
	}

	ed, cancel := newTestEventDispatcher(sub)
	defer cancel()
	go ed.deliverEvents()

	mdi := ed.database.(*databasemocks.Plugin)
	mei := ed.transport.(*eventsmocks.PluginAll)

	eventDeliveries := make(chan *fftypes.EventDelivery)
	deliveryRequestMock := mei.On("DeliveryRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deliveryRequestMock.RunFn = func(a mock.Arguments) {
		eventDeliveries <- a.Get(2).(*fftypes.EventDelivery)
	}

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()
	ref2 := fftypes.NewUUID()
	ev2 := fftypes.NewUUID()
	ref3 := fftypes.NewUUID()
	ev3 := fftypes.NewUUID()
	ref4 := fftypes.NewUUID()
	ev4 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{
		{Header: fftypes.MessageHeader{ID: ref1}},
		{Header: fftypes.MessageHeader{ID: ref2}},
		{Header: fftypes.MessageHeader{ID: ref3}},
		{Header: fftypes.MessageHeader{ID: ref4}},
	}, nil, nil)

	// Deliver a batch of messages
	batch1Done := make(chan struct{})
	go func() {
		repoll, err := ed.bufferedDelivery([]fftypes.LocallySequenced{
			&fftypes.Event{ID: ev1, Sequence: 10000001, Reference: ref1, Type: fftypes.EventTypeMessageConfirmed}, // match
			&fftypes.Event{ID: ev2, Sequence: 10000002, Reference: ref2, Type: fftypes.EventTypeMessageConfirmed}, // match
			&fftypes.Event{ID: ev3, Sequence: 10000003, Reference: ref3, Type: fftypes.EventTypeMessageConfirmed}, // match
			&fftypes.Event{ID: ev4, Sequence: 10000004, Reference: ref4, Type: fftypes.EventTypeMessageConfirmed}, // match
		})
		assert.NoError(t, err)
		assert.True(t, repoll)
		close(batch1Done)
	}()

	// Wait for the two calls to deliver the matching messages to the client (read ahead allows this)
	event1 := <-eventDeliveries
	assert.Equal(t, *ev1, *event1.ID)
	assert.Equal(t, *ref1, *event1.Message.Header.ID)
	select {
	case <-eventDeliveries:
		assert.Fail(t, "should not have read ahead")
	default:
	}
	ed.deliveryResponse(&fftypes.EventDeliveryResponse{ID: event1.ID})

	event2 := <-eventDeliveries
	ed.deliveryResponse(&fftypes.EventDeliveryResponse{ID: event2.ID})

	event3 := <-eventDeliveries
	ed.deliveryResponse(&fftypes.EventDeliveryResponse{ID: event3.ID})

	event4 := <-eventDeliveries
	ed.deliveryResponse(&fftypes.EventDeliveryResponse{ID: event4.ID})

	// This should complete the batch
	<-batch1Done

	mdi.AssertExpectations(t)
	mei.AssertExpectations(t)
}

func TestEventDispatcherChangeEvents(t *testing.T) {
	log.SetLevel("debug")
	sub := &subscription{
		dispatcherElection: make(chan bool, 1),
		definition: &fftypes.Subscription{
			SubscriptionRef: fftypes.SubscriptionRef{ID: fftypes.NewUUID(), Namespace: "ns1", Name: "sub1"},
			Ephemeral:       true,
			Options: fftypes.SubscriptionOptions{
				ChangeEvents: true,
			},
		},
	}

	ed, cancel := newTestEventDispatcher(sub)
	defer cancel()
	go ed.deliverEvents()

	mdi := ed.database.(*databasemocks.Plugin)
	mei := ed.transport.(*eventsmocks.PluginAll)

	changeEvents := make(chan *fftypes.ChangeEvent)
	deliveryRequestMock := mei.On("ChangeEvent", mock.Anything, mock.Anything).Return()
	deliveryRequestMock.RunFn = func(a mock.Arguments) {
		changeEvents <- a.Get(1).(*fftypes.ChangeEvent)
	}

	go func() {
		ed.dispatchChangeEvent(&fftypes.ChangeEvent{
			Collection: "widgets",
		})
	}()

	ce := <-changeEvents
	assert.Equal(t, "widgets", ce.Collection)

	mdi.AssertExpectations(t)
	mei.AssertExpectations(t)
}

func TestEventDispatcherChangeEventsNotSupported(t *testing.T) {
	log.SetLevel("debug")
	sub := &subscription{
		dispatcherElection: make(chan bool, 1),
		definition: &fftypes.Subscription{
			SubscriptionRef: fftypes.SubscriptionRef{ID: fftypes.NewUUID(), Namespace: "ns1", Name: "sub1"},
			Ephemeral:       true,
			Options: fftypes.SubscriptionOptions{
				ChangeEvents: true,
			},
		},
	}

	ed, cancel := newTestEventDispatcher(sub)
	defer cancel()

	mei := &eventsmocks.Plugin{}
	mei.On("Name").Return("ut")
	mei.On("Capabilities").Return(&events.Capabilities{})
	ed.transport = mei
	go ed.deliverEvents()

	ed.dispatchChangeEvent(&fftypes.ChangeEvent{
		Collection: "widgets",
	})
}

func TestEnrichEventsFailGetMessages(t *testing.T) {

	sub := &subscription{
		definition: &fftypes.Subscription{},
	}
	ed, cancel := newTestEventDispatcher(sub)
	defer cancel()

	mdi := ed.database.(*databasemocks.Plugin)
	mdi.On("GetMessages", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	id1 := fftypes.NewUUID()
	_, err := ed.enrichEvents([]fftypes.LocallySequenced{&fftypes.Event{ID: id1}})

	assert.EqualError(t, err, "pop")
}

func TestFilterEventsMatch(t *testing.T) {

	sub := &subscription{
		definition: &fftypes.Subscription{},
	}
	ed, cancel := newTestEventDispatcher(sub)
	defer cancel()

	gid1 := fftypes.NewRandB32()
	id1 := fftypes.NewUUID()
	id2 := fftypes.NewUUID()
	id3 := fftypes.NewUUID()
	events := ed.filterEvents([]*fftypes.EventDelivery{
		{
			Event: fftypes.Event{
				ID:   id1,
				Type: fftypes.EventTypeMessageConfirmed,
			},
			Message: &fftypes.Message{
				Header: fftypes.MessageHeader{
					Topics: fftypes.FFNameArray{"topic1"},
					Tag:    "tag1",
					Group:  nil,
					Identity: fftypes.Identity{
						Author: "signingOrg",
						Key:    "0x12345",
					},
				},
			},
		},
		{
			Event: fftypes.Event{
				ID:   id2,
				Type: fftypes.EventTypeMessageConfirmed,
			},
			Message: &fftypes.Message{
				Header: fftypes.MessageHeader{
					Topics: fftypes.FFNameArray{"topic1"},
					Tag:    "tag2",
					Group:  gid1,
					Identity: fftypes.Identity{
						Author: "org2",
						Key:    "0x23456",
					},
				},
			},
		},
		{
			Event: fftypes.Event{
				ID:   id3,
				Type: fftypes.EventTypeMessageRejected,
			},
			Message: &fftypes.Message{
				Header: fftypes.MessageHeader{
					Topics: fftypes.FFNameArray{"topic2"},
					Tag:    "tag1",
					Group:  nil,
					Identity: fftypes.Identity{
						Author: "signingOrg",
						Key:    "0x12345",
					},
				},
			},
		},
	})

	ed.subscription.eventMatcher = regexp.MustCompile(fmt.Sprintf("^%s$", fftypes.EventTypeMessageConfirmed))
	ed.subscription.topicsFilter = regexp.MustCompile(".*")
	ed.subscription.tagFilter = regexp.MustCompile(".*")
	ed.subscription.groupFilter = regexp.MustCompile(".*")
	matched := ed.filterEvents(events)
	assert.Equal(t, 2, len(matched))
	assert.Equal(t, *id1, *matched[0].ID)
	assert.Equal(t, *id2, *matched[1].ID)
	// id three has the wrong event type

	ed.subscription.eventMatcher = nil
	ed.subscription.topicsFilter = nil
	ed.subscription.tagFilter = nil
	ed.subscription.groupFilter = nil
	matched = ed.filterEvents(events)
	assert.Equal(t, 3, len(matched))
	assert.Equal(t, *id1, *matched[0].ID)
	assert.Equal(t, *id2, *matched[1].ID)
	assert.Equal(t, *id3, *matched[2].ID)

	ed.subscription.topicsFilter = regexp.MustCompile("topic1")
	matched = ed.filterEvents(events)
	assert.Equal(t, 2, len(matched))
	assert.Equal(t, *id1, *matched[0].ID)
	assert.Equal(t, *id2, *matched[1].ID)

	ed.subscription.topicsFilter = nil
	ed.subscription.tagFilter = regexp.MustCompile("tag2")
	matched = ed.filterEvents(events)
	assert.Equal(t, 1, len(matched))
	assert.Equal(t, *id2, *matched[0].ID)

	ed.subscription.topicsFilter = nil
	ed.subscription.authorFilter = nil
	ed.subscription.groupFilter = regexp.MustCompile(gid1.String())
	matched = ed.filterEvents(events)
	assert.Equal(t, 1, len(matched))
	assert.Equal(t, *id2, *matched[0].ID)

	ed.subscription.groupFilter = regexp.MustCompile("^$")
	matched = ed.filterEvents(events)
	assert.Equal(t, 0, len(matched))

	ed.subscription.groupFilter = nil
	ed.subscription.topicsFilter = nil
	ed.subscription.tagFilter = nil
	ed.subscription.authorFilter = regexp.MustCompile("0x23456")
	matched = ed.filterEvents(events)
	assert.Equal(t, 1, len(matched))
	assert.Equal(t, *id2, *matched[0].ID)

}

func TestBufferedDeliveryNoEvents(t *testing.T) {

	sub := &subscription{
		definition: &fftypes.Subscription{},
	}
	ed, cancel := newTestEventDispatcher(sub)
	defer cancel()

	repoll, err := ed.bufferedDelivery([]fftypes.LocallySequenced{})
	assert.False(t, repoll)
	assert.Nil(t, err)

}

func TestBufferedDeliveryEnrichFail(t *testing.T) {

	sub := &subscription{
		definition: &fftypes.Subscription{},
	}
	ed, cancel := newTestEventDispatcher(sub)
	defer cancel()

	mdi := ed.database.(*databasemocks.Plugin)
	mdi.On("GetMessages", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	repoll, err := ed.bufferedDelivery([]fftypes.LocallySequenced{&fftypes.Event{ID: fftypes.NewUUID()}})
	assert.False(t, repoll)
	assert.EqualError(t, err, "pop")

}

func TestBufferedDeliveryClosedContext(t *testing.T) {

	sub := &subscription{
		definition: &fftypes.Subscription{},
	}
	ed, cancel := newTestEventDispatcher(sub)
	go ed.deliverEvents()
	cancel()

	mdi := ed.database.(*databasemocks.Plugin)
	mei := ed.transport.(*eventsmocks.PluginAll)
	mdi.On("GetMessages", mock.Anything, mock.Anything).Return(nil, nil, nil)
	mdi.On("GetDataRefs", mock.Anything, mock.Anything).Return(nil, nil, nil)
	mei.On("DeliveryRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	repoll, err := ed.bufferedDelivery([]fftypes.LocallySequenced{&fftypes.Event{ID: fftypes.NewUUID()}})
	assert.False(t, repoll)
	assert.Regexp(t, "FF10182", err)

}

func TestBufferedDeliveryNackRewind(t *testing.T) {

	sub := &subscription{
		definition: &fftypes.Subscription{},
	}
	ed, cancel := newTestEventDispatcher(sub)
	defer cancel()
	go ed.deliverEvents()

	mdi := ed.database.(*databasemocks.Plugin)
	mei := ed.transport.(*eventsmocks.PluginAll)
	mdi.On("GetMessages", mock.Anything, mock.Anything).Return(nil, nil, nil)
	mdi.On("GetDataRefs", mock.Anything, mock.Anything).Return(nil, nil, nil)
	mdi.On("UpdateOffset", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	delivered := make(chan struct{})
	deliver := mei.On("DeliveryRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deliver.RunFn = func(a mock.Arguments) {
		close(delivered)
	}

	bdDone := make(chan struct{})
	ev1 := fftypes.NewUUID()
	ed.eventPoller.pollingOffset = 100050 // ahead of nack
	go func() {
		repoll, err := ed.bufferedDelivery([]fftypes.LocallySequenced{&fftypes.Event{ID: ev1, Sequence: 100001}})
		assert.NoError(t, err)
		assert.True(t, repoll)
		close(bdDone)
	}()

	<-delivered
	ed.deliveryResponse(&fftypes.EventDeliveryResponse{
		ID:       ev1,
		Rejected: true,
	})

	<-bdDone
	assert.Equal(t, int64(100001), ed.eventPoller.pollingOffset)
}

func TestBufferedDeliveryAckFail(t *testing.T) {

	sub := &subscription{
		definition: &fftypes.Subscription{},
	}
	ed, cancel := newTestEventDispatcher(sub)
	defer cancel()
	go ed.deliverEvents()
	ed.readAhead = 50

	mdi := ed.database.(*databasemocks.Plugin)
	mei := ed.transport.(*eventsmocks.PluginAll)
	mdi.On("GetMessages", mock.Anything, mock.Anything).Return(nil, nil, nil)
	mdi.On("GetDataRefs", mock.Anything, mock.Anything).Return(nil, nil, nil)
	mdi.On("UpdateOffset", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	delivered := make(chan bool)
	deliver := mei.On("DeliveryRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deliver.RunFn = func(a mock.Arguments) {
		delivered <- true
	}

	bdDone := make(chan struct{})
	ev1 := fftypes.NewUUID()
	ev2 := fftypes.NewUUID()
	ed.eventPoller.pollingOffset = 100000
	go func() {
		repoll, err := ed.bufferedDelivery([]fftypes.LocallySequenced{
			&fftypes.Event{ID: ev1, Sequence: 100001},
			&fftypes.Event{ID: ev2, Sequence: 100002},
		})
		assert.EqualError(t, err, "pop")
		assert.False(t, repoll)
		close(bdDone)
	}()

	<-delivered
	<-delivered
	ed.deliveryResponse(&fftypes.EventDeliveryResponse{
		ID: ev1,
	})

	<-bdDone
	assert.Equal(t, int64(100001), ed.eventPoller.pollingOffset)

}

func TestBufferedDeliveryFailNack(t *testing.T) {
	log.SetLevel("trace")

	sub := &subscription{
		definition: &fftypes.Subscription{},
	}
	ed, cancel := newTestEventDispatcher(sub)
	defer cancel()
	go ed.deliverEvents()
	ed.readAhead = 50

	mdi := ed.database.(*databasemocks.Plugin)
	mei := ed.transport.(*eventsmocks.PluginAll)
	mdi.On("GetMessages", mock.Anything, mock.Anything).Return(nil, nil, nil)
	mdi.On("GetDataRefs", mock.Anything, mock.Anything).Return(nil, nil, nil)
	mdi.On("UpdateOffset", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	failNacked := make(chan bool)
	deliver := mei.On("DeliveryRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	deliver.RunFn = func(a mock.Arguments) {
		failNacked <- true
	}

	bdDone := make(chan struct{})
	ev1 := fftypes.NewUUID()
	ev2 := fftypes.NewUUID()
	ed.eventPoller.pollingOffset = 100000
	go func() {
		repoll, err := ed.bufferedDelivery([]fftypes.LocallySequenced{
			&fftypes.Event{ID: ev1, Sequence: 100001},
			&fftypes.Event{ID: ev2, Sequence: 100002},
		})
		assert.NoError(t, err)
		assert.True(t, repoll)
		close(bdDone)
	}()

	<-failNacked
	<-failNacked

	<-bdDone
	assert.Equal(t, int64(100000), ed.eventPoller.pollingOffset)

}

func TestBufferedFinalAckFail(t *testing.T) {

	sub := &subscription{
		definition:   &fftypes.Subscription{},
		topicsFilter: regexp.MustCompile("never matches"),
	}
	ed, cancel := newTestEventDispatcher(sub)
	defer cancel()
	go ed.deliverEvents()
	ed.readAhead = 50

	mdi := ed.database.(*databasemocks.Plugin)
	mdi.On("GetMessages", mock.Anything, mock.Anything).Return(nil, nil, nil)
	mdi.On("GetDataRefs", mock.Anything, mock.Anything).Return(nil, nil, nil)
	mdi.On("UpdateOffset", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	ev1 := fftypes.NewUUID()
	ev2 := fftypes.NewUUID()
	ed.eventPoller.pollingOffset = 100000
	repoll, err := ed.bufferedDelivery([]fftypes.LocallySequenced{
		&fftypes.Event{ID: ev1, Sequence: 100001},
		&fftypes.Event{ID: ev2, Sequence: 100002},
	})
	assert.EqualError(t, err, "pop")
	assert.False(t, repoll)

	assert.Equal(t, int64(100002), ed.eventPoller.pollingOffset)

}

func TestAckNotInFlightNoop(t *testing.T) {

	sub := &subscription{
		definition: &fftypes.Subscription{},
	}
	ed, cancel := newTestEventDispatcher(sub)
	defer cancel()

	ed.deliveryResponse(&fftypes.EventDeliveryResponse{ID: fftypes.NewUUID()})
}

func TestEventDeliveryClosed(t *testing.T) {

	sub := &subscription{
		definition: &fftypes.Subscription{},
	}
	ed, cancel := newTestEventDispatcher(sub)
	close(ed.eventDelivery)

	ed.deliverEvents()
	cancel()
}

func TestAckClosed(t *testing.T) {

	sub := &subscription{
		definition: &fftypes.Subscription{},
	}
	ed, cancel := newTestEventDispatcher(sub)
	cancel()

	id1 := fftypes.NewUUID()
	ed.inflight[*id1] = &fftypes.Event{ID: id1}
	ed.deliveryResponse(&fftypes.EventDeliveryResponse{ID: id1})
}

func TestGetEvents(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()

	sub := &subscription{
		definition: &fftypes.Subscription{},
	}

	ed, cancel := newTestEventDispatcher(sub)
	cancel()

	mdi := ed.database.(*databasemocks.Plugin)

	mdi.On("GetEvents", ag.ctx, mock.Anything).Return([]*fftypes.Event{
		{Sequence: 12345},
	}, nil, nil)

	lc, err := ed.getEvents(ag.ctx, database.EventQueryFactory.NewFilter(ag.ctx).Gte("sequence", 12345))
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), lc[0].LocalSequence())
}

func TestDeliverEventsWithDataFail(t *testing.T) {
	yes := true
	sub := &subscription{
		definition: &fftypes.Subscription{
			Options: fftypes.SubscriptionOptions{
				SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
					WithData: &yes,
				},
			},
		},
	}

	ed, cancel := newTestEventDispatcher(sub)
	defer cancel()

	mdm := ed.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ed.ctx, mock.Anything, true).Return(nil, false, fmt.Errorf("pop"))

	id1 := fftypes.NewUUID()
	ed.eventDelivery <- &fftypes.EventDelivery{
		Event: fftypes.Event{
			ID: id1,
		},
		Message: &fftypes.Message{
			Header: fftypes.MessageHeader{
				ID: fftypes.NewUUID(),
			},
			Data: fftypes.DataRefs{
				{ID: fftypes.NewUUID()},
			},
		},
	}

	ed.inflight[*id1] = &fftypes.Event{ID: id1}
	go ed.deliverEvents()

	an := <-ed.acksNacks
	assert.True(t, an.isNack)

}

func TestEventDispatcherWithReply(t *testing.T) {
	log.SetLevel("debug")
	var two = uint16(5)
	sub := &subscription{
		dispatcherElection: make(chan bool, 1),
		definition: &fftypes.Subscription{
			SubscriptionRef: fftypes.SubscriptionRef{ID: fftypes.NewUUID(), Namespace: "ns1", Name: "sub1"},
			Options: fftypes.SubscriptionOptions{
				SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
					ReadAhead: &two,
				},
			},
		},
		eventMatcher: regexp.MustCompile(fmt.Sprintf("^%s|%s$", fftypes.EventTypeMessageConfirmed, fftypes.EventTypeMessageConfirmed)),
	}

	ed, cancel := newTestEventDispatcher(sub)
	cancel()
	ed.acksNacks = make(chan ackNack, 2)
	msh := ed.syshandlers.(*syshandlersmocks.SystemHandlers)
	msh.On("SendReply", ed.ctx, mock.Anything, mock.Anything).Return(&fftypes.Message{}, nil)

	event1 := fftypes.NewUUID()
	event2 := fftypes.NewUUID()
	ed.inflight[*event1] = &fftypes.Event{
		ID:        event1,
		Namespace: "ns1",
	}
	ed.inflight[*event2] = &fftypes.Event{
		ID:        event2,
		Namespace: "ns1",
	}

	ed.deliveryResponse(&fftypes.EventDeliveryResponse{
		ID: event1,
		Reply: &fftypes.MessageInOut{
			Message: fftypes.Message{
				Header: fftypes.MessageHeader{
					Tag:  "myreplytag1",
					CID:  fftypes.NewUUID(),
					Type: fftypes.MessageTypeBroadcast,
				},
			},
			InlineData: fftypes.InlineData{
				{Value: fftypes.Byteable(`"my reply"`)},
			},
		},
	})
	ed.deliveryResponse(&fftypes.EventDeliveryResponse{
		ID: event2,
		Reply: &fftypes.MessageInOut{
			Message: fftypes.Message{
				Header: fftypes.MessageHeader{
					Tag:   "myreplytag2",
					CID:   fftypes.NewUUID(),
					Type:  fftypes.MessageTypePrivate,
					Group: fftypes.NewRandB32(),
				},
			},
			InlineData: fftypes.InlineData{
				{Value: fftypes.Byteable(`"my reply"`)},
			},
		},
	})

	msh.AssertExpectations(t)
}

func TestDispatchChangeEventBlockedClose(t *testing.T) {
	yes := true
	sub := &subscription{
		definition: &fftypes.Subscription{
			Options: fftypes.SubscriptionOptions{
				SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
					WithData: &yes,
				},
			},
		},
	}

	ed, cancel := newTestEventDispatcher(sub)
	cancel()
	close(ed.eventPoller.closed)

	ed.dispatchChangeEvent(&fftypes.ChangeEvent{})
}
