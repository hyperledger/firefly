// Copyright © 2021 Kaleido, Inc.
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
	"time"

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/retry"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

type eventDispatcher struct {
	ctx          context.Context
	database     database.Plugin
	subscription *fftypes.Subscription
	eventPoller  *eventPoller
	cancelCtx    func()
}

func newEventDispatcher(ctx context.Context, di database.Plugin, subscription *fftypes.Subscription) *eventDispatcher {
	ctx, cancelCtx := context.WithCancel(ctx)
	ed := &eventDispatcher{
		ctx:       ctx,
		database:  di,
		cancelCtx: cancelCtx,
	}

	pollerConf := eventPollerConf{
		eventBatchSize:             config.GetInt(config.SubscriptionDefaultsBatchSize),
		eventBatchTimeout:          config.GetDuration(config.SubscriptionDefaultsBatchTimeout),
		eventPollTimeout:           config.GetDuration(config.EventDispatcherPollTimeout),
		startupOffsetRetryAttempts: config.GetInt(config.OrchestratorStartupAttempts),
		retry: retry.Retry{
			InitialDelay: config.GetDuration(config.EventDispatcherRetryInitDelay),
			MaximumDelay: config.GetDuration(config.EventDispatcherRetryMaxDelay),
			Factor:       config.GetFloat64(config.EventDispatcherRetryFactor),
		},
		offsetType:      fftypes.OffsetTypeSubscription,
		offsetNamespace: subscription.Namespace,
		offsetName:      subscription.Name,
		processEvent:    ed.processEvent,
		ephemeral:       subscription.Ephemeral,
	}
	if subscription.Options.BatchSize != nil {
		pollerConf.eventBatchSize = int(*subscription.Options.BatchSize)
	}
	if subscription.Options.BatchTimeout != nil {
		pollerConf.eventBatchTimeout = time.Duration(*subscription.Options.BatchTimeout)
	}

	ed.eventPoller = newEventPoller(ctx, di, pollerConf)
	return ed
}

func (ed *eventDispatcher) start() error {
	return ed.eventPoller.start()
}

func (ed *eventDispatcher) processEvent(ctx context.Context, event *fftypes.Event) (bool, error) {
	l := log.L(ctx)
	l.Debugf("Dispatching event: %.10d/%s [%s]: %s/%s", event.Sequence, event.ID, event.Type, event.Namespace, event.Reference)
	return false, nil
}

func (ed *eventDispatcher) close() {
	ed.cancelCtx()
	<-ed.eventPoller.closed
}
