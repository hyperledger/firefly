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

package orchestrator

import (
	"testing"

	"github.com/hyperledger/firefly/mocks/admineventsmocks"
	"github.com/hyperledger/firefly/mocks/batchmocks"
	"github.com/hyperledger/firefly/mocks/eventmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/mock"
)

func TestMessageCreated(t *testing.T) {
	mb := &batchmocks.Manager{}
	mae := &admineventsmocks.Manager{}
	o := &orchestrator{
		batch:       mb,
		adminEvents: mae,
	}
	mb.On("NewMessages").Return((chan<- int64)(make(chan int64, 1)))
	mae.On("Dispatch", mock.Anything).Return()
	o.OrderedUUIDCollectionNSEvent(database.CollectionMessages, fftypes.ChangeEventTypeCreated, "ns1", fftypes.NewUUID(), 12345)
	mb.AssertExpectations(t)
	mae.AssertExpectations(t)
}

func TestPinCreated(t *testing.T) {
	mem := &eventmocks.EventManager{}
	mae := &admineventsmocks.Manager{}
	o := &orchestrator{
		adminEvents: mae,
		events:      mem,
	}
	mae.On("Dispatch", mock.Anything).Return()
	mem.On("NewPins").Return((chan<- int64)(make(chan int64, 1)))
	o.OrderedCollectionEvent(database.CollectionPins, fftypes.ChangeEventTypeCreated, 12345)
	mem.AssertExpectations(t)
	mae.AssertExpectations(t)
}

func TestEventCreated(t *testing.T) {
	mem := &eventmocks.EventManager{}
	mae := &admineventsmocks.Manager{}
	o := &orchestrator{
		adminEvents: mae,
		events:      mem,
	}
	mae.On("Dispatch", mock.Anything).Return()
	mem.On("NewEvents").Return((chan<- int64)(make(chan int64, 1)))
	o.OrderedUUIDCollectionNSEvent(database.CollectionEvents, fftypes.ChangeEventTypeCreated, "ns1", fftypes.NewUUID(), 12345)
	mem.AssertExpectations(t)
	mae.AssertExpectations(t)
}

func TestSubscriptionCreated(t *testing.T) {
	mem := &eventmocks.EventManager{}
	mae := &admineventsmocks.Manager{}
	o := &orchestrator{
		adminEvents: mae,
		events:      mem,
	}
	mae.On("Dispatch", mock.Anything).Return()
	mem.On("NewSubscriptions").Return((chan<- *fftypes.UUID)(make(chan *fftypes.UUID, 1)))
	o.UUIDCollectionNSEvent(database.CollectionSubscriptions, fftypes.ChangeEventTypeCreated, "ns1", fftypes.NewUUID())
	mem.AssertExpectations(t)
	mae.AssertExpectations(t)
}

func TestSubscriptionUpdated(t *testing.T) {
	mem := &eventmocks.EventManager{}
	mae := &admineventsmocks.Manager{}
	o := &orchestrator{
		adminEvents: mae,
		events:      mem,
	}
	mae.On("Dispatch", mock.Anything).Return()
	mem.On("SubscriptionUpdates").Return((chan<- *fftypes.UUID)(make(chan *fftypes.UUID, 1)))
	o.UUIDCollectionNSEvent(database.CollectionSubscriptions, fftypes.ChangeEventTypeUpdated, "ns1", fftypes.NewUUID())
	mem.AssertExpectations(t)
	mae.AssertExpectations(t)
}

func TestSubscriptionDeleted(t *testing.T) {
	mem := &eventmocks.EventManager{}
	mae := &admineventsmocks.Manager{}
	o := &orchestrator{
		adminEvents: mae,
		events:      mem,
	}
	mae.On("Dispatch", mock.Anything).Return()
	mem.On("DeletedSubscriptions").Return((chan<- *fftypes.UUID)(make(chan *fftypes.UUID, 1)))
	o.UUIDCollectionNSEvent(database.CollectionSubscriptions, fftypes.ChangeEventTypeDeleted, "ns1", fftypes.NewUUID())
	mem.AssertExpectations(t)
	mae.AssertExpectations(t)
}

func TestUUIDCollectionEventFull(t *testing.T) {
	mae := &admineventsmocks.Manager{}
	o := &orchestrator{
		adminEvents: mae,
	}
	mae.On("Dispatch", mock.Anything).Return()
	o.UUIDCollectionEvent(database.CollectionNamespaces, fftypes.ChangeEventTypeDeleted, fftypes.NewUUID())
	mae.AssertExpectations(t)
}

func TestHashCollectionNSEventOk(t *testing.T) {
	mae := &admineventsmocks.Manager{}
	o := &orchestrator{
		adminEvents: mae,
	}
	mae.On("Dispatch", mock.Anything).Return()
	o.HashCollectionNSEvent(database.CollectionGroups, fftypes.ChangeEventTypeDeleted, "ns1", fftypes.NewRandB32())
	mae.AssertExpectations(t)
}
