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
	"fmt"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
)

func buildBlockchainEvent(ns string, subID *fftypes.UUID, event *blockchain.Event, tx *core.BlockchainTransactionRef) *core.BlockchainEvent {
	ev := &core.BlockchainEvent{
		ID:         fftypes.NewUUID(),
		Namespace:  ns,
		Listener:   subID,
		Source:     event.Source,
		ProtocolID: event.ProtocolID,
		Name:       event.Name,
		Output:     event.Output,
		Info:       event.Info,
		Timestamp:  event.Timestamp,
	}
	if tx != nil {
		ev.TX = *tx
	}
	return ev
}

func (em *eventManager) getChainListenerByProtocolIDCached(ctx context.Context, protocolID string) (*core.ContractListener, error) {
	return em.getChainListenerCached(fmt.Sprintf("pid:%s", protocolID), func() (*core.ContractListener, error) {
		return em.database.GetContractListenerByBackendID(ctx, protocolID)
	})
}

func (em *eventManager) getChainListenerByIDCached(ctx context.Context, id *fftypes.UUID) (*core.ContractListener, error) {
	return em.getChainListenerCached(fmt.Sprintf("id:%s", id), func() (*core.ContractListener, error) {
		return em.database.GetContractListenerByID(ctx, id)
	})
}

func (em *eventManager) getChainListenerCached(cacheKey string, getter func() (*core.ContractListener, error)) (*core.ContractListener, error) {
	cached := em.chainListenerCache.Get(cacheKey)
	if cached != nil {
		cached.Extend(em.chainListenerCacheTTL)
		return cached.Value().(*core.ContractListener), nil
	}
	listener, err := getter()
	if listener == nil || err != nil {
		return nil, err
	}
	em.chainListenerCache.Set(cacheKey, listener, em.chainListenerCacheTTL)
	return listener, err
}

func (em *eventManager) getTopicForChainListener(ctx context.Context, listenerID *fftypes.UUID) (string, error) {
	if listenerID == nil {
		return core.SystemBatchPinTopic, nil
	}
	listener, err := em.getChainListenerByIDCached(ctx, listenerID)
	if err != nil {
		return "", err
	}
	var topic string
	if listener != nil && listener.Topic != "" {
		topic = listener.Topic
	} else {
		topic = listenerID.String()
	}
	return topic, nil
}

func (em *eventManager) maybePersistBlockchainEvent(ctx context.Context, chainEvent *core.BlockchainEvent) error {
	if existing, err := em.database.GetBlockchainEventByProtocolID(ctx, chainEvent.Namespace, chainEvent.Listener, chainEvent.ProtocolID); err != nil {
		return err
	} else if existing != nil {
		log.L(ctx).Debugf("Ignoring duplicate blockchain event %s", chainEvent.ProtocolID)
		// Return the ID of the existing event
		chainEvent.ID = existing.ID
		return nil
	}
	if err := em.txHelper.InsertBlockchainEvent(ctx, chainEvent); err != nil {
		return err
	}
	topic, err := em.getTopicForChainListener(ctx, chainEvent.Listener)
	if err != nil {
		return err
	}
	ffEvent := core.NewEvent(core.EventTypeBlockchainEventReceived, chainEvent.Namespace, chainEvent.ID, chainEvent.TX.ID, topic)
	if err := em.database.InsertEvent(ctx, ffEvent); err != nil {
		return err
	}
	return nil
}

func (em *eventManager) emitBlockchainEventMetric(event *blockchain.Event) {
	if em.metrics.IsMetricsEnabled() && event.Location != "" && event.Signature != "" {
		em.metrics.BlockchainEvent(event.Location, event.Signature)
	}
}

func (em *eventManager) BlockchainEvent(event *blockchain.EventWithSubscription) error {
	return em.retry.Do(em.ctx, "persist contract event", func(attempt int) (bool, error) {
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			sub, err := em.getChainListenerByProtocolIDCached(ctx, event.Subscription)
			if err != nil {
				return err
			}
			if sub == nil {
				log.L(ctx).Warnf("Event received from unknown subscription %s", event.Subscription)
				return nil // no retry
			}

			chainEvent := buildBlockchainEvent(sub.Namespace, sub.ID, &event.Event, &core.BlockchainTransactionRef{
				BlockchainID: event.BlockchainTXID,
			})
			if err := em.maybePersistBlockchainEvent(ctx, chainEvent); err != nil {
				return err
			}
			em.emitBlockchainEventMetric(&event.Event)
			return nil
		})
		return err != nil, err
	})
}
