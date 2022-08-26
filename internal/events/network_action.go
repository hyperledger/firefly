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
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
)

func (em *eventManager) actionTerminate(location *fftypes.JSONAny, event *blockchain.Event) error {
	return em.multiparty.TerminateContract(em.ctx, location, event)
}

func (em *eventManager) BlockchainNetworkAction(action string, location *fftypes.JSONAny, event *blockchain.Event, signingKey *core.VerifierRef) error {
	if em.multiparty == nil {
		log.L(em.ctx).Errorf("Ignoring network action from non-multiparty network!")
		return nil
	}

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
			err = em.actionTerminate(location, event)
		} else {
			log.L(em.ctx).Errorf("Ignoring unrecognized network action: %s", action)
			return false, nil
		}

		if err == nil {
			chainEvent := buildBlockchainEvent(em.namespace.Name, nil, event, &core.BlockchainTransactionRef{
				BlockchainID: event.BlockchainTXID,
			})
			err = em.maybePersistBlockchainEvent(em.ctx, chainEvent, nil)
		}
		return true, err
	})
}
