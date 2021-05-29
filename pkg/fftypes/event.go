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

package fftypes

// EventType indicates what the event means, as well as what the Reference in the event refers to
type EventType string

const (
	// EventTypeMessageConfirmed is the most important event type in the system. This means a message and all of its data
	// is available for processing by an application. Most applications only need to listen to this event type
	EventTypeMessageConfirmed EventType = "MessageConfirmed"
	// EventTypeMessageInvalid occurs if a message is received and confirmed from a sequencing perspective, but is invalid
	EventTypeMessageInvalid EventType = "MessageInvalid"
	// EventTypeDataArrivedBroadcast indicates that some data has arrived, over a broadcast transport
	EventTypeDataArrivedBroadcast EventType = "DataArrivedBroadcast"
	// EventTypeMessageSequencedBroadcast indicates that a deterministically sequenced message has arrived pinned to a blockchain
	EventTypeMessageSequencedBroadcast EventType = "MessageSequencedBroadcast"
	// EventTypeMessagesUnblocked is a special event to indidate a previously blocked context, has become unblocked
	EventTypeMessagesUnblocked EventType = "MessagesUnblocked"
)

// Event is an activity in the system, delivered reliably to applications, that indicates something has happened in the network
type Event struct {
	ID        *UUID     `json:"id"`
	Sequence  int64     `json:"sequence"`
	Type      EventType `json:"type"`
	Namespace string    `json:"namespace"`
	Reference *UUID     `json:"reference"`
	Created   *FFTime   `json:"created"`
}

// EventDelivery adds the referred object to an event, as well as details of the subscription that caused the event to
// be dispatched to an applciation.
type EventDelivery struct {
	Event
	Subscription SubscriptionRef `json:"subscription"`
	Message      *Message        `json:"message,omitempty"`
	Data         *DataRef        `json:"data,omitempty"`
}

// EventDeliveryResponse is the payload an application sends back, to confirm it has accepted (or rejected) the event and as such
// does not need to receive it again.
type EventDeliveryResponse struct {
	ID           *UUID           `json:"id"`
	Rejected     bool            `json:"rejected,omitempty"`
	Info         string          `json:"info,omitempty"`
	Subscription SubscriptionRef `json:"subscription"`
}

func NewEvent(t EventType, ns string, ref *UUID) *Event {
	return &Event{
		ID:        NewUUID(),
		Type:      t,
		Namespace: ns,
		Reference: ref,
		Created:   Now(),
	}
}
