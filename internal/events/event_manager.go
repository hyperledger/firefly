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
	"bytes"
	"context"
	"encoding/json"
	"strconv"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/events/eifactory"
	"github.com/hyperledger/firefly/internal/events/system"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/internal/syshandlers"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/publicstorage"
	"github.com/hyperledger/firefly/pkg/tokens"
)

type EventManager interface {
	NewPins() chan<- int64
	NewEvents() chan<- int64
	NewSubscriptions() chan<- *fftypes.UUID
	SubscriptionUpdates() chan<- *fftypes.UUID
	DeletedSubscriptions() chan<- *fftypes.UUID
	ChangeEvents() chan<- *fftypes.ChangeEvent
	DeleteDurableSubscription(ctx context.Context, subDef *fftypes.Subscription) (err error)
	CreateUpdateDurableSubscription(ctx context.Context, subDef *fftypes.Subscription, mustNew bool) (err error)
	Start() error
	WaitStop()

	// Bound blockchain callbacks
	TxSubmissionUpdate(plugin fftypes.Named, tx string, txState blockchain.TransactionStatus, errorMessage string, additionalInfo fftypes.JSONObject) error
	BatchPinComplete(bi blockchain.Plugin, batch *blockchain.BatchPin, author string, protocolTxID string, additionalInfo fftypes.JSONObject) error

	// Bound dataexchange callbacks
	TransferResult(dx dataexchange.Plugin, trackingID string, status fftypes.OpStatus, info string, additionalInfo fftypes.JSONObject) error
	BLOBReceived(dx dataexchange.Plugin, peerID string, hash fftypes.Bytes32, payloadRef string) error
	MessageReceived(dx dataexchange.Plugin, peerID string, data []byte) error

	// Bound token callbacks
	TokenPoolCreated(tk tokens.Plugin, pool *fftypes.TokenPool, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error

	// Internal events
	sysmessaging.SystemEvents
}

type eventManager struct {
	ctx                  context.Context
	publicstorage        publicstorage.Plugin
	database             database.Plugin
	identity             identity.Manager
	syshandlers          syshandlers.SystemHandlers
	data                 data.Manager
	subManager           *subscriptionManager
	retry                retry.Retry
	aggregator           *aggregator
	newEventNotifier     *eventNotifier
	newPinNotifier       *eventNotifier
	opCorrelationRetries int
	defaultTransport     string
	internalEvents       *system.Events
}

func NewEventManager(ctx context.Context, pi publicstorage.Plugin, di database.Plugin, im identity.Manager, sh syshandlers.SystemHandlers, dm data.Manager) (EventManager, error) {
	if pi == nil || di == nil || im == nil || dm == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	newPinNotifier := newEventNotifier(ctx, "pins")
	newEventNotifier := newEventNotifier(ctx, "events")
	em := &eventManager{
		ctx:           log.WithLogField(ctx, "role", "event-manager"),
		publicstorage: pi,
		database:      di,
		identity:      im,
		syshandlers:   sh,
		data:          dm,
		retry: retry.Retry{
			InitialDelay: config.GetDuration(config.EventAggregatorRetryInitDelay),
			MaximumDelay: config.GetDuration(config.EventAggregatorRetryMaxDelay),
			Factor:       config.GetFloat64(config.EventAggregatorRetryFactor),
		},
		defaultTransport:     config.GetString(config.EventTransportsDefault),
		opCorrelationRetries: config.GetInt(config.EventAggregatorOpCorrelationRetries),
		newEventNotifier:     newEventNotifier,
		newPinNotifier:       newPinNotifier,
		aggregator:           newAggregator(ctx, di, sh, dm, newPinNotifier),
	}
	ie, _ := eifactory.GetPlugin(ctx, system.SystemEventsTransport)
	em.internalEvents = ie.(*system.Events)

	var err error
	if em.subManager, err = newSubscriptionManager(ctx, di, dm, newEventNotifier, sh); err != nil {
		return nil, err
	}

	return em, nil
}

func (em *eventManager) Start() (err error) {
	err = em.subManager.start()
	if err == nil {
		em.aggregator.start()
	}
	return err
}

func (em *eventManager) NewEvents() chan<- int64 {
	return em.newEventNotifier.newEvents
}

func (em *eventManager) NewPins() chan<- int64 {
	return em.newPinNotifier.newEvents
}

func (em *eventManager) NewSubscriptions() chan<- *fftypes.UUID {
	return em.subManager.newOrUpdatedSubscriptions
}

func (em *eventManager) SubscriptionUpdates() chan<- *fftypes.UUID {
	return em.subManager.newOrUpdatedSubscriptions
}

func (em *eventManager) DeletedSubscriptions() chan<- *fftypes.UUID {
	return em.subManager.deletedSubscriptions
}

func (em *eventManager) ChangeEvents() chan<- *fftypes.ChangeEvent {
	return em.subManager.cel.changeEvents
}

func (em *eventManager) WaitStop() {
	em.subManager.close()
	<-em.aggregator.eventPoller.closed
}

func (em *eventManager) CreateUpdateDurableSubscription(ctx context.Context, subDef *fftypes.Subscription, mustNew bool) (err error) {
	if subDef.Namespace == "" || subDef.Name == "" || subDef.ID == nil {
		return i18n.NewError(ctx, i18n.MsgInvalidSubscription)
	}

	if subDef.Transport == "" {
		subDef.Transport = em.defaultTransport
	}

	// Check it can be parsed before inserting (the submanager will check again when processing the creation, so we discard the result)
	if _, err = em.subManager.parseSubscriptionDef(ctx, subDef); err != nil {
		return err
	}

	// Do a check first for existence, to give a nice 409 if we find one
	existing, _ := em.database.GetSubscriptionByName(ctx, subDef.Namespace, subDef.Name)
	if existing != nil {
		if mustNew {
			return i18n.NewError(ctx, i18n.MsgAlreadyExists, "subscription", subDef.Namespace, subDef.Name)
		}
		// Copy over the generated fields, so we can do a compare
		subDef.Created = existing.Created
		subDef.ID = existing.ID
		subDef.Updated = fftypes.Now()
		subDef.Options.FirstEvent = existing.Options.FirstEvent // we do not reset the sub position
		existing.Updated = subDef.Updated
		def1, _ := json.Marshal(existing)
		def2, _ := json.Marshal(subDef)
		if bytes.Equal(def1, def2) {
			log.L(ctx).Infof("Subscription already exists, and is identical")
			return nil
		}
	} else {
		// We lock in the starting sequence at creation time, rather than when the first dispatcher
		// starts, as that's a more obvious behavior for users
		sequence, err := calcFirstOffset(ctx, em.database, subDef.Options.FirstEvent)
		if err != nil {
			return err
		}
		lockedInFirstEvent := fftypes.SubOptsFirstEvent(strconv.FormatInt(sequence, 10))
		subDef.Options.FirstEvent = &lockedInFirstEvent
	}

	// The event in the database for the creation of the susbscription, will asynchronously update the submanager
	return em.database.UpsertSubscription(ctx, subDef, !mustNew)
}

func (em *eventManager) DeleteDurableSubscription(ctx context.Context, subDef *fftypes.Subscription) (err error) {
	// The event in the database for the deletion of the susbscription, will asynchronously update the submanager
	return em.database.DeleteSubscriptionByID(ctx, subDef.ID)
}

func (em *eventManager) AddSystemEventListener(ns string, el system.EventListener) error {
	return em.internalEvents.AddListener(ns, el)
}
