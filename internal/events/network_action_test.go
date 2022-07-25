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
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/multipartymocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNetworkAction(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	location := fftypes.JSONAnyPtr("{}")
	event := &blockchain.Event{ProtocolID: "0001"}
	verifier := &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x1234",
	}

	mmp := em.multiparty.(*multipartymocks.Manager)
	mdi := em.database.(*databasemocks.Plugin)
	mth := em.txHelper.(*txcommonmocks.Helper)
	mii := em.identity.(*identitymanagermocks.Manager)

	mii.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeOrg}, verifier).Return(&core.Identity{}, nil)
	mdi.On("GetBlockchainEventByProtocolID", em.ctx, "ns1", (*fftypes.UUID)(nil), "0001").Return(nil, nil)
	mth.On("InsertBlockchainEvent", em.ctx, mock.MatchedBy(func(be *core.BlockchainEvent) bool {
		return be.ProtocolID == "0001"
	})).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.Anything).Return(nil)
	mmp.On("TerminateContract", em.ctx, location, mock.AnythingOfType("*blockchain.Event")).Return(nil)

	err := em.BlockchainNetworkAction("terminate", location, event, verifier)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mth.AssertExpectations(t)
	mii.AssertExpectations(t)
}

func TestNetworkActionUnknownIdentity(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	location := fftypes.JSONAnyPtr("{}")
	verifier := &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x1234",
	}

	mii := em.identity.(*identitymanagermocks.Manager)

	mii.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeOrg}, verifier).Return(nil, fmt.Errorf("pop")).Once()
	mii.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeOrg}, verifier).Return(nil, nil).Once()

	err := em.BlockchainNetworkAction("terminate", location, &blockchain.Event{}, verifier)
	assert.NoError(t, err)

	mii.AssertExpectations(t)
}

func TestNetworkActionNonRootIdentity(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	location := fftypes.JSONAnyPtr("{}")
	verifier := &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x1234",
	}

	mii := em.identity.(*identitymanagermocks.Manager)

	mii.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeOrg}, verifier).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			Parent: fftypes.NewUUID(),
		},
	}, nil)

	err := em.BlockchainNetworkAction("terminate", location, &blockchain.Event{}, verifier)
	assert.NoError(t, err)

	mii.AssertExpectations(t)
}

func TestNetworkActionNonMultiparty(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	em.multiparty = nil

	location := fftypes.JSONAnyPtr("{}")
	verifier := &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x1234",
	}

	err := em.BlockchainNetworkAction("terminate", location, &blockchain.Event{}, verifier)
	assert.NoError(t, err)
}

func TestNetworkActionUnknown(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	location := fftypes.JSONAnyPtr("{}")
	verifier := &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x1234",
	}

	mii := em.identity.(*identitymanagermocks.Manager)

	mii.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeOrg}, verifier).Return(&core.Identity{}, nil)

	err := em.BlockchainNetworkAction("bad", location, &blockchain.Event{}, verifier)
	assert.NoError(t, err)

	mii.AssertExpectations(t)
}

func TestActionTerminateFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	location := fftypes.JSONAnyPtr("{}")

	mmp := em.multiparty.(*multipartymocks.Manager)
	mdi := em.database.(*databasemocks.Plugin)

	mmp.On("TerminateContract", em.ctx, location, mock.AnythingOfType("*blockchain.Event")).Return(fmt.Errorf("pop"))

	err := em.actionTerminate(location, &blockchain.Event{})
	assert.EqualError(t, err, "pop")

	mmp.AssertExpectations(t)
	mdi.AssertExpectations(t)
}
