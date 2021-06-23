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
	"strconv"

	"github.com/hyperledger-labs/firefly/internal/broadcast"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/data"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/privatemessaging"
	"github.com/hyperledger-labs/firefly/internal/retry"
	"github.com/hyperledger-labs/firefly/pkg/blockchain"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/dataexchange"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/identity"
	"github.com/hyperledger-labs/firefly/pkg/publicstorage"
)

type EventManager interface {
	NewPins() chan<- int64
	NewEvents() chan<- int64
	NewSubscriptions() chan<- *fftypes.UUID
	DeletedSubscriptions() chan<- *fftypes.UUID
	DeleteDurableSubscription(ctx context.Context, subDef *fftypes.Subscription) (err error)
	CreateDurableSubscription(ctx context.Context, subDef *fftypes.Subscription) (err error)
	Start() error
	WaitStop()

	// Bound blockchain callbacks
	TxSubmissionUpdate(bi blockchain.Plugin, txTrackingID string, txState blockchain.TransactionStatus, protocolTxID, errorMessage string, additionalInfo fftypes.JSONObject) error
	BatchPinComplete(bi blockchain.Plugin, batch *blockchain.BatchPin, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error

	// Bound dataexchange callbacks
	TransferResult(dx dataexchange.Plugin, trackingID string, status fftypes.OpStatus, info string, additionalInfo fftypes.JSONObject) error
	BLOBReceived(dx dataexchange.Plugin, peerID string, hash fftypes.Bytes32, payloadRef string) error
	MessageReceived(dx dataexchange.Plugin, peerID string, data []byte) error
}

type eventManager struct {
	ctx                  context.Context
	publicstorage        publicstorage.Plugin
	database             database.Plugin
	identity             identity.Plugin
	broadcast            broadcast.Manager
	messaging            privatemessaging.Manager
	data                 data.Manager
	subManager           *subscriptionManager
	retry                retry.Retry
	aggregator           *aggregator
	newEventNotifier     *eventNotifier
	newPinNotifier       *eventNotifier
	opCorrelationRetries int
	defaultTransport     string
}

func NewEventManager(ctx context.Context, pi publicstorage.Plugin, di database.Plugin, ii identity.Plugin, bm broadcast.Manager, pm privatemessaging.Manager, dm data.Manager) (EventManager, error) {
	if pi == nil || di == nil || ii == nil || dm == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	newPinNotifier := newEventNotifier(ctx, "pins")
	newEventNotifier := newEventNotifier(ctx, "events")
	em := &eventManager{
		ctx:           log.WithLogField(ctx, "role", "event-manager"),
		publicstorage: pi,
		database:      di,
		identity:      ii,
		broadcast:     bm,
		messaging:     pm,
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
		aggregator:           newAggregator(ctx, di, bm, pm, dm, newPinNotifier),
	}

	var err error
	if em.subManager, err = newSubscriptionManager(ctx, di, dm, newEventNotifier, &replySender{
		broadcast: bm,
		messaging: pm,
	}); err != nil {
		return nil, err
	}

	return em, nil
}

func (em *eventManager) Start() (err error) {
	err = em.subManager.start()
	if err == nil {
		err = em.aggregator.start()
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
	return em.subManager.newSubscriptions
}

func (em *eventManager) DeletedSubscriptions() chan<- *fftypes.UUID {
	return em.subManager.deletedSubscriptions
}

func (em *eventManager) WaitStop() {
	em.subManager.close()
	<-em.aggregator.eventPoller.closed
}

func (em *eventManager) CreateDurableSubscription(ctx context.Context, subDef *fftypes.Subscription) (err error) {
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
		return i18n.NewError(ctx, i18n.MsgAlreadyExists, "subscription", subDef.Namespace, subDef.Name)
	}

	// We lock in the starting sequence at creation time, rather than when the first dispatcher
	// starts, as that's a more obvious behavior for users
	sequence, err := calcFirstOffset(ctx, em.database, subDef.Options.FirstEvent)
	if err != nil {
		return err
	}
	lockedInFirstEvent := fftypes.SubOptsFirstEvent(strconv.FormatInt(sequence, 10))
	subDef.Options.FirstEvent = &lockedInFirstEvent

	// The event in the database for the creation of the susbscription, will asynchronously update the submanager
	return em.database.UpsertSubscription(ctx, subDef, false)
}

func (em *eventManager) DeleteDurableSubscription(ctx context.Context, subDef *fftypes.Subscription) (err error) {
	// The event in the database for the deletion of the susbscription, will asynchronously update the submanager
	return em.database.DeleteSubscriptionByID(ctx, subDef.ID)
}
