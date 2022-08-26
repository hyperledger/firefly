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

	OperationUpdate(ctx context.Context, plugin core.Named, nsOpID string, status core.OpStatus, blockchainTXID, errorMessage string, opOutput fftypes.JSONObject)
	BatchPinOrNetworkAction(ctx context.Context, nsOrAction string, subInfo *SubscriptionInfo, location *fftypes.JSONAny, event *blockchain.Event, signingKey *core.VerifierRef, params *BatchPinParams) error
	BlockchainEvent(ctx context.Context, namespace string, event *blockchain.EventWithSubscription) error
}

type FireflySubscriptions interface {
	AddSubscription(ctx context.Context, namespace *core.Namespace, version int, subID string, extra interface{})
	RemoveSubscription(ctx context.Context, subID string)
	GetSubscription(subID string) *SubscriptionInfo
}

type callbacks struct {
	handlers   map[string]blockchain.Callbacks
	opHandlers map[string]core.OperationCallbacks
}

type subscriptions struct {
	subs map[string]*SubscriptionInfo
}

type BatchPinParams struct {
	UUIDs      string
	BatchHash  string
	Contexts   []string
	PayloadRef string
}

// A single subscription on network version 1 may receive events from many remote namespaces,
// which in turn map to one or more local namespaces.
// A subscription on network version 2 is always specific to a single local namespace.
type SubscriptionInfo struct {
	Version     int
	V1Namespace map[string][]string
	V2Namespace string
	Extra       interface{}
}

func NewBlockchainCallbacks() BlockchainCallbacks {
	return &callbacks{
		handlers:   make(map[string]blockchain.Callbacks),
		opHandlers: make(map[string]core.OperationCallbacks),
	}
}

func NewFireflySubscriptions() FireflySubscriptions {
	return &subscriptions{
		subs: make(map[string]*SubscriptionInfo),
	}
}

func (cb *callbacks) SetHandler(namespace string, handler blockchain.Callbacks) {
	cb.handlers[namespace] = handler
}

func (cb *callbacks) SetOperationalHandler(namespace string, handler core.OperationCallbacks) {
	cb.opHandlers[namespace] = handler
}

func (cb *callbacks) OperationUpdate(ctx context.Context, plugin core.Named, nsOpID string, status core.OpStatus, blockchainTXID, errorMessage string, opOutput fftypes.JSONObject) {
	namespace, _, _ := core.ParseNamespacedOpID(ctx, nsOpID)
	if handler, ok := cb.opHandlers[namespace]; ok {
		handler.OperationUpdate(&core.OperationUpdate{
			Plugin:         plugin.Name(),
			NamespacedOpID: nsOpID,
			Status:         status,
			BlockchainTXID: blockchainTXID,
			ErrorMessage:   errorMessage,
			Output:         opOutput,
		})
		return
	}
	log.L(ctx).Errorf("No handler found for blockchain operation '%s'", nsOpID)
}

func (cb *callbacks) BatchPinOrNetworkAction(ctx context.Context, nsOrAction string, subInfo *SubscriptionInfo, location *fftypes.JSONAny, event *blockchain.Event, signingKey *core.VerifierRef, params *BatchPinParams) error {
	// Check if this is actually an operator action
	if strings.HasPrefix(nsOrAction, blockchain.FireFlyActionPrefix) {
		action := nsOrAction[len(blockchain.FireFlyActionPrefix):]

		// For V1 of the FireFly contract, action is sent to all namespaces.
		// For V2+, namespace is inferred from the subscription.
		if subInfo.Version == 1 {
			namespaces := make([]string, 0)
			for _, localNames := range subInfo.V1Namespace {
				namespaces = append(namespaces, localNames...)
			}
			return cb.networkAction(ctx, namespaces, action, location, event, signingKey)
		}
		return cb.networkAction(ctx, []string{subInfo.V2Namespace}, action, location, event, signingKey)
	}

	batch, err := buildBatchPin(ctx, event, params)
	if err != nil {
		return nil // move on
	}

	// For V1 of the FireFly contract, namespace is passed explicitly, but needs to be mapped to local name(s).
	// For V2+, namespace is inferred from the subscription.
	if subInfo.Version == 1 {
		namespaces := subInfo.V1Namespace[nsOrAction]
		if len(namespaces) == 0 {
			log.L(ctx).Errorf("No handler found for blockchain batch pin on remote namespace '%s'", nsOrAction)
			return nil
		}
		return cb.batchPinComplete(ctx, namespaces, batch, signingKey)
	}
	return cb.batchPinComplete(ctx, []string{subInfo.V2Namespace}, batch, signingKey)
}

func (cb *callbacks) batchPinComplete(ctx context.Context, namespaces []string, batch *blockchain.BatchPin, signingKey *core.VerifierRef) error {
	for _, namespace := range namespaces {
		if handler, ok := cb.handlers[namespace]; ok {
			if err := handler.BatchPinComplete(namespace, batch, signingKey); err != nil {
				return err
			}
		} else {
			log.L(ctx).Errorf("No handler found for blockchain batch pin on local namespace '%s'", namespace)
		}
	}
	return nil
}

func (cb *callbacks) networkAction(ctx context.Context, namespaces []string, action string, location *fftypes.JSONAny, event *blockchain.Event, signingKey *core.VerifierRef) error {
	for _, namespace := range namespaces {
		if handler, ok := cb.handlers[namespace]; ok {
			if err := handler.BlockchainNetworkAction(action, location, event, signingKey); err != nil {
				return err
			}
		} else {
			log.L(ctx).Errorf("No handler found for blockchain network action on local namespace '%s'", namespace)
		}
	}
	return nil
}

func (cb *callbacks) BlockchainEvent(ctx context.Context, namespace string, event *blockchain.EventWithSubscription) error {
	if namespace == "" {
		// Older subscriptions don't populate namespace, so deliver the event to every handler
		for _, cb := range cb.handlers {
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

func buildBatchPin(ctx context.Context, event *blockchain.Event, params *BatchPinParams) (batch *blockchain.BatchPin, err error) {
	if params.UUIDs == "" || params.BatchHash == "" {
		log.L(ctx).Errorf("BatchPin event is not valid - missing data: %+v", params)
		return nil, i18n.NewError(ctx, coremsgs.MsgInvalidBatchPinEvent, "missing data", params.UUIDs, "")
	}

	hexUUIDs, err := hex.DecodeString(strings.TrimPrefix(params.UUIDs, "0x"))
	if err != nil || len(hexUUIDs) != 32 {
		log.L(ctx).Errorf("BatchPin event is not valid - bad uuids (%s): %s", params.UUIDs, err)
		return nil, i18n.NewError(ctx, coremsgs.MsgInvalidBatchPinEvent, "bad uuids", params.UUIDs, err)
	}
	var txnID fftypes.UUID
	copy(txnID[:], hexUUIDs[0:16])
	var batchID fftypes.UUID
	copy(batchID[:], hexUUIDs[16:32])

	var batchHash fftypes.Bytes32
	err = batchHash.UnmarshalText([]byte(params.BatchHash))
	if err != nil {
		log.L(ctx).Errorf("BatchPin event is not valid - bad batchHash (%s): %s", params.BatchHash, err)
		return nil, err
	}

	contexts := make([]*fftypes.Bytes32, len(params.Contexts))
	for i, sHash := range params.Contexts {
		var hash fftypes.Bytes32
		err = hash.UnmarshalText([]byte(sHash))
		if err != nil {
			log.L(ctx).Errorf("BatchPin event is not valid - bad pin %d (%s): %s", i, sHash, err)
			return nil, err
		}
		contexts[i] = &hash
	}

	return &blockchain.BatchPin{
		TransactionID:   &txnID,
		BatchID:         &batchID,
		BatchHash:       &batchHash,
		BatchPayloadRef: params.PayloadRef,
		Contexts:        contexts,
		Event:           *event,
	}, nil
}

func GetNamespaceFromSubName(subName string) string {
	var parts = strings.Split(subName, "-")
	// Subscription names post version 1.1 are in the format `ff-sub-<namespace>-<listener ID>`
	if len(parts) != 4 {
		// Assume older subscription and return empty string
		return ""
	}
	return parts[2]
}

func (s *subscriptions) AddSubscription(ctx context.Context, namespace *core.Namespace, version int, subID string, extra interface{}) {
	if version == 1 {
		// The V1 contract shares a single subscription per contract, and the remote namespace name is passed on chain.
		// Therefore, it requires a map of remote->local in order to farm out notifications to one or more local handlers.
		existing, ok := s.subs[subID]
		if !ok {
			existing = &SubscriptionInfo{
				Version:     version,
				V1Namespace: make(map[string][]string),
				Extra:       extra,
			}
			s.subs[subID] = existing
		}
		existing.V1Namespace[namespace.NetworkName] = append(existing.V1Namespace[namespace.NetworkName], namespace.Name)
	} else {
		// The V2 contract does not pass the namespace on chain, and requires a separate contract instance (and subscription) per namespace.
		// Therefore, the local namespace name can simply be cached alongside each subscription.
		s.subs[subID] = &SubscriptionInfo{
			Version:     version,
			V2Namespace: namespace.Name,
			Extra:       extra,
		}
	}
}

func (s *subscriptions) RemoveSubscription(ctx context.Context, subID string) {
	if _, ok := s.subs[subID]; ok {
		delete(s.subs, subID)
	} else {
		log.L(ctx).Debugf("Invalid subscription ID: %s", subID)
	}
}

func (s *subscriptions) GetSubscription(subID string) *SubscriptionInfo {
	return s.subs[subID]
}
