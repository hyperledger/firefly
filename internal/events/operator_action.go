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
	"strconv"

	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
)

func (em *eventManager) actionMigrate(bi blockchain.Plugin, payload string) error {
	ns, err := em.database.GetNamespace(em.ctx, core.SystemNamespace)
	if err != nil {
		return err
	}
	idx, err := strconv.Atoi(payload)
	if err != nil {
		return err
	}
	if ns.ContractIndex == idx {
		log.L(em.ctx).Debugf("Ignoring namespace migration for %s (already at %d)", ns.Name, ns.ContractIndex)
		return nil
	}
	ns.ContractIndex = idx
	log.L(em.ctx).Infof("Migrating namespace %s to contract index %d", ns.Name, ns.ContractIndex)
	bi.Stop()
	if err := bi.Start(ns.ContractIndex); err != nil {
		return err
	}
	return em.database.UpsertNamespace(em.ctx, ns, true)
}

func (em *eventManager) BlockchainOperatorAction(bi blockchain.Plugin, action, payload string, signingKey *core.VerifierRef) error {
	return em.retry.Do(em.ctx, "handle operator action", func(attempt int) (bool, error) {
		// TODO: verify signing identity
		if action == blockchain.OperatorActionMigrate {
			return true, em.actionMigrate(bi, payload)
		}
		log.L(em.ctx).Errorf("Ignoring unrecognized operator action: %s", action)
		return false, nil
	})
}
