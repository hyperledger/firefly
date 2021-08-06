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
	"github.com/hyperledger-labs/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger-labs/firefly/mocks/identitymocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRegisterNodeOk(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(config.NodeDescription, "Node 1")
	config.Set(config.OrgIdentity, "0x23456")

	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", nm.ctx, "0x23456").Return(&fftypes.Organization{
		Identity:    "0x23456",
		Description: "owning organization",
	}, nil)

	mii := nm.identity.(*identitymocks.Plugin)
	childID := &fftypes.Identity{OnChain: "0x12345"}
	parentID := &fftypes.Identity{OnChain: "0x23456"}
	mii.On("Resolve", nm.ctx, "0x12345").Return(childID, nil)
	mii.On("Resolve", nm.ctx, "0x23456").Return(parentID, nil)

	mdx := nm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("GetEndpointInfo", nm.ctx).Return("peer1", fftypes.JSONObject{"endpoint": "details"}, nil)

	mockMsg := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)
	mbm.On("BroadcastDefinition", nm.ctx, mock.Anything, parentID, fftypes.SystemTagDefineNode).Return(mockMsg, nil)

	msg, err := nm.RegisterNode(nm.ctx, false)
	assert.NoError(t, err)
	assert.Equal(t, mockMsg, msg)

}

func TestRegisterNodeMissingConfig(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(config.NodeDescription, nil)
	config.Set(config.NodeName, nil)
	config.Set(config.OrgIdentity, nil)

	_, err := nm.RegisterNode(nm.ctx, false)
	assert.Regexp(t, "FF10216", err)

}

func TestRegisterNodeBadParentID(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(config.NodeDescription, "Node 1")
	config.Set(config.NodeName, "node1")
	config.Set(config.OrgIdentity, "0x23456")

	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", nm.ctx, "0x23456").Return(&fftypes.Organization{
		Identity:    "0x23456",
		Description: "owning organization",
	}, nil)

	mii := nm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", nm.ctx, "0x23456").Return(nil, fmt.Errorf("pop"))

	mdx := nm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("GetEndpointInfo", nm.ctx).Return("peer1", fftypes.JSONObject{"endpoint": "details"}, nil)

	_, err := nm.RegisterNode(nm.ctx, false)
	assert.Regexp(t, "FF10215", err)

}

func TestRegisterNodeBadNodeID(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(config.NodeDescription, "Node 1")
	config.Set(config.NodeName, "node1")
	config.Set(config.OrgIdentity, "0x23456")

	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", nm.ctx, "0x23456").Return(&fftypes.Organization{
		Identity:    "0x23456",
		Description: "owning organization",
	}, nil)

	mii := nm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", nm.ctx, "0x23456").Return(nil, fmt.Errorf("pop"))

	mdx := nm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("GetEndpointInfo", nm.ctx).Return("peer1", fftypes.JSONObject{"endpoint": "details"}, nil)

	_, err := nm.RegisterNode(nm.ctx, false)
	assert.Regexp(t, "pop", err)

}

func TestRegisterNodeParentNotFound(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(config.NodeDescription, "Node 1")
	config.Set(config.NodeName, "node1")
	config.Set(config.OrgIdentity, "0x23456")

	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", nm.ctx, "0x23456").Return(nil, nil)

	mii := nm.identity.(*identitymocks.Plugin)
	childID := &fftypes.Identity{OnChain: "0x12345"}
	mii.On("Resolve", nm.ctx, "0x12345").Return(childID, nil)

	mdx := nm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("GetEndpointInfo", nm.ctx).Return("peer1", fftypes.JSONObject{"endpoint": "details"}, nil)

	_, err := nm.RegisterNode(nm.ctx, false)
	assert.Regexp(t, "FF10214", err)

}

func TestRegisterNodeParentBadNode(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(config.NodeDescription, string(make([]byte, 4097)))
	config.Set(config.NodeName, "node1")
	config.Set(config.OrgIdentity, "0x23456")

	mdx := nm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("GetEndpointInfo", nm.ctx).Return("peer1", fftypes.JSONObject{"endpoint": "details"}, nil)

	_, err := nm.RegisterNode(nm.ctx, false)
	assert.Regexp(t, "FF10188", err)

}

func TestRegisterNodeParentDXEndpointFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(config.NodeDescription, string(make([]byte, 4097)))
	config.Set(config.NodeName, "node1")
	config.Set(config.OrgIdentity, "0x23456")

	mdx := nm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("GetEndpointInfo", nm.ctx).Return("", nil, fmt.Errorf("pop"))

	_, err := nm.RegisterNode(nm.ctx, false)
	assert.Regexp(t, "pop", err)

}
