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
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/retry"
)

type eventPoller struct {
	ctx         context.Context
	database    database.Plugin
	shoulderTap chan bool
	newEvents   chan *uuid.UUID
	closed      chan struct{}
	offset      int64
	conf        eventPollerConf
}

type eventHandler func(ctx context.Context, event *fftypes.Event) (bool, error)

type eventPollerConf struct {
	eventBatchSize             int
	eventBatchTimeout          time.Duration
	eventPollTimeout           time.Duration
	startupOffsetRetryAttempts int
	retry                      retry.Retry
	processEvent               eventHandler
	offsetType                 fftypes.OffsetType
	offsetNamespace            string
	offsetName                 string
}

func newEventPoller(ctx context.Context, di database.Plugin, conf eventPollerConf) *eventPoller {
	ep := &eventPoller{
		ctx:         log.WithLogField(ctx, "role", fmt.Sprintf("events:%s:%s", conf.offsetName, conf.offsetNamespace)),
		database:    di,
		shoulderTap: make(chan bool, 1),
		newEvents:   make(chan *uuid.UUID),
		closed:      make(chan struct{}),
		conf:        conf,
	}
	return ep
}

func (ep *eventPoller) restoreOffset() error {
	return ep.conf.retry.Do(ep.ctx, "restore offset", func(attempt int) (retry bool, err error) {
		offset, err := ep.database.GetOffset(ep.ctx, ep.conf.offsetType, ep.conf.offsetNamespace, ep.conf.offsetName)
		if err == nil {
			if offset == nil {
				err = ep.updateOffset(ep.ctx, 0)
			} else {
				ep.offset = offset.Current
			}
		}
		if err != nil {
			return (attempt <= ep.conf.startupOffsetRetryAttempts), err
		}
		log.L(ep.ctx).Infof("Event offset restored %d", ep.offset)
		return false, nil
	})
}

func (ep *eventPoller) start() error {
	if err := ep.restoreOffset(); err != nil {
		return err
	}
	go ep.newEventNotifications()
	go ep.eventLoop()
	return nil
}

func (ep *eventPoller) updateOffset(ctx context.Context, newOffset int64) error {
	l := log.L(ctx)
	ep.offset = newOffset
	offset := &fftypes.Offset{
		Type:      ep.conf.offsetType,
		Namespace: ep.conf.offsetNamespace,
		Name:      ep.conf.offsetName,
		Current:   ep.offset,
	}
	if err := ep.database.UpsertOffset(ctx, offset, true); err != nil {
		return err
	}
	l.Infof("Event offset committed %d", newOffset)
	ep.offset = newOffset
	return nil
}

func (ep *eventPoller) readPage() ([]*fftypes.Event, error) {
	var msgs []*fftypes.Event
	err := ep.conf.retry.Do(ep.ctx, "retrieve events", func(attempt int) (retry bool, err error) {
		fb := database.MessageQueryFactory.NewFilter(ep.ctx)
		msgs, err = ep.database.GetEvents(ep.ctx, fb.Gt("sequence", ep.offset).Sort("sequence").Limit(uint64(ep.conf.eventBatchSize)))
		if err != nil {
			return true, err // Retry indefinitely, until context cancelled
		}
		return false, nil
	})
	return msgs, err
}

func (ep *eventPoller) eventLoop() {
	l := log.L(ep.ctx)
	l.Debugf("Started event detector")
	defer close(ep.closed)

	for {
		// Read messages from the DB - in an error condition we retry until success, or a closed context
		events, err := ep.readPage()
		if err != nil {
			l.Debugf("Exiting: %s", err)
			return
		}

		eventCount := len(events)
		repoll := false
		if eventCount > 0 {
			// We process all the events in the page in a single database run group, and
			// keep retrying on all retryable errors, indefinitely ().
			var err error
			repoll, err = ep.processEventRetryAndGroup(events)
			if err != nil {
				l.Debugf("Exiting: %s", err)
				return
			}
		}

		// Once we run out of events, wait to be woken
		if !repoll {
			if ok := ep.waitForShoulderTapOrPollTimeout(eventCount); !ok {
				return
			}
		}
	}
}

func (ep *eventPoller) processEventRetryAndGroup(events []*fftypes.Event) (repoll bool, err error) {
	err = ep.conf.retry.Do(ep.ctx, "process events", func(attempt int) (retry bool, err error) {
		err = ep.database.RunAsGroup(ep.ctx, func(ctx context.Context) (err error) {
			repoll, err = ep.processEvents(ctx, events)
			return err
		})
		return err != nil, err // always retry (retry will end on cancelled context)
	})
	return repoll, err
}

func (ep *eventPoller) processEvents(ctx context.Context, events []*fftypes.Event) (repoll bool, err error) {
	var highestSequence int64
	for _, event := range events {
		repoll, err = ep.conf.processEvent(ctx, event)
		if err != nil {
			return false, err
		}
		if event.Sequence > highestSequence {
			highestSequence = event.Sequence
		}
	}
	if highestSequence > 0 {
		err = ep.updateOffset(ctx, highestSequence)
		return repoll, err
	}
	return repoll, nil
}

// newEventNotifications just consumes new events, logs them, then ensures there's a shoulderTap
// in the channel - without blocking. This is important as we must not block the notifier
// - which might be our own eventLoop
func (ep *eventPoller) newEventNotifications() {
	l := log.L(ep.ctx).WithField("role", "eventPoller-newevents")
	for {
		select {
		case m, ok := <-ep.newEvents:
			if !ok {
				l.Debugf("Exiting due to close")
				return
			}
			l.Debugf("Absorbing trigger for message %s", m)
		case <-ep.ctx.Done():
			l.Debugf("Exiting due to cancelled context")
			return
		}
		// Do not block sending to the shoulderTap - as it can only contain one
		select {
		case ep.shoulderTap <- true:
		default:
		}
	}
}

func (ep *eventPoller) waitForShoulderTapOrPollTimeout(lastEventCount int) bool {
	l := log.L(ep.ctx)
	longTimeout := ep.conf.eventPollTimeout
	// We avoid a tight spin with the eventBatchingTimeout to allow messages to arrive
	if lastEventCount < ep.conf.eventBatchSize {
		time.Sleep(ep.conf.eventBatchTimeout)
		longTimeout = longTimeout - ep.conf.eventBatchTimeout
	}

	timeout := time.NewTimer(longTimeout)
	select {
	case <-timeout.C:
		l.Debugf("Woken after poll timeout")
	case <-ep.shoulderTap:
		l.Debug("Woken for trigger on event")
	case <-ep.ctx.Done():
		l.Debugf("Exiting due to cancelled context")
		return false
	}
	return true
}
