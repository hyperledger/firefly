// Copyright © 2023 Kaleido, Inc.
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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/events"
)

type WebHooks struct {
	ctx           context.Context
	capabilities  *events.Capabilities
	callbacks     callbacks
	client        *resty.Client
	connID        string
	ffrestyConfig *ffresty.Config
}

type callbacks struct {
	writeLock sync.Mutex
	handlers  map[string]events.Callbacks
}

type whRequest struct {
	r         *resty.Request
	url       string
	method    string
	body      fftypes.JSONObject
	forceJSON bool
	replyTx   string
}

type whResponse struct {
	Status  int                `json:"status"`
	Headers fftypes.JSONObject `json:"headers"`
	Body    *fftypes.JSONAny   `json:"body"`
}

func (wh *WebHooks) Name() string { return "webhooks" }

func (wh *WebHooks) Init(ctx context.Context, config config.Section) (err error) {
	connID := fftypes.ShortID()

	ffrestyConfig, err := ffresty.GenerateConfig(ctx, config)
	if err != nil {
		return err
	}

	client := ffresty.NewWithConfig(ctx, *ffrestyConfig)

	*wh = WebHooks{
		ctx:          log.WithLogField(ctx, "webhook", wh.connID),
		capabilities: &events.Capabilities{},
		callbacks: callbacks{
			handlers: make(map[string]events.Callbacks),
		},
		client:        client,
		connID:        connID,
		ffrestyConfig: ffrestyConfig,
	}
	return nil
}

func (wh *WebHooks) SetHandler(namespace string, handler events.Callbacks) error {
	wh.callbacks.writeLock.Lock()
	defer wh.callbacks.writeLock.Unlock()
	if handler == nil {
		delete(wh.callbacks.handlers, namespace)
		return nil
	}
	wh.callbacks.handlers[namespace] = handler
	// We have a single logical connection, that matches all subscriptions
	return handler.RegisterConnection(wh.connID, func(sr core.SubscriptionRef) bool { return true })
}

func (wh *WebHooks) Capabilities() *events.Capabilities {
	return wh.capabilities
}

func (wh *WebHooks) buildRequest(ctx context.Context, restyClient *resty.Client, options fftypes.JSONObject, firstData fftypes.JSONObject) (req *whRequest, err error) {
	req = &whRequest{
		r: restyClient.R().
			SetDoNotParseResponse(true).
			SetContext(wh.ctx),
		url:       options.GetString("url"),
		method:    options.GetString("method"),
		forceJSON: options.GetBool("json"),
		replyTx:   options.GetString("replytx"),
	}
	if req.url == "" {
		return nil, i18n.NewError(ctx, coremsgs.MsgWebhookURLEmpty)
	}
	if req.method == "" {
		req.method = http.MethodPost
	}
	headers := options.GetObject("headers")
	for h, v := range headers {
		s, ok := v.(string)
		if !ok {
			return nil, i18n.NewError(ctx, coremsgs.MsgWebhookInvalidStringMap, "headers", h, v)
		}
		_ = req.r.SetHeader(h, s)
	}
	if req.r.Header.Get("Content-Type") == "" {
		req.r.Header.Set("Content-Type", "application/json")
	}
	// Static query support
	query := options.GetObject("query")
	for q, v := range query {
		s, ok := v.(string)
		if !ok {
			return nil, i18n.NewError(ctx, coremsgs.MsgWebhookInvalidStringMap, "query", q, v)
		}
		_ = req.r.SetQueryParam(q, s)
	}
	if firstData != nil {
		// Options on how to process the input
		input := options.GetObject("input")
		// Dynamic query support from input
		inputQuery := input.GetString("query")
		if inputQuery != "" {
			iq := firstData.GetObject(inputQuery)
			for q := range iq {
				_ = req.r.SetQueryParam(q, iq.GetString(q))
			}
		}
		// Dynamic header support from input
		inputHeaders := input.GetString("headers")
		if inputHeaders != "" {
			ih := firstData.GetObject(inputHeaders)
			for h := range ih {
				_ = req.r.SetHeader(h, ih.GetString(h))
			}
		}
		// Choose to sub-select a field to send as the body
		inputBody := input.GetString("body")
		if inputBody != "" {
			req.body = firstData.GetObject(inputBody)
		}
		// Choose to add an additional dynamic path
		inputPath := input.GetString("path")
		if inputPath != "" {
			extraPath := strings.TrimPrefix(firstData.GetString(inputPath), "/")
			if len(extraPath) > 0 {
				pathSegments := strings.Split(extraPath, "/")
				for _, ps := range pathSegments {
					req.url = strings.TrimSuffix(req.url, "/") + "/" + url.PathEscape(ps)
				}
			}
		}
		// Choose to add an additional dynamic path
		inputTxtype := input.GetString("replytx")
		if inputTxtype != "" {
			txType := firstData.GetString(inputTxtype)
			if len(txType) > 0 {
				req.replyTx = txType
				if strings.EqualFold(txType, "true") {
					req.replyTx = string(core.TransactionTypeBatchPin)
				}
			}
		}
	}
	return req, err
}

func (wh *WebHooks) ValidateOptions(ctx context.Context, options *core.SubscriptionOptions) error {
	if options.WithData == nil {
		defaultTrue := true
		options.WithData = &defaultTrue
	}

	newFFRestyConfig := ffresty.Config{}
	if wh.ffrestyConfig != nil {
		// Take a copy of the webhooks global resty config
		newFFRestyConfig = *wh.ffrestyConfig
	}
	if options.Retry.Enabled {
		newFFRestyConfig.Retry = true
		if options.Retry.Count > 0 {
			newFFRestyConfig.RetryCount = options.Retry.Count
		}

		if options.Retry.InitialDelay != "" {
			ffd, err := fftypes.ParseDurationString(options.Retry.InitialDelay, time.Millisecond)
			if err != nil {
				return err
			}
			newFFRestyConfig.RetryInitialDelay = time.Duration(ffd)
		}

		if options.Retry.MaximumDelay != "" {
			ffd, err := fftypes.ParseDurationString(options.Retry.MaximumDelay, time.Millisecond)
			if err != nil {
				return err
			}
			newFFRestyConfig.RetryMaximumDelay = time.Duration(ffd)
		}
	}

	if options.HTTPOptions.HTTPMaxIdleConns > 0 {
		newFFRestyConfig.HTTPMaxIdleConns = options.HTTPOptions.HTTPMaxIdleConns
	}

	if options.HTTPOptions.HTTPRequestTimeout != "" {
		ffd, err := fftypes.ParseDurationString(options.HTTPOptions.HTTPRequestTimeout, time.Millisecond)
		if err != nil {
			return err
		}
		newFFRestyConfig.HTTPRequestTimeout = time.Duration(ffd)
	}

	if options.HTTPOptions.HTTPIdleConnTimeout != "" {
		ffd, err := fftypes.ParseDurationString(options.HTTPOptions.HTTPIdleConnTimeout, time.Millisecond)
		if err != nil {
			return err
		}
		newFFRestyConfig.HTTPIdleConnTimeout = time.Duration(ffd)
	}

	if options.HTTPOptions.HTTPExpectContinueTimeout != "" {
		ffd, err := fftypes.ParseDurationString(options.HTTPOptions.HTTPExpectContinueTimeout, time.Millisecond)
		if err != nil {
			return err
		}
		newFFRestyConfig.HTTPExpectContinueTimeout = time.Duration(ffd)
	}

	if options.HTTPOptions.HTTPConnectionTimeout != "" {
		ffd, err := fftypes.ParseDurationString(options.HTTPOptions.HTTPConnectionTimeout, time.Millisecond)
		if err != nil {
			return err
		}
		newFFRestyConfig.HTTPConnectionTimeout = time.Duration(ffd)
	}

	if options.HTTPOptions.HTTPTLSHandshakeTimeout != "" {
		ffd, err := fftypes.ParseDurationString(options.HTTPOptions.HTTPTLSHandshakeTimeout, time.Millisecond)
		if err != nil {
			return err
		}
		newFFRestyConfig.HTTPTLSHandshakeTimeout = time.Duration(ffd)
	}

	if options.TLSConfig != nil {
		newFFRestyConfig.TLSClientConfig = options.TLSConfig
	}

	// NOTE: this is the plugin context, as the context passed through can be terminated as part of a
	// API call or anything else and we want to use this client later on!!
	// So these clients should live as long as the plugin exists
	options.RestyClient = ffresty.NewWithConfig(wh.ctx, newFFRestyConfig)

	_, err := wh.buildRequest(ctx, options.RestyClient, options.TransportOptions(), fftypes.JSONObject{})
	return err
}

func (wh *WebHooks) buildBody(withData bool, event *core.EventDelivery, data core.DataArray) (body *fftypes.JSONAny, firstData fftypes.JSONObject, err error) {
	allData := make([]*fftypes.JSONAny, 0, len(data))
	if withData {
		for _, d := range data {
			if d.Value != nil {
				allData = append(allData, d.Value)
			}
		}
		if len(allData) == 0 {
			firstData = fftypes.JSONObject{}
			return fftypes.JSONAnyPtr("{}"), firstData, nil
		} else {
			// Use JSONObjectOk instead of JSONObject
			// JSONObject fails for datatypes such as array, string, bool, number etc
			var valid bool
			firstData, valid = allData[0].JSONObjectOk()
			if !valid {
				firstData = fftypes.JSONObject{
					"value": allData[0],
				}
			}

			if len(allData) == 1 {
				encodedFirstData, err := json.Marshal(firstData)
				if err != nil {
					return nil, nil, err
				}
				return fftypes.JSONAnyPtrBytes(encodedFirstData), firstData, nil
			}
		}
		encodedData, err := json.Marshal(allData)
		if err != nil {
			return nil, nil, err
		}

		return fftypes.JSONAnyPtrBytes(encodedData), firstData, nil
	}

	encodedEvent, err := json.Marshal(event)
	if err != nil {
		return nil, nil, err
	}

	return fftypes.JSONAnyPtrBytes(encodedEvent), nil, nil
}

// func (wh *WebHooks) buildBody(events []*core.EventDelivery, data []core.DataArray, withData bool, batch bool) (body interface{}, firstData fftypes.JSONObject) {
// 	if len(events) == 0 && len(data) == 0 {
// 		return nil, nil
// 	}

// 	myArray := []*fftypes.JSONAny{}

// 	var foo *fftypes.JSONAny = myArray
// 	json

// 	fftypes.JSONAnyPtrBytes(stuff)

// 	if batch {
// 		if withData {
// 			if len(data) > 0 {
// 				allData := [][]*fftypes.JSONAny{}
// 				for _, eventData := range data {
// 					allEventData := []*fftypes.JSONAny{}
// 					for _, d := range eventData {
// 						if d.Value != nil {
// 							allEventData = append(allEventData, d.Value)
// 						}
// 					}
// 					allData = append(allData, allEventData)
// 				}
// 				return allData, nil
// 			}
// 		}

// 		// [{"event1": "stuff"},"event2": "stuff"}]
// 		// [{"event1": "stuff"}]
// 		return events, nil
// 	}

// 	if withData {
// 		if len(data) == 1 {
// 			eventData := []*fftypes.JSONAny{}
// 			for _, d := range data[0] {
// 				if d.Value != nil {
// 					eventData = append(eventData, d.Value)
// 				}
// 			}

// 			if len(eventData) == 0 {
// 				// Send an empty object if ask withData but no data available
// 				firstData = fftypes.JSONObject{}
// 				body = firstData
// 			}

// 			if len(eventData) >= 1 {
// 				// Use JSONObjectOk instead of JSONObject
// 				// JSONObject fails for datatypes such as array, string, bool, number etc
// 				var valid bool
// 				firstData, valid = eventData[0].JSONObjectOk()
// 				if !valid {
// 					firstData = fftypes.JSONObject{
// 						"value": eventData[0],
// 					}
// 				}

// 				if len(eventData) == 1 {
// 					body = firstData
// 				} else {
// 					body = eventData
// 				}
// 			}

// 			return body, firstData
// 		}
// 	}

// 	if body == nil && len(events) > 0 {
// 		// {"event1": "stuff"}
// 		body = events[0]
// 	}

// 	return body, firstData
// }

func (wh *WebHooks) attemptRequest(ctx context.Context, sub *core.Subscription, events []*core.EventDelivery, data []core.DataArray, batch bool) (req *whRequest, res *whResponse, err error) {
	withData := sub.Options.WithData != nil && *sub.Options.WithData

	var body *fftypes.JSONAny
	var firstData fftypes.JSONObject
	if len(events) == 1 && len(data) == 1 {
		body, firstData, err = wh.buildBody(withData, events[0], data[0])
		if err != nil {
			return nil, nil, err
		}
	} else {
		batchBody := []*fftypes.JSONAny{}
		for i := 0; i < len(events); i++ {
			eventBody, _, err := wh.buildBody(withData, events[i], data[i])
			if err != nil {
				return nil, nil, err
			}
			batchBody = append(batchBody, eventBody)
		}

		encodedBody, err := json.Marshal(batchBody)
		if err != nil {
			return nil, nil, err
		}

		body = fftypes.JSONAnyPtrBytes(encodedBody)
	}

	client := wh.client
	if sub.Options.RestyClient != nil {
		client = sub.Options.RestyClient
	}

	req, err = wh.buildRequest(ctx, client, sub.Options.TransportOptions(), firstData)
	if err != nil {
		return nil, nil, err
	}

	if req.method == http.MethodPost || req.method == http.MethodPatch || req.method == http.MethodPut {
		switch {
		case req.body != nil:
			// We might have been told to extract a body from the first data record
			req.r.SetBody(req.body)
		default:
			req.r.SetBody(body.Bytes())
		}
	}

	// log.L(wh.ctx).Debugf("Webhook-> %s %s event %s on subscription %s", req.method, req.url, event.ID, sub.ID)
	resp, err := req.r.Execute(req.method, req.url)
	if err != nil {
		// log.L(ctx).Errorf("Webhook<- %s %s event %s on subscription %s failed: %s", req.method, req.url, event.ID, sub.ID, err)
		return nil, nil, err
	}
	defer func() { _ = resp.RawBody().Close() }()

	res = &whResponse{
		Status:  resp.StatusCode(),
		Headers: fftypes.JSONObject{},
	}
	// log.L(wh.ctx).Infof("Webhook<- %s %s event %s on subscription %s returned %d", req.method, req.url, event.ID, sub.ID, res.Status)
	header := resp.Header()
	for h := range header {
		res.Headers[h] = header.Get(h)
	}
	contentType := header.Get("Content-Type")
	log.L(ctx).Debugf("Response content-type '%s' forceJSON=%t", contentType, req.forceJSON)
	if req.forceJSON {
		contentType = "application/json"
	}
	res.Headers["Content-Type"] = contentType
	if req.forceJSON || strings.HasPrefix(contentType, "application/json") {
		var resData interface{}
		err = json.NewDecoder(resp.RawBody()).Decode(&resData)
		if err != nil {
			return nil, nil, i18n.WrapError(ctx, err, coremsgs.MsgWebhooksReplyBadJSON)
		}
		b, _ := json.Marshal(&resData) // we know we can re-marshal It
		res.Body = fftypes.JSONAnyPtrBytes(b)
	} else {
		// Anything other than JSON, gets returned as a JSON string in base64 encoding
		buf := &bytes.Buffer{}
		buf.WriteByte('"')
		b64Encoder := base64.NewEncoder(base64.StdEncoding, buf)
		_, _ = io.Copy(b64Encoder, resp.RawBody())
		_ = b64Encoder.Close()
		buf.WriteByte('"')
		res.Body = fftypes.JSONAnyPtrBytes(buf.Bytes())
	}

	return req, res, nil
}

func (wh *WebHooks) doDelivery(ctx context.Context, connID string, reply bool, sub *core.Subscription, event *core.EventDelivery, data core.DataArray, fastAck bool) {
	req, res, gwErr := wh.attemptRequest(ctx, sub, []*core.EventDelivery{event}, []core.DataArray{data}, false)
	if gwErr != nil {
		// Generate a bad-gateway error response - we always want to send something back,
		// rather than just causing timeouts
		log.L(wh.ctx).Errorf("Failed to invoke webhook: %s", gwErr)
		b, _ := json.Marshal(&fftypes.RESTError{
			Error: gwErr.Error(),
		})
		res = &whResponse{
			Status: http.StatusBadGateway,
			Headers: fftypes.JSONObject{
				"Content-Type": "application/json",
			},
			Body: fftypes.JSONAnyPtrBytes(b),
		}
	}
	b, _ := json.Marshal(&res)
	log.L(wh.ctx).Tracef("Webhook response: %s", string(b))

	// Emit the response
	if reply && event.Message != nil {
		txType := fftypes.FFEnum(strings.ToLower(sub.Options.TransportOptions().GetString("replytx")))
		if req != nil && req.replyTx != "" {
			txType = fftypes.FFEnum(strings.ToLower(req.replyTx))
		}
		if cb, ok := wh.callbacks.handlers[sub.Namespace]; ok {
			log.L(wh.ctx).Debugf("Sending reply message for %s CID=%s", event.ID, event.Message.Header.ID)
			cb.DeliveryResponse(connID, &core.EventDeliveryResponse{
				ID:           event.ID,
				Rejected:     false,
				Subscription: event.Subscription,
				Reply: &core.MessageInOut{
					Message: core.Message{
						Header: core.MessageHeader{
							CID:    event.Message.Header.ID,
							Group:  event.Message.Header.Group,
							Type:   event.Message.Header.Type,
							Topics: event.Message.Header.Topics,
							Tag:    sub.Options.TransportOptions().GetString("replytag"),
							TxType: txType,
						},
					},
					InlineData: core.InlineData{
						{Value: fftypes.JSONAnyPtrBytes(b)},
					},
				},
			})
		}
	} else if !fastAck {
		if cb, ok := wh.callbacks.handlers[sub.Namespace]; ok {
			cb.DeliveryResponse(connID, &core.EventDeliveryResponse{
				ID:           event.ID,
				Rejected:     false,
				Subscription: event.Subscription,
			})
		}
	}
}

func (wh *WebHooks) doBatchedDelivery(ctx context.Context, connID string, reply bool, sub *core.Subscription, events []*core.EventDelivery, data []core.DataArray, fastAck bool) {
	req, res, gwErr := wh.attemptRequest(ctx, sub, events, data, true)
	if gwErr != nil {
		// Generate a bad-gateway error response - we always want to send something back,
		// rather than just causing timeouts
		log.L(wh.ctx).Errorf("Failed to invoke webhook: %s", gwErr)
		b, _ := json.Marshal(&fftypes.RESTError{
			Error: gwErr.Error(),
		})
		res = &whResponse{
			Status: http.StatusBadGateway,
			Headers: fftypes.JSONObject{
				"Content-Type": "application/json",
			},
			Body: fftypes.JSONAnyPtrBytes(b),
		}
	}
	b, _ := json.Marshal(&res)
	log.L(wh.ctx).Tracef("Webhook response: %s", string(b))

	// For each event emit a response
	for _, event := range events {
		// Emit the response
		if reply && event.Message != nil {
			txType := fftypes.FFEnum(strings.ToLower(sub.Options.TransportOptions().GetString("replytx")))
			if req != nil && req.replyTx != "" {
				txType = fftypes.FFEnum(strings.ToLower(req.replyTx))
			}
			if cb, ok := wh.callbacks.handlers[sub.Namespace]; ok {
				log.L(wh.ctx).Debugf("Sending reply message for %s CID=%s", event.ID, event.Message.Header.ID)
				cb.DeliveryResponse(connID, &core.EventDeliveryResponse{
					ID:           event.ID,
					Rejected:     false,
					Subscription: event.Subscription,
					Reply: &core.MessageInOut{
						Message: core.Message{
							Header: core.MessageHeader{
								CID:    event.Message.Header.ID,
								Group:  event.Message.Header.Group,
								Type:   event.Message.Header.Type,
								Topics: event.Message.Header.Topics,
								Tag:    sub.Options.TransportOptions().GetString("replytag"),
								TxType: txType,
							},
						},
						InlineData: core.InlineData{
							{Value: fftypes.JSONAnyPtrBytes(b)},
						},
					},
				})
			}
		} else if !fastAck {
			if cb, ok := wh.callbacks.handlers[sub.Namespace]; ok {
				cb.DeliveryResponse(connID, &core.EventDeliveryResponse{
					ID:           event.ID,
					Rejected:     false,
					Subscription: event.Subscription,
				})
			}
		}
	}
}
func (wh *WebHooks) DeliveryRequest(ctx context.Context, connID string, sub *core.Subscription, event *core.EventDelivery, data core.DataArray) error {
	reply := sub.Options.TransportOptions().GetBool("reply")
	if reply && event.Message != nil && event.Message.Header.CID != nil {
		// We cowardly refuse to dispatch a message that is itself a reply, as it's hard for users to
		// avoid loops - and there's no way for us to detect here if a user has configured correctly
		// to avoid a loop.
		log.L(wh.ctx).Debugf("Webhook subscription with reply enabled called with reply event '%s'", event.ID)
		if cb, ok := wh.callbacks.handlers[sub.Namespace]; ok {
			cb.DeliveryResponse(connID, &core.EventDeliveryResponse{
				ID:           event.ID,
				Rejected:     false,
				Subscription: event.Subscription,
			})
		}
		return nil
	}

	// In fastack mode we drive calls in parallel to the backend, immediately acknowledging the event
	// NOTE: We cannot use this with reply mode, as when we're sending a reply the `DeliveryResponse`
	//       callback must include the reply in-line.
	if !reply && sub.Options.TransportOptions().GetBool("fastack") {
		if cb, ok := wh.callbacks.handlers[sub.Namespace]; ok {
			cb.DeliveryResponse(connID, &core.EventDeliveryResponse{
				ID:           event.ID,
				Rejected:     false,
				Subscription: event.Subscription,
			})
		}
		go wh.doDelivery(ctx, connID, reply, sub, event, data, true)
		return nil
	}

	// NOTE: We could check here for batching and accumulate but we can't return because this causes the offset to jump...

	// TODO we don't look at the error here?
	wh.doDelivery(ctx, connID, reply, sub, event, data, false)
	return nil
}

func (wh *WebHooks) BatchDeliveryRequest(ctx context.Context, connID string, sub *core.Subscription, events []*core.EventDelivery, data []core.DataArray) error {
	reply := sub.Options.TransportOptions().GetBool("reply")
	if reply {
		nonReplyEvents := []*core.EventDelivery{}
		for _, event := range events {
			// We cowardly refuse to dispatch a message that is itself a reply, as it's hard for users to
			// avoid loops - and there's no way for us to detect here if a user has configured correctly
			// to avoid a loop.
			if event.Message != nil && event.Message.Header.CID != nil {
				log.L(wh.ctx).Debugf("Webhook subscription with reply enabled called with reply event '%s'", event.ID)
				if cb, ok := wh.callbacks.handlers[sub.Namespace]; ok {
					cb.DeliveryResponse(connID, &core.EventDeliveryResponse{
						ID:           event.ID,
						Rejected:     false,
						Subscription: event.Subscription,
					})
				}
				continue
			}

			nonReplyEvents = append(nonReplyEvents, event)
		}
		// Override the events to send without the reply events
		events = nonReplyEvents
	}

	// // In fastack mode we drive calls in parallel to the backend, immediately acknowledging the event
	// NOTE: We cannot use this with reply mode, as when we're sending a reply the `DeliveryResponse`
	//       callback must include the reply in-line.
	if !reply && sub.Options.TransportOptions().GetBool("fastack") {
		for _, event := range events {
			if cb, ok := wh.callbacks.handlers[sub.Namespace]; ok {
				cb.DeliveryResponse(connID, &core.EventDeliveryResponse{
					ID:           event.ID,
					Rejected:     false,
					Subscription: event.Subscription,
				})
			}
		}
		go wh.doBatchedDelivery(ctx, connID, reply, sub, events, data, true)
		return nil
	}

	wh.doBatchedDelivery(ctx, connID, reply, sub, events, data, false)
	return nil
}

func (wh *WebHooks) NamespaceRestarted(ns string, startTime time.Time) {
	// no-op
}
