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

package websockets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/config/wsconfig"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/restclient"
	"github.com/hyperledger/firefly/mocks/eventsmocks"
	"github.com/hyperledger/firefly/pkg/events"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/wsclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestWebsockets(t *testing.T, cbs *eventsmocks.Callbacks, queryParams ...string) (ws *WebSockets, wsc wsclient.WSClient, cancel func()) {
	config.Reset()

	ws = &WebSockets{}
	ctx, cancelCtx := context.WithCancel(context.Background())
	svrPrefix := config.NewPluginConfig("ut.websockets")
	ws.InitPrefix(svrPrefix)
	ws.Init(ctx, svrPrefix, cbs)
	assert.Equal(t, "websockets", ws.Name())
	assert.NotNil(t, ws.Capabilities())
	assert.NotNil(t, ws.GetOptionsSchema(context.Background()))
	cbs.On("ConnnectionClosed", mock.Anything).Return(nil).Maybe()

	svr := httptest.NewServer(ws)

	clientPrefix := config.NewPluginConfig("ut.wsclient")
	wsconfig.InitPrefix(clientPrefix)
	qs := ""
	if len(queryParams) > 0 {
		qs = fmt.Sprintf("?%s", strings.Join(queryParams, "&"))
	}
	clientPrefix.Set(restclient.HTTPConfigURL, fmt.Sprintf("http://%s%s", svr.Listener.Addr(), qs))
	wsConfig := wsconfig.GenerateConfigFromPrefix(clientPrefix)

	wsc, err := wsclient.New(ctx, wsConfig, nil)
	assert.NoError(t, err)
	err = wsc.Connect()
	assert.NoError(t, err)

	var wsi interface{} = ws
	_, ok := wsi.(events.PluginAll)
	assert.True(t, ok)

	return ws, wsc, func() {
		cancelCtx()
		wsc.Close()
		ws.WaitClosed()
		svr.Close()
	}
}

func TestValidateOptionsFail(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	ws, _, cancel := newTestWebsockets(t, cbs)
	defer cancel()

	yes := true
	err := ws.ValidateOptions(&fftypes.SubscriptionOptions{
		SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
			WithData: &yes,
		},
	})
	assert.Regexp(t, "FF10244", err)
}

func TestValidateOptionsOk(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	ws, _, cancel := newTestWebsockets(t, cbs)
	defer cancel()

	opts := &fftypes.SubscriptionOptions{}
	err := ws.ValidateOptions(opts)
	assert.NoError(t, err)
	assert.False(t, *opts.WithData)
}

func TestSendBadData(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	_, wsc, cancel := newTestWebsockets(t, cbs)
	defer cancel()

	cbs.On("ConnnectionClosed", mock.Anything).Return(nil)

	err := wsc.Send(context.Background(), []byte(`!json`))
	assert.NoError(t, err)
	b := <-wsc.Receive()
	var res fftypes.WSProtocolErrorPayload
	err = json.Unmarshal(b, &res)
	assert.NoError(t, err)
	assert.Equal(t, fftypes.WSProtocolErrorEventType, res.Type)
	assert.Regexp(t, "FF10176", res.Error)
}

func TestSendBadAction(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	_, wsc, cancel := newTestWebsockets(t, cbs)
	defer cancel()
	cbs.On("ConnnectionClosed", mock.Anything).Return(nil)

	err := wsc.Send(context.Background(), []byte(`{"type":"lobster"}`))
	assert.NoError(t, err)
	b := <-wsc.Receive()
	var res fftypes.WSProtocolErrorPayload
	err = json.Unmarshal(b, &res)
	assert.NoError(t, err)
	assert.Equal(t, fftypes.WSProtocolErrorEventType, res.Type)
	assert.Regexp(t, "FF10176", res.Error)
}

func TestSendEmptyStartAction(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	_, wsc, cancel := newTestWebsockets(t, cbs)
	defer cancel()
	cbs.On("ConnnectionClosed", mock.Anything).Return(nil)

	err := wsc.Send(context.Background(), []byte(`{"type":"start"}`))
	assert.NoError(t, err)
	b := <-wsc.Receive()
	var res fftypes.WSProtocolErrorPayload
	err = json.Unmarshal(b, &res)
	assert.NoError(t, err)
	assert.Equal(t, fftypes.WSProtocolErrorEventType, res.Type)
	assert.Regexp(t, "FF10176", res.Error)
}

func TestStartReceiveAckEphemeral(t *testing.T) {
	log.SetLevel("trace")

	cbs := &eventsmocks.Callbacks{}
	ws, wsc, cancel := newTestWebsockets(t, cbs)
	defer cancel()
	var connID string
	sub := cbs.On("EphemeralSubscription",
		mock.MatchedBy(func(s string) bool { connID = s; return true }),
		"ns1", mock.Anything, mock.Anything).Return(nil)
	ack := cbs.On("DeliveryResponse",
		mock.MatchedBy(func(s string) bool { return s == connID }),
		mock.Anything).Return(nil)

	waitSubscribed := make(chan struct{})
	sub.RunFn = func(a mock.Arguments) {
		close(waitSubscribed)
	}

	waitAcked := make(chan struct{})
	ack.RunFn = func(a mock.Arguments) {
		close(waitAcked)
	}

	err := wsc.Send(context.Background(), []byte(`{"type":"start","namespace":"ns1","ephemeral":true}`))
	assert.NoError(t, err)

	<-waitSubscribed
	ws.DeliveryRequest(connID, nil, &fftypes.EventDelivery{
		Event:        fftypes.Event{ID: fftypes.NewUUID()},
		Subscription: fftypes.SubscriptionRef{ID: fftypes.NewUUID()},
	}, nil)

	b := <-wsc.Receive()
	var res fftypes.EventDelivery
	err = json.Unmarshal(b, &res)
	assert.NoError(t, err)

	err = wsc.Send(context.Background(), []byte(`{"type":"ack"}`))
	assert.NoError(t, err)

	<-waitAcked
	cbs.AssertExpectations(t)
}

func TestStartReceiveDurable(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	ws, wsc, cancel := newTestWebsockets(t, cbs)
	defer cancel()
	var connID string
	sub := cbs.On("RegisterConnection",
		mock.MatchedBy(func(s string) bool { connID = s; return true }),
		mock.MatchedBy(func(subMatch events.SubscriptionMatcher) bool {
			return subMatch(fftypes.SubscriptionRef{Namespace: "ns1", Name: "sub1"}) &&
				!subMatch(fftypes.SubscriptionRef{Namespace: "ns2", Name: "sub1"}) &&
				!subMatch(fftypes.SubscriptionRef{Namespace: "ns1", Name: "sub2"})
		}),
	).Return(nil)
	ack := cbs.On("DeliveryResponse",
		mock.MatchedBy(func(s string) bool { return s == connID }),
		mock.Anything).Return(nil)

	waitSubscribed := make(chan struct{})
	sub.RunFn = func(a mock.Arguments) {
		close(waitSubscribed)
	}

	waitAcked := make(chan struct{})
	ack.RunFn = func(a mock.Arguments) {
		close(waitAcked)
	}

	err := wsc.Send(context.Background(), []byte(`{"type":"start","namespace":"ns1","name":"sub1"}`))
	assert.NoError(t, err)

	<-waitSubscribed
	ws.DeliveryRequest(connID, nil, &fftypes.EventDelivery{
		Event: fftypes.Event{ID: fftypes.NewUUID()},
		Subscription: fftypes.SubscriptionRef{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "sub1",
		},
	}, nil)
	// Put a second in flight
	ws.DeliveryRequest(connID, nil, &fftypes.EventDelivery{
		Event: fftypes.Event{ID: fftypes.NewUUID()},
		Subscription: fftypes.SubscriptionRef{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "sub2",
		},
	}, nil)

	b := <-wsc.Receive()
	var res fftypes.EventDelivery
	err = json.Unmarshal(b, &res)
	assert.NoError(t, err)

	assert.Equal(t, "ns1", res.Subscription.Namespace)
	assert.Equal(t, "sub1", res.Subscription.Name)
	err = wsc.Send(context.Background(), []byte(fmt.Sprintf(`{
		"type":"ack",
		"id": "%s",
		"subscription": {
			"namespace": "ns1",
			"name": "sub1"
		}
	}`, res.ID)))
	assert.NoError(t, err)

	<-waitAcked

	// Check we left the right one behind
	conn := ws.connections[connID]
	assert.Equal(t, 1, len(conn.inflight))
	assert.Equal(t, "sub2", conn.inflight[0].Subscription.Name)

	cbs.AssertExpectations(t)
}

func TestAutoStartReceiveAckEphemeral(t *testing.T) {
	var connID string
	cbs := &eventsmocks.Callbacks{}
	sub := cbs.On("EphemeralSubscription",
		mock.MatchedBy(func(s string) bool { connID = s; return true }),
		"ns1", mock.Anything, mock.Anything).Return(nil)
	ack := cbs.On("DeliveryResponse",
		mock.MatchedBy(func(s string) bool { return s == connID }),
		mock.Anything).Return(nil)

	waitSubscribed := make(chan struct{})
	sub.RunFn = func(a mock.Arguments) {
		close(waitSubscribed)
	}

	waitAcked := make(chan struct{})
	ack.RunFn = func(a mock.Arguments) {
		close(waitAcked)
	}

	ws, wsc, cancel := newTestWebsockets(t, cbs, "ephemeral", "namespace=ns1")
	defer cancel()

	<-waitSubscribed
	ws.DeliveryRequest(connID, nil, &fftypes.EventDelivery{
		Event:        fftypes.Event{ID: fftypes.NewUUID()},
		Subscription: fftypes.SubscriptionRef{ID: fftypes.NewUUID()},
	}, nil)

	b := <-wsc.Receive()
	var res fftypes.EventDelivery
	err := json.Unmarshal(b, &res)
	assert.NoError(t, err)

	err = wsc.Send(context.Background(), []byte(`{"type":"ack"}`))
	assert.NoError(t, err)

	<-waitAcked
	cbs.AssertExpectations(t)
}

func TestAutoStartBadOptions(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	_, wsc, cancel := newTestWebsockets(t, cbs, "name=missingnamespace")
	defer cancel()

	b := <-wsc.Receive()
	var res fftypes.WSProtocolErrorPayload
	err := json.Unmarshal(b, &res)
	assert.NoError(t, err)
	assert.Regexp(t, "FF10178", res.Error)
	cbs.AssertExpectations(t)
}

func TestHandleAckWithAutoAck(t *testing.T) {
	eventUUID := fftypes.NewUUID()
	wsc := &websocketConnection{
		ctx:          context.Background(),
		started:      []*websocketStartedSub{{ephemeral: false, name: "name1", namespace: "ns1"}},
		sendMessages: make(chan interface{}, 1),
		inflight: []*fftypes.EventDeliveryResponse{
			{ID: eventUUID},
		},
		autoAck: true,
	}
	err := wsc.handleAck(&fftypes.WSClientActionAckPayload{
		ID: eventUUID,
	})
	assert.Regexp(t, "FF10180", err)
}

func TestHandleStartFlippingAutoAck(t *testing.T) {
	eventUUID := fftypes.NewUUID()
	wsc := &websocketConnection{
		ctx:          context.Background(),
		started:      []*websocketStartedSub{{ephemeral: false, name: "name1", namespace: "ns1"}},
		sendMessages: make(chan interface{}, 1),
		inflight: []*fftypes.EventDeliveryResponse{
			{ID: eventUUID},
		},
		autoAck: true,
	}
	no := false
	err := wsc.handleStart(&fftypes.WSClientActionStartPayload{
		AutoAck: &no,
	})
	assert.Regexp(t, "FF10179", err)
}

func TestHandleStartWithChangeEvents(t *testing.T) {
	mcb := &eventsmocks.Callbacks{}
	wsc := &websocketConnection{
		ctx:          context.Background(),
		connID:       "conn1",
		started:      []*websocketStartedSub{{ephemeral: false, name: "name1", namespace: "ns1"}},
		sendMessages: make(chan interface{}, 1),
		ws: &WebSockets{
			callbacks: mcb,
		},
	}
	wsc.ws.connections = map[string]*websocketConnection{
		"conn1": wsc,
	}
	mcb.On("EphemeralSubscription", "conn1", "ns1", mock.Anything, mock.MatchedBy(func(opts *fftypes.SubscriptionOptions) bool {
		return opts.ChangeEvents // expect change events to be enabled
	})).Return(nil)
	err := wsc.handleStart(&fftypes.WSClientActionStartPayload{
		Namespace:    "ns1",
		Ephemeral:    true,
		ChangeEvents: ".*",
	})
	assert.NoError(t, err)

	wsc.ws.ChangeEvent("conn1", &fftypes.ChangeEvent{
		Collection: "ut1",
	})
	wscn := (<-wsc.sendMessages).(*fftypes.WSChangeNotification)
	assert.Equal(t, fftypes.WSClientActionChangeNotifcation, wscn.Type)
	assert.Equal(t, "ut1", wscn.ChangeEvent.Collection)

	mcb.AssertExpectations(t)
}

func TestHandleChangeEventsDispatchFail(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already closed
	mcb := &eventsmocks.Callbacks{}
	wsc := &websocketConnection{
		ctx:          ctx,
		connID:       "conn1",
		started:      []*websocketStartedSub{{ephemeral: false, name: "name1", namespace: "ns1"}},
		sendMessages: make(chan interface{}), // wil block
		ws: &WebSockets{
			ctx:       ctx,
			callbacks: mcb,
		},
	}
	wsc.ws.connections = map[string]*websocketConnection{
		"conn1": wsc,
	}
	mcb.On("EphemeralSubscription", "conn1", "ns1", mock.Anything, mock.MatchedBy(func(opts *fftypes.SubscriptionOptions) bool {
		return opts.ChangeEvents // expect change events to be enabled
	})).Return(nil)
	err := wsc.handleStart(&fftypes.WSClientActionStartPayload{
		Namespace:    "ns1",
		Ephemeral:    true,
		ChangeEvents: ".*",
	})
	assert.NoError(t, err)

	wsc.ws.ChangeEvent("conn1", &fftypes.ChangeEvent{
		Collection: "ut1",
	})
	mcb.AssertExpectations(t)
}

func TestHandleStartWithBadChangeEventsRegex(t *testing.T) {
	eventUUID := fftypes.NewUUID()
	mcb := &eventsmocks.Callbacks{}
	wsc := &websocketConnection{
		ctx:          context.Background(),
		connID:       "conn1",
		started:      []*websocketStartedSub{{ephemeral: false, name: "name1", namespace: "ns1"}},
		sendMessages: make(chan interface{}, 1),
		inflight: []*fftypes.EventDeliveryResponse{
			{ID: eventUUID},
		},
		autoAck: true,
		ws: &WebSockets{
			callbacks: mcb,
		},
	}
	mcb.On("EphemeralSubscription", "conn1", "ns1", mock.Anything, mock.MatchedBy(func(opts *fftypes.SubscriptionOptions) bool {
		return !opts.ChangeEvents // expect change events to be disabled due to bad regex
	})).Return(nil)
	err := wsc.handleStart(&fftypes.WSClientActionStartPayload{
		Namespace:    "ns1",
		Ephemeral:    true,
		ChangeEvents: "!!!!(not a regex",
	})
	assert.NoError(t, err)

	// no-op
	wsc.dispatchChangeEvent(&fftypes.ChangeEvent{})

	mcb.AssertExpectations(t)
}

func TestHandleAckMultipleStartedMissingSub(t *testing.T) {
	eventUUID := fftypes.NewUUID()
	wsc := &websocketConnection{
		ctx: context.Background(),
		started: []*websocketStartedSub{
			{ephemeral: false, name: "name1", namespace: "ns1"},
			{ephemeral: false, name: "name2", namespace: "ns1"},
			{ephemeral: false, name: "name3", namespace: "ns1"},
		},
		sendMessages: make(chan interface{}, 1),
		inflight: []*fftypes.EventDeliveryResponse{
			{ID: eventUUID},
		},
	}
	err := wsc.handleAck(&fftypes.WSClientActionAckPayload{
		ID: eventUUID,
	})
	assert.Regexp(t, "FF10175", err)

}

func TestHandleAckMultipleStartedNoSubSingleMatch(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	cbs.On("DeliveryResponse", mock.Anything, mock.Anything).Return(nil)
	eventUUID := fftypes.NewUUID()
	wsc := &websocketConnection{
		ctx: context.Background(),
		ws: &WebSockets{
			ctx:       context.Background(),
			callbacks: cbs,
		},
		started:      []*websocketStartedSub{{ephemeral: false, name: "name1", namespace: "ns1"}},
		sendMessages: make(chan interface{}, 1),
		inflight: []*fftypes.EventDeliveryResponse{
			{ID: eventUUID},
		},
	}
	err := wsc.handleAck(&fftypes.WSClientActionAckPayload{
		ID: eventUUID,
	})
	assert.NoError(t, err)

}

func TestHandleAckNoneInflight(t *testing.T) {
	wsc := &websocketConnection{
		ctx:          context.Background(),
		sendMessages: make(chan interface{}, 1),
		inflight:     []*fftypes.EventDeliveryResponse{},
	}
	err := wsc.handleAck(&fftypes.WSClientActionAckPayload{})
	assert.Regexp(t, "FF10175", err)
}

func TestProtocolErrorSwallowsSendError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	wsc := &websocketConnection{
		ctx:          ctx,
		sendMessages: make(chan interface{}),
	}
	wsc.protocolError(fmt.Errorf("pop"))

}

func TestSendLoopBadData(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	ws, wsc, cancel := newTestWebsockets(t, cbs)
	defer cancel()

	subscribedConn := make(chan string, 1)
	cbs.On("EphemeralSubscription",
		mock.MatchedBy(func(s string) bool {
			subscribedConn <- s
			return true
		}),
		"ns1", mock.Anything, mock.Anything).Return(nil)

	err := wsc.Send(context.Background(), []byte(`{"type":"start","namespace":"ns1","ephemeral":true}`))
	assert.NoError(t, err)

	connID := <-subscribedConn
	connection := ws.connections[connID]
	connection.sendMessages <- map[bool]bool{false: true} // no JSON representation

	// Connection should close on its own with that bad data
	<-connection.senderDone

}

func TestUpgradeFail(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	_, wsc, cancel := newTestWebsockets(t, cbs)
	defer cancel()

	u, _ := url.Parse(wsc.URL())
	u.Scheme = "http"
	res, err := http.Get(u.String())
	assert.NoError(t, err)
	assert.Equal(t, 400, res.StatusCode)

}
func TestConnectionDispatchAfterClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	wsc := &websocketConnection{
		ctx: ctx,
	}
	err := wsc.dispatch(&fftypes.EventDelivery{})
	assert.Regexp(t, "FF10160", err)
}

func TestWebsocketDispatchAfterClose(t *testing.T) {
	ws := &WebSockets{
		ctx:         context.Background(),
		connections: make(map[string]*websocketConnection),
	}
	err := ws.DeliveryRequest("gone", nil, &fftypes.EventDelivery{}, nil)
	assert.Regexp(t, "FF10173", err)
}

func TestDispatchAutoAck(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	cbs.On("DeliveryResponse", mock.Anything, mock.Anything).Return(nil)
	wsc := &websocketConnection{
		ctx:    context.Background(),
		connID: fftypes.NewUUID().String(),
		ws: &WebSockets{
			ctx:         context.Background(),
			callbacks:   cbs,
			connections: make(map[string]*websocketConnection),
		},
		started:      []*websocketStartedSub{{ephemeral: false, name: "name1", namespace: "ns1"}},
		sendMessages: make(chan interface{}, 1),
		autoAck:      true,
	}
	wsc.ws.connections[wsc.connID] = wsc
	err := wsc.ws.DeliveryRequest(wsc.connID, nil, &fftypes.EventDelivery{
		Event:        fftypes.Event{ID: fftypes.NewUUID()},
		Subscription: fftypes.SubscriptionRef{ID: fftypes.NewUUID(), Namespace: "ns1", Name: "sub1"},
	}, nil)
	assert.NoError(t, err)
	cbs.AssertExpectations(t)
}
