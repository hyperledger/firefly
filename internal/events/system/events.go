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

package system

import (
	"context"
	"sync"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/events"
)

const (
	SystemEventsTransport = "system"
)

// Events the system events listener is used for system components that need access to confirmed events,
// such as the sync<->async bridge.
type Events struct {
	ctx          context.Context
	capabilities *events.Capabilities
	callbacks    map[string]events.Callbacks
	mux          sync.Mutex
	listeners    map[string][]EventListener
	connID       string
	readAhead    uint16
}

type EventInterface interface {
	AddSystemEventListener(ns string, el EventListener) error
}

type EventListener func(event *core.EventDelivery) error

func (se *Events) Name() string { return SystemEventsTransport }

func (se *Events) Init(ctx context.Context, config config.Section) (err error) {
	*se = Events{
		ctx:          ctx,
		capabilities: &events.Capabilities{},
		callbacks:    make(map[string]events.Callbacks),
		listeners:    make(map[string][]EventListener),
		readAhead:    uint16(config.GetInt(SystemEventsConfReadAhead)),
		connID:       fftypes.ShortID(),
	}
	return nil
}

func (se *Events) SetHandler(namespace string, handler events.Callbacks) error {
	se.callbacks[namespace] = handler
	// We have a single logical connection, that matches all subscriptions
	return handler.RegisterConnection(se.connID, func(sr core.SubscriptionRef) bool { return true })
}

func (se *Events) Capabilities() *events.Capabilities {
	return se.capabilities
}

func (se *Events) ValidateOptions(options *core.SubscriptionOptions) error {
	return nil
}

func (se *Events) AddListener(ns string, el EventListener) error {
	no := false
	newest := core.SubOptsFirstEventNewest
	if cb, ok := se.callbacks[ns]; ok {
		err := cb.EphemeralSubscription(se.connID, ns, &core.SubscriptionFilter{ /* all events */ }, &core.SubscriptionOptions{
			SubscriptionCoreOptions: core.SubscriptionCoreOptions{
				WithData:   &no,
				ReadAhead:  &se.readAhead,
				FirstEvent: &newest,
			},
		})
		if err != nil {
			return err
		}
		se.mux.Lock()
		se.listeners[ns] = append(se.listeners[ns], el)
		se.mux.Unlock()
	}
	return nil
}

func (se *Events) DeliveryRequest(connID string, sub *core.Subscription, event *core.EventDelivery, data core.DataArray) error {
	se.mux.Lock()
	defer se.mux.Unlock()
	for ns, listeners := range se.listeners {
		if event.Event.Namespace == ns {
			for _, el := range listeners {
				if err := el(event); err != nil {
					return err
				}
			}
		}
	}
	if cb, ok := se.callbacks[sub.Namespace]; ok {
		cb.DeliveryResponse(connID, &core.EventDeliveryResponse{
			ID:           event.ID,
			Rejected:     false,
			Subscription: event.Subscription,
			Reply:        nil,
		})
	}
	return nil
}
