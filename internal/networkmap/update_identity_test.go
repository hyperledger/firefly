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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUpdateIdentityProfileOk(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	identity := testOrg("org1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", nm.ctx, identity.ID).Return(identity, nil)
	signerRef := &core.SignerRef{Key: "0x12345"}
	mim.On("ResolveIdentitySigner", nm.ctx, identity).Return(signerRef, nil)

	mockMsg1 := &core.Message{Header: core.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)

	mbm.On("BroadcastDefinition", nm.ctx,
		core.SystemNamespace,
		mock.AnythingOfType("*core.IdentityUpdate"),
		mock.MatchedBy(func(sr *core.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		core.SystemTagIdentityUpdate, true).Return(mockMsg1, nil)

	org, err := nm.UpdateIdentity(nm.ctx, identity.Namespace, identity.ID.String(), &core.IdentityUpdateDTO{
		IdentityProfile: core.IdentityProfile{
			Description: "new desc",
			Profile:     fftypes.JSONObject{"new": "profile"},
		},
	}, true)
	assert.NoError(t, err)
	assert.Equal(t, *mockMsg1.Header.ID, *org.Messages.Update)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestUpdateIdentityProfileBroadcastFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	identity := testOrg("org1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", nm.ctx, identity.ID).Return(identity, nil)
	signerRef := &core.SignerRef{Key: "0x12345"}
	mim.On("ResolveIdentitySigner", nm.ctx, identity).Return(signerRef, nil)

	mbm := nm.broadcast.(*broadcastmocks.Manager)
	mbm.On("BroadcastDefinition", nm.ctx,
		core.SystemNamespace,
		mock.AnythingOfType("*core.IdentityUpdate"),
		mock.MatchedBy(func(sr *core.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		core.SystemTagIdentityUpdate, true).Return(nil, fmt.Errorf("pop"))

	_, err := nm.UpdateIdentity(nm.ctx, identity.Namespace, identity.ID.String(), &core.IdentityUpdateDTO{
		IdentityProfile: core.IdentityProfile{
			Description: "new desc",
			Profile:     fftypes.JSONObject{"new": "profile"},
		},
	}, true)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestUpdateIdentityProfileBadProfile(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	identity := testOrg("org1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", nm.ctx, identity.ID).Return(identity, nil)
	signerRef := &core.SignerRef{Key: "0x12345"}
	mim.On("ResolveIdentitySigner", nm.ctx, identity).Return(signerRef, nil)

	_, err := nm.UpdateIdentity(nm.ctx, identity.Namespace, identity.ID.String(), &core.IdentityUpdateDTO{
		IdentityProfile: core.IdentityProfile{
			Description: string(make([]byte, 4097)),
			Profile:     fftypes.JSONObject{"new": "profile"},
		},
	}, true)
	assert.Regexp(t, "FF00135", err)

	mim.AssertExpectations(t)
}

func TestUpdateIdentityProfileNotFound(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	identity := testOrg("org1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", nm.ctx, identity.ID).Return(nil, nil)

	_, err := nm.UpdateIdentity(nm.ctx, identity.Namespace, identity.ID.String(), &core.IdentityUpdateDTO{
		IdentityProfile: core.IdentityProfile{
			Description: string(make([]byte, 4097)),
			Profile:     fftypes.JSONObject{"new": "profile"},
		},
	}, true)
	assert.Regexp(t, "FF10143", err)

	mim.AssertExpectations(t)
}

func TestUpdateIdentityProfileLookupFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	identity := testOrg("org1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", nm.ctx, identity.ID).Return(nil, fmt.Errorf("pop"))

	_, err := nm.UpdateIdentity(nm.ctx, identity.Namespace, identity.ID.String(), &core.IdentityUpdateDTO{
		IdentityProfile: core.IdentityProfile{
			Description: string(make([]byte, 4097)),
			Profile:     fftypes.JSONObject{"new": "profile"},
		},
	}, true)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
}

func TestUpdateIdentityProfileClaimLookupFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	identity := testOrg("org1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", nm.ctx, identity.ID).Return(identity, nil)
	signerRef := &core.SignerRef{Key: "0x12345"}
	mim.On("ResolveIdentitySigner", nm.ctx, identity).Return(signerRef, fmt.Errorf("pop"))

	_, err := nm.UpdateIdentity(nm.ctx, identity.Namespace, identity.ID.String(), &core.IdentityUpdateDTO{
		IdentityProfile: core.IdentityProfile{
			Description: "Desc1",
			Profile:     fftypes.JSONObject{"new": "profile"},
		},
	}, true)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
}

func TestUpdateIdentityProfileBadID(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	_, err := nm.UpdateIdentity(nm.ctx, "ns1", "badness", &core.IdentityUpdateDTO{}, true)
	assert.Regexp(t, "FF00138", err)
}
