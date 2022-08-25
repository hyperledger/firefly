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
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNetworkAction(t *testing.T) {
	em := newTestEventManager(t)
	defer em.cleanup(t)

	location := fftypes.JSONAnyPtr("{}")
	event := &blockchain.Event{ProtocolID: "0001"}
	verifier := &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x1234",
	}

	em.mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeOrg}, verifier).Return(&core.Identity{}, nil)
	em.mth.On("InsertOrGetBlockchainEvent", em.ctx, mock.MatchedBy(func(be *core.BlockchainEvent) bool {
		return be.ProtocolID == "0001"
	})).Return(nil, nil)
	em.mdi.On("InsertEvent", em.ctx, mock.Anything).Return(nil)
	em.mmp.On("TerminateContract", em.ctx, location, mock.AnythingOfType("*blockchain.Event")).Return(nil)

	err := em.BlockchainNetworkAction("terminate", location, event, verifier)
	assert.NoError(t, err)
}

func TestNetworkActionUnknownIdentity(t *testing.T) {
	em := newTestEventManager(t)
	defer em.cleanup(t)

	location := fftypes.JSONAnyPtr("{}")
	verifier := &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x1234",
	}

	em.mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeOrg}, verifier).Return(nil, fmt.Errorf("pop")).Once()
	em.mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeOrg}, verifier).Return(nil, nil).Once()

	err := em.BlockchainNetworkAction("terminate", location, &blockchain.Event{}, verifier)
	assert.NoError(t, err)
}

func TestNetworkActionNonRootIdentity(t *testing.T) {
	em := newTestEventManager(t)
	defer em.cleanup(t)

	location := fftypes.JSONAnyPtr("{}")
	verifier := &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x1234",
	}

	em.mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeOrg}, verifier).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			Parent: fftypes.NewUUID(),
		},
	}, nil)

	err := em.BlockchainNetworkAction("terminate", location, &blockchain.Event{}, verifier)
	assert.NoError(t, err)
}

func TestNetworkActionNonMultiparty(t *testing.T) {
	em := newTestEventManager(t)
	defer em.cleanup(t)
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
	em := newTestEventManager(t)
	defer em.cleanup(t)

	location := fftypes.JSONAnyPtr("{}")
	verifier := &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x1234",
	}

	em.mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeOrg}, verifier).Return(&core.Identity{}, nil)

	err := em.BlockchainNetworkAction("bad", location, &blockchain.Event{}, verifier)
	assert.NoError(t, err)
}

func TestActionTerminateFail(t *testing.T) {
	em := newTestEventManager(t)
	defer em.cleanup(t)

	location := fftypes.JSONAnyPtr("{}")

	em.mmp.On("TerminateContract", em.ctx, location, mock.AnythingOfType("*blockchain.Event")).Return(fmt.Errorf("pop"))

	err := em.actionTerminate(location, &blockchain.Event{})
	assert.EqualError(t, err, "pop")
}
