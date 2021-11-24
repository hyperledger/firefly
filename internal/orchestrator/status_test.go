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

package orchestrator

import (
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetStatusRegistered(t *testing.T) {
	or := newTestOrchestrator()

	config.Reset()
	config.Set(config.NamespacesDefault, "default")
	config.Set(config.OrgName, "org1")
	config.Set(config.NodeName, "node1")

	orgID := fftypes.NewUUID()
	nodeID := fftypes.NewUUID()

	mdi := or.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", or.ctx, "org1").Return(&fftypes.Organization{
		ID:       orgID,
		Identity: "0x1111111",
		Name:     "org1",
	}, nil)
	mdi.On("GetNode", or.ctx, "0x1111111", "node1").Return(&fftypes.Node{
		ID:    nodeID,
		Name:  "node1",
		Owner: "0x1111111",
	}, nil)
	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetOrgKey", mock.Anything).Return("0x1111111")

	status, err := or.GetStatus(or.ctx)
	assert.NoError(t, err)

	assert.Equal(t, "default", status.Defaults.Namespace)

	assert.Equal(t, "org1", status.Org.Name)
	assert.True(t, status.Org.Registered)
	assert.Equal(t, "0x1111111", status.Org.Identity)

	assert.Equal(t, *orgID, *status.Org.ID)
	assert.Equal(t, "node1", status.Node.Name)
	assert.True(t, status.Node.Registered)
	assert.Equal(t, *nodeID, *status.Node.ID)

	assert.True(t, or.GetNodeUUID(or.ctx).Equals(nodeID))
	assert.True(t, or.GetNodeUUID(or.ctx).Equals(nodeID)) // cached

}

func TestGetStatusUnregistered(t *testing.T) {
	or := newTestOrchestrator()

	config.Reset()
	config.Set(config.NamespacesDefault, "default")
	config.Set(config.OrgName, "org1")
	config.Set(config.NodeName, "node1")

	mdi := or.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", or.ctx, "org1").Return(nil, nil)
	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetOrgKey", mock.Anything).Return("0x1111111")

	status, err := or.GetStatus(or.ctx)
	assert.NoError(t, err)

	assert.Equal(t, "default", status.Defaults.Namespace)

	assert.Equal(t, "org1", status.Org.Name)
	assert.False(t, status.Org.Registered)

	assert.Equal(t, "node1", status.Node.Name)
	assert.False(t, status.Node.Registered)

	assert.Nil(t, or.GetNodeUUID(or.ctx))

}

func TestGetStatusOrgOnlyRegistered(t *testing.T) {
	or := newTestOrchestrator()

	config.Reset()
	config.Set(config.NamespacesDefault, "default")
	config.Set(config.OrgName, "org1")
	config.Set(config.NodeName, "node1")

	orgID := fftypes.NewUUID()

	mdi := or.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", or.ctx, "org1").Return(&fftypes.Organization{
		ID:       orgID,
		Identity: "0x1111111",
		Name:     "org1",
	}, nil)
	mdi.On("GetNode", or.ctx, "0x1111111", "node1").Return(nil, nil)
	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetOrgKey", mock.Anything).Return("0x1111111")

	status, err := or.GetStatus(or.ctx)
	assert.NoError(t, err)

	assert.Equal(t, "default", status.Defaults.Namespace)

	assert.Equal(t, "org1", status.Org.Name)
	assert.True(t, status.Org.Registered)
	assert.Equal(t, "0x1111111", status.Org.Identity)
	assert.Equal(t, *orgID, *status.Org.ID)

	assert.Equal(t, "node1", status.Node.Name)
	assert.False(t, status.Node.Registered)

	assert.Nil(t, or.GetNodeUUID(or.ctx))
}

func TestGetStatuOrgError(t *testing.T) {
	or := newTestOrchestrator()

	config.Reset()
	config.Set(config.NamespacesDefault, "default")
	config.Set(config.OrgName, "org1")
	config.Set(config.NodeName, "node1")

	mdi := or.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", or.ctx, "org1").Return(nil, fmt.Errorf("pop"))
	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetOrgKey", mock.Anything).Return("0x1111111")

	_, err := or.GetStatus(or.ctx)
	assert.EqualError(t, err, "pop")
}

func TestGetStatusNodeError(t *testing.T) {
	or := newTestOrchestrator()

	config.Reset()
	config.Set(config.NamespacesDefault, "default")
	config.Set(config.OrgName, "org1")
	config.Set(config.NodeName, "node1")

	orgID := fftypes.NewUUID()

	mdi := or.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", or.ctx, "org1").Return(&fftypes.Organization{
		ID:       orgID,
		Identity: "0x1111111",
		Name:     "org1",
	}, nil)
	mdi.On("GetNode", or.ctx, "0x1111111", "node1").Return(nil, fmt.Errorf("pop"))
	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetOrgKey", mock.Anything).Return("0x1111111")

	_, err := or.GetStatus(or.ctx)
	assert.EqualError(t, err, "pop")

	assert.Nil(t, or.GetNodeUUID(or.ctx))

}
