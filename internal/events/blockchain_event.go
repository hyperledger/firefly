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

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func buildBlockchainEvent(ns string, subID *fftypes.UUID, event *blockchain.Event, tx *fftypes.TransactionRef) *fftypes.BlockchainEvent {
	ev := &fftypes.BlockchainEvent{
		ID:           fftypes.NewUUID(),
		Namespace:    ns,
		Subscription: subID,
		Source:       event.Source,
		ProtocolID:   event.ProtocolID,
		Name:         event.Name,
		Output:       event.Output,
		Info:         event.Info,
		Timestamp:    event.Timestamp,
	}
	if tx != nil {
		ev.TX = *tx
	}
	return ev
}

func (em *eventManager) persistBlockchainEvent(ctx context.Context, chainEvent *fftypes.BlockchainEvent) error {
	if err := em.database.InsertBlockchainEvent(ctx, chainEvent); err != nil {
		return err
	}
	ffEvent := fftypes.NewEvent(fftypes.EventTypeBlockchainEvent, chainEvent.Namespace, chainEvent.ID)
	if err := em.database.InsertEvent(ctx, ffEvent); err != nil {
		return err
	}
	return nil
}

func (em *eventManager) BlockchainEvent(event *blockchain.EventWithContext) error {
	return em.retry.Do(em.ctx, "persist contract event", func(attempt int) (bool, error) {
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			// TODO: should cache this lookup for efficiency
			sub, err := em.database.GetContractSubscriptionByProtocolID(ctx, event.Subscription)
			if err != nil {
				return err
			}
			if sub == nil {
				log.L(ctx).Warnf("Event received from unknown subscription %s", event.Subscription)
				return nil // no retry
			}

			chainEvent := buildBlockchainEvent(sub.Namespace, sub.ID, &event.Event, nil)
			if err := em.persistBlockchainEvent(ctx, chainEvent); err != nil {
				return err
			}
			return nil
		})
		return err != nil, err
	})
}
