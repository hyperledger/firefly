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
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	pluginsResult = core.NamespaceStatusPlugins{
		Blockchain: []*core.NamespaceStatusPlugin{
			{
				PluginType: "mock-bi",
			},
		},
		Database: []*core.NamespaceStatusPlugin{
			{
				PluginType: "mock-di",
			},
		},
		DataExchange: []*core.NamespaceStatusPlugin{
			{
				PluginType: "mock-dx",
			},
		},
		Events: []*core.NamespaceStatusPlugin{
			{
				PluginType: "mock-ei",
			},
		},
		Identity: []*core.NamespaceStatusPlugin{
			{
				PluginType: "mock-ii",
			},
		},
		SharedStorage: []*core.NamespaceStatusPlugin{
			{
				PluginType: "mock-ps",
			},
		},
		Tokens: []*core.NamespaceStatusPlugin{
			{
				Name:       "token",
				PluginType: "mock-tk",
			},
		},
	}

	mockEventPlugins = []*core.NamespaceStatusPlugin{
		{
			PluginType: "mock-ei",
		},
	}
)

func TestGetStatusRegistered(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)

	coreconfig.Reset()
	config.Set(coreconfig.NamespacesDefault, "default")

	orgID := fftypes.NewUUID()
	nodeID := fftypes.NewUUID()

	or.mim.On("GetMultipartyRootOrg", or.ctx).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:        orgID,
			Name:      "org1",
			Namespace: "ns",
			DID:       "did:firefly:org/org1",
		},
	}, nil)
	or.mim.On("GetLocalNode", or.ctx).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:     nodeID,
			Name:   "node1",
			Parent: orgID,
		},
	}, nil)
	or.mdi.On("GetVerifiers", or.ctx, "ns", mock.Anything).Return([]*core.Verifier{
		{Hash: fftypes.NewRandB32(), VerifierRef: core.VerifierRef{
			Type:  core.VerifierTypeEthAddress,
			Value: "0x12345",
		}},
	}, nil, nil)

	or.config.Multiparty.Org.Name = "org1"
	or.config.Multiparty.Node.Name = "node1"

	or.mem.On("GetPlugins").Return(mockEventPlugins)

	status, err := or.GetStatus(or.ctx)
	assert.NoError(t, err)

	assert.Equal(t, "ns", status.Namespace.Name)

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
	assert.ElementsMatch(t, pluginsResult.SharedStorage, status.Plugins.SharedStorage)
	assert.ElementsMatch(t, pluginsResult.Tokens, status.Plugins.Tokens)

}

func TestGetStatusVerifierLookupFail(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)

	coreconfig.Reset()
	config.Set(coreconfig.NamespacesDefault, "default")

	orgID := fftypes.NewUUID()

	or.mim.On("GetMultipartyRootOrg", or.ctx).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:        orgID,
			Name:      "org1",
			Namespace: "ns",
			DID:       "did:firefly:org/org1",
		},
	}, nil)
	or.mdi.On("GetVerifiers", or.ctx, "ns", mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	or.mem.On("GetPlugins").Return(mockEventPlugins)

	_, err := or.GetStatus(or.ctx)
	assert.Regexp(t, "pop", err)

}

func TestGetStatusWrongNodeOwner(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)

	coreconfig.Reset()
	config.Set(coreconfig.NamespacesDefault, "default")

	orgID := fftypes.NewUUID()
	nodeID := fftypes.NewUUID()

	or.mim.On("GetMultipartyRootOrg", or.ctx).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:        orgID,
			Name:      "org1",
			Namespace: "ns",
			DID:       "did:firefly:org/org1",
		},
	}, nil)
	or.mim.On("GetLocalNode", or.ctx).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:     nodeID,
			Name:   "node1",
			Parent: fftypes.NewUUID(),
		},
	}, nil)
	or.mdi.On("GetVerifiers", or.ctx, "ns", mock.Anything).Return([]*core.Verifier{
		{Hash: fftypes.NewRandB32(), VerifierRef: core.VerifierRef{
			Type:  core.VerifierTypeEthAddress,
			Value: "0x12345",
		}},
	}, nil, nil)

	or.config.Multiparty.Org.Name = "org1"
	or.config.Multiparty.Node.Name = "node1"

	or.mem.On("GetPlugins").Return(mockEventPlugins)

	status, err := or.GetStatus(or.ctx)
	assert.NoError(t, err)

	assert.Equal(t, "ns", status.Namespace.Name)

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
	defer or.cleanup(t)

	coreconfig.Reset()
	config.Set(coreconfig.NamespacesDefault, "default")

	or.mim.On("GetMultipartyRootOrg", or.ctx).Return(nil, fmt.Errorf("pop"))

	or.config.Multiparty.Org.Name = "org1"
	or.config.Multiparty.Node.Name = "node1"

	or.mem.On("GetPlugins").Return(mockEventPlugins)

	status, err := or.GetStatus(or.ctx)
	assert.NoError(t, err)

	assert.Equal(t, "ns", status.Namespace.Name)

	assert.Equal(t, "org1", status.Org.Name)
	assert.False(t, status.Org.Registered)

	assert.Equal(t, "node1", status.Node.Name)
	assert.False(t, status.Node.Registered)

}

func TestGetStatusOrgOnlyRegistered(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)

	coreconfig.Reset()
	config.Set(coreconfig.NamespacesDefault, "default")

	orgID := fftypes.NewUUID()

	or.mim.On("GetMultipartyRootOrg", or.ctx).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:        orgID,
			Name:      "org1",
			Namespace: "ns",
			DID:       "did:firefly:org/org1",
		},
	}, nil)
	or.mim.On("GetLocalNode", or.ctx).Return(nil, nil)
	or.mdi.On("GetVerifiers", or.ctx, "ns", mock.Anything).Return([]*core.Verifier{
		{Hash: fftypes.NewRandB32(), VerifierRef: core.VerifierRef{
			Type:  core.VerifierTypeEthAddress,
			Value: "0x12345",
		}},
	}, nil, nil)

	or.config.Multiparty.Org.Name = "org1"
	or.config.Multiparty.Node.Name = "node1"

	or.mem.On("GetPlugins").Return(mockEventPlugins)

	status, err := or.GetStatus(or.ctx)
	assert.NoError(t, err)

	assert.Equal(t, "ns", status.Namespace.Name)

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
	assert.ElementsMatch(t, pluginsResult.SharedStorage, status.Plugins.SharedStorage)
	assert.ElementsMatch(t, pluginsResult.Tokens, status.Plugins.Tokens)
}

func TestGetStatusNodeError(t *testing.T) {
	or := newTestOrchestrator()
	defer or.cleanup(t)

	coreconfig.Reset()
	config.Set(coreconfig.NamespacesDefault, "default")

	orgID := fftypes.NewUUID()

	or.mim.On("GetMultipartyRootOrg", or.ctx).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:        orgID,
			Name:      "org1",
			Namespace: "ns",
			DID:       "did:firefly:org/org1",
		},
	}, nil)
	or.mim.On("GetLocalNode", or.ctx).Return(nil, fmt.Errorf("pop"))
	or.mdi.On("GetVerifiers", or.ctx, "ns", mock.Anything).Return([]*core.Verifier{
		{Hash: fftypes.NewRandB32(), VerifierRef: core.VerifierRef{
			Type:  core.VerifierTypeEthAddress,
			Value: "0x12345",
		}},
	}, nil, nil)

	or.config.Multiparty.Node.Name = "node1"

	or.mem.On("GetPlugins").Return(mockEventPlugins)

	_, err := or.GetStatus(or.ctx)
	assert.EqualError(t, err, "pop")

}
