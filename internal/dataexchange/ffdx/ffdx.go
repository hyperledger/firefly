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
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/wsclient"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/dataexchange"
)

type FFDX struct {
	ctx          context.Context
	capabilities *dataexchange.Capabilities
	callbacks    callbacks
	client       *resty.Client
	wsconn       wsclient.WSClient
	needsInit    bool
	initialized  bool
	initMutex    sync.Mutex
	nodes        []fftypes.JSONObject
	ackChannel   chan *ack
}

type callbacks struct {
	listeners []dataexchange.Callbacks
}

func (cb *callbacks) DXEvent(event dataexchange.DXEvent) {
	for _, cb := range cb.listeners {
		cb.DXEvent(event)
	}
}

const (
	dxHTTPHeaderHash = "dx-hash"
	dxHTTPHeaderSize = "dx-size"
)

type msgType string

const (
	messageReceived     msgType = "message-received"
	messageDelivered    msgType = "message-delivered"
	messageAcknowledged msgType = "message-acknowledged"
	messageFailed       msgType = "message-failed"
	blobReceived        msgType = "blob-received"
	blobDelivered       msgType = "blob-delivered"
	blobAcknowledged    msgType = "blob-acknowledged"
	blobFailed          msgType = "blob-failed"
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
	ID       string `json:"id"`
	Manifest string `json:"manifest,omitempty"` // FireFly core determined that DX should propagate opaquely to TransferResult, if this DX supports delivery acknowledgements.
}

type dxStatus struct {
	Status string `json:"status"`
}

type ack struct {
	eventID  string
	manifest string
}

func (h *FFDX) Name() string {
	return "ffdx"
}

func (h *FFDX) Init(ctx context.Context, config config.Section) (err error) {
	h.ctx = log.WithLogField(ctx, "dx", "https")
	h.ackChannel = make(chan *ack)

	h.needsInit = config.GetBool(DataExchangeInitEnabled)

	if config.GetString(ffresty.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, coremsgs.MsgMissingPluginConfig, "url", "dataexchange.ffdx")
	}

	h.client = ffresty.New(h.ctx, config)
	h.capabilities = &dataexchange.Capabilities{
		Manifest: config.GetBool(DataExchangeManifestEnabled),
	}

	wsConfig := wsclient.GenerateConfig(config)

	h.wsconn, err = wsclient.New(ctx, wsConfig, h.beforeConnect, nil)
	if err != nil {
		return err
	}
	go h.eventLoop()
	go h.ackLoop()
	return nil
}

func (h *FFDX) SetNodes(nodes []fftypes.JSONObject) {
	h.nodes = nodes
}

func (h *FFDX) RegisterListener(listener dataexchange.Callbacks) {
	h.callbacks.listeners = append(h.callbacks.listeners, listener)
}

func (h *FFDX) Start() error {
	return h.wsconn.Connect()
}

func (h *FFDX) Capabilities() *dataexchange.Capabilities {
	return h.capabilities
}

func (h *FFDX) beforeConnect(ctx context.Context) error {
	h.initMutex.Lock()
	defer h.initMutex.Unlock()

	if h.needsInit {
		h.initialized = false
		var status dxStatus
		res, err := h.client.R().SetContext(ctx).
			SetBody(h.nodes).
			SetResult(&status).
			Post("/api/v1/init")
		if err != nil || !res.IsSuccess() {
			return ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgDXRESTErr)
		}
		if status.Status != "ready" {
			return fmt.Errorf("DX returned non-ready status: %s", status.Status)
		}
	}
	h.initialized = true
	return nil
}

func (h *FFDX) checkInitialized(ctx context.Context) error {
	h.initMutex.Lock()
	defer h.initMutex.Unlock()

	if !h.initialized {
		return i18n.NewError(ctx, coremsgs.MsgDXNotInitialized)
	}
	return nil
}

func (h *FFDX) GetEndpointInfo(ctx context.Context) (peer fftypes.JSONObject, err error) {
	if err := h.checkInitialized(ctx); err != nil {
		return peer, err
	}

	res, err := h.client.R().SetContext(ctx).
		SetResult(&peer).
		Get("/api/v1/id")
	if err != nil || !res.IsSuccess() {
		return peer, ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgDXRESTErr)
	}
	id := peer.GetString("id")
	if id == "" {
		log.L(ctx).Errorf("Invalid DX info: %s", peer.String())
		return nil, i18n.NewError(ctx, coremsgs.MsgDXInfoMissingID)
	}
	h.nodes = append(h.nodes, peer)
	return peer, nil
}

func (h *FFDX) AddPeer(ctx context.Context, peer fftypes.JSONObject) (err error) {
	if err := h.checkInitialized(ctx); err != nil {
		return err
	}

	res, err := h.client.R().SetContext(ctx).
		SetBody(peer).
		Put(fmt.Sprintf("/api/v1/peers/%s", peer.GetString("id")))
	if err != nil || !res.IsSuccess() {
		return ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgDXRESTErr)
	}
	return nil
}

func (h *FFDX) UploadBlob(ctx context.Context, ns string, id fftypes.UUID, content io.Reader) (payloadRef string, hash *fftypes.Bytes32, size int64, err error) {
	payloadRef = fmt.Sprintf("%s/%s", ns, &id)
	var upload uploadBlob
	res, err := h.client.R().SetContext(ctx).
		SetFileReader("file", id.String(), content).
		SetResult(&upload).
		Put(fmt.Sprintf("/api/v1/blobs/%s", payloadRef))
	if err != nil || !res.IsSuccess() {
		err = ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgDXRESTErr)
		return "", nil, -1, err
	}
	if hash, err = fftypes.ParseBytes32(ctx, upload.Hash); err != nil {
		return "", nil, -1, i18n.WrapError(ctx, err, coremsgs.MsgDXBadResponse, "hash", upload.Hash)
	}
	return payloadRef, hash, upload.Size, nil
}

func (h *FFDX) DownloadBlob(ctx context.Context, payloadRef string) (content io.ReadCloser, err error) {
	res, err := h.client.R().SetContext(ctx).
		SetDoNotParseResponse(true).
		Get(fmt.Sprintf("/api/v1/blobs/%s", payloadRef))
	if err != nil || !res.IsSuccess() {
		if err == nil {
			_ = res.RawBody().Close()
		}
		return nil, ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgDXRESTErr)
	}
	return res.RawBody(), nil
}

func (h *FFDX) SendMessage(ctx context.Context, nsOpID, peerID string, data []byte) (err error) {
	if err := h.checkInitialized(ctx); err != nil {
		return err
	}

	var responseData responseWithRequestID
	res, err := h.client.R().SetContext(ctx).
		SetBody(&sendMessage{
			Message:   string(data),
			Recipient: peerID,
			RequestID: nsOpID,
		}).
		SetResult(&responseData).
		Post("/api/v1/messages")
	if err != nil || !res.IsSuccess() {
		return ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgDXRESTErr)
	}
	return nil
}

func (h *FFDX) TransferBlob(ctx context.Context, nsOpID, peerID, payloadRef string) (err error) {
	if err := h.checkInitialized(ctx); err != nil {
		return err
	}

	var responseData responseWithRequestID
	res, err := h.client.R().SetContext(ctx).
		SetBody(&transferBlob{
			Path:      fmt.Sprintf("/%s", payloadRef),
			Recipient: peerID,
			RequestID: nsOpID,
		}).
		SetResult(&responseData).
		Post("/api/v1/transfers")
	if err != nil || !res.IsSuccess() {
		return ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgDXRESTErr)
	}
	return nil
}

func (h *FFDX) CheckBlobReceived(ctx context.Context, peerID, ns string, id fftypes.UUID) (hash *fftypes.Bytes32, size int64, err error) {
	var responseData responseWithRequestID
	res, err := h.client.R().SetContext(ctx).
		SetResult(&responseData).
		Head(fmt.Sprintf("/api/v1/blobs/%s/%s/%s", peerID, ns, id.String()))
	if err == nil && res.StatusCode() == http.StatusNotFound {
		return nil, -1, nil
	}
	if err != nil || !res.IsSuccess() {
		return nil, -1, ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgDXRESTErr)
	}
	hashString := res.Header().Get(dxHTTPHeaderHash)
	if hash, err = fftypes.ParseBytes32(ctx, hashString); err != nil {
		return nil, -1, i18n.WrapError(ctx, err, coremsgs.MsgDXBadResponse, "hash", hashString)
	}
	sizeString := res.Header().Get(dxHTTPHeaderSize)
	if sizeString != "" {
		if size, err = strconv.ParseInt(sizeString, 10, 64); err != nil {
			return nil, -1, i18n.WrapError(ctx, err, coremsgs.MsgDXBadResponse, "size", sizeString)
		}
	}
	return hash, size, nil
}

func (h *FFDX) ackLoop() {
	for {
		select {
		case <-h.ctx.Done():
			log.L(h.ctx).Debugf("Ack loop exiting")
			return
		case ack := <-h.ackChannel:
			// Send the ack
			ackBytes, _ := json.Marshal(&wsAck{
				Action:   "ack",
				ID:       ack.eventID,
				Manifest: ack.manifest,
			})
			err := h.wsconn.Send(h.ctx, ackBytes)
			if err != nil {
				// Note we only get the error in the case we're closing down, so no need to retry
				log.L(h.ctx).Warnf("Ack loop send failed: %s", err)
			}
		}
	}
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
			h.dispatchEvent(&msg)
		}
	}
}
