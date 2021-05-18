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

	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestShutdownOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	mdi := &databasemocks.Plugin{}
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeAggregator, fftypes.SystemNamespace, aggregatorOffsetName).Return(&fftypes.Offset{
		Type:      fftypes.OffsetTypeAggregator,
		Namespace: fftypes.SystemNamespace,
		Name:      aggregatorOffsetName,
		Current:   12345,
	}, nil)
	mdi.On("GetEvents", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	ag := newAggregator(ctx, mdi)
	err := ag.start()
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), ag.offset)
	ag.newEvents <- fftypes.NewUUID()
	cancel()
	<-ag.closed
}

func TestRestoreOffsetNew(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeAggregator, fftypes.SystemNamespace, aggregatorOffsetName).Return(nil, nil)
	mdi.On("UpsertOffset", mock.Anything, mock.Anything).Return(nil)
	err := ag.restoreOffset()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), ag.offset)
	mdi.AssertExpectations(t)
}

func TestRestoreOffsetFailRead(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	ag.startupOffsetRetryAttempts = 1
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeAggregator, fftypes.SystemNamespace, aggregatorOffsetName).Return(nil, fmt.Errorf("pop"))
	err := ag.start()
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestRestoreOffsetFailWrite(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeAggregator, fftypes.SystemNamespace, aggregatorOffsetName).Return(nil, nil)
	mdi.On("UpsertOffset", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	err := ag.restoreOffset()
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestReadPageExit(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ag := newAggregator(ctx, mdi)
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	ag.eventLoop()
	mdi.AssertExpectations(t)
}

func TestReadPageSingleCommitEvent(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ag := newAggregator(ctx, mdi)
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{
		fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, "ns1", fftypes.NewUUID()),
	}, nil).Once()
	mdi.On("RunAsGroup", mock.Anything, mock.Anything).Return(nil)
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	ag.eventLoop()

	dbFn := mdi.Calls[1].Arguments[1].(func(ctx context.Context) error)
	err := dbFn(ctx)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestReadPageTransactionRetryExit(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ag := newAggregator(ctx, mdi)
	mdi.On("GetEvents", mock.Anything, mock.Anything).Return([]*fftypes.Event{
		fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, "ns1", fftypes.NewUUID()),
	}, nil).Once()
	mdi.On("RunAsGroup", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	ag.eventLoop()
	mdi.AssertExpectations(t)
}

func TestProcessEventsFailToReadMessage(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	mdi.On("GetMessageById", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	err := ag.processEvents(context.Background(), []*fftypes.Event{
		fftypes.NewEvent(fftypes.EventTypeMessageSequencedBroadcast, "ns1", fftypes.NewUUID()),
	})
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestProcessEventsNoopIncrement(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	mdi.On("UpsertOffset", mock.Anything, mock.Anything).Return(nil, nil)
	ev1 := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, "ns1", fftypes.NewUUID())
	ev1.Sequence = 111
	ev2 := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, "ns1", fftypes.NewUUID())
	ev2.Sequence = 112
	ev3 := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, "ns1", fftypes.NewUUID())
	ev3.Sequence = 113
	err := ag.processEvents(context.Background(), []*fftypes.Event{
		ev1, ev2, ev3,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(113), ag.offset)
	mdi.AssertExpectations(t)
}

func TestProcessEventCheckCompleteDataNotAvailable(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mdi.On("GetMessageById", mock.Anything, mock.Anything).Return(msg, nil)
	mdi.On("CheckDataAvailable", mock.Anything, mock.Anything).Return(false, nil)
	ev1 := fftypes.NewEvent(fftypes.EventTypeMessageSequencedBroadcast, "ns1", fftypes.NewUUID())
	ev1.Sequence = 111
	err := ag.processEvent(context.Background(), ev1)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), ag.offset)
	mdi.AssertExpectations(t)
}

func TestProcessEventDataArrivedNoMsgs(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	mdi.On("GetMessagesForData", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Message{}, nil)
	ev1 := fftypes.NewEvent(fftypes.EventTypeDataArrivedBroadcast, "ns1", fftypes.NewUUID())
	ev1.Sequence = 111
	err := ag.processEvent(context.Background(), ev1)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), ag.offset)
	mdi.AssertExpectations(t)
}

func TestProcessDataArrivedError(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	mdi.On("GetMessagesForData", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	err := ag.processDataArrived(context.Background(), "ns1", fftypes.NewUUID())
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestProcessDataCompleteMessageBlocked(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mdi.On("GetMessagesForData", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Message{msg}, nil)
	mdi.On("CheckDataAvailable", mock.Anything, mock.Anything).Return(true, nil)
	mdi.On("GetMessageRefs", mock.Anything, mock.Anything).Return([]*fftypes.MessageRef{{ID: fftypes.NewUUID(), Sequence: 111}}, nil)
	err := ag.processDataArrived(context.Background(), "ns1", fftypes.NewUUID())
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestProcessDataCompleteQueryBlockedFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mdi.On("GetMessagesForData", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Message{msg}, nil)
	mdi.On("CheckDataAvailable", mock.Anything, mock.Anything).Return(true, nil)
	mdi.On("GetMessageRefs", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	err := ag.processDataArrived(context.Background(), "ns1", fftypes.NewUUID())
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestCheckMessageCompleteDataAvailFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mdi.On("CheckDataAvailable", mock.Anything, mock.Anything).Return(false, fmt.Errorf("pop"))
	err := ag.checkMessageComplete(context.Background(), msg)
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestCheckMessageCompleteUpdateFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mdi.On("CheckDataAvailable", mock.Anything, mock.Anything).Return(true, nil)
	mdi.On("GetMessageRefs", mock.Anything, mock.Anything).Return([]*fftypes.MessageRef{}, nil)
	mdi.On("UpdateMessage", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	err := ag.checkMessageComplete(context.Background(), msg)
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestCheckMessageCompleteInsertEventFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mdi.On("CheckDataAvailable", mock.Anything, mock.Anything).Return(true, nil)
	mdi.On("GetMessageRefs", mock.Anything, mock.Anything).Return([]*fftypes.MessageRef{}, nil)
	mdi.On("UpdateMessage", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertEvent", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	err := ag.checkMessageComplete(context.Background(), msg)
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestCheckMessageCompleteGetUnblockedFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mdi.On("CheckDataAvailable", mock.Anything, mock.Anything).Return(true, nil)
	mdi.On("GetMessageRefs", mock.Anything, mock.Anything).Return([]*fftypes.MessageRef{}, nil).Once()
	mdi.On("UpdateMessage", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertEvent", mock.Anything, mock.Anything).Return(nil)
	mdi.On("GetMessageRefs", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	err := ag.checkMessageComplete(context.Background(), msg)
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestCheckMessageCompleteInsertUnblockEventFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mdi.On("CheckDataAvailable", mock.Anything, mock.Anything).Return(true, nil)
	mdi.On("GetMessageRefs", mock.Anything, mock.Anything).Return([]*fftypes.MessageRef{}, nil).Once()
	mdi.On("UpdateMessage", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertEvent", mock.Anything, mock.Anything).Return(nil).Once()
	mdi.On("GetMessageRefs", mock.Anything, mock.Anything).Return([]*fftypes.MessageRef{
		{ID: fftypes.NewUUID(), Sequence: 111},
	}, nil)
	mdi.On("UpsertEvent", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	err := ag.checkMessageComplete(context.Background(), msg)
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)
}

func TestCheckMessageCompleteInsertUnblockOK(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	mdi.On("CheckDataAvailable", mock.Anything, mock.Anything).Return(true, nil)
	mdi.On("GetMessageRefs", mock.Anything, mock.Anything).Return([]*fftypes.MessageRef{}, nil).Once()
	mdi.On("UpdateMessage", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertEvent", mock.Anything, mock.Anything).Return(nil).Once()
	mdi.On("GetMessageRefs", mock.Anything, mock.Anything).Return([]*fftypes.MessageRef{
		{ID: fftypes.NewUUID(), Sequence: 111},
	}, nil)
	mdi.On("UpsertEvent", mock.Anything, mock.Anything).Return(nil)
	err := ag.checkMessageComplete(context.Background(), msg)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestNewEventNotificationsExitOnClose(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	close(ag.newEvents)
	ag.newEventNotifications()
}

func TestWaitForShoulderTapOrPollTimeout(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ag := newAggregator(context.Background(), mdi)
	ag.eventPollTimeout = 1 * time.Microsecond
	assert.True(t, ag.waitForShoulderTapOrPollTimeout())
}
