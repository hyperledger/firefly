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

	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/pkg/config"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/log"
)

func (or *orchestrator) getPlugins() fftypes.NodeStatusPlugins {
	// Tokens can have more than one name, so they must be iterated over
	tokensArray := make([]*fftypes.NodeStatusPlugin, 0)
	for name, plugin := range or.tokens {
		tokensArray = append(tokensArray, &fftypes.NodeStatusPlugin{
			Name:       name,
			PluginType: plugin.Name(),
		})
	}

	return fftypes.NodeStatusPlugins{
		Blockchain: []*fftypes.NodeStatusPlugin{
			{
				PluginType: or.blockchain.Name(),
			},
		},
		Database: []*fftypes.NodeStatusPlugin{
			{
				PluginType: or.database.Name(),
			},
		},
		DataExchange: []*fftypes.NodeStatusPlugin{
			{
				PluginType: or.dataexchange.Name(),
			},
		},
		Events: or.events.GetPlugins(),
		Identity: []*fftypes.NodeStatusPlugin{
			{
				PluginType: or.identityPlugin.Name(),
			},
		},
		SharedStorage: []*fftypes.NodeStatusPlugin{
			{
				PluginType: or.sharedstorage.Name(),
			},
		},
		Tokens: tokensArray,
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

func (or *orchestrator) GetStatus(ctx context.Context) (status *fftypes.NodeStatus, err error) {

	org, err := or.identity.GetNodeOwnerOrg(ctx)
	if err != nil {
		log.L(ctx).Warnf("Failed to query local org for status: %s", err)
	}
	status = &fftypes.NodeStatus{
		Node: fftypes.NodeStatusNode{
			Name: config.GetString(coreconfig.NodeName),
		},
		Org: fftypes.NodeStatusOrg{
			Name: config.GetString(coreconfig.OrgName),
		},
		Defaults: fftypes.NodeStatusDefaults{
			Namespace: config.GetString(coreconfig.NamespacesDefault),
		},
		Plugins: or.getPlugins(),
	}

	if org != nil {
		status.Org.Registered = true
		status.Org.ID = org.ID
		status.Org.DID = org.DID
		verifiers, _, err := or.networkmap.GetIdentityVerifiers(ctx, fftypes.SystemNamespace, org.ID.String(), database.VerifierQueryFactory.NewFilter(ctx).And())
		if err != nil {
			return nil, err
		}
		status.Org.Verifiers = make([]*fftypes.VerifierRef, len(verifiers))
		for i, v := range verifiers {
			status.Org.Verifiers[i] = &v.VerifierRef
		}

		node, _, err := or.identity.CachedIdentityLookupNilOK(ctx, fmt.Sprintf("%s%s", fftypes.FireFlyNodeDIDPrefix, status.Node.Name))
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
