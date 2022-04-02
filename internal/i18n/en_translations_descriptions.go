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
	MessageHeaderID        = ffm("MessageHeader.id", "The UUID of the message. Unique to each message")
	MessageHeaderCID       = ffm("MessageHeader.cid", "The correlation ID of the message. Set this when a message is a response to another message")
	MessageHeaderType      = ffm("MessageHeader.type", "The type of the message")
	MessageHeaderTxType    = ffm("MessageHeader.txtype", "The type of transaction used to order/deliver this message")
	MessageHeaderCreated   = ffm("MessageHeader.created", "The creation time of the message")
	MessageHeaderNamespace = ffm("MessageHeader.namespace", "The namespace of the message")
	MessageHeaderGroup     = ffm("MessageHeader.group", "Private messages only - the identifier hash of the privacy group. Derived from the name and member list of the group")
	MessageHeaderTopics    = ffm("MessageHeader.topics", "A message topic associates this message with an ordered stream of data. A custom topic should be assigned - using the default topic is discouraged")
	MessageHeaderTag       = ffm("MessageHeader.tag", "The message tag indicates the purpose of the message to the applications that process it")
	MessageHeaderDataHash  = ffm("MessageHeader.datahash", "A single hash representing all data in the message. Derived from the array of data ids+hashes attached to this message")

	// Message field descriptions
	MessageHeader    = ffm("Message.header", "The message header contains all fields that are used to build the message hash")
	MessageHash      = ffm("Message.hash", "The hash of the message. Derived from the header, which includes the data hash")
	MessageBatchID   = ffm("Messages.batch", "The UUID of the batch in which the message was pinned/transferred")
	MessageState     = ffm("Message.state", "The current state of the message")
	MessageConfirmed = ffm("Message.confirmed", "The timestamp of when the message was confirmed/rejected")
	MessageData      = ffm("Message.data", "The list of data elements attached to the message")
	MessagePins      = ffm("Message.pins", "For private messages, a unique pin hash:nonce is assigned for each topic")

	// MessageInOut field descriptions
	MessageInOutData  = ffm("MessageInOut.data", "For input allows you to specify data in-line in the message, that will be turned into data attachments. For output when fetchdata is used on API calls, includes the in-line data payloads of all data attachments")
	MessageInOutGroup = ffm("MessageInOut.group", "Allows you to specify details of the private group of recipients in-line in the message. Alternative to using the header.group to specify the hash of a group that has been previously resolved")

	// InputGroup field descriptions
	InputGroupName    = ffm("InputGroup.name", "Optional name for the group. Allows you to have multiple separate groups with the same list of participants")
	InputGroupMembers = ffm("InputGroup.members", "An array of members of the group. If no identities local to the sending node are included, then the organization owner of the local node is added automatically")

	// DataRefOrValue field descriptions
	DataRefOrValueValidator = ffm("DataRefOrValue.validator", "The data validator type to use for in-line data")
	DataRefOrValueDatatype  = ffm("DataRefOrValue.datatype", "The optional datatype to use for validation of the in-line data")
	DataRefOrValueValue     = ffm("DataRefOrValue.value", "The in-line value for the data. Can be any JSON type - object, array, string, number or boolean")
	DataRefOrValueBlob      = ffm("DataRefOrValue.blob", "An optional in-line hash reference to a previously uploaded binary data blob")

	// MessageRef field descriptions
	MessageRefID   = ffm("MessageRef.id", "The UUID of the referenced message")
	MessageRefHash = ffm("MessageRef.hash", "The hash of the referenced message")

	// Group field descriptions
	GroupNamespace = ffm("Group.namespace", "The namespace of the group")
	GroupName      = ffm("Group.name", "The optional name of the group, allowing multiple unique groups to exist with the same list of recipients")
	GroupMembers   = ffm("Group.members", "The list of members in this privacy group")
	GroupMessage   = ffm("Group.message", "The message used to broadcast this group privately to the members")
	GroupHash      = ffm("Group.hash", "The identifier hash of this group. Derived from the name and group members")
	GroupCreated   = ffm("Group.created", "The time when the group was first used to send a message in the network")

	// DataRef field descriptions
	DataRefID   = ffm("DataRef.id", "The UUID of the referenced data resource")
	DataRefHash = ffm("DataRef.hash", "The hash of the referenced data")

	// BlobRef field descriptions
	BlobRefHash   = ffm("BlobRef.hash", "The hash of the binary blob data")
	BlobRefSize   = ffm("BlobRef.size", "The size of the binary data")
	BlobRefName   = ffm("BlobRef.name", "The name field from the metadata attached to the blob, commonly used as a path/filename, and indexed for search")
	BlobRefPublic = ffm("BlobRef.public", "If this data has been published to shared storage, this field is the id of the data in the shared storage plugin (IPFS hash etc.)")

	// Data field descriptions
	DataID        = ffm("Data.id", "The UUID of the data resource")
	DataValidator = ffm("Data.validator", "The data validator type")
	DataNamespace = ffm("Data.namespace", "The namespace of the data resource")
	DataHash      = ffm("Data.hash", "The hash of the data resource. Derived from the value and the hash of any binary blob attachment")
	DataCreated   = ffm("Data.created", "The creation time of the data resource")
	DataDatatype  = ffm("Data.datatype", "The optional datatype to use of validation of this data")
	DataValue     = ffm("Data.value", "The value for the data, stored in the FireFly core database. Can be any JSON type - object, array, string, number or boolean. Can be combined with a binary blob attachment")
	DataBlob      = ffm("Data.blob", "An optional hash reference to a binary blob attachment")

	// SignerRef field descriptions
	SignerRefAuthor = ffm("SignerRef.author", "The DID of identity of the submitter")
	SignerRefKey    = ffm("SignerRef.key", "The on-chain signing key used to sign the transaction")

	// IdentityClaim field descriptions
	IdentityClaimIdentity = ffm("IdentityClaim.identity", "The identity being claimed")

	// IdentityVerification field descriptions
	IdentityVerificationClaim    = ffm("IdentityVerification.claim", "The UUID of the message containing the identity claim being verified")
	IdentityVerificationIdentity = ffm("IdentityVerification.identity", "The identity being verified")

	// IdentityUpdate field descriptions
	IdentityUpdateIdentity = ffm("IdentityUpdate.identity", "The identity being updated")
	IdentityUpdateProfile  = ffm("IdentityUpdate.profile", "The new profile, which is replaced in its entirety when the update is confirmed")

	// MessageManifestEntry field descriptions
	MessageManifestEntry = ffm("MessageManifestEntry.topics", "The count of topics in the message")

	// BatchHeader field descriptions
	BatchHeaderID        = ffm("BatchHeader.id", "The UUID of the batch")
	BatchHeaderType      = ffm("BatchHeader.type", "The type of the batch")
	BatchHeaderNamespace = ffm("BatchHeader.namespace", "The namespace of the batch")
	BatchHeaderNode      = ffm("BatchHeader.node", "The UUID of the node that generated the batch")
	BatchHeaderGroup     = ffm("BatchHeader.group", "The privacy group the batch is sent to, for private batches")
	BatchHeaderCreated   = ffm("BatchHeader.create", "The time the batch was sealed")

	// BatchManifest field descriptions
	BatchManifestVersion  = ffm("BatchManifest.version", "The version of the manifest generated")
	BatchManifestID       = ffm("BatchManifest.id", "The UUID of the batch")
	BatchManifestTX       = ffm("BatchManifest.tx", "The FireFly transaction associated with this batch")
	BatchManifestMessages = ffm("BatchManifest.messages", "Array of manifest entries, succinctly summarizing the messages in the batch")
	BatchManifestData     = ffm("BatchManifest.data", "Array of manifest entries, succinctly summarizing the data in the batch")

	// BatchPersisted field descriptions
	BatchPersistedHash       = ffm("BatchPersisted.version", "The hash of the manifest of the batch")
	BatchPersistedManifest   = ffm("BatchPersisted.manifest", "The manifest of the batch")
	BatchPersistedTX         = ffm("BatchPersisted.tx", "The FireFly transaction associated with this batch")
	BatchPersistedPayloadRef = ffm("BatchPersisted.payloadRef", "For broadcast batches, this is the reference to the binary batch in shared storage")
	BatchPersistedConfirmed  = ffm("BatchPersisted.confirmed", "The time when the batch was confirmed")
)
