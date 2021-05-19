// Copyright © 2021 Kaleido, Inc.
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

import "github.com/google/uuid"

// WSClientActionType actions go from client->server
type WSClientPayloadType string

const (
	// Request the server to start delivering messages to the client
	WSClientActionStart WSClientPayloadType = "start"
	// Acknowledge an event that was delivered, allowing further messages to be sent
	WSClientActionAck WSClientPayloadType = "ack"

	// Special event "type" field for server to send the client, if it performs a ProtocolError
	WSProtocolErrorEventType WSClientPayloadType = "ProtocolError"
)

type WSClientActionBase struct {
	Type WSClientPayloadType `json:"type,omitempty"`
}

type WSClientActionStartPayload struct {
	WSClientActionBase

	AutoAck   *bool               `json:"autoack"`
	Namespace string              `json:"namespace"`
	Name      string              `json:"name"`
	Ephemeral bool                `json:"ephemeral"`
	Filter    SubscriptionFilter  `json:"filter"`
	Options   SubscriptionOptions `json:"options"`
}

type WSClientActionAckPayload struct {
	WSClientActionBase

	ID           *uuid.UUID       `json:"id,omitempty"`
	Subscription *SubscriptionRef `json:"subscription,omitempty"`
}

type WSProtocolErrorPayload struct {
	Type  WSClientPayloadType `json:"type"`
	Error string              `json:"error"`
}
