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

package system

import (
	"context"
	"sync"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/pkg/events"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

const (
	SystemEventsName = "system"
)

// Events the system events listener is used for system components that need access to confirmed events,
// such as the sync<->async bridge.
type Events struct {
	ctx          context.Context
	capabilities *events.Capabilities
	callbacks    events.Callbacks
	mux          sync.Mutex
	listeners    map[string][]EventListener
	connID       string
	readAhead    uint16
}

type EventListener func(event *fftypes.EventDelivery) error

func (ie *Events) Name() string { return SystemEventsName }

func (ie *Events) Init(ctx context.Context, prefix config.Prefix, callbacks events.Callbacks) (err error) {
	*ie = Events{
		ctx:          ctx,
		capabilities: &events.Capabilities{},
		callbacks:    callbacks,
		listeners:    make(map[string][]EventListener),
		readAhead:    uint16(prefix.GetInt(SystemEventsConfReadAhead)),
		connID:       fftypes.ShortID(),
	}
	// We have a single logical connection, that matches all subscriptions
	return callbacks.RegisterConnection(ie.connID, func(sr fftypes.SubscriptionRef) bool { return true })
}

func (ie *Events) Capabilities() *events.Capabilities {
	return ie.capabilities
}

func (ie *Events) GetOptionsSchema(ctx context.Context) string {
	return `{}`
}

func (ie *Events) ValidateOptions(options *fftypes.SubscriptionOptions) error {
	return nil
}

func (ie *Events) AddListener(ns string, el EventListener) error {
	no := false
	newest := fftypes.SubOptsFirstEventNewest
	err := ie.callbacks.EphemeralSubscription(ie.connID, ns, &fftypes.SubscriptionFilter{ /* all events */ }, &fftypes.SubscriptionOptions{
		SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
			WithData:   &no,
			ReadAhead:  &ie.readAhead,
			FirstEvent: &newest,
		},
	})
	if err != nil {
		return err
	}
	ie.mux.Lock()
	ie.listeners[ns] = append(ie.listeners[ns], el)
	ie.mux.Unlock()
	return nil
}

func (ie *Events) DeliveryRequest(connID string, sub *fftypes.Subscription, event *fftypes.EventDelivery, data []*fftypes.Data) error {
	ie.mux.Lock()
	defer ie.mux.Unlock()
	for ns, listeners := range ie.listeners {
		if event.Event.Namespace == ns {
			for _, el := range listeners {
				if err := el(event); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
