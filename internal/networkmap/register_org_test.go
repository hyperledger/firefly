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

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/mocks/broadcastmocks"
	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRegisterOrganizationChildOk(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", nm.ctx, "0x23456").Return(&fftypes.Organization{
		Identity:    "0x23456",
		Description: "parent organization",
	}, nil)

	mim := nm.identity.(*identitymanagermocks.Manager)
	parentID := &fftypes.Identity{Key: "0x23456"}
	mim.On("ResolveInputIdentity", nm.ctx, mock.MatchedBy(func(i *fftypes.Identity) bool { return i.Key == "0x12345" })).Return(nil)

	mockMsg := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)
	mbm.On("BroadcastRootOrgDefinition", nm.ctx, mock.Anything, parentID, fftypes.SystemTagDefineOrganization, false).Return(mockMsg, nil)

	msg, err := nm.RegisterOrganization(nm.ctx, &fftypes.Organization{
		Name:        "org1",
		Identity:    "0x12345",
		Parent:      "0x23456",
		Description: "my organization",
	}, false)
	assert.NoError(t, err)
	assert.Equal(t, mockMsg, msg)

	mim.AssertExpectations(t)
}

func TestRegisterNodeOrganizationRootOk(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(config.OrgIdentityDeprecated, "0x12345")
	config.Set(config.OrgName, "org1")
	config.Set(config.OrgDescription, "my organization")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveSigningKey", nm.ctx, "0x12345").Return("0x12345", nil)
	mim.On("ResolveInputIdentity", nm.ctx, mock.MatchedBy(func(i *fftypes.Identity) bool { return i.Key == "0x12345" })).Return(nil)

	mockMsg := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)
	mbm.On("BroadcastRootOrgDefinition", nm.ctx, mock.Anything, mock.MatchedBy(func(i *fftypes.Identity) bool { return i.Key == "0x12345" }), fftypes.SystemTagDefineOrganization, true).Return(mockMsg, nil)

	org, msg, err := nm.RegisterNodeOrganization(nm.ctx, true)
	assert.NoError(t, err)
	assert.Equal(t, mockMsg, msg)
	assert.Equal(t, *mockMsg.Header.ID, *org.Message)

}

func TestRegisterNodeOrganizationMissingOrgKey(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveSigningKey", nm.ctx, "").Return("", nil)

	_, _, err := nm.RegisterNodeOrganization(nm.ctx, true)
	assert.Regexp(t, "FF10216", err)

}

func TestRegisterNodeOrganizationMissingName(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(config.OrgKey, "0x2345")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveSigningKey", nm.ctx, "0x2345").Return("0x2345", nil)

	_, _, err := nm.RegisterNodeOrganization(nm.ctx, true)
	assert.Regexp(t, "FF10216", err)

}

func TestRegisterOrganizationBadObject(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	_, err := nm.RegisterOrganization(nm.ctx, &fftypes.Organization{
		Name:        "org1",
		Description: string(make([]byte, 4097)),
	}, false)
	assert.Regexp(t, "FF10188", err)

}

func TestRegisterOrganizationBadIdentity(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputIdentity", nm.ctx, mock.MatchedBy(func(i *fftypes.Identity) bool { return i.Key == "wrongun" })).Return(fmt.Errorf("pop"))
	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", nm.ctx, "wrongun").Return(nil, nil)

	_, err := nm.RegisterOrganization(nm.ctx, &fftypes.Organization{
		Name:     "org1",
		Identity: "wrongun",
		Parent:   "ok",
	}, false)
	assert.Regexp(t, "pop", err)

}

func TestRegisterOrganizationBadParent(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputIdentity", nm.ctx, mock.MatchedBy(func(i *fftypes.Identity) bool { return i.Key == "0x12345" })).Return(nil)
	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", nm.ctx, "wrongun").Return(nil, nil)

	_, err := nm.RegisterOrganization(nm.ctx, &fftypes.Organization{
		Name:     "org1",
		Identity: "0x12345",
		Parent:   "wrongun",
	}, false)
	assert.Regexp(t, "FF10214", err)

}

func TestRegisterOrganizationParentLookupFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputIdentity", nm.ctx, mock.MatchedBy(func(i *fftypes.Identity) bool { return i.Key == "0x12345" })).Return(nil)
	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", nm.ctx, "0x23456").Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterOrganization(nm.ctx, &fftypes.Organization{
		Name:     "org1",
		Identity: "0x12345",
		Parent:   "0x23456",
	}, false)
	assert.Regexp(t, "pop", err)

}
