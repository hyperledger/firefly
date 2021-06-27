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

package system

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/mocks/eventsmocks"
	"github.com/hyperledger-labs/firefly/pkg/events"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestEvents(t *testing.T) (ie *Events, cancel func()) {
	config.Reset()

	cbs := &eventsmocks.Callbacks{}
	rc := cbs.On("RegisterConnection", mock.Anything, mock.Anything).Return(nil)
	rc.RunFn = func(a mock.Arguments) {
		assert.Equal(t, true, a[1].(events.SubscriptionMatcher)(fftypes.SubscriptionRef{}))
	}
	ie = &Events{}
	ctx, cancelCtx := context.WithCancel(context.Background())
	svrPrefix := config.NewPluginConfig("ut.events")
	ie.InitPrefix(svrPrefix)
	ie.Init(ctx, svrPrefix, cbs)
	assert.Equal(t, "system", ie.Name())
	assert.NotNil(t, ie.Capabilities())
	assert.NotNil(t, ie.GetOptionsSchema(ie.ctx))
	assert.Nil(t, ie.ValidateOptions(&fftypes.SubscriptionOptions{}))
	return ie, cancelCtx
}

func TestDeliveryRequestOk(t *testing.T) {

	ie, cancel := newTestEvents(t)
	defer cancel()

	cbs := ie.callbacks.(*eventsmocks.Callbacks)
	cbs.On("EphemeralSubscription", mock.Anything, "ns1", mock.Anything, mock.Anything).Return(nil)

	called := 0
	err := ie.AddListener("ns1", func(event *fftypes.EventDelivery) error {
		called++
		return nil
	})
	assert.NoError(t, err)

	err = ie.DeliveryRequest(mock.Anything, &fftypes.Subscription{}, &fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns1",
		},
	}, nil)
	assert.NoError(t, err)

	err = ie.DeliveryRequest(mock.Anything, &fftypes.Subscription{}, &fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns2",
		},
	}, nil)
	assert.NoError(t, err)

	assert.Equal(t, 1, called)

}

func TestDeliveryRequestFail(t *testing.T) {

	ie, cancel := newTestEvents(t)
	defer cancel()

	cbs := ie.callbacks.(*eventsmocks.Callbacks)
	cbs.On("EphemeralSubscription", mock.Anything, "ns1", mock.Anything, mock.Anything).Return(nil)

	err := ie.AddListener("ns1", func(event *fftypes.EventDelivery) error {
		return fmt.Errorf("pop")
	})
	assert.NoError(t, err)

	err = ie.DeliveryRequest(mock.Anything, &fftypes.Subscription{}, &fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns1",
		},
	}, nil)
	assert.EqualError(t, err, "pop")

}

func TestAddListenerFail(t *testing.T) {

	ie, cancel := newTestEvents(t)
	defer cancel()

	cbs := ie.callbacks.(*eventsmocks.Callbacks)
	cbs.On("EphemeralSubscription", mock.Anything, "ns1", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := ie.AddListener("ns1", func(event *fftypes.EventDelivery) error { return nil })
	assert.EqualError(t, err, "pop")

}
