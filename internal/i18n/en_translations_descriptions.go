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

package i18n

//revive:disable

/*
This file contains the English field level descriptions that are used in
OpenAPI Spec generation. Each struct field that wants to use one of these
needs to have an ffstruct tag on it, indicating the name of the struct.
That will be combined with the JSON field name (note, it is not the GO
field name, but the JSON serialized name), separated by a "." This is the
key used to lookup the translation below. If it is not found, the description
is left blank in the OpenAPI spec
*/
var (
	// MessageHeader field descriptions
	MessageHeaderID        = ffm("MessageHeader.id", "The UUID of the message")
	MessageHeaderCID       = ffm("MessageHeader.cid", "The CID of the message")
	MessageHeaderType      = ffm("MessageHeader.type", "The type of the message")
	MessageHeaderTxType    = ffm("MessageHeader.txtype", "The type of transaction")
	MessageHeaderCreated   = ffm("MessageHeader.created", "The creation time of the message")
	MessageHeaderNamespace = ffm("MessageHeader.namespace", "The namespace of the message")
	MessageHeaderGroup     = ffm("MessageHeader.group", "The group of recipients for a private message")
	MessageHeaderTopics    = ffm("MessageHeader.topics", "The message topics. Messages will be ordered within a given topic or combination of topics")
	MessageHeaderTag       = ffm("MessageHeader.tag", "The message tag. Useful for indicating the purpose of the message to other nodes.")
	MessageHeaderDataHash  = ffm("MessageHeader.datahash", "The hash of all of the data items attached to this message.")

	// Message field descriptions
	MessageHeader    = ffm("Message.header", "The message header")
	MessageHash      = ffm("Message.hash", "The hash of the entire message, including data elements and headers")
	MessageBatchID   = ffm("Messages.batch", "The UUID of the batch in which the message was processed")
	MessageState     = ffm("Message.state", "The current state of the message")
	MessageConfirmed = ffm("Message.confirmed", "The timestamp of when the message was confirmed")
	MessageData      = ffm("Message.data", "The list of data elements attached to the message")
	MessagePins      = ffm("Message.pins", "The list of pins from this message")
)
