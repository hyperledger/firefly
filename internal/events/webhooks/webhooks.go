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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/restclient"
	"github.com/hyperledger-labs/firefly/pkg/events"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

type WebHooks struct {
	ctx          context.Context
	capabilities *events.Capabilities
	callbacks    events.Callbacks
	client       *resty.Client
}

type whRequest struct {
	r         *resty.Request
	url       string
	method    string
	body      fftypes.JSONObject
	reply     bool
	forceJSON bool
}

type whResponse struct {
	Status  int                `json:"status"`
	Headers fftypes.JSONObject `json:"headers"`
	Body    fftypes.Byteable   `json:"body"`
}

func (wh *WebHooks) Name() string { return "webhooks" }

func (wh *WebHooks) Init(ctx context.Context, prefix config.Prefix, callbacks events.Callbacks) (err error) {
	*wh = WebHooks{
		ctx:          ctx,
		capabilities: &events.Capabilities{},
		callbacks:    callbacks,
		client:       restclient.New(ctx, prefix),
	}
	// We have a single logical connection, that matches all subscriptions
	return callbacks.RegisterConnection("*", func(sr fftypes.SubscriptionRef) bool { return true })
}

func (wh *WebHooks) Capabilities() *events.Capabilities {
	return wh.capabilities
}

func (wh *WebHooks) GetOptionsSchema() string {
	return fmt.Sprintf(`{
		"properties": {
			"url": {
				"type": "string",
				"description": "%s"
			},
			"method": {
				"type": "string",
				"description": "%s"
			},
			"json": {
				"type": "boolean",
				"description": "%s"
			},
			"reply": {
				"type": "boolean",
				"description": "%s"
			},
			"headers": {
				"type": "object",
				"description": "%s",
				"additionalProperties": {
					"type": "string"
				}
			},
			"query": {
				"type": "object",
				"description": "%s",
				"additionalProperties": {
					"type": "string"
				}
			},
			"input": {
				"type": "object",
				"properties": {
					"description": "%s",
					"query": {
						"type": "string",
						"description": "%s"
					},
					"headers": {
						"type": "string",
						"description": "%s"
					},
					"body": {
						"type": "string",
						"description": "%s"
					}
				}
			}
		}
	}`,
		i18n.Expand(wh.ctx, i18n.MsgWebhooksOptURL),
		i18n.Expand(wh.ctx, i18n.MsgWebhooksOptMethod),
		i18n.Expand(wh.ctx, i18n.MsgWebhooksOptJSON),
		i18n.Expand(wh.ctx, i18n.MsgWebhooksOptReply),
		i18n.Expand(wh.ctx, i18n.MsgWebhooksOptHeaders),
		i18n.Expand(wh.ctx, i18n.MsgWebhooksOptQuery),
		i18n.Expand(wh.ctx, i18n.MsgWebhooksOptInput),
		i18n.Expand(wh.ctx, i18n.MsgWebhooksOptInputQuery),
		i18n.Expand(wh.ctx, i18n.MsgWebhooksOptInputHeaders),
		i18n.Expand(wh.ctx, i18n.MsgWebhooksOptInputBody),
	)
}

func (wh *WebHooks) buildRequest(options fftypes.JSONObject, firstData fftypes.JSONObject) (req *whRequest, err error) {
	req = &whRequest{
		r:         wh.client.R().SetDoNotParseResponse(true),
		url:       options.GetString("url"),
		method:    options.GetString("method"),
		forceJSON: options.GetBool("json"),
		reply:     options.GetBool("reply"),
	}
	if req.url == "" {
		return nil, i18n.NewError(wh.ctx, i18n.MsgWebhookURLEmpty)
	}
	if req.method == "" {
		req.method = http.MethodPost
	}
	headers := options.GetObject("headers")
	for h, v := range headers {
		s, ok := v.(string)
		if !ok {
			return nil, i18n.NewError(wh.ctx, i18n.MsgWebhookInvalidStringMap, "headers", h, v)
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
			return nil, i18n.NewError(wh.ctx, i18n.MsgWebhookInvalidStringMap, "query", q, v)
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
	}
	return req, err
}

func (wh *WebHooks) ValidateOptions(options *fftypes.SubscriptionOptions) error {
	if options.WithData == nil {
		defaultTrue := true
		options.WithData = &defaultTrue
	}
	return nil
}

func (wh *WebHooks) attemptRequest(sub *fftypes.Subscription, event *fftypes.EventDelivery, data []*fftypes.Data) (req *whRequest, res *whResponse, err error) {

	withData := sub.Options.WithData != nil && *sub.Options.WithData
	allData := make([]fftypes.Byteable, 0, len(data))
	var firstData fftypes.JSONObject
	if withData {
		for _, d := range data {
			if d.Value != nil {
				allData = append(allData, d.Value)
			}
		}
		if len(allData) == 0 {
			firstData = fftypes.JSONObject{}
		} else {
			firstData = allData[0].JSONObject()
		}
	}

	req, err = wh.buildRequest(sub.Options.TransportOptions(), firstData)
	if err != nil {
		return nil, nil, err
	}

	if req.r.Method == http.MethodPost || req.r.Method == http.MethodPatch || req.r.Method == http.MethodPut {
		switch {
		case !withData:
			// We are just sending the event itself
			req.r.SetBody(&event)
		case req.body != nil:
			// We might have been told to extract a body from the first data record
			req.r.SetBody(&req.body)
		case len(allData) > 1:
			// We've got an array of data to POST
			req.r.SetBody(&allData)
		default:
			// Otherwise just send the first object directly
			req.r.SetBody(&firstData)
		}
	}

	resp, err := req.r.Execute(req.method, req.url)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.RawBody().Close() }()

	res = &whResponse{
		Status:  resp.StatusCode(),
		Headers: fftypes.JSONObject{},
	}
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
			return nil, nil, err
		}
		res.Body, _ = json.Marshal(&resData) // we know we can re-marshal it
	} else {
		// Anything other than JSON, gets returned as a JSON string in base64 encoding
		buf := &bytes.Buffer{}
		buf.WriteRune('"')
		b64Encoder := base64.NewEncoder(base64.StdEncoding, buf)
		_, err := io.Copy(b64Encoder, resp.RawBody())
		if err != nil {
			return nil, nil, err
		}
		buf.WriteRune('"')
		res.Body = buf.Bytes()
	}

	return req, res, nil
}

func (wh *WebHooks) doDelivery(connID string, sub *fftypes.Subscription, event *fftypes.EventDelivery, data []*fftypes.Data) error {
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
			Body: b,
		}
	}
	b, _ := json.MarshalIndent(&res, "", "  ")
	log.L(wh.ctx).Tracef("Webhook response: %s", string(b))

	// Emit the response
	if req.reply {
		return wh.callbacks.DeliveryResponse(connID, &fftypes.EventDeliveryResponse{
			ID:           event.ID,
			Rejected:     false,
			Subscription: event.Subscription,
			Reply: &fftypes.MessageInput{
				Message: fftypes.Message{
					Header: fftypes.MessageHeader{
						CID: event.Message.Header.ID,
					},
				},
				InputData: fftypes.InputData{
					{Value: b},
				},
			},
		})
	}
	return nil
}

func (wh *WebHooks) DeliveryRequest(connID string, sub *fftypes.Subscription, event *fftypes.EventDelivery, data []*fftypes.Data) error {
	if event.Message == nil {
		log.L(wh.ctx).Warnf("Webhook subscription called with non-message event '%s'", event.ID)
		return nil
	}

	// In fastack mode we drive calls in parallel to the backend, immediately acknowledging the event
	if sub.Options.TransportOptions().GetBool("fastack") {
		go func() {
			err := wh.doDelivery(connID, sub, event, data)
			log.L(wh.ctx).Warnf("Webhook delivery failed in fastack mode for event '%s': %s", event.ID, err)
		}()
		return nil
	}

	return wh.doDelivery(connID, sub, event, data)
}
