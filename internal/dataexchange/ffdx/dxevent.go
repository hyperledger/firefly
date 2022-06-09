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
	"strings"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/dataexchange"
)

type wsEvent struct {
	Type      msgType            `json:"type"`
	EventID   string             `json:"id"`
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

type dxEvent struct {
	ffdx                *FFDX
	id                  string
	dxType              dataexchange.DXEventType
	messageReceived     *dataexchange.MessageReceived
	privateBlobReceived *dataexchange.PrivateBlobReceived
	transferResult      *dataexchange.TransferResult
}

func (e *dxEvent) NamespacedID() string {
	return e.id
}

func (e *dxEvent) Type() dataexchange.DXEventType {
	return e.dxType
}

func (e *dxEvent) AckWithManifest(manifest string) {
	select {
	case e.ffdx.ackChannel <- &ack{
		eventID:  e.id,
		manifest: manifest,
	}:
	case <-e.ffdx.ctx.Done():
		log.L(e.ffdx.ctx).Debugf("Ack received after close: %s", e.id)
	}
}

func (e *dxEvent) Ack() {
	e.AckWithManifest("")
}

func (e *dxEvent) MessageReceived() *dataexchange.MessageReceived {
	return e.messageReceived
}

func (e *dxEvent) PrivateBlobReceived() *dataexchange.PrivateBlobReceived {
	return e.privateBlobReceived
}

func (e *dxEvent) TransferResult() *dataexchange.TransferResult {
	return e.transferResult
}

func (h *FFDX) dispatchEvent(msg *wsEvent) {
	var err error
	e := &dxEvent{ffdx: h, id: msg.EventID}
	switch msg.Type {
	case messageFailed:
		e.dxType = dataexchange.DXEventTypeTransferResult
		e.transferResult = &dataexchange.TransferResult{
			TrackingID: msg.RequestID,
			Status:     core.OpStatusFailed,
			TransportStatusUpdate: core.TransportStatusUpdate{
				Error: msg.Error,
				Info:  msg.Info,
			},
		}
	case messageDelivered:
		status := core.OpStatusSucceeded
		if h.capabilities.Manifest {
			status = core.OpStatusPending
		}
		e.dxType = dataexchange.DXEventTypeTransferResult
		e.transferResult = &dataexchange.TransferResult{
			TrackingID: msg.RequestID,
			Status:     status,
			TransportStatusUpdate: core.TransportStatusUpdate{
				Info: msg.Info,
			},
		}
	case messageAcknowledged:
		e.dxType = dataexchange.DXEventTypeTransferResult
		e.transferResult = &dataexchange.TransferResult{
			TrackingID: msg.RequestID,
			Status:     core.OpStatusSucceeded,
			TransportStatusUpdate: core.TransportStatusUpdate{
				Manifest: msg.Manifest,
				Info:     msg.Info,
			},
		}
	case messageReceived:
		e.dxType = dataexchange.DXEventTypeMessageReceived
		e.messageReceived = &dataexchange.MessageReceived{
			PeerID: msg.Sender,
			Data:   []byte(msg.Message),
		}
	case blobFailed:
		e.dxType = dataexchange.DXEventTypeTransferResult
		e.transferResult = &dataexchange.TransferResult{
			TrackingID: msg.RequestID,
			Status:     core.OpStatusFailed,
			TransportStatusUpdate: core.TransportStatusUpdate{
				Error: msg.Error,
				Info:  msg.Info,
			},
		}
	case blobDelivered:
		status := core.OpStatusSucceeded
		if h.capabilities.Manifest {
			status = core.OpStatusPending
		}
		e.dxType = dataexchange.DXEventTypeTransferResult
		e.transferResult = &dataexchange.TransferResult{
			TrackingID: msg.RequestID,
			Status:     status,
			TransportStatusUpdate: core.TransportStatusUpdate{
				Info: msg.Info,
			},
		}
	case blobReceived:
		var hash *fftypes.Bytes32
		hash, err = fftypes.ParseBytes32(h.ctx, msg.Hash)
		if err == nil {
			pathParts := strings.Split(msg.Path, "/")
			e.dxType = dataexchange.DXEventTypePrivateBlobReceived
			e.privateBlobReceived = &dataexchange.PrivateBlobReceived{
				Namespace:  pathParts[0],
				PeerID:     msg.Sender,
				Hash:       *hash,
				Size:       msg.Size,
				PayloadRef: msg.Path,
			}
		}
	case blobAcknowledged:
		e.dxType = dataexchange.DXEventTypeTransferResult
		e.transferResult = &dataexchange.TransferResult{
			TrackingID: msg.RequestID,
			Status:     core.OpStatusSucceeded,
			TransportStatusUpdate: core.TransportStatusUpdate{
				Hash: msg.Hash,
				Info: msg.Info,
			},
		}
	default:
		err = i18n.NewError(h.ctx, coremsgs.MsgUnexpectedDXMessageType, msg.Type)
	}
	// If we couldn't dispatch the event we received, we still ack it
	if err != nil {
		log.L(h.ctx).Warnf("Failed to dispatch DX event: %s", err)
		e.Ack()
	} else {
		h.callbacks.DXEvent(e)
	}
}
