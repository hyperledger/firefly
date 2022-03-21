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
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/hyperledger/firefly/internal/assets"
	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/definitions"
	"github.com/hyperledger/firefly/internal/events/eifactory"
	"github.com/hyperledger/firefly/internal/events/system"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/privatemessaging"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/internal/shareddownload"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/karlseguin/ccache"
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
	OperationUpdate(plugin fftypes.Named, operationID *fftypes.UUID, txState blockchain.TransactionStatus, blockchainTXID, errorMessage string, opOutput fftypes.JSONObject) error
	BatchPinComplete(bi blockchain.Plugin, batch *blockchain.BatchPin, signingKey *fftypes.VerifierRef) error
	BlockchainEvent(event *blockchain.EventWithSubscription) error

	// Bound dataexchange callbacks
	TransferResult(dx dataexchange.Plugin, trackingID string, status fftypes.OpStatus, update fftypes.TransportStatusUpdate) error
	PrivateBLOBReceived(dx dataexchange.Plugin, peerID string, hash fftypes.Bytes32, size int64, payloadRef string) error
	MessageReceived(dx dataexchange.Plugin, peerID string, data []byte) (manifest string, err error)

	// Bound sharedstorage callbacks
	SharedStorageBatchDownloaded(ss sharedstorage.Plugin, ns, payloadRef string, data []byte) (*fftypes.UUID, error)
	SharedStorageBLOBDownloaded(ss sharedstorage.Plugin, hash fftypes.Bytes32, size int64, payloadRef string) error

	// Bound token callbacks
	TokenPoolCreated(ti tokens.Plugin, pool *tokens.TokenPool) error
	TokensTransferred(ti tokens.Plugin, transfer *tokens.TokenTransfer) error
	TokensApproved(ti tokens.Plugin, approval *tokens.TokenApproval) error

	// Internal events
	sysmessaging.SystemEvents
}

type eventManager struct {
	ctx                   context.Context
	ni                    sysmessaging.LocalNodeInfo
	sharedstorage         sharedstorage.Plugin
	database              database.Plugin
	txHelper              txcommon.Helper
	identity              identity.Manager
	definitions           definitions.DefinitionHandlers
	data                  data.Manager
	subManager            *subscriptionManager
	retry                 retry.Retry
	aggregator            *aggregator
	broadcast             broadcast.Manager
	messaging             privatemessaging.Manager
	assets                assets.Manager
	sharedDownload        shareddownload.Manager
	newEventNotifier      *eventNotifier
	newPinNotifier        *eventNotifier
	opCorrelationRetries  int
	defaultTransport      string
	internalEvents        *system.Events
	metrics               metrics.Manager
	chainListenerCache    *ccache.Cache
	chainListenerCacheTTL time.Duration
}

func NewEventManager(ctx context.Context, ni sysmessaging.LocalNodeInfo, si sharedstorage.Plugin, di database.Plugin, bi blockchain.Plugin, im identity.Manager, dh definitions.DefinitionHandlers, dm data.Manager, bm broadcast.Manager, pm privatemessaging.Manager, am assets.Manager, sd shareddownload.Manager, mm metrics.Manager, txHelper txcommon.Helper) (EventManager, error) {
	if ni == nil || si == nil || di == nil || bi == nil || im == nil || dh == nil || dm == nil || bm == nil || pm == nil || am == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	newPinNotifier := newEventNotifier(ctx, "pins")
	newEventNotifier := newEventNotifier(ctx, "events")
	em := &eventManager{
		ctx:            log.WithLogField(ctx, "role", "event-manager"),
		ni:             ni,
		sharedstorage:  si,
		database:       di,
		txHelper:       txHelper,
		identity:       im,
		definitions:    dh,
		data:           dm,
		broadcast:      bm,
		messaging:      pm,
		assets:         am,
		sharedDownload: sd,
		retry: retry.Retry{
			InitialDelay: config.GetDuration(config.EventAggregatorRetryInitDelay),
			MaximumDelay: config.GetDuration(config.EventAggregatorRetryMaxDelay),
			Factor:       config.GetFloat64(config.EventAggregatorRetryFactor),
		},
		defaultTransport:      config.GetString(config.EventTransportsDefault),
		opCorrelationRetries:  config.GetInt(config.EventAggregatorOpCorrelationRetries),
		newEventNotifier:      newEventNotifier,
		newPinNotifier:        newPinNotifier,
		aggregator:            newAggregator(ctx, di, bi, dh, im, dm, newPinNotifier, mm),
		metrics:               mm,
		chainListenerCache:    ccache.New(ccache.Configure().MaxSize(config.GetByteSize(config.EventListenerTopicCacheSize))),
		chainListenerCacheTTL: config.GetDuration(config.EventListenerTopicCacheTTL),
	}
	ie, _ := eifactory.GetPlugin(ctx, system.SystemEventsTransport)
	em.internalEvents = ie.(*system.Events)

	var err error
	if em.subManager, err = newSubscriptionManager(ctx, di, dm, newEventNotifier, dh, txHelper); err != nil {
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
