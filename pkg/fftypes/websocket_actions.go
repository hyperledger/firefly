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

package fftypes

// WSClientPayloadType actions go from client->server
type WSClientPayloadType = FFEnum

var (
	// WSClientActionStart is a request to the server to start delivering messages to the client
	WSClientActionStart = ffEnum("wstype", "start")
	// WSClientActionAck acknowledges an event that was delivered, allowing further messages to be sent
	WSClientActionAck = ffEnum("wstype", "ack")

	// WSProtocolErrorEventType is a special event "type" field for server to send the client, if it performs a ProtocolError
	WSProtocolErrorEventType = ffEnum("wstype", "protocol_error")

	// WSClientActionChangeNotifcation a special event type that is a local database change event, and never requires an ack
	WSClientActionChangeNotifcation = ffEnum("wstype", "change_notification")
)

// WSClientActionBase is the base fields of all client actions sent on the websocket
type WSClientActionBase struct {
	Type WSClientPayloadType `json:"type,omitempty"`
}

// WSClientActionStartPayload starts a subscription on this socket - either an existing one, or creating an ephemeral one
type WSClientActionStartPayload struct {
	WSClientActionBase

	AutoAck      *bool               `json:"autoack"`
	Namespace    string              `json:"namespace"`
	Name         string              `json:"name"`
	Ephemeral    bool                `json:"ephemeral"`
	Filter       SubscriptionFilter  `json:"filter"`
	Options      SubscriptionOptions `json:"options"`
	ChangeEvents string              `json:"changeEvents,omitempty"`
}

// WSClientActionAckPayload acknowldges a received event (not applicable in AutoAck mode)
type WSClientActionAckPayload struct {
	WSClientActionBase

	ID           *UUID            `json:"id,omitempty"`
	Subscription *SubscriptionRef `json:"subscription,omitempty"`
}

// WSProtocolErrorPayload is sent to the client by the server in the case of a protocol error
type WSProtocolErrorPayload struct {
	Type  WSClientPayloadType `json:"type" ffenum:"wstype"`
	Error string              `json:"error"`
}

// WSChangeNotification is a special notification type for a change event, that does *not* require an ack
type WSChangeNotification struct {
	WSClientActionBase

	ChangeEvent *ChangeEvent `json:"change"`
}
