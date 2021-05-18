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
	if err := ag.database.UpsertOffset(ctx, offset); err != nil {
		return err
	}
	l.Infof("Event aggregator committed offset %d", newOffset)
	ag.offset = newOffset
	return nil
}

func (ag *aggregator) readPage() ([]*fftypes.Event, error) {
	var msgs []*fftypes.Event
	err := ag.retry.Do(ag.ctx, "retrieve events", func(attempt int) (retry bool, err error) {
		fb := database.MessageQueryFactory.NewFilter(ag.ctx, ag.readPageSize)
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
		}

		// Wait to be woken again
		if ok := ag.waitForShoulderTapOrPollTimeout(); !ok {
			return
		}
	}
}

func (ag *aggregator) processEventRetryAndGroup(events []*fftypes.Event) error {
	return ag.retry.Do(ag.ctx, "process events", func(attempt int) (retry bool, err error) {
		err = ag.database.RunAsGroup(ag.ctx, func(ctx context.Context) error {
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
		})
		return err != nil, err // always retry (retry will end on cancelled context)
	})
}

func (ag *aggregator) processEvent(ctx context.Context, event *fftypes.Event) error {
	l := log.L(ctx)
	l.Debugf("Aggregating event %-10d/%s: %s", event.Sequence, event.ID, event.Type)
	l.Tracef("Event data: %+v", event.Type)
	switch event.Type {
	case fftypes.EventTypeDataArrivedBroadcast:
		return ag.processDataArrived(ctx, event)
	case fftypes.EventTypeMessageSequencedBroadcast:
		return ag.processMessageSequenced(ctx, event)
	default:
		// Other events do not need aggregation.
		// Note this MUST include all events that are generated via aggregation, or we would infinite loop
		l.Debugf("No action for %-10d/%s: %s", event.Sequence, event.ID, event.Type)
		return nil
	}
}

func (ag *aggregator) processDataArrived(ctx context.Context, event *fftypes.Event) error {
	// Find all unconfirmed messages associated with this data
	return nil
}

func (ag *aggregator) processMessageSequenced(ctx context.Context, event *fftypes.Event) error {
	return nil
}

func (ag *aggregator) waitForShoulderTapOrPollTimeout() bool {
	l := log.L(ag.ctx)
	timeout := time.NewTimer(ag.eventPollTimeout)
	select {
	case <-timeout.C:
		l.Debugf("Woken after poll timeout")
	case m := <-ag.newEvents:
		l.Debugf("Woken for trigger for message %s", m)
	case <-ag.ctx.Done():
		l.Debugf("Exiting due to cancelled context")
		return false
	}
	var drained bool
	for !drained {
		select {
		case m := <-ag.newEvents:
			l.Debugf("Absorbing trigger for message %s", m)
		default:
			drained = true
		}
	}
	return true
}
