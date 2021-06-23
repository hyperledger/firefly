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
	"github.com/hyperledger-labs/firefly/mocks/identitymocks"
	"github.com/hyperledger-labs/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger-labs/firefly/mocks/publicstoragemocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestEventManager(t *testing.T) (*eventManager, func()) {
	config.Reset()
	ctx, cancel := context.WithCancel(context.Background())
	mdi := &databasemocks.Plugin{}
	mii := &identitymocks.Plugin{}
	mpi := &publicstoragemocks.Plugin{}
	met := &eventsmocks.Plugin{}
	mbm := &broadcastmocks.Manager{}
	mpm := &privatemessagingmocks.Manager{}
	mdm := &datamocks.Manager{}
	met.On("Name").Return("ut").Maybe()
	em, err := NewEventManager(ctx, mpi, mdi, mii, mbm, mpm, mdm)
	assert.NoError(t, err)
	return em.(*eventManager), cancel
}

func TestStartStop(t *testing.T) {
	em, cancel := newTestEventManager(t)
	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeAggregator, fftypes.SystemNamespace, aggregatorOffsetName).Return(&fftypes.Offset{
		Type:      fftypes.OffsetTypeAggregator,
		Namespace: fftypes.SystemNamespace,
		Name:      aggregatorOffsetName,
		Current:   12345,
	}, nil)
	mdi.On("GetPins", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Pin{}, nil)
	mdi.On("GetSubscriptions", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Subscription{}, nil)
	assert.NoError(t, em.Start())
	em.NewEvents() <- 12345
	em.NewPins() <- 12345
	cancel()
	em.WaitStop()
}

func TestStartStopBadDependencies(t *testing.T) {
	_, err := NewEventManager(context.Background(), nil, nil, nil, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)

}

func TestStartStopBadTransports(t *testing.T) {
	config.Set(config.EventTransportsEnabled, []string{"wrongun"})
	defer config.Reset()
	mdi := &databasemocks.Plugin{}
	mii := &identitymocks.Plugin{}
	mpi := &publicstoragemocks.Plugin{}
	mbm := &broadcastmocks.Manager{}
	mpm := &privatemessagingmocks.Manager{}
	mdm := &datamocks.Manager{}
	_, err := NewEventManager(context.Background(), mpi, mdi, mii, mbm, mpm, mdm)
	assert.Regexp(t, "FF10172", err)

}

func TestEmitSubscriptionEventsNoops(t *testing.T) {
	em, cancel := newTestEventManager(t)
	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeAggregator, fftypes.SystemNamespace, aggregatorOffsetName).Return(&fftypes.Offset{
		Type:      fftypes.OffsetTypeAggregator,
		Namespace: fftypes.SystemNamespace,
		Name:      aggregatorOffsetName,
		Current:   12345,
	}, nil)
	mdi.On("GetPins", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Pin{}, nil)
	mdi.On("GetSubscriptions", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Subscription{}, nil)

	getSubCallReady := make(chan bool, 1)
	getSubCalled := make(chan bool)
	getSub := mdi.On("GetSubscriptionByID", mock.Anything, mock.Anything).Return(nil, nil)
	getSub.RunFn = func(a mock.Arguments) {
		<-getSubCallReady
		getSubCalled <- true
	}

	assert.NoError(t, em.Start())
	defer cancel()

	// Wait until the gets occur for these events, which will return nil
	getSubCallReady <- true
	em.NewSubscriptions() <- fftypes.NewUUID()
	<-getSubCalled

	em.DeletedSubscriptions() <- fftypes.NewUUID()
	close(getSubCallReady)
}

func TestCreateDurableSubscriptionBadSub(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	err := em.CreateDurableSubscription(em.ctx, &fftypes.Subscription{})
	assert.Regexp(t, "FF10189", err)
}

func TestCreateDurableSubscriptionDupName(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	sub := &fftypes.Subscription{
		SubscriptionRef: fftypes.SubscriptionRef{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "sub1",
		},
	}
	mdi.On("GetSubscriptionByName", mock.Anything, "ns1", "sub1").Return(sub, nil)
	err := em.CreateDurableSubscription(em.ctx, sub)
	assert.Regexp(t, "FF10193", err)
}

func TestCreateDurableSubscriptionDefaultSubCannotParse(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	sub := &fftypes.Subscription{
		SubscriptionRef: fftypes.SubscriptionRef{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "sub1",
		},
		Filter: fftypes.SubscriptionFilter{
			Events: "![[[[[",
		},
	}
	mdi.On("GetSubscriptionByName", mock.Anything, "ns1", "sub1").Return(nil, nil)
	err := em.CreateDurableSubscription(em.ctx, sub)
	assert.Regexp(t, "FF10171", err)
}

func TestCreateDurableSubscriptionBadFirstEvent(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	wrongFirstEvent := fftypes.SubOptsFirstEvent("lobster")
	sub := &fftypes.Subscription{
		SubscriptionRef: fftypes.SubscriptionRef{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "sub1",
		},
		Options: fftypes.SubscriptionOptions{
			SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
				FirstEvent: &wrongFirstEvent,
			},
		},
	}
	mdi.On("GetSubscriptionByName", mock.Anything, "ns1", "sub1").Return(nil, nil)
	err := em.CreateDurableSubscription(em.ctx, sub)
	assert.Regexp(t, "FF10191", err)
}

func TestCreateDurableSubscriptionNegativeFirstEvent(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	wrongFirstEvent := fftypes.SubOptsFirstEvent("-12345")
	mdi := em.database.(*databasemocks.Plugin)
	sub := &fftypes.Subscription{
		SubscriptionRef: fftypes.SubscriptionRef{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "sub1",
		},
		Options: fftypes.SubscriptionOptions{
			SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
				FirstEvent: &wrongFirstEvent,
			},
		},
	}
	mdi.On("GetSubscriptionByName", mock.Anything, "ns1", "sub1").Return(nil, nil)
	err := em.CreateDurableSubscription(em.ctx, sub)
	assert.Regexp(t, "FF10192", err)
}

func TestCreateDurableSubscriptionGetHighestSequenceFailure(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	sub := &fftypes.Subscription{
		SubscriptionRef: fftypes.SubscriptionRef{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "sub1",
		},
	}
	mdi.On("GetSubscriptionByName", mock.Anything, "ns1", "sub1").Return(nil, nil)
	mdi.On("GetEvents", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	err := em.CreateDurableSubscription(em.ctx, sub)
	assert.EqualError(t, err, "pop")
}

func TestCreateDurableSubscriptionOk(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	sub := &fftypes.Subscription{
		SubscriptionRef: fftypes.SubscriptionRef{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "sub1",
		},
	}
	mdi.On("GetSubscriptionByName", mock.Anything, "ns1", "sub1").Return(nil, nil)
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{
		{Sequence: 12345},
	}, nil)
	mdi.On("UpsertSubscription", mock.Anything, mock.Anything, false).Return(nil)
	err := em.CreateDurableSubscription(em.ctx, sub)
	assert.NoError(t, err)
	// Check genreated fields
	assert.NotNil(t, sub.ID)
	assert.Equal(t, "websockets", sub.Transport)
	assert.Equal(t, "12345", string(*sub.Options.FirstEvent))
}

func TestCreateDeleteDurableSubscriptionOk(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	subId := fftypes.NewUUID()
	sub := &fftypes.Subscription{SubscriptionRef: fftypes.SubscriptionRef{ID: subId, Namespace: "ns1"}}
	mdi.On("GetSubscriptionByID", mock.Anything, subId).Return(sub, nil)
	mdi.On("DeleteSubscriptionByID", mock.Anything, subId).Return(nil)
	err := em.DeleteDurableSubscription(em.ctx, sub)
	assert.NoError(t, err)
}
