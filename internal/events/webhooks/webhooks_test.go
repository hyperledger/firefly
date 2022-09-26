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

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/mocks/eventsmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestWebHooks(t *testing.T) (wh *WebHooks, cancel func()) {
	coreconfig.Reset()

	cbs := &eventsmocks.Callbacks{}
	rc := cbs.On("RegisterConnection", mock.Anything, mock.Anything).Return(nil)
	rc.RunFn = func(a mock.Arguments) {
		assert.Equal(t, true, a[1].(events.SubscriptionMatcher)(core.SubscriptionRef{}))
	}
	wh = &WebHooks{}
	ctx, cancelCtx := context.WithCancel(context.Background())
	svrConfig := config.RootSection("ut.webhooks")
	wh.InitConfig(svrConfig)
	wh.Init(ctx, svrConfig)
	wh.SetHandler("ns1", cbs)
	assert.Equal(t, "webhooks", wh.Name())
	assert.NotNil(t, wh.Capabilities())
	return wh, cancelCtx
}

func TestValidateOptionsWithDataFalse(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	no := false
	opts := &core.SubscriptionOptions{
		SubscriptionCoreOptions: core.SubscriptionCoreOptions{
			WithData: &no,
		},
	}
	opts.TransportOptions()["url"] = "/anything"
	err := wh.ValidateOptions(opts)
	assert.NoError(t, err)
	assert.False(t, *opts.WithData)
}

func TestValidateOptionsWithDataDefaulTrue(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	opts := &core.SubscriptionOptions{}
	opts.TransportOptions()["url"] = "/anything"
	err := wh.ValidateOptions(opts)
	assert.NoError(t, err)
	assert.True(t, *opts.WithData)
}

func TestValidateOptionsBadURL(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	opts := &core.SubscriptionOptions{}
	opts.TransportOptions()
	err := wh.ValidateOptions(opts)
	assert.Regexp(t, "FF10242", err)
}

func TestValidateOptionsBadHeaders(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	opts := &core.SubscriptionOptions{}
	opts.TransportOptions()
	opts.TransportOptions()["url"] = "/anything"
	opts.TransportOptions()["headers"] = fftypes.JSONObject{
		"bad": map[bool]bool{false: true},
	}
	err := wh.ValidateOptions(opts)
	assert.Regexp(t, "FF10243.*headers", err)
}

func TestValidateOptionsBadQuery(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	opts := &core.SubscriptionOptions{}
	opts.TransportOptions()
	opts.TransportOptions()["url"] = "/anything"
	opts.TransportOptions()["query"] = fftypes.JSONObject{
		"bad": map[bool]bool{false: true},
	}
	err := wh.ValidateOptions(opts)
	assert.Regexp(t, "FF10243.*query", err)
}

func TestRequestWithBodyReplyEndToEnd(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	r := mux.NewRouter()
	r.HandleFunc("/myapi/my/sub/path?escape_query", func(res http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "myheaderval", req.Header.Get("My-Header"))
		assert.Equal(t, "dynamicheaderval", req.Header.Get("Dynamic-Header"))
		assert.Equal(t, "myqueryval", req.URL.Query().Get("my-query"))
		assert.Equal(t, "dynamicqueryval", req.URL.Query().Get("dynamic-query"))
		var body fftypes.JSONObject
		err := json.NewDecoder(req.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "inputvalue", body.GetString("inputfield"))
		res.Header().Set("my-reply-header", "myheaderval2")
		res.WriteHeader(200)
		res.Write([]byte(`{
			"replyfield": "replyvalue"
		}`))
	}).Methods(http.MethodPut)
	server := httptest.NewServer(r)
	defer server.Close()

	yes := true
	dataID := fftypes.NewUUID()
	msgID := fftypes.NewUUID()
	groupHash := fftypes.NewRandB32()
	sub := &core.Subscription{
		SubscriptionRef: core.SubscriptionRef{
			Namespace: "ns1",
		},
		Options: core.SubscriptionOptions{
			SubscriptionCoreOptions: core.SubscriptionCoreOptions{
				WithData: &yes,
			},
		},
	}
	to := sub.Options.TransportOptions()
	to["reply"] = true
	to["json"] = true
	to["method"] = "PUT"
	to["url"] = fmt.Sprintf("http://%s/myapi/", server.Listener.Addr())
	to["headers"] = map[string]interface{}{
		"my-header": "myheaderval",
	}
	to["query"] = map[string]interface{}{
		"my-query": "myqueryval",
	}
	to["input"] = map[string]interface{}{
		"query":   "in_query",
		"headers": "in_headers",
		"body":    "in_body",
		"path":    "in_path",
		"replytx": "in_replytx",
	}
	event := &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{
				ID: fftypes.NewUUID(),
			},
			Message: &core.Message{
				Header: core.MessageHeader{
					ID:    msgID,
					Group: groupHash,
					Type:  core.MessageTypePrivate,
				},
				Data: core.DataRefs{
					{ID: dataID},
				},
			},
		},
		Subscription: core.SubscriptionRef{
			ID: sub.ID,
		},
	}
	data := &core.Data{
		ID: dataID,
		Value: fftypes.JSONAnyPtr(`{
			"in_body": {
				"inputfield": "inputvalue"
			},
			"in_query": {
				"dynamic-query": "dynamicqueryval"
			},
			"in_headers": {
				"dynamic-header": "dynamicheaderval"
			},
			"in_path": "/my/sub/path?escape_query",
			"in_replytx": true
		}`),
	}

	mcb := wh.callbacks["ns1"].(*eventsmocks.Callbacks)
	mcb.On("DeliveryResponse", mock.Anything, mock.MatchedBy(func(response *core.EventDeliveryResponse) bool {
		assert.Equal(t, *msgID, *response.Reply.Message.Header.CID)
		assert.Equal(t, *groupHash, *response.Reply.Message.Header.Group)
		assert.Equal(t, core.MessageTypePrivate, response.Reply.Message.Header.Type)
		assert.Equal(t, core.TransactionTypeBatchPin, response.Reply.Message.Header.TxType)
		assert.Equal(t, "myheaderval2", response.Reply.InlineData[0].Value.JSONObject().GetObject("headers").GetString("My-Reply-Header"))
		assert.Equal(t, "replyvalue", response.Reply.InlineData[0].Value.JSONObject().GetObject("body").GetString("replyfield"))
		assert.Equal(t, float64(200), response.Reply.InlineData[0].Value.JSONObject()["status"])
		return true
	})).Return(nil)

	err := wh.DeliveryRequest(mock.Anything, sub, event, core.DataArray{data})
	assert.NoError(t, err)

	mcb.AssertExpectations(t)
}

func TestRequestWithEmptyStringBodyReplyEndToEnd(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	r := mux.NewRouter()
	r.HandleFunc("/myapi/my/sub/path?escape_query", func(res http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "myheaderval", req.Header.Get("My-Header"))
		assert.Equal(t, "dynamicheaderval", req.Header.Get("Dynamic-Header"))
		assert.Equal(t, "myqueryval", req.URL.Query().Get("my-query"))
		assert.Equal(t, "dynamicqueryval", req.URL.Query().Get("dynamic-query"))
		var body fftypes.JSONObject
		err := json.NewDecoder(req.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "", body.GetString("inputfield"))
		res.Header().Set("my-reply-header", "myheaderval2")
		res.WriteHeader(200)
		res.Write([]byte(`{
			"replyfield": ""
		}`))
	}).Methods(http.MethodPut)
	server := httptest.NewServer(r)
	defer server.Close()

	yes := true
	dataID := fftypes.NewUUID()
	msgID := fftypes.NewUUID()
	groupHash := fftypes.NewRandB32()
	sub := &core.Subscription{
		SubscriptionRef: core.SubscriptionRef{
			Namespace: "ns1",
		},
		Options: core.SubscriptionOptions{
			SubscriptionCoreOptions: core.SubscriptionCoreOptions{
				WithData: &yes,
			},
		},
	}
	to := sub.Options.TransportOptions()
	to["reply"] = true
	to["json"] = true
	to["method"] = "PUT"
	to["url"] = fmt.Sprintf("http://%s/myapi/", server.Listener.Addr())
	to["headers"] = map[string]interface{}{
		"my-header": "myheaderval",
	}
	to["query"] = map[string]interface{}{
		"my-query": "myqueryval",
	}
	to["input"] = map[string]interface{}{
		"query":   "in_query",
		"headers": "in_headers",
		"body":    "in_body",
		"path":    "in_path",
		"replytx": "in_replytx",
	}
	event := &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{
				ID: fftypes.NewUUID(),
			},
			Message: &core.Message{
				Header: core.MessageHeader{
					ID:    msgID,
					Group: groupHash,
					Type:  core.MessageTypePrivate,
				},
				Data: core.DataRefs{
					{ID: dataID},
				},
			},
		},
		Subscription: core.SubscriptionRef{
			ID: sub.ID,
		},
	}
	data := &core.Data{
		ID: dataID,
		Value: fftypes.JSONAnyPtr(`{
			"in_body": {
				"inputfield": ""
			},
			"in_query": {
				"dynamic-query": "dynamicqueryval"
			},
			"in_headers": {
				"dynamic-header": "dynamicheaderval"
			},
			"in_path": "/my/sub/path?escape_query",
			"in_replytx": true
		}`),
	}

	mcb := wh.callbacks["ns1"].(*eventsmocks.Callbacks)
	mcb.On("DeliveryResponse", mock.Anything, mock.MatchedBy(func(response *core.EventDeliveryResponse) bool {
		assert.Equal(t, *msgID, *response.Reply.Message.Header.CID)
		assert.Equal(t, *groupHash, *response.Reply.Message.Header.Group)
		assert.Equal(t, core.MessageTypePrivate, response.Reply.Message.Header.Type)
		assert.Equal(t, core.TransactionTypeBatchPin, response.Reply.Message.Header.TxType)
		assert.Equal(t, "myheaderval2", response.Reply.InlineData[0].Value.JSONObject().GetObject("headers").GetString("My-Reply-Header"))
		assert.Equal(t, "", response.Reply.InlineData[0].Value.JSONObject().GetObject("body").GetString("replyfield"))
		assert.Equal(t, float64(200), response.Reply.InlineData[0].Value.JSONObject()["status"])
		return true
	})).Return(nil)

	err := wh.DeliveryRequest(mock.Anything, sub, event, core.DataArray{data})
	assert.NoError(t, err)

	mcb.AssertExpectations(t)
}

func TestRequestNoBodyNoReply(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	msgID := fftypes.NewUUID()

	called := false
	r := mux.NewRouter()
	r.HandleFunc("/myapi", func(res http.ResponseWriter, req *http.Request) {
		var body fftypes.JSONObject
		err := json.NewDecoder(req.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, msgID.String(), body.GetObject("message").GetObject("header").GetString("id"))
		res.WriteHeader(200)
		called = true
	}).Methods(http.MethodPost)
	server := httptest.NewServer(r)
	defer server.Close()

	dataID := fftypes.NewUUID()
	groupHash := fftypes.NewRandB32()
	sub := &core.Subscription{
		SubscriptionRef: core.SubscriptionRef{
			Namespace: "ns1",
		},
	}
	to := sub.Options.TransportOptions()
	to["url"] = fmt.Sprintf("http://%s/myapi", server.Listener.Addr())
	event := &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{
				ID: fftypes.NewUUID(),
			},
			Message: &core.Message{
				Header: core.MessageHeader{
					ID:    msgID,
					Group: groupHash,
					Type:  core.MessageTypePrivate,
				},
				Data: core.DataRefs{
					{ID: dataID},
				},
			},
		},
		Subscription: core.SubscriptionRef{
			ID:        sub.ID,
			Namespace: "ns1",
		},
	}
	data := &core.Data{
		ID: dataID,
		Value: fftypes.JSONAnyPtr(`{
			"inputfield": "inputvalue"
		}`),
	}

	mcb := wh.callbacks["ns1"].(*eventsmocks.Callbacks)
	mcb.On("DeliveryResponse", mock.Anything, mock.MatchedBy(func(response *core.EventDeliveryResponse) bool {
		return !response.Rejected
	})).Return(nil)

	err := wh.DeliveryRequest(mock.Anything, sub, event, core.DataArray{data})
	assert.NoError(t, err)
	assert.True(t, called)

	mcb.AssertExpectations(t)
}

func TestRequestReplyEmptyData(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	msgID := fftypes.NewUUID()

	called := false
	r := mux.NewRouter()
	r.HandleFunc("/myapi", func(res http.ResponseWriter, req *http.Request) {
		var body fftypes.JSONObject
		err := json.NewDecoder(req.Body).Decode(&body)
		assert.NoError(t, err)
		res.WriteHeader(200)
		called = true
	}).Methods(http.MethodPost)
	server := httptest.NewServer(r)
	defer server.Close()

	yes := true
	sub := &core.Subscription{
		SubscriptionRef: core.SubscriptionRef{
			Namespace: "ns1",
		},
		Options: core.SubscriptionOptions{
			SubscriptionCoreOptions: core.SubscriptionCoreOptions{
				WithData: &yes,
			},
		},
	}
	to := sub.Options.TransportOptions()
	to["url"] = fmt.Sprintf("http://%s/myapi", server.Listener.Addr())
	to["reply"] = true
	event := &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{
				ID: fftypes.NewUUID(),
			},
			Message: &core.Message{
				Header: core.MessageHeader{
					ID:   msgID,
					Type: core.MessageTypeBroadcast,
				},
			},
		},
		Subscription: core.SubscriptionRef{
			ID: sub.ID,
		},
	}

	mcb := wh.callbacks["ns1"].(*eventsmocks.Callbacks)
	mcb.On("DeliveryResponse", mock.Anything, mock.MatchedBy(func(response *core.EventDeliveryResponse) bool {
		assert.Equal(t, *msgID, *response.Reply.Message.Header.CID)
		assert.Nil(t, response.Reply.Message.Header.Group)
		assert.Equal(t, core.MessageTypeBroadcast, response.Reply.Message.Header.Type)
		return true
	})).Return(nil)

	err := wh.DeliveryRequest(mock.Anything, sub, event, core.DataArray{})
	assert.NoError(t, err)
	assert.True(t, called)

	mcb.AssertExpectations(t)
}

func TestRequestReplyBadJSON(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	msgID := fftypes.NewUUID()

	r := mux.NewRouter()
	r.HandleFunc("/myapi", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(200)
		res.Write([]byte(`!badjson`))
	}).Methods(http.MethodPost)
	server := httptest.NewServer(r)
	defer server.Close()

	sub := &core.Subscription{
		SubscriptionRef: core.SubscriptionRef{
			Namespace: "ns1",
		},
	}
	to := sub.Options.TransportOptions()
	to["url"] = fmt.Sprintf("http://%s/myapi", server.Listener.Addr())
	to["reply"] = true
	to["json"] = true
	event := &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{
				ID: fftypes.NewUUID(),
			},
			Message: &core.Message{
				Header: core.MessageHeader{
					ID:   msgID,
					Type: core.MessageTypeBroadcast,
				},
			},
		},
		Subscription: core.SubscriptionRef{
			ID: sub.ID,
		},
	}

	mcb := wh.callbacks["ns1"].(*eventsmocks.Callbacks)
	mcb.On("DeliveryResponse", mock.Anything, mock.MatchedBy(func(response *core.EventDeliveryResponse) bool {
		assert.Equal(t, float64(502), response.Reply.InlineData[0].Value.JSONObject()["status"])
		assert.Regexp(t, "FF10257", response.Reply.InlineData[0].Value.JSONObject().GetObject("body")["error"])
		return true
	})).Return(nil)

	err := wh.DeliveryRequest(mock.Anything, sub, event, core.DataArray{})
	assert.NoError(t, err)

	mcb.AssertExpectations(t)
}

func TestRequestReplyDataArrayBadStatusB64(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	msgID := fftypes.NewUUID()

	called := false
	r := mux.NewRouter()
	r.HandleFunc("/myapi", func(res http.ResponseWriter, req *http.Request) {
		var body []string
		err := json.NewDecoder(req.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Len(t, body, 2)
		assert.Equal(t, "value1", body[0])
		assert.Equal(t, "value2", body[1])
		res.WriteHeader(500)
		res.Write([]byte(`some bytes`))
		called = true
	}).Methods(http.MethodPost)
	server := httptest.NewServer(r)
	defer server.Close()

	yes := true
	sub := &core.Subscription{
		SubscriptionRef: core.SubscriptionRef{
			Namespace: "ns1",
		},
		Options: core.SubscriptionOptions{
			SubscriptionCoreOptions: core.SubscriptionCoreOptions{
				WithData: &yes,
			},
		},
	}
	to := sub.Options.TransportOptions()
	to["url"] = fmt.Sprintf("http://%s/myapi", server.Listener.Addr())
	to["reply"] = true
	event := &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{
				ID: fftypes.NewUUID(),
			},
			Message: &core.Message{
				Header: core.MessageHeader{
					ID:   msgID,
					Type: core.MessageTypeBroadcast,
				},
			},
		},
		Subscription: core.SubscriptionRef{
			ID: sub.ID,
		},
	}

	mcb := wh.callbacks["ns1"].(*eventsmocks.Callbacks)
	mcb.On("DeliveryResponse", mock.Anything, mock.MatchedBy(func(response *core.EventDeliveryResponse) bool {
		assert.Equal(t, *msgID, *response.Reply.Message.Header.CID)
		assert.Nil(t, response.Reply.Message.Header.Group)
		assert.Equal(t, core.MessageTypeBroadcast, response.Reply.Message.Header.Type)
		assert.Equal(t, float64(500), response.Reply.InlineData[0].Value.JSONObject()["status"])
		assert.Equal(t, `c29tZSBieXRlcw==`, response.Reply.InlineData[0].Value.JSONObject()["body"]) // base64 val
		return true
	})).Return(nil)

	err := wh.DeliveryRequest(mock.Anything, sub, event, core.DataArray{
		{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"value1"`)},
		{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"value2"`)},
	})
	assert.NoError(t, err)
	assert.True(t, called)

	mcb.AssertExpectations(t)
}

func TestRequestReplyDataArrayError(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	msgID := fftypes.NewUUID()
	r := mux.NewRouter()
	server := httptest.NewServer(r)
	server.Close()

	sub := &core.Subscription{
		SubscriptionRef: core.SubscriptionRef{
			Namespace: "ns1",
		},
	}
	to := sub.Options.TransportOptions()
	to["url"] = fmt.Sprintf("http://%s/myapi", server.Listener.Addr())
	to["reply"] = true
	event := &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{
				ID: fftypes.NewUUID(),
			},
			Message: &core.Message{
				Header: core.MessageHeader{
					ID:   msgID,
					Type: core.MessageTypeBroadcast,
				},
			},
		},
		Subscription: core.SubscriptionRef{
			ID: sub.ID,
		},
	}

	mcb := wh.callbacks["ns1"].(*eventsmocks.Callbacks)
	mcb.On("DeliveryResponse", mock.Anything, mock.MatchedBy(func(response *core.EventDeliveryResponse) bool {
		assert.Equal(t, *msgID, *response.Reply.Message.Header.CID)
		assert.Nil(t, response.Reply.Message.Header.Group)
		assert.Equal(t, core.MessageTypeBroadcast, response.Reply.Message.Header.Type)
		assert.Equal(t, float64(502), response.Reply.InlineData[0].Value.JSONObject()["status"])
		assert.NotEmpty(t, response.Reply.InlineData[0].Value.JSONObject().GetObject("body")["error"])
		return true
	})).Return(nil)

	err := wh.DeliveryRequest(mock.Anything, sub, event, core.DataArray{
		{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"value1"`)},
		{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"value2"`)},
	})
	assert.NoError(t, err)

	mcb.AssertExpectations(t)
}

func TestWebhookFailFastAsk(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	msgID := fftypes.NewUUID()
	r := mux.NewRouter()
	server := httptest.NewServer(r)
	server.Close()

	sub := &core.Subscription{
		SubscriptionRef: core.SubscriptionRef{
			Namespace: "ns1",
		},
	}
	sub.Options.TransportOptions()["fastack"] = true
	event := &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{
				ID: fftypes.NewUUID(),
			},
			Message: &core.Message{
				Header: core.MessageHeader{
					ID:   msgID,
					Type: core.MessageTypeBroadcast,
				},
			},
		},
		Subscription: core.SubscriptionRef{
			ID: sub.ID,
		},
	}

	waiter := make(chan struct{})
	mcb := wh.callbacks["ns1"].(*eventsmocks.Callbacks)
	mcb.On("DeliveryResponse", mock.Anything, mock.Anything).
		Return(nil).
		Run(func(a mock.Arguments) {
			close(waiter)
		})

	err := wh.DeliveryRequest(mock.Anything, sub, event, core.DataArray{
		{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"value1"`)},
		{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"value2"`)},
	})
	assert.NoError(t, err)
	<-waiter

	mcb.AssertExpectations(t)
}

func TestDeliveryRequestNilMessage(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	yes := true
	sub := &core.Subscription{
		SubscriptionRef: core.SubscriptionRef{
			Namespace: "ns1",
		},
		Options: core.SubscriptionOptions{
			SubscriptionCoreOptions: core.SubscriptionCoreOptions{
				WithData: &yes,
			},
		},
	}
	sub.Options.TransportOptions()["reply"] = true
	event := &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{
				ID: fftypes.NewUUID(),
			},
		},
		Subscription: core.SubscriptionRef{
			ID: sub.ID,
		},
	}

	err := wh.DeliveryRequest(mock.Anything, sub, event, nil)
	assert.NoError(t, err)
}

func TestDeliveryRequestReplyToReply(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	yes := true
	sub := &core.Subscription{
		SubscriptionRef: core.SubscriptionRef{
			Namespace: "ns1",
		},
		Options: core.SubscriptionOptions{
			SubscriptionCoreOptions: core.SubscriptionCoreOptions{
				WithData: &yes,
			},
		},
	}
	sub.Options.TransportOptions()["reply"] = true
	event := &core.EventDelivery{
		EnrichedEvent: core.EnrichedEvent{
			Event: core.Event{
				ID: fftypes.NewUUID(),
			},
			Message: &core.Message{
				Header: core.MessageHeader{
					ID:   fftypes.NewUUID(),
					Type: core.MessageTypeBroadcast,
					CID:  fftypes.NewUUID(),
				},
			},
		},
		Subscription: core.SubscriptionRef{
			ID: sub.ID,
		},
	}

	mcb := wh.callbacks["ns1"].(*eventsmocks.Callbacks)
	mcb.On("DeliveryResponse", mock.Anything, mock.MatchedBy(func(response *core.EventDeliveryResponse) bool {
		return !response.Rejected // should be accepted as a no-op so we can move on to other events
	}))

	err := wh.DeliveryRequest(mock.Anything, sub, event, nil)
	assert.NoError(t, err)

	mcb.AssertExpectations(t)
}
