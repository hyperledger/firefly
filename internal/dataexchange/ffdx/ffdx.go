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

package ffdx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/config/wsconfig"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/restclient"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/wsclient"
)

type FFDX struct {
	ctx          context.Context
	capabilities *dataexchange.Capabilities
	callbacks    dataexchange.Callbacks
	client       *resty.Client
	wsconn       wsclient.WSClient
}

type wsEvent struct {
	Type      msgType            `json:"type"`
	Sender    string             `json:"sender"`
	Recipient string             `json:"recipient"`
	RequestID string             `json:"requestId"`
	Path      string             `json:"path"`
	Message   string             `json:"message"`
	Hash      string             `json:"hash"`
	Size      int64              `json:"size"`
	Error     string             `json:"error"`
	Manifest  string             `json:"manifest"`
	Info      fftypes.JSONObject `json:"info"`
}

const (
	dxHTTPHeaderHash = "dx-hash"
	dxHTTPHeaderSize = "dx-size"
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
	Size       int64       `json:"size"`
	LastUpdate json.Number `json:"lastUpdate"`
}

type sendMessage struct {
	Message   string `json:"message"`
	Recipient string `json:"recipient"`
	RequestID string `json:"requestId"`
}

type transferBlob struct {
	Path      string `json:"path"`
	Recipient string `json:"recipient"`
	RequestID string `json:"requestId"`
}

type wsAck struct {
	Action   string `json:"action"`
	Manifest string `json:"manifest,omitempty"` // FireFly core determined that DX should propagate opaquely to TransferResult, if this DX supports delivery acknowledgements.
}

func (h *FFDX) Name() string {
	return "ffdx"
}

func (h *FFDX) Init(ctx context.Context, prefix config.Prefix, callbacks dataexchange.Callbacks) (err error) {
	h.ctx = log.WithLogField(ctx, "dx", "ffdx")
	h.callbacks = callbacks

	if prefix.GetString(restclient.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "url", "dataexchange.ffdx")
	}

	h.client = restclient.New(h.ctx, prefix)
	h.capabilities = &dataexchange.Capabilities{
		Manifest: prefix.GetBool(DataExchangeManifestEnabled),
	}

	wsConfig := wsconfig.GenerateConfigFromPrefix(prefix)

	h.wsconn, err = wsclient.New(ctx, wsConfig, nil, nil)
	if err != nil {
		return err
	}
	go h.eventLoop()
	return nil
}

func (h *FFDX) Start() error {
	return h.wsconn.Connect()
}

func (h *FFDX) Capabilities() *dataexchange.Capabilities {
	return h.capabilities
}

func (h *FFDX) GetEndpointInfo(ctx context.Context) (peerID string, endpoint fftypes.JSONObject, err error) {
	res, err := h.client.R().SetContext(ctx).
		SetResult(&endpoint).
		Get("/api/v1/id")
	if err != nil || !res.IsSuccess() {
		return peerID, endpoint, restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	return endpoint.GetString("id"), endpoint, nil
}

func (h *FFDX) AddPeer(ctx context.Context, peerID string, endpoint fftypes.JSONObject) (err error) {
	res, err := h.client.R().SetContext(ctx).
		SetBody(endpoint).
		Put(fmt.Sprintf("/api/v1/peers/%s", peerID))
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	return nil
}

func (h *FFDX) UploadBLOB(ctx context.Context, ns string, id fftypes.UUID, content io.Reader) (payloadRef string, hash *fftypes.Bytes32, size int64, err error) {
	payloadRef = fmt.Sprintf("%s/%s", ns, &id)
	var upload uploadBlob
	res, err := h.client.R().SetContext(ctx).
		SetFileReader("file", id.String(), content).
		SetResult(&upload).
		Put(fmt.Sprintf("/api/v1/blobs/%s", payloadRef))
	if err != nil || !res.IsSuccess() {
		err = restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
		return "", nil, -1, err
	}
	if hash, err = fftypes.ParseBytes32(ctx, upload.Hash); err != nil {
		return "", nil, -1, i18n.WrapError(ctx, err, i18n.MsgDXBadResponse, "hash", upload.Hash)
	}
	return payloadRef, hash, upload.Size, nil
}

func (h *FFDX) DownloadBLOB(ctx context.Context, payloadRef string) (content io.ReadCloser, err error) {
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

func (h *FFDX) SendMessage(ctx context.Context, opID *fftypes.UUID, peerID string, data []byte) (err error) {
	var responseData responseWithRequestID
	res, err := h.client.R().SetContext(ctx).
		SetBody(&sendMessage{
			Message:   string(data),
			Recipient: peerID,
			RequestID: opID.String(),
		}).
		SetResult(&responseData).
		Post("/api/v1/messages")
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	return nil
}

func (h *FFDX) TransferBLOB(ctx context.Context, opID *fftypes.UUID, peerID, payloadRef string) (err error) {
	var responseData responseWithRequestID
	res, err := h.client.R().SetContext(ctx).
		SetBody(&transferBlob{
			Path:      fmt.Sprintf("/%s", payloadRef),
			Recipient: peerID,
			RequestID: opID.String(),
		}).
		SetResult(&responseData).
		Post("/api/v1/transfers")
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	return nil
}

func (h *FFDX) CheckBLOBReceived(ctx context.Context, peerID, ns string, id fftypes.UUID) (hash *fftypes.Bytes32, size int64, err error) {
	var responseData responseWithRequestID
	res, err := h.client.R().SetContext(ctx).
		SetResult(&responseData).
		Head(fmt.Sprintf("/api/v1/blobs/%s/%s/%s", peerID, ns, id.String()))
	if err == nil && res.StatusCode() == http.StatusNotFound {
		return nil, -1, nil
	}
	if err != nil || !res.IsSuccess() {
		return nil, -1, restclient.WrapRestErr(ctx, res, err, i18n.MsgDXRESTErr)
	}
	hashString := res.Header().Get(dxHTTPHeaderHash)
	if hash, err = fftypes.ParseBytes32(ctx, hashString); err != nil {
		return nil, -1, i18n.WrapError(ctx, err, i18n.MsgDXBadResponse, "hash", hashString)
	}
	sizeString := res.Header().Get(dxHTTPHeaderSize)
	if sizeString != "" {
		if size, err = strconv.ParseInt(sizeString, 10, 64); err != nil {
			return nil, -1, i18n.WrapError(ctx, err, i18n.MsgDXBadResponse, "size", sizeString)
		}
	}
	return hash, size, nil
}

func (h *FFDX) eventLoop() {
	defer h.wsconn.Close()
	l := log.L(h.ctx).WithField("role", "event-loop")
	ctx := log.WithLogger(h.ctx, l)
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
			var manifest string
			switch msg.Type {
			case messageFailed:
				err = h.callbacks.TransferResult(msg.RequestID, fftypes.OpStatusFailed, fftypes.TransportStatusUpdate{Error: msg.Error})
			case messageDelivered:
				err = h.callbacks.TransferResult(msg.RequestID, fftypes.OpStatusSucceeded, fftypes.TransportStatusUpdate{
					Manifest: msg.Manifest,
					Info:     msg.Info,
				})
			case messageReceived:
				manifest, err = h.callbacks.MessageReceived(msg.Sender, []byte(msg.Message))
			case blobFailed:
				err = h.callbacks.TransferResult(msg.RequestID, fftypes.OpStatusFailed, fftypes.TransportStatusUpdate{Error: msg.Error})
			case blobDelivered:
				err = h.callbacks.TransferResult(msg.RequestID, fftypes.OpStatusSucceeded, fftypes.TransportStatusUpdate{})
			case blobReceived:
				var hash *fftypes.Bytes32
				hash, err = fftypes.ParseBytes32(ctx, msg.Hash)
				if err != nil {
					l.Errorf("Invalid hash received in DX event: '%s'", msg.Hash)
					err = nil // still confirm the message
				} else {
					err = h.callbacks.BLOBReceived(msg.Sender, *hash, msg.Size, msg.Path)
				}
			default:
				l.Errorf("Message unexpected: %s", msg.Type)
			}

			// Send the ack - as long as we didn't fail processing (which should only happen in core
			// if core itself is shutting down)
			if err == nil {
				ackBytes, _ := json.Marshal(&wsAck{
					Action:   "commit",
					Manifest: manifest,
				})
				err = h.wsconn.Send(ctx, ackBytes)
			}
			if err != nil {
				l.Errorf("Event loop exiting: %s", err)
				return
			}
		}
	}
}
