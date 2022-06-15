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

package events

import (
	"context"

	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/multiparty"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
)

func (em *eventManager) actionTerminate(mm multiparty.Manager, event *blockchain.Event) error {
	namespace, err := em.database.GetNamespace(em.ctx, em.namespace)
	if err != nil {
		return err
	}
	if err := mm.TerminateContract(em.ctx, &namespace.Contracts, event); err != nil {
		return err
	}
	// Currently, a termination event is implied to apply to ALL namespaces
	return em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
		if err := em.database.UpsertNamespace(em.ctx, namespace, true); err != nil {
			return err
		}
		return nil
	})
}

func (em *eventManager) BlockchainNetworkAction(mm multiparty.Manager, action string, event *blockchain.Event, signingKey *core.VerifierRef) error {
	return em.retry.Do(em.ctx, "handle network action", func(attempt int) (retry bool, err error) {
		// Verify that the action came from a registered root org
		resolvedAuthor, err := em.identity.FindIdentityForVerifier(em.ctx, []core.IdentityType{core.IdentityTypeOrg}, signingKey)
		if err != nil {
			return true, err
		}
		if resolvedAuthor == nil {
			log.L(em.ctx).Errorf("Ignoring network action %s from unknown identity %s", action, signingKey.Value)
			return false, nil
		}
		if resolvedAuthor.Parent != nil {
			log.L(em.ctx).Errorf("Ignoring network action %s from non-root identity %s", action, signingKey.Value)
			return false, nil
		}

		if action == core.NetworkActionTerminate.String() {
			err = em.actionTerminate(mm, event)
		} else {
			log.L(em.ctx).Errorf("Ignoring unrecognized network action: %s", action)
			return false, nil
		}

		if err == nil {
			chainEvent := buildBlockchainEvent(core.LegacySystemNamespace, nil, event, &core.BlockchainTransactionRef{
				BlockchainID: event.BlockchainTXID,
			})
			err = em.maybePersistBlockchainEvent(em.ctx, chainEvent)
		}
		return true, err
	})
}
