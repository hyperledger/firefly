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

package common

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
)

type BlockchainCallbacks interface {
	SetHandler(namespace string, handler blockchain.Callbacks)
	SetOperationalHandler(namespace string, handler core.OperationCallbacks)

	OperationUpdate(ctx context.Context, plugin blockchain.Plugin, nsOpID string, status core.OpStatus, blockchainTXID, errorMessage string, opOutput fftypes.JSONObject)
	BatchPinComplete(ctx context.Context, batch *blockchain.BatchPin, signingKey *core.VerifierRef) error
	BlockchainNetworkAction(ctx context.Context, namespace, action string, location *fftypes.JSONAny, event *blockchain.Event, signingKey *core.VerifierRef) error
	BlockchainEvent(ctx context.Context, namespace string, event *blockchain.EventWithSubscription) error
}

type callbacks struct {
	handlers   map[string]blockchain.Callbacks
	opHandlers map[string]core.OperationCallbacks
}

func NewBlockchainCallbacks() BlockchainCallbacks {
	return &callbacks{
		handlers:   make(map[string]blockchain.Callbacks),
		opHandlers: make(map[string]core.OperationCallbacks),
	}
}

func (cb *callbacks) SetHandler(namespace string, handler blockchain.Callbacks) {
	cb.handlers[namespace] = handler
}

func (cb *callbacks) SetOperationalHandler(namespace string, handler core.OperationCallbacks) {
	cb.opHandlers[namespace] = handler
}

func (cb *callbacks) OperationUpdate(ctx context.Context, plugin blockchain.Plugin, nsOpID string, status core.OpStatus, blockchainTXID, errorMessage string, opOutput fftypes.JSONObject) {
	namespace, _, _ := core.ParseNamespacedOpID(ctx, nsOpID)
	if handler, ok := cb.opHandlers[namespace]; ok {
		handler.OperationUpdate(plugin, nsOpID, status, blockchainTXID, errorMessage, opOutput)
		return
	}
	log.L(ctx).Errorf("No handler found for blockchain operation '%s'", nsOpID)
}

func (cb *callbacks) BatchPinComplete(ctx context.Context, batch *blockchain.BatchPin, signingKey *core.VerifierRef) error {
	if handler, ok := cb.handlers[batch.Namespace]; ok {
		return handler.BatchPinComplete(batch, signingKey)
	}
	log.L(ctx).Errorf("No handler found for blockchain batch pin on namespace '%s'", batch.Namespace)
	return nil
}

func (cb *callbacks) BlockchainNetworkAction(ctx context.Context, namespace, action string, location *fftypes.JSONAny, event *blockchain.Event, signingKey *core.VerifierRef) error {
	if namespace == "" {
		// V1 networks don't populate namespace, so deliver the event to every handler
		for _, handler := range cb.handlers {
			if err := handler.BlockchainNetworkAction(action, location, event, signingKey); err != nil {
				return err
			}
		}
	} else {
		if handler, ok := cb.handlers[namespace]; ok {
			return handler.BlockchainNetworkAction(action, location, event, signingKey)
		}
		log.L(ctx).Errorf("No handler found for blockchain network action on namespace '%s'", namespace)
	}
	return nil
}

func (cb *callbacks) BlockchainEvent(ctx context.Context, namespace string, event *blockchain.EventWithSubscription) error {
	if namespace == "" {
		// Older token subscriptions don't populate namespace, so deliver the event to every handler
		for _, cb := range cb.handlers {
			// Send the event to all handlers and let them match it to a contract listener
			if err := cb.BlockchainEvent(event); err != nil {
				return err
			}
		}
	} else {
		if handler, ok := cb.handlers[namespace]; ok {
			return handler.BlockchainEvent(event)
		}
		log.L(ctx).Errorf("No handler found for blockchain event on namespace '%s'", namespace)
	}
	return nil
}

func BuildBatchPin(ctx context.Context, namespace string, event *blockchain.Event, sUUIDs string, sBatchHash string, sContexts []string, sPayloadRef string) (batch *blockchain.BatchPin, err error) {
	hexUUIDs, err := hex.DecodeString(strings.TrimPrefix(sUUIDs, "0x"))
	if err != nil || len(hexUUIDs) != 32 {
		log.L(ctx).Errorf("BatchPin event is not valid - bad uuids (%s): %s", sUUIDs, err)
		return nil, i18n.NewError(ctx, coremsgs.MsgInvalidBatchPinEvent, "bad uuids", sUUIDs, err)
	}
	var txnID fftypes.UUID
	copy(txnID[:], hexUUIDs[0:16])
	var batchID fftypes.UUID
	copy(batchID[:], hexUUIDs[16:32])

	var batchHash fftypes.Bytes32
	err = batchHash.UnmarshalText([]byte(sBatchHash))
	if err != nil {
		log.L(ctx).Errorf("BatchPin event is not valid - bad batchHash (%s): %s", sBatchHash, err)
		return nil, err
	}

	contexts := make([]*fftypes.Bytes32, len(sContexts))
	for i, sHash := range sContexts {
		var hash fftypes.Bytes32
		err = hash.UnmarshalText([]byte(sHash))
		if err != nil {
			log.L(ctx).Errorf("BatchPin event is not valid - bad pin %d (%s): %s", i, sHash, err)
			return nil, err
		}
		contexts[i] = &hash
	}

	return &blockchain.BatchPin{
		Namespace:       namespace,
		TransactionID:   &txnID,
		BatchID:         &batchID,
		BatchHash:       &batchHash,
		BatchPayloadRef: sPayloadRef,
		Contexts:        contexts,
		Event:           *event,
	}, nil
}
