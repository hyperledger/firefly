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
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (em *eventManager) TokenPoolCreated(bi blockchain.Plugin, pool *blockchain.TokenPool) error {

	log.L(em.ctx).Infof("-> TokenPoolCreated PoolID=%s, Type=%v", pool.PoolID, pool.Type)
	defer func() {
		log.L(em.ctx).Infof("<- TokenPoolCreated PoolID=%s, Type=%v", pool.PoolID, pool.Type)
	}()

	return em.retry.Do(em.ctx, "persist token pool", func(attempt int) (bool, error) {
		p := &fftypes.TokenPool{
			PoolID:  pool.PoolID,
			Type:    pool.Type,
			BaseURI: pool.BaseURI,
		}
		err := em.database.UpsertTokenPool(em.ctx, p, true)
		return err != nil, err // retry indefinitely (until context closes)
	})
}
