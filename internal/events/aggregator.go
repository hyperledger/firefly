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

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/retry"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

const (
	aggregatorOffsetName = "ff-aggregator"
)

type aggregator struct {
	ctx         context.Context
	database    database.Plugin
	eventPoller *eventPoller
}

func newAggregator(ctx context.Context, di database.Plugin) *aggregator {
	ag := &aggregator{
		ctx:      log.WithLogField(ctx, "role", "aggregator"),
		database: di,
	}
	ag.eventPoller = newEventPoller(ctx, di, eventPollerConf{
		eventBatchSize:             config.GetInt(config.EventAggregatorBatchSize),
		eventBatchTimeout:          config.GetDuration(config.EventAggregatorBatchTimeout),
		eventPollTimeout:           config.GetDuration(config.EventAggregatorPollTimeout),
		startupOffsetRetryAttempts: config.GetInt(config.OrchestratorStartupAttempts),
		retry: retry.Retry{
			InitialDelay: config.GetDuration(config.EventAggregatorRetryInitDelay),
			MaximumDelay: config.GetDuration(config.EventAggregatorRetryMaxDelay),
			Factor:       config.GetFloat64(config.EventAggregatorRetryFactor),
		},
		offsetType:       fftypes.OffsetTypeAggregator,
		offsetNamespace:  fftypes.SystemNamespace,
		offsetName:       aggregatorOffsetName,
		newEventsHandler: ag.processEventRetryAndGroup,
	})
	return ag
}

func (ag *aggregator) start() error {
	return ag.eventPoller.start()
}

func (ag *aggregator) processEventRetryAndGroup(events []*fftypes.Event) (repoll bool, err error) {
	err = ag.database.RunAsGroup(ag.ctx, func(ctx context.Context) (err error) {
		repoll, err = ag.processEvents(ctx, events)
		return err
	})
	return repoll, err
}

func (ag *aggregator) processEvents(ctx context.Context, events []*fftypes.Event) (repoll bool, err error) {
	for _, event := range events {
		repoll, err = ag.processEvent(ctx, event)
		if err != nil {
			return false, err
		}
	}
	err = ag.eventPoller.commitOffset(ctx, events[len(events)-1].Sequence)
	return repoll, err
}

func (ag *aggregator) processEvent(ctx context.Context, event *fftypes.Event) (bool, error) {
	l := log.L(ctx)
	l.Debugf("Aggregating event %.10d/%s [%s]: %s/%s", event.Sequence, event.ID, event.Type, event.Namespace, event.Reference)
	switch event.Type {
	case fftypes.EventTypeDataArrivedBroadcast:
		return ag.processDataArrived(ctx, event.Namespace, event.Reference)
	case fftypes.EventTypeMessageSequencedBroadcast:
		msg, err := ag.database.GetMessageById(ctx, event.Reference)
		if err != nil {
			return false, err
		}
		if msg != nil && msg.Confirmed == nil {
			return ag.checkMessageComplete(ctx, msg)
		}
	default:
		// Other events do not need aggregation.
		// Note this MUST include all events that are generated via aggregation, or we would infinite loop
	}
	l.Debugf("No action for event %.10d/%s [%s]: %s/%s", event.Sequence, event.ID, event.Type, event.Namespace, event.Reference)
	return false, nil
}

func (ag *aggregator) processDataArrived(ctx context.Context, ns string, dataId *fftypes.UUID) (bool, error) {
	// Find all unconfirmed messages associated with this data
	fb := database.MessageQueryFactory.NewFilter(ctx)
	msgs, err := ag.database.GetMessagesForData(ctx, dataId, fb.And(
		fb.Eq("namespace", ns),
		fb.Eq("confirmed", nil),
	).Sort("sequence"))
	if err != nil {
		return false, err
	}
	repoll := false
	for _, msg := range msgs {
		msgRepoll, err := ag.checkMessageComplete(ctx, msg)
		if err != nil {
			return false, err
		}
		repoll = repoll || msgRepoll
	}
	return repoll, nil
}

func (ag *aggregator) checkMessageComplete(ctx context.Context, msg *fftypes.Message) (bool, error) {
	repoll := false
	complete, err := ag.database.CheckDataAvailable(ctx, msg)
	if err != nil {
		return false, err // CheckDataAvailable should only return an error if there's a problem with persistence
	}
	if complete {
		// Check backwards to see if there are any earlier messages on this context that are incomplete
		fb := database.MessageQueryFactory.NewFilter(ctx)
		filter := fb.And(
			fb.Eq("namespace", msg.Header.Namespace),
			fb.Eq("group", msg.Header.Group),
			fb.Eq("context", msg.Header.Context),
			fb.Lt("sequence", msg.Sequence), // Below our sequence
			fb.Eq("confirmed", nil),
		).Sort("sequence").Limit(1) // only need one message
		incomplete, err := ag.database.GetMessageRefs(ctx, filter)
		if err != nil {
			return false, err
		}
		if len(incomplete) > 0 {
			log.L(ctx).Infof("Message %s (seq=%d) blocked by incompleted message %s (seq=%d)", msg.Header.ID, msg.Sequence, incomplete[0].ID, incomplete[0].Sequence)
			return false, nil
		}

		// This message is now confirmed
		setConfirmed := database.MessageQueryFactory.NewUpdate(ctx).Set("confirmed", fftypes.Now())
		err = ag.database.UpdateMessage(ctx, msg.Header.ID, setConfirmed)
		if err != nil {
			return false, err
		}

		// Emit the confirmed event
		event := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, msg.Header.Namespace, msg.Header.ID)
		if err = ag.database.UpsertEvent(ctx, event, false); err != nil {
			return false, err
		}

		// Check forwards to see if there is a message we could unblock
		fb = database.MessageQueryFactory.NewFilter(ctx)
		filter = fb.And(
			fb.Eq("namespace", msg.Header.Namespace),
			fb.Eq("group", msg.Header.Group),
			fb.Eq("context", msg.Header.Context),
			fb.Gt("sequence", msg.Sequence), // Above our sequence
			fb.Eq("confirmed", nil),
		).Sort("sequence").Limit(1) // only need one message
		unblockable, err := ag.database.GetMessageRefs(ctx, filter)
		if err != nil {
			return false, err
		}
		if len(unblockable) > 0 {
			log.L(ctx).Infof("Message %s (seq=%d) unblocks message %s (seq=%d)", msg.Header.ID, msg.Sequence, unblockable[0].ID, unblockable[0].Sequence)
			event := fftypes.NewEvent(fftypes.EventTypeMessagesUnblocked, msg.Header.Namespace, unblockable[0].ID)
			if err = ag.database.UpsertEvent(ctx, event, false); err != nil {
				return false, err
			}
			// We want the loop to fire again, to pick up this unblock event
			repoll = true
		}

	}
	return repoll, nil
}
