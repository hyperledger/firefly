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
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
)

func (em *eventManager) actionTerminate(bi blockchain.Plugin, event *blockchain.Event) error {
	return em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
		ns, err := em.database.GetNamespace(ctx, core.SystemNamespace)
		if err != nil {
			return err
		}
		if err := bi.TerminateContract(ctx, &ns.Contracts, event); err != nil {
			return err
		}
		return em.database.UpsertNamespace(ctx, ns, true)
	})
}

func (em *eventManager) BlockchainNetworkAction(bi blockchain.Plugin, action string, event *blockchain.Event, signingKey *core.VerifierRef) error {
	return em.retry.Do(em.ctx, "handle network action", func(attempt int) (retry bool, err error) {
		// TODO: verify signing identity

		if action == core.NetworkActionTerminate.String() {
			err = em.actionTerminate(bi, event)
		} else {
			log.L(em.ctx).Errorf("Ignoring unrecognized network action: %s", action)
			return false, nil
		}

		if err == nil {
			chainEvent := buildBlockchainEvent(core.SystemNamespace, nil, event, &core.BlockchainTransactionRef{
				BlockchainID: event.BlockchainTXID,
			})
			err = em.maybePersistBlockchainEvent(em.ctx, chainEvent)
		}
		return true, err
	})
}
