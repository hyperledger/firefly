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
		return em.database.GetContractListenerByBackendID(ctx, em.namespace.Name, protocolID)
	})
}

func (em *eventManager) getChainListenerCached(cacheKey string, getter func() (*core.ContractListener, error)) (*core.ContractListener, error) {

	if cachedValue := em.chainListenerCache.Get(cacheKey); cachedValue != nil {
		return cachedValue.(*core.ContractListener), nil
	}
	listener, err := getter()
	if listener == nil || err != nil {
		return nil, err
	}
	em.chainListenerCache.Set(cacheKey, listener)
	return listener, err
}

func (em *eventManager) getTopicForChainListener(listener *core.ContractListener) string {
	if listener == nil {
		return core.SystemBatchPinTopic
	}
	var topic string
	if listener != nil && listener.Topic != "" {
		topic = listener.Topic
	} else {
		topic = listener.ID.String()
	}
	return topic
}

func (em *eventManager) maybePersistBlockchainEvent(ctx context.Context, chainEvent *core.BlockchainEvent, listener *core.ContractListener) error {
	if existing, err := em.txHelper.InsertOrGetBlockchainEvent(ctx, chainEvent); err != nil {
		return err
	} else if existing != nil {
		log.L(ctx).Debugf("Ignoring duplicate blockchain event %s", chainEvent.ProtocolID)
		// Return the ID of the existing event
		chainEvent.ID = existing.ID
		return nil
	}
	topic := em.getTopicForChainListener(listener)
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
	return em.retry.Do(em.ctx, "persist blockchain event", func(attempt int) (bool, error) {
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			listener, err := em.getChainListenerByProtocolIDCached(ctx, event.Subscription)
			if err != nil {
				return err
			}
			if listener == nil {
				log.L(ctx).Warnf("Event received from unknown subscription %s", event.Subscription)
				return nil // no retry
			}
			if listener.Namespace != em.namespace.Name {
				log.L(em.ctx).Debugf("Ignoring blockchain event from different namespace '%s'", listener.Namespace)
				return nil
			}
			listener.Namespace = em.namespace.Name

			chainEvent := buildBlockchainEvent(listener.Namespace, listener.ID, &event.Event, &core.BlockchainTransactionRef{
				BlockchainID: event.BlockchainTXID,
			})
			if err := em.maybePersistBlockchainEvent(ctx, chainEvent, listener); err != nil {
				return err
			}
			em.emitBlockchainEventMetric(&event.Event)
			return nil
		})
		return err != nil, err
	})
}
