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
	"fmt"
	"regexp"
	"testing"

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/mocks/eventsmocks"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/events"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestEventDispatcher(mdi database.Plugin, mei events.Plugin, sub *subscription) (*eventDispatcher, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	config.Reset()
	return newEventDispatcher(ctx, mei, mdi, fftypes.NewUUID().String(), sub), cancel
}

func TestEventDispatcherStartStop(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mei := &eventsmocks.Plugin{}
	ten := uint64(10)
	ed, cancel := newTestEventDispatcher(mdi, mei, &subscription{
		dispatcherElection: make(chan bool, 1),
		definition: &fftypes.Subscription{
			SubscriptionRef: fftypes.SubscriptionRef{Namespace: "ns1", Name: "sub1"},
			Ephemeral:       true,
			Options: fftypes.SubscriptionOptions{
				ReadAhead: &ten,
			},
		},
	})
	defer cancel()
	mdi.On("GetEvents", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	assert.Equal(t, int(10), ed.readAhead)
	ed.start()
	ed.eventPoller.newEvents <- fftypes.NewUUID()
	ed.close()
}

func TestEventDispatcherLeaderElection(t *testing.T) {
	log.SetLevel("debug")

	sub := &subscription{
		dispatcherElection: make(chan bool, 1),
		definition: &fftypes.Subscription{
			SubscriptionRef: fftypes.SubscriptionRef{Namespace: "ns1", Name: "sub1"},
		},
	}

	gev1Wait := make(chan bool)
	gev1Done := make(chan struct{})
	mei := &eventsmocks.Plugin{}
	mdi1 := &databasemocks.Plugin{}
	gev1 := mdi1.On("GetEvents", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	mdi1.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "ns1", "sub1").Return(&fftypes.Offset{
		Type:      fftypes.OffsetTypeSubscription,
		Namespace: "ns1",
		Name:      "sub1",
		Current:   12345,
	}, nil)
	gev1.RunFn = func(a mock.Arguments) {
		gev1Wait <- true
		<-gev1Done
	}
	ed1, cancel1 := newTestEventDispatcher(mdi1, mei, sub)

	mdi2 := &databasemocks.Plugin{} // No mocks, so will bail if called
	ed2, cancel2 := newTestEventDispatcher(mdi2, mei, sub /* same sub */)

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
	var five = uint64(5)
	sub := &subscription{
		dispatcherElection: make(chan bool, 1),
		definition: &fftypes.Subscription{
			SubscriptionRef: fftypes.SubscriptionRef{ID: fftypes.NewUUID(), Namespace: "ns1", Name: "sub1"},
			Options: fftypes.SubscriptionOptions{
				ReadAhead: &five,
			},
		},
		eventMatcher: regexp.MustCompile(fmt.Sprintf("^%s|%s$", fftypes.EventTypeMessageConfirmed, fftypes.EventTypeDataArrivedBroadcast)),
	}

	mdi := &databasemocks.Plugin{}

	eventDeliveries := make(chan fftypes.EventDelivery)
	mei := &eventsmocks.Plugin{}
	deliveryRequestMock := mei.On("DeliveryRequest", mock.Anything, mock.Anything).Return(nil)
	deliveryRequestMock.RunFn = func(a mock.Arguments) {
		eventDeliveries <- a.Get(1).(fftypes.EventDelivery)
	}

	ed, cancel := newTestEventDispatcher(mdi, mei, sub)
	defer cancel()

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
	uof := mdi.On("UpdateOffset", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	uof.RunFn = func(a mock.Arguments) {
		f, _ := a.Get(2).(database.Update).Finalize()
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
		{Header: fftypes.MessageHeader{ID: ref4}},
	}, nil)
	mdi.On("GetDataRefs", mock.Anything, mock.MatchedBy(func(filter database.Filter) bool {
		fi, err := filter.Finalize()
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf(`( id IN ['%s','%s','%s','%s'] ) && ( namespace == 'ns1' )`, ref1, ref2, ref3, ref4), fi.String())
		return true
	})).Return(fftypes.DataRefs{
		{ID: ref3},
	}, nil)

	// Deliver a batch of messages
	batch1Done := make(chan struct{})
	go func() {
		repoll, err := ed.bufferedDelivery([]*fftypes.Event{
			{ID: ev1, Sequence: 10000001, Reference: ref1, Type: fftypes.EventTypeMessageConfirmed}, // match
			{ID: ev2, Sequence: 10000002, Reference: ref2, Type: fftypes.EventTypeMessagesUnblocked},
			{ID: ev3, Sequence: 10000003, Reference: ref3, Type: fftypes.EventTypeDataArrivedBroadcast}, // match
			{ID: ev4, Sequence: 10000004, Reference: ref4, Type: fftypes.EventTypeMessageConfirmed},     // match
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
	assert.Equal(t, *ref3, *event3.Data.ID)
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
	// Note there's no need update for 10000003, as we were all done at that point
	assert.Equal(t, int64(10000001), <-offsetUpdates)
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

	mdi := &databasemocks.Plugin{}

	eventDeliveries := make(chan fftypes.EventDelivery)
	mei := &eventsmocks.Plugin{}
	deliveryRequestMock := mei.On("DeliveryRequest", mock.Anything, mock.Anything).Return(nil)
	deliveryRequestMock.RunFn = func(a mock.Arguments) {
		eventDeliveries <- a.Get(1).(fftypes.EventDelivery)
	}

	ed, cancel := newTestEventDispatcher(mdi, mei, sub)
	defer cancel()

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
	}, nil)
	mdi.On("GetDataRefs", mock.Anything, mock.Anything).Return(fftypes.DataRefs{}, nil)

	// Deliver a batch of messages
	batch1Done := make(chan struct{})
	go func() {
		repoll, err := ed.bufferedDelivery([]*fftypes.Event{
			{ID: ev1, Sequence: 10000001, Reference: ref1, Type: fftypes.EventTypeMessageConfirmed}, // match
			{ID: ev2, Sequence: 10000002, Reference: ref2, Type: fftypes.EventTypeMessageConfirmed}, // match
			{ID: ev3, Sequence: 10000003, Reference: ref3, Type: fftypes.EventTypeMessageConfirmed}, // match
			{ID: ev4, Sequence: 10000004, Reference: ref4, Type: fftypes.EventTypeMessageConfirmed}, // match
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
