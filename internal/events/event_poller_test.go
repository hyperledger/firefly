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

package events

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger-labs/firefly/internal/retry"
	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestEventPoller(t *testing.T, mdi *databasemocks.Plugin, neh newEventsHandler, rewinder func() (bool, int64)) (ep *eventPoller, cancel func()) {
	ctx, cancel := context.WithCancel(context.Background())
	ep = newEventPoller(ctx, mdi, newEventNotifier(ctx, "ut"), &eventPollerConf{
		eventBatchSize:             10,
		eventBatchTimeout:          1 * time.Millisecond,
		eventPollTimeout:           10 * time.Second,
		startupOffsetRetryAttempts: 1,
		retry: retry.Retry{
			InitialDelay: 1 * time.Microsecond,
			MaximumDelay: 1 * time.Microsecond,
			Factor:       2.0,
		},
		newEventsHandler: neh,
		offsetType:       fftypes.OffsetTypeSubscription,
		offsetNamespace:  "unit",
		offsetName:       "test",
		queryFactory:     database.EventQueryFactory,
		getItems: func(c context.Context, f database.Filter) ([]fftypes.LocallySequenced, error) {
			events, err := mdi.GetEvents(c, f)
			ls := make([]fftypes.LocallySequenced, len(events))
			for i, e := range events {
				ls[i] = e
			}
			return ls, err
		},
		maybeRewind: rewinder,
		addCriteria: func(af database.AndFilter) database.AndFilter { return af },
	})
	return ep, cancel
}

func TestStartStopEventPoller(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil, nil)
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(&fftypes.Offset{
		Type:      fftypes.OffsetTypeAggregator,
		Namespace: fftypes.SystemNamespace,
		Name:      aggregatorOffsetName,
		Current:   12345,
	}, nil)
	mdi.On("GetEvents", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	ep.start()
	assert.Equal(t, int64(12345), ep.pollingOffset)
	ep.eventNotifier.newEvents <- 12345
	cancel()
	<-ep.closed
}

func TestRestoreOffsetNewestOK(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil, nil)
	defer cancel()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(nil, nil).Once()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(&fftypes.Offset{Current: 12345}, nil).Once()
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{{Sequence: 12345}}, nil)
	mdi.On("UpsertOffset", mock.Anything, mock.MatchedBy(func(offset *fftypes.Offset) bool {
		return offset.Current == 12345
	}), false).Return(nil)
	err := ep.restoreOffset()
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), ep.pollingOffset)
	mdi.AssertExpectations(t)
}

func TestRestoreOffsetNewestNoEvents(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil, nil)
	defer cancel()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(nil, nil).Once()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(&fftypes.Offset{Current: -1}, nil).Once()
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	mdi.On("UpsertOffset", mock.Anything, mock.MatchedBy(func(offset *fftypes.Offset) bool {
		return offset.Current == -1
	}), false).Return(nil)
	err := ep.restoreOffset()
	assert.NoError(t, err)
	assert.Equal(t, int64(-1), ep.pollingOffset)
	mdi.AssertExpectations(t)
}

func TestRestoreOffsetNewestFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil, nil)
	defer cancel()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(nil, nil)
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	err := ep.restoreOffset()
	assert.EqualError(t, err, "pop")
	assert.Equal(t, int64(0), ep.pollingOffset)
	mdi.AssertExpectations(t)
}

func TestRestoreOffsetOldest(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil, nil)
	firstEvent := fftypes.SubOptsFirstEventOldest
	ep.conf.firstEvent = &firstEvent
	defer cancel()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(nil, nil).Once()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(&fftypes.Offset{Current: -1}, nil).Once()
	mdi.On("UpsertOffset", mock.Anything, mock.MatchedBy(func(offset *fftypes.Offset) bool {
		return offset.Current == -1
	}), false).Return(nil)
	err := ep.restoreOffset()
	assert.NoError(t, err)
	assert.Equal(t, int64(-1), ep.pollingOffset)
	mdi.AssertExpectations(t)
}

func TestRestoreOffsetSpecific(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil, nil)
	firstEvent := fftypes.SubOptsFirstEvent("123456")
	ep.conf.firstEvent = &firstEvent
	defer cancel()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(nil, nil).Once()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(&fftypes.Offset{Current: 123456}, nil)
	mdi.On("UpsertOffset", mock.Anything, mock.MatchedBy(func(offset *fftypes.Offset) bool {
		return offset.Current == 123456
	}), false).Return(nil)
	err := ep.restoreOffset()
	assert.NoError(t, err)
	assert.Equal(t, int64(123456), ep.pollingOffset)
	mdi.AssertExpectations(t)
}

func TestRestoreOffsetFailRead(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil, nil)
	cancel() // to avoid infinite retry
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(nil, fmt.Errorf("pop"))
	ep.start()
	mdi.AssertExpectations(t)
}

func TestRestoreOffsetFailWrite(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil, nil)
	firstEvent := fftypes.SubOptsFirstEventOldest
	ep.conf.firstEvent = &firstEvent
	defer cancel()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(nil, nil)
	mdi.On("UpsertOffset", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	err := ep.restoreOffset()
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestRestoreOffsetEphemeral(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil, nil)
	firstEvent := fftypes.SubOptsFirstEventOldest
	ep.conf.firstEvent = &firstEvent
	ep.conf.ephemeral = true
	defer cancel()
	err := ep.restoreOffset()
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestReadPageExit(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil, nil)
	cancel()
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	ep.eventLoop()
	mdi.AssertExpectations(t)
}

func TestReadPageSingleCommitEvent(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	processEventCalled := make(chan fftypes.LocallySequenced, 1)
	ep, cancel := newTestEventPoller(t, mdi, func(events []fftypes.LocallySequenced) (bool, error) {
		processEventCalled <- events[0]
		return false, nil
	}, nil)
	cancel()
	ev1 := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, "ns1", fftypes.NewUUID())
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{ev1}, nil).Once()
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	ep.eventLoop()

	event := <-processEventCalled
	assert.Equal(t, *ev1.ID, *event.(*fftypes.Event).ID)
	mdi.AssertExpectations(t)
}

func TestReadPageRewind(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	processEventCalled := make(chan fftypes.LocallySequenced, 1)
	ep, cancel := newTestEventPoller(t, mdi, func(events []fftypes.LocallySequenced) (bool, error) {
		processEventCalled <- events[0]
		return false, nil
	}, func() (bool, int64) {
		return true, 12345
	})
	cancel()
	ev1 := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, "ns1", fftypes.NewUUID())
	mdi.On("GetEvents", mock.Anything, mock.MatchedBy(func(filter database.Filter) bool {
		f, err := filter.Finalize()
		assert.NoError(t, err)
		assert.Equal(t, "sequence", f.Children[0].Field)
		v, _ := f.Children[0].Value.Value()
		assert.Equal(t, int64(12345), v)
		return true
	})).Return([]*fftypes.Event{ev1}, nil).Once()
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	ep.eventLoop()

	event := <-processEventCalled
	assert.Equal(t, *ev1.ID, *event.(*fftypes.Event).ID)
	mdi.AssertExpectations(t)
}

func TestReadPageProcessEventsRetryExit(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, func(events []fftypes.LocallySequenced) (bool, error) { return false, fmt.Errorf("pop") }, nil)
	cancel()
	ev1 := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, "ns1", fftypes.NewUUID())
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{ev1}, nil).Once()
	ep.eventLoop()

	mdi.AssertExpectations(t)
}

func TestProcessEventsFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, func(events []fftypes.LocallySequenced) (bool, error) {
		return false, fmt.Errorf("pop")
	}, nil)
	defer cancel()
	_, err := ep.conf.newEventsHandler([]fftypes.LocallySequenced{
		fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, "ns1", fftypes.NewUUID()),
	})
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestWaitForShoulderTapOrExitCloseBatch(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil, nil)
	cancel()
	ep.conf.eventBatchTimeout = 1 * time.Minute
	ep.conf.eventBatchSize = 50
	assert.False(t, ep.waitForShoulderTapOrPollTimeout(1))
}

func TestWaitForShoulderTapOrExitClosePoll(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil, nil)
	cancel()
	ep.conf.eventBatchTimeout = 1 * time.Minute
	ep.conf.eventBatchSize = 1
	assert.False(t, ep.waitForShoulderTapOrPollTimeout(1))
}

func TestWaitForShoulderTapOrPollTimeoutBatchAndPoll(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil, nil)
	defer cancel()
	ep.conf.eventBatchTimeout = 1 * time.Microsecond
	ep.conf.eventPollTimeout = 1 * time.Microsecond
	ep.conf.eventBatchSize = 50
	assert.True(t, ep.waitForShoulderTapOrPollTimeout(1))
}

func TestWaitForShoulderTapOrPollTimeoutTap(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil, nil)
	defer cancel()
	ep.shoulderTap()
	assert.True(t, ep.waitForShoulderTapOrPollTimeout(ep.conf.eventBatchSize))
}

func TestDoubleTap(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil, nil)
	defer cancel()
	ep.shoulderTap()
	ep.shoulderTap() // this should not block
}
