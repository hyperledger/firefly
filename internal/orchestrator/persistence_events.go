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

package orchestrator

import (
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (or *orchestrator) attemptChangeEventDispatch(ev *fftypes.ChangeEvent) {
	// For change events we're not processing as a system, we don't block our processing to dispatch
	// them remotely. So if the queue is full, we discard the event rather than blocking.
	select {
	case or.events.ChangeEvents() <- ev:
	default:
		log.L(or.ctx).Warnf("Database change event queue is exhausted")
	}
}

func (or *orchestrator) OrderedUUIDCollectionNSEvent(resType database.OrderedUUIDCollectionNS, eventType fftypes.ChangeEventType, ns string, id *fftypes.UUID, sequence int64) {
	switch {
	case eventType == fftypes.ChangeEventTypeCreated && resType == database.CollectionMessages:
		or.batch.NewMessages() <- sequence
	case eventType == fftypes.ChangeEventTypeCreated && resType == database.CollectionEvents:
		or.events.NewEvents() <- sequence
	}
	or.attemptChangeEventDispatch(&fftypes.ChangeEvent{
		Collection: string(resType),
		Type:       eventType,
		Namespace:  ns,
		ID:         id,
		Sequence:   &sequence,
	})
}

func (or *orchestrator) OrderedCollectionEvent(resType database.OrderedCollection, eventType fftypes.ChangeEventType, sequence int64) {
	if eventType == fftypes.ChangeEventTypeCreated && resType == database.CollectionPins {
		or.events.NewPins() <- sequence
	}
	or.attemptChangeEventDispatch(&fftypes.ChangeEvent{
		Collection: string(resType),
		Type:       eventType,
		Sequence:   &sequence,
	})
}

func (or *orchestrator) UUIDCollectionNSEvent(resType database.UUIDCollectionNS, eventType fftypes.ChangeEventType, ns string, id *fftypes.UUID) {
	switch {
	case eventType == fftypes.ChangeEventTypeCreated && resType == database.CollectionSubscriptions:
		or.events.NewSubscriptions() <- id
	case eventType == fftypes.ChangeEventTypeDeleted && resType == database.CollectionSubscriptions:
		or.events.DeletedSubscriptions() <- id
	}
	or.attemptChangeEventDispatch(&fftypes.ChangeEvent{
		Collection: string(resType),
		Type:       eventType,
		Namespace:  ns,
		ID:         id,
	})
}

func (or *orchestrator) UUIDCollectionEvent(resType database.UUIDCollection, eventType fftypes.ChangeEventType, id *fftypes.UUID) {
	or.attemptChangeEventDispatch(&fftypes.ChangeEvent{
		Collection: string(resType),
		Type:       eventType,
		ID:         id,
	})
}

func (or *orchestrator) HashCollectionNSEvent(resType database.HashCollectionNS, eventType fftypes.ChangeEventType, ns string, hash *fftypes.Bytes32) {
	or.attemptChangeEventDispatch(&fftypes.ChangeEvent{
		Collection: string(resType),
		Type:       eventType,
		Namespace:  ns,
		Hash:       hash,
	})

}
