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
	"strings"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/wsclient"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/dataexchange"
)

const DXIDSeparator = "/"

type FFDX struct {
	ctx          context.Context
	cancelCtx    context.CancelFunc
	capabilities *dataexchange.Capabilities
	callbacks    callbacks
	client       *resty.Client
	wsconn       wsclient.WSClient
	needsInit    bool
	initialized  bool
	initMutex    sync.Mutex
	nodes        map[string]*dxNode
	ackChannel   chan *ack
}

type dxNode struct {
	Name string
	Peer fftypes.JSONObject
}

type callbacks struct {
	plugin     *FFDX
	handlers   map[string]dataexchange.Callbacks
	opHandlers map[string]core.OperationCallbacks
}

func (cb *callbacks) OperationUpdate(ctx context.Context, update *core.OperationUpdate) {
	namespace, _, _ := core.ParseNamespacedOpID(ctx, update.NamespacedOpID)
	if handler, ok := cb.opHandlers[namespace]; ok {
		handler.OperationUpdate(update)
	} else {
		log.L(ctx).Errorf("No handler found for DX operation '%s'", update.NamespacedOpID)
		update.OnComplete()
	}
}

func (cb *callbacks) DXEvent(ctx context.Context, namespace, recipient string, event dataexchange.DXEvent) {
	node := cb.plugin.findNode(namespace, recipient)
	if node != nil {
		key := namespace + ":" + node.Name
		if handler, ok := cb.handlers[key]; ok {
			handler.DXEvent(cb.plugin, event)
		} else {
			log.L(ctx).Errorf("No handler found for DX event '%s' namespace=%s node=%s", event.EventID(), namespace, node.Name)
			event.Ack()
		}
	} else {
		log.L(ctx).Errorf("Unknown local node for DX event '%s' recipient=%s", event.EventID(), recipient)
		event.Ack()
	}
}

func splitLast(s string, sep string) (string, string) {
	split := strings.LastIndex(s, sep)
	if split == -1 {
		return "", s
	}
	return s[:split], s[split+1:]
}

func splitBlobPath(path string) (prefix, namespace, id string) {
	path, id = splitLast(path, "/")
	path, namespace = splitLast(path, "/")
	return path, namespace, id
}

func joinBlobPath(namespace, id string) string {
	return fmt.Sprintf("%s/%s", namespace, id)
}

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
	Sender    string `json:"sender"`
}

type transferBlob struct {
	Path      string `json:"path"`
	Recipient string `json:"recipient"`
	RequestID string `json:"requestId"`
	Sender    string `json:"sender"`
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

func (h *FFDX) Init(ctx context.Context, cancelCtx context.CancelFunc, config config.Section) (err error) {
	h.ctx = log.WithLogField(ctx, "dx", "https")
	h.cancelCtx = cancelCtx
	h.ackChannel = make(chan *ack)
	h.callbacks = callbacks{
		plugin:     h,
		handlers:   make(map[string]dataexchange.Callbacks),
		opHandlers: make(map[string]core.OperationCallbacks),
	}
	h.needsInit = config.GetBool(DataExchangeInitEnabled)
	h.nodes = make(map[string]*dxNode)

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

func (h *FFDX) SetHandler(networkNamespace, nodeName string, handler dataexchange.Callbacks) {
	key := networkNamespace + ":" + nodeName
	h.callbacks.handlers[key] = handler
}

func (h *FFDX) SetOperationHandler(namespace string, handler core.OperationCallbacks) {
	h.callbacks.opHandlers[namespace] = handler
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
		body := make([]fftypes.JSONObject, 0)
		for _, node := range h.nodes {
			body = append(body, node.Peer)
		}
		res, err := h.client.R().SetContext(ctx).
			SetBody(body).
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

func (h *FFDX) GetPeerID(peer fftypes.JSONObject) string {
	return peer.GetString("id")
}

func (h *FFDX) GetEndpointInfo(ctx context.Context, nodeName string) (peer fftypes.JSONObject, err error) {
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
	peer["id"] = fmt.Sprintf("%s%s%s", id, DXIDSeparator, nodeName)
	return peer, nil
}

func (h *FFDX) AddNode(ctx context.Context, networkNamespace, nodeName string, peer fftypes.JSONObject) (err error) {
	h.initMutex.Lock()
	defer h.initMutex.Unlock()

	key := networkNamespace + ":" + h.GetPeerID(peer)
	h.nodes[key] = &dxNode{
		Peer: peer,
		Name: nodeName,
	}

	if h.initialized {
		res, err := h.client.R().SetContext(ctx).
			SetBody(peer).
			Put(fmt.Sprintf("/api/v1/peers/%s", peer.GetString("id")))
		if err != nil || !res.IsSuccess() {
			return ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgDXRESTErr)
		}
	}

	return nil
}

func (h *FFDX) findNode(namespace, recipient string) *dxNode {
	h.initMutex.Lock()
	defer h.initMutex.Unlock()
	node := h.nodes[namespace+":"+recipient]
	if node == nil {
		// Fall back to nodes registered on the legacy system namespace
		// (further verification of the off-chain identity will be performed by the event handler)
		node = h.nodes[core.LegacySystemNamespace+":"+recipient]
	}
	return node
}

func (h *FFDX) UploadBlob(ctx context.Context, ns string, id fftypes.UUID, content io.Reader) (payloadRef string, hash *fftypes.Bytes32, size int64, err error) {
	payloadRef = joinBlobPath(ns, id.String())
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

func (h *FFDX) SendMessage(ctx context.Context, nsOpID string, peer, sender fftypes.JSONObject, data []byte) (err error) {
	if err := h.checkInitialized(ctx); err != nil {
		return err
	}

	var responseData responseWithRequestID
	res, err := h.client.R().SetContext(ctx).
		SetBody(&sendMessage{
			Message:   string(data),
			Recipient: h.GetPeerID(peer),
			RequestID: nsOpID,
			Sender:    h.GetPeerID(sender),
		}).
		SetResult(&responseData).
		Post("/api/v1/messages")
	if err != nil || !res.IsSuccess() {
		return ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgDXRESTErr)
	}
	return nil
}

func (h *FFDX) TransferBlob(ctx context.Context, nsOpID string, peer, sender fftypes.JSONObject, payloadRef string) (err error) {
	if err := h.checkInitialized(ctx); err != nil {
		return err
	}

	var responseData responseWithRequestID
	res, err := h.client.R().SetContext(ctx).
		SetBody(&transferBlob{
			Path:      fmt.Sprintf("/%s", payloadRef),
			Recipient: h.GetPeerID(peer),
			RequestID: nsOpID,
			Sender:    h.GetPeerID(sender),
		}).
		SetResult(&responseData).
		Post("/api/v1/transfers")
	if err != nil || !res.IsSuccess() {
		return ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgDXRESTErr)
	}
	return nil
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
				l.Debugf("Event loop exiting (receive channel closed). Terminating server!")
				h.cancelCtx()
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
