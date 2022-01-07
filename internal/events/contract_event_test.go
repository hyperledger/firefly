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

	ev := &blockchain.ContractEvent{
		Subscription: "sb-1",
		Name:         "Changed",
		Outputs: fftypes.JSONObject{
			"value": "1",
		},
		Info: fftypes.JSONObject{
			"blockNumber": "10",
		},
	}
	sub := &fftypes.ContractSubscription{
		Namespace: "ns",
		ID:        fftypes.NewUUID(),
	}
	var eventID *fftypes.UUID

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetContractSubscriptionByProtocolID", mock.Anything, "sb-1").Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetContractSubscriptionByProtocolID", mock.Anything, "sb-1").Return(sub, nil).Times(3)
	mdi.On("InsertBlockchainEvent", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Once()
	mdi.On("InsertBlockchainEvent", mock.Anything, mock.MatchedBy(func(e *fftypes.BlockchainEvent) bool {
		eventID = e.ID
		return *e.Subscription == *sub.ID && e.Name == "Changed" && e.Namespace == "ns"
	})).Return(nil).Times(2)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Once()
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeBlockchainEvent && e.Reference != nil && e.Reference == eventID
	})).Return(nil).Once()

	err := em.ContractEvent(ev)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestContractEventUnknownSubscription(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	ev := &blockchain.ContractEvent{
		Subscription: "sb-1",
		Name:         "Changed",
		Outputs: fftypes.JSONObject{
			"value": "1",
		},
		Info: fftypes.JSONObject{
			"blockNumber": "10",
		},
	}

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetContractSubscriptionByProtocolID", mock.Anything, "sb-1").Return(nil, nil)

	err := em.ContractEvent(ev)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}
