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

// EventType indicates what the event means, as well as what the Reference in the event refers to
type EventType = FFEnum

var (
	// EventTypeTransactionSubmitted occurs only on the node that initiates a tranaction, when the transaction is submitted
	EventTypeTransactionSubmitted EventType = ffEnum("eventtype", "transaction_submitted")
	// EventTypeMessageConfirmed is the most important event type in the system. This means a message and all of its data
	// is available for processing by an application. Most applications only need to listen to this event type
	EventTypeMessageConfirmed EventType = ffEnum("eventtype", "message_confirmed")
	// EventTypeMessageRejected occurs if a message is received and confirmed from a sequencing perspective, but is rejected as invalid (mismatch to schema, or duplicate system broadcast)
	EventTypeMessageRejected EventType = ffEnum("eventtype", "message_rejected")
	// EventTypeNamespaceConfirmed occurs when a new namespace is ready for use (on the namespace itself)
	EventTypeNamespaceConfirmed EventType = ffEnum("eventtype", "namespace_confirmed")
	// EventTypeDatatypeConfirmed occurs when a new datatype is ready for use (on the namespace of the datatype)
	EventTypeDatatypeConfirmed EventType = ffEnum("eventtype", "datatype_confirmed")
	// EventTypeIdentityConfirmed occurs when a new identity has been confirmed, as as result of a signed claim broadcast, and any associated claim verification
	EventTypeIdentityConfirmed EventType = ffEnum("eventtype", "identity_confirmed")
	// EventTypeIdentityUpdated occurs when an existing identity is update by the owner of that identity
	EventTypeIdentityUpdated EventType = ffEnum("eventtype", "identity_updated")
	// EventTypePoolConfirmed occurs when a new token pool is ready for use
	EventTypePoolConfirmed EventType = ffEnum("eventtype", "token_pool_confirmed")
	// EventTypeTransferConfirmed occurs when a token transfer has been confirmed
	EventTypeTransferConfirmed EventType = ffEnum("eventtype", "token_transfer_confirmed")
	// EventTypeTransferOpFailed occurs when a token transfer submitted by this node has failed (based on feedback from connector)
	EventTypeTransferOpFailed EventType = ffEnum("eventtype", "token_transfer_op_failed")
	// EventTypeApprovalConfirmed occurs when a token approval has been confirmed
	EventTypeApprovalConfirmed EventType = ffEnum("eventtype", "token_approval_confirmed")
	// EventTypeApprovalOpFailed occurs when a token approval submitted by this node has failed (based on feedback from connector)
	EventTypeApprovalOpFailed EventType = ffEnum("eventtype", "token_approval_op_failed")
	// EventTypeContractInterfaceConfirmed occurs when a new contract interface has been confirmed
	EventTypeContractInterfaceConfirmed EventType = ffEnum("eventtype", "contract_interface_confirmed")
	// EventTypeContractAPIConfirmed occurs when a new contract API has been confirmed
	EventTypeContractAPIConfirmed EventType = ffEnum("eventtype", "contract_api_confirmed")
	// EventTypeBlockchainEvent occurs when a new event has been recorded from the blockchain
	EventTypeBlockchainEvent EventType = ffEnum("eventtype", "blockchain_event")
)

// Event is an activity in the system, delivered reliably to applications, that indicates something has happened in the network
type Event struct {
	ID          *UUID     `json:"id"`
	Sequence    int64     `json:"sequence"`
	Type        EventType `json:"type" ffenum:"eventtype"`
	Namespace   string    `json:"namespace"`
	Reference   *UUID     `json:"reference"`
	Correlator  *UUID     `json:"correlator,omitempty"`
	Transaction *UUID     `json:"tx,omitempty"`
	Created     *FFTime   `json:"created"`
}

// EventDelivery adds the referred object to an event, as well as details of the subscription that caused the event to
// be dispatched to an applciation.
type EventDelivery struct {
	Event
	Subscription SubscriptionRef `json:"subscription"`
	Message      *Message        `json:"message,omitempty"`
}

// EventDeliveryResponse is the payload an application sends back, to confirm it has accepted (or rejected) the event and as such
// does not need to receive it again.
type EventDeliveryResponse struct {
	ID           *UUID           `json:"id"`
	Rejected     bool            `json:"rejected,omitempty"`
	Info         string          `json:"info,omitempty"`
	Subscription SubscriptionRef `json:"subscription"`
	Reply        *MessageInOut   `json:"reply,omitempty"`
}

func NewEvent(t EventType, ns string, ref *UUID, tx *UUID) *Event {
	return &Event{
		ID:          NewUUID(),
		Type:        t,
		Namespace:   ns,
		Reference:   ref,
		Transaction: tx,
		Created:     Now(),
	}
}

func (e *Event) LocalSequence() int64 {
	return e.Sequence
}
