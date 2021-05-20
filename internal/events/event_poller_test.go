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
	"testing"
	"time"

	"github.com/kaleido-io/firefly/internal/retry"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestEventPoller(t *testing.T, mdi *databasemocks.Plugin, neh newEventsHandler) (ep *eventPoller, cancel func()) {
	ctx, cancel := context.WithCancel(context.Background())
	ep = newEventPoller(ctx, mdi, eventPollerConf{
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
	})
	return ep, cancel
}

func TestStartStopEventPoller(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil)
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(&fftypes.Offset{
		Type:      fftypes.OffsetTypeAggregator,
		Namespace: fftypes.SystemNamespace,
		Name:      aggregatorOffsetName,
		Current:   12345,
	}, nil)
	mdi.On("GetEvents", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	err := ep.start()
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), ep.pollingOffset)
	ep.newEvents <- fftypes.NewUUID()
	cancel()
	<-ep.closed
}

func TestRestoreOffsetNewestOK(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil)
	defer cancel()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(nil, nil)
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{
		{Sequence: 12345},
	}, nil)
	mdi.On("UpsertOffset", mock.Anything, mock.Anything, true).Return(nil)
	err := ep.restoreOffset()
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), ep.pollingOffset)
	mdi.AssertExpectations(t)
}

func TestRestoreOffsetNewestNoEvents(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil)
	defer cancel()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(nil, nil)
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	mdi.On("UpsertOffset", mock.Anything, mock.Anything, true).Return(nil)
	err := ep.restoreOffset()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), ep.pollingOffset)
	mdi.AssertExpectations(t)
}

func TestRestoreOffsetNewestFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil)
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
	ep, cancel := newTestEventPoller(t, mdi, nil)
	ep.conf.firstEvent = fftypes.SubOptsFirstEventOldest
	defer cancel()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(nil, nil)
	mdi.On("UpsertOffset", mock.Anything, mock.Anything, true).Return(nil)
	err := ep.restoreOffset()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), ep.pollingOffset)
	mdi.AssertExpectations(t)
}

func TestRestoreOffsetSpecific(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil)
	ep.conf.firstEvent = fftypes.SubOptsFirstEvent("123456")
	defer cancel()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(nil, nil)
	mdi.On("UpsertOffset", mock.Anything, mock.Anything, true).Return(nil)
	err := ep.restoreOffset()
	assert.NoError(t, err)
	assert.Equal(t, int64(123456), ep.pollingOffset)
	mdi.AssertExpectations(t)
}

func TestRestoreOffsetFailRead(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil)
	defer cancel()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(nil, fmt.Errorf("pop"))
	err := ep.start()
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestRestoreOffsetFailWrite(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil)
	ep.conf.firstEvent = fftypes.SubOptsFirstEventOldest
	defer cancel()
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeSubscription, "unit", "test").Return(nil, nil)
	mdi.On("UpsertOffset", mock.Anything, mock.Anything, true).Return(fmt.Errorf("pop"))
	err := ep.restoreOffset()
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestRestoreOffsetEphemeral(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil)
	ep.conf.firstEvent = fftypes.SubOptsFirstEventOldest
	ep.conf.ephemeral = true
	defer cancel()
	err := ep.restoreOffset()
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestReadPageExit(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil)
	cancel()
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	ep.eventLoop()
	mdi.AssertExpectations(t)
}

func TestReadPageSingleCommitEvent(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	processEventCalled := make(chan *fftypes.Event, 1)
	ep, cancel := newTestEventPoller(t, mdi, func(events []*fftypes.Event) (bool, error) {
		processEventCalled <- events[0]
		return false, nil
	})
	cancel()
	ev1 := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, "ns1", fftypes.NewUUID())
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{ev1}, nil).Once()
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	ep.eventLoop()

	event := <-processEventCalled
	assert.Equal(t, *ev1.ID, *event.ID)
	mdi.AssertExpectations(t)
}

func TestReadPageProcessEventsRetryExit(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, func(events []*fftypes.Event) (bool, error) { return false, fmt.Errorf("pop") })
	cancel()
	ev1 := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, "ns1", fftypes.NewUUID())
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{ev1}, nil).Once()
	ep.eventLoop()

	mdi.AssertExpectations(t)
}

func TestProcessEventsFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, func(events []*fftypes.Event) (bool, error) {
		return false, fmt.Errorf("pop")
	})
	defer cancel()
	_, err := ep.conf.newEventsHandler([]*fftypes.Event{
		fftypes.NewEvent(fftypes.EventTypeMessageSequencedBroadcast, "ns1", fftypes.NewUUID()),
	})
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestNewEventNotificationsExitOnClose(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil)
	defer cancel()
	close(ep.newEvents)
	ep.newEventNotifications()
}

func TestWaitForShoulderTapOrExitCloseBatch(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil)
	cancel()
	ep.conf.eventBatchTimeout = 1 * time.Minute
	assert.False(t, ep.waitForShoulderTapOrPollTimeout(0))
}

func TestWaitForShoulderTapOrExitClosePoll(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil)
	cancel()
	ep.conf.eventBatchTimeout = 1 * time.Minute
	ep.conf.eventBatchSize = 1
	assert.False(t, ep.waitForShoulderTapOrPollTimeout(1))
}

func TestWaitForShoulderTapOrPollTimeoutBatchAndPoll(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil)
	defer cancel()
	ep.conf.eventBatchTimeout = 1 * time.Microsecond
	ep.conf.eventPollTimeout = 1 * time.Microsecond
	assert.True(t, ep.waitForShoulderTapOrPollTimeout(0))
}

func TestWaitForShoulderTapOrPollTimeoutTap(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil)
	defer cancel()
	ep.shoulderTap()
	assert.True(t, ep.waitForShoulderTapOrPollTimeout(ep.conf.eventBatchSize))
}

func TestDoubleTap(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ep, cancel := newTestEventPoller(t, mdi, nil)
	defer cancel()
	ep.shoulderTap()
	ep.shoulderTap() // this should not block
}
