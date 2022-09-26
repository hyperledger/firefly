// Copyright © 2022 Kaleido, Inc.
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
	ctx          context.Context
	capabilities *events.Capabilities
	callbacks    map[string]events.Callbacks
	client       *resty.Client
	connID       string
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
	*wh = WebHooks{
		ctx:          log.WithLogField(ctx, "webhook", wh.connID),
		capabilities: &events.Capabilities{},
		callbacks:    make(map[string]events.Callbacks),
		client:       ffresty.New(ctx, config),
		connID:       connID,
	}
	return nil
}

func (wh *WebHooks) SetHandler(namespace string, handler events.Callbacks) error {
	wh.callbacks[namespace] = handler
	// We have a single logical connection, that matches all subscriptions
	return handler.RegisterConnection(wh.connID, func(sr core.SubscriptionRef) bool { return true })
}

func (wh *WebHooks) Capabilities() *events.Capabilities {
	return wh.capabilities
}

func (wh *WebHooks) buildRequest(options fftypes.JSONObject, firstData fftypes.JSONObject) (req *whRequest, err error) {
	req = &whRequest{
		r: wh.client.R().
			SetDoNotParseResponse(true).
			SetContext(wh.ctx),
		url:       options.GetString("url"),
		method:    options.GetString("method"),
		forceJSON: options.GetBool("json"),
		replyTx:   options.GetString("replytx"),
	}
	if req.url == "" {
		return nil, i18n.NewError(wh.ctx, coremsgs.MsgWebhookURLEmpty)
	}
	if req.method == "" {
		req.method = http.MethodPost
	}
	headers := options.GetObject("headers")
	for h, v := range headers {
		s, ok := v.(string)
		if !ok {
			return nil, i18n.NewError(wh.ctx, coremsgs.MsgWebhookInvalidStringMap, "headers", h, v)
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
			return nil, i18n.NewError(wh.ctx, coremsgs.MsgWebhookInvalidStringMap, "query", q, v)
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

func (wh *WebHooks) ValidateOptions(options *core.SubscriptionOptions) error {
	if options.WithData == nil {
		defaultTrue := true
		options.WithData = &defaultTrue
	}
	_, err := wh.buildRequest(options.TransportOptions(), fftypes.JSONObject{})
	return err
}

func (wh *WebHooks) attemptRequest(sub *core.Subscription, event *core.EventDelivery, data core.DataArray) (req *whRequest, res *whResponse, err error) {
	withData := sub.Options.WithData != nil && *sub.Options.WithData
	allData := make([]*fftypes.JSONAny, 0, len(data))
	var firstData fftypes.JSONObject
	var valid bool
	if withData {
		for _, d := range data {
			if d.Value != nil {
				allData = append(allData, d.Value)
			}
		}
		if len(allData) == 0 {
			firstData = fftypes.JSONObject{}
		} else {
			// Use JSONObjectOk instead of JSONObject
			// JSONObject fails for datatypes such as array, string, bool, number etc
			firstData, valid = allData[0].JSONObjectOk()
			if !valid {
				firstData = fftypes.JSONObject{
					"value": allData[0],
				}
			}
		}
	}

	req, err = wh.buildRequest(sub.Options.TransportOptions(), firstData)
	if err != nil {
		return nil, nil, err
	}

	if req.method == http.MethodPost || req.method == http.MethodPatch || req.method == http.MethodPut {
		switch {
		case !withData:
			// We are just sending the event itself
			req.r.SetBody(event)
		case req.body != nil:
			// We might have been told to extract a body from the first data record
			req.r.SetBody(req.body)
		case len(allData) > 1:
			// We've got an array of data to POST
			req.r.SetBody(allData)
		default:
			// Otherwise just send the first object directly
			req.r.SetBody(firstData)
		}
	}

	log.L(wh.ctx).Debugf("Webhook-> %s %s event %s on subscription %s", req.method, req.url, event.ID, sub.ID)
	resp, err := req.r.Execute(req.method, req.url)
	if err != nil {
		log.L(wh.ctx).Errorf("Webhook<- %s %s event %s on subscription %s failed: %s", req.method, req.url, event.ID, sub.ID, err)
		return nil, nil, err
	}
	defer func() { _ = resp.RawBody().Close() }()

	res = &whResponse{
		Status:  resp.StatusCode(),
		Headers: fftypes.JSONObject{},
	}
	log.L(wh.ctx).Infof("Webhook<- %s %s event %s on subscription %s returned %d", req.method, req.url, event.ID, sub.ID, res.Status)
	header := resp.Header()
	for h := range header {
		res.Headers[h] = header.Get(h)
	}
	contentType := header.Get("Content-Type")
	log.L(wh.ctx).Debugf("Response content-type '%s' forceJSON=%t", contentType, req.forceJSON)
	if req.forceJSON {
		contentType = "application/json"
	}
	res.Headers["Content-Type"] = contentType
	if req.forceJSON || strings.HasPrefix(contentType, "application/json") {
		var resData interface{}
		err = json.NewDecoder(resp.RawBody()).Decode(&resData)
		if err != nil {
			return nil, nil, i18n.WrapError(wh.ctx, err, coremsgs.MsgWebhooksReplyBadJSON)
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

func (wh *WebHooks) doDelivery(connID string, reply bool, sub *core.Subscription, event *core.EventDelivery, data core.DataArray, fastAck bool) error {
	req, res, gwErr := wh.attemptRequest(sub, event, data)
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
	if reply {
		txType := fftypes.FFEnum(strings.ToLower(sub.Options.TransportOptions().GetString("replytx")))
		if req != nil && req.replyTx != "" {
			txType = fftypes.FFEnum(strings.ToLower(req.replyTx))
		}
		if cb, ok := wh.callbacks[sub.Namespace]; ok {
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
		if cb, ok := wh.callbacks[sub.Namespace]; ok {
			cb.DeliveryResponse(connID, &core.EventDeliveryResponse{
				ID:           event.ID,
				Rejected:     false,
				Subscription: event.Subscription,
			})
		}
	}
	return nil
}

func (wh *WebHooks) DeliveryRequest(connID string, sub *core.Subscription, event *core.EventDelivery, data core.DataArray) error {
	if event.Message == nil && sub.Options.WithData != nil && *sub.Options.WithData {
		log.L(wh.ctx).Debugf("Webhook withData=true subscription called with non-message event '%s'", event.ID)
		return nil
	}

	reply := sub.Options.TransportOptions().GetBool("reply")
	if reply && event.Message.Header.CID != nil {
		// We cowardly refuse to dispatch a message that is itself a reply, as it's hard for users to
		// avoid loops - and there's no way for us to detect here if a user has configured correctly
		// to avoid a loop.
		log.L(wh.ctx).Debugf("Webhook subscription with reply enabled called with reply event '%s'", event.ID)
		if cb, ok := wh.callbacks[sub.Namespace]; ok {
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
		if cb, ok := wh.callbacks[sub.Namespace]; ok {
			cb.DeliveryResponse(connID, &core.EventDeliveryResponse{
				ID:           event.ID,
				Rejected:     false,
				Subscription: event.Subscription,
			})
		}
		go func() {
			err := wh.doDelivery(connID, reply, sub, event, data, true)
			log.L(wh.ctx).Warnf("Webhook delivery failed in fastack mode for event '%s': %s", event.ID, err)
		}()
		return nil
	}

	return wh.doDelivery(connID, reply, sub, event, data, false)
}
