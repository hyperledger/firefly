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
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/restclient"
	"github.com/hyperledger-labs/firefly/internal/wsclient"
	"github.com/hyperledger-labs/firefly/pkg/dataexchange"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
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

const (
	dxHTTPHeaderHash = "dx-hash"
)

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

type uploadBlob struct {
	Hash       string      `json:"hash"`
	LastUpdate json.Number `json:"lastUpdate"`
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

	if prefix.GetString(restclient.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "url", "dataexchange.https")
	}

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

func (h *HTTPS) AddPeer(ctx context.Context, peerID string, endpoint fftypes.JSONObject) (err error) {
	res, err := h.client.R().SetContext(ctx).
		SetBody(endpoint).
		Put(fmt.Sprintf("/api/v1/peers/%s", peerID))
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	return nil
}

func (h *HTTPS) UploadBLOB(ctx context.Context, ns string, id fftypes.UUID, content io.Reader) (payloadRef string, hash *fftypes.Bytes32, err error) {
	payloadRef = fmt.Sprintf("%s/%s", ns, &id)
	var upload uploadBlob
	res, err := h.client.R().SetContext(ctx).
		SetFileReader("file", id.String(), content).
		SetResult(&upload).
		Put(fmt.Sprintf("/api/v1/blobs/%s", payloadRef))
	if err != nil || !res.IsSuccess() {
		err = restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
		return "", nil, err
	}
	if hash, err = fftypes.ParseBytes32(ctx, upload.Hash); err != nil {
		return "", nil, i18n.WrapError(ctx, err, i18n.MsgDXBadResponse, "hash", upload.Hash)
	}
	return payloadRef, hash, nil
}

func (h *HTTPS) DownloadBLOB(ctx context.Context, payloadRef string) (content io.ReadCloser, err error) {
	res, err := h.client.R().SetContext(ctx).
		SetDoNotParseResponse(true).
		Get(fmt.Sprintf("/api/v1/blobs/%s", payloadRef))
	if err != nil || !res.IsSuccess() {
		if err == nil {
			_ = res.RawBody().Close()
		}
		return nil, restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	return res.RawBody(), nil
}

func (h *HTTPS) SendMessage(ctx context.Context, peerID string, data []byte) (trackingID string, err error) {
	var responseData responseWithRequestID
	res, err := h.client.R().SetContext(ctx).
		SetBody(&sendMessage{
			Message:   string(data),
			Recipient: peerID,
		}).
		SetResult(&responseData).
		Post("/api/v1/messages")
	if err != nil || !res.IsSuccess() {
		return "", restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	return responseData.RequestID, nil
}

func (h *HTTPS) TransferBLOB(ctx context.Context, peerID, payloadRef string) (trackingID string, err error) {
	var responseData responseWithRequestID
	res, err := h.client.R().SetContext(ctx).
		SetBody(&transferBlob{
			Path:      payloadRef,
			Recipient: peerID,
		}).
		SetResult(&responseData).
		Post("/api/v1/transfers")
	if err != nil || !res.IsSuccess() {
		return "", restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	return responseData.RequestID, nil
}

func (h *HTTPS) CheckBLOBReceived(ctx context.Context, peerID, ns string, id fftypes.UUID) (hash *fftypes.Bytes32, err error) {
	var responseData responseWithRequestID
	res, err := h.client.R().SetContext(ctx).
		SetResult(&responseData).
		Head(fmt.Sprintf("/api/v1/blobs/%s/%s/%s", peerID, ns, id.String()))
	if err == nil && res.StatusCode() == http.StatusNotFound {
		return nil, nil
	}
	if err != nil || !res.IsSuccess() {
		return nil, restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	hashString := res.Header().Get(dxHTTPHeaderHash)
	if hash, err = fftypes.ParseBytes32(ctx, hashString); err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDXBadResponse, "hash", hashString)
	}
	return hash, nil
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
				err = h.callbacks.TransferResult(msg.RequestID, fftypes.OpStatusFailed, msg.Error, nil)
			case messageDelivered:
				err = h.callbacks.TransferResult(msg.RequestID, fftypes.OpStatusSucceeded, "", nil)
			case messageReceived:
				err = h.callbacks.MessageReceived(msg.Sender, fftypes.Byteable(msg.Message))
			case blobFailed:
				err = h.callbacks.TransferResult(msg.RequestID, fftypes.OpStatusFailed, msg.Error, nil)
			case blobDelivered:
				err = h.callbacks.TransferResult(msg.RequestID, fftypes.OpStatusSucceeded, "", nil)
			case blobReceived:
				var hash *fftypes.Bytes32
				hash, err = fftypes.ParseBytes32(ctx, msg.Hash)
				if err != nil {
					l.Errorf("Invalid hash received in DX event: '%s'", msg.Hash)
					err = nil // still confirm the message
				} else {
					err = h.callbacks.BLOBReceived(msg.Sender, *hash, msg.Path)
				}
			default:
				l.Errorf("Message unexpected: %s", msg.Type)
			}

			// Send the ack - as long as we didn't fail processing (which should only happen in core
			// if core itself is shutting down)
			if err == nil {
				err = h.wsconn.Send(ctx, ack)
			}
			if err != nil {
				l.Errorf("Event loop exiting: %s", err)
				return
			}
		}
	}
}
