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
	"context"
	"testing"

	"github.com/hyperledger-labs/firefly/mocks/batchmocks"
	"github.com/hyperledger-labs/firefly/mocks/eventmocks"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func TestMessageCreated(t *testing.T) {
	mb := &batchmocks.Manager{}
	mem := &eventmocks.EventManager{}
	o := &orchestrator{
		batch:  mb,
		events: mem,
	}
	mb.On("NewMessages").Return((chan<- int64)(make(chan int64, 1)))
	mem.On("ChangeEvents").Return((chan<- *fftypes.ChangeEvent)(make(chan *fftypes.ChangeEvent, 1)))
	o.OrderedUUIDCollectionNSEvent(database.CollectionMessages, fftypes.ChangeEventTypeCreated, "ns1", fftypes.NewUUID(), 12345)
	mb.AssertExpectations(t)
}

func TestPinCreated(t *testing.T) {
	mem := &eventmocks.EventManager{}
	o := &orchestrator{
		events: mem,
	}
	mem.On("NewPins").Return((chan<- int64)(make(chan int64, 1)))
	mem.On("ChangeEvents").Return((chan<- *fftypes.ChangeEvent)(make(chan *fftypes.ChangeEvent, 1)))
	o.OrderedCollectionEvent(database.CollectionPins, fftypes.ChangeEventTypeCreated, 12345)
	mem.AssertExpectations(t)
}

func TestEventCreated(t *testing.T) {
	mem := &eventmocks.EventManager{}
	o := &orchestrator{
		events: mem,
	}
	mem.On("NewEvents").Return((chan<- int64)(make(chan int64, 1)))
	mem.On("ChangeEvents").Return((chan<- *fftypes.ChangeEvent)(make(chan *fftypes.ChangeEvent, 1)))
	o.OrderedUUIDCollectionNSEvent(database.CollectionEvents, fftypes.ChangeEventTypeCreated, "ns1", fftypes.NewUUID(), 12345)
	mem.AssertExpectations(t)
}

func TestSubscriptionCreated(t *testing.T) {
	mem := &eventmocks.EventManager{}
	o := &orchestrator{
		events: mem,
	}
	mem.On("NewSubscriptions").Return((chan<- *fftypes.UUID)(make(chan *fftypes.UUID, 1)))
	mem.On("ChangeEvents").Return((chan<- *fftypes.ChangeEvent)(make(chan *fftypes.ChangeEvent, 1)))
	o.UUIDCollectionNSEvent(database.CollectionSubscriptions, fftypes.ChangeEventTypeCreated, "ns1", fftypes.NewUUID())
	mem.AssertExpectations(t)
}

func TestSubscriptionUpdated(t *testing.T) {
	mem := &eventmocks.EventManager{}
	o := &orchestrator{
		events: mem,
	}
	mem.On("SubscriptionUpdates").Return((chan<- *fftypes.UUID)(make(chan *fftypes.UUID, 1)))
	mem.On("ChangeEvents").Return((chan<- *fftypes.ChangeEvent)(make(chan *fftypes.ChangeEvent, 1)))
	o.UUIDCollectionNSEvent(database.CollectionSubscriptions, fftypes.ChangeEventTypeUpdated, "ns1", fftypes.NewUUID())
	mem.AssertExpectations(t)
}

func TestSubscriptionDeleted(t *testing.T) {
	mem := &eventmocks.EventManager{}
	o := &orchestrator{
		events: mem,
	}
	mem.On("DeletedSubscriptions").Return((chan<- *fftypes.UUID)(make(chan *fftypes.UUID, 1)))
	mem.On("ChangeEvents").Return((chan<- *fftypes.ChangeEvent)(make(chan *fftypes.ChangeEvent, 1)))
	o.UUIDCollectionNSEvent(database.CollectionSubscriptions, fftypes.ChangeEventTypeDeleted, "ns1", fftypes.NewUUID())
	mem.AssertExpectations(t)
}

func TestUUIDCollectionEventFull(t *testing.T) {
	mem := &eventmocks.EventManager{}
	o := &orchestrator{
		ctx:    context.Background(),
		events: mem,
	}
	mem.On("ChangeEvents").Return((chan<- *fftypes.ChangeEvent)(make(chan *fftypes.ChangeEvent, 0)))
	o.UUIDCollectionEvent(database.CollectionNamespaces, fftypes.ChangeEventTypeDeleted, fftypes.NewUUID())
	mem.AssertExpectations(t)
}

func TestHashCollectionNSEventOk(t *testing.T) {
	mem := &eventmocks.EventManager{}
	o := &orchestrator{
		ctx:    context.Background(),
		events: mem,
	}
	mem.On("ChangeEvents").Return((chan<- *fftypes.ChangeEvent)(make(chan *fftypes.ChangeEvent, 1)))
	o.HashCollectionNSEvent(database.CollectionGroups, fftypes.ChangeEventTypeDeleted, "ns1", fftypes.NewRandB32())
	mem.AssertExpectations(t)
}
