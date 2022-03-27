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
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestGetStatusRegistered(t *testing.T) {
	or := newTestOrchestrator()

	config.Reset()
	config.Set(config.NamespacesDefault, "default")
	config.Set(config.OrgName, "org1")
	config.Set(config.NodeName, "node1")

	orgID := fftypes.NewUUID()
	nodeID := fftypes.NewUUID()

	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", or.ctx).Return(&fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:   orgID,
			Name: "org1",
			DID:  "did:firefly:org/org1",
		},
	}, nil)
	mim.On("CachedIdentityLookupNilOK", or.ctx, "did:firefly:node/node1").Return(&fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:     nodeID,
			Name:   "node1",
			Parent: orgID,
		},
	}, false, nil)

	status, err := or.GetStatus(or.ctx)
	assert.NoError(t, err)

	assert.Equal(t, "default", status.Defaults.Namespace)

	assert.Equal(t, "org1", status.Org.Name)
	assert.True(t, status.Org.Registered)
	assert.Equal(t, "did:firefly:org/org1", status.Org.DID)

	assert.Equal(t, *orgID, *status.Org.ID)
	assert.Equal(t, "node1", status.Node.Name)
	assert.True(t, status.Node.Registered)
	assert.Equal(t, *nodeID, *status.Node.ID)

	assert.True(t, or.GetNodeUUID(or.ctx).Equals(nodeID))
	assert.True(t, or.GetNodeUUID(or.ctx).Equals(nodeID)) // cached

}

func TestGetStatusWrongNodeOwner(t *testing.T) {
	or := newTestOrchestrator()

	config.Reset()
	config.Set(config.NamespacesDefault, "default")
	config.Set(config.OrgName, "org1")
	config.Set(config.NodeName, "node1")

	orgID := fftypes.NewUUID()
	nodeID := fftypes.NewUUID()

	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", or.ctx).Return(&fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:   orgID,
			Name: "org1",
			DID:  "did:firefly:org/org1",
		},
	}, nil)
	mim.On("CachedIdentityLookupNilOK", or.ctx, "did:firefly:node/node1").Return(&fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:     nodeID,
			Name:   "node1",
			Parent: fftypes.NewUUID(),
		},
	}, false, nil)

	status, err := or.GetStatus(or.ctx)
	assert.NoError(t, err)

	assert.Equal(t, "default", status.Defaults.Namespace)

	assert.Equal(t, "org1", status.Org.Name)
	assert.True(t, status.Org.Registered)
	assert.Equal(t, "did:firefly:org/org1", status.Org.DID)

	assert.Equal(t, *orgID, *status.Org.ID)
	assert.Equal(t, "node1", status.Node.Name)
	assert.False(t, status.Node.Registered)
	assert.Nil(t, status.Node.ID)

}

func TestGetStatusUnregistered(t *testing.T) {
	or := newTestOrchestrator()

	config.Reset()
	config.Set(config.NamespacesDefault, "default")
	config.Set(config.OrgName, "org1")
	config.Set(config.NodeName, "node1")

	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", or.ctx).Return(nil, fmt.Errorf("pop"))

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

	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", or.ctx).Return(&fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:   orgID,
			Name: "org1",
			DID:  "did:firefly:org/org1",
		},
	}, nil)
	mim.On("CachedIdentityLookupNilOK", or.ctx, "did:firefly:node/node1").Return(nil, false, nil)

	status, err := or.GetStatus(or.ctx)
	assert.NoError(t, err)

	assert.Equal(t, "default", status.Defaults.Namespace)

	assert.Equal(t, "org1", status.Org.Name)
	assert.True(t, status.Org.Registered)
	assert.Equal(t, "did:firefly:org/org1", status.Org.DID)
	assert.Equal(t, *orgID, *status.Org.ID)

	assert.Equal(t, "node1", status.Node.Name)
	assert.False(t, status.Node.Registered)

	assert.Nil(t, or.GetNodeUUID(or.ctx))
}

func TestGetStatusNodeError(t *testing.T) {
	or := newTestOrchestrator()

	config.Reset()
	config.Set(config.NamespacesDefault, "default")
	config.Set(config.OrgName, "org1")
	config.Set(config.NodeName, "node1")

	orgID := fftypes.NewUUID()

	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", or.ctx).Return(&fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:   orgID,
			Name: "org1",
			DID:  "did:firefly:org/org1",
		},
	}, nil)
	mim.On("CachedIdentityLookupNilOK", or.ctx, "did:firefly:node/node1").Return(nil, false, fmt.Errorf("pop"))

	_, err := or.GetStatus(or.ctx)
	assert.EqualError(t, err, "pop")

	assert.Nil(t, or.GetNodeUUID(or.ctx))

}
