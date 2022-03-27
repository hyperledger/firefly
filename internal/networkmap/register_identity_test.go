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
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRegisterIdentityOrgWithParentOk(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	parentIdentity := testOrg("parent1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(parentIdentity, false, nil)
	mim.On("ResolveIdentitySigner", nm.ctx, parentIdentity).Return(&fftypes.SignerRef{
		Key: "0x23456",
	}, nil)

	mockMsg1 := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mockMsg2 := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)

	mbm.On("BroadcastIdentityClaim", nm.ctx,
		fftypes.SystemNamespace,
		mock.AnythingOfType("*fftypes.IdentityClaim"),
		mock.MatchedBy(func(sr *fftypes.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		fftypes.SystemTagIdentityClaim, false).Return(mockMsg1, nil)

	mbm.On("BroadcastDefinition", nm.ctx,
		fftypes.SystemNamespace,
		mock.AnythingOfType("*fftypes.IdentityVerification"),
		mock.MatchedBy(func(sr *fftypes.SignerRef) bool {
			return sr.Key == "0x23456"
		}),
		fftypes.SystemTagIdentityVerification, false).Return(mockMsg2, nil)

	org, err := nm.RegisterIdentity(nm.ctx, fftypes.SystemNamespace, &fftypes.IdentityCreateDTO{
		Name:   "child1",
		Key:    "0x12345",
		Parent: fftypes.NewUUID().String(),
	}, false)
	assert.NoError(t, err)
	assert.Equal(t, *mockMsg1.Header.ID, *org.Messages.Claim)
	assert.Equal(t, *mockMsg2.Header.ID, *org.Messages.Verification)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestRegisterIdentityOrgWithParentWaitConfirmOk(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	parentIdentity := testOrg("parent1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(parentIdentity, false, nil)
	mim.On("ResolveIdentitySigner", nm.ctx, parentIdentity).Return(&fftypes.SignerRef{
		Key: "0x23456",
	}, nil)

	msa := nm.syncasync.(*syncasyncmocks.Bridge)
	msa.On("WaitForIdentity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		ctx := args[0].(context.Context)
		ns := args[1].(string)
		id := args[2].(*fftypes.UUID)
		assert.Equal(t, parentIdentity.Namespace, ns)
		assert.NotNil(t, id)
		cb := args[3].(syncasync.RequestSender)
		err := cb(ctx)
		assert.NoError(t, err)
	}).Return(nil, nil)

	mockMsg1 := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mockMsg2 := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)

	mbm.On("BroadcastIdentityClaim", nm.ctx,
		fftypes.SystemNamespace,
		mock.AnythingOfType("*fftypes.IdentityClaim"),
		mock.MatchedBy(func(sr *fftypes.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		fftypes.SystemTagIdentityClaim, false).Return(mockMsg1, nil)

	mbm.On("BroadcastDefinition", nm.ctx,
		fftypes.SystemNamespace,
		mock.AnythingOfType("*fftypes.IdentityVerification"),
		mock.MatchedBy(func(sr *fftypes.SignerRef) bool {
			return sr.Key == "0x23456"
		}),
		fftypes.SystemTagIdentityVerification, false).Return(mockMsg2, nil)

	_, err := nm.RegisterIdentity(nm.ctx, fftypes.SystemNamespace, &fftypes.IdentityCreateDTO{
		Name:   "child1",
		Key:    "0x12345",
		Parent: fftypes.NewUUID().String(),
	}, true)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	msa.AssertExpectations(t)
}

func TestRegisterIdentityCustomWithParentFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	parentIdentity := testOrg("parent1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(parentIdentity, false, nil)
	mim.On("CachedIdentityLookupMustExist", nm.ctx, "did:firefly:org/parent1").Return(&fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:  fftypes.NewUUID(),
			DID: "did:firefly:org/parent1",
		},
	}, false, nil)
	mim.On("ResolveIdentitySigner", nm.ctx, parentIdentity).Return(&fftypes.SignerRef{
		Key: "0x23456",
	}, nil)

	mockMsg := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)

	mbm.On("BroadcastIdentityClaim", nm.ctx,
		"ns1",
		mock.AnythingOfType("*fftypes.IdentityClaim"),
		mock.MatchedBy(func(sr *fftypes.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		fftypes.SystemTagIdentityClaim, false).Return(mockMsg, nil)

	mbm.On("BroadcastDefinition", nm.ctx,
		"ns1",
		mock.AnythingOfType("*fftypes.IdentityVerification"),
		mock.MatchedBy(func(sr *fftypes.SignerRef) bool {
			return sr.Key == "0x23456"
		}),
		fftypes.SystemTagIdentityVerification, false).Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterIdentity(nm.ctx, "ns1", &fftypes.IdentityCreateDTO{
		Name:   "custom1",
		Key:    "0x12345",
		Parent: "did:firefly:org/parent1",
	}, false)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestRegisterIdentityGetParentMsgFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	parentIdentity := testOrg("parent1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(parentIdentity, false, nil)
	mim.On("ResolveIdentitySigner", nm.ctx, parentIdentity).Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterIdentity(nm.ctx, "ns1", &fftypes.IdentityCreateDTO{
		Name:   "custom1",
		Key:    "0x12345",
		Parent: fftypes.NewUUID().String(),
	}, false)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
}

func TestRegisterIdentityRootBroadcastFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(nil, false, nil)

	mbm := nm.broadcast.(*broadcastmocks.Manager)
	mbm.On("BroadcastIdentityClaim", nm.ctx,
		"ns1",
		mock.AnythingOfType("*fftypes.IdentityClaim"),
		mock.MatchedBy(func(sr *fftypes.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		fftypes.SystemTagIdentityClaim, false).Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterIdentity(nm.ctx, "ns1", &fftypes.IdentityCreateDTO{
		Name:   "custom1",
		Key:    "0x12345",
		Parent: fftypes.NewUUID().String(),
	}, false)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestRegisterIdentityMissingKey(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(nil, false, nil)

	_, err := nm.RegisterIdentity(nm.ctx, "ns1", &fftypes.IdentityCreateDTO{
		Name:   "custom1",
		Parent: fftypes.NewUUID().String(),
	}, false)
	assert.Regexp(t, "FF10352", err)

	mim.AssertExpectations(t)
}

func TestRegisterIdentityVerifyFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(nil, false, fmt.Errorf("pop"))

	_, err := nm.RegisterIdentity(nm.ctx, "ns1", &fftypes.IdentityCreateDTO{
		Name:   "custom1",
		Parent: fftypes.NewUUID().String(),
	}, false)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
}

func TestRegisterIdentityBadParent(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupMustExist", nm.ctx, "did:firefly:org/1").Return(nil, false, fmt.Errorf("pop"))

	_, err := nm.RegisterIdentity(nm.ctx, "ns1", &fftypes.IdentityCreateDTO{
		Name:   "custom1",
		Parent: "did:firefly:org/1",
	}, false)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
}
