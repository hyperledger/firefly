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

package networkmap

import (
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRegisterIdentityOrgWithParentOk(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	parentIdentity := testOrg("parent1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(parentIdentity, nil)

	parentClaimMsg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			SignerRef: fftypes.SignerRef{
				Key: "0x23456",
			},
		},
	}
	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", nm.ctx, parentIdentity.Messages.Claim).Return(parentClaimMsg, nil)

	mockMsg1 := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mockMsg2 := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)

	mbm.On("BroadcastDefinition", nm.ctx,
		fftypes.SystemNamespace,
		mock.AnythingOfType("*fftypes.IdentityClaim"),
		mock.MatchedBy(func(sr *fftypes.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		fftypes.SystemTagIdentityClaim, true).Return(mockMsg1, nil)

	mbm.On("BroadcastDefinition", nm.ctx,
		fftypes.SystemNamespace,
		mock.AnythingOfType("*fftypes.IdentityVerification"),
		mock.MatchedBy(func(sr *fftypes.SignerRef) bool {
			return sr.Key == "0x23456"
		}),
		fftypes.SystemTagIdentityVerification, true).Return(mockMsg2, nil)

	org, err := nm.RegisterIdentity(nm.ctx, &fftypes.IdentityCreateDTO{
		Name:   "child1",
		Key:    "0x12345",
		Parent: fftypes.NewUUID(),
	}, true)
	assert.NoError(t, err)
	assert.Equal(t, *mockMsg1.Header.ID, *org.Messages.Claim)
	assert.Equal(t, *mockMsg2.Header.ID, *org.Messages.Verification)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestRegisterIdentityCustomWithParentFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	parentIdentity := testOrg("parent1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(parentIdentity, nil)

	parentClaimMsg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			SignerRef: fftypes.SignerRef{
				Key: "0x23456",
			},
		},
	}
	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", nm.ctx, parentIdentity.Messages.Claim).Return(parentClaimMsg, nil)

	mockMsg := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)

	mbm.On("BroadcastDefinition", nm.ctx,
		"ns1",
		mock.AnythingOfType("*fftypes.IdentityClaim"),
		mock.MatchedBy(func(sr *fftypes.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		fftypes.SystemTagIdentityClaim, true).Return(mockMsg, nil)

	mbm.On("BroadcastDefinition", nm.ctx,
		"ns1",
		mock.AnythingOfType("*fftypes.IdentityVerification"),
		mock.MatchedBy(func(sr *fftypes.SignerRef) bool {
			return sr.Key == "0x23456"
		}),
		fftypes.SystemTagIdentityVerification, true).Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterIdentity(nm.ctx, &fftypes.IdentityCreateDTO{
		Namespace: "ns1",
		Name:      "custom1",
		Key:       "0x12345",
		Parent:    fftypes.NewUUID(),
	}, true)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestRegisterIdentityGetParentMsgFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	parentIdentity := testOrg("parent1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(parentIdentity, nil)

	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", nm.ctx, parentIdentity.Messages.Claim).Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterIdentity(nm.ctx, &fftypes.IdentityCreateDTO{
		Namespace: "ns1",
		Name:      "custom1",
		Key:       "0x12345",
		Parent:    fftypes.NewUUID(),
	}, true)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestRegisterIdentityGetParentMsgNotFound(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	parentIdentity := testOrg("parent1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(parentIdentity, nil)

	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", nm.ctx, parentIdentity.Messages.Claim).Return(nil, nil)

	_, err := nm.RegisterIdentity(nm.ctx, &fftypes.IdentityCreateDTO{
		Namespace: "ns1",
		Name:      "custom1",
		Key:       "0x12345",
		Parent:    fftypes.NewUUID(),
	}, true)
	assert.Regexp(t, "FF10363", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestRegisterIdentityRootBroadcastFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(nil, nil)

	mbm := nm.broadcast.(*broadcastmocks.Manager)
	mbm.On("BroadcastDefinition", nm.ctx,
		"ns1",
		mock.AnythingOfType("*fftypes.IdentityClaim"),
		mock.MatchedBy(func(sr *fftypes.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		fftypes.SystemTagIdentityClaim, true).Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterIdentity(nm.ctx, &fftypes.IdentityCreateDTO{
		Namespace: "ns1",
		Name:      "custom1",
		Key:       "0x12345",
	}, true)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestRegisterIdentityMissingKey(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(nil, nil)

	_, err := nm.RegisterIdentity(nm.ctx, &fftypes.IdentityCreateDTO{
		Namespace: "ns1",
		Name:      "custom1",
	}, true)
	assert.Regexp(t, "FF10349", err)

	mim.AssertExpectations(t)
}

func TestRegisterIdentityVerifyFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterIdentity(nm.ctx, &fftypes.IdentityCreateDTO{
		Namespace: "ns1",
		Name:      "custom1",
	}, true)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
}
