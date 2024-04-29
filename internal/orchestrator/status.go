// Copyright © 2023 Kaleido, Inc.
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
	"database/sql/driver"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

func (or *orchestrator) getPlugins() core.NamespaceStatusPlugins {
	// Plugins can have more than one name, so they must be iterated over
	tokensArray := make([]*core.NamespaceStatusPlugin, 0)
	for _, plugin := range or.plugins.Tokens {
		tokensArray = append(tokensArray, &core.NamespaceStatusPlugin{
			Name:       plugin.Name,
			PluginType: plugin.Plugin.Name(),
		})
	}

	blockchainsArray := make([]*core.NamespaceStatusPlugin, 0)
	if or.plugins.Blockchain.Plugin != nil {
		blockchainsArray = append(blockchainsArray, &core.NamespaceStatusPlugin{
			Name:       or.plugins.Blockchain.Name,
			PluginType: or.plugins.Blockchain.Plugin.Name(),
		})
	}

	databasesArray := make([]*core.NamespaceStatusPlugin, 0)
	if or.plugins.Database.Plugin != nil {
		databasesArray = append(databasesArray, &core.NamespaceStatusPlugin{
			Name:       or.plugins.Database.Name,
			PluginType: or.plugins.Database.Plugin.Name(),
		})
	}

	sharedstorageArray := make([]*core.NamespaceStatusPlugin, 0)
	if or.plugins.SharedStorage.Plugin != nil {
		sharedstorageArray = append(sharedstorageArray, &core.NamespaceStatusPlugin{
			Name:       or.plugins.SharedStorage.Name,
			PluginType: or.plugins.SharedStorage.Plugin.Name(),
		})
	}

	dataexchangeArray := make([]*core.NamespaceStatusPlugin, 0)
	if or.plugins.DataExchange.Plugin != nil {
		dataexchangeArray = append(dataexchangeArray, &core.NamespaceStatusPlugin{
			Name:       or.plugins.DataExchange.Name,
			PluginType: or.plugins.DataExchange.Plugin.Name(),
		})
	}

	return core.NamespaceStatusPlugins{
		Blockchain:    blockchainsArray,
		Database:      databasesArray,
		SharedStorage: sharedstorageArray,
		DataExchange:  dataexchangeArray,
		Events:        or.events.GetPlugins(),
		Tokens:        tokensArray,
		Identity:      []*core.NamespaceStatusPlugin{},
	}
}

func (or *orchestrator) GetStatus(ctx context.Context) (status *core.NamespaceStatus, err error) {

	status = &core.NamespaceStatus{
		Namespace: or.namespace,
		Plugins:   or.getPlugins(),
		Multiparty: core.NamespaceStatusMultiparty{
			Enabled: or.config.Multiparty.Enabled,
		},
	}

	if or.config.Multiparty.Enabled {
		status.Node = &core.NamespaceStatusNode{Name: or.config.Multiparty.Node.Name}
		status.Org = &core.NamespaceStatusOrg{Name: or.config.Multiparty.Org.Name}
		status.Multiparty.Contracts = or.namespace.Contracts

		org, err := or.identity.GetRootOrg(ctx)
		if err != nil {
			log.L(ctx).Warnf("Failed to query local org for status: %s", err)
		}

		if org != nil {
			status.Org.Registered = true
			status.Org.ID = org.ID
			status.Org.DID = org.DID

			fb := database.VerifierQueryFactory.NewFilter(ctx)
			verifiers, _, err := or.database().GetVerifiers(ctx, org.Namespace, fb.And(fb.Eq("identity", org.ID)))
			if err != nil {
				return nil, err
			}
			status.Org.Verifiers = make([]*core.VerifierRef, len(verifiers))
			for i, v := range verifiers {
				status.Org.Verifiers[i] = &v.VerifierRef
			}

			node, err := or.identity.GetLocalNode(ctx)
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
	}

	return status, nil
}

// TODO can we assume that the first message here is the earliest pending registration attempt?
// or do we need to dig into the message data to validate more fields?
func (or *orchestrator) getRegistrationMessage(ctx context.Context) (msg *core.Message, err error) {
	fb := database.MessageQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("type", core.MessageTypeDefinition),
		fb.Eq("tag", core.SystemTagIdentityClaim),
		fb.Eq("author", core.FireFlyOrgDIDPrefix+or.config.Multiparty.Org.Name),
		fb.In("state", []driver.Value{core.MessageStateStaged, core.MessageStateReady, core.MessageStateSent, core.MessageStatePending}),
	).Sort("created").Limit(1)
	msgs, _, err := or.database().GetMessages(ctx, or.namespace.Name, filter)
	if err != nil {
		return nil, err
	}
	if len(msgs) > 0 {
		return msgs[0], nil
	}
	return nil, nil
}

func (or *orchestrator) GetMultipartyStatus(ctx context.Context) (mpStatus *core.NamespaceMultipartyStatus, err error) {

	if !or.config.Multiparty.Enabled {
		return nil, i18n.NewError(ctx, coremsgs.MsgMultipartyNotEnabled)
	}

	status, err := or.GetStatus(ctx)
	if err != nil {
		return nil, err
	}

	mpStatus = &core.NamespaceMultipartyStatus{
		Org:  core.NamespaceMultipartyStatusOrg{},
		Node: core.NamespaceMultipartyStatusNode{},
	}

	if status.Org.Registered {
		mpStatus.Org.Status = core.NamespaceRegistrationStatusRegistered
		if status.Node.Registered {
			mpStatus.Node.Status = core.NamespaceRegistrationStatusRegistered
		} else {
			msg, err := or.getRegistrationMessage(ctx)
			if err != nil {
				return nil, err
			}
			mpStatus.Node.Status = core.NamespaceRegistrationStatusUnregistered
			if msg != nil {
				mpStatus.Node.RegistrationMessageID = msg.Header.ID
				mpStatus.Node.Status = core.NamespaceRegistrationStatusRegistering
			}
		}
	} else {
		msg, err := or.getRegistrationMessage(ctx)
		if err != nil {
			return nil, err
		}
		mpStatus.Org.Status = core.NamespaceRegistrationStatusUnregistered
		if msg != nil {
			mpStatus.Org.RegistrationMessageID = msg.Header.ID
			mpStatus.Org.Status = core.NamespaceRegistrationStatusRegistering
		}
		mpStatus.Node.Status = core.NamespaceRegistrationStatusUnregistered
	}

	return mpStatus, nil
}
