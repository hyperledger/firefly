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
	"testing"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/mocks/broadcastmocks"
	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/mocks/datamocks"
	"github.com/hyperledger-labs/firefly/mocks/eventsmocks"
	"github.com/hyperledger-labs/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger-labs/firefly/pkg/events"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestSubManager(t *testing.T, mei *eventsmocks.Plugin) (*subscriptionManager, func()) {
	config.Reset()
	config.Set(config.EventTransportsEnabled, []string{})

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}

	ctx, cancel := context.WithCancel(context.Background())
	mei.On("Name").Return("ut")
	mei.On("InitPrefix", mock.Anything).Return()
	mei.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("GetEvents", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil).Maybe()
	mdi.On("GetOffset", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&fftypes.Offset{ID: fftypes.NewUUID(), Current: 0}, nil).Maybe()
	rs := &replySender{
		broadcast: &broadcastmocks.Manager{},
		messaging: &privatemessagingmocks.Manager{},
	}
	sm, err := newSubscriptionManager(ctx, mdi, mdm, newEventNotifier(ctx, "ut"), rs)
	assert.NoError(t, err)
	sm.transports = map[string]events.Plugin{
		"ut": mei,
	}
	return sm, cancel
}

func TestRegisterDurableSubscriptions(t *testing.T) {

	sub1 := fftypes.NewUUID()
	sub2 := fftypes.NewUUID()

	// Set some existing ones to be cleaned out
	testED1, cancel1 := newTestEventDispatcher(&subscription{definition: &fftypes.Subscription{SubscriptionRef: fftypes.SubscriptionRef{ID: sub1}}})
	testED1.start()
	defer cancel1()

	mei := testED1.transport.(*eventsmocks.Plugin)
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()

	mdi := sm.database.(*databasemocks.Plugin)
	mdi.On("GetSubscriptions", mock.Anything, mock.Anything).Return([]*fftypes.Subscription{
		{SubscriptionRef: fftypes.SubscriptionRef{
			ID: sub1,
		}, Transport: "ut"},
		{SubscriptionRef: fftypes.SubscriptionRef{
			ID: sub2,
		}, Transport: "ut"},
	}, nil)
	mei.On("ValidateOptions", mock.Anything).Return(nil)
	err := sm.start()
	assert.NoError(t, err)

	sm.connections["conn1"] = &connection{
		ei:        mei,
		id:        "conn1",
		transport: "ut",
		dispatchers: map[fftypes.UUID]*eventDispatcher{
			*sub1: testED1,
		},
	}
	be := &boundCallbacks{sm: sm, ei: mei}

	be.RegisterConnection("conn1", func(sr fftypes.SubscriptionRef) bool {
		return *sr.ID == *sub2
	})
	be.RegisterConnection("conn2", func(sr fftypes.SubscriptionRef) bool {
		return *sr.ID == *sub1
	})

	assert.Equal(t, 1, len(sm.connections["conn1"].dispatchers))
	assert.Equal(t, *sub2, *sm.connections["conn1"].dispatchers[*sub2].subscription.definition.ID)
	assert.Equal(t, 1, len(sm.connections["conn2"].dispatchers))
	assert.Equal(t, *sub1, *sm.connections["conn2"].dispatchers[*sub1].subscription.definition.ID)

	// Close with active conns
	sm.close()
	assert.Nil(t, sm.connections["conn1"])
	assert.Nil(t, sm.connections["conn2"])
}

func TestRegisterEphemeralSubscriptions(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mdi := sm.database.(*databasemocks.Plugin)

	mdi.On("GetSubscriptions", mock.Anything, mock.Anything).Return([]*fftypes.Subscription{}, nil)
	mei.On("ValidateOptions", mock.Anything).Return(nil)

	err := sm.start()
	assert.NoError(t, err)
	be := &boundCallbacks{sm: sm, ei: mei}

	err = be.EphemeralSubscription("conn1", "ns1", &fftypes.SubscriptionFilter{}, &fftypes.SubscriptionOptions{})
	assert.NoError(t, err)

	assert.Equal(t, 1, len(sm.connections["conn1"].dispatchers))
	for _, d := range sm.connections["conn1"].dispatchers {
		assert.True(t, d.subscription.definition.Ephemeral)
	}

	be.ConnnectionClosed("conn1")
	assert.Nil(t, sm.connections["conn1"])
	// Check we swallow dup closes without errors
	be.ConnnectionClosed("conn1")
	assert.Nil(t, sm.connections["conn1"])
}

func TestRegisterEphemeralSubscriptionsFail(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mdi := sm.database.(*databasemocks.Plugin)

	mdi.On("GetSubscriptions", mock.Anything, mock.Anything).Return([]*fftypes.Subscription{}, nil)
	mei.On("ValidateOptions", mock.Anything).Return(nil)
	err := sm.start()
	assert.NoError(t, err)
	be := &boundCallbacks{sm: sm, ei: mei}

	err = be.EphemeralSubscription("conn1", "ns1", &fftypes.SubscriptionFilter{
		Topics: "[[[[[ !wrong",
	}, &fftypes.SubscriptionOptions{})
	assert.Regexp(t, "FF10171", err)
	assert.Empty(t, sm.connections["conn1"].dispatchers)

}

func TestSubManagerBadPlugin(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	config.Reset()
	config.Set(config.EventTransportsEnabled, []string{"!unknown!"})
	_, err := newSubscriptionManager(context.Background(), mdi, mdm, newEventNotifier(context.Background(), "ut"), nil)
	assert.Regexp(t, "FF10172", err)
}

func TestSubManagerTransportInitError(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	mei.On("Name").Return("ut")
	mei.On("InitPrefix", mock.Anything).Return()
	mei.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	sm, cancel := newTestSubManager(t, mei)
	defer cancel()

	err := sm.initTransports()
	assert.EqualError(t, err, "pop")
}

func TestStartSubRestoreFail(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mdi := sm.database.(*databasemocks.Plugin)

	mdi.On("GetSubscriptions", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	err := sm.start()
	assert.EqualError(t, err, "pop")
}

func TestStartSubRestoreOkSubsFail(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mdi := sm.database.(*databasemocks.Plugin)

	mdi.On("GetSubscriptions", mock.Anything, mock.Anything).Return([]*fftypes.Subscription{
		{SubscriptionRef: fftypes.SubscriptionRef{
			ID: fftypes.NewUUID(),
		},
			Filter: fftypes.SubscriptionFilter{
				Events: "[[[[[[not a regex",
			}},
	}, nil)
	err := sm.start()
	assert.NoError(t, err) // swallowed and startup continues
}

func TestStartSubRestoreOkSubsOK(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mdi := sm.database.(*databasemocks.Plugin)

	mdi.On("GetSubscriptions", mock.Anything, mock.Anything).Return([]*fftypes.Subscription{
		{SubscriptionRef: fftypes.SubscriptionRef{
			ID: fftypes.NewUUID(),
		},
			Filter: fftypes.SubscriptionFilter{
				Events: ".*",
				Topics: ".*",
				Tag:    ".*",
				Group:  ".*",
				Author: ".*",
			}},
	}, nil)
	err := sm.start()
	assert.NoError(t, err) // swallowed and startup continues
}

func TestCreateSubscriptionBadTransport(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	_, err := sm.parseSubscriptionDef(sm.ctx, &fftypes.Subscription{})
	assert.Regexp(t, "FF1017", err)
}

func TestCreateSubscriptionBadTransportOptions(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	sub := &fftypes.Subscription{
		Transport: "ut",
		Options:   fftypes.SubscriptionOptions{},
	}
	sub.Options.TransportOptions()["myoption"] = "badvalue"
	mei.On("ValidateOptions", mock.MatchedBy(func(opts *fftypes.SubscriptionOptions) bool {
		return opts.TransportOptions()["myoption"] == "badvalue"
	})).Return(fmt.Errorf("pop"))
	_, err := sm.parseSubscriptionDef(sm.ctx, sub)
	assert.Regexp(t, "pop", err)
}

func TestCreateSubscriptionBadEventilter(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mei.On("ValidateOptions", mock.Anything).Return(nil)
	_, err := sm.parseSubscriptionDef(sm.ctx, &fftypes.Subscription{
		Filter: fftypes.SubscriptionFilter{
			Events: "[[[[! badness",
		},
		Transport: "ut",
	})
	assert.Regexp(t, "FF10171.*events", err)
}

func TestCreateSubscriptionBadTopicFilter(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mei.On("ValidateOptions", mock.Anything).Return(nil)
	_, err := sm.parseSubscriptionDef(sm.ctx, &fftypes.Subscription{
		Filter: fftypes.SubscriptionFilter{
			Topics: "[[[[! badness",
		},
		Transport: "ut",
	})
	assert.Regexp(t, "FF10171.*topic", err)
}

func TestCreateSubscriptionBadContextFilter(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mei.On("ValidateOptions", mock.Anything).Return(nil)
	_, err := sm.parseSubscriptionDef(sm.ctx, &fftypes.Subscription{
		Filter: fftypes.SubscriptionFilter{
			Tag: "[[[[! badness",
		},
		Transport: "ut",
	})
	assert.Regexp(t, "FF10171.*tag", err)
}

func TestCreateSubscriptionBadGroupFilter(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mei.On("ValidateOptions", mock.Anything).Return(nil)
	_, err := sm.parseSubscriptionDef(sm.ctx, &fftypes.Subscription{
		Filter: fftypes.SubscriptionFilter{
			Group: "[[[[! badness",
		},
		Transport: "ut",
	})
	assert.Regexp(t, "FF10171.*group", err)
}

func TestCreateSubscriptionBadAuthorFilter(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mei.On("ValidateOptions", mock.Anything).Return(nil)
	_, err := sm.parseSubscriptionDef(sm.ctx, &fftypes.Subscription{
		Filter: fftypes.SubscriptionFilter{
			Author: "[[[[! badness",
		},
		Transport: "ut",
	})
	assert.Regexp(t, "FF10171.*author", err)
}

func TestDispatchDeliveryResponseOK(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mdi := sm.database.(*databasemocks.Plugin)
	mdi.On("GetSubscriptions", mock.Anything, mock.Anything).Return([]*fftypes.Subscription{}, nil)
	mei.On("ValidateOptions", mock.Anything).Return(nil)
	err := sm.start()
	assert.NoError(t, err)
	be := &boundCallbacks{sm: sm, ei: mei}

	err = be.EphemeralSubscription("conn1", "ns1", &fftypes.SubscriptionFilter{}, &fftypes.SubscriptionOptions{})
	assert.NoError(t, err)

	assert.Equal(t, 1, len(sm.connections["conn1"].dispatchers))
	var subID *fftypes.UUID
	for _, d := range sm.connections["conn1"].dispatchers {
		assert.True(t, d.subscription.definition.Ephemeral)
		subID = d.subscription.definition.ID
	}

	be.DeliveryResponse("conn1", &fftypes.EventDeliveryResponse{
		ID: fftypes.NewUUID(), // Won't be in-flight, but that's fine
		Subscription: fftypes.SubscriptionRef{
			ID: subID,
		},
	})
	mdi.AssertExpectations(t)
}

func TestDispatchDeliveryResponseInvalidSubscription(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mdi := sm.database.(*databasemocks.Plugin)
	mdi.On("GetSubscriptions", mock.Anything, mock.Anything).Return([]*fftypes.Subscription{}, nil)
	err := sm.start()
	assert.NoError(t, err)
	be := &boundCallbacks{sm: sm, ei: mei}

	be.DeliveryResponse("conn1", &fftypes.EventDeliveryResponse{
		ID: fftypes.NewUUID(),
		Subscription: fftypes.SubscriptionRef{
			ID: fftypes.NewUUID(),
		},
	})
	mdi.AssertExpectations(t)
}

func TestConnIDSafetyChecking(t *testing.T) {
	mei1 := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei1)
	defer cancel()
	mei2 := &eventsmocks.Plugin{}
	mei2.On("Name").Return("ut2")
	be2 := &boundCallbacks{sm: sm, ei: mei2}

	sm.connections["conn1"] = &connection{
		ei:          mei1,
		id:          "conn1",
		transport:   "ut",
		dispatchers: map[fftypes.UUID]*eventDispatcher{},
	}

	err := be2.RegisterConnection("conn1", func(sr fftypes.SubscriptionRef) bool { return true })
	assert.Regexp(t, "FF10190", err)

	err = be2.EphemeralSubscription("conn1", "ns1", &fftypes.SubscriptionFilter{}, &fftypes.SubscriptionOptions{})
	assert.Regexp(t, "FF10190", err)

	be2.DeliveryResponse("conn1", &fftypes.EventDeliveryResponse{})

	be2.ConnnectionClosed("conn1")

	assert.NotNil(t, sm.connections["conn1"])

}

func TestNewDurableSubscriptionBadSub(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mdi := sm.database.(*databasemocks.Plugin)

	subID := fftypes.NewUUID()
	mdi.On("GetSubscriptionByID", mock.Anything, subID).Return(&fftypes.Subscription{
		Filter: fftypes.SubscriptionFilter{
			Events: "![[[[badness",
		},
	}, nil)
	sm.newDurableSubscription(subID)

	assert.Empty(t, sm.durableSubs)
}

func TestNewDurableSubscriptionUnknownTransport(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mdi := sm.database.(*databasemocks.Plugin)

	sm.connections["conn1"] = &connection{
		ei:        mei,
		id:        "conn1",
		transport: "ut",
		matcher: func(sr fftypes.SubscriptionRef) bool {
			return sr.Namespace == "ns1" && sr.Name == "sub1"
		},
		dispatchers: map[fftypes.UUID]*eventDispatcher{},
	}

	subID := fftypes.NewUUID()
	mdi.On("GetSubscriptionByID", mock.Anything, subID).Return(&fftypes.Subscription{
		SubscriptionRef: fftypes.SubscriptionRef{
			ID:        subID,
			Namespace: "ns1",
			Name:      "sub1",
		},
		Transport: "unknown",
	}, nil)
	sm.newDurableSubscription(subID)

	assert.Empty(t, sm.connections["conn1"].dispatchers)
	assert.Empty(t, sm.durableSubs)
}

func TestNewDurableSubscriptionOK(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mdi := sm.database.(*databasemocks.Plugin)
	mei.On("ValidateOptions", mock.Anything).Return(nil)

	sm.connections["conn1"] = &connection{
		ei:        mei,
		id:        "conn1",
		transport: "ut",
		matcher: func(sr fftypes.SubscriptionRef) bool {
			return sr.Namespace == "ns1" && sr.Name == "sub1"
		},
		dispatchers: map[fftypes.UUID]*eventDispatcher{},
	}

	subID := fftypes.NewUUID()
	mdi.On("GetSubscriptionByID", mock.Anything, subID).Return(&fftypes.Subscription{
		SubscriptionRef: fftypes.SubscriptionRef{
			ID:        subID,
			Namespace: "ns1",
			Name:      "sub1",
		},
		Transport: "ut",
	}, nil)
	sm.newDurableSubscription(subID)

	assert.NotEmpty(t, sm.connections["conn1"].dispatchers)
	assert.NotEmpty(t, sm.durableSubs)
}

func TestMatchedSubscriptionWithLockUnknownTransport(t *testing.T) {
	mei := &eventsmocks.Plugin{}
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()

	conn := &connection{}
	sm.matchSubToConnLocked(conn, &subscription{definition: &fftypes.Subscription{Transport: "Wrong!"}})
	assert.Nil(t, conn.dispatchers)
}

func TestDeletewDurableSubscriptionOk(t *testing.T) {
	subID := fftypes.NewUUID()
	subDef := &fftypes.Subscription{
		SubscriptionRef: fftypes.SubscriptionRef{
			ID:        subID,
			Namespace: "ns1",
			Name:      "sub1",
		},
		Transport: "websockets",
	}
	sub := &subscription{
		definition: subDef,
	}
	testED1, _ := newTestEventDispatcher(sub)

	mei := testED1.transport.(*eventsmocks.Plugin)
	sm, cancel := newTestSubManager(t, mei)
	defer cancel()
	mdi := sm.database.(*databasemocks.Plugin)

	sm.durableSubs[*subID] = sub
	ed, _ := newTestEventDispatcher(sub)
	ed.start()
	sm.connections["conn1"] = &connection{
		ei:        mei,
		id:        "conn1",
		transport: "ut",
		matcher: func(sr fftypes.SubscriptionRef) bool {
			return sr.Namespace == "ns1" && sr.Name == "sub1"
		},
		dispatchers: map[fftypes.UUID]*eventDispatcher{
			*subID: ed,
		},
	}

	mdi.On("GetSubscriptionByID", mock.Anything, subID).Return(subDef, nil)
	sm.deletedDurableSubscription(subID)

	assert.Empty(t, sm.connections["conn1"].dispatchers)
	assert.Empty(t, sm.durableSubs)
	<-ed.closed
}
