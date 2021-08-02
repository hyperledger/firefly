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

package events

import (
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/blockchain"
)

func (em *eventManager) TokenBalanceChanged(bi blockchain.Plugin, poolID string, identity string, amount int) error {

	log.L(em.ctx).Infof("-> TokenBalanceChanged PoolID=%s, Identity=%s", poolID, identity)
	defer func() {
		log.L(em.ctx).Infof("<- TokenBalanceChanged PoolID=%s, Identity=%s", poolID, identity)
	}()

	return em.retry.Do(em.ctx, "persist token account balance", func(attempt int) (bool, error) {
		err := em.database.UpdateTokenBalance(em.ctx, poolID, identity, amount)
		return err != nil, err // retry indefinitely (until context closes)
	})
}
