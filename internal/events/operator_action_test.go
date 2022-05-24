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
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOperatorAction(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	event := &blockchain.Event{ProtocolID: "0001"}

	mbi := &blockchainmocks.Plugin{}
	mdi := em.database.(*databasemocks.Plugin)
	mth := em.txHelper.(*txcommonmocks.Helper)

	mdi.On("GetBlockchainEventByProtocolID", em.ctx, "ff_system", (*fftypes.UUID)(nil), "0001").Return(nil, nil)
	mth.On("InsertBlockchainEvent", em.ctx, mock.MatchedBy(func(be *core.BlockchainEvent) bool {
		return be.ProtocolID == "0001"
	})).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.Anything).Return(nil)
	mdi.On("GetNamespace", em.ctx, "ff_system").Return(&core.Namespace{}, nil)
	mdi.On("UpsertNamespace", em.ctx, mock.AnythingOfType("*core.Namespace"), true).Return(nil)
	mbi.On("TerminateContract", mock.AnythingOfType("*core.FireFlyContracts"), mock.AnythingOfType("*blockchain.Event")).Return(nil)

	err := em.BlockchainOperatorAction(mbi, "terminate", event, &core.VerifierRef{})
	assert.NoError(t, err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
}

func TestOperatorActionUnknown(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mbi := &blockchainmocks.Plugin{}

	err := em.BlockchainOperatorAction(mbi, "bad", &blockchain.Event{}, &core.VerifierRef{})
	assert.NoError(t, err)

	mbi.AssertExpectations(t)
}

func TestActionTerminateQueryFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mbi := &blockchainmocks.Plugin{}
	mdi := em.database.(*databasemocks.Plugin)

	mdi.On("GetNamespace", em.ctx, "ff_system").Return(nil, fmt.Errorf("pop"))

	err := em.actionTerminate(mbi, &blockchain.Event{})
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestActionTerminateUpsertFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mbi := &blockchainmocks.Plugin{}
	mdi := em.database.(*databasemocks.Plugin)

	mdi.On("GetNamespace", em.ctx, "ff_system").Return(&core.Namespace{}, nil)
	mdi.On("UpsertNamespace", em.ctx, mock.AnythingOfType("*core.Namespace"), true).Return(fmt.Errorf("pop"))
	mbi.On("TerminateContract", mock.AnythingOfType("*core.FireFlyContracts"), mock.AnythingOfType("*blockchain.Event")).Return(nil)

	err := em.actionTerminate(mbi, &blockchain.Event{})
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}
