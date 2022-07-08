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

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/wsclient"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/mocks/eventsmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testAuthorizer struct{}

func (t *testAuthorizer) Authorize(ctx context.Context, authReq *fftypes.AuthReq) error {
	if authReq.Namespace == "ns1" {
		return nil
	}
	return i18n.NewError(ctx, i18n.MsgUnauthorized)
}

func newTestWebsockets(t *testing.T, cbs *eventsmocks.Callbacks, authorizer core.Authorizer, queryParams ...string) (ws *WebSockets, wsc wsclient.WSClient, cancel func()) {
	coreconfig.Reset()

	ws = &WebSockets{}
	ctx, cancelCtx := context.WithCancel(context.Background())
	svrConfig := config.RootSection("ut.websockets")
	ws.InitConfig(svrConfig)
	ws.Init(ctx, svrConfig)
	ws.SetHandler("ns1", cbs)
	ws.SetAuthorizer(authorizer)
	assert.Equal(t, "websockets", ws.Name())
	assert.NotNil(t, ws.Capabilities())
	cbs.On("ConnectionClosed", mock.Anything).Return(nil).Maybe()

	svr := httptest.NewServer(ws)

	clientConfig := config.RootSection("ut.wsclient")
	wsclient.InitConfig(clientConfig)
	qs := ""
	if len(queryParams) > 0 {
		qs = fmt.Sprintf("?%s", strings.Join(queryParams, "&"))
	}
	clientConfig.Set(ffresty.HTTPConfigURL, fmt.Sprintf("http://%s%s", svr.Listener.Addr(), qs))
	wsConfig := wsclient.GenerateConfig(clientConfig)

	wsc, err := wsclient.New(ctx, wsConfig, nil, nil)
	assert.NoError(t, err)
	err = wsc.Connect()
	assert.NoError(t, err)

	return ws, wsc, func() {
		cancelCtx()
		wsc.Close()
		ws.WaitClosed()
		svr.Close()
	}
}
func TestValidateOptionsFail(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	ws, _, cancel := newTestWebsockets(t, cbs, nil)
	defer cancel()

	yes := true
	err := ws.ValidateOptions(&core.SubscriptionOptions{
		SubscriptionCoreOptions: core.SubscriptionCoreOptions{
			WithData: &yes,
		},
	})
	assert.Regexp(t, "FF10244", err)
}

func TestValidateOptionsOk(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	ws, _, cancel := newTestWebsockets(t, cbs, nil)
	defer cancel()

	opts := &core.SubscriptionOptions{}
	err := ws.ValidateOptions(opts)
	assert.NoError(t, err)
	assert.False(t, *opts.WithData)
}

func TestSendBadData(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	_, wsc, cancel := newTestWebsockets(t, cbs, nil)
	defer cancel()

	cbs.On("ConnectionClosed", mock.Anything).Return(nil)

	err := wsc.Send(context.Background(), []byte(`!json`))
	assert.NoError(t, err)
	b := <-wsc.Receive()
	var res core.WSError
	err = json.Unmarshal(b, &res)
	assert.NoError(t, err)
	assert.Equal(t, core.WSProtocolErrorEventType, res.Type)
	assert.Regexp(t, "FF10176", res.Error)
}

func TestSendBadAction(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	_, wsc, cancel := newTestWebsockets(t, cbs, nil)
	defer cancel()
	cbs.On("ConnectionClosed", mock.Anything).Return(nil)

	err := wsc.Send(context.Background(), []byte(`{"type":"lobster"}`))
	assert.NoError(t, err)
	b := <-wsc.Receive()
	var res core.WSError
	err = json.Unmarshal(b, &res)
	assert.NoError(t, err)
	assert.Equal(t, core.WSProtocolErrorEventType, res.Type)
	assert.Regexp(t, "FF10176", res.Error)
}

func TestSendEmptyStartAction(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	_, wsc, cancel := newTestWebsockets(t, cbs, nil)
	defer cancel()
	cbs.On("ConnectionClosed", mock.Anything).Return(nil)

	err := wsc.Send(context.Background(), []byte(`{"type":"start"}`))
	assert.NoError(t, err)
	b := <-wsc.Receive()
	var res core.WSError
	err = json.Unmarshal(b, &res)
	assert.NoError(t, err)
	assert.Equal(t, core.WSProtocolErrorEventType, res.Type)
	assert.Regexp(t, "FF10176", res.Error)
}

func TestStartReceiveAckEphemeral(t *testing.T) {
	log.SetLevel("trace")

	cbs := &eventsmocks.Callbacks{}
	ws, wsc, cancel := newTestWebsockets(t, cbs, nil)
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
	ws.DeliveryRequest(connID, nil, &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{ID: fftypes.NewUUID()},
		},
		Subscription: core.SubscriptionRef{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
		},
	}, nil)

	b := <-wsc.Receive()
	var res core.EventDelivery
	err = json.Unmarshal(b, &res)
	assert.NoError(t, err)

	err = wsc.Send(context.Background(), []byte(`{"type":"ack"}`))
	assert.NoError(t, err)

	<-waitAcked
	cbs.AssertExpectations(t)
}

func TestStartReceiveDurable(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	ws, wsc, cancel := newTestWebsockets(t, cbs, nil)
	defer cancel()
	var connID string
	sub := cbs.On("RegisterConnection",
		mock.MatchedBy(func(s string) bool { connID = s; return true }),
		mock.MatchedBy(func(subMatch events.SubscriptionMatcher) bool {
			return subMatch(core.SubscriptionRef{Namespace: "ns1", Name: "sub1"}) &&
				!subMatch(core.SubscriptionRef{Namespace: "ns2", Name: "sub1"}) &&
				!subMatch(core.SubscriptionRef{Namespace: "ns1", Name: "sub2"})
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
	ws.DeliveryRequest(connID, nil, &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{ID: fftypes.NewUUID()},
		},
		Subscription: core.SubscriptionRef{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "sub1",
		},
	}, nil)
	// Put a second in flight
	ws.DeliveryRequest(connID, nil, &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{ID: fftypes.NewUUID()},
		},
		Subscription: core.SubscriptionRef{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "sub2",
		},
	}, nil)

	b := <-wsc.Receive()
	var res core.EventDelivery
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

func TestStartReceiveDurableWithAuth(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	ws, wsc, cancel := newTestWebsockets(t, cbs, &testAuthorizer{})
	defer cancel()
	var connID string
	sub := cbs.On("RegisterConnection",
		mock.MatchedBy(func(s string) bool { connID = s; return true }),
		mock.MatchedBy(func(subMatch events.SubscriptionMatcher) bool {
			return subMatch(core.SubscriptionRef{Namespace: "ns1", Name: "sub1"}) &&
				!subMatch(core.SubscriptionRef{Namespace: "ns2", Name: "sub1"}) &&
				!subMatch(core.SubscriptionRef{Namespace: "ns1", Name: "sub2"})
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
	ws.DeliveryRequest(connID, nil, &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{ID: fftypes.NewUUID()},
		},
		Subscription: core.SubscriptionRef{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "sub1",
		},
	}, nil)
	// Put a second in flight
	ws.DeliveryRequest(connID, nil, &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{ID: fftypes.NewUUID()},
		},
		Subscription: core.SubscriptionRef{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "sub2",
		},
	}, nil)

	b := <-wsc.Receive()
	var res core.EventDelivery
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

func TestStartReceiveDurableUnauthorized(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	_, wsc, cancel := newTestWebsockets(t, cbs, &testAuthorizer{})
	defer cancel()
	var connID string
	sub := cbs.On("RegisterConnection",
		mock.MatchedBy(func(s string) bool { connID = s; return true }),
		mock.MatchedBy(func(subMatch events.SubscriptionMatcher) bool {
			return subMatch(core.SubscriptionRef{Namespace: "ns2", Name: "sub1"}) &&
				!subMatch(core.SubscriptionRef{Namespace: "ns1", Name: "sub2"})
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

	err := wsc.Send(context.Background(), []byte(`{"type":"start","namespace":"ns2","name":"sub1"}`))
	assert.NoError(t, err)

	b := <-wsc.Receive()
	var res fftypes.JSONObject
	err = json.Unmarshal(b, &res)
	assert.NoError(t, err)
	assert.Regexp(t, "FF00169", res.GetString("error"))
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

	ws, wsc, cancel := newTestWebsockets(t, cbs, nil, "ephemeral", "namespace=ns1")
	defer cancel()

	<-waitSubscribed
	ws.DeliveryRequest(connID, nil, &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{ID: fftypes.NewUUID()},
		},
		Subscription: core.SubscriptionRef{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
		},
	}, nil)

	b := <-wsc.Receive()
	var res core.EventDelivery
	err := json.Unmarshal(b, &res)
	assert.NoError(t, err)

	err = wsc.Send(context.Background(), []byte(`{"type":"ack"}`))
	assert.NoError(t, err)

	<-waitAcked
	cbs.AssertExpectations(t)
}

func TestAutoStartBadOptions(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	_, wsc, cancel := newTestWebsockets(t, cbs, nil, "name=missingnamespace")
	defer cancel()

	b := <-wsc.Receive()
	var res core.WSError
	err := json.Unmarshal(b, &res)
	assert.NoError(t, err)
	assert.Regexp(t, "FF10178", res.Error)
	cbs.AssertExpectations(t)
}

func TestAutoStartBadNamespace(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	_, wsc, cancel := newTestWebsockets(t, cbs, nil, "ephemeral", "namespace=ns2")
	defer cancel()

	b := <-wsc.Receive()
	var res core.WSError
	err := json.Unmarshal(b, &res)
	assert.NoError(t, err)
	assert.Regexp(t, "FF10187", res.Error)
	cbs.AssertExpectations(t)
}

func TestHandleAckWithAutoAck(t *testing.T) {
	eventUUID := fftypes.NewUUID()
	wsc := &websocketConnection{
		ctx:          context.Background(),
		started:      []*websocketStartedSub{{ephemeral: false, name: "name1", namespace: "ns1"}},
		sendMessages: make(chan interface{}, 1),
		inflight: []*core.EventDeliveryResponse{
			{ID: eventUUID},
		},
		autoAck: true,
	}
	err := wsc.handleAck(&core.WSAck{
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
		inflight: []*core.EventDeliveryResponse{
			{ID: eventUUID},
		},
		autoAck: true,
	}
	no := false
	err := wsc.handleStart(&core.WSStart{
		AutoAck: &no,
	})
	assert.Regexp(t, "FF10179", err)
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
		inflight: []*core.EventDeliveryResponse{
			{ID: eventUUID},
		},
	}
	err := wsc.handleAck(&core.WSAck{
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
			callbacks: map[string]events.Callbacks{"ns1": cbs},
		},
		started:      []*websocketStartedSub{{ephemeral: false, name: "name1", namespace: "ns1"}},
		sendMessages: make(chan interface{}, 1),
		inflight: []*core.EventDeliveryResponse{
			{ID: eventUUID},
		},
	}
	err := wsc.handleAck(&core.WSAck{
		ID: eventUUID,
	})
	assert.NoError(t, err)

}

func TestHandleAckNoneInflight(t *testing.T) {
	wsc := &websocketConnection{
		ctx:          context.Background(),
		sendMessages: make(chan interface{}, 1),
		inflight:     []*core.EventDeliveryResponse{},
	}
	err := wsc.handleAck(&core.WSAck{})
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
	ws, wsc, cancel := newTestWebsockets(t, cbs, nil)
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
	_, wsc, cancel := newTestWebsockets(t, cbs, nil)
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
	err := wsc.dispatch(&core.EventDelivery{})
	assert.Regexp(t, "FF00147", err)
}

func TestWebsocketDispatchAfterClose(t *testing.T) {
	ws := &WebSockets{
		ctx:         context.Background(),
		connections: make(map[string]*websocketConnection),
	}
	err := ws.DeliveryRequest("gone", nil, &core.EventDelivery{}, nil)
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
			callbacks:   map[string]events.Callbacks{"ns1": cbs},
			connections: make(map[string]*websocketConnection),
		},
		started:      []*websocketStartedSub{{ephemeral: false, name: "name1", namespace: "ns1"}},
		sendMessages: make(chan interface{}, 1),
		autoAck:      true,
	}
	wsc.ws.connections[wsc.connID] = wsc
	err := wsc.ws.DeliveryRequest(wsc.connID, nil, &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{ID: fftypes.NewUUID()},
		},
		Subscription: core.SubscriptionRef{ID: fftypes.NewUUID(), Namespace: "ns1", Name: "sub1"},
	}, nil)
	assert.NoError(t, err)
	cbs.AssertExpectations(t)
}

func TestWebsocketSendAfterClose(t *testing.T) {
	cbs := &eventsmocks.Callbacks{}
	ws, wsc, cancel := newTestWebsockets(t, cbs, nil)
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
	connection.wsConn.Close()
	<-connection.senderDone
	err = connection.send(map[string]string{"foo": "bar"})
	assert.Regexp(t, "FF10290", err)
}

func TestWebsocketStatus(t *testing.T) {
	ws := &WebSockets{
		ctx:         context.Background(),
		connections: make(map[string]*websocketConnection),
	}
	ws.connections["id1"] = &websocketConnection{
		connID: "id1",
		started: []*websocketStartedSub{
			{ephemeral: false, name: "name1", namespace: "ns1"},
		},
		remoteAddr: "otherhost",
		userAgent:  "user",
	}

	status := ws.GetStatus()

	assert.Equal(t, true, status.Enabled)
	assert.Len(t, status.Connections, 1)
	assert.Equal(t, "id1", status.Connections[0].ID)
	assert.Equal(t, "otherhost", status.Connections[0].RemoteAddress)
	assert.Equal(t, "user", status.Connections[0].UserAgent)
	assert.Len(t, status.Connections[0].Subscriptions, 1)
	assert.Equal(t, false, status.Connections[0].Subscriptions[0].Ephemeral)
	assert.Equal(t, "ns1", status.Connections[0].Subscriptions[0].Namespace)
	assert.Equal(t, "name1", status.Connections[0].Subscriptions[0].Name)
}
