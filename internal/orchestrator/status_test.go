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

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/mocks/eventmocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/networkmapmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	pluginsResult = core.NodeStatusPlugins{
		Blockchain: []*core.NodeStatusPlugin{
			{
				Name:       "ethereum",
				PluginType: "mock-bi",
			},
		},
		Database: []*core.NodeStatusPlugin{
			{
				PluginType: "mock-di",
			},
		},
		DataExchange: []*core.NodeStatusPlugin{
			{
				PluginType: "mock-dx",
			},
		},
		Events: []*core.NodeStatusPlugin{
			{
				PluginType: "mock-ei",
			},
		},
		Identity: []*core.NodeStatusPlugin{
			{
				PluginType: "mock-ii",
			},
		},
		SharedStorage: []*core.NodeStatusPlugin{
			{
				PluginType: "mock-ps",
			},
		},
		Tokens: []*core.NodeStatusPlugin{
			{
				Name:       "token",
				PluginType: "mock-tk",
			},
		},
	}

	mockEventPlugins = []*core.NodeStatusPlugin{
		{
			PluginType: "mock-ei",
		},
	}
)

func TestGetStatusRegistered(t *testing.T) {
	or := newTestOrchestrator()

	coreconfig.Reset()
	config.Set(coreconfig.NamespacesDefault, "default")
	config.Set(coreconfig.OrgName, "org1")
	config.Set(coreconfig.NodeName, "node1")

	orgID := fftypes.NewUUID()
	nodeID := fftypes.NewUUID()

	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", or.ctx).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:   orgID,
			Name: "org1",
			DID:  "did:firefly:org/org1",
		},
	}, nil)
	mim.On("CachedIdentityLookupNilOK", or.ctx, "did:firefly:node/node1").Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:     nodeID,
			Name:   "node1",
			Parent: orgID,
		},
	}, false, nil)
	nmn := or.networkmap.(*networkmapmocks.Manager)
	nmn.On("GetIdentityVerifiers", or.ctx, core.SystemNamespace, orgID.String(), mock.Anything).Return([]*core.Verifier{
		{Hash: fftypes.NewRandB32(), VerifierRef: core.VerifierRef{
			Type:  core.VerifierTypeEthAddress,
			Value: "0x12345",
		}},
	}, nil, nil)

	mem := or.events.(*eventmocks.EventManager)
	mem.On("GetPlugins").Return(mockEventPlugins)

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
	assert.Equal(t, "0x12345", status.Org.Verifiers[0].Value)

	// Plugins
	assert.ElementsMatch(t, pluginsResult.Blockchain, status.Plugins.Blockchain)
	assert.ElementsMatch(t, pluginsResult.Database, status.Plugins.Database)
	assert.ElementsMatch(t, pluginsResult.DataExchange, status.Plugins.DataExchange)
	assert.ElementsMatch(t, pluginsResult.Events, status.Plugins.Events)
	assert.ElementsMatch(t, pluginsResult.Identity, status.Plugins.Identity)
	assert.ElementsMatch(t, pluginsResult.SharedStorage, status.Plugins.SharedStorage)
	assert.ElementsMatch(t, pluginsResult.Tokens, status.Plugins.Tokens)

	assert.True(t, or.GetNodeUUID(or.ctx).Equals(nodeID))
	assert.True(t, or.GetNodeUUID(or.ctx).Equals(nodeID)) // cached

}

func TestGetStatusVerifierLookupFail(t *testing.T) {
	or := newTestOrchestrator()

	coreconfig.Reset()
	config.Set(coreconfig.NamespacesDefault, "default")
	config.Set(coreconfig.OrgName, "org1")
	config.Set(coreconfig.NodeName, "node1")

	orgID := fftypes.NewUUID()
	nodeID := fftypes.NewUUID()

	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", or.ctx).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:   orgID,
			Name: "org1",
			DID:  "did:firefly:org/org1",
		},
	}, nil)
	mim.On("CachedIdentityLookupNilOK", or.ctx, "did:firefly:node/node1").Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:     nodeID,
			Name:   "node1",
			Parent: fftypes.NewUUID(),
		},
	}, false, nil)
	nmn := or.networkmap.(*networkmapmocks.Manager)
	nmn.On("GetIdentityVerifiers", or.ctx, core.SystemNamespace, orgID.String(), mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	mem := or.events.(*eventmocks.EventManager)
	mem.On("GetPlugins").Return(mockEventPlugins)

	_, err := or.GetStatus(or.ctx)
	assert.Regexp(t, "pop", err)

}

func TestGetStatusWrongNodeOwner(t *testing.T) {
	or := newTestOrchestrator()

	coreconfig.Reset()
	config.Set(coreconfig.NamespacesDefault, "default")
	config.Set(coreconfig.OrgName, "org1")
	config.Set(coreconfig.NodeName, "node1")

	orgID := fftypes.NewUUID()
	nodeID := fftypes.NewUUID()

	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", or.ctx).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:   orgID,
			Name: "org1",
			DID:  "did:firefly:org/org1",
		},
	}, nil)
	mim.On("CachedIdentityLookupNilOK", or.ctx, "did:firefly:node/node1").Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:     nodeID,
			Name:   "node1",
			Parent: fftypes.NewUUID(),
		},
	}, false, nil)
	nmn := or.networkmap.(*networkmapmocks.Manager)
	nmn.On("GetIdentityVerifiers", or.ctx, core.SystemNamespace, orgID.String(), mock.Anything).Return([]*core.Verifier{
		{Hash: fftypes.NewRandB32(), VerifierRef: core.VerifierRef{
			Type:  core.VerifierTypeEthAddress,
			Value: "0x12345",
		}},
	}, nil, nil)

	mem := or.events.(*eventmocks.EventManager)
	mem.On("GetPlugins").Return(mockEventPlugins)

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

	coreconfig.Reset()
	config.Set(coreconfig.NamespacesDefault, "default")
	config.Set(coreconfig.OrgName, "org1")
	config.Set(coreconfig.NodeName, "node1")

	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", or.ctx).Return(nil, fmt.Errorf("pop"))

	mem := or.events.(*eventmocks.EventManager)
	mem.On("GetPlugins").Return(mockEventPlugins)

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

	coreconfig.Reset()
	config.Set(coreconfig.NamespacesDefault, "default")
	config.Set(coreconfig.OrgName, "org1")
	config.Set(coreconfig.NodeName, "node1")

	orgID := fftypes.NewUUID()

	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", or.ctx).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:   orgID,
			Name: "org1",
			DID:  "did:firefly:org/org1",
		},
	}, nil)
	mim.On("CachedIdentityLookupNilOK", or.ctx, "did:firefly:node/node1").Return(nil, false, nil)
	nmn := or.networkmap.(*networkmapmocks.Manager)
	nmn.On("GetIdentityVerifiers", or.ctx, core.SystemNamespace, orgID.String(), mock.Anything).Return([]*core.Verifier{
		{Hash: fftypes.NewRandB32(), VerifierRef: core.VerifierRef{
			Type:  core.VerifierTypeEthAddress,
			Value: "0x12345",
		}},
	}, nil, nil)

	mem := or.events.(*eventmocks.EventManager)
	mem.On("GetPlugins").Return(mockEventPlugins)

	status, err := or.GetStatus(or.ctx)
	assert.NoError(t, err)

	assert.Equal(t, "default", status.Defaults.Namespace)

	assert.Equal(t, "org1", status.Org.Name)
	assert.True(t, status.Org.Registered)
	assert.Equal(t, "did:firefly:org/org1", status.Org.DID)
	assert.Equal(t, *orgID, *status.Org.ID)

	assert.Equal(t, "node1", status.Node.Name)
	assert.False(t, status.Node.Registered)

	// Plugins
	assert.ElementsMatch(t, pluginsResult.Blockchain, status.Plugins.Blockchain)
	assert.ElementsMatch(t, pluginsResult.Database, status.Plugins.Database)
	assert.ElementsMatch(t, pluginsResult.DataExchange, status.Plugins.DataExchange)
	assert.ElementsMatch(t, pluginsResult.Events, status.Plugins.Events)
	assert.ElementsMatch(t, pluginsResult.Identity, status.Plugins.Identity)
	assert.ElementsMatch(t, pluginsResult.SharedStorage, status.Plugins.SharedStorage)
	assert.ElementsMatch(t, pluginsResult.Tokens, status.Plugins.Tokens)

	assert.Nil(t, or.GetNodeUUID(or.ctx))
}

func TestGetStatusNodeError(t *testing.T) {
	or := newTestOrchestrator()

	coreconfig.Reset()
	config.Set(coreconfig.NamespacesDefault, "default")
	config.Set(coreconfig.OrgName, "org1")
	config.Set(coreconfig.NodeName, "node1")

	orgID := fftypes.NewUUID()

	mim := or.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", or.ctx).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:   orgID,
			Name: "org1",
			DID:  "did:firefly:org/org1",
		},
	}, nil)
	mim.On("CachedIdentityLookupNilOK", or.ctx, "did:firefly:node/node1").Return(nil, false, fmt.Errorf("pop"))
	nmn := or.networkmap.(*networkmapmocks.Manager)
	nmn.On("GetIdentityVerifiers", or.ctx, core.SystemNamespace, orgID.String(), mock.Anything).Return([]*core.Verifier{
		{Hash: fftypes.NewRandB32(), VerifierRef: core.VerifierRef{
			Type:  core.VerifierTypeEthAddress,
			Value: "0x12345",
		}},
	}, nil, nil)

	mem := or.events.(*eventmocks.EventManager)
	mem.On("GetPlugins").Return(mockEventPlugins)

	_, err := or.GetStatus(or.ctx)
	assert.EqualError(t, err, "pop")

	assert.Nil(t, or.GetNodeUUID(or.ctx))

}
