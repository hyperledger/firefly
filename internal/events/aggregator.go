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

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/retry"
)

const (
	aggregatorOffsetName = "ff-aggregator"
)

type aggregator struct {
	ctx                        context.Context
	database                   database.Plugin
	shoulderTap                chan bool
	newEvents                  chan *uuid.UUID
	closed                     chan struct{}
	eventPollTimeout           time.Duration
	readPageSize               uint64
	startupOffsetRetryAttempts int
	retry                      retry.Retry
	offset                     int64
}

func newAggregator(ctx context.Context, di database.Plugin) *aggregator {
	readPageSize := config.GetUint(config.EventAggregatorReadPageSize)
	ag := &aggregator{
		ctx:                        log.WithLogField(ctx, "role", "aggregator"),
		database:                   di,
		readPageSize:               uint64(readPageSize),
		eventPollTimeout:           config.GetDuration(config.EventAggregatorReadPollTimeout),
		startupOffsetRetryAttempts: config.GetInt(config.EventAggregatorStartupAttempts),
		shoulderTap:                make(chan bool, 1),
		newEvents:                  make(chan *uuid.UUID),
		closed:                     make(chan struct{}),
		retry: retry.Retry{
			InitialDelay: config.GetDuration(config.EventAggregatorRetryInitDelay),
			MaximumDelay: config.GetDuration(config.EventAggregatorRetryMaxDelay),
			Factor:       config.GetFloat64(config.EventAggregatorRetryFactor),
		},
	}
	return ag
}

func (ag *aggregator) restoreOffset() error {
	return ag.retry.Do(ag.ctx, "restore offset", func(attempt int) (retry bool, err error) {
		offset, err := ag.database.GetOffset(ag.ctx, fftypes.OffsetTypeAggregator, fftypes.SystemNamespace, aggregatorOffsetName)
		if err == nil {
			if offset == nil {
				err = ag.updateOffset(ag.ctx, 0)
			} else {
				ag.offset = offset.Current
			}
		}
		if err != nil {
			return (attempt <= ag.startupOffsetRetryAttempts), err
		}
		log.L(ag.ctx).Infof("Event aggregator restored offset %d", ag.offset)
		return false, nil
	})
}

func (ag *aggregator) start() error {
	if err := ag.restoreOffset(); err != nil {
		return err
	}
	go ag.newEventNotifications()
	go ag.eventLoop()
	return nil
}

func (ag *aggregator) updateOffset(ctx context.Context, newOffset int64) error {
	l := log.L(ctx)
	ag.offset = newOffset
	offset := &fftypes.Offset{
		Type:      fftypes.OffsetTypeAggregator,
		Namespace: fftypes.SystemNamespace,
		Name:      aggregatorOffsetName,
		Current:   ag.offset,
	}
	if err := ag.database.UpsertOffset(ctx, offset, true); err != nil {
		return err
	}
	l.Infof("Event aggregator committed offset %d", newOffset)
	ag.offset = newOffset
	return nil
}

func (ag *aggregator) readPage() ([]*fftypes.Event, error) {
	var msgs []*fftypes.Event
	err := ag.retry.Do(ag.ctx, "retrieve events", func(attempt int) (retry bool, err error) {
		fb := database.MessageQueryFactory.NewFilterLimit(ag.ctx, ag.readPageSize)
		msgs, err = ag.database.GetEvents(ag.ctx, fb.Gt("sequence", ag.offset).Sort("sequence").Limit(ag.readPageSize))
		if err != nil {
			return true, err // Retry indefinitely, until context cancelled
		}
		return false, nil
	})
	return msgs, err
}

func (ag *aggregator) eventLoop() {
	l := log.L(ag.ctx)
	l.Debugf("Started event aggregator")
	defer close(ag.closed)

	for {
		// Read messages from the DB - in an error condition we retry until success, or a closed context
		events, err := ag.readPage()
		if err != nil {
			l.Debugf("Exiting: %s", err)
			return
		}

		if len(events) > 0 {
			// We process all the events in the page in a single database run group, and
			// keep retrying on all retryable errors, indefinitely ().
			if err := ag.processEventRetryAndGroup(events); err != nil {
				l.Debugf("Exiting: %s", err)
				return
			}
		} else {
			// Once we run out of events, wait to be woken
			if ok := ag.waitForShoulderTapOrPollTimeout(); !ok {
				return
			}
		}

	}
}

func (ag *aggregator) processEventRetryAndGroup(events []*fftypes.Event) error {
	return ag.retry.Do(ag.ctx, "process events", func(attempt int) (retry bool, err error) {
		err = ag.database.RunAsGroup(ag.ctx, func(ctx context.Context) error {
			return ag.processEvents(ctx, events)
		})
		return err != nil, err // always retry (retry will end on cancelled context)
	})
}

func (ag *aggregator) processEvents(ctx context.Context, events []*fftypes.Event) error {
	var highestSequence int64
	for _, event := range events {
		if err := ag.processEvent(ctx, event); err != nil {
			return err
		}
		if event.Sequence > highestSequence {
			highestSequence = event.Sequence
		}
	}
	if highestSequence > 0 {
		return ag.updateOffset(ctx, highestSequence)
	}
	return nil
}

func (ag *aggregator) processEvent(ctx context.Context, event *fftypes.Event) error {
	l := log.L(ctx)
	l.Debugf("Aggregating event %.10d/%s type:%s ref:%s/%s", event.Sequence, event.ID, event.Type, event.Namespace, event.Reference)
	l.Tracef("Event data: %+v", event.Type)
	switch event.Type {
	case fftypes.EventTypeDataArrivedBroadcast:
		return ag.processDataArrived(ctx, event.Namespace, event.Reference)
	case fftypes.EventTypeMessageSequencedBroadcast:
		msg, err := ag.database.GetMessageById(ctx, event.Reference)
		if err != nil {
			return err
		}
		if msg != nil && msg.Confirmed == nil {
			return ag.checkMessageComplete(ctx, msg)
		}
	default:
		// Other events do not need aggregation.
		// Note this MUST include all events that are generated via aggregation, or we would infinite loop
	}
	l.Debugf("No action for %.10d/%s: %s", event.Sequence, event.ID, event.Type)
	return nil
}

func (ag *aggregator) processDataArrived(ctx context.Context, ns string, dataId *uuid.UUID) error {
	// Find all unconfirmed messages associated with this data
	fb := database.MessageQueryFactory.NewFilter(ctx)
	msgs, err := ag.database.GetMessagesForData(ctx, dataId, fb.And(
		fb.Eq("namespace", ns),
		fb.Eq("confirmed", nil),
	).Sort("sequence"))
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		if err = ag.checkMessageComplete(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (ag *aggregator) checkMessageComplete(ctx context.Context, msg *fftypes.Message) error {
	complete, err := ag.database.CheckDataAvailable(ctx, msg)
	if err != nil {
		return err // CheckDataAvailable should only return an error if there's a problem with persistence
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
			return err
		}
		if len(incomplete) > 0 {
			log.L(ctx).Infof("Message %s (seq=%d) blocked by incompleted message %s (seq=%d)", msg.Header.ID, msg.Sequence, incomplete[0].ID, incomplete[0].Sequence)
			return nil
		}

		// This message is now confirmed
		setConfirmed := database.MessageQueryFactory.NewUpdate(ctx).Set("confirmed", fftypes.Now())
		err = ag.database.UpdateMessage(ctx, msg.Header.ID, setConfirmed)
		if err != nil {
			return err
		}

		// Emit the confirmed event
		event := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, msg.Header.Namespace, msg.Header.ID)
		if err = ag.database.UpsertEvent(ctx, event, false); err != nil {
			return err
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
			return err
		}
		if len(unblockable) > 0 {
			log.L(ctx).Infof("Message %s (seq=%d) unblocks message %s (seq=%d)", msg.Header.ID, msg.Sequence, unblockable[0].ID, unblockable[0].Sequence)
			event := fftypes.NewEvent(fftypes.EventTypeMessagesUnblocked, msg.Header.Namespace, unblockable[0].ID)
			if err = ag.database.UpsertEvent(ctx, event, false); err != nil {
				return err
			}
		}

	}
	return nil
}

// newEventNotifications just consumes new events, logs them, then ensures there's a shoulderTap
// in the channel - without blocking. This is important as we must not block the notifier
// - which might be our own eventLoop
func (ag *aggregator) newEventNotifications() {
	l := log.L(ag.ctx).WithField("role", "aggregator-newevents")
	for {
		select {
		case m, ok := <-ag.newEvents:
			if !ok {
				l.Debugf("Exiting due to close")
				return
			}
			l.Debugf("Absorbing trigger for message %s", m)
		case <-ag.ctx.Done():
			l.Debugf("Exiting due to cancelled context")
			return
		}
		// Do not block sending to the shoulderTap - as it can only contain one
		select {
		case ag.shoulderTap <- true:
		default:
		}
	}
}

func (ag *aggregator) waitForShoulderTapOrPollTimeout() bool {
	l := log.L(ag.ctx)
	timeout := time.NewTimer(ag.eventPollTimeout)
	select {
	case <-timeout.C:
		l.Debugf("Woken after poll timeout")
	case <-ag.shoulderTap:
		l.Debug("Woken for trigger on event")
	case <-ag.ctx.Done():
		l.Debugf("Exiting due to cancelled context")
		return false
	}
	return true
}
