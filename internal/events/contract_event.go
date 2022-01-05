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
	"context"

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (em *eventManager) ContractEvent(blockchainEvent *blockchain.ContractEvent) error {
	return em.retry.Do(em.ctx, "persist contract event", func(attempt int) (bool, error) {
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			// TODO: should cache this lookup for efficiency
			sub, err := em.database.GetContractSubscriptionByProtocolID(ctx, blockchainEvent.Subscription)
			if err != nil {
				return err
			}
			if sub == nil {
				log.L(ctx).Warnf("Event received from unknown subscription %s", blockchainEvent.Subscription)
				return nil // no retry
			}
			contractEvent := &fftypes.ContractEvent{
				ID:           fftypes.NewUUID(),
				Namespace:    sub.Namespace,
				Subscription: sub.ID,
				Name:         blockchainEvent.Name,
				Outputs:      blockchainEvent.Outputs,
				Info:         blockchainEvent.Info,
				Timestamp:    blockchainEvent.Timestamp,
			}
			if err = em.database.InsertContractEvent(ctx, contractEvent); err != nil {
				return err
			}
			event := fftypes.NewEvent(fftypes.EventTypeContractEvent, contractEvent.Namespace, contractEvent.ID)
			if err = em.database.InsertEvent(ctx, event); err != nil {
				return err
			}
			return nil
		})
		return err != nil, err
	})
}
