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

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/fftypes"
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
			Name:           "Changed",
			Output: fftypes.JSONObject{
				"value": "1",
			},
			Info: fftypes.JSONObject{
				"blockNumber": "10",
			},
		},
	}
	sub := &fftypes.ContractListener{
		Namespace: "ns",
		ID:        fftypes.NewUUID(),
		Topic:     "topic1",
	}
	var eventID *fftypes.UUID

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetContractListenerByProtocolID", mock.Anything, "sb-1").Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetContractListenerByProtocolID", mock.Anything, "sb-1").Return(sub, nil).Times(1) // cached
	mdi.On("InsertBlockchainEvent", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Once()
	mdi.On("InsertBlockchainEvent", mock.Anything, mock.MatchedBy(func(e *fftypes.BlockchainEvent) bool {
		eventID = e.ID
		return *e.Listener == *sub.ID && e.Name == "Changed" && e.Namespace == "ns"
	})).Return(nil).Times(2)
	mdi.On("GetContractListenerByID", mock.Anything, sub.ID).Return(sub, nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Once()
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeBlockchainEventReceived && e.Reference != nil && e.Reference == eventID && e.Topic == "topic1"
	})).Return(nil).Once()

	err := em.BlockchainEvent(ev)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
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
	mdi.On("GetContractListenerByProtocolID", mock.Anything, "sb-1").Return(nil, nil)

	err := em.BlockchainEvent(ev)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestPersistBlockchainEventChainListenerLoopkupFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	ev := &fftypes.BlockchainEvent{
		Name: "Changed",
		Output: fftypes.JSONObject{
			"value": "1",
		},
		Info: fftypes.JSONObject{
			"blockNumber": "10",
		},
		Listener: fftypes.NewUUID(),
	}

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("InsertBlockchainEvent", mock.Anything, mock.Anything).Return(nil)
	mdi.On("GetContractListenerByID", mock.Anything, ev.Listener).Return(nil, fmt.Errorf("pop"))

	err := em.persistBlockchainEvent(em.ctx, ev)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestGetTopicForChainListenerFallback(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	sub := &fftypes.ContractListener{
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
