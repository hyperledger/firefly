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
	"testing"

	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestEventDispatcher(mdi database.Plugin) *eventDispatcher {
	ten := uint64(10)
	secs15, _ := fftypes.ParseDurationString("15s")
	return newEventDispatcher(context.Background(), mdi, &fftypes.Subscription{
		Namespace: "ns1",
		Name:      "sub1",
		Ephemeral: true,
		Options: fftypes.SubscriptionOptions{
			BatchSize:    &ten,
			BatchTimeout: &secs15,
		},
	})
}

func TestEventDispatcherStartStop(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ed := newTestEventDispatcher(mdi)
	mdi.On("GetEvents", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	assert.Equal(t, int(10), ed.eventPoller.conf.eventBatchSize)
	assert.Equal(t, fftypes.ParseToDuration("15s"), ed.eventPoller.conf.eventBatchTimeout)
	err := ed.start()
	assert.NoError(t, err)
	ed.eventPoller.newEvents <- fftypes.NewUUID()
	ed.close()
}

func TestProcessEventNoop(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	ed := newTestEventDispatcher(mdi)
	ed.processEvent(context.Background(), fftypes.NewEvent(fftypes.EventTypeDataArrivedBroadcast, "ns1", fftypes.NewUUID()))
}
