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
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/retry"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

type eventPoller struct {
	ctx            context.Context
	database       database.Plugin
	shoulderTaps   chan bool
	newEvents      chan *uuid.UUID
	closed         chan struct{}
	pollingOffset  int64
	comittedOffset int64
	mux            sync.Mutex
	conf           eventPollerConf
}

type newEventsHandler func(events []*fftypes.Event) (bool, error)

type eventPollerConf struct {
	ephemeral                  bool
	eventBatchSize             int
	eventBatchTimeout          time.Duration
	eventPollTimeout           time.Duration
	firstEvent                 fftypes.SubOptsFirstEvent
	limitNamespace             string
	newEventsHandler           newEventsHandler
	offsetName                 string
	offsetNamespace            string
	offsetType                 fftypes.OffsetType
	retry                      retry.Retry
	startupOffsetRetryAttempts int
}

func newEventPoller(ctx context.Context, di database.Plugin, conf eventPollerConf) *eventPoller {
	ep := &eventPoller{
		ctx:          log.WithLogField(ctx, "role", fmt.Sprintf("ep[%s:%s]", conf.offsetName, conf.offsetNamespace)),
		database:     di,
		shoulderTaps: make(chan bool, 1),
		newEvents:    make(chan *uuid.UUID),
		closed:       make(chan struct{}),
		conf:         conf,
	}
	return ep
}

func (ep *eventPoller) calcFirstOffset(ctx context.Context) (firstOffset int64, err error) {
	var useNewest bool
	switch ep.conf.firstEvent {
	case "", fftypes.SubOptsFirstEventNewest:
		useNewest = true
	case fftypes.SubOptsFirstEventOldest:
		useNewest = false
	default:
		specificSequence, err := strconv.ParseInt(string(ep.conf.firstEvent), 10, 64)
		if err == nil {
			firstOffset = specificSequence
			useNewest = false
		}
	}
	if useNewest {
		f := database.EventQueryFactory.NewFilter(ctx).And().Sort("sequence").Descending().Limit(1)
		newestEvents, err := ep.database.GetEvents(ctx, f)
		if err != nil {
			return 0, err
		}
		if len(newestEvents) > 0 {
			firstOffset = newestEvents[0].Sequence
		}
	}
	return
}

func (ep *eventPoller) restoreOffset() error {
	return ep.conf.retry.Do(ep.ctx, "restore offset", func(attempt int) (retry bool, err error) {
		var offset *fftypes.Offset
		if !ep.conf.ephemeral {
			offset, err = ep.database.GetOffset(ep.ctx, ep.conf.offsetType, ep.conf.offsetNamespace, ep.conf.offsetName)
		}
		if err == nil {
			if offset == nil {
				ep.pollingOffset, err = ep.calcFirstOffset(ep.ctx)
				if err == nil {
					err = ep.commitOffset(ep.ctx)
				}
			} else {
				ep.pollingOffset = offset.Current
			}
		}
		if err != nil {
			return (attempt <= ep.conf.startupOffsetRetryAttempts), err
		}
		log.L(ep.ctx).Infof("Event offset restored %d", ep.pollingOffset)
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

func (ep *eventPoller) rewindPollingOffset(offset int64) {
	log.L(ep.ctx).Infof("Event polling rewind to: %d", offset)
	ep.mux.Lock()
	defer ep.mux.Unlock()
	if offset < ep.pollingOffset {
		ep.pollingOffset = offset
	}
}

func (ep *eventPoller) commitOffset(ctx context.Context) error {
	l := log.L(ctx)
	// No persistence for ephemeral (non-durable) subscriptions
	if !ep.conf.ephemeral {
		offset := &fftypes.Offset{
			Type:      ep.conf.offsetType,
			Namespace: ep.conf.offsetNamespace,
			Name:      ep.conf.offsetName,
			Current:   ep.pollingOffset,
		}
		if err := ep.database.UpsertOffset(ctx, offset, true); err != nil {
			return err
		}
	}
	ep.comittedOffset = ep.pollingOffset
	l.Debugf("Event polling offset committed %d", ep.comittedOffset)
	return nil
}

func (ep *eventPoller) readPage() ([]*fftypes.Event, error) {
	var msgs []*fftypes.Event
	ep.mux.Lock()
	pollingOffset := ep.pollingOffset // ensure we pickup a rewound value
	ep.mux.Unlock()
	err := ep.conf.retry.Do(ep.ctx, "retrieve events", func(attempt int) (retry bool, err error) {
		fb := database.MessageQueryFactory.NewFilter(ep.ctx)
		filter := fb.Gte("sequence", pollingOffset)
		if ep.conf.limitNamespace != "" {
			filter = fb.And(filter, fb.Eq("namespace", ep.conf.limitNamespace))
		}
		msgs, err = ep.database.GetEvents(ep.ctx, filter.Sort("sequence").Limit(uint64(ep.conf.eventBatchSize)))
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
			// Update our polling offset for the next time round
			ep.pollingOffset = events[len(events)-1].Sequence

			// We process all the events in the page in a single database run group, and
			// keep retrying on all retryable errors, indefinitely ().
			var err error
			repoll, err = ep.dispatchEventsRetry(events)
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

func (ep *eventPoller) dispatchEventsRetry(events []*fftypes.Event) (repoll bool, err error) {
	err = ep.conf.retry.Do(ep.ctx, "process events", func(attempt int) (retry bool, err error) {
		repoll, err = ep.conf.newEventsHandler(events)
		return err != nil, err // always retry (retry will end on cancelled context)
	})
	return repoll, err
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
		ep.shoulderTap()
	}
}

func (ep *eventPoller) shoulderTap() {
	// Do not block sending to the shoulderTap - as it can only contain one
	select {
	case ep.shoulderTaps <- true:
	default:
	}
}

func (ep *eventPoller) waitForShoulderTapOrPollTimeout(lastEventCount int) bool {
	l := log.L(ep.ctx)
	longTimeoutDuration := ep.conf.eventPollTimeout
	// We avoid a tight spin with the eventBatchingTimeout to allow messages to arrive
	if lastEventCount < ep.conf.eventBatchSize {
		shortTimeout := time.NewTimer(ep.conf.eventBatchTimeout)
		select {
		case <-shortTimeout.C:
			l.Debugf("Woken after poll timeout")
		case <-ep.ctx.Done():
			l.Debugf("Exiting due to cancelled context")
			return false
		}
		longTimeoutDuration = longTimeoutDuration - ep.conf.eventBatchTimeout
	}

	longTimeout := time.NewTimer(longTimeoutDuration)
	select {
	case <-longTimeout.C:
		l.Debugf("Woken after poll timeout")
	case <-ep.shoulderTaps:
		l.Debug("Woken for trigger on event")
	case <-ep.ctx.Done():
		l.Debugf("Exiting due to cancelled context")
		return false
	}
	return true
}
