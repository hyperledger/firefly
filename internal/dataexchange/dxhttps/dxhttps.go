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

package dxhttps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/restclient"
	"github.com/kaleido-io/firefly/internal/wsclient"
	"github.com/kaleido-io/firefly/pkg/dataexchange"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

type HTTPS struct {
	ctx          context.Context
	capabilities *dataexchange.Capabilities
	callbacks    dataexchange.Callbacks
	client       *resty.Client
	wsconn       wsclient.WSClient
}

type wsEvent struct {
	Type      msgType `json:"type"`
	Sender    string  `json:"sender"`
	Recipient string  `json:"recipient"`
	RequestID string  `json:"requestID"`
	Path      string  `json:"path"`
	Message   string  `json:"message"`
	Hash      string  `json:"hash"`
	Error     string  `json:"error"`
}

type msgType string

const (
	messageReceived  msgType = "message-received"
	messageDelivered msgType = "message-delivered"
	messageFailed    msgType = "message-failed"
	blobReceived     msgType = "blob-received"
	blobDelivered    msgType = "blob-delivered"
	blobFailed       msgType = "blob-failed"
)

type responseWithRequestID struct {
	RequestID string `json:"requestID"`
}

type sendMessage struct {
	Message   string `json:"message"`
	Recipient string `json:"recipient"`
}

type transferBlob struct {
	Path      string `json:"path"`
	Recipient string `json:"recipient"`
}

func (h *HTTPS) Name() string {
	return "https"
}

func (h *HTTPS) Init(ctx context.Context, prefix config.Prefix, callbacks dataexchange.Callbacks) (err error) {
	h.ctx = log.WithLogField(ctx, "dx", "https")
	h.callbacks = callbacks
	h.client = restclient.New(h.ctx, prefix)
	h.capabilities = &dataexchange.Capabilities{}
	h.wsconn, err = wsclient.New(ctx, prefix, nil)
	if err != nil {
		return err
	}
	go h.eventLoop()
	return nil
}

func (h *HTTPS) Start() error {
	return h.wsconn.Connect()
}

func (h *HTTPS) Capabilities() *dataexchange.Capabilities {
	return h.capabilities
}

func (h *HTTPS) GetEndpointInfo(ctx context.Context) (peerID string, endpoint fftypes.JSONObject, err error) {
	res, err := h.client.R().SetContext(ctx).
		SetResult(&endpoint).
		Get("/api/v1/id")
	if err != nil || !res.IsSuccess() {
		return peerID, endpoint, restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	return endpoint.GetString("id"), endpoint, nil
}

func (h *HTTPS) AddPeer(ctx context.Context, node *fftypes.Node) (err error) {
	res, err := h.client.R().SetContext(ctx).
		SetBody(node.DX.Endpoint).
		Put(fmt.Sprintf("/api/v1/peers/%s", node.DX.Peer))
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	return nil
}

func (h *HTTPS) UploadBLOB(ctx context.Context, ns string, id fftypes.UUID, content io.Reader) (err error) {
	res, err := h.client.R().SetContext(ctx).
		SetFileReader("file", id.String(), content).
		Put(fmt.Sprintf("/api/v1/blobs/%s/%s", ns, &id))
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	return nil
}

func (h *HTTPS) DownloadBLOB(ctx context.Context, ns string, id fftypes.UUID) (content io.ReadCloser, err error) {
	res, err := h.client.R().SetContext(ctx).
		SetDoNotParseResponse(true).
		Get(fmt.Sprintf("/api/v1/blobs/%s/%s", ns, &id))
	if err != nil || !res.IsSuccess() {
		if err == nil {
			_ = res.RawBody().Close()
		}
		return nil, restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	return res.RawBody(), nil
}

func (h *HTTPS) SendMessage(ctx context.Context, node *fftypes.Node, data []byte) (trackingID string, err error) {
	var responseData responseWithRequestID
	res, err := h.client.R().SetContext(ctx).
		SetBody(&sendMessage{
			Message:   string(data),
			Recipient: node.DX.Peer,
		}).
		SetResult(&responseData).
		Post("/api/v1/message")
	if err != nil || !res.IsSuccess() {
		return "", restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	return responseData.RequestID, nil
}

func (h *HTTPS) TransferBLOB(ctx context.Context, node *fftypes.Node, ns string, id fftypes.UUID) (trackingID string, err error) {
	var responseData responseWithRequestID
	res, err := h.client.R().SetContext(ctx).
		SetBody(&transferBlob{
			Path:      fmt.Sprintf("%s/%s", ns, id),
			Recipient: node.DX.Peer,
		}).
		SetResult(&responseData).
		Post("/api/v1/transfers")
	if err != nil || !res.IsSuccess() {
		return "", restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	return responseData.RequestID, nil
}

func (h *HTTPS) extractBlobPath(ctx context.Context, path string) (ns string, id *fftypes.UUID) {
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		log.L(ctx).Errorf("Invalid blob path: %s", path)
		return "", nil
	}
	ns = parts[0]
	if err := fftypes.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
		log.L(ctx).Errorf("Invalid blob namespace: %s", path)
		return "", nil
	}
	id, err := fftypes.ParseUUID(ctx, parts[1])
	if err != nil {
		log.L(ctx).Errorf("Invalid blob UUID: %s", path)
		return "", nil
	}
	return ns, id
}

func (h *HTTPS) eventLoop() {
	l := log.L(h.ctx).WithField("role", "event-loop")
	ctx := log.WithLogger(h.ctx, l)
	ack, _ := json.Marshal(map[string]string{"action": "commit"})
	for {
		select {
		case <-ctx.Done():
			l.Debugf("Event loop exiting (context cancelled)")
			return
		case msgBytes, ok := <-h.wsconn.Receive():
			if !ok {
				l.Debugf("Event loop exiting (receive channel closed)")
				return
			}

			l.Tracef("DX message: %s", msgBytes)
			var msg wsEvent
			err := json.Unmarshal(msgBytes, &msg)
			if err != nil {
				l.Errorf("Message cannot be parsed as JSON: %s\n%s", err, string(msgBytes))
				continue // Swallow this and move on
			}
			l.Debugf("Received %s event from DX sender=%s", msg.Type, msg.Sender)
			switch msg.Type {
			case messageFailed:
				h.callbacks.TransferResult(msg.RequestID, fftypes.OpStatusFailed, msg.Error)
			case messageDelivered:
				h.callbacks.TransferResult(msg.RequestID, fftypes.OpStatusSucceeded, "")
			case messageReceived:
				h.callbacks.MessageReceived(msg.Sender, fftypes.Byteable(msg.Message))
			case blobFailed:
				h.callbacks.TransferResult(msg.RequestID, fftypes.OpStatusFailed, msg.Error)
			case blobDelivered:
				h.callbacks.TransferResult(msg.RequestID, fftypes.OpStatusSucceeded, "")
			case blobReceived:
				if ns, id := h.extractBlobPath(ctx, msg.Path); id != nil {
					h.callbacks.BLOBReceived(msg.Sender, ns, *id)
				}
			default:
				l.Errorf("Message unexpected: %s", msg.Type)
			}

			// Send the ack - only fails if shutting down
			err = h.wsconn.Send(ctx, ack)
			if err != nil {
				l.Errorf("Event loop exiting: %s", err)
				return
			}
		}
	}
}
