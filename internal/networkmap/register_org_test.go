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
	"github.com/hyperledger-labs/firefly/mocks/identitymocks"
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

	mii := nm.identity.(*identitymocks.Plugin)
	childID := &fftypes.Identity{OnChain: "0x12345"}
	parentID := &fftypes.Identity{OnChain: "0x23456"}
	mii.On("Resolve", nm.ctx, "0x12345").Return(childID, nil)
	mii.On("Resolve", nm.ctx, "0x23456").Return(parentID, nil)

	mockMsg := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)
	mbm.On("BroadcastDefinition", nm.ctx, mock.Anything, parentID, fftypes.SystemTagDefineOrganization).Return(mockMsg, nil)

	msg, err := nm.RegisterOrganization(nm.ctx, &fftypes.Organization{
		Name:        "org1",
		Identity:    "0x12345",
		Parent:      "0x23456",
		Description: "my organization",
	}, false)
	assert.NoError(t, err)
	assert.Equal(t, mockMsg, msg)

}

func TestRegisterNodeOrganizationRootOk(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(config.OrgName, "org1")
	config.Set(config.OrgIdentity, "0x12345")
	config.Set(config.OrgDescription, "my organization")

	mii := nm.identity.(*identitymocks.Plugin)
	rootID := &fftypes.Identity{OnChain: "0x12345"}
	mii.On("Resolve", nm.ctx, "0x12345").Return(rootID, nil)

	mockMsg := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)
	mbm.On("BroadcastDefinition", nm.ctx, mock.Anything, rootID, fftypes.SystemTagDefineOrganization).Return(mockMsg, nil)

	msg, err := nm.RegisterNodeOrganization(nm.ctx, false)
	assert.NoError(t, err)
	assert.Equal(t, mockMsg, msg)

}

func TestRegisterNodeOrganizationMissingConfig(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(config.OrgIdentity, nil)

	_, err := nm.RegisterNodeOrganization(nm.ctx, false)
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

	mii := nm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", nm.ctx, "!wrong").Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterOrganization(nm.ctx, &fftypes.Organization{
		Name:     "org1",
		Identity: "!wrong",
	}, false)
	assert.Regexp(t, "FF10215.*pop", err)

}

func TestRegisterOrganizationBadParent(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mii := nm.identity.(*identitymocks.Plugin)
	childID := &fftypes.Identity{OnChain: "0x12345"}
	mii.On("Resolve", nm.ctx, "0x12345").Return(childID, nil)
	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", nm.ctx, "wrongun").Return(nil, nil)

	_, err := nm.RegisterOrganization(nm.ctx, &fftypes.Organization{
		Name:     "org1",
		Identity: "0x12345",
		Parent:   "wrongun",
	}, false)
	assert.Regexp(t, "FF10214", err)

}

func TestRegisterOrganizationChildResolveFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mii := nm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", nm.ctx, "0x12345").Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterOrganization(nm.ctx, &fftypes.Organization{
		Name:     "org1",
		Identity: "0x12345",
		Parent:   "0x23456",
	}, false)
	assert.Regexp(t, "pop", err)

}

func TestRegisterOrganizationParentLookupFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mii := nm.identity.(*identitymocks.Plugin)
	childID := &fftypes.Identity{OnChain: "0x12345"}
	mii.On("Resolve", nm.ctx, "0x12345").Return(childID, nil)
	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", nm.ctx, "0x23456").Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterOrganization(nm.ctx, &fftypes.Organization{
		Name:     "org1",
		Identity: "0x12345",
		Parent:   "0x23456",
	}, false)
	assert.Regexp(t, "pop", err)

}
