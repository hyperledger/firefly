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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestContractEventWithRetries(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	ev := &blockchain.EventWithSubscription{
		Subscription: "sb-1",
		Event: blockchain.Event{
			BlockchainTXID: "0xabcd1234",
			ProtocolID:     "10/20/30",
			Name:           "Changed",
			Output: fftypes.JSONObject{
				"value": "1",
			},
			Info: fftypes.JSONObject{
				"blockNumber": "10",
			},
		},
	}
	sub := &core.ContractListener{
		Namespace: "ns",
		ID:        fftypes.NewUUID(),
		Topic:     "topic1",
	}
	var eventID *fftypes.UUID

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetContractListenerByBackendID", mock.Anything, "sb-1").Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetContractListenerByBackendID", mock.Anything, "sb-1").Return(sub, nil).Times(1) // cached
	mth := em.txHelper.(*txcommonmocks.Helper)
	mdi.On("GetBlockchainEventByProtocolID", mock.Anything, "ns", sub.ID, ev.ProtocolID).Return(nil, nil)
	mth.On("InsertBlockchainEvent", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Once()
	mth.On("InsertBlockchainEvent", mock.Anything, mock.MatchedBy(func(e *core.BlockchainEvent) bool {
		eventID = e.ID
		return *e.Listener == *sub.ID && e.Name == "Changed" && e.Namespace == "ns"
	})).Return(nil).Times(2)
	mdi.On("GetContractListenerByID", mock.Anything, sub.ID).Return(sub, nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Once()
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(e *core.Event) bool {
		return e.Type == core.EventTypeBlockchainEventReceived && e.Reference != nil && e.Reference == eventID && e.Topic == "topic1"
	})).Return(nil).Once()

	err := em.BlockchainEvent(ev)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestContractEventUnknownSubscription(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	ev := &blockchain.EventWithSubscription{
		Subscription: "sb-1",
		Event: blockchain.Event{
			BlockchainTXID: "0xabcd1234",
			Name:           "Changed",
			Output: fftypes.JSONObject{
				"value": "1",
			},
			Info: fftypes.JSONObject{
				"blockNumber": "10",
			},
		},
	}

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetContractListenerByBackendID", mock.Anything, "sb-1").Return(nil, nil)

	err := em.BlockchainEvent(ev)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestPersistBlockchainEventDuplicate(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	ev := &core.BlockchainEvent{
		Name:       "Changed",
		Namespace:  "ns",
		ProtocolID: "10/20/30",
		Output: fftypes.JSONObject{
			"value": "1",
		},
		Info: fftypes.JSONObject{
			"blockNumber": "10",
		},
		Listener: fftypes.NewUUID(),
	}

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetBlockchainEventByProtocolID", mock.Anything, "ns", ev.Listener, ev.ProtocolID).Return(&core.BlockchainEvent{}, nil)

	err := em.maybePersistBlockchainEvent(em.ctx, ev)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestPersistBlockchainEventLookupFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	ev := &core.BlockchainEvent{
		Name:       "Changed",
		Namespace:  "ns",
		ProtocolID: "10/20/30",
		Output: fftypes.JSONObject{
			"value": "1",
		},
		Info: fftypes.JSONObject{
			"blockNumber": "10",
		},
		Listener: fftypes.NewUUID(),
	}

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetBlockchainEventByProtocolID", mock.Anything, "ns", ev.Listener, ev.ProtocolID).Return(nil, fmt.Errorf("pop"))

	err := em.maybePersistBlockchainEvent(em.ctx, ev)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestPersistBlockchainEventChainListenerLookupFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	ev := &core.BlockchainEvent{
		Name:       "Changed",
		Namespace:  "ns",
		ProtocolID: "10/20/30",
		Output: fftypes.JSONObject{
			"value": "1",
		},
		Info: fftypes.JSONObject{
			"blockNumber": "10",
		},
		Listener: fftypes.NewUUID(),
	}

	mdi := em.database.(*databasemocks.Plugin)
	mth := em.txHelper.(*txcommonmocks.Helper)
	mdi.On("GetBlockchainEventByProtocolID", mock.Anything, "ns", ev.Listener, ev.ProtocolID).Return(nil, nil)
	mth.On("InsertBlockchainEvent", mock.Anything, mock.Anything).Return(nil)
	mdi.On("GetContractListenerByID", mock.Anything, ev.Listener).Return(nil, fmt.Errorf("pop"))

	err := em.maybePersistBlockchainEvent(em.ctx, ev)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestGetTopicForChainListenerFallback(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	sub := &core.ContractListener{
		Namespace: "ns",
		ID:        fftypes.NewUUID(),
		Topic:     "",
	}

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetContractListenerByID", mock.Anything, mock.Anything).Return(sub, nil)

	topic, err := em.getTopicForChainListener(em.ctx, sub.ID)
	assert.NoError(t, err)
	assert.Equal(t, sub.ID.String(), topic)

	mdi.AssertExpectations(t)
}

func TestBlockchainEventMetric(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mm := &metricsmocks.Manager{}
	em.metrics = mm
	mm.On("IsMetricsEnabled").Return(true)
	mm.On("BlockchainEvent", mock.Anything, mock.Anything).Return()

	event := blockchain.Event{
		BlockchainTXID: "0xabcd1234",
		Name:           "Changed",
		Output: fftypes.JSONObject{
			"value": "1",
		},
		Info: fftypes.JSONObject{
			"blockNumber": "10",
		},
		Location:  "0x12345",
		Signature: "John Hancock",
	}

	em.emitBlockchainEventMetric(&event)
	mm.AssertExpectations(t)
}
