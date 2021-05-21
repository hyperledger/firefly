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

type EventType string

const (
	EventTypeDataArrivedBroadcast      EventType = "DataArrivedBroadcast"
	EventTypeMessageSequencedBroadcast EventType = "DessageSequencedBroadcast"
	EventTypeMessageConfirmed          EventType = "MessageConfirmed"
	EventTypeMessagesUnblocked         EventType = "MessagesUnblocked"
)

type Event struct {
	ID        *UUID     `json:"id"`
	Type      EventType `json:"type"`
	Namespace string    `json:"namespace"`
	Reference *UUID     `json:"reference"`
	Sequence  int64     `json:"sequence"`
	Created   *FFTime   `json:"created"`
}

type EventDelivery struct {
	Event
	Subscription SubscriptionRef `json:"subscription"`
	Message      *Message        `json:"message,omitempty"`
	Data         *DataRef        `json:"data,omitempty"`
}

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
