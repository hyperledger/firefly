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
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/mocks/eventsmocks"
	"github.com/hyperledger-labs/firefly/pkg/events"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestWebHooks(t *testing.T) (wh *WebHooks, cancel func()) {
	config.Reset()

	cbs := &eventsmocks.Callbacks{}
	rc := cbs.On("RegisterConnection", mock.Anything, mock.Anything).Return(nil)
	rc.RunFn = func(a mock.Arguments) {
		assert.Equal(t, true, a[1].(events.SubscriptionMatcher)(fftypes.SubscriptionRef{}))
	}
	wh = &WebHooks{}
	ctx, cancelCtx := context.WithCancel(context.Background())
	svrPrefix := config.NewPluginConfig("ut.webhooks")
	wh.InitPrefix(svrPrefix)
	wh.Init(ctx, svrPrefix, cbs)
	assert.Equal(t, "webhooks", wh.Name())
	assert.NotNil(t, wh.Capabilities())
	assert.NotNil(t, wh.GetOptionsSchema(wh.ctx))
	return wh, cancelCtx
}

func TestValidateOptionsWithDataFalse(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	no := false
	opts := &fftypes.SubscriptionOptions{
		SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
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

	opts := &fftypes.SubscriptionOptions{}
	opts.TransportOptions()["url"] = "/anything"
	err := wh.ValidateOptions(opts)
	assert.NoError(t, err)
	assert.True(t, *opts.WithData)
}

func TestValidateOptionsBadURL(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	opts := &fftypes.SubscriptionOptions{}
	opts.TransportOptions()
	err := wh.ValidateOptions(opts)
	assert.Regexp(t, "FF10242", err)
}

func TestValidateOptionsBadHeaders(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	opts := &fftypes.SubscriptionOptions{}
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

	opts := &fftypes.SubscriptionOptions{}
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
	sub := &fftypes.Subscription{
		Options: fftypes.SubscriptionOptions{
			SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
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
	event := &fftypes.EventDelivery{
		Event: fftypes.Event{
			ID: fftypes.NewUUID(),
		},
		Subscription: fftypes.SubscriptionRef{
			ID: sub.ID,
		},
		Message: &fftypes.Message{
			Header: fftypes.MessageHeader{
				ID:    msgID,
				Group: groupHash,
				Type:  fftypes.MessageTypePrivate,
			},
			Data: fftypes.DataRefs{
				{ID: dataID},
			},
		},
	}
	data := &fftypes.Data{
		ID: dataID,
		Value: fftypes.Byteable(`{
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

	mcb := wh.callbacks.(*eventsmocks.Callbacks)
	mcb.On("DeliveryResponse", mock.Anything, mock.MatchedBy(func(response *fftypes.EventDeliveryResponse) bool {
		assert.Equal(t, *msgID, *response.Reply.Message.Header.CID)
		assert.Equal(t, *groupHash, *response.Reply.Message.Header.Group)
		assert.Equal(t, fftypes.MessageTypePrivate, response.Reply.Message.Header.Type)
		assert.Equal(t, fftypes.TransactionTypeBatchPin, response.Reply.Message.Header.TxType)
		assert.Equal(t, "myheaderval2", response.Reply.InlineData[0].Value.JSONObject().GetObject("headers").GetString("My-Reply-Header"))
		assert.Equal(t, "replyvalue", response.Reply.InlineData[0].Value.JSONObject().GetObject("body").GetString("replyfield"))
		assert.Equal(t, float64(200), response.Reply.InlineData[0].Value.JSONObject()["status"])
		return true
	})).Return(nil)

	err := wh.DeliveryRequest(mock.Anything, sub, event, []*fftypes.Data{data})
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
	sub := &fftypes.Subscription{}
	to := sub.Options.TransportOptions()
	to["url"] = fmt.Sprintf("http://%s/myapi", server.Listener.Addr())
	event := &fftypes.EventDelivery{
		Event: fftypes.Event{
			ID: fftypes.NewUUID(),
		},
		Subscription: fftypes.SubscriptionRef{
			ID: sub.ID,
		},
		Message: &fftypes.Message{
			Header: fftypes.MessageHeader{
				ID:    msgID,
				Group: groupHash,
				Type:  fftypes.MessageTypePrivate,
			},
			Data: fftypes.DataRefs{
				{ID: dataID},
			},
		},
	}
	data := &fftypes.Data{
		ID: dataID,
		Value: fftypes.Byteable(`{
			"inputfield": "inputvalue"
		}`),
	}

	err := wh.DeliveryRequest(mock.Anything, sub, event, []*fftypes.Data{data})
	assert.NoError(t, err)
	assert.True(t, called)
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
	sub := &fftypes.Subscription{
		Options: fftypes.SubscriptionOptions{
			SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
				WithData: &yes,
			},
		},
	}
	to := sub.Options.TransportOptions()
	to["url"] = fmt.Sprintf("http://%s/myapi", server.Listener.Addr())
	to["reply"] = true
	event := &fftypes.EventDelivery{
		Event: fftypes.Event{
			ID: fftypes.NewUUID(),
		},
		Subscription: fftypes.SubscriptionRef{
			ID: sub.ID,
		},
		Message: &fftypes.Message{
			Header: fftypes.MessageHeader{
				ID:   msgID,
				Type: fftypes.MessageTypeBroadcast,
			},
		},
	}

	mcb := wh.callbacks.(*eventsmocks.Callbacks)
	mcb.On("DeliveryResponse", mock.Anything, mock.MatchedBy(func(response *fftypes.EventDeliveryResponse) bool {
		assert.Equal(t, *msgID, *response.Reply.Message.Header.CID)
		assert.Nil(t, response.Reply.Message.Header.Group)
		assert.Equal(t, fftypes.MessageTypeBroadcast, response.Reply.Message.Header.Type)
		return true
	})).Return(nil)

	err := wh.DeliveryRequest(mock.Anything, sub, event, []*fftypes.Data{})
	assert.NoError(t, err)
	assert.True(t, called)
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

	sub := &fftypes.Subscription{}
	to := sub.Options.TransportOptions()
	to["url"] = fmt.Sprintf("http://%s/myapi", server.Listener.Addr())
	to["reply"] = true
	to["json"] = true
	event := &fftypes.EventDelivery{
		Event: fftypes.Event{
			ID: fftypes.NewUUID(),
		},
		Subscription: fftypes.SubscriptionRef{
			ID: sub.ID,
		},
		Message: &fftypes.Message{
			Header: fftypes.MessageHeader{
				ID:   msgID,
				Type: fftypes.MessageTypeBroadcast,
			},
		},
	}

	mcb := wh.callbacks.(*eventsmocks.Callbacks)
	mcb.On("DeliveryResponse", mock.Anything, mock.MatchedBy(func(response *fftypes.EventDeliveryResponse) bool {
		assert.Equal(t, float64(502), response.Reply.InlineData[0].Value.JSONObject()["status"])
		assert.Regexp(t, "FF10257", response.Reply.InlineData[0].Value.JSONObject().GetObject("body")["error"])
		return true
	})).Return(nil)

	err := wh.DeliveryRequest(mock.Anything, sub, event, []*fftypes.Data{})
	assert.NoError(t, err)
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
	sub := &fftypes.Subscription{
		Options: fftypes.SubscriptionOptions{
			SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
				WithData: &yes,
			},
		},
	}
	to := sub.Options.TransportOptions()
	to["url"] = fmt.Sprintf("http://%s/myapi", server.Listener.Addr())
	to["reply"] = true
	event := &fftypes.EventDelivery{
		Event: fftypes.Event{
			ID: fftypes.NewUUID(),
		},
		Subscription: fftypes.SubscriptionRef{
			ID: sub.ID,
		},
		Message: &fftypes.Message{
			Header: fftypes.MessageHeader{
				ID:   msgID,
				Type: fftypes.MessageTypeBroadcast,
			},
		},
	}

	mcb := wh.callbacks.(*eventsmocks.Callbacks)
	mcb.On("DeliveryResponse", mock.Anything, mock.MatchedBy(func(response *fftypes.EventDeliveryResponse) bool {
		assert.Equal(t, *msgID, *response.Reply.Message.Header.CID)
		assert.Nil(t, response.Reply.Message.Header.Group)
		assert.Equal(t, fftypes.MessageTypeBroadcast, response.Reply.Message.Header.Type)
		assert.Equal(t, float64(500), response.Reply.InlineData[0].Value.JSONObject()["status"])
		assert.Equal(t, `c29tZSBieXRlcw==`, response.Reply.InlineData[0].Value.JSONObject()["body"]) // base64 val
		return true
	})).Return(nil)

	err := wh.DeliveryRequest(mock.Anything, sub, event, []*fftypes.Data{
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(`"value1"`)},
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(`"value2"`)},
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

	sub := &fftypes.Subscription{}
	to := sub.Options.TransportOptions()
	to["url"] = fmt.Sprintf("http://%s/myapi", server.Listener.Addr())
	to["reply"] = true
	event := &fftypes.EventDelivery{
		Event: fftypes.Event{
			ID: fftypes.NewUUID(),
		},
		Subscription: fftypes.SubscriptionRef{
			ID: sub.ID,
		},
		Message: &fftypes.Message{
			Header: fftypes.MessageHeader{
				ID:   msgID,
				Type: fftypes.MessageTypeBroadcast,
			},
		},
	}

	mcb := wh.callbacks.(*eventsmocks.Callbacks)
	mcb.On("DeliveryResponse", mock.Anything, mock.MatchedBy(func(response *fftypes.EventDeliveryResponse) bool {
		assert.Equal(t, *msgID, *response.Reply.Message.Header.CID)
		assert.Nil(t, response.Reply.Message.Header.Group)
		assert.Equal(t, fftypes.MessageTypeBroadcast, response.Reply.Message.Header.Type)
		assert.Equal(t, float64(502), response.Reply.InlineData[0].Value.JSONObject()["status"])
		assert.NotEmpty(t, response.Reply.InlineData[0].Value.JSONObject().GetObject("body")["error"])
		return true
	})).Return(nil)

	err := wh.DeliveryRequest(mock.Anything, sub, event, []*fftypes.Data{
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(`"value1"`)},
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(`"value2"`)},
	})
	assert.NoError(t, err)

	mcb.AssertExpectations(t)
}

func TestRequestReplyBuildRequestFailFastAsk(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	msgID := fftypes.NewUUID()
	r := mux.NewRouter()
	server := httptest.NewServer(r)
	server.Close()

	sub := &fftypes.Subscription{}
	sub.Options.TransportOptions()["reply"] = true
	sub.Options.TransportOptions()["fastack"] = true
	event := &fftypes.EventDelivery{
		Event: fftypes.Event{
			ID: fftypes.NewUUID(),
		},
		Subscription: fftypes.SubscriptionRef{
			ID: sub.ID,
		},
		Message: &fftypes.Message{
			Header: fftypes.MessageHeader{
				ID:   msgID,
				Type: fftypes.MessageTypeBroadcast,
			},
		},
	}

	waiter := make(chan struct{})
	mcb := wh.callbacks.(*eventsmocks.Callbacks)
	dr := mcb.On("DeliveryResponse", mock.Anything, mock.MatchedBy(func(response *fftypes.EventDeliveryResponse) bool {
		assert.Equal(t, *msgID, *response.Reply.Message.Header.CID)
		assert.Nil(t, response.Reply.Message.Header.Group)
		assert.Equal(t, fftypes.MessageTypeBroadcast, response.Reply.Message.Header.Type)
		assert.Equal(t, float64(502), response.Reply.InlineData[0].Value.JSONObject()["status"])
		assert.Regexp(t, "FF10242", response.Reply.InlineData[0].Value.JSONObject().GetObject("body")["error"])
		return true
	})).Return(nil)
	dr.RunFn = func(a mock.Arguments) {
		close(waiter)
	}

	err := wh.DeliveryRequest(mock.Anything, sub, event, []*fftypes.Data{
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(`"value1"`)},
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(`"value2"`)},
	})
	assert.NoError(t, err)
	<-waiter

	mcb.AssertExpectations(t)
}

func TestDeliveryRequestNilMessage(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	yes := true
	sub := &fftypes.Subscription{
		Options: fftypes.SubscriptionOptions{
			SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
				WithData: &yes,
			},
		},
	}
	sub.Options.TransportOptions()["reply"] = true
	event := &fftypes.EventDelivery{
		Event: fftypes.Event{
			ID: fftypes.NewUUID(),
		},
		Subscription: fftypes.SubscriptionRef{
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
	sub := &fftypes.Subscription{
		Options: fftypes.SubscriptionOptions{
			SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
				WithData: &yes,
			},
		},
	}
	sub.Options.TransportOptions()["reply"] = true
	event := &fftypes.EventDelivery{
		Event: fftypes.Event{
			ID: fftypes.NewUUID(),
		},
		Subscription: fftypes.SubscriptionRef{
			ID: sub.ID,
		},
		Message: &fftypes.Message{
			Header: fftypes.MessageHeader{
				ID:   fftypes.NewUUID(),
				Type: fftypes.MessageTypeBroadcast,
				CID:  fftypes.NewUUID(),
			},
		},
	}

	mcb := wh.callbacks.(*eventsmocks.Callbacks)
	mcb.On("DeliveryResponse", mock.Anything, mock.MatchedBy(func(response *fftypes.EventDeliveryResponse) bool {
		return !response.Rejected // should be accepted as a no-op so we can move on to other events
	}))

	err := wh.DeliveryRequest(mock.Anything, sub, event, nil)
	assert.NoError(t, err)
}
