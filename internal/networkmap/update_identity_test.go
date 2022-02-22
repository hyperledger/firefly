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

func TestUpdateIdentityProfileOk(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	identity := testOrg("org1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", nm.ctx, identity.ID).Return(identity, nil)

	claimMsg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			SignerRef: fftypes.SignerRef{
				Key: "0x12345",
			},
		},
	}
	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", nm.ctx, identity.Messages.Claim).Return(claimMsg, nil)

	mockMsg1 := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)

	mbm.On("BroadcastDefinition", nm.ctx,
		fftypes.SystemNamespace,
		mock.AnythingOfType("*fftypes.IdentityProfileUpdate"),
		mock.MatchedBy(func(sr *fftypes.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		fftypes.SystemTagIdentityUpdate, true).Return(mockMsg1, nil)

	org, err := nm.UpdateIdentityProfile(nm.ctx, &fftypes.IdentityUpdateDTO{
		ID: identity.ID,
		IdentityProfile: fftypes.IdentityProfile{
			Description: "new desc",
			Profile:     fftypes.JSONObject{"new": "profile"},
		},
	}, true)
	assert.NoError(t, err)
	assert.Equal(t, *mockMsg1.Header.ID, *org.Messages.Update)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestUpdateIdentityProfileBroadcastFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	identity := testOrg("org1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", nm.ctx, identity.ID).Return(identity, nil)

	claimMsg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			SignerRef: fftypes.SignerRef{
				Key: "0x12345",
			},
		},
	}
	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", nm.ctx, identity.Messages.Claim).Return(claimMsg, nil)

	mbm := nm.broadcast.(*broadcastmocks.Manager)
	mbm.On("BroadcastDefinition", nm.ctx,
		fftypes.SystemNamespace,
		mock.AnythingOfType("*fftypes.IdentityProfileUpdate"),
		mock.MatchedBy(func(sr *fftypes.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		fftypes.SystemTagIdentityUpdate, true).Return(nil, fmt.Errorf("pop"))

	_, err := nm.UpdateIdentityProfile(nm.ctx, &fftypes.IdentityUpdateDTO{
		ID: identity.ID,
		IdentityProfile: fftypes.IdentityProfile{
			Description: "new desc",
			Profile:     fftypes.JSONObject{"new": "profile"},
		},
	}, true)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mbm.AssertExpectations(t)
}

func TestUpdateIdentityProfileBadProfile(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	identity := testOrg("org1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", nm.ctx, identity.ID).Return(identity, nil)

	claimMsg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			SignerRef: fftypes.SignerRef{
				Key: "0x12345",
			},
		},
	}
	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", nm.ctx, identity.Messages.Claim).Return(claimMsg, nil)

	_, err := nm.UpdateIdentityProfile(nm.ctx, &fftypes.IdentityUpdateDTO{
		ID: identity.ID,
		IdentityProfile: fftypes.IdentityProfile{
			Description: string(make([]byte, 4097)),
			Profile:     fftypes.JSONObject{"new": "profile"},
		},
	}, true)
	assert.Regexp(t, "FF10188", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestUpdateIdentityProfileNotFound(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	identity := testOrg("org1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("CachedIdentityLookupByID", nm.ctx, identity.ID).Return(nil, nil)

	_, err := nm.UpdateIdentityProfile(nm.ctx, &fftypes.IdentityUpdateDTO{
		ID: identity.ID,
		IdentityProfile: fftypes.IdentityProfile{
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

	_, err := nm.UpdateIdentityProfile(nm.ctx, &fftypes.IdentityUpdateDTO{
		ID: identity.ID,
		IdentityProfile: fftypes.IdentityProfile{
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

	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", nm.ctx, identity.Messages.Claim).Return(nil, nil)

	_, err := nm.UpdateIdentityProfile(nm.ctx, &fftypes.IdentityUpdateDTO{
		ID: identity.ID,
		IdentityProfile: fftypes.IdentityProfile{
			Description: string(make([]byte, 4097)),
			Profile:     fftypes.JSONObject{"new": "profile"},
		},
	}, true)
	assert.Regexp(t, "FF10366", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
}
