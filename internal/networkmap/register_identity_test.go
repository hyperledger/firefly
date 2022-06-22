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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRegisterIdentityOrgWithParentOk(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	parentIdentity := testOrg("parent1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*core.Identity")).Return(parentIdentity, false, nil)
	mim.On("ResolveIdentitySigner", nm.ctx, parentIdentity).Return(&core.SignerRef{
		Key: "0x23456",
	}, nil)

	mdm := nm.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", nm.ctx, "ns1").Return(nil)

	mockMsg1 := &core.Message{Header: core.MessageHeader{ID: fftypes.NewUUID()}}
	mockMsg2 := &core.Message{Header: core.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)

	mbm.On("BroadcastIdentityClaim", nm.ctx,
		mock.AnythingOfType("*core.IdentityClaim"),
		mock.MatchedBy(func(sr *core.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		core.SystemTagIdentityClaim, false).Return(mockMsg1, nil)

	mbm.On("BroadcastDefinition", nm.ctx,
		mock.AnythingOfType("*core.IdentityVerification"),
		mock.MatchedBy(func(sr *core.SignerRef) bool {
			return sr.Key == "0x23456"
		}),
		core.SystemTagIdentityVerification, false).Return(mockMsg2, nil)

	org, err := nm.RegisterIdentity(nm.ctx, &core.IdentityCreateDTO{
		Name:   "child1",
		Key:    "0x12345",
		Parent: fftypes.NewUUID().String(),
	}, false)
	assert.NoError(t, err)
	assert.Equal(t, *mockMsg1.Header.ID, *org.Messages.Claim)
	assert.Equal(t, *mockMsg2.Header.ID, *org.Messages.Verification)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestRegisterIdentityOrgWithParentWaitConfirmOk(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	parentIdentity := testOrg("parent1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*core.Identity")).Return(parentIdentity, false, nil)
	mim.On("ResolveIdentitySigner", nm.ctx, parentIdentity).Return(&core.SignerRef{
		Key: "0x23456",
	}, nil)

	msa := nm.syncasync.(*syncasyncmocks.Bridge)
	msa.On("WaitForIdentity", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		ctx := args[0].(context.Context)
		id := args[1].(*fftypes.UUID)
		assert.NotNil(t, id)
		cb := args[2].(syncasync.RequestSender)
		err := cb(ctx)
		assert.NoError(t, err)
	}).Return(nil, nil)

	mdm := nm.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", nm.ctx, "ns1").Return(nil)

	mockMsg1 := &core.Message{Header: core.MessageHeader{ID: fftypes.NewUUID()}}
	mockMsg2 := &core.Message{Header: core.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)

	mbm.On("BroadcastIdentityClaim", nm.ctx,
		mock.AnythingOfType("*core.IdentityClaim"),
		mock.MatchedBy(func(sr *core.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		core.SystemTagIdentityClaim, false).Return(mockMsg1, nil)

	mbm.On("BroadcastDefinition", nm.ctx,
		mock.AnythingOfType("*core.IdentityVerification"),
		mock.MatchedBy(func(sr *core.SignerRef) bool {
			return sr.Key == "0x23456"
		}),
		core.SystemTagIdentityVerification, false).Return(mockMsg2, nil)

	_, err := nm.RegisterIdentity(nm.ctx, &core.IdentityCreateDTO{
		Name:   "child1",
		Key:    "0x12345",
		Parent: fftypes.NewUUID().String(),
	}, true)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	msa.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestRegisterIdentityCustomBadNS(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupMustExist", nm.ctx, "did:firefly:org/parent1").Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:  fftypes.NewUUID(),
			DID: "did:firefly:org/parent1",
		},
	}, false, nil)

	mdm := nm.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", nm.ctx, "ns1").Return(fmt.Errorf("pop"))

	_, err := nm.RegisterIdentity(nm.ctx, &core.IdentityCreateDTO{
		Name:   "custom1",
		Key:    "0x12345",
		Parent: "did:firefly:org/parent1",
	}, false)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestRegisterIdentityCustomWithParentFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	parentIdentity := testOrg("parent1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*core.Identity")).Return(parentIdentity, false, nil)
	mim.On("CachedIdentityLookupMustExist", nm.ctx, "did:firefly:org/parent1").Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:  fftypes.NewUUID(),
			DID: "did:firefly:org/parent1",
		},
	}, false, nil)
	mim.On("ResolveIdentitySigner", nm.ctx, parentIdentity).Return(&core.SignerRef{
		Key: "0x23456",
	}, nil)

	mdm := nm.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", nm.ctx, "ns1").Return(nil)

	mockMsg := &core.Message{Header: core.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)

	mbm.On("BroadcastIdentityClaim", nm.ctx,
		mock.AnythingOfType("*core.IdentityClaim"),
		mock.MatchedBy(func(sr *core.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		core.SystemTagIdentityClaim, false).Return(mockMsg, nil)

	mbm.On("BroadcastDefinition", nm.ctx,
		mock.AnythingOfType("*core.IdentityVerification"),
		mock.MatchedBy(func(sr *core.SignerRef) bool {
			return sr.Key == "0x23456"
		}),
		core.SystemTagIdentityVerification, false).Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterIdentity(nm.ctx, &core.IdentityCreateDTO{
		Name:   "custom1",
		Key:    "0x12345",
		Parent: "did:firefly:org/parent1",
	}, false)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestRegisterIdentityGetParentMsgFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	parentIdentity := testOrg("parent1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*core.Identity")).Return(parentIdentity, false, nil)
	mim.On("ResolveIdentitySigner", nm.ctx, parentIdentity).Return(nil, fmt.Errorf("pop"))

	mdm := nm.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", nm.ctx, "ns1").Return(nil)

	_, err := nm.RegisterIdentity(nm.ctx, &core.IdentityCreateDTO{
		Name:   "custom1",
		Key:    "0x12345",
		Parent: fftypes.NewUUID().String(),
	}, false)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestRegisterIdentityRootBroadcastFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*core.Identity")).Return(nil, false, nil)

	mdm := nm.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", nm.ctx, "ns1").Return(nil)

	mbm := nm.broadcast.(*broadcastmocks.Manager)
	mbm.On("BroadcastIdentityClaim", nm.ctx,
		mock.AnythingOfType("*core.IdentityClaim"),
		mock.MatchedBy(func(sr *core.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		core.SystemTagIdentityClaim, false).Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterIdentity(nm.ctx, &core.IdentityCreateDTO{
		Name:   "custom1",
		Key:    "0x12345",
		Parent: fftypes.NewUUID().String(),
	}, false)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestRegisterIdentityMissingKey(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*core.Identity")).Return(nil, false, nil)

	mdm := nm.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", nm.ctx, "ns1").Return(nil)

	_, err := nm.RegisterIdentity(nm.ctx, &core.IdentityCreateDTO{
		Name:   "custom1",
		Parent: fftypes.NewUUID().String(),
	}, false)
	assert.Regexp(t, "FF10352", err)

	mim.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestRegisterIdentityVerifyFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*core.Identity")).Return(nil, false, fmt.Errorf("pop"))

	mdm := nm.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", nm.ctx, "ns1").Return(nil)

	_, err := nm.RegisterIdentity(nm.ctx, &core.IdentityCreateDTO{
		Name:   "custom1",
		Parent: fftypes.NewUUID().String(),
	}, false)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestRegisterIdentityBadParent(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupMustExist", nm.ctx, "did:firefly:org/1").Return(nil, false, fmt.Errorf("pop"))

	_, err := nm.RegisterIdentity(nm.ctx, &core.IdentityCreateDTO{
		Name:   "custom1",
		Parent: "did:firefly:org/1",
	}, false)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
}
