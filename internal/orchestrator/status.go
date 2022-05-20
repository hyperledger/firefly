// Copyright © 2022 Kaleido, Inc.
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
	"context"
	"fmt"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

func (or *orchestrator) getPlugins() core.NodeStatusPlugins {
	// Plugins can have more than one name, so they must be iterated over
	tokensArray := make([]*core.NodeStatusPlugin, 0)
	for name, plugin := range or.tokens {
		tokensArray = append(tokensArray, &core.NodeStatusPlugin{
			Name:       name,
			PluginType: plugin.Name(),
		})
	}

	blockchainsArray := make([]*core.NodeStatusPlugin, 0)
	for name, plugin := range or.blockchains {
		blockchainsArray = append(blockchainsArray, &core.NodeStatusPlugin{
			Name:       name,
			PluginType: plugin.Name(),
		})
	}

	databasesArray := make([]*core.NodeStatusPlugin, 0)
	for name, plugin := range or.databases {
		databasesArray = append(databasesArray, &core.NodeStatusPlugin{
			Name:       name,
			PluginType: plugin.Name(),
		})
	}

	sharedstorageArray := make([]*core.NodeStatusPlugin, 0)
	for name, plugin := range or.sharedstoragePlugins {
		sharedstorageArray = append(sharedstorageArray, &core.NodeStatusPlugin{
			Name:       name,
			PluginType: plugin.Name(),
		})
	}

	dataexchangeArray := make([]*core.NodeStatusPlugin, 0)
	for name, plugin := range or.dataexchangePlugins {
		dataexchangeArray = append(dataexchangeArray, &core.NodeStatusPlugin{
			Name:       name,
			PluginType: plugin.Name(),
		})
	}

	identityPluginArray := make([]*core.NodeStatusPlugin, 0)
	for name, plugin := range or.identityPlugins {
		identityPluginArray = append(identityPluginArray, &core.NodeStatusPlugin{
			Name:       name,
			PluginType: plugin.Name(),
		})
	}

	return core.NodeStatusPlugins{
		Blockchain:    blockchainsArray,
		Database:      databasesArray,
		SharedStorage: sharedstorageArray,
		DataExchange:  dataexchangeArray,
		Events:        or.events.GetPlugins(),
		Identity:      identityPluginArray,
		Tokens:        tokensArray,
	}
}

func (or *orchestrator) GetNodeUUID(ctx context.Context) (node *fftypes.UUID) {
	if or.node != nil {
		return or.node
	}
	status, err := or.GetStatus(ctx)
	if err != nil {
		log.L(or.ctx).Warnf("Failed to query local node UUID: %s", err)
		return nil
	}
	if status.Node.Registered {
		or.node = status.Node.ID
	} else {
		log.L(or.ctx).Infof("Node not yet registered")
	}
	return or.node
}

func (or *orchestrator) GetStatus(ctx context.Context) (status *core.NodeStatus, err error) {

	org, err := or.identity.GetNodeOwnerOrg(ctx)
	if err != nil {
		log.L(ctx).Warnf("Failed to query local org for status: %s", err)
	}
	status = &core.NodeStatus{
		Node: core.NodeStatusNode{
			Name: config.GetString(coreconfig.NodeName),
		},
		Org: core.NodeStatusOrg{
			Name: config.GetString(coreconfig.OrgName),
		},
		Defaults: core.NodeStatusDefaults{
			Namespace: config.GetString(coreconfig.NamespacesDefault),
		},
		Plugins: or.getPlugins(),
	}

	if org != nil {
		status.Org.Registered = true
		status.Org.ID = org.ID
		status.Org.DID = org.DID
		verifiers, _, err := or.networkmap.GetIdentityVerifiers(ctx, core.SystemNamespace, org.ID.String(), database.VerifierQueryFactory.NewFilter(ctx).And())
		if err != nil {
			return nil, err
		}
		status.Org.Verifiers = make([]*core.VerifierRef, len(verifiers))
		for i, v := range verifiers {
			status.Org.Verifiers[i] = &v.VerifierRef
		}

		node, _, err := or.identity.CachedIdentityLookupNilOK(ctx, fmt.Sprintf("%s%s", core.FireFlyNodeDIDPrefix, status.Node.Name))
		if err != nil {
			return nil, err
		}
		if node != nil && !node.Parent.Equals(org.ID) {
			log.L(ctx).Errorf("Specified node name is in use by another org: %s", err)
			node = nil
		}
		if node != nil {
			status.Node.Registered = true
			status.Node.ID = node.ID
		}
	}

	return status, nil
}
