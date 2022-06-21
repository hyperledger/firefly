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

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/retry"
	"github.com/hyperledger/firefly/internal/assets"
	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/definitions"
	"github.com/hyperledger/firefly/internal/events/eifactory"
	"github.com/hyperledger/firefly/internal/events/system"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/privatemessaging"
	"github.com/hyperledger/firefly/internal/shareddownload"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/events"
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
	DeleteDurableSubscription(ctx context.Context, subDef *core.Subscription) (err error)
	CreateUpdateDurableSubscription(ctx context.Context, subDef *core.Subscription, mustNew bool) (err error)
	Start() error
	WaitStop()

	// Bound blockchain callbacks
	BatchPinComplete(bi blockchain.Plugin, batch *blockchain.BatchPin, signingKey *core.VerifierRef) error
	BlockchainEvent(event *blockchain.EventWithSubscription) error
	BlockchainNetworkAction(bi blockchain.Plugin, action string, event *blockchain.Event, signingKey *core.VerifierRef) error

	// Bound dataexchange callbacks
	DXEvent(dx dataexchange.Plugin, event dataexchange.DXEvent)

	// Bound sharedstorage callbacks
	SharedStorageBatchDownloaded(ss sharedstorage.Plugin, ns, payloadRef string, data []byte) (*fftypes.UUID, error)
	SharedStorageBlobDownloaded(ss sharedstorage.Plugin, hash fftypes.Bytes32, size int64, payloadRef string)

	// Bound token callbacks
	TokenPoolCreated(ti tokens.Plugin, pool *tokens.TokenPool) error
	TokensTransferred(ti tokens.Plugin, transfer *tokens.TokenTransfer) error
	TokensApproved(ti tokens.Plugin, approval *tokens.TokenApproval) error

	GetPlugins() []*core.NodeStatusPlugin

	// Internal events
	sysmessaging.SystemEvents
}

type eventManager struct {
	ctx                   context.Context
	namespace             string
	ni                    sysmessaging.LocalNodeInfo
	sharedstorage         sharedstorage.Plugin
	database              database.Plugin
	txHelper              txcommon.Helper
	identity              identity.Manager
	definitions           definitions.DefinitionHandler
	data                  data.Manager
	subManager            *subscriptionManager
	retry                 retry.Retry
	aggregator            *aggregator
	broadcast             broadcast.Manager
	messaging             privatemessaging.Manager
	assets                assets.Manager
	sharedDownload        shareddownload.Manager
	blobReceiver          *blobReceiver
	newEventNotifier      *eventNotifier
	newPinNotifier        *eventNotifier
	defaultTransport      string
	internalEvents        *system.Events
	metrics               metrics.Manager
	chainListenerCache    *ccache.Cache
	chainListenerCacheTTL time.Duration
}

func NewEventManager(ctx context.Context, ns string, ni sysmessaging.LocalNodeInfo, si sharedstorage.Plugin, di database.Plugin, bi blockchain.Plugin, im identity.Manager, dh definitions.DefinitionHandler, dm data.Manager, bm broadcast.Manager, pm privatemessaging.Manager, am assets.Manager, sd shareddownload.Manager, mm metrics.Manager, txHelper txcommon.Helper, transports map[string]events.Plugin) (EventManager, error) {
	if ni == nil || si == nil || di == nil || bi == nil || im == nil || dh == nil || dm == nil || bm == nil || pm == nil || am == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "EventManager")
	}
	newPinNotifier := newEventNotifier(ctx, "pins")
	newEventNotifier := newEventNotifier(ctx, "events")
	em := &eventManager{
		ctx:            log.WithLogField(ctx, "role", "event-manager"),
		namespace:      ns,
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
			InitialDelay: config.GetDuration(coreconfig.EventAggregatorRetryInitDelay),
			MaximumDelay: config.GetDuration(coreconfig.EventAggregatorRetryMaxDelay),
			Factor:       config.GetFloat64(coreconfig.EventAggregatorRetryFactor),
		},
		defaultTransport:      config.GetString(coreconfig.EventTransportsDefault),
		newEventNotifier:      newEventNotifier,
		newPinNotifier:        newPinNotifier,
		aggregator:            newAggregator(ctx, ns, di, bi, pm, dh, im, dm, newPinNotifier, mm),
		metrics:               mm,
		chainListenerCache:    ccache.New(ccache.Configure().MaxSize(config.GetByteSize(coreconfig.EventListenerTopicCacheSize))),
		chainListenerCacheTTL: config.GetDuration(coreconfig.EventListenerTopicCacheTTL),
	}
	ie, _ := eifactory.GetPlugin(ctx, system.SystemEventsTransport)
	em.internalEvents = ie.(*system.Events)
	em.blobReceiver = newBlobReceiver(ctx, em.aggregator)

	var err error
	if em.subManager, err = newSubscriptionManager(ctx, ns, di, dm, newEventNotifier, bm, pm, txHelper, transports); err != nil {
		return nil, err
	}

	return em, nil
}

func (em *eventManager) Start() (err error) {
	err = em.subManager.start()
	if err == nil {
		em.aggregator.start()
		em.blobReceiver.start()
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

func (em *eventManager) WaitStop() {
	em.subManager.close()
	em.blobReceiver.stop()
	<-em.aggregator.eventPoller.closed
}

func (em *eventManager) CreateUpdateDurableSubscription(ctx context.Context, subDef *core.Subscription, mustNew bool) (err error) {
	if subDef.Namespace == "" || subDef.Name == "" || subDef.ID == nil {
		return i18n.NewError(ctx, coremsgs.MsgInvalidSubscription)
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
			return i18n.NewError(ctx, coremsgs.MsgAlreadyExists, "subscription", subDef.Namespace, subDef.Name)
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
		sequence, err := calcFirstOffset(ctx, em.namespace, em.database, subDef.Options.FirstEvent)
		if err != nil {
			return err
		}
		lockedInFirstEvent := core.SubOptsFirstEvent(strconv.FormatInt(sequence, 10))
		subDef.Options.FirstEvent = &lockedInFirstEvent
	}

	// The event in the database for the creation of the susbscription, will asynchronously update the submanager
	return em.database.UpsertSubscription(ctx, subDef, !mustNew)
}

func (em *eventManager) DeleteDurableSubscription(ctx context.Context, subDef *core.Subscription) (err error) {
	// The event in the database for the deletion of the susbscription, will asynchronously update the submanager
	return em.database.DeleteSubscriptionByID(ctx, subDef.ID)
}

func (em *eventManager) AddSystemEventListener(ns string, el system.EventListener) error {
	return em.internalEvents.AddListener(ns, el)
}

func (em *eventManager) GetPlugins() []*core.NodeStatusPlugin {
	eventsArray := make([]*core.NodeStatusPlugin, 0)
	plugins := em.subManager.transports

	for name := range plugins {
		eventsArray = append(eventsArray, &core.NodeStatusPlugin{
			PluginType: name,
		})
	}

	return eventsArray
}
